module miner-pool

go 1.16

require (
	github.com/deckarep/golang-set v0.0.0-20180603214616-504e848d77ea // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/ethereum/go-ethereum v1.10.14
	github.com/go-redis/redis/v8 v8.11.4
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/mitchellh/mapstructure v1.4.3
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v1.3.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
)

replace github.com/ethereum/go-ethereum => github.com/ihuangzhu/go-ethereum v1.10.14-0.20211227061312-8d756c1aebb7
