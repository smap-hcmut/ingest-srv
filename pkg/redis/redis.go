package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Set stores a key-value pair with TTL.
func (c *redisImpl) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value by key.
func (c *redisImpl) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Delete removes keys.
func (c *redisImpl) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists.
func (c *redisImpl) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// TTL returns the remaining TTL of a key.
func (c *redisImpl) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// Close closes the Redis connection.
func (c *redisImpl) Close() error {
	return c.client.Close()
}

// Ping checks if Redis is reachable.
func (c *redisImpl) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// GetClient returns the underlying go-redis Client for advanced operations.
func (c *redisImpl) GetClient() *goredis.Client {
	return c.client
}
