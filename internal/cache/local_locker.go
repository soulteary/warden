// Package cache provides user data caching functionality.
// Supports both in-memory cache and Redis cache implementations, as well as Redis-based distributed locks.
package cache

import (
	"sync"
)

// LocalLocker provides local lock functionality, compatible with gocron.Locker interface
// Suitable for single-machine deployment scenarios, does not support distributed environments
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type LocalLocker struct {
	mu    sync.Mutex
	locks map[string]bool
}

// NewLocalLocker creates a new local lock instance
func NewLocalLocker() *LocalLocker {
	return &LocalLocker{
		locks: make(map[string]bool),
	}
}

// Lock acquires local lock, compatible with gocron.Locker interface
//
// This function implements sync.Mutex-based local lock, suitable for single-machine deployment scenarios.
// In distributed environments, multiple instances cannot coordinate, may lead to duplicate execution.
//
// Parameters:
//   - key: lock key name, used to identify different locks
//
// Returns:
//   - success: true means successfully acquired lock, false means lock is already held
//   - err: error occurred when acquiring lock (local lock will not return error)
//
// Notes:
//   - Local lock only suitable for single-machine deployment, cannot prevent duplicate execution in multi-instance scenarios
//   - Lock will be automatically released when process exits
func (l *LocalLocker) Lock(key string) (success bool, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// If lock is already held, return false
	if l.locks[key] {
		return false, nil
	}

	// Acquire lock
	l.locks[key] = true
	return true, nil
}

// Unlock releases local lock, compatible with gocron.Locker interface
//
// Parameters:
//   - key: lock key name, must match the key name used in Lock
//
// Returns:
//   - error: error occurred when releasing lock (local lock will not return error)
func (l *LocalLocker) Unlock(key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Release lock
	delete(l.locks, key)
	return nil
}
