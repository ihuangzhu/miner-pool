package model

import "time"

// Miner
type Miner struct {
	tableName struct{} `pg:"miners"`

	Id        uint32    `pg:"id"`
	Miner     string    `pg:"miner"`
	Worker    string    `pg:"worker"`
	Hashrate  float64   `pg:"hashrate"`
	CreatedAt time.Time `pg:"created_at"`
}
