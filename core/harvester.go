package core

import (
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
	"math/big"
	"miner-pool/model"
	"miner-pool/util"
	"strconv"
	"strings"
	"sync"
	"time"
)

const minDepth = 16

// Harvester
type Harvester struct {
	svr *Server

	wg   sync.WaitGroup
	quit chan struct{}

	intervalTimer *time.Timer
}

// NewHarvester
func NewHarvester(svr *Server) *Harvester {
	if len(*svr.cfg.Harvester.PoolFeeAddress) != 0 && !util.IsValidHexAddress(*svr.cfg.Harvester.PoolFeeAddress) {
		log.Fatalf("Invalid poolFeeAddress %v", *svr.cfg.Harvester.PoolFeeAddress)
	}
	if *svr.cfg.Harvester.Depth < minDepth*2 {
		log.Fatalf("Block maturity depth can't be < %v, your depth is %v", minDepth*2, *svr.cfg.Harvester.Depth)
	}
	if *svr.cfg.Harvester.ImmatureDepth < minDepth {
		log.Fatalf("Immature depth can't be < %v, your depth is %v", minDepth, *svr.cfg.Harvester.ImmatureDepth)
	}

	h := &Harvester{
		svr:  svr,
		quit: make(chan struct{}),
	}

	h.wg.Add(1)
	go h.listen()
	return h
}

func (h *Harvester) listen() {
	defer h.wg.Done()

	log.Info("Starting block harvester")
	interval := util.MustParseDuration(*h.svr.cfg.Harvester.Interval)
	h.intervalTimer = time.NewTimer(interval)
	log.Infof("Set block harvester interval to %v", interval)

	for {
		select {
		case <-h.quit:
			return

		case <-h.intervalTimer.C:
			h.harvestPendingBlocks()
			h.intervalTimer.Reset(interval)
		}
	}
}

// harvestPendingBlocks
func (h *Harvester) harvestPendingBlocks() {
	currentBlockNumber, err := h.svr.daemon.BlockNumber()
	if err != nil {
		log.Infof("Unable to get current blockchain height from node: %v", err)
		return
	}

	var blocks []model.Block
	depth := currentBlockNumber - *h.svr.cfg.Harvester.ImmatureDepth
	if err := h.svr.postgres.db.Model(&blocks).Where("block <= ? and status = ?", depth, model.BlockStatusPending).Select(); err != nil {
		log.Infof("Failed to get block candidates from backend: %v", err)
		return
	}

	// 循环块
	for _, block := range blocks {
		orphan := true
		reward := big.NewInt(0)

		for i := int64(minDepth * -1); i < minDepth; i++ {
			height := big.NewInt(int64(block.Block) + i)
			if height.Int64() < 0 {
				continue
			}

			// 收益
			uBlock, err := h.svr.daemon.GetBlockByNumber(height.Uint64())
			if err != nil {
				continue
			}

			// 块收益
			if block.Nonce == uBlock.Nonce {
				orphan = false
				block.Hash = uBlock.Hash
				block.Type = model.BlockTypeBlock

				// 获取收益
				reward = h.calculateRewardBlock(uBlock, h.svr.daemon)
			}

			if len(uBlock.Uncles) == 0 {
				continue
			}

			// 叔块收益
			for uncleIndex := range uBlock.Uncles {
				if uncle, err := h.svr.daemon.GetUncleByBlockNumberAndIndex(height.Uint64(), uncleIndex); err == nil {
					if block.Nonce == uncle.Nonce {
						orphan = false
						block.Hash = uncle.Hash
						block.Type = model.BlockTypeUncle
						block.UncleIndex = uint(uncleIndex)

						// 获取收益
						reward = h.calculateRewardUncleBlock(height.Uint64(), uncle)
					}
				}
			}
		}

		log.Infof("orphan: %v, reward: %v", orphan, reward)

		// 结算块
		rewardEth := new(big.Float).Quo(new(big.Float).SetInt(reward), new(big.Float).SetInt(util.Ether))
		block.Reward, _ = rewardEth.Float64()
		if orphan {
			block.Status = model.BlockStatusOrphaned
		} else {
			block.Status = model.BlockStatusConfirmed
		}
		h.svr.postgres.db.Model(&block).WherePK().Update()

		// 分账
		go func(block model.Block) {
			var shares []model.Share
			if err := h.svr.postgres.db.Model(&shares).Where("block = ?", block.Block).Select(); err != nil {
				log.Infof("Failed to get share from backend: %v", err)
				return
			}

			// 统计
			times := make(map[string]int32)
			for _, share := range shares {
				times[share.Miner] += 1
			}

			// 分账
			rw := new(big.Rat).SetFloat64(block.Reward)
			for k, v := range times {
				rat := big.NewRat(int64(v), int64(len(shares)))
				rew := new(big.Rat).Mul(rw, rat)
				re, _ := rew.Float64()

				// 保存
				if err = h.svr.postgres.db.RunInTransaction(h.svr.postgres.ctx, func(tx *pg.Tx) error {
					var balance model.Balance
					if err := tx.Model(&balance).Where("wallet = ?", k).First(); err != nil {
						if err == pg.ErrNoRows {
							balance.Wallet = k
							balance.Amount = re
							balance.CreatedAt = time.Now()
							balance.UpdatedAt = time.Now()
							_, err := tx.Model(&balance).Insert()

							return err
						}

						log.Infof("Failed to get balance from backend: %v", err)
						return err
					}

					// 更新余额
					amount := new(big.Float).SetFloat64(balance.Amount)
					amount = amount.Add(amount, new(big.Float).SetFloat64(re))
					balance.Amount, _ = amount.Float64()
					balance.UpdatedAt = time.Now()
					if _, err := tx.Model(&balance).WherePK().Update(); err != nil {
						return err
					}

					// 记录余额变更
					var balanceChange model.BalanceChange
					balanceChange.Wallet = k
					balanceChange.Amount = re
					balanceChange.Balance = balance.Amount
					balanceChange.Usage = "mining reward"
					balanceChange.Type = model.TypeIncome
					balanceChange.CreatedAt = time.Now()
					if _, err := tx.Model(&balanceChange).Insert(); err != nil {
						return err
					}

					return nil
				}); err != nil {
					log.Error(err)
				}

			}
		}(block)
	}
}

