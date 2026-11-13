package loader

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/remote"
)

type countingTransport struct{ requests atomic.Int64 }

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.requests.Add(1)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`[{"phone":"13800138000","status":"active"}]`)),
		Request:    req,
	}, nil
}

// observeClientBuilds replaces the client constructor with one that counts its
// calls and returns a client backed by transport.
func observeClientBuilds(t *testing.T, transport http.RoundTripper) *atomic.Int64 {
	t.Helper()
	var builds atomic.Int64
	original := newRestrictedClient
	t.Cleanup(func() { newRestrictedClient = original })
	newRestrictedClient = func(opts remote.ClientOptions) (*http.Client, error) {
		builds.Add(1)
		return &http.Client{Timeout: opts.Timeout, Transport: transport}, nil
	}
	return &builds
}

// One loader keeps one HTTP client for all its refreshes, so idle connections
// to the remote source are reused.
func TestRemoteClientIsBuiltOncePerLoader(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "data.json")
	require.NoError(t, os.WriteFile(dataFile, []byte(`[]`), 0o600))

	for name, cfg := range map[string]*cmd.Config{
		"plain":     {HTTPTimeout: 5},
		"encrypted": {HTTPTimeout: 5, RemoteDecryptEnabled: true, RemoteRSAPrivateKey: "not-a-key"},
	} {
		t.Run(name, func(t *testing.T) {
			transport := &countingTransport{}
			builds := observeClientBuilds(t, transport)
			r, err := NewRulesLoader(cfg, "DEFAULT")
			require.NoError(t, err)

			for i := 0; i < 3; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				r.LoadWithResult(ctx, dataFile, "", "https://rules.example.com/data.json", "")
				cancel()
			}
			require.EqualValues(t, 3, transport.requests.Load(), "every refresh reaches the remote source")
			require.EqualValues(t, 1, builds.Load(), "the client is built once and reused")
		})
	}
}

// A client that cannot be built is reported on every refresh and retried on
// the next one.
func TestRemoteClientBuildErrorIsNotCached(t *testing.T) {
	var builds atomic.Int64
	original := newRestrictedClient
	t.Cleanup(func() { newRestrictedClient = original })
	newRestrictedClient = func(remote.ClientOptions) (*http.Client, error) {
		builds.Add(1)
		return nil, errors.New("invalid REMOTE_PRIVATE_ORIGINS")
	}
	for name, cfg := range map[string]*cmd.Config{
		"plain":     {HTTPTimeout: 5},
		"encrypted": {HTTPTimeout: 5, RemoteDecryptEnabled: true, RemoteRSAPrivateKey: "not-a-key"},
	} {
		t.Run(name, func(t *testing.T) {
			r, err := NewRulesLoader(cfg, "ONLY_REMOTE")
			require.NoError(t, err)
			before := builds.Load()
			first := r.LoadWithResult(context.Background(), "", "", "https://rules.example.com/data.json", "")
			require.Error(t, first.Err)
			afterFirst := builds.Load()
			require.Greater(t, afterFirst, before)
			second := r.LoadWithResult(context.Background(), "", "", "https://rules.example.com/data.json", "")
			require.Error(t, second.Err)
			require.Greater(t, builds.Load(), afterFirst, "the next refresh tries again")
		})
	}
}

// A client supplied by the embedding caller is used as given.
func TestSuppliedRemoteClientIsUsedUnchanged(t *testing.T) {
	builds := observeClientBuilds(t, &countingTransport{})
	supplied := &http.Client{Transport: &countingTransport{}}
	r, err := NewRulesLoaderWithHTTPClient(&cmd.Config{HTTPTimeout: 5}, "DEFAULT", supplied)
	require.NoError(t, err)
	client, err := r.remoteClient()
	require.NoError(t, err)
	require.Same(t, supplied, client)
	require.Same(t, supplied, r.decryptClient())
	require.Zero(t, builds.Load())
}
