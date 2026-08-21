package identity

import (
	"errors"
	"reflect"
	"testing"

	"github.com/soulteary/warden/internal/define"
)

func u(phone, mail, uid string) define.AllowListUser {
	return define.AllowListUser{Phone: phone, Mail: mail, UserID: uid}
}

func TestValidateAndIndexUsers_Uniqueness(t *testing.T) {
	// Ensure legacy strategy so derived ids are deterministic and non-empty.
	define.SetUserIDStrategy(define.UserIDStrategyLegacy)
	define.SetRequireExplicitUserID(false)

	tests := []struct {
		name      string
		users     []define.AllowListUser
		wantField FieldType
		wantErr   error
	}{
		{
			name:    "clean set",
			users:   []define.AllowListUser{u("13800000001", "a@example.com", ""), u("13800000002", "b@example.com", "")},
			wantErr: nil,
		},
		{
			name:      "duplicate phone",
			users:     []define.AllowListUser{u("13800000001", "a@example.com", ""), u("13800000001", "c@example.com", "")},
			wantField: FieldPhone,
			wantErr:   ErrIdentityConflict,
		},
		{
			name:      "duplicate mail case-insensitive",
			users:     []define.AllowListUser{u("", "Dup@Example.com", ""), u("", "dup@example.com", "")},
			wantField: FieldMail,
			wantErr:   ErrIdentityConflict,
		},
		{
			name:      "duplicate explicit user_id",
			users:     []define.AllowListUser{u("13800000001", "", "same-id"), u("13800000002", "", "same-id")},
			wantField: FieldUserID,
			wantErr:   ErrIdentityConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateAndIndexUsers(tc.users, Options{})
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
			var ce *ConflictError
			if errors.As(err, &ce) {
				if ce.Field != tc.wantField {
					t.Fatalf("expected field %s, got %s", tc.wantField, ce.Field)
				}
				// Masked value must not equal the raw value.
				if ce.MaskedValue == "13800000001" {
					t.Fatalf("conflict error leaked raw value: %q", ce.MaskedValue)
				}
			}
		})
	}
}

func TestValidateAndIndexUsers_RequireExplicitUserID(t *testing.T) {
	define.SetUserIDStrategy(define.UserIDStrategyLegacy)
	users := []define.AllowListUser{u("13800000001", "a@example.com", "")}

	// Without requirement: derives id, no error.
	res, err := ValidateAndIndexUsers(users, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Users[0].UserID == "" {
		t.Fatalf("expected derived user_id")
	}

	// With requirement: reject.
	define.SetRequireExplicitUserID(true)
	defer define.SetRequireExplicitUserID(false)
	_, err = ValidateAndIndexUsers(users, Options{RequireExplicitUserID: true})
	if !errors.Is(err, ErrMissingUserID) {
		t.Fatalf("expected ErrMissingUserID, got %v", err)
	}
	var me *MissingUserIDError
	if errors.As(err, &me) {
		if me.MaskedIdentifier == "13800000001" {
			t.Fatalf("missing-id error leaked raw phone")
		}
	}
}

func TestValidateAndIndexUsers_DeterministicOrdering(t *testing.T) {
	define.SetUserIDStrategy(define.UserIDStrategyLegacy)
	define.SetRequireExplicitUserID(false)

	a := u("13800000003", "c@example.com", "id-c")
	b := u("13800000001", "a@example.com", "id-a")
	c := u("13800000002", "b@example.com", "id-b")

	res1, err := ValidateAndIndexUsers([]define.AllowListUser{a, b, c}, Options{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	res2, err := ValidateAndIndexUsers([]define.AllowListUser{c, a, b}, Options{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !reflect.DeepEqual(res1.Users, res2.Users) {
		t.Fatalf("ordering not deterministic:\n%v\n%v", res1.Users, res2.Users)
	}
	// Sorted by user_id ascending.
	if res1.Users[0].UserID != "id-a" || res1.Users[2].UserID != "id-c" {
		t.Fatalf("unexpected order: %v", res1.Users)
	}
}

func TestValidateAndIndexUsers_ReportOnly(t *testing.T) {
	define.SetUserIDStrategy(define.UserIDStrategyLegacy)
	define.SetRequireExplicitUserID(false)

	users := []define.AllowListUser{
		u("13800000001", "a@example.com", ""),
		u("13800000001", "b@example.com", ""), // dup phone
		u("", "a@example.com", ""),            // dup mail
	}
	res, err := ValidateAndIndexUsers(users, Options{ReportOnly: true})
	if err != nil {
		t.Fatalf("report-only must not error, got %v", err)
	}
	if res.ConflictCount == 0 {
		t.Fatalf("expected conflicts counted, got 0")
	}
}
