// Package cache 提供了用户数据的缓存功能。
// 支持内存缓存和 Redis 缓存两种实现，以及基于 Redis 的分布式锁。
package cache

import (
	// 标准库
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	// 第三方库
	"github.com/redis/go-redis/v9"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/define"
)

const (
	// LOCK_OPERATION_TIMEOUT 锁操作超时时间
	//nolint:revive // 常量使用 ALL_CAPS 符合项目规范
	LOCK_OPERATION_TIMEOUT = 5 * time.Second
)

// Locker 提供分布式锁功能，兼容 gocron.Locker 接口
type Locker struct {
	Cache     *redis.Client
	lockStore sync.Map // 存储 key -> lockValue 的映射
}

// generateLockValue 生成唯一的锁值
func generateLockValue() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Lock 获取分布式锁，兼容 gocron.Locker 接口
//
// 该函数实现了基于 Redis 的分布式锁，使用 SETNX 命令确保原子性。
// 锁的默认过期时间为 DefaultLockTime 秒，防止死锁。
//
// 参数:
//   - key: 锁的键名，用于标识不同的锁
//
// 返回:
//   - success: true 表示成功获取锁，false 表示锁已被其他进程持有
//   - err: 获取锁时发生的错误（如 Redis 连接错误）
//
// 副作用:
//   - 在 Redis 中设置锁键值对
//   - 在本地 lockStore 中存储锁值，用于后续解锁验证
//   - 使用随机生成的锁值，确保只有锁的持有者才能释放锁
//
// 注意:
//   - 锁有自动过期时间，即使进程崩溃也不会导致永久死锁
//   - 锁值存储在本地内存中，进程重启后会丢失，但锁会在过期时间后自动释放
func (s *Locker) Lock(key string) (success bool, err error) {
	lockValue, err := generateLockValue()
	if err != nil {
		return false, fmt.Errorf("生成锁值失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), LOCK_OPERATION_TIMEOUT)
	defer cancel()

	res, err := s.Cache.SetNX(ctx, key, lockValue, time.Second*define.DEFAULT_LOCK_TIME).Result()
	if err != nil {
		return false, err
	}

	if res {
		// 存储 lockValue 以便后续解锁时使用
		s.lockStore.Store(key, lockValue)
	}
	return res, nil
}

// Unlock 释放分布式锁，验证锁的所有者
//
// 该函数实现了安全的锁释放机制，使用 Lua 脚本确保原子性。
// 只有锁值匹配时才会释放锁，防止误释放其他进程的锁。
//
// 参数:
//   - key: 锁的键名，必须与 Lock 时使用的键名一致
//
// 返回:
//   - error: 释放锁时发生的错误，可能的原因包括：
//   - 锁值不匹配（锁已被其他进程持有或已过期）
//   - Redis 操作失败
//   - 锁值类型错误（内部错误）
//
// 副作用:
//   - 从 Redis 中删除锁键（仅当锁值匹配时）
//   - 从本地 lockStore 中删除锁值记录
//
// 安全机制:
//   - 使用 Lua 脚本确保检查和删除的原子性
//   - 验证锁值，防止误释放其他进程的锁
//   - 如果本地没有存储锁值，会直接删除（向后兼容，但不安全）
//
// 注意:
//   - 必须使用与 Lock 时相同的 key
//   - 如果锁已过期或被其他进程持有，会返回错误
func (s *Locker) Unlock(key string) error {
	// 获取存储的 lockValue
	value, ok := s.lockStore.LoadAndDelete(key)
	if !ok {
		// 如果没有存储的 lockValue，直接删除（向后兼容）
		ctx, cancel := context.WithTimeout(context.Background(), LOCK_OPERATION_TIMEOUT)
		defer cancel()
		return s.Cache.Del(ctx, key).Err()
	}

	lockValue, ok := value.(string)
	if !ok {
		// 类型不匹配，返回错误
		return fmt.Errorf("锁值类型错误")
	}
	ctx, cancel := context.WithTimeout(context.Background(), LOCK_OPERATION_TIMEOUT)
	defer cancel()

	// 使用 Lua 脚本确保原子性：只有锁值匹配时才删除
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	result, err := s.Cache.Eval(ctx, script, []string{key}, lockValue).Result()
	if err != nil {
		return fmt.Errorf("解锁失败: %w", err)
	}

	// 使用安全的类型断言
	if val, ok := result.(int64); !ok || val == 0 {
		return fmt.Errorf("锁值不匹配或锁已过期")
	}

	return nil
}
