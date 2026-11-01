// Package identity provides centralized validation and indexing of allow-list
// users before they enter the shared cache. It enforces identifier uniqueness
// (no last-write-wins), produces deterministic ordering, and returns structured,
// masked errors that never leak full phone numbers or email addresses.
package identity

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/soulteary/warden/internal/define"
)

// ErrIdentityConflict is the sentinel wrapped by ConflictError.
var ErrIdentityConflict = errors.New("identity: duplicate identifier across users")

// ErrMissingUserID is returned when REQUIRE_EXPLICIT_USER_ID is set but a record has none.
var ErrMissingUserID = errors.New("identity: explicit user_id required but missing")

// FieldType identifies which unique field caused a conflict.
type FieldType string

// Field type constants identify which unique field caused a conflict.
const (
	FieldPhone  FieldType = "phone"
	FieldMail   FieldType = "mail"
	FieldUserID FieldType = "user_id"
)

// ConflictError describes a uniqueness violation. Values are masked; only positions
// (record indices) and the field type are exposed for diagnostics.
type ConflictError struct {
	Field       FieldType
	MaskedValue string
	FirstIndex  int
	SecondIndex int
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%v: field=%s value=%s records=[%d,%d]",
		ErrIdentityConflict, e.Field, e.MaskedValue, e.FirstIndex, e.SecondIndex)
}

func (e *ConflictError) Unwrap() error { return ErrIdentityConflict }

// MissingUserIDError describes a record missing a required explicit user_id.
type MissingUserIDError struct {
	// MaskedIdentifier is a masked phone/mail used only for diagnostics.
	MaskedIdentifier string
	Index            int
}

func (e *MissingUserIDError) Error() string {
	return fmt.Sprintf("%v: record=%d identifier=%s", ErrMissingUserID, e.Index, e.MaskedIdentifier)
}

func (e *MissingUserIDError) Unwrap() error { return ErrMissingUserID }

// Result is the outcome of ValidateAndIndexUsers.
//
//nolint:govet // fieldalignment: readability over a few bytes
type Result struct {
	// Users is the validated, deterministically-ordered rule set (only when Err == nil).
	Users []define.AllowListUser
	// ConflictCount / MissingIDCount are populated in read-only (report) mode.
	ConflictCount  int
	MissingIDCount int
}

// Options controls validation behavior.
type Options struct {
	// RequireExplicitUserID rejects records whose user_id is empty after normalization.
	RequireExplicitUserID bool
	// ReportOnly performs a dry run: it collects conflict/missing counts without
	// failing on the first violation and does not require a clean set.
	ReportOnly bool
}

// ValidateAndIndexUsers normalizes, validates identifier uniqueness, and returns a
// deterministically-ordered copy of users. Two different records may not share any
// non-empty user_id, phone, or normalized mail (last-write-wins is rejected).
//
// In normal mode it returns the first violation as a typed error. In ReportOnly mode
// it returns counts of all violations without an error (used by the read-only check).
func ValidateAndIndexUsers(users []define.AllowListUser, opts Options) (Result, error) {
	owned := make([]define.AllowListUser, len(users))
	copy(owned, users)
	return validateAndIndexUsers(owned, opts, stableSort)
}

// ValidateAndIndexOwnedUsers consumes an exclusively owned outer users slice.
// It normalizes records in place, then may reorder that outer slice after
// validation returns no error. On error, normalized records retain their original
// positions. Callers must not retain the original outer values. Scope contents
// are neither copied nor modified;
// callers retain responsibility for the ownership of those nested slices.
func ValidateAndIndexOwnedUsers(users []define.AllowListUser, opts Options) (Result, error) {
	return validateAndIndexUsers(users, opts, stableSortOwned)
}

// Both entry points validate before ordering, preserving original error indexes.
func validateAndIndexUsers(users []define.AllowListUser, opts Options, orderUsers func([]define.AllowListUser) []define.AllowListUser) (Result, error) {
	normalized := users
	for i := range normalized {
		normalized[i].Normalize()
	}

	phoneIdx := make(map[string]int, len(normalized))
	mailIdx := make(map[string]int, len(normalized))
	userIDIdx := make(map[string]int, len(normalized))

	var res Result
	for i := range normalized {
		u := &normalized[i]
		phone := strings.TrimSpace(u.Phone)
		mail := strings.ToLower(strings.TrimSpace(u.Mail))
		uid := strings.TrimSpace(u.UserID)

		if opts.RequireExplicitUserID && uid == "" {
			mErr := &MissingUserIDError{Index: i, MaskedIdentifier: maskIdentifier(phone, mail)}
			if opts.ReportOnly {
				res.MissingIDCount++
				continue
			}
			return Result{}, mErr
		}

		err := indexIdentifiers(&res, opts, i, phone, mail, uid, phoneIdx, mailIdx, userIDIdx)
		if err != nil {
			return Result{}, err
		}
	}

	res.Users = orderUsers(normalized)
	return res, nil
}

