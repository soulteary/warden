package warden

import (
	"sync"
	"time"
)

// Cache provides thread-safe caching for user list.
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Cache struct {
	mu          sync.RWMutex
	users       []AllowListUser
	expiresAt   time.Time
	ttl         time.Duration
	now         func() time.Time
	sourceBound bool
	epoch       uint64
}

type cacheRequest struct{ epoch uint64 }

// NewCache creates a new cache instance with the specified TTL.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl: ttl,
	}
}

// Get returns the cached user list if it's still valid, nil otherwise.
func (c *Cache) Get() []AllowListUser {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.getLocked()
}

// beginRead captures the invalidation generation with the cache lookup.
func (c *Cache) beginRead() ([]AllowListUser, cacheRequest) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getLocked(), cacheRequest{epoch: c.epoch}
}

func (c *Cache) getLocked() []AllowListUser {
	if c.users == nil {
		return nil
	}

	now := c.cacheNow()
	if now.After(c.expiresAt) || (c.sourceBound && !now.Before(c.expiresAt)) {
		return nil
	}

	// Return a copy to prevent external modification
	return cloneCachedUsers(c.users)
}

// Set stores the user list in cache with expiration.
func (c *Cache) Set(users []AllowListUser) {
	c.setUntil(users, time.Time{})
}

// setUntil publishes rows and their bounded expiry under one lock. A response
// that expired while decoding cannot replace a newer concurrent cache entry.
func (c *Cache) setUntil(users []AllowListUser, deadline time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.setUntilLocked(users, deadline)
}

// commitRead rejects work started before the most recent invalidation.
func (c *Cache) commitRead(users []AllowListUser, deadline time.Time, request cacheRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if request.epoch != c.epoch {
		return NewError(ErrCodeRequestFailed, "cache invalidated during request", ErrCacheInvalidated)
	}
	if !c.setUntilLocked(users, deadline) {
		return sourceFreshnessError(ErrSourceStale)
	}
	return nil
}

func (c *Cache) setUntilLocked(users []AllowListUser, deadline time.Time) bool {
	now := c.cacheNow()
	if !deadline.IsZero() && !now.Before(deadline) {
		return false
	}
	expires := now.Add(c.ttl)
	if !deadline.IsZero() && deadline.Before(expires) {
		expires = deadline
	}
	c.users = cloneCachedUsers(users)
	c.expiresAt = expires
	c.sourceBound = !deadline.IsZero()
	return true
}

func (c *Cache) cacheNow() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// Clear clears the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.epoch++
	c.users = nil
	c.expiresAt = time.Time{}
	c.sourceBound = false
}

// cloneCachedUsers preserves the cache's non-nil outer slice and detaches Scope.
func cloneCachedUsers(users []AllowListUser) []AllowListUser {
	result := make([]AllowListUser, len(users))
	copy(result, users)
	for i := range result {
		if users[i].Scope != nil {
			result[i].Scope = append([]string{}, users[i].Scope...)
		}
	}
	return result
}
