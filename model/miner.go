package model

import "time"

// Miner
type Miner struct {
	tableName struct{} `pg:"miners"`

	Id        uint32    `pg:"id,pk"`
	Miner     string    `pg:"miner"`
	Worker    string    `pg:"worker"`
	Hashrate  string    `pg:"hashrate"`
	CreatedAt time.Time `pg:"created_at"`
}
