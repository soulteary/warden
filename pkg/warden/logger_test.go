package warden

import (
	"net/url"
	"testing"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"abc", "***"},
		{"abcd", "***"},
		{"abcde", "ab*de"},
		{"12345678", "12****78"},
	}

	for _, tt := range tests {
		if got := sanitizeString(tt.input); got != tt.expected {
			t.Fatalf("sanitizeString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSanitizeURLString(t *testing.T) {
	input := "http://example.com/user?phone=13800138000&EMAIL=User@Example.Com&other=keep"
	sanitized := sanitizeURLString(input)

	u, err := url.Parse(sanitized)
	if err != nil {
		t.Fatalf("failed to parse sanitized URL: %v", err)
	}

	query := u.Query()
	if got := query.Get("phone"); got != sanitizeString("13800138000") {
		t.Fatalf("phone sanitized = %q, want %q", got, sanitizeString("13800138000"))
	}
	if got := query.Get("EMAIL"); got != sanitizeString("User@Example.Com") {
		t.Fatalf("EMAIL sanitized = %q, want %q", got, sanitizeString("User@Example.Com"))
	}
	if got := query.Get("other"); got != "keep" {
		t.Fatalf("other param = %q, want %q", got, "keep")
	}
}

func TestSanitizeURLNil(t *testing.T) {
	if got := sanitizeURL(nil); got != "" {
		t.Fatalf("sanitizeURL(nil) = %q, want empty string", got)
	}
}
