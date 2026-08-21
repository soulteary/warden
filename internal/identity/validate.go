// Package identity provides centralized validation and indexing of allow-list
// users before they enter the shared cache. It enforces identifier uniqueness
// (no last-write-wins), produces deterministic ordering, and returns structured,
// masked errors that never leak full phone numbers or email addresses.
package identity

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/soulteary/warden/internal/define"
)

// ErrIdentityConflict is the sentinel wrapped by ConflictError.
var ErrIdentityConflict = errors.New("identity: duplicate identifier across users")

// ErrMissingUserID is returned when REQUIRE_EXPLICIT_USER_ID is set but a record has none.
var ErrMissingUserID = errors.New("identity: explicit user_id required but missing")

// FieldType identifies which unique field caused a conflict.
type FieldType string

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
	Index int
	// MaskedIdentifier is a masked phone/mail used only for diagnostics.
	MaskedIdentifier string
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
	normalized := make([]define.AllowListUser, len(users))
	copy(normalized, users)
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

		if phone != "" {
			if prev, ok := phoneIdx[phone]; ok {
				if cErr := handleConflict(&res, opts, FieldPhone, MaskPhone(phone), prev, i); cErr != nil {
					return Result{}, cErr
				}
				continue
			}
			phoneIdx[phone] = i
		}
		if mail != "" {
			if prev, ok := mailIdx[mail]; ok {
				if cErr := handleConflict(&res, opts, FieldMail, MaskMail(mail), prev, i); cErr != nil {
					return Result{}, cErr
				}
				continue
			}
			mailIdx[mail] = i
		}
		if uid != "" {
			if prev, ok := userIDIdx[uid]; ok {
				if cErr := handleConflict(&res, opts, FieldUserID, MaskUserID(uid), prev, i); cErr != nil {
					return Result{}, cErr
				}
				continue
			}
			userIDIdx[uid] = i
		}
	}

	if opts.ReportOnly {
		res.Users = stableSort(normalized)
		return res, nil
	}
	res.Users = stableSort(normalized)
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

// stableSort returns a copy of users deterministically ordered: primarily by user_id,
// falling back to the canonical key (phone, else normalized mail), then by mail. This
// guarantees identical output regardless of input map/insertion order.
func stableSort(users []define.AllowListUser) []define.AllowListUser {
	out := make([]define.AllowListUser, len(users))
	copy(out, users)
	sort.SliceStable(out, func(i, j int) bool {
		ki, kj := out[i].UserID, out[j].UserID
		if ki != kj {
			return ki < kj
		}
		ci, cj := canonicalKey(out[i]), canonicalKey(out[j])
		if ci != cj {
			return ci < cj
		}
		return strings.ToLower(out[i].Mail) < strings.ToLower(out[j].Mail)
	})
	return out
}

// canonicalKey mirrors the cache/loader primary key (phone else normalized mail).
func canonicalKey(u define.AllowListUser) string {
	k := strings.TrimSpace(u.Phone)
	if k == "" {
		k = strings.ToLower(strings.TrimSpace(u.Mail))
	}
	return k
}
