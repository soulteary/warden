package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	middlewarekit "github.com/soulteary/middleware-kit/v3"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/define"
)

// operationalRoutesApp returns an app whose rate limiter allows rate requests
// per client in the test's lifetime.
func operationalRoutesApp(t *testing.T, rate int) *App {
	t.Helper()
	app := responseMetadataApp(t)
	cfg := middlewarekit.DefaultRateLimiterConfig()
	cfg.Rate = rate
	cfg.Window = time.Minute
	limiter := middlewarekit.NewRateLimiter(cfg)
	t.Cleanup(limiter.Stop)
	app.rateLimiter = limiter
	return app
}

func TestOperationalRoutesAreRateLimited(t *testing.T) {
	t.Setenv("WARDEN_METRICS_REQUIRE_AUTH", "")
	for _, path := range []string{define.PATH_METRICS, define.PATH_LOG_LEVEL} {
		t.Run(path, func(t *testing.T) {
			mux := registerRoutes(operationalRoutesApp(t, 2))
			request := func() int {
				req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
				req.RemoteAddr = "198.51.100.20:1234"
				req.Header.Set("X-API-Key", "key")
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, req)
				return rec.Code
			}
			require.Equal(t, http.StatusOK, request())
			require.Equal(t, http.StatusOK, request())
			require.Equal(t, http.StatusTooManyRequests, request(), "%s must share the per-client rate limit", path)
		})
	}
}

func TestOperationalRoutesShareTheDataRateLimitBudget(t *testing.T) {
	t.Setenv("WARDEN_METRICS_REQUIRE_AUTH", "")
	mux := registerRoutes(operationalRoutesApp(t, 2))
	request := func(path string) int {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		req.RemoteAddr = "198.51.100.21:1234"
		req.Header.Set("X-API-Key", "key")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}
	require.Equal(t, http.StatusOK, request("/user?user_id=member"))
	require.Equal(t, http.StatusOK, request(define.PATH_METRICS))
	require.Equal(t, http.StatusTooManyRequests, request(define.PATH_LOG_LEVEL))
	// Health probes stay outside the limiter.
	require.NotEqual(t, http.StatusTooManyRequests, request(define.PATH_HEALTH))
}

func TestOperationalRoutesEnforceBodyLimit(t *testing.T) {
	t.Setenv("WARDEN_METRICS_REQUIRE_AUTH", "")
	mux := registerRoutes(operationalRoutesApp(t, 100))
	oversized := strings.Repeat("a", define.MAX_REQUEST_BODY_SIZE+1)
	for _, path := range []string{define.PATH_METRICS, define.PATH_LOG_LEVEL} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(oversized))
		req.RemoteAddr = "198.51.100.22:1234"
		req.Header.Set("X-API-Key", "key")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, path)
	}

	// A body within the limit still reaches the log-level handler.
	req := httptest.NewRequest(http.MethodPost, define.PATH_LOG_LEVEL, strings.NewReader(`{"level":"info"}`))
	req.RemoteAddr = "198.51.100.22:1234"
	req.Header.Set("X-API-Key", "key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
