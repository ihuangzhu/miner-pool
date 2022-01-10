package core

import (
	log "github.com/sirupsen/logrus"
	"miner-pool/jsonrpc"
	"miner-pool/model"
	"net"
	"time"
)

type Session struct {
	ip    string
	miner *model.Miner

	proxy *Proxy
	conn  *net.TCPConn
}

func (ss *Session) handleTCPMessage(request *jsonrpc.RequestStratum) error {
	// Handle RPC methods
	switch request.Method {
	case StratumSubmitLogin:
		if err := ss.HandleSubmitLogin(request.Params, request.Worker); err != nil {
			return ss.Send(request.Id, err.Error(), nil)
		}
		return ss.Send(request.Id, nil, true)
	case StratumGetWork:
		return ss.Send(request.Id, nil, ss.HandleGetWork())
	case StratumSubmitHashrate:
		return ss.Send(request.Id, nil, true)
	case StratumSubmitWork:
		err := ss.HandleSubmitWork(request.Params)
		if err != nil {
			return ss.Send(request.Id, err, nil)
		}
		return ss.Send(request.Id, nil, true)
	default:
		return ss.Send(request.Id, "Method not found", nil)
	}
}

func (ss *Session) Notify(work []string) {
	log.Debugf("Send work: %v", work)
	if err := ss.Send(0, nil, work); err != nil {
		ss.proxy.removeSession(ss)
	} else {
		ss.conn.SetDeadline(time.Now().Add(ss.proxy.timeout))
	}
}

// Send 发送数据到客户端
func (ss *Session) Send(id int, msg interface{}, result interface{}) error {
	// 写回客户端
	if _, err := ss.conn.Write(append(jsonrpc.MarshalResponse(jsonrpc.Response{
		Id:      id,
		Version: jsonrpc.Version,
		Result:  result,
		Error:   msg,
	}), '\n')); err != nil {
		log.Debugf("Send data error: %v", err)
		return err
	}

	return nil
}
