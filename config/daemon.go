package config

type Daemon struct {
	Host          *string `json:"host"`
	Port          *int    `json:"port"`
	PortWs        *int    `json:"portWs"`
	NotifyWorkUrl *string `json:"notifyWorkUrl"`
}
