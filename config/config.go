package config

type Config struct {
	Threads *int    `json:"threads"`
	Name    *string `json:"name"`

	Debugger *Debugger `json:"debugger"`
	Logger   *Logger   `json:"logger"`
	Postgres *Postgres `json:"postgres"`
	Redis    *Redis    `json:"redis"`
	Proxy    *Proxy    `json:"proxy"`
	Api      *Api      `json:"api"`
}
