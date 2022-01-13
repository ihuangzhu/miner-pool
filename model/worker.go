package model

import "time"

// Worker
type Worker struct {
	tableName struct{} `pg:"workers"`

	Id               uint32    `pg:"id,pk"`
	Miner            string    `pg:"miner"`
	Worker           string    `pg:"worker"`
	Hashrate         string    `pg:"hashrate"`
	InvalidShares    uint64    `pg:"invalid_shares"`
	StaleShares      uint64    `pg:"stale_shares"`
	ValidShares      uint64    `pg:"valid_shares"`
	Online           bool      `pg:"online"`
	CreatedAt        time.Time `pg:"created_at"`
	LastValidShareAt time.Time `pg:"last_valid_share_at"`
}
