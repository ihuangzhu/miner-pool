package model

import "time"

// Work 任务对象
type Share struct {
	tableName struct{} `pg:"shares"`

	Block             uint64    `pg:"block"`
<<<<<<< Updated upstream
	Difficulty        string    `pg:"difficulty"`
	NetworkDifficulty string    `pg:"network_difficulty"`
=======
	Difficulty        float64   `pg:"difficulty"`
	NetworkDifficulty float64   `pg:"network_difficulty"`
>>>>>>> Stashed changes
	Miner             string    `pg:"miner"`
	Worker            string    `pg:"worker"`
	Pow               string    `pg:"pow"`
	CreatedAt         time.Time `pg:"created_at"`
}
