package redis

import (
	"smap-collector/config"
	"time"

	"github.com/redis/go-redis/v9"
)

type ClientOptions struct {
	clo   *redis.Options
	csclo *redis.ClusterOptions
}

// NewClientOptions creates a new ClientOptions instance.
func NewClientOptions() ClientOptions {
	return ClientOptions{
		clo: &redis.Options{},
	}
}

func (co ClientOptions) SetOptions(opts config.RedisConfig) ClientOptions {
	// Only standalone mode is supported
	co.clo.Addr = opts.Host
	co.clo.MinIdleConns = opts.MinIdleConns
	co.clo.PoolSize = opts.PoolSize
	co.clo.PoolTimeout = time.Duration(opts.PoolTimeout) * time.Second
	co.clo.Password = opts.Password
	co.clo.DB = opts.DB
	return co
}

// SetDB sets the Redis database number for standalone mode.
// Note: Cluster mode does not support database selection.
func (co ClientOptions) SetDB(db int) ClientOptions {
	if co.clo != nil {
		co.clo.DB = db
	}
	return co
}
