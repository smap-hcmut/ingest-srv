package redis

import goredis "github.com/redis/go-redis/v9"

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// redisImpl implements IRedis using go-redis.
type redisImpl struct {
	client *goredis.Client
}
