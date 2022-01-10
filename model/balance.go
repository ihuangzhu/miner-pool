package model

import "time"

// Balance
type Balance struct {
	tableName struct{} `pg:"balances"`

	Wallet    string    `pg:"wallet,pk"`
	Amount    float64   `pg:"amount"`
	CreatedAt time.Time `pg:"created_at"`
	UpdatedAt time.Time `pg:"updated_at"`
}
