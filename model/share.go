package model

import "time"

// 类型
const (
	ShareTypeInvalid string = "invalid"
	ShareTypeStale          = "stale"
	ShareTypeValid          = "valid"
)

// Work 任务对象
type Share struct {
	tableName struct{} `pg:"shares"`

	Id                uint64    `pg:"id,pk"`
	Block             uint64    `pg:"block"`
	Difficulty        float64   `pg:"difficulty"`
	NetworkDifficulty float64   `pg:"network_difficulty"`
	Miner             string    `pg:"miner"`
	Worker            string    `pg:"worker"`
	Pow               string    `pg:"pow"`
	Type              string    `pg:"type"`
	CreatedAt         time.Time `pg:"created_at"`
}
