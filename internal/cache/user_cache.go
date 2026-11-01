// Package cache provides user data caching functionality.
// Supports both in-memory cache and Redis cache implementations, as well as Redis-based distributed locks.
package cache

import (
	// Standard library
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	// External packages
	cache "github.com/soulteary/cache-kit/v2"
	"github.com/soulteary/cli-kit/validator"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
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
	current atomic.Pointer[ReadView]
}

// NewSafeUserCache creates a new thread-safe user cache
func NewSafeUserCache() *SafeUserCache {
	c := &SafeUserCache{}
	c.current.Store(&ReadView{cache: newUserIndexes(validateUser)})
	return c
}

func newUserIndexes(validate func(define.AllowListUser) error) *cache.MemoryCache[define.AllowListUser] {
	// Primary key: phone if non-empty, else normalized mail (supports email-only users)
	config := cache.DefaultConfig[define.AllowListUser]().
		WithPrimaryKey(primaryKeyForUser).
		WithValidateFunc(validate).
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

	return c
}

// Stable, non-PII reason codes for a record rejected by per-record format validation.
const (
	reasonBothIdentifiersEmpty = "both_identifiers_empty"
	reasonInvalidPhone         = "invalid_phone"
	reasonInvalidMail          = "invalid_email"
)

// checkUser applies the per-record format rules and returns a stable reason code plus the
// underlying validator error. It logs nothing, so it can also answer "would this record be
// accepted?" for callers that must not emit a second round of rejection warnings — see
// AcceptableCount. validateUser is the logging wrapper used as the cache's ValidateFunc,
// which keeps a single definition of the rules for both callers.
//
//nolint:gocritic // hugeParam: mirrors the cache.ValidateFunc signature used by validateUser
func checkUser(user define.AllowListUser) (reason string, err error) {
	phoneTrim := strings.TrimSpace(user.Phone)
	mailTrim := strings.TrimSpace(user.Mail)
	if phoneTrim == "" && mailTrim == "" {
		return reasonBothIdentifiersEmpty, errBothIdentifierEmpty
	}
	phoneOpts := &validator.PhoneOptions{AllowEmpty: true}
	emailOpts := &validator.EmailOptions{AllowEmpty: true}

	if err := validator.ValidatePhone(user.Phone, phoneOpts); err != nil {
		return reasonInvalidPhone, err
	}
	if err := validator.ValidateEmail(user.Mail, emailOpts); err != nil {
		return reasonInvalidMail, err
	}
	return "", nil
}

// validateUser validates user data using cli-kit validator and logs rejections.
// At least one of phone or mail must be non-empty (email-only users are supported).
//
//nolint:gocritic // hugeParam: function signature must match cache.ValidateFunc interface
func validateUser(user define.AllowListUser) error {
	reason, err := checkUser(user)
	if err == nil {
		return nil
	}
	if reason == reasonBothIdentifiersEmpty {
		log.Warn().Msg("Skipping user with both phone and mail empty")
		return err
	}
	// NOTE: the validator error message can embed the raw phone/mail value (e.g. "%q
	// does not match"), so we must NOT log err directly. Log a stable, non-PII reason
	// code plus masked identifiers only.
	field := "phone"
	if reason == reasonInvalidMail {
		field = "email"
	}
	log.Warn().
		Str("reason", reason).
		Str("phone", identity.MaskPhone(user.Phone)).
		Str("mail", identity.MaskMail(user.Mail)).
		Str("field", field).
		Msg("Skipping invalid user data")
	return err
}

// AcceptableCount reports how many records of users would survive a Set: how large the
// effective rule set becomes once empty-identifier records are dropped, per-record format
// validation has run, and duplicates sharing a primary key have collapsed.
//
// It exists so a caller can learn "this load would leave the cache empty" BEFORE mutating
// the cache. Set is the single atomic swap point, so probing by writing and inspecting the
// result would expose an empty rule set to concurrent readers for as long as the caller
// takes to undo it. AcceptableCount answers the same question with no window at all.
//
// It deliberately stays silent: the records it inspects are validated again for real by
// Set, which logs every rejection. Counting here must not duplicate those warnings.
//
// The steps mirror SafeUserCache.Set and the cache-kit Set it delegates to, in the same
// order (normalize, validate, primary key, dedup); acceptable_count_test.go pins the two
// against each other so they cannot drift apart.
func AcceptableCount(users []define.AllowListUser) int {
	if len(users) == 0 {
		return 0
	}
	accepted := make(map[string]struct{}, len(users))
	for i := range users {
		// Set drops records whose raw primary key is empty before handing the rest to
		// cache-kit; mirror that first so both filters are represented.
		if primaryKeyForUser(users[i]) == "" {
			continue
		}
		normalized := normalizeUser(users[i])
		if _, err := checkUser(normalized); err != nil {
			continue
		}
		key := primaryKeyForUser(normalized)
		if key == "" {
			continue
		}
		accepted[key] = struct{}{}
	}
	return len(accepted)
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
		sum := sha256.Sum256([]byte("empty"))
		return hex.EncodeToString(sum[:])
	}
	sorted := make([]struct {
		key  string
		user define.AllowListUser
	}, len(users))
	for i := range users {
		sorted[i].user = users[i]
		sorted[i].user.Normalize()
		sorted[i].key = primaryKeyForUser(sorted[i].user)
	}
	// Keep the previous sort and comparator semantics for equal keys. Only the
	// repeated key calculation moves out of the comparison callback.
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].key < sorted[j].key })
	digest := sha256.New()
	// Bound temporary encoding storage without reserving 4 KiB for tiny lists.
	bufferSize := 4096
	if len(sorted) < 32 {
		bufferSize = len(sorted) * 128
	}
	encoded := bufio.NewWriterSize(digest, bufferSize)
	var prefix [21]byte
	writeField := func(value string) {
		framedLength := strconv.AppendInt(prefix[:0], int64(len(value)), 10)
		framedLength = append(framedLength, ':')
		// SHA256's Write never returns an error, including through this buffer.
		_, _ = encoded.Write(framedLength) //nolint:errcheck // The underlying SHA256 writer cannot fail.
		_, _ = encoded.WriteString(value)  //nolint:errcheck // The underlying SHA256 writer cannot fail.
	}
	for i := range sorted {
		writeField(sorted[i].user.Phone)
		writeField(sorted[i].user.Mail)
		writeField(sorted[i].user.UserID)
		writeField(sorted[i].user.Status)
		writeField(strconv.Itoa(len(sorted[i].user.Scope)))
		for _, scope := range sorted[i].user.Scope {
			writeField(scope)
		}
		writeField(sorted[i].user.Role)
		writeField(sorted[i].user.Name)
		writeField(sorted[i].user.DingtalkUserID)
	}
	_ = encoded.Flush() //nolint:errcheck // The underlying SHA256 writer cannot fail.
	var sum [sha256.Size]byte
	return hex.EncodeToString(digest.Sum(sum[:0]))
}

