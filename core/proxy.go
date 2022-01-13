package core

import (
	"bufio"
	log "github.com/sirupsen/logrus"
	"io"
	"math/big"
	"miner-pool/jsonrpc"
	"miner-pool/model"
	"miner-pool/util"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const MaxReqSize = 1024

var jsonRpcPattern = regexp.MustCompile("({[^}]+})+")

type Proxy struct {
	svr *Server

	sender   *Sender
	receiver *Receiver
	timeout  time.Duration
	listener *net.TCPListener

	stateInterval time.Duration
	stateTimer    *time.Timer

	wg   sync.WaitGroup
	quit chan struct{}

	sessionsMu sync.RWMutex
	sessions   map[*Session]struct{}
}

func NewProxy(svr *Server) *Proxy {
	p := &Proxy{
		svr: svr,

		sender:  NewSender(svr.cfg.Proxy),
		timeout: util.MustParseDuration(*svr.cfg.Proxy.Timeout),

		stateInterval: util.MustParseDuration(*svr.cfg.Proxy.StateInterval),

		quit: make(chan struct{}),

		sessions: make(map[*Session]struct{}),
	}
	p.receiver = StartReceiver(p)

	p.wg.Add(1)
	go p.state()
	p.wg.Add(1)
	go p.listen()
	return p
}

func (p *Proxy) listen() {
	defer p.wg.Done()

	addr, err := net.ResolveTCPAddr("tcp", *p.svr.cfg.Proxy.Listen)
	if err != nil {
		log.Fatalf("Proxy listen err: %v", err)
	}

	p.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalf("Proxy listen tcp err: %v", err)
	}

	log.Infof("Proxy listening on %s", *p.svr.cfg.Proxy.Listen)
	var accept = make(chan int, *p.svr.cfg.Proxy.MaxConn)
	n := 0

	for {
		select {
		case <-p.quit:
			return

		default:
			conn, err := p.listener.AcceptTCP()
			if err != nil {
				continue
			}
			conn.SetKeepAlive(true)

			// 获取IP地址
			ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

			n += 1
			accept <- n
			ss := &Session{ip: ip, proxy: p, conn: conn}
			// 开启连接会话
			p.wg.Add(1)
			go func(ss *Session) {
				if err = p.handleConnection(ss); err != nil {
					p.removeSession(ss)
					conn.Close()
				}
				p.wg.Done()
				<-accept
			}(ss)
		}
	}
}

func (p *Proxy) state() {
	defer p.wg.Done()

	p.stateTimer = time.NewTimer(p.stateInterval)

	for {
		select {
		case <-p.quit:
			return

		case <-p.stateTimer.C:
			p.persistenceState()
			p.stateTimer.Reset(p.stateInterval)
		}
	}
}

func (p *Proxy) handleConnection(ss *Session) error {
	defer ss.conn.Close()

	// 读取客户端的消息
	connBuff := bufio.NewReaderSize(ss.conn, MaxReqSize)
	ss.conn.SetDeadline(time.Now().Add(p.timeout))

	for {
		select {
		case <-p.quit:
			log.Print("handler is exiting")
			return nil

		default:
			data, isPrefix, err := connBuff.ReadLine()
			if isPrefix {
				log.Debugf("Socket flood detected from: %p", ss.conn)
				return err
			} else if err == io.EOF {
				log.Debugf("Client %p disconnected", ss.conn)
				p.removeSession(ss)
				return err
			} else if err != nil {
				log.Debugf("Error reading from socket: %v", err)
				return err
			}

			// 解析RPC请求数据
			if len(data) > 0 {
				log.Debugf("Request data: %v", string(data))
				request, err := jsonrpc.UnmarshalRequest(data)
				if err != nil {
					// 有时会有两条json同时传递上来：{json1}{json2}
					params := jsonRpcPattern.FindStringSubmatch(string(data))
					for _, param := range params {
						if request, err = jsonrpc.UnmarshalRequest([]byte(param)); err == nil {
							if request.Method == StratumSubmitWork {
								break
							}
						}
					}

					// 解析成功则继续执行
					if &request != nil {
						goto GOON
					}

					log.Errorf("Invalid JSONRPC Request data: %v", string(data))
					return err
				}
			GOON:
				log.Debugf("Request method: %v", request.Method)

				// 更新超时时间
				ss.conn.SetDeadline(time.Now().Add(p.timeout))
				if err = ss.handleTCPMessage(&request); err != nil {
					return err
				}
			}
		}
	}
}

