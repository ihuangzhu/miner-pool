package core

import (
	"context"
	"errors"
	"github.com/go-pg/pg/v10"
	"math/big"
	"miner-pool/config"
	"miner-pool/model"
	"strings"
	"time"
)

type Postgres struct {
	db *pg.DB

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewPostgres(cfg *config.Postgres) *Postgres {
	ctx := context.Background()

	db := pg.Connect(&pg.Options{
		Addr:     *cfg.Address,
		User:     *cfg.Username,
		Password: *cfg.Password,
		Database: *cfg.Database,
	})
	if err := db.Ping(ctx); err != nil {
		panic(err)
	}

	return &Postgres{
		db: db,

		ctx: ctx,
	}
}

func (p *Postgres) Close() {
	p.db.Close()
}

// MinerLogin 矿工登录
func (p *Postgres) MinerLogin(wallet string, worker string) (*model.Miner, error) {
	// 查询用户
	var miner model.Miner
	wallet = strings.ToLower(wallet)
	if err := p.db.Model(&miner).Where("miner = ? and worker = ?", wallet, worker).First(); err != nil {
		if err == pg.ErrNoRows {
			miner.Miner = wallet
			miner.Worker = worker
			miner.CreatedAt = time.Now()
			if _, err := p.db.Model(&miner).Insert(); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &miner, nil
}

// CheckPoWExist 验证工作证明（Proof-of-Work）
func (p *Postgres) CheckPoWExist(share *model.Share) (bool, error) {
	block := big.NewInt(0).SetUint64(share.Block)
	blockMin := big.NewInt(0).Sub(block, big.NewInt(8))
	count, err := p.db.Model(&model.Share{}).Where("block > ? and pow = ?", blockMin.String(), share.Pow).For("UPDATE").Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// WriteShare 写入任务数据
func (p *Postgres) WriteShare(share *model.Share) error {
	return p.db.RunInTransaction(p.ctx, func(tx *pg.Tx) error {
		if exist, err := p.CheckPoWExist(share); err != nil {
			return err
		} else if exist {
			return errors.New("Pow exists.")
		}

		share.CreatedAt = time.Now()
		if _, err := tx.Model(share).Insert(); err != nil {
			return err
		}

		return nil
	})
}

// WriteBlock 写入块数据
func (p *Postgres) WriteBlock(share *model.Share, block *model.Block) error {
	return p.db.RunInTransaction(p.ctx, func(tx *pg.Tx) error {
		// 写入任务数据
		if err := p.WriteShare(share); err != nil {
			return err
		}

		// 写入块数据
		block.CreatedAt = time.Now()
		if _, err := tx.Model(block).Insert(); err != nil {
			return err
		}

		return nil
	})
}

func (p *Postgres) WriteState(pool *model.Pool) error {
	return p.db.RunInTransaction(p.ctx, func(tx *pg.Tx) error {

		// 写入块数据
		pool.CreatedAt = time.Now()
		if _, err := tx.Model(pool).Insert(); err != nil {
			return err
		}

		return nil
	})
}
