package warden

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func requireLookupError(t *testing.T, err error, code string) {
	t.Helper()
	var sdkErr *Error
	require.True(t, errors.As(err, &sdkErr), "%v", err)
	require.Equal(t, code, sdkErr.Code)
}

// Lookup reports why no result is available: Warden unreachable, a body that
// is not a lookup result, and a result that outlived its source deadline while
// it was being read.
func TestLookupFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("unreachable", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		url := server.URL
		server.Close()
		client, err := NewClient(DefaultOptions().WithBaseURL(url))
		require.NoError(t, err)
		result, err := client.Lookup(ctx, "member")
		require.Nil(t, result)
		requireLookupError(t, err, ErrCodeRequestFailed)
	})

	t.Run("body is not a lookup result", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`["not", "an", "object"]`)) //nolint:errcheck // test server
		}))
		t.Cleanup(server.Close)
		client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
		require.NoError(t, err)
		result, err := client.Lookup(ctx, "member")
		require.Nil(t, result)
		requireLookupError(t, err, ErrCodeInvalidResponse)
	})

	t.Run("deadline reached while the body was read", func(t *testing.T) {
		clock := newSourceTestClock()
		source := clock.Now()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			setSourceTestHeaders(w.Header(), source, source.Add(time.Minute))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user_id":"member","status":"active"}`)) //nolint:errcheck // test server
		}))
		t.Cleanup(server.Close)
		client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithSourceFreshness(time.Minute, 0).WithClock(clock))
		require.NoError(t, err)

		// In time: the result is returned.
		result, err := client.Lookup(ctx, "member")
		require.NoError(t, err)
		require.Equal(t, "member", result.UserID)

		// The first reading of the clock, when the headers are checked, is in
		// time; the deadline passes before the second, after the body.
		var readings atomic.Int64
		clock.onNow = func() {
			if readings.Add(1) == 1 {
				clock.set(source.Add(2 * time.Minute))
			}
		}
		result, err = client.Lookup(ctx, "member")
		require.Nil(t, result)
		requireLookupError(t, err, ErrCodeSourceFreshness)
		require.ErrorIs(t, err, ErrSourceStale)
	})
}
