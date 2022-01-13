package model

import "time"

// Pool
type Pool struct {
	tableName struct{} `pg:"pools"`

	Id                uint64    `pg:"id,pk"`
	Miners            uint32    `pg:"miners"`
	Workers           uint32    `pg:"workers"`
	Block             uint64    `pg:"block"`
	PoolHashrate      string    `pg:"pool_hashrate"`
	NetworkHashrate   string    `pg:"network_hashrate"`
	NetworkDifficulty string    `pg:"network_difficulty"`
	CreatedAt         time.Time `pg:"created_at"`
}
