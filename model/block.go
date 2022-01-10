package model

import "time"

// 状态
const (
	BlockStatusPending   string = "pending"
	BlockStatusOrphaned         = "orphaned"
	BlockStatusConfirmed        = "confirmed"
)

// Block 块对象
type Block struct {
	tableName struct{} `pg:"blocks"`

	Id                uint64    `pg:"id"`
	Block             uint64    `pg:"block"`
	NetworkDifficulty string    `pg:"network_difficulty"`
	Miner             string    `pg:"miner"`
	Worker            string    `pg:"worker"`
	Nonce             string    `pg:"nonce"`
	Hash              string    `pg:"hash"`
	Reward            float64   `pg:"reward"`
	Status            string    `pg:"status"`
	CreatedAt         time.Time `pg:"created_at"`
}
