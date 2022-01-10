package config

// Harvester 收集器
type Harvester struct {
	Enabled        *bool    `json:"enabled"`
	PoolFee        *float64 `json:"poolFee"`
	PoolFeeAddress *string  `json:"poolFeeAddress"`
	Depth          *uint64  `json:"depth"`
	ImmatureDepth  *uint64  `json:"immatureDepth"`
	KeepTxFees     *bool    `json:"keepTxFees"`
	Interval       *string  `json:"interval"`
}
