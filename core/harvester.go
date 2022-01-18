package core

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/params"
	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
	"math/big"
	"miner-pool/model"
	"miner-pool/util"
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
	chainConfig, err := h.svr.daemon.GetChainConfig()
	if err != nil {
		log.Errorf("Unable to get chain config: %v", err)
		return
	}

	currentBlockNumber, err := h.svr.daemon.BlockNumber()
	if err != nil {
		log.Errorf("Unable to get current blockchain height from node: %v", err)
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
				reward = calculateBlockRewards(chainConfig, uBlock)
				reward.Add(reward, calculateBlockTxnFees(h.svr.daemon, uBlock))
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
						reward = calculateUncleBlockRewards(chainConfig, height, uncle)
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

func (h *Harvester) Close() {
	close(h.quit)

	// 等待服务关闭
	h.wg.Wait()
}

// Some weird constants to avoid constant memory allocs for them.
var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

func selectStaticBlockReward(config *params.ChainConfig, blockNumber *big.Int) *big.Int {
	blockReward := ethash.FrontierBlockReward
	if config.IsByzantium(blockNumber) {
		blockReward = ethash.ByzantiumBlockReward
	}
	if config.IsConstantinople(blockNumber) {
		blockReward = ethash.ConstantinopleBlockReward
	}
	return blockReward
}

func calculateBlockRewards(config *params.ChainConfig, block *Block) *big.Int {
	// Select the correct block reward based on chain progression
	blockReward := selectStaticBlockReward(config, hexutil.MustDecodeBig(block.Number))

	// Calculate the block for the uncles inclusion reward
	uncleInclusionReward := new(big.Int).Mul(big.NewInt(int64(len(block.Uncles))), new(big.Int).Div(blockReward, big32))

	// Accumulate the total rewards
	reward := new(big.Int).Set(blockReward)
	reward.Add(reward, uncleInclusionReward)
	return reward
}

func calculateBlockTxnFees(daemon *Daemon, block *Block) *big.Int {
	gasFee := big.NewInt(0)
	for _, value := range block.Transactions {
		receipt, err := daemon.GetTxReceipt(value.Hash)
		if err != nil {
			log.Errorf("Get transaction receipt err: %v", err)
		}

		gasUsed := hexutil.MustDecodeBig(receipt.GasUsed)
		gasPrice := hexutil.MustDecodeBig(value.GasPrice)
		gasFee.Add(gasFee, new(big.Int).Mul(gasUsed, gasPrice))
	}
	gasUsed := hexutil.MustDecodeBig(block.GasUsed)
	baseFeePerGas := hexutil.MustDecodeBig(block.BaseFee)
	gasBurnt := new(big.Int).Mul(gasUsed, baseFeePerGas)
	gasReward := new(big.Int).Sub(gasFee, gasBurnt)
	log.Infof("Transaction gas fee total: %v wei, burnt: %v wei, remaing: %v wei", gasFee, gasBurnt, gasReward)

	return gasReward
}

func calculateUncleBlockRewards(config *params.ChainConfig, blockNumber *big.Int, uncle *Block) *big.Int {
	// Select the correct block reward based on chain progression
	blockReward := selectStaticBlockReward(config, blockNumber)

	// Accumulate the total rewards
	reward := new(big.Int)
	reward.Add(hexutil.MustDecodeBig(uncle.Number), big8)
	reward.Sub(reward, blockNumber)
	reward.Mul(reward, blockReward)
	reward.Div(reward, big8)
	return reward
}
