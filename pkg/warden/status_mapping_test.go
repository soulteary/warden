package warden

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func statusServer(t *testing.T, status int, header http.Header) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for name, values := range header {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		http.Error(w, http.StatusText(status), status)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// A 401 is described by the credential the client sent.
func TestUnauthorizedMessageNamesTheCredentialInUse(t *testing.T) {
	url := statusServer(t, http.StatusUnauthorized, nil)

	withHMAC := DefaultOptions().WithBaseURL(url)
	withHMAC.HMACKeyID, withHMAC.HMACSecret = "gateway", "secret"
	withKey := DefaultOptions().WithBaseURL(url).WithAPIKey("key")
	without := DefaultOptions().WithBaseURL(url)

	for name, tc := range map[string]struct {
		opts        *Options
		contains    string
		notContains string
	}{
		"hmac":          {withHMAC, "HMAC signature", "API key"},
		"api key":       {withKey, "invalid API key", "HMAC"},
		"no credential": {without, "no API key or HMAC signature", "invalid"},
	} {
		client, err := NewClient(tc.opts)
		require.NoError(t, err, name)
		_, err = client.GetUserByIdentifier(context.Background(), "13800138000", "", "")
		var sdkErr *Error
		require.True(t, errors.As(err, &sdkErr), name)
		require.Equal(t, ErrCodeUnauthorized, sdkErr.Code, name)
		require.Contains(t, sdkErr.Message, tc.contains, name)
		require.NotContains(t, sdkErr.Message, tc.notContains, name)
	}
}

// A 429 has its own error code and is not retried.
func TestRateLimitedResponseHasItsOwnCode(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Retry-After", "17")
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	opts := DefaultOptions().WithBaseURL(server.URL).WithAPIKey("key")
	opts.Retry = DefaultRetryOptions()
	opts.Retry.MaxRetries = 3
	opts.Retry.RetryDelay = time.Millisecond
	client, err := NewClient(opts)
	require.NoError(t, err)

	_, err = client.GetUserByIdentifier(context.Background(), "13800138000", "", "")
	var sdkErr *Error
	require.True(t, errors.As(err, &sdkErr))
	require.Equal(t, "RATE_LIMITED", sdkErr.Code)
	require.Equal(t, 1, requests, "a rate-limited request is not retried")
}
