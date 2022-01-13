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
	HR1h     float64
	HR6h     float64
	HR12h    float64
	HR24h    float64
	Online   bool
	LastBeat time.Time
}
type Miner struct {
	HR      float64
	HR1h    float64
	HR6h    float64
	HR12h   float64
	HR24h   float64
	workers map[string]Worker
}

func (p *Proxy) persistenceState() {
	work := p.sender.LastWork
	if work == nil {
		return
	}

	// 查询
	var shares []model.Share
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	err := p.svr.postgres.db.Model((*model.Share)(nil)).
		Column("miner", "worker", "difficulty", "created_at").
		Where("created_at >= ?", twentyFourHoursAgo).
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

		// 十分钟
		if share.CreatedAt.After(time.Now().Add(-10 * time.Minute)) {
			worker.HR += share.Difficulty
		}

		// 一小时
		if share.CreatedAt.After(time.Now().Add(-1 * time.Hour)) {
			worker.HR1h += share.Difficulty
		}

		// 六小时
		if share.CreatedAt.After(time.Now().Add(-6 * time.Hour)) {
			worker.HR6h += share.Difficulty
		}

		// 十二小时
		if share.CreatedAt.After(time.Now().Add(-12 * time.Hour)) {
			worker.HR12h += share.Difficulty
		}

		// 二十四小时
		if share.CreatedAt.After(time.Now().Add(-24 * time.Hour)) {
			worker.HR24h += share.Difficulty
		}

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
		onlineWorkers := uint64(0)
		offlineWorkers := uint64(0)
		for workerId, worker := range miner.workers {
			worker.HR = worker.HR / 600
			worker.HR1h = worker.HR1h / 1 / 60 / 60
			worker.HR6h = worker.HR6h / 6 / 60 / 60
			worker.HR12h = worker.HR12h / 12 / 60 / 60
			worker.HR24h = worker.HR24h / 24 / 60 / 60
			if worker.LastBeat.After(time.Now().Add(-300 * time.Second)) {
				online++
				onlineWorkers++
				worker.Online = true
			} else {
				offlineWorkers++
			}

			p.svr.postgres.db.Model((*model.Worker)(nil)).
				Set("hashrate = ?, online = ?", worker.HR, worker.Online).
				Where("miner = ? and worker = ?", wallet, workerId).Update()

			miner.HR += worker.HR
			miner.HR1h += worker.HR1h
			miner.HR6h += worker.HR6h
			miner.HR12h += worker.HR12h
			miner.HR24h += worker.HR24h
		}

		var hashrate model.Hashrate
		hashrate.Miner = wallet
		hashrate.Hashrate = strconv.FormatFloat(miner.HR, 'f', 10, 64)
		hashrate.Hashrate1h = strconv.FormatFloat(miner.HR1h, 'f', 10, 64)
		hashrate.Hashrate6h = strconv.FormatFloat(miner.HR6h, 'f', 10, 64)
		hashrate.Hashrate12h = strconv.FormatFloat(miner.HR12h, 'f', 10, 64)
		hashrate.Hashrate24h = strconv.FormatFloat(miner.HR24h, 'f', 10, 64)
		hashrate.CreatedAt = time.Now()
		p.svr.postgres.db.Model(&hashrate).Insert()

		p.svr.postgres.db.Model((*model.Miner)(nil)).
			Set("hashrate = ?, online_workers = ?, offline_workers = ?", miner.HR, onlineWorkers, offlineWorkers).
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
