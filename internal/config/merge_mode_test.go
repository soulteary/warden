package config

import "testing"

func TestParseMergeMode(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want MergeMode
		ok   bool
	}{
		{"empty defaults", "", MergeDefault, true},
		{"whitespace defaults", "  ", MergeDefault, true},
		{"default explicit", "DEFAULT", MergeDefault, true},
		{"lower default normalizes", "default", MergeDefault, true},
		{"only local", "ONLY_LOCAL", MergeOnlyLocal, true},
		{"only remote", "ONLY_REMOTE", MergeOnlyRemote, true},
		{"remote first", "REMOTE_FIRST", MergeRemoteFirst, true},
		{"remote first allow fail", "REMOTE_FIRST_ALLOW_REMOTE_FAILED", MergeRemoteFirstAllowRemoteFail, true},
		{"local first", "LOCAL_FIRST", MergeLocalFirst, true},
		{"local first allow fail", "LOCAL_FIRST_ALLOW_REMOTE_FAILED", MergeLocalFirstAllowRemoteFail, true},
		{"mixed case normalizes", "remote_first", MergeRemoteFirst, true},
		{"padded normalizes", "  ONLY_LOCAL  ", MergeOnlyLocal, true},
		{"production is not a merge mode", "production", MergeDefault, false},
		{"unknown falls back but not ok", "SOMETHING", MergeDefault, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseMergeMode(tc.in)
			if got != tc.want {
				t.Fatalf("ParseMergeMode(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if ok != tc.ok {
				t.Fatalf("ParseMergeMode(%q) ok = %v, want %v", tc.in, ok, tc.ok)
			}
		})
	}
}

func TestMergeModeValidate(t *testing.T) {
	valid := []MergeMode{
		MergeDefault, MergeOnlyLocal, MergeOnlyRemote, MergeRemoteFirst,
		MergeRemoteFirstAllowRemoteFail, MergeLocalFirst, MergeLocalFirstAllowRemoteFail,
	}
	for _, m := range valid {
		if !m.Validate() {
			t.Fatalf("expected %q to be valid", m)
		}
	}
	invalid := []MergeMode{"", "production", "prod", "only_local", "bogus"}
	for _, m := range invalid {
		if m.Validate() {
			t.Fatalf("expected %q to be invalid (Validate operates on canonical values)", m)
		}
	}
}

func TestMergeModeIsOnlyLocal(t *testing.T) {
	if !MergeOnlyLocal.IsOnlyLocal() {
		t.Fatal("ONLY_LOCAL must report IsOnlyLocal")
	}
	for _, m := range []MergeMode{MergeDefault, MergeRemoteFirst, MergeLocalFirst} {
		if m.IsOnlyLocal() {
			t.Fatalf("%q must not report IsOnlyLocal", m)
		}
	}
}

func TestDefaultMergeMode(t *testing.T) {
	if DefaultMergeMode() != MergeDefault {
		t.Fatalf("DefaultMergeMode() = %q, want %q", DefaultMergeMode(), MergeDefault)
	}
}
