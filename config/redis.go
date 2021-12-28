package config

type Redis struct {
	Url      *string `json:"url"`
	Password *string `json:"password"`
	Prefix   *string `json:"prefix"`
	Database *int    `json:"database"`
	PoolSize *int    `json:"poolSize"`
}
