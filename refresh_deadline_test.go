package main

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/loader"
)

// deadlineRecorder answers every request and records how long the request's
// context still had to run.
type deadlineRecorder struct {
	remaining []time.Duration
	mu        sync.Mutex
}

func (d *deadlineRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	deadline, ok := req.Context().Deadline()
	d.mu.Lock()
	if ok {
		d.remaining = append(d.remaining, time.Until(deadline))
	} else {
		d.remaining = append(d.remaining, time.Duration(1<<62))
	}
	d.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`[{"phone":"13800138000","status":"active"}]`)),
		Request:    req,
	}, nil
}

// A background refresh must give the remote request the configured HTTP
// timeout. It used to cancel every refresh after ten seconds.
func TestBackgroundRefreshHonoursTheHTTPTimeout(t *testing.T) {
	for _, key := range []string{"IP_WHITELIST", "HEALTH_CHECK_IP_WHITELIST", "TRUSTED_PROXY_IPS", "SNAPSHOT_MAX_AGE"} {
		t.Setenv(key, "")
	}
	cfg := &cmd.Config{Port: "8081", Mode: "ONLY_REMOTE", Environment: "development", TaskInterval: 60, HTTPTimeout: 30, APIKey: "key"}
	app := NewApp(cfg)
	t.Cleanup(app.rateLimiter.Stop)

	recorder := &deadlineRecorder{}
	rulesLoader, err := loader.NewRulesLoaderWithHTTPClient(cfg, cfg.Mode, &http.Client{Transport: recorder})
	require.NoError(t, err)
	app.rulesLoader = rulesLoader
	app.configURL = "https://rules.example.com/data.json"

	app.backgroundTask("", "")

	require.Len(t, recorder.remaining, 1)
	require.Greater(t, recorder.remaining[0], 25*time.Second, "the request may run for HTTP_TIMEOUT=30s")
	require.Equal(t, 1, app.userCache.Len())
}
