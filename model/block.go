package model

import "time"

// 状态
const (
	BlockStatusPending   string = "pending"
	BlockStatusOrphaned         = "orphaned"
	BlockStatusConfirmed        = "confirmed"
)

// 类型
const (
	BlockTypeBlock string = "block"
	BlockTypeUncle        = "uncle"
)

// Block 块对象
type Block struct {
	tableName struct{} `pg:"blocks"`

	Id                uint64    `pg:"id,pk"`
	Block             uint64    `pg:"block"`
	NetworkDifficulty float64   `pg:"network_difficulty"`
	Miner             string    `pg:"miner"`
	Worker            string    `pg:"worker"`
	Nonce             string    `pg:"nonce"`
	Hash              string    `pg:"hash"`
	Type              string    `pg:"type"`
	UncleIndex        uint      `pg:"uncle_index,use_zero"`
	Reward            float64   `pg:"reward"`
	Status            string    `pg:"status"`
	CreatedAt         time.Time `pg:"created_at"`
}
