package warden

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "***"},             // secure-kit returns "***" for empty
		{"abc", "***"},          // too short
		{"abcd", "***"},         // too short
		{"abcde", "ab***de"},    // secure-kit uses fixed 3 asterisks
		{"12345678", "12***78"}, // secure-kit uses fixed 3 asterisks
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

func TestNoOpLogger_AllMethods(t *testing.T) {
	var l NoOpLogger
	// All methods must not panic
	l.Debug("msg")
	l.Debugf("format %s", "arg")
	l.Info("msg")
	l.Infof("format %s", "arg")
	l.Warn("msg")
	l.Warnf("format %s", "arg")
	l.Error("msg")
	l.Errorf("format %s", "arg")
}

func TestSanitizeURLMasksUserID(t *testing.T) {
	got := sanitizeURLString("http://warden.internal/user?user_id=0123456789abcdef")
	if strings.Contains(got, "0123456789abcdef") {
		t.Fatalf("user_id is logged in full: %s", got)
	}
}

// The SDK's debug log must not carry a user ID in full: a derived ID is an
// unsalted digest of the phone or mail.
func TestClientDoesNotLogTheFullUserID(t *testing.T) {
	const userID = "0123456789abcdef"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"phone":"13800138000","mail":"","user_id":"` + userID + `","status":"active","scope":[],"role":""}`)) //nolint:errcheck // test server
	}))
	defer server.Close()

	log := &mockLogger{}
	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithLogger(log))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetUserByIdentifier(context.Background(), "", "", userID); err != nil {
		t.Fatal(err)
	}
	if !client.CheckUserInList(context.Background(), "13800138000", "") {
		t.Fatal("the user is active")
	}
	logged := strings.Join(append(append(append(log.debugs, log.infos...), log.warns...), log.errors...), "\n")
	if logged == "" {
		t.Fatal("expected debug output")
	}
	if strings.Contains(logged, userID) {
		t.Fatalf("user ID logged in full:\n%s", logged)
	}
}
