// Package cache provides user data caching functionality.
// Supports both in-memory cache and Redis cache implementations, as well as Redis-based distributed locks.
package cache

import (
	// Standard library
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	// Third-party libraries
	"github.com/redis/go-redis/v9"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
)

const (
	// LOCK_OPERATION_TIMEOUT lock operation timeout
	//nolint:revive // Constants use ALL_CAPS which conforms to project standards
	LOCK_OPERATION_TIMEOUT = 5 * time.Second
)

// Locker provides distributed lock functionality, compatible with gocron.Locker interface
// Supports both Redis distributed lock and local lock modes
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Locker struct {
	Cache     *redis.Client // Redis client, if nil then use local lock
	lockStore sync.Map      // Stores key -> lockValue mapping (only for Redis mode)
	localLock *LocalLocker  // Local lock instance (used when Redis is unavailable)
}

// generateLockValue generates unique lock value
func generateLockValue() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Lock acquires distributed lock, compatible with gocron.Locker interface
//
// This function implements Redis-based distributed lock using SETNX command to ensure atomicity.
// If Redis is unavailable, automatically falls back to local lock.
// Lock default expiration time is DefaultLockTime seconds, preventing deadlocks.
//
// Parameters:
//   - key: lock key name, used to identify different locks
//
// Returns:
//   - success: true means successfully acquired lock, false means lock is held by another process
//   - err: error occurred when acquiring lock (e.g., Redis connection error)
//
// Side effects:
//   - Sets lock key-value pair in Redis (Redis mode)
//   - Stores lock value in local lockStore for subsequent unlock verification (Redis mode)
//   - Uses randomly generated lock value to ensure only lock holder can release lock (Redis mode)
//
// Notes:
//   - Lock has automatic expiration time, even if process crashes won't cause permanent deadlock (Redis mode)
//   - Lock value is stored in local memory, will be lost after process restart, but lock will be automatically released after expiration time (Redis mode)
//   - Local lock only suitable for single-machine deployment, cannot prevent duplicate execution in multi-instance scenarios
func (s *Locker) Lock(key string) (success bool, err error) {
	// If Redis is unavailable, use local lock
	if s.Cache == nil {
		if s.localLock == nil {
			s.localLock = NewLocalLocker()
		}
		return s.localLock.Lock(key)
	}

	lockValue, err := generateLockValue()
	if err != nil {
		return false, fmt.Errorf("failed to generate lock value: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), LOCK_OPERATION_TIMEOUT)
	defer cancel()

	res, err := s.Cache.SetNX(ctx, key, lockValue, time.Second*define.DEFAULT_LOCK_TIME).Result()
	if err != nil {
		// Redis operation failed, fallback to local lock
		if s.localLock == nil {
			s.localLock = NewLocalLocker()
		}
		return s.localLock.Lock(key)
	}

	if res {
		// Store lockValue for subsequent unlock use
		s.lockStore.Store(key, lockValue)
	}
	return res, nil
}

// Unlock releases distributed lock, verifies lock owner
//
// This function implements secure lock release mechanism using Lua script to ensure atomicity.
// If Redis is unavailable, automatically falls back to local lock.
// Only releases lock when lock value matches, preventing accidental release of other process's lock.
//
// Parameters:
//   - key: lock key name, must match the key name used in Lock
//
// Returns:
//   - error: error occurred when releasing lock, possible reasons include:
//   - Lock value mismatch (lock is held by another process or has expired)
//   - Redis operation failure
//   - Lock value type error (internal error)
//
// Side effects:
//   - Deletes lock key from Redis (only when lock value matches, Redis mode)
//   - Deletes lock value record from local lockStore (Redis mode)
//
// Security mechanisms:
//   - Uses Lua script to ensure atomicity of check and delete (Redis mode)
//   - Verifies lock value to prevent accidental release of other process's lock (Redis mode)
//   - If no lock value stored locally, will directly delete (backward compatible, but not secure)
//
// Notes:
//   - Must use the same key as in Lock
//   - If lock has expired or is held by another process, will return error (Redis mode)
func (s *Locker) Unlock(key string) error {
	// If Redis is unavailable, use local lock
	if s.Cache == nil {
		if s.localLock == nil {
			s.localLock = NewLocalLocker()
		}
		return s.localLock.Unlock(key)
	}

	// Get stored lockValue
	value, ok := s.lockStore.LoadAndDelete(key)
	if !ok {
		// If no stored lockValue, directly delete (backward compatibility)
		ctx, cancel := context.WithTimeout(context.Background(), LOCK_OPERATION_TIMEOUT)
		defer cancel()
		return s.Cache.Del(ctx, key).Err()
	}

	lockValue, ok := value.(string)
	if !ok {
		// Type mismatch, return error
		return fmt.Errorf("lock value type error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), LOCK_OPERATION_TIMEOUT)
	defer cancel()

	// Use Lua script to ensure atomicity: only delete when lock value matches
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	result, err := s.Cache.Eval(ctx, script, []string{key}, lockValue).Result()
	if err != nil {
		// Redis operation failed, fallback to local lock
		if s.localLock == nil {
			s.localLock = NewLocalLocker()
		}
		return s.localLock.Unlock(key)
	}

	// Use safe type assertion
	if val, ok := result.(int64); !ok || val == 0 {
		return fmt.Errorf("lock value mismatch or lock has expired")
	}

	return nil
}
