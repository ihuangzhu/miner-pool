package model

import "time"

// Miner
type Miner struct {
	tableName struct{} `pg:"miners"`

	Id               uint32    `pg:"id,pk"`
	Miner            string    `pg:"miner"`
	Hashrate         string    `pg:"hashrate"`
	InvalidShares    uint64    `pg:"invalid_shares"`
	StaleShares      uint64    `pg:"stale_shares"`
	ValidShares      uint64    `pg:"valid_shares"`
	OnlineWorkers    uint64    `pg:"online_workers"`
	OfflineWorkers   uint64    `pg:"offline_workers"`
	CreatedAt        time.Time `pg:"created_at"`
	LastValidShareAt time.Time `pg:"last_valid_share_at"`
}