type Worker struct {
	HR       float64
	Online   bool
	LastBeat time.Time
}
type Miner struct {
	HR       float64
	Online   bool
	workers  map[string]Worker
}

func (p *Proxy) persistenceState() {
	work := p.sender.LastWork
	if work == nil {
		return
	}

	// 查询
	var shares []model.Share
	tenMinutesAgo := time.Now().Add(-600 * time.Second)
	err := p.svr.postgres.db.Model((*model.Share)(nil)).
		Column("miner", "worker", "difficulty", "created_at").
		Where("created_at >= ?", tenMinutesAgo).
		Select(&shares)
	if err != nil {
		log.Infof("Failed to get share from backend: %v", err)
		return
	}

	// 统计
	miners := make(map[string]Miner)
	for _, share := range shares {
		miner := miners[share.Miner]
		worker := miner.workers[share.Worker]

		worker.HR += share.Difficulty
		if worker.LastBeat.Before(share.CreatedAt) {
			worker.LastBeat = share.CreatedAt
		}

		if miner.workers == nil {
			miner.workers = make(map[string]Worker)
		}
		miner.workers[share.Worker] = worker
		miners[share.Miner] = miner
	}

	// 计算
	online := int64(0)
	totalHR := float64(0)
	for wallet, miner := range miners {
		for workerId, worker := range miner.workers {
			worker.HR = worker.HR / 600
			if worker.LastBeat.After(time.Now().Add(-300 * time.Second)) {
				online ++
				worker.Online = true
				miner.Online = true
			}

			p.svr.postgres.db.Model((*model.Worker)(nil)).
				Set("hashrate = ?, online = ?", worker.HR, worker.Online).
				Where("miner = ? and worker = ?", wallet, workerId).Update()

			miner.HR += worker.HR
		}

		p.svr.postgres.db.Model((*model.Miner)(nil)).
			Set("hashrate = ?, online = ?", miner.HR, miner.Online).
			Where("miner = ?", wallet).Update()

		totalHR += miner.HR
	}

	block := util.Hex2uint64(work[3])
	poolHashrate := big.NewFloat(totalHR)
	networkDifficulty := util.Target2diff(work[2])
	networkHashrate, _ := p.svr.daemon.GetNetworkHashrate(600)

	p.svr.postgres.WriteState(&model.Pool{
		Miners:            uint32(len(miners)),
		Workers:           uint32(online),
		Block:             block,
		PoolHashrate:      poolHashrate.String(),
		NetworkHashrate:   strconv.FormatUint(networkHashrate, 10),
		NetworkDifficulty: networkDifficulty.String(),
	})
}

func (p *Proxy) registerSession(ss *Session) {
	p.sessionsMu.Lock()
	defer p.sessionsMu.Unlock()

	p.sender.Attach(ss)
	p.sessions[ss] = struct{}{}
	if len(p.sessions)%100 == 0 {
		log.Infof("[REG] Total number of sessions: %v", len(p.sessions))
	}
}

func (p *Proxy) removeSession(ss *Session) {
	p.sessionsMu.Lock()
	defer p.sessionsMu.Unlock()

	p.sender.Detach(ss)
	delete(p.sessions, ss)
	if len(p.sessions)%100 == 0 {
		log.Infof("[RM] Total number of sessions: %v", len(p.sessions))
	}
}

func (p *Proxy) Close() {
	close(p.quit)
	p.sender.Close()
	p.receiver.Close()
	p.listener.Close()

	// 等待TCP服务关闭
	p.wg.Wait()
}
