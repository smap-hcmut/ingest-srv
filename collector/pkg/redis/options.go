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
	if opts.RedisStandAlone {
		co.clo.Addr = opts.RedisAddr[0]
		co.clo.MinIdleConns = opts.MinIdleConns
		co.clo.PoolSize = opts.PoolSize
		co.clo.PoolTimeout = time.Duration(opts.PoolTimeout) * time.Second
		co.clo.Password = opts.Password
		co.clo.DB = opts.DB
		return co
	}
	co.csclo = &redis.ClusterOptions{
		Addrs:        opts.RedisAddr,
		MinIdleConns: opts.MinIdleConns,
		PoolSize:     opts.PoolSize,
		PoolTimeout:  time.Duration(opts.PoolTimeout) * time.Second,
		Password:     opts.Password,
	}
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
