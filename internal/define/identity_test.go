package define

import "testing"

func TestParseUserIDStrategy(t *testing.T) {
	tests := []struct {
		in    string
		want  UserIDStrategy
		valid bool
	}{
		{"", UserIDStrategyLegacy, true},
		{"legacy", UserIDStrategyLegacy, true},
		{"LEGACY", UserIDStrategyLegacy, true},
		{"sha256-128", UserIDStrategySHA256_128, true},
		{"bogus", UserIDStrategyLegacy, false},
	}
	for _, tc := range tests {
		got, ok := ParseUserIDStrategy(tc.in)
		if got != tc.want || ok != tc.valid {
			t.Fatalf("ParseUserIDStrategy(%q)=(%q,%v) want (%q,%v)", tc.in, got, ok, tc.want, tc.valid)
		}
	}
}

func TestDeriveUserID_LegacyStable(t *testing.T) {
	SetUserIDStrategy(UserIDStrategyLegacy)
	defer SetUserIDStrategy(UserIDStrategyLegacy)

	id1 := deriveUserID("13800138000", "")
	id2 := deriveUserID("13800138000", "")
	if id1 == "" || id1 != id2 {
		t.Fatalf("legacy derivation not stable: %q vs %q", id1, id2)
	}
	if len(id1) != 16 {
		t.Fatalf("legacy id length = %d, want 16", len(id1))
	}
}

func TestDeriveUserID_SHA256_128DomainSeparation(t *testing.T) {
	SetUserIDStrategy(UserIDStrategySHA256_128)
	defer SetUserIDStrategy(UserIDStrategyLegacy)

	// Same string as phone vs mail must not collide (domain separation).
	phoneID := deriveUserID("shared", "")
	mailID := deriveUserID("", "shared")
	if phoneID == mailID {
		t.Fatalf("domain separation failed: phone and mail derived identical id")
	}
	if len(phoneID) != 32 || len(mailID) != 32 {
		t.Fatalf("sha256-128 id length: phone=%d mail=%d, want 32", len(phoneID), len(mailID))
	}
}

func TestDeriveUserID_Empty(t *testing.T) {
	SetUserIDStrategy(UserIDStrategyLegacy)
	if deriveUserID("", "") != "" {
		t.Fatalf("expected empty id for empty identifiers")
	}
}

func TestNormalize_RespectsRequireExplicit(t *testing.T) {
	SetUserIDStrategy(UserIDStrategyLegacy)
	SetRequireExplicitUserID(true)
	defer SetRequireExplicitUserID(false)

	u := AllowListUser{Phone: "13800138000"}
	u.Normalize()
	if u.UserID != "" {
		t.Fatalf("expected empty user_id when explicit id required, got %q", u.UserID)
	}

	SetRequireExplicitUserID(false)
	u2 := AllowListUser{Phone: "13800138000"}
	u2.Normalize()
	if u2.UserID == "" {
		t.Fatalf("expected derived user_id when not required")
	}
}
