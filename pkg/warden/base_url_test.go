package warden

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestValidateRejectsBaseURLWithoutScheme(t *testing.T) {
	for _, base := range []string{"localhost:8081", "warden:8081", "warden.example.com", "//warden:8081", "ftp://warden:8081", "http://", "https:///path"} {
		opts := &Options{BaseURL: base, Timeout: time.Second}
		err := opts.Validate()
		if err == nil || !strings.Contains(err.Error(), "http:// or https://") {
			t.Errorf("Validate(%q) error = %v, want a scheme error", base, err)
		}
	}
	for _, base := range []string{"http://localhost:8081", "https://warden.example.com", "http://warden:8081/", "https://[::1]:8443"} {
		opts := &Options{BaseURL: base, Timeout: time.Second}
		if err := opts.Validate(); err != nil {
			t.Errorf("Validate(%q) error = %v", base, err)
		}
	}
}

func TestRequireHTTPSRefusesAPIKeyOverPlainHTTP(t *testing.T) {
	for _, tc := range []struct {
		opts    *Options
		name    string
		wantErr bool
	}{
		{DefaultOptions().WithBaseURL("http://warden:8081").WithAPIKey("key"), "API key over http to a remote host", true},
		{DefaultOptions().WithBaseURL("http://10.0.0.5:8081").WithAPIKey("key"), "API key over http to a remote address", true},
		{DefaultOptions().WithBaseURL("https://warden:8081").WithAPIKey("key"), "API key over https", false},
		{DefaultOptions().WithBaseURL("http://localhost:8081").WithAPIKey("key"), "API key over http to localhost", false},
		{DefaultOptions().WithBaseURL("http://127.0.0.1:8081").WithAPIKey("key"), "API key over http to loopback IPv4", false},
		{DefaultOptions().WithBaseURL("http://[::1]:8081").WithAPIKey("key"), "API key over http to loopback IPv6", false},
		{DefaultOptions().WithBaseURL("http://warden:8081"), "no credential over http", false},
		// A signed request carries no secret: the API key is not sent when HMAC is configured.
		{DefaultOptions().WithBaseURL("http://warden:8081").WithAPIKey("key").WithHMAC("gateway", "secret"), "HMAC over http", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.WithRequireHTTPS(true).Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "https") {
				t.Fatalf("error does not name https: %v", err)
			}
		})
	}
}

type warnRecorder struct {
	warns []string
	NoOpLogger
	mu sync.Mutex
}

func (r *warnRecorder) Warn(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.warns = append(r.warns, msg)
}

// Without RequireHTTPS the configuration is still accepted, and reported.
func TestNewClientWarnsWhenAPIKeyIsSentInClear(t *testing.T) {
	for _, tc := range []struct {
		opts *Options
		name string
		warn bool
	}{
		{DefaultOptions().WithBaseURL("http://warden:8081").WithAPIKey("secret-key"), "remote http with API key", true},
		{DefaultOptions().WithBaseURL("https://warden:8081").WithAPIKey("secret-key"), "https with API key", false},
		{DefaultOptions().WithBaseURL("http://127.0.0.1:8081").WithAPIKey("secret-key"), "loopback http with API key", false},
		{DefaultOptions().WithBaseURL("http://warden:8081").WithAPIKey("secret-key").WithHMAC("gateway", "secret"), "remote http with HMAC", false},
		{DefaultOptions().WithBaseURL("http://warden:8081"), "remote http without credential", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := &warnRecorder{}
			client, err := NewClient(tc.opts.WithLogger(rec))
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			defer client.Close()
			if got := len(rec.warns) == 1; got != tc.warn {
				t.Fatalf("warnings = %v, want warning: %v", rec.warns, tc.warn)
			}
			for _, w := range rec.warns {
				if strings.Contains(w, "secret-key") {
					t.Fatalf("warning contains the API key: %q", w)
				}
			}
		})
	}
}
