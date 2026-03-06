package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// IRedis defines the interface for Redis operations.
// Implementations are safe for concurrent use.
type IRedis interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Close() error
	Ping(ctx context.Context) error
	GetClient() *goredis.Client
}

// New creates a new Redis client. Returns an implementation of IRedis.
func NewRedis(cfg RedisConfig) (IRedis, error) {
	if cfg.Host == "" {
		return nil, ErrHostRequired
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, ErrInvalidPort
	}

	client := goredis.NewClient(&goredis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &redisImpl{client: client}, nil
}
