// Package cache provides user data caching functionality.
// Supports both in-memory cache and Redis cache implementations, as well as Redis-based distributed locks.
package cache

import (
	// Standard library
	"errors"
	"sort"
	"strings"

	// External packages
	cache "github.com/soulteary/cache-kit"
	"github.com/soulteary/cli-kit/validator"
	secure "github.com/soulteary/secure-kit"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

var log = logger.GetLoggerKit()

// errBothIdentifierEmpty is returned when user has neither phone nor mail.
var errBothIdentifierEmpty = errors.New("at least one of phone or mail required")

// Index names for multi-index cache
const (
	IndexPhone  = "phone"
	IndexMail   = "mail"
	IndexUserID = "user_id"
)

// primaryKeyForUser returns the dedup/primary key: phone if non-empty, else normalized mail.
// Matches loader's allowListUserKey so cache and merge strategy stay consistent.
//
//nolint:gocritic // hugeParam: signature must match cache-kit PrimaryKeyFunc[T](T)(string)
func primaryKeyForUser(u define.AllowListUser) string {
	k := strings.TrimSpace(u.Phone)
	if k == "" {
		k = strings.ToLower(strings.TrimSpace(u.Mail))
	}
	return k
}

// SafeUserCache provides thread-safe user cache using cache-kit MultiIndexCache
// Maintains multiple indexes to support fast queries by phone, mail, user_id
type SafeUserCache struct {
	cache *cache.MemoryCache[define.AllowListUser]
}

// NewSafeUserCache creates a new thread-safe user cache
func NewSafeUserCache() *SafeUserCache {
	// Primary key: phone if non-empty, else normalized mail (supports email-only users)
	config := cache.DefaultConfig[define.AllowListUser]().
		WithPrimaryKey(primaryKeyForUser).
		WithValidateFunc(validateUser).
		WithNormalizeFunc(normalizeUser).
		WithHashFunc(hashUsers)

	c := cache.NewMultiIndexCache(config)

	// Add indexes for mail and user_id lookups
	c.AddIndex(IndexMail, func(u define.AllowListUser) string {
		return strings.ToLower(strings.TrimSpace(u.Mail))
	})
	c.AddIndex(IndexUserID, func(u define.AllowListUser) string {
		return u.UserID
	})

	return &SafeUserCache{
		cache: c,
	}
}

// validateUser validates user data using cli-kit validator.
// At least one of phone or mail must be non-empty (email-only users are supported).
//
//nolint:gocritic // hugeParam: function signature must match cache.ValidateFunc interface
func validateUser(user define.AllowListUser) error {
	phoneTrim := strings.TrimSpace(user.Phone)
	mailTrim := strings.TrimSpace(user.Mail)
	if phoneTrim == "" && mailTrim == "" {
		log.Warn().Msg("Skipping user with both phone and mail empty")
		return errBothIdentifierEmpty
	}
	phoneOpts := &validator.PhoneOptions{AllowEmpty: true}
	emailOpts := &validator.EmailOptions{AllowEmpty: true}

	if err := validator.ValidatePhone(user.Phone, phoneOpts); err != nil {
		log.Warn().
			Err(err).
			Str("phone", user.Phone).
			Str("mail", user.Mail).
			Str("field", "phone").
			Msg("Skipping invalid user data")
		return err
	}
	if err := validator.ValidateEmail(user.Mail, emailOpts); err != nil {
		log.Warn().
			Err(err).
			Str("phone", user.Phone).
			Str("mail", user.Mail).
			Str("field", "email").
			Msg("Skipping invalid user data")
		return err
	}
	return nil
}

// normalizeUser normalizes user data
//
//nolint:gocritic // hugeParam: function signature must match cache.NormalizeFunc interface
func normalizeUser(user define.AllowListUser) define.AllowListUser {
	user.Normalize()
	return user
}

// HashUserList computes SHA256 hash of user list for change detection.
// Sorts by primary key (phone or mail), normalizes each user, then hashes.
// Used by SafeUserCache internally and by main for backgroundTask comparison.
func HashUserList(users []define.AllowListUser) string {
	if len(users) == 0 {
		return secure.GetSHA256Hash("empty")
	}
	sorted := make([]define.AllowListUser, len(users))
	copy(sorted, users)
	for i := range sorted {
		sorted[i].Normalize()
	}
	sort.Slice(sorted, func(i, j int) bool {
		ki, kj := primaryKeyForUser(sorted[i]), primaryKeyForUser(sorted[j])
		return ki < kj
	})
	var sb strings.Builder
	for _, user := range sorted {
		scopeStr := strings.Join(user.Scope, ",")
		sb.WriteString(user.Phone + ":" + user.Mail + ":" + user.UserID + ":" + user.Status + ":" + scopeStr + ":" + user.Role + "\n")
	}
	return secure.GetSHA256Hash(sb.String())
}

// hashUsers calculates hash value of user data for change detection (cache-kit callback)
func hashUsers(users []define.AllowListUser) string {
	return HashUserList(users)
}

// Get gets a copy of user list (thread-safe)
// Returns slice format to maintain API compatibility
// Return order matches the order when Set was called
func (c *SafeUserCache) Get() []define.AllowListUser {
	return c.cache.GetAll()
}

// Set sets user list (thread-safe)
// Accepts slice format, internally converts to map for storage
// Preserves input order. Keeps users with at least one of phone or mail (email-only users supported).
func (c *SafeUserCache) Set(users []define.AllowListUser) {
	// Filter out users with both phone and mail empty
	validUsers := make([]define.AllowListUser, 0, len(users))
	for _, user := range users {
		if primaryKeyForUser(user) != "" {
			validUsers = append(validUsers, user)
		}
	}

	// Track statistics for logging
	invalidCount := len(users) - len(validUsers)
	beforeLen := len(validUsers)

	c.cache.Set(validUsers)

	afterLen := c.cache.Len()
	duplicateCount := beforeLen - afterLen - invalidCount
	if duplicateCount < 0 {
		duplicateCount = 0
	}

	// The cache internally handles validation and deduplication
	// Log summary if there were any issues
	if invalidCount > 0 || duplicateCount > 0 {
		log.Info().
			Int("total", len(users)).
			Int("valid", afterLen).
			Int("invalid", invalidCount).
			Int("duplicates", duplicateCount).
			Msg("Data validation completed")
	}
}

// Len gets user count (thread-safe)
func (c *SafeUserCache) Len() int {
	return c.cache.Len()
}

// GetByPhone gets user by phone number (thread-safe, O(1) lookup)
func (c *SafeUserCache) GetByPhone(phone string) (define.AllowListUser, bool) {
	// Primary key is trimmed phone (when user has phone), so normalize lookup
	return c.cache.Get(strings.TrimSpace(phone))
}

// GetByMail gets user by email (thread-safe, O(1) lookup)
func (c *SafeUserCache) GetByMail(mail string) (define.AllowListUser, bool) {
	// Index uses normalized mail (lowercase, trimmed)
	return c.cache.GetByIndex(IndexMail, strings.ToLower(strings.TrimSpace(mail)))
}

// GetByUserID gets user by user ID (thread-safe, O(1) lookup)
func (c *SafeUserCache) GetByUserID(userID string) (define.AllowListUser, bool) {
	return c.cache.GetByIndex(IndexUserID, userID)
}

// Iterate iterates all users, avoiding copying entire slice (thread-safe)
// Callback function receives user data in insertion order
// If callback function returns false, iteration will stop
func (c *SafeUserCache) Iterate(fn func(user define.AllowListUser) bool) {
	c.cache.Iterate(fn)
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
	return c.cache.GetHash()
}
