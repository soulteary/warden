// Package cache provides user data caching functionality.
// Supports both in-memory cache and Redis cache implementations, as well as Redis-based distributed locks.
package cache

import (
	// Standard library
	"sort"
	"strings"
	"sync"

	// External packages
	"github.com/soulteary/cli-kit/validator"
	secure "github.com/soulteary/secure-kit"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

var log = logger.GetLogger()

// slicePool reuses slices to reduce memory allocation
// slicePool has been removed because we need to return independent copies, cannot directly use pool

// SafeUserCache provides thread-safe user cache
// Uses map structure to improve lookup efficiency while maintaining API compatibility
// Maintains order list to preserve insertion order
// Caches hash value to optimize data change detection
// Maintains multiple indexes to support fast queries by phone, mail, user_id
type SafeUserCache struct { //nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
	mu       sync.RWMutex                    // 24 bytes
	order    []string                        // 24 bytes (8 pointer + 8 len + 8 cap)
	hash     string                          // 16 bytes (8 pointer + 8 len)
	users    map[string]define.AllowListUser // 8 bytes pointer (using phone as key)
	byMail   map[string]string               // 8 bytes pointer (mail -> phone mapping)
	byUserID map[string]string               // 8 bytes pointer (user_id -> phone mapping)
}

// NewSafeUserCache creates a new thread-safe user cache
func NewSafeUserCache() *SafeUserCache {
	return &SafeUserCache{
		users:    make(map[string]define.AllowListUser),
		order:    make([]string, 0),
		byMail:   make(map[string]string),
		byUserID: make(map[string]string),
	}
}

// Get gets a copy of user list (thread-safe)
// Returns slice format to maintain API compatibility
// Return order matches the order when Set was called
// Uses sync.Pool to optimize memory allocation
func (c *SafeUserCache) Get() []define.AllowListUser {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get slice from pool (if available)
	// Note: Since we need to return an independent copy, cannot directly use slice from pool
	// But we can leverage pool to reduce initial allocation
	expectedLen := len(c.order)

	// For small data, allocate directly; for large data, consider using pool
	// But since we need to return independent copy, still need to copy here
	// Optimization: pre-allocate sufficient capacity
	result := make([]define.AllowListUser, 0, expectedLen)
	for _, phone := range c.order {
		if user, exists := c.users[phone]; exists {
			result = append(result, user)
		}
	}
	return result
}

// Set sets user list (thread-safe)
// Accepts slice format, internally converts to map for storage
// Preserves input order
func (c *SafeUserCache) Set(users []define.AllowListUser) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear existing data
	c.users = make(map[string]define.AllowListUser, len(users))
	c.order = make([]string, 0, len(users))
	c.byMail = make(map[string]string, len(users))
	c.byUserID = make(map[string]string, len(users))

	// Use map to deduplicate and store (using phone as key)
	// Simultaneously maintain order list and indexes
	validCount := 0
	invalidCount := 0
	duplicateCount := 0

	for _, user := range users {
		if user.Phone == "" {
			continue
		}
		// Normalize user data (set default values, generate user_id)
		user.Normalize()

		// Validate user data using cli-kit validator
		phoneOpts := &validator.PhoneOptions{AllowEmpty: true}
		emailOpts := &validator.EmailOptions{AllowEmpty: true}

		if err := validator.ValidatePhone(user.Phone, phoneOpts); err != nil {
			invalidCount++
			log.Warn().
				Err(err).
				Str("phone", user.Phone).
				Str("mail", user.Mail).
				Str("field", "phone").
				Msg("Skipping invalid user data")
			continue
		}
		if err := validator.ValidateEmail(user.Mail, emailOpts); err != nil {
			invalidCount++
			log.Warn().
				Err(err).
				Str("phone", user.Phone).
				Str("mail", user.Mail).
				Str("field", "email").
				Msg("Skipping invalid user data")
			continue
		}
		// If phone already exists, update data but maintain first occurrence position
		if _, exists := c.users[user.Phone]; !exists {
			c.order = append(c.order, user.Phone)
			validCount++
		} else {
			duplicateCount++
			log.Debug().
				Str("phone", user.Phone).
				Msg("Updating existing user data")
		}
		c.users[user.Phone] = user

		// Build mail index (if exists)
		if user.Mail != "" {
			mailKey := strings.ToLower(strings.TrimSpace(user.Mail))
			c.byMail[mailKey] = user.Phone
		}

		// Build user_id index (if exists)
		if user.UserID != "" {
			c.byUserID[user.UserID] = user.Phone
		}
	}

	// Record validation statistics
	if invalidCount > 0 || duplicateCount > 0 {
		log.Info().
			Int("total", len(users)).
			Int("valid", validCount).
			Int("invalid", invalidCount).
			Int("duplicates", duplicateCount).
			Msg("Data validation completed")
	}

	// Calculate and cache hash value (optimize data change detection)
	c.hash = calculateHashInternal(c.users, c.order)
}

