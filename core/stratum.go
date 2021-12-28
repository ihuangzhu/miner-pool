package core

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
	"miner-pool/ethash"
	"miner-pool/util"
	"strings"
)

const (
	StratumSubmitLogin    string = "eth_submitLogin"
	StratumGetWork               = "eth_getWork"
	StratumSubmitHashrate        = "eth_submitHashrate"
	StratumSubmitWork            = "eth_submitWork"
)

var sharedEthash *ethash.Ethash

func init() {
	sharedEthash = ethash.NewEthashShared()
}

// HandleSubmitLogin
func (ss *Session) HandleSubmitLogin(params []string, worker string) error {
	// 验证数据
	if len(params) == 0 {
		return errors.New(fmt.Sprintf("Invalid params: %v", params))
	}

	// 切分钱包地址
	wallet := params[0]
	if worker == "eth1.0" {
		ps := strings.Split(params[0], ".")
		if len(ps) > 1 {
			wallet = ps[0]
			worker = ps[1]
		}
	}

	// 验证钱包地址
	if !util.IsValidHexAddress(wallet) {
		return errors.New(fmt.Sprintf("Invalid wallet: %v", wallet))
	}
	if len(worker) == 0 {
		worker = "0"
	}

	// 设置矿工
	ss.worker = worker
	ss.wallet = strings.ToLower(wallet)

	// 注册会话
	ss.proxy.registerSession(ss)

	return nil
}

// HandleGetWork
func (ss *Session) HandleGetWork() []string {
	return ss.proxy.sender.GetLastWork()
}

// HandleSubmitHashrate
func (ss *Session) HandleSubmitHashrate(params []string) error {
	return nil
}

// HandleSubmitWork
func (ss *Session) HandleSubmitWork(params []string) error {
	// check share
	if len(params) != 3 {
		return errors.New("Invalid params")
	}

	// workerName is required to know who mined the block, if there share mines it
	fullWork, ok := ss.proxy.sender.GetWorkByHeader(params[1])
	if !ok {
		// Work was not requested, or is older than 8 blocks
		return errors.New("Work is outdated, or not requested")
	}

	if fullWork[3] != ss.proxy.sender.GetLastWork()[3] {
		log.Warnf("Submit Stale Work")
	}

	header := types.Header{
		Difficulty: util.Target2diff(fullWork[2]), // 原始任务难度
		Number: hexutil.MustDecodeBig(fullWork[3]),
		Nonce: types.EncodeNonce(util.Hex2uint64(params[0])),
		MixDigest: common.HexToHash(params[2]),
	}

	// 验证工作证明
	if err := sharedEthash.VerifySeal(common.HexToHash(params[1]), &header, true); err != nil {
		log.Errorf("Invalid proof-of-work submitted, err: %v", err)
		return err
	}

	ok, err := ss.proxy.daemon.SubmitWork(params)
	if err != nil {
		log.Debugf("Unable to submit mined block! work: %v", params)
		return err
	}
	if !ok {
		log.Debugf("Submitted block marked as invalid! work: %v", params)
		return errors.New("Submit fail")
	}

	return nil
}
