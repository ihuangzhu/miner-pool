package model

import "time"

const (
	TypeIncome      string = "income"
	TypeExpenditure        = "expenditure"
)

// BalanceChange
type BalanceChange struct {
	tableName struct{} `pg:"balance_changes"`

	Id        uint64    `pg:"id,pk"`
	Wallet    string    `pg:"wallet"`
	Amount    float64   `pg:"amount"`
	Balance   float64   `pg:"balance"`
	Usage     string    `pg:"usage"`
	Type      string    `pg:"type"`
	CreatedAt time.Time `pg:"created_at"`
}