// calculateHashInternal calculates hash value of user data (internal method)
// Uses map and order list to avoid creating temporary slices
func calculateHashInternal(users map[string]define.AllowListUser, order []string) string {
	if len(users) == 0 {
		return secure.GetSHA256Hash("empty")
	}

	// Create sorted user list for hash calculation
	sortedUsers := make([]define.AllowListUser, 0, len(order))
	for _, phone := range order {
		if user, exists := users[phone]; exists {
			sortedUsers = append(sortedUsers, user)
		}
	}

	// Sort user list to ensure same data produces same hash
	sort.Slice(sortedUsers, func(i, j int) bool {
		if sortedUsers[i].Phone != sortedUsers[j].Phone {
			return sortedUsers[i].Phone < sortedUsers[j].Phone
		}
		return sortedUsers[i].Mail < sortedUsers[j].Mail
	})

	// Calculate hash (includes all fields to ensure accurate data change detection)
	var sb strings.Builder
	for _, user := range sortedUsers {
		scopeStr := strings.Join(user.Scope, ",")
		sb.WriteString(user.Phone + ":" + user.Mail + ":" + user.UserID + ":" + user.Status + ":" + scopeStr + ":" + user.Role + "\n")
	}
	return secure.GetSHA256Hash(sb.String())
}

// Len gets user count (thread-safe)
func (c *SafeUserCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.users)
}

// GetByPhone gets user by phone number (thread-safe, O(1) lookup)
func (c *SafeUserCache) GetByPhone(phone string) (define.AllowListUser, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	user, exists := c.users[phone]
	return user, exists
}

// GetByMail gets user by email (thread-safe, O(1) lookup)
func (c *SafeUserCache) GetByMail(mail string) (define.AllowListUser, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	mailKey := strings.ToLower(strings.TrimSpace(mail))
	phone, exists := c.byMail[mailKey]
	if !exists {
		return define.AllowListUser{}, false
	}
	user, exists := c.users[phone]
	return user, exists
}

// GetByUserID gets user by user ID (thread-safe, O(1) lookup)
func (c *SafeUserCache) GetByUserID(userID string) (define.AllowListUser, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	phone, exists := c.byUserID[userID]
	if !exists {
		return define.AllowListUser{}, false
	}
	user, exists := c.users[phone]
	return user, exists
}

// Iterate iterates all users, avoiding copying entire slice (thread-safe)
// Callback function receives user data in insertion order
// If callback function returns false, iteration will stop
func (c *SafeUserCache) Iterate(fn func(user define.AllowListUser) bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, phone := range c.order {
		if user, exists := c.users[phone]; exists {
			if !fn(user) {
				return
			}
		}
	}
}

// GetReadOnly gets read-only view (actually returns copy, but semantically represents read-only)
// For read-only scenarios, recommend using Iterate method to avoid copying
func (c *SafeUserCache) GetReadOnly() []define.AllowListUser {
	return c.Get()
}

// GetHash gets cached hash value (thread-safe)
// If hash value is not calculated, returns empty string
// Using cached hash value can avoid redundant calculations and improve performance
func (c *SafeUserCache) GetHash() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hash
}
