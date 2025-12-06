package redis

import (
	"smap-collector/internal/state/repository"
	pkgLog "smap-collector/pkg/log"
	pkgRedis "smap-collector/pkg/redis"
)

// RedisClientAdapter wraps pkg/redis.Client to implement RedisClient interface.
// This adapter bridges the gap between the generic Redis client and the state repository.
type redisRepository struct {
	l      pkgLog.Logger
	client pkgRedis.Client
}

// NewRedisClientAdapter creates a new adapter wrapping pkg/redis.Client.
func NewRedisRepository(l pkgLog.Logger, client pkgRedis.Client) repository.StateRepository {
	return &redisRepository{
		l:      l,
		client: client,
	}
}
