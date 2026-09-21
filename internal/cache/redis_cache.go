// Package cache provides user data caching functionality.
// Supports both in-memory cache and Redis cache implementations, as well as Redis-based distributed locks.
//
//nolint:revive // Constants use ALL_CAPS which conforms to project standards
package cache

import (
	// Standard library
	"context"
	"time"

	// Third-party libraries
	"github.com/redis/go-redis/v9"
	rediscache "github.com/soulteary/cache-kit/v2/rediscache"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
)

const (
	// REDIS_CACHE_KEY Redis key for storing user data
	REDIS_CACHE_KEY = "warden:users:cache"
	// REDIS_CACHE_VERSION_KEY Redis key for storing cache version
	REDIS_CACHE_VERSION_KEY = "warden:users:cache:version"
	// REDIS_CACHE_TTL Redis cache expiration time (1 hour)
	REDIS_CACHE_TTL = 1 * time.Hour
	// REDIS_OPERATION_TIMEOUT Redis operation timeout
	REDIS_OPERATION_TIMEOUT = 5 * time.Second
)

// RedisUserCache provides Redis-based user cache using cache-kit
type RedisUserCache struct {
	cache *rediscache.Cache[define.AllowListUser]
}

// NewRedisUserCache creates a new Redis user cache
func NewRedisUserCache(client *redis.Client) *RedisUserCache {
	config := rediscache.DefaultConfig().
		WithKeyPrefix(""). // We use custom key name directly
		WithTTL(REDIS_CACHE_TTL).
		WithOperationTimeout(REDIS_OPERATION_TIMEOUT).
		WithVersionKeySuffix(":version")

	return &RedisUserCache{
		cache: rediscache.NewWithKey[define.AllowListUser](client, REDIS_CACHE_KEY, config),
	}
}

// Set stores user list to Redis and updates version number
func (c *RedisUserCache) Set(users []define.AllowListUser) error {
	return c.cache.Set(context.Background(), users)
}

// Get gets user list from Redis
func (c *RedisUserCache) Get() ([]define.AllowListUser, error) {
	return c.cache.Get(context.Background())
}

// Exists checks if cache exists
func (c *RedisUserCache) Exists() (bool, error) {
	return c.cache.Exists(context.Background())
}

// GetVersion gets cache version number
func (c *RedisUserCache) GetVersion() (int64, error) {
	return c.cache.GetVersion(context.Background())
}

// Clear clears cache
func (c *RedisUserCache) Clear() error {
	return c.cache.Clear(context.Background())
}
