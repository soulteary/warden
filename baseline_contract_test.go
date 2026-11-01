package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	middlewarekit "github.com/soulteary/middleware-kit/v3"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/stretchr/testify/require"
)

// These tests record the current HTTP contract; they do not enforce freshness on reads.
func TestBaselineSnapshotReadContract(t *testing.T) {
	for _, tc := range []struct {
		loadedAt   time.Time
		name       string
		healthCode int
		userCode   int
		empty      bool
	}{
		{loadedAt: time.Now().Add(-time.Hour), name: "stale_known_good", healthCode: http.StatusServiceUnavailable, userCode: http.StatusOK},
		{loadedAt: time.Now(), name: "fresh_empty", healthCode: http.StatusOK, userCode: http.StatusNotFound, empty: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("IP_WHITELIST", "")
			t.Setenv("HEALTH_CHECK_IP_WHITELIST", "")
			t.Setenv("TRUSTED_PROXY_IPS", "")
			users := cache.NewSafeUserCache()
			if !tc.empty {
				users.Set([]define.AllowListUser{{Phone: "13800138000", Mail: "member@example.com", UserID: "member", Status: "active"}})
			}
			snapshots := newSnapshotStore()
			snapshots.Store(&Snapshot{Users: users.Get(), Count: users.Len(), Source: "remote", Version: cache.HashUserList(users.Get()), LoadedAt: tc.loadedAt})
			limiter := middlewarekit.NewRateLimiter(middlewarekit.DefaultRateLimiterConfig())
			t.Cleanup(limiter.Stop)
			app := &App{userCache: users, snapshots: snapshots, snapshotMaxAge: time.Minute, appMode: "ONLY_REMOTE", environment: "development", apiKey: "contract-key", rateLimiter: limiter, log: logger.GetLoggerKit()}
			mux := registerRoutes(app)
			for _, path := range []string{"/health", "/healthcheck", "/v1/health", "/v1/healthcheck"} {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, http.NoBody))
				require.Equal(t, tc.healthCode, rec.Code, path)
			}
			for _, path := range []string{"/", "/data.json", "/v1/users"} {
				req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
				req.Header.Set("X-API-Key", "contract-key")
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, req)
				require.Equal(t, http.StatusOK, rec.Code, path)
				var got []define.AllowListUser
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got), path)
				require.Len(t, got, users.Len(), path)
				if !tc.empty {
					require.Equal(t, "member", got[0].UserID, path)
				}
			}
			for _, path := range []string{"/user?phone=13800138000", "/v1/user?phone=13800138000", "/v1/lookup?identifier=13800138000"} {
				req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
				req.Header.Set("X-API-Key", "contract-key")
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, req)
				require.Equal(t, tc.userCode, rec.Code, path)
				if !tc.empty {
					var got struct {
						UserID string `json:"user_id"`
					}
					require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got), path)
					require.Equal(t, "member", got.UserID, path)
				}
			}
		})
	}
}
