package model

import "time"

// Miner
type Miner struct {
	tableName struct{} `pg:"miners"`

<<<<<<< Updated upstream
	Id        uint32    `pg:"id"`
	Miner     string    `pg:"miner"`
	Worker    string    `pg:"worker"`
	Hashrate  float64   `pg:"hashrate"`
=======
	Id        uint32    `pg:"id,pk"`
	Miner     string    `pg:"miner"`
	Worker    string    `pg:"worker"`
	Hashrate  string    `pg:"hashrate"`
>>>>>>> Stashed changes
	CreatedAt time.Time `pg:"created_at"`
}
