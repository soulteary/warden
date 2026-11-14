package loader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/remote"
)

// A source the connection policy refuses fails with that refusal, at once,
// instead of being retried until the request timeout and reported as a
// deadline error.
func TestAddressPolicyRefusalIsReportedWithoutRetries(t *testing.T) {
	var hits atomic.Int32
	// A loopback listener: the restricted client never connects to it.
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		if _, err := w.Write([]byte(`[{"phone":"13800138000"}]`)); err != nil {
			t.Errorf("write fixture response: %v", err)
		}
	}))
	defer source.Close()

	for _, mode := range []string{"ONLY_REMOTE", "REMOTE_FIRST"} {
		t.Run(mode, func(t *testing.T) {
			// The server path: the loader builds the restricted client. The
			// constructor is replaced only to count the requests sent through
			// that client.
			var attempts atomic.Int32
			original := newRestrictedClient
			t.Cleanup(func() { newRestrictedClient = original })
			newRestrictedClient = func(opts remote.ClientOptions) (*http.Client, error) {
				client, err := original(opts)
				if err == nil {
					client.Transport = countingRoundTripper{inner: client.Transport, calls: &attempts}
				}
				return client, err
			}
			rulesLoader, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 3}, mode)
			require.NoError(t, err)

			result := rulesLoader.LoadWithResult(context.Background(), "", "", source.URL, "")

			require.ErrorIs(t, result.Err, remote.ErrAddressPolicy)
			require.NotErrorIs(t, result.Err, context.DeadlineExceeded)
			require.NotContains(t, result.Err.Error(), "deadline")
			require.Equal(t, int32(1), attempts.Load(), "a refused address must not be retried")
			require.Zero(t, hits.Load())
		})
	}
}

type countingRoundTripper struct {
	inner http.RoundTripper
	calls *atomic.Int32
}

func (c countingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	c.calls.Add(1)
	return c.inner.RoundTrip(req)
}

// The wrapper leaves every other failure and every success as it was.
func TestAddressPolicyFetcherPassesOtherResultsThrough(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		if _, err := w.Write([]byte(`[{"phone":"13800138000"}]`)); err != nil {
			t.Errorf("write fixture response: %v", err)
		}
	}))
	defer source.Close()
	opts := RemoteOptions{HTTPClient: source.Client(), Timeout: 3 * time.Second}

	raw, err := newRemoteFetcher(source.URL, "", opts).Fetch(context.Background(), 1024)
	require.NoError(t, err)
	require.JSONEq(t, `[{"phone":"13800138000"}]`, string(raw))

	_, err = newRemoteFetcher(source.URL+"/missing", "", opts).Fetch(context.Background(), 1024)
	require.ErrorContains(t, err, "unexpected status code: 404")
	require.NotErrorIs(t, err, remote.ErrAddressPolicy)
}