// hashUsers calculates hash value of user data for change detection (cache-kit callback)
func hashUsers(users []define.AllowListUser) string {
	return HashUserList(users)
}

// Get gets a copy of user list (thread-safe)
// Returns slice format to maintain API compatibility
// Return order matches the order when Set was called
func (c *SafeUserCache) Get() []define.AllowListUser { return c.ReadView().Get() }

// ReadView pins the current immutable generation for a multi-step read.
func (c *SafeUserCache) ReadView() *ReadView { return c.current.Load() }

// Set preserves the existing unvalidated-set API and insertion order. A new
// generation is built privately, then published atomically.
func (c *SafeUserCache) Set(users []define.AllowListUser) {
	c.current.Store(buildReadView(users, true))
}

func buildReadView(users []define.AllowListUser, report bool) *ReadView {
	indexes := newUserIndexes(func(u define.AllowListUser) error { //nolint:gocritic // hugeParam: cache-kit callback
		_, err := checkUser(u)
		return err
	})
	valid := make([]define.AllowListUser, 0, len(users))
	for i := range users {
		if primaryKeyForUser(users[i]) != "" {
			valid = append(valid, cloneUser(users[i]))
		}
	}
	indexes.Set(valid)
	view := &ReadView{cache: indexes, rows: indexes.GetAll()}
	if report {
		reportSet(users, view)
	}
	return view
}

func reportSet(users []define.AllowListUser, view *ReadView) {
	beforeLen := 0
	for i := range users {
		if primaryKeyForUser(users[i]) == "" {
			continue
		}
		beforeLen++
		// Match cache-kit's normalization before reporting a rejected record.
		if err := validateUser(normalizeUser(users[i])); err != nil {
			continue
		}
	}
	invalidCount := len(users) - beforeLen
	duplicateCount := beforeLen - view.Len()
	if duplicateCount < 0 {
		duplicateCount = 0
	}
	if invalidCount > 0 || duplicateCount > 0 {
		log.Info().Int("total", len(users)).Int("valid", view.Len()).Int("invalid", invalidCount).Int("duplicates", duplicateCount).Msg("Data validation completed")
	}
}

// SetValidated runs centralized identity validation (identifier uniqueness, optional
// explicit user_id requirement) BEFORE swapping the data into the shared cache. On a
// uniqueness/missing-id conflict it returns a typed error and leaves the existing cache
// contents untouched, so a bad remote/local rule set can never overwrite good data with
// a partially-deduped ("last-write-wins") set. On success the cache is replaced with the
// deterministically-ordered validated set.
func (c *SafeUserCache) SetValidated(users []define.AllowListUser, opts identity.Options) error {
	prepared, err := PrepareRules(users, opts)
	if err != nil {
		return err
	}
	c.ApplyPrepared(prepared)
	return nil
}

// Len gets user count (thread-safe)
func (c *SafeUserCache) Len() int {
	return c.ReadView().Len()
}

// GetByPhone gets user by phone number (thread-safe, O(1) lookup)
func (c *SafeUserCache) GetByPhone(phone string) (define.AllowListUser, bool) {
	// Primary key is trimmed phone (when user has phone), so normalize lookup
	return c.ReadView().GetByPhone(phone)
}

// GetByMail gets user by email (thread-safe, O(1) lookup)
func (c *SafeUserCache) GetByMail(mail string) (define.AllowListUser, bool) {
	// Index uses normalized mail (lowercase, trimmed)
	return c.ReadView().GetByMail(mail)
}

// GetByUserID gets user by user ID (thread-safe, O(1) lookup)
func (c *SafeUserCache) GetByUserID(userID string) (define.AllowListUser, bool) {
	return c.ReadView().GetByUserID(userID)
}

// Iterate iterates all users, avoiding copying entire slice (thread-safe)
// Callback function receives user data in insertion order
// If callback function returns false, iteration will stop
func (c *SafeUserCache) Iterate(fn func(user define.AllowListUser) bool) {
	c.ReadView().Iterate(fn)
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
	return c.ReadView().GetHash()
}
