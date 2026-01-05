// Package cache 提供了用户数据的缓存功能。
// 支持内存缓存和 Redis 缓存两种实现，以及基于 Redis 的分布式锁。
package cache

import (
	// 标准库
	"context"
	"encoding/json"
	"time"

	// 第三方库
	"github.com/redis/go-redis/v9"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/define"
)

const (
	// REDIS_CACHE_KEY 用于存储用户数据的 Redis key
	REDIS_CACHE_KEY = "warden:users:cache"
	// REDIS_CACHE_VERSION_KEY 用于存储缓存版本的 Redis key
	REDIS_CACHE_VERSION_KEY = "warden:users:cache:version"
	// REDIS_CACHE_TTL Redis 缓存过期时间（1小时）
	REDIS_CACHE_TTL = 1 * time.Hour
	// REDIS_OPERATION_TIMEOUT Redis 操作超时时间
	REDIS_OPERATION_TIMEOUT = 5 * time.Second
)

// RedisUserCache 提供基于 Redis 的用户缓存
type RedisUserCache struct {
	client *redis.Client
}

// NewRedisUserCache 创建新的 Redis 用户缓存
func NewRedisUserCache(client *redis.Client) *RedisUserCache {
	return &RedisUserCache{
		client: client,
	}
}

// getContext 创建带超时的 context
func (c *RedisUserCache) getContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), REDIS_OPERATION_TIMEOUT)
}

// Set 将用户列表存储到 Redis，并更新版本号
func (c *RedisUserCache) Set(users []define.AllowListUser) error {
	data, err := json.Marshal(users)
	if err != nil {
		return err
	}

	ctx, cancel := c.getContext()
	defer cancel()

	// 使用事务确保数据 and 版本号同时更新
	pipe := c.client.Pipeline()
	pipe.Set(ctx, REDIS_CACHE_KEY, data, REDIS_CACHE_TTL)
	// 版本号使用时间戳，确保每次更新都有新版本
	pipe.Incr(ctx, REDIS_CACHE_VERSION_KEY)
	pipe.Expire(ctx, REDIS_CACHE_VERSION_KEY, REDIS_CACHE_TTL)

	_, err = pipe.Exec(ctx)
	return err
}

// Get 从 Redis 获取用户列表
func (c *RedisUserCache) Get() ([]define.AllowListUser, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	data, err := c.client.Get(ctx, REDIS_CACHE_KEY).Bytes()
	if err != nil {
		if err == redis.Nil {
			// 缓存未命中，返回空切片
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

// Exists 检查缓存是否存在
func (c *RedisUserCache) Exists() (bool, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	count, err := c.client.Exists(ctx, REDIS_CACHE_KEY).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetVersion 获取缓存版本号
func (c *RedisUserCache) GetVersion() (int64, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	version, err := c.client.Get(ctx, REDIS_CACHE_VERSION_KEY).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil // 版本不存在时返回 0
		}
		return 0, err
	}
	return version, nil
}

// Clear 清除缓存
func (c *RedisUserCache) Clear() error {
	ctx, cancel := c.getContext()
	defer cancel()

	return c.client.Del(ctx, REDIS_CACHE_KEY).Err()
}
