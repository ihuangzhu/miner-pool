package config

type Proxy struct {
	Enabled       *bool   `json:"enabled"`
	Listen        *string `json:"listen"`
	Timeout       *string `json:"timeout"`
	MaxConn       *int    `json:"maxConn"`
	Target        *string `json:"target"`
	StateInterval *string `json:"stateInterval"`

	Daemon *Daemon `json:"daemon"`
}