// calculateRewardBlock 计算块奖励
func (h *Harvester) calculateRewardBlock(block *Block, daemon *Daemon) *big.Int {
	reward := new(big.Int)

	// 计算固定奖励
	height, err := strconv.ParseUint(strings.Replace(block.Number, "0x", "", -1), 16, 64)
	if err != nil {
		log.Error(err)
	}
	staticReward := GetStaticReward(height)
	reward.Set(staticReward)

	// 计算转账燃油奖励
	gasFee := new(big.Int)
	for key, value := range block.Transactions {
		log.Infof("Transaction: %v => %v", key, value)

		receipt, err := daemon.GetTxReceipt(value.Hash)
		if err != nil {
			log.Error(err)
		}

		gasUsed := big.NewInt(util.Hex2int64(receipt.GasUsed))
		gasPrice := big.NewInt(util.Hex2int64(value.GasPrice))
		txnFee := new(big.Int).Mul(gasUsed, gasPrice)
		gasFee.Add(gasFee, txnFee)
	}
	gasUsed := big.NewInt(util.Hex2int64(block.GasUsed))
	baseFeePerGas := big.NewInt(util.Hex2int64(block.BaseFee))
	gasBurnt := new(big.Int).Mul(gasUsed, baseFeePerGas)
	gasReward := new(big.Int).Sub(gasFee, gasBurnt)
	reward.Add(reward, gasReward)
	log.Infof("Transaction gas fee total: %v wei, burnt: %v wei, remaing: %v wei", gasFee, gasBurnt, gasReward)

	// 计算叔块奖励
	staticUncleReward := new(big.Int).Div(staticReward, new(big.Int).SetInt64(32))
	uncleReward := big.NewInt(0).Mul(staticUncleReward, big.NewInt(int64(len(block.Uncles))))
	reward.Add(reward, uncleReward)
	log.Infof("Uncle block reward: %v wei", uncleReward)

	return reward
}

// calculateRewardUncleBlock 计算叔块奖励
func (h *Harvester) calculateRewardUncleBlock(height uint64, uncle *Block) *big.Int {
	reward := big.NewInt(0)
	staticReward := GetStaticReward(height)
	uncleHeight, err := strconv.ParseUint(strings.Replace(uncle.Number, "0x", "", -1), 16, 64)
	if err != nil {
		log.Print(err)
	}

	reward = staticReward
	k := height - uncleHeight
	reward.Mul(big.NewInt(int64(8-k)), reward)
	reward.Div(reward, big.NewInt(8))

	log.Infof("Uncle block reward: %v wei", reward)

	return reward
}

func (h *Harvester) Close() {
	close(h.quit)

	// 等待服务关闭
	h.wg.Wait()
}

// For blocks 0 -> 4,369,999 the reward was 5 ETH.
// For blocks 4,370,000 -> 7,279,999 the reward was 3 ETH.
// For blocks 7,280,000 onward the reward is 2 ETH.
const (
	ByzantiumHardFork      = 4370000
	ConstantinopleHardFork = 7280000
)

func GetStaticReward(height uint64) *big.Int {
	if height < ByzantiumHardFork {
		return math.MustParseBig256("5000000000000000000")
	} else if height < ConstantinopleHardFork {
		return math.MustParseBig256("3000000000000000000")
	} else {
		return math.MustParseBig256("2000000000000000000")
	}
}
