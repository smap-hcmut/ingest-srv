package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Lua script để verify và delete lock atomically
// Script này đảm bảo chỉ delete lock nếu lockValue khớp
// Tránh unlock nhầm lock của process khác
const unlockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end
`

// lockStore lưu lockValue cho mỗi lock key để verify khi unlock
type lockStore struct {
	mu    sync.RWMutex
	locks map[string]string // map[lockKey]lockValue
}

var globalLockStore = &lockStore{
	locks: make(map[string]string),
}

func (ls *lockStore) set(key, value string) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.locks[key] = value
}

func (ls *lockStore) get(key string) (string, bool) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	value, ok := ls.locks[key]
	return value, ok
}

func (ls *lockStore) delete(key string) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	delete(ls.locks, key)
}

// Connect creates a new Redis client from ClientOptions.
func Connect(opts ClientOptions) (Client, error) {
	if opts.csclo != nil {
		cscl := redis.NewClusterClient(opts.csclo)
		return &redisClient{cscl: cscl}, nil
	}
	cl := redis.NewClient(opts.clo)
	return &redisClient{cl: cl}, nil
}

// Database interface for accessing Redis client.
type Database interface {
	Client() Client
}

// Client defines the Redis client interface for storage and pub/sub operations.
type Client interface {
	// Connection
	Disconnect() error
	Ping(ctx context.Context) error
	IsReady(ctx context.Context) bool

	// Basic Key-Value Operations
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value interface{}, expiration int) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, expiration int) error

	// Hash Operations
	HSet(ctx context.Context, key string, field string, value interface{}) error
	HGet(ctx context.Context, key string, field string) ([]byte, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HIncrBy(ctx context.Context, key string, field string, incr int64) (int64, error)

	// Multiple Key Operations
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)

	// Distributed Lock
	Lock(ctx context.Context, key string, expiration int) (bool, error)
	Unlock(ctx context.Context, key string) error

	// Pub/Sub Operations
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub

	// Pipeline Operations
	Pipeline() redis.Pipeliner
	TxPipeline() redis.Pipeliner
}

// redisClient implements Client interface supporting both standalone and cluster modes.
type redisClient struct {
	cl   *redis.Client
	cscl *redis.ClusterClient
}

// Disconnect closes the Redis connection.
func (rc *redisClient) Disconnect() error {
	if rc.cscl != nil {
		return rc.cscl.Close()
	}
	return rc.cl.Close()
}

// Ping checks the Redis connection.
func (rc *redisClient) Ping(ctx context.Context) error {
	if rc.cscl != nil {
		return rc.cscl.Ping(ctx).Err()
	}
	return rc.cl.Ping(ctx).Err()
}

// IsReady returns true if the client is connected and ready.
func (rc *redisClient) IsReady(ctx context.Context) bool {
	return rc.Ping(ctx) == nil
}

// --- Basic Key-Value Operations ---

// Get returns the value of key.
func (rc *redisClient) Get(ctx context.Context, key string) ([]byte, error) {
	if rc.cscl != nil {
		key = fmt.Sprintf("%s%s", PREFIX, key)
		return rc.cscl.Get(ctx, key).Bytes()
	}
	return rc.cl.Get(ctx, key).Bytes()
}

// Set sets key to hold the string value with expiration in seconds.
func (rc *redisClient) Set(ctx context.Context, key string, value interface{}, expiration int) error {
	if rc.cscl != nil {
		key = fmt.Sprintf("%s%s", PREFIX, key)
		return rc.cscl.Set(ctx, key, value, time.Second*time.Duration(expiration)).Err()
	}
	return rc.cl.Set(ctx, key, value, time.Second*time.Duration(expiration)).Err()
}

// Del removes the specified keys.
func (rc *redisClient) Del(ctx context.Context, keys ...string) error {
	if rc.cscl != nil {
		for i, key := range keys {
			keys[i] = fmt.Sprintf("%s%s", PREFIX, key)
		}
		return rc.cscl.Del(ctx, keys...).Err()
	}
	return rc.cl.Del(ctx, keys...).Err()
}

// Exists returns if key exists.
func (rc *redisClient) Exists(ctx context.Context, key string) (bool, error) {
	var result int64
	var err error
	if rc.cscl != nil {
		key = fmt.Sprintf("%s%s", PREFIX, key)
		result, err = rc.cscl.Exists(ctx, key).Result()
	} else {
		result, err = rc.cl.Exists(ctx, key).Result()
	}
	return result > 0, err
}

// Expire sets a timeout on key in seconds.
func (rc *redisClient) Expire(ctx context.Context, key string, expiration int) error {
	if rc.cscl != nil {
		key = fmt.Sprintf("%s%s", PREFIX, key)
		return rc.cscl.Expire(ctx, key, time.Second*time.Duration(expiration)).Err()
	}
	return rc.cl.Expire(ctx, key, time.Second*time.Duration(expiration)).Err()
}

// --- Hash Operations ---

// HSet sets field in the hash stored at key to value.
func (rc *redisClient) HSet(ctx context.Context, key string, field string, value interface{}) error {
	if rc.cscl != nil {
		key = fmt.Sprintf("%s%s", PREFIX, key)
		return rc.cscl.HSet(ctx, key, field, value).Err()
	}
	return rc.cl.HSet(ctx, key, field, value).Err()
}

// HGet returns the value associated with field in the hash stored at key.
func (rc *redisClient) HGet(ctx context.Context, key string, field string) ([]byte, error) {
	if rc.cscl != nil {
		key = fmt.Sprintf("%s%s", PREFIX, key)
		return rc.cscl.HGet(ctx, key, field).Bytes()
	}
	return rc.cl.HGet(ctx, key, field).Bytes()
}

// HGetAll returns all fields and values of the hash stored at key.
func (rc *redisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if rc.cscl != nil {
		key = fmt.Sprintf("%s%s", PREFIX, key)
		return rc.cscl.HGetAll(ctx, key).Result()
	}
	return rc.cl.HGetAll(ctx, key).Result()
}

// HIncrBy increments the number stored at field in the hash by increment.
func (rc *redisClient) HIncrBy(ctx context.Context, key string, field string, incr int64) (int64, error) {
	if rc.cscl != nil {
		key = fmt.Sprintf("%s%s", PREFIX, key)
		return rc.cscl.HIncrBy(ctx, key, field, incr).Result()
	}
	return rc.cl.HIncrBy(ctx, key, field, incr).Result()
}

// --- Multiple Key Operations ---

// MGet returns the values of all specified keys.
func (rc *redisClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	if rc.cscl != nil {
		for i, key := range keys {
			keys[i] = fmt.Sprintf("%s%s", PREFIX, key)
		}
		return rc.cscl.MGet(ctx, keys...).Result()
	}
	return rc.cl.MGet(ctx, keys...).Result()
}

// --- Distributed Lock Operations ---

// Lock acquires a distributed lock using SET NX EX pattern.
// Returns true if lock acquired, false if already locked.
func (rc *redisClient) Lock(ctx context.Context, key string, expiration int) (bool, error) {
	lockKey := fmt.Sprintf("%slock:%s", PREFIX, key)
	// Sử dụng unique value (timestamp) để verify khi unlock
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	expirationDuration := time.Second * time.Duration(expiration)

	var result bool
	var err error

	if rc.cscl != nil {
		result, err = rc.cscl.SetNX(ctx, lockKey, lockValue, expirationDuration).Result()
	} else {
		result, err = rc.cl.SetNX(ctx, lockKey, lockValue, expirationDuration).Result()
	}

	if err != nil {
		return false, err
	}

	// Lưu lockValue vào memory để verify khi unlock
	if result {
		globalLockStore.set(lockKey, lockValue)
	}

	return result, nil
}

// Unlock releases the distributed lock.
// Sử dụng Lua script để verify lockValue trước khi delete atomically.
func (rc *redisClient) Unlock(ctx context.Context, key string) error {
	lockKey := fmt.Sprintf("%slock:%s", PREFIX, key)

	// Lấy lockValue từ memory store
	lockValue, exists := globalLockStore.get(lockKey)
	if !exists {
		// Nếu không có trong store, thử unlock trực tiếp
		if rc.cscl != nil {
			return rc.cscl.Del(ctx, lockKey).Err()
		}
		return rc.cl.Del(ctx, lockKey).Err()
	}

	// Sử dụng Lua script để verify và delete atomically
	var result interface{}
	var err error

	if rc.cscl != nil {
		result, err = rc.cscl.Eval(ctx, unlockScript, []string{lockKey}, lockValue).Result()
	} else {
		result, err = rc.cl.Eval(ctx, unlockScript, []string{lockKey}, lockValue).Result()
	}

	globalLockStore.delete(lockKey)

	if err != nil {
		return err
	}

	if deleted, ok := result.(int64); ok && deleted == 0 {
		return fmt.Errorf("lock value mismatch or lock expired for key: %s", key)
	}

	return nil
}

// --- Pub/Sub Operations ---

// Publish publishes a message to a Redis channel.
func (rc *redisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	if rc.cscl != nil {
		return rc.cscl.Publish(ctx, channel, message).Err()
	}
	return rc.cl.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to the specified channels.
func (rc *redisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	if rc.cscl != nil {
		return rc.cscl.Subscribe(ctx, channels...)
	}
	return rc.cl.Subscribe(ctx, channels...)
}

// --- Pipeline Operations ---

// Pipeline returns a pipeline for executing multiple commands.
func (rc *redisClient) Pipeline() redis.Pipeliner {
	if rc.cscl != nil {
		return rc.cscl.Pipeline()
	}
	return rc.cl.Pipeline()
}

// TxPipeline returns a transactional pipeline (MULTI/EXEC).
func (rc *redisClient) TxPipeline() redis.Pipeliner {
	if rc.cscl != nil {
		return rc.cscl.TxPipeline()
	}
	return rc.cl.TxPipeline()
}
