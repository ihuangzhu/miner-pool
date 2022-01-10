package config

type Postgres struct {
	Address  *string `json:"address"`
	Database *string `json:"database"`
	Username *string `json:"username"`
	Password *string `json:"password"`
}
