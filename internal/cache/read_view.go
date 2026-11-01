package cache

import (
	"strings"

	cachekit "github.com/soulteary/cache-kit/v2"
	"github.com/soulteary/warden/internal/define"
)

// ReadView is one immutable generation of effective users and their indexes.
// Values returned to callers own their Scope slices; none can mutate this view.
type ReadView struct {
	cache *cachekit.MemoryCache[define.AllowListUser]
	// rows is the cache's effective insertion order, captured once before publication.
	// Its Scope slices are shared only with the private immutable indexes.
	rows []define.AllowListUser
}

// ViewSource supplies a generation that a consumer can retain for an entire read.
type ViewSource interface {
	ReadView() *ReadView
}

func cloneUser(user define.AllowListUser) define.AllowListUser { //nolint:gocritic // hugeParam: copies the record before detaching its only slice
	if user.Scope != nil {
		user.Scope = append([]string{}, user.Scope...)
	}
	return user
}

func cloneUsers(users []define.AllowListUser) []define.AllowListUser {
	if users == nil {
		return nil
	}
	result := make([]define.AllowListUser, len(users))
	for i := range users {
		result[i] = cloneUser(users[i])
	}
	return result
}

// Get returns a detached copy in the existing insertion order.
func (v *ReadView) Get() []define.AllowListUser {
	// make preserves the non-nil empty result of GetAll for the initial view.
	users := make([]define.AllowListUser, len(v.rows))
	for i := range v.rows {
		users[i] = cloneUser(v.rows[i])
	}
	return users
}

// Len returns the number of effective users.
func (v *ReadView) Len() int { return v.cache.Len() }

// GetHash returns the version of the effective, normalized user set.
func (v *ReadView) GetHash() string { return v.cache.GetHash() }

// GetByPhone looks up a trimmed primary key.
func (v *ReadView) GetByPhone(phone string) (define.AllowListUser, bool) {
	u, ok := v.cache.Get(strings.TrimSpace(phone))
	return cloneUser(u), ok
}

// GetByMail looks up a normalized email index.
func (v *ReadView) GetByMail(mail string) (define.AllowListUser, bool) {
	u, ok := v.cache.GetByIndex(IndexMail, strings.ToLower(strings.TrimSpace(mail)))
	return cloneUser(u), ok
}

// GetByUserID looks up the existing user ID index without changing its semantics.
func (v *ReadView) GetByUserID(userID string) (define.AllowListUser, bool) {
	u, ok := v.cache.GetByIndex(IndexUserID, userID)
	return cloneUser(u), ok
}

// Iterate visits detached records in the existing insertion order.
func (v *ReadView) Iterate(fn func(define.AllowListUser) bool) {
	for i := range v.rows {
		if !fn(cloneUser(v.rows[i])) {
			return
		}
	}
}

// Page returns detached rows from this generation, together with its totals.
// It copies only the requested page, including Scope, without scanning other rows.
// Invalid page sizes return an empty page and zero total pages.
func (v *ReadView) Page(page, pageSize int) (rows []define.AllowListUser, total, totalPages int) {
	total = len(v.rows)
	if total == 0 || pageSize <= 0 {
		return []define.AllowListUser{}, total, 0
	}
	totalPages = total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}
	if page < 1 || page > totalPages {
		return []define.AllowListUser{}, total, totalPages
	}
	start := (page - 1) * pageSize
	count := pageSize
	if count > total-start {
		count = total - start
	}
	return cloneUsers(v.rows[start : start+count]), total, totalPages
}
