package model

import "time"

// Work 任务对象
type Share struct {
	tableName struct{} `pg:"shares"`

	Block             uint64    `pg:"block"`
	Difficulty        string    `pg:"difficulty"`
	NetworkDifficulty string    `pg:"network_difficulty"`
	Miner             string    `pg:"miner"`
	Worker            string    `pg:"worker"`
	Pow               string    `pg:"pow"`
	CreatedAt         time.Time `pg:"created_at"`
}
