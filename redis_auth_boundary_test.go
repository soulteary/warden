package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	middlewarekit "github.com/soulteary/middleware-kit/v3"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/stretchr/testify/require"
)

func authBoundaryApp(t *testing.T, enabled bool, client *redis.Client) *App {
	t.Helper()
	for _, key := range []string{"IP_WHITELIST", "HEALTH_CHECK_IP_WHITELIST", "TRUSTED_PROXY_IPS"} {
		t.Setenv(key, "")
	}
	prepared, snap := preparedTestSnapshot(t, []define.AllowListUser{{Mail: "member@example.com", UserID: "member"}})
	users := cache.NewSafeUserCache()
	users.ApplyPrepared(prepared)
	store := newSnapshotStore()
	store.Store(snap)
	limiter := middlewarekit.NewRateLimiter(middlewarekit.DefaultRateLimiterConfig())
	t.Cleanup(limiter.Stop)
	return &App{userCache: users, snapshots: store, redisEnabled: enabled, redisClient: client, hmacKeys: map[string]string{"boundary": "test-secret"}, hmacToleranceSec: 60, snapshotMaxAge: time.Minute, appMode: "DEFAULT", environment: "development", apiKey: "api-key", rateLimiter: limiter, log: logger.GetLoggerKit()}
}

func boundarySignedRequest(t *testing.T, nonce string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/users", http.NoBody)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	bodyHash := sha256.Sum256(nil)
	canonical := "GET\n/v1/users\nboundary\n" + ts + "\n" + nonce + "\n" + hex.EncodeToString(bodyHash[:])
	mac := hmac.New(sha256.New, []byte("test-secret"))
	_, err := mac.Write([]byte(canonical))
	require.NoError(t, err)
	req.Header.Set("X-Signature", hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Key-Id", "boundary")
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature-Version", "v2")
	return req
}

func TestRequiredRedisReplayGuardRejectsUninitializedClient(t *testing.T) {
	mux := registerRoutes(authBoundaryApp(t, true, nil))
	health := httptest.NewRecorder()
	mux.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", http.NoBody))
	require.Equal(t, http.StatusServiceUnavailable, health.Code)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, boundarySignedRequest(t, t.Name()))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	// Independent authentication methods must remain available.
	req := httptest.NewRequest(http.MethodGet, "/v1/users", http.NoBody)
	req.Header.Set("X-API-Key", "api-key")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	req = httptest.NewRequest(http.MethodGet, "/v1/users", http.NoBody)
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{}}}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestDisabledRedisKeepsLocalReplayProtection(t *testing.T) {
	mux := registerRoutes(authBoundaryApp(t, false, nil))
	for _, want := range []int{http.StatusOK, http.StatusUnauthorized} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, boundarySignedRequest(t, t.Name()))
		require.Equal(t, want, rec.Code)
	}
}

func TestRequiredRedisReplayProtectionSharedAcrossRouters(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	first := registerRoutes(authBoundaryApp(t, true, client))
	second := registerRoutes(authBoundaryApp(t, true, client))
	rec := httptest.NewRecorder()
	first.ServeHTTP(rec, boundarySignedRequest(t, t.Name()))
	require.Equal(t, http.StatusOK, rec.Code)
	rec = httptest.NewRecorder()
	second.ServeHTTP(rec, boundarySignedRequest(t, t.Name()))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	// A connected client which later loses Redis also fails closed.
	mr.Close()
	rec = httptest.NewRecorder()
	first.ServeHTTP(rec, boundarySignedRequest(t, t.Name()+"-outage"))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
