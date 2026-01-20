// Package cache provides user data caching functionality.
// Supports both in-memory cache and Redis cache implementations, as well as Redis-based distributed locks.
//
//nolint:revive // Constants use ALL_CAPS which conforms to project standards
package cache

import (
	// Standard library
	"context"
	"encoding/json"
	"time"

	// Third-party libraries
	"github.com/redis/go-redis/v9"

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

// RedisUserCache provides Redis-based user cache
type RedisUserCache struct {
	client *redis.Client
}

// NewRedisUserCache creates a new Redis user cache
func NewRedisUserCache(client *redis.Client) *RedisUserCache {
	return &RedisUserCache{
		client: client,
	}
}

// getContext creates context with timeout
func (c *RedisUserCache) getContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), REDIS_OPERATION_TIMEOUT)
}

// Set stores user list to Redis and updates version number
func (c *RedisUserCache) Set(users []define.AllowListUser) error {
	data, err := json.Marshal(users)
	if err != nil {
		return err
	}

	ctx, cancel := c.getContext()
	defer cancel()

	// Use transaction to ensure data and version number are updated simultaneously
	pipe := c.client.Pipeline()
	pipe.Set(ctx, REDIS_CACHE_KEY, data, REDIS_CACHE_TTL)
	// Version number uses timestamp to ensure each update has a new version
	pipe.Incr(ctx, REDIS_CACHE_VERSION_KEY)
	pipe.Expire(ctx, REDIS_CACHE_VERSION_KEY, REDIS_CACHE_TTL)

	_, err = pipe.Exec(ctx)
	return err
}

// Get gets user list from Redis
func (c *RedisUserCache) Get() ([]define.AllowListUser, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	data, err := c.client.Get(ctx, REDIS_CACHE_KEY).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Cache miss, return empty slice
			return []define.AllowListUser{}, nil
		}
		return nil, err
	}

	var users []define.AllowListUser
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, err
	}

	return users, nil
}

// Exists checks if cache exists
func (c *RedisUserCache) Exists() (bool, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	count, err := c.client.Exists(ctx, REDIS_CACHE_KEY).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetVersion gets cache version number
func (c *RedisUserCache) GetVersion() (int64, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	version, err := c.client.Get(ctx, REDIS_CACHE_VERSION_KEY).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil // Return 0 when version does not exist
		}
		return 0, err
	}
	return version, nil
}

// Clear clears cache
func (c *RedisUserCache) Clear() error {
	ctx, cancel := c.getContext()
	defer cancel()

	return c.client.Del(ctx, REDIS_CACHE_KEY).Err()
}
