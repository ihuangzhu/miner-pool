package model

import "time"

// Pool
type Pool struct {
<<<<<<< Updated upstream
	tableName struct{} `pg:"blocks"`

	Id                uint64    `pg:"id"`
=======
	tableName struct{} `pg:"pools"`

	Id                uint64    `pg:"id,pk"`
>>>>>>> Stashed changes
	Miners            uint32    `pg:"miners"`
	Block             uint64    `pg:"block"`
	PoolHashrate      string    `pg:"pool_hashrate"`
	NetworkHashrate   string    `pg:"network_hashrate"`
	NetworkDifficulty string    `pg:"network_difficulty"`
	CreatedAt         time.Time `pg:"created_at"`
}
