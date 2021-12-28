package core

import (
	"bufio"
	log "github.com/sirupsen/logrus"
	"io"
	"miner-pool/jsonrpc"
	"miner-pool/util"
	"net"
	"regexp"
	"sync"
	"time"
)

const MaxReqSize = 1024

var jsonRpcPattern = regexp.MustCompile("({[^}]+})+")

type Proxy struct {
	server  *Server

	sender   *Sender
	daemon   *Daemon
	receiver *Receiver
	timeout time.Duration
	listener *net.TCPListener

	wg   sync.WaitGroup
	quit chan struct{}

	sessionsMu sync.RWMutex
	sessions   map[*Session]struct{}
}

func NewProxy(server *Server) *Proxy {
	p := &Proxy{
		server:  server,

		sender: NewSender(server.cfg.Proxy),
		daemon: NewDaemon(server.cfg.Proxy.Daemon),
		timeout: util.MustParseDuration(*server.cfg.Proxy.Timeout),

		quit: make(chan struct{}),

		sessions: make(map[*Session]struct{}),
	}
	p.receiver = StartReceiver(p)

	p.wg.Add(1)
	go p.listen()
	return p
}

func (p *Proxy) listen() {
	defer p.wg.Done()

	addr, err := net.ResolveTCPAddr("tcp", *p.server.cfg.Proxy.Listen)
	if err != nil {
		log.Fatalf("Proxy listen err: %v", err)
	}

	p.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalf("Proxy listen tcp err: %v", err)
	}

	log.Infof("Proxy listening on %s", *p.server.cfg.Proxy.Listen)
	var accept = make(chan int, *p.server.cfg.Proxy.MaxConn)
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
			ss := &Session{name: ip, conn: conn, proxy: p}
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
