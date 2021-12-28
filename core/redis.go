package core

import (
	"context"
	"github.com/go-redis/redis/v8"
	"miner-pool/config"
)

const Separator = ":"

type Redis struct {
	Prefix string
	Client *redis.Client

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewRedis(cfg *config.Redis) *Redis {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr:     *cfg.Url,
		Password: *cfg.Password,
		DB:       *cfg.Database,
		PoolSize: *cfg.PoolSize,
	})

	return &Redis{
		Prefix: *cfg.Prefix,
		Client: client,

		ctx: ctx,
	}
}
