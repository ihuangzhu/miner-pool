package model

import "time"

// Hashrate
type Hashrate struct {
	tableName struct{} `pg:"hashrates"`

	Id          uint32    `pg:"id,pk"`
	Miner       string    `pg:"miner"`
	Hashrate    string    `pg:"hashrate"`
	Hashrate1h  string    `pg:"hashrate1h"`
	Hashrate6h  string    `pg:"hashrate6h"`
	Hashrate12h string    `pg:"hashrate12h"`
	Hashrate24h string    `pg:"hashrate24h"`
	CreatedAt   time.Time `pg:"created_at"`
}
