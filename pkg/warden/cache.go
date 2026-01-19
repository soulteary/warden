package warden

import (
	"sync"
	"time"
)

// Cache provides thread-safe caching for user list.
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
	result := make([]AllowListUser, len(c.users))
	copy(result, c.users)
	return result
}

// Set stores the user list in cache with expiration.
func (c *Cache) Set(users []AllowListUser) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store a copy to prevent external modification
	c.users = make([]AllowListUser, len(users))
	copy(c.users, users)
	c.expiresAt = time.Now().Add(c.ttl)
}

// Clear clears the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.users = nil
	c.expiresAt = time.Time{}
}
