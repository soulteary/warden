package identity

import (
	"strings"
	"testing"
)

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"1", "*"},
		{"12", "**"},
		{"13800138000", "*********00"},
	}
	for _, tc := range tests {
		if got := MaskPhone(tc.in); got != tc.want {
			t.Fatalf("MaskPhone(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	// Never leaks more than the last two digits.
	full := "13912345678"
	if got := MaskPhone(full); strings.Contains(got, full[:len(full)-2]) {
		t.Fatalf("MaskPhone leaked leading digits: %q", got)
	}
}

func TestMaskMail(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"a@example.com", "a*@example.com"},
		{"alice@example.com", "a****@example.com"},
		{"ALICE@Example.com", "a****@example.com"},
	}
	for _, tc := range tests {
		if got := MaskMail(tc.in); got != tc.want {
			t.Fatalf("MaskMail(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestMaskUserID(t *testing.T) {
	if got := MaskUserID("abcd1234ef"); got != "abcd******" {
		t.Fatalf("MaskUserID=%q", got)
	}
	if got := MaskUserID("ab"); got != "**" {
		t.Fatalf("MaskUserID short=%q", got)
	}
}
