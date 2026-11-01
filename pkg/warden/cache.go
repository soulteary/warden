package warden

import (
	"sync"
	"time"
)

// Cache provides thread-safe caching for user list.
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Cache struct {
	mu        sync.RWMutex
	users     []AllowListUser
	expiresAt time.Time
	ttl       time.Duration
}

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

	if c.users == nil {
		return nil
	}

	if time.Now().After(c.expiresAt) {
		return nil
	}

	// Return a copy to prevent external modification
	return cloneCachedUsers(c.users)
}

// Set stores the user list in cache with expiration.
func (c *Cache) Set(users []AllowListUser) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store a copy to prevent external modification
	c.users = cloneCachedUsers(users)
	c.expiresAt = time.Now().Add(c.ttl)
}

// Clear clears the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.users = nil
	c.expiresAt = time.Time{}
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
