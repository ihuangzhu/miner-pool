package config

type Daemon struct {
	Chain         *string `json:"chain"`
	Host          *string `json:"host"`
	Port          *int    `json:"port"`
	NotifyWorkUrl *string `json:"notifyWorkUrl"`
}
