package core

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
	"math/big"
	"miner-pool/ethash"
	"miner-pool/model"
	"miner-pool/util"
	"strings"
)

// stratum 方法
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
func (ss *Session) HandleSubmitLogin(params []string, workerId string) error {
	// 验证数据
	if len(params) == 0 {
		return errors.New(fmt.Sprintf("Invalid params: %v", params))
	}

	// 切分钱包地址
	wallet := params[0]
	if workerId == "eth1.0" {
		ps := strings.Split(params[0], ".")
		if len(ps) > 1 {
			wallet = ps[0]
			workerId = ps[1]
		}
	}

	// 验证钱包地址
	if !util.IsValidHexAddress(wallet) {
		return errors.New(fmt.Sprintf("Invalid wallet: %v", wallet))
	}
	if len(workerId) == 0 {
		workerId = "0"
	}

	// 矿工登录

	worker, err := ss.proxy.svr.postgres.MinerLogin(wallet, workerId)
	if err != nil {
		return errors.New("Miner login fail: " + err.Error())
	}

	// 设置矿工
	ss.worker = worker

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

	isStale := false
	if fullWork[3] != ss.proxy.sender.GetLastWork()[3] {
		log.Warnf("Submit Stale Work")
		isStale = true
	}

	// 验证工作证明
	share := &types.Header{
		Difficulty: util.Target2diff(ss.proxy.sender.WorkTarget), // 发布任务难度
		Number:     hexutil.MustDecodeBig(fullWork[3]),
		Nonce:      types.EncodeNonce(util.Hex2uint64(params[0])),
		MixDigest:  common.HexToHash(params[2]),
	}
	difficulty, _ := new(big.Float).SetInt(share.Difficulty).Float64()
	networkDifficulty, _ := new(big.Float).SetInt(util.Target2diff(fullWork[2])).Float64()

	if err := sharedEthash.VerifySeal(common.HexToHash(params[1]), share, true); err != nil {
		log.Errorf("Invalid proof-of-work submitted, err: %v", err)
		ss.proxy.svr.postgres.WriteShare(&model.Share{
			Block:             share.Number.Uint64(),
			Miner:             ss.worker.Miner,
			Worker:            ss.worker.Worker,
			Pow:               strings.Join(params, ":"),
			Type:              model.ShareTypeInvalid,
			Difficulty:        difficulty,
			NetworkDifficulty: networkDifficulty,
		})
		return err
	}

	// 提交任务结果
	submitWorkResult := model.ShareTypeValid
	if isStale {
		submitWorkResult = model.ShareTypeStale
	}

	// 提交任务
	ok, err := ss.proxy.svr.daemon.SubmitWork(params)
	if !ok || err != nil {
		ss.proxy.svr.postgres.WriteShare(&model.Share{
			Block:             share.Number.Uint64(),
			Miner:             ss.worker.Miner,
			Worker:            ss.worker.Worker,
			Pow:               strings.Join(params, ":"),
			Type:              submitWorkResult,
			Difficulty:        difficulty,
			NetworkDifficulty: networkDifficulty,
		})

		if err != nil {
			log.Debugf("Unable to submit mined block! work: %v", params)
			return err
		}

		if !ok {
			log.Debugf("Submitted block marked as invalid! work: %v", params)
			return errors.New("Submit fail")
		}
	}

	ss.proxy.svr.postgres.WriteBlock(&model.Share{
		Block:             share.Number.Uint64(),
		Miner:             ss.worker.Miner,
		Worker:            ss.worker.Worker,
		Pow:               strings.Join(params, ":"),
		Type:              submitWorkResult,
		Difficulty:        difficulty,
		NetworkDifficulty: networkDifficulty,
	}, &model.Block{
		Block:             share.Number.Uint64(),
		Miner:             ss.worker.Miner,
		Worker:            ss.worker.Worker,
		Nonce:             params[0],
		NetworkDifficulty: networkDifficulty,
		Status:            model.BlockStatusPending,
	})

	return nil
}
