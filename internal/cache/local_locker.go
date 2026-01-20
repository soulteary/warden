// Package cache 提供了用户数据的缓存功能。
// 支持内存缓存和 Redis 缓存两种实现，以及基于 Redis 的分布式锁。
package cache

import (
	"sync"
)

// LocalLocker 提供本地锁功能，兼容 gocron.Locker 接口
// 适用于单机部署场景，不支持分布式环境
type LocalLocker struct {
	mu    sync.Mutex
	locks map[string]bool
}

// NewLocalLocker 创建新的本地锁实例
func NewLocalLocker() *LocalLocker {
	return &LocalLocker{
		locks: make(map[string]bool),
	}
}

// Lock 获取本地锁，兼容 gocron.Locker 接口
//
// 该函数实现了基于 sync.Mutex 的本地锁，适用于单机部署场景。
// 在分布式环境下，多个实例之间无法协调，可能导致重复执行。
//
// 参数:
//   - key: 锁的键名，用于标识不同的锁
//
// 返回:
//   - success: true 表示成功获取锁，false 表示锁已被持有
//   - err: 获取锁时发生的错误（本地锁不会返回错误）
//
// 注意:
//   - 本地锁仅适用于单机部署，多实例时无法防止重复执行
//   - 锁在进程退出时会自动释放
func (l *LocalLocker) Lock(key string) (success bool, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 如果锁已被持有，返回 false
	if l.locks[key] {
		return false, nil
	}

	// 获取锁
	l.locks[key] = true
	return true, nil
}

// Unlock 释放本地锁，兼容 gocron.Locker 接口
//
// 参数:
//   - key: 锁的键名，必须与 Lock 时使用的键名一致
//
// 返回:
//   - error: 释放锁时发生的错误（本地锁不会返回错误）
func (l *LocalLocker) Unlock(key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 释放锁
	delete(l.locks, key)
	return nil
}
