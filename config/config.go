package config

type Config struct {
	Threads *int    `json:"threads"`
	Name    *string `json:"name"`

<<<<<<< Updated upstream
	Debugger *Debugger `json:"debugger"`
	Logger   *Logger   `json:"logger"`
	Postgres *Postgres `json:"postgres"`
	Redis    *Redis    `json:"redis"`
	Proxy    *Proxy    `json:"proxy"`
	Api      *Api      `json:"api"`
=======
	Debugger  *Debugger  `json:"debugger"`
	Logger    *Logger    `json:"logger"`
	Postgres  *Postgres  `json:"postgres"`
	Harvester *Harvester `json:"harvester"`
	Proxy     *Proxy     `json:"proxy"`
	Api       *Api       `json:"api"`
>>>>>>> Stashed changes
}