// handleConflict either accumulates (ReportOnly) or returns a typed ConflictError.
func handleConflict(res *Result, opts Options, field FieldType, masked string, first, second int) error {
	if opts.ReportOnly {
		res.ConflictCount++
		return nil
	}
	return &ConflictError{Field: field, MaskedValue: masked, FirstIndex: first, SecondIndex: second}
}

// stableSort returns a detached outer slice ordered by user_id, canonical key,
// then lowercase mail. Equal keys retain their original relative order.
func stableSort(users []define.AllowListUser) []define.AllowListUser {
	out := make([]define.AllowListUser, len(users))
	if len(users) < 2 {
		copy(out, users)
		return out
	}
	order := stableUserOrder(users)
	for i, index := range order {
		out[i] = users[index]
	}
	return out
}

// stableSortOwned reuses a private outer slice while retaining the legacy empty
// result and capacity boundary. It never modifies existing Scope contents.
func stableSortOwned(users []define.AllowListUser) []define.AllowListUser {
	if len(users) == 0 {
		return []define.AllowListUser{}
	}
	if len(users) > 1 {
		applyUserPermutation(users, stableUserOrder(users))
	}
	return users[:len(users):len(users)]
}

// stableUserOrder computes all keys before any record is moved. Both copy and
// owned paths use the same stable comparator and tie handling.
func stableUserOrder(users []define.AllowListUser) []int {
	order := make([]int, len(users))
	for i := range order {
		order[i] = i
	}
	// Move indexes rather than whole user records during the stable sort.
	slices.SortStableFunc(order, func(left, right int) int {
		return strings.Compare(users[left].UserID, users[right].UserID)
	})
	for start := 0; start < len(order); {
		end := start + 1
		for end < len(order) && users[order[end]].UserID == users[order[start]].UserID {
			end++
		}
		if end-start > 1 {
			sortIdentityTies(order[start:end], users)
		}
		start = end
	}
	return order
}

// order maps each destination to its original source index. Consume its cycles,
// marking visited destinations as fixed points, without a separate visited array.
func applyUserPermutation(users []define.AllowListUser, order []int) {
	for start := range order {
		if order[start] == start {
			continue
		}
		saved := users[start]
		current := start
		for {
			next := order[current]
			order[current] = current
			if next == start {
				users[current] = saved
				break
			}
			users[current] = users[next]
			current = next
		}
	}
}

type identitySortKey struct {
	canonical string
	mail      string
	index     int
}

// Only equal user_id groups need the secondary keys. Compute each once rather
// than normalizing strings in every comparison, leaving unique IDs inexpensive.
func sortIdentityTies(order []int, users []define.AllowListUser) {
	keys := make([]identitySortKey, len(order))
	for i, index := range order {
		keys[i] = identitySortKey{canonical: canonicalKey(&users[index]), mail: strings.ToLower(users[index].Mail), index: index}
	}
	slices.SortStableFunc(keys, func(left, right identitySortKey) int {
		if result := strings.Compare(left.canonical, right.canonical); result != 0 {
			return result
		}
		return strings.Compare(left.mail, right.mail)
	})
	for i, key := range keys {
		order[i] = key.index
	}
}

// canonicalKey mirrors the cache/loader primary key (phone else normalized mail).
func canonicalKey(u *define.AllowListUser) string {
	k := strings.TrimSpace(u.Phone)
	if k == "" {
		k = strings.ToLower(strings.TrimSpace(u.Mail))
	}
	return k
}

// indexIdentifiers stops at the first conflicting field, preserving report counts
// and the partial indexes built before that conflict.
func indexIdentifiers(res *Result, opts Options, index int, phone, mail, uid string, phoneIdx, mailIdx, userIDIdx map[string]int) error {
	fields := []struct {
		indexes map[string]int
		mask    func(string) string
		value   string
		field   FieldType
	}{
		{phoneIdx, MaskPhone, phone, FieldPhone},
		{mailIdx, MaskMail, mail, FieldMail},
		{userIDIdx, MaskUserID, uid, FieldUserID},
	}
	for _, field := range fields {
		if field.value == "" {
			continue
		}
		if previous, ok := field.indexes[field.value]; ok {
			return handleConflict(res, opts, field.field, field.mask(field.value), previous, index)
		}
		field.indexes[field.value] = index
	}
	return nil
}
