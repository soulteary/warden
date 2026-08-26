package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type replaySetNXResult struct {
	err      error
	key      string
	ttl      time.Duration
	inserted bool
}

func replaySeen(t *testing.T, guard ReplayGuard, key string, ttl time.Duration) bool {
	t.Helper()
	seen, err := guard.SeenBefore(key, ttl)
	require.NoError(t, err)
	return seen
}

func (f *replaySetNXResult) SetNX(_ context.Context, key string, _ interface{}, expiration time.Duration) *redis.BoolCmd {
	f.key = key
	f.ttl = expiration
	return redis.NewBoolResult(f.inserted, f.err)
}

// computeHMACv2 builds a v2 signature over the canonical form
// METHOD\nPATH_AND_QUERY\nTIMESTAMP\nNONCE\nSHA256_HEX(body).
func computeHMACv2(method, pathAndQuery, body, secret string, ts int64, nonce string) string {
	h := sha256.Sum256([]byte(body))
	bodyHash := hex.EncodeToString(h[:])
	canonical := method + "\n" + pathAndQuery + "\n" + strconv.FormatInt(ts, 10) + "\n" + nonce + "\n" + bodyHash
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// newV2Request builds a request carrying valid v2 headers.
func newV2Request(method, target, body, secret, keyID string, ts int64, nonce string) *http.Request {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, http.NoBody)
	} else {
		req = httptest.NewRequest(method, target, bytes.NewReader([]byte(body)))
	}
	sig := computeHMACv2(method, escapedPathAndQuery(req), body, secret, ts, nonce)
	req.Header.Set(headerSignature, sig)
	req.Header.Set(headerTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(headerKeyID, keyID)
	req.Header.Set(headerNonce, nonce)
	req.Header.Set(headerVersion, signatureV2)
	return req
}

func TestHMACv2_ValidSignature_Returns200(t *testing.T) {
	secret := "v2-secret"
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
		ReplayGuard:           NewMemoryReplayGuard(0),
	}
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := newV2Request("GET", "/user?phone=13800138000", "", secret, "key1", time.Now().Unix(), "nonce-abc")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHMACv2_ValidWithBody_Returns200(t *testing.T) {
	secret := "v2-secret"
	body := `{"a":1}`
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
		ReplayGuard:           NewMemoryReplayGuard(0),
	}
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := newV2Request("POST", "/post", body, secret, "key1", time.Now().Unix(), "n1")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestHMACv2_TamperedFields verifies that altering method/path/query/body/timestamp/
// nonce all break verification.
func TestHMACv2_TamperedFields(t *testing.T) {
	secret := "v2-secret"
	keyID := "key1"
	base := func() HMACConfig {
		return HMACConfig{
			Keys:                  map[string]string{keyID: secret},
			TimestampToleranceSec: 60,
			ReplayGuard:           NewMemoryReplayGuard(0),
		}
	}

	cases := []struct { //nolint:govet // fieldalignment: table-test readability
		name   string
		mutate func(req *http.Request)
	}{
		{"tamper_method", func(req *http.Request) { req.Method = "POST" }},
		{"tamper_path", func(req *http.Request) { req.URL.Path = "/other" }},
		{"tamper_query", func(req *http.Request) { req.URL.RawQuery = "phone=99999999999" }},
		{"tamper_timestamp", func(req *http.Request) {
			req.Header.Set(headerTimestamp, strconv.FormatInt(time.Now().Unix()-1, 10))
		}},
		{"tamper_nonce", func(req *http.Request) { req.Header.Set(headerNonce, "different-nonce") }},
		{"tamper_signature", func(req *http.Request) { req.Header.Set(headerSignature, "deadbeef") }},
		{"tamper_body", func(req *http.Request) {
			req.Body = http.NoBody
			req.ContentLength = 0
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base()
			nextCalled := false
			mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			}))
			// Sign with a body so the tamper_body case has something to strip.
			req := newV2Request("GET", "/user?phone=13800138000", `{"x":1}`, secret, keyID, time.Now().Unix(), "nonce-1")
			tc.mutate(req)
			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)
			assert.False(t, nextCalled, "tampered request must not reach handler")
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

// TestHMACv2_ReplayRejected ensures a captured, still-in-window request cannot be
// replayed: the second identical request is rejected.
func TestHMACv2_ReplayRejected(t *testing.T) {
	secret := "v2-secret"
	replayCount := 0
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
		ReplayGuard:           NewMemoryReplayGuard(0),
		OnReplayRejected:      func() { replayCount++ },
	}
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ts := time.Now().Unix()
	nonce := "fixed-nonce"

	// First request succeeds.
	req1 := newV2Request("GET", "/user", "", secret, "key1", ts, nonce)
	rec1 := httptest.NewRecorder()
	mw.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	// Replay with identical signature/nonce is rejected.
	req2 := newV2Request("GET", "/user", "", secret, "key1", ts, nonce)
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
	assert.Equal(t, 1, replayCount, "replay observer should fire once")
}

func TestHMACv2_ReplayStoreErrorDoesNotCountAsReplay(t *testing.T) {
	secret := "v2-secret"
	replayCount := 0
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
		ReplayGuard:           newRedisReplayGuard(&replaySetNXResult{err: errors.New("redis unavailable")}),
		OnReplayRejected:      func() { replayCount++ },
	}
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newV2Request("GET", "/user", "", secret, "key1", time.Now().Unix(), "nonce")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Zero(t, replayCount, "replay store failures must not increment the replay counter")
}

// TestHMACv2_DifferentNonce_NotReplay ensures a fresh nonce (as a retry would use)
// is accepted even with the same timestamp.
func TestHMACv2_DifferentNonce_NotReplay(t *testing.T) {
	secret := "v2-secret"
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
		ReplayGuard:           NewMemoryReplayGuard(0),
	}
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	ts := time.Now().Unix()

	req1 := newV2Request("GET", "/user", "", secret, "key1", ts, "nonce-A")
	rec1 := httptest.NewRecorder()
	mw.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	req2 := newV2Request("GET", "/user", "", secret, "key1", ts, "nonce-B")
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

// TestHMACv2_TimestampSkewBoundary tests the tolerance boundary: within tolerance is
// accepted, just outside is rejected, on both sides.
func TestHMACv2_TimestampSkewBoundary(t *testing.T) {
	secret := "v2-secret"
	tol := 60
	cases := []struct { //nolint:govet // fieldalignment: table-test readability
		name     string
		offset   int64
		wantOK   bool
		useNonce string
	}{
		{"within_past", -int64(tol) + 1, true, "n-a"},
		{"within_future", int64(tol) - 1, true, "n-b"},
		{"boundary_past", -int64(tol), true, "n-c"},
		{"boundary_future", int64(tol), true, "n-d"},
		{"outside_past", -int64(tol) - 5, false, "n-e"},
		{"outside_future", int64(tol) + 5, false, "n-f"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := HMACConfig{
				Keys:                  map[string]string{"key1": secret},
				TimestampToleranceSec: tol,
				ReplayGuard:           NewMemoryReplayGuard(0),
			}
			mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			ts := time.Now().Unix() + tc.offset
			req := newV2Request("GET", "/user", "", secret, "key1", ts, tc.useNonce)
			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)
			if tc.wantOK {
				assert.Equal(t, http.StatusOK, rec.Code)
			} else {
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			}
		})
	}
}

// TestHMACv2_MissingNonce_Rejected ensures an explicit v2 request without a nonce is
// rejected (nonce is mandatory for v2).
func TestHMACv2_MissingNonce_Rejected(t *testing.T) {
	secret := "v2-secret"
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
	}
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	ts := time.Now().Unix()
	req := newV2Request("GET", "/user", "", secret, "key1", ts, "n")
	req.Header.Del(headerNonce) // strip nonce but keep version=v2
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestHMAC_V1StillWorks_AndFiresDeprecation ensures the legacy v1 form still verifies
// during migration and triggers the deprecation observer.
func TestHMAC_V1StillWorks_AndFiresDeprecation(t *testing.T) {
	secret := "legacy-secret"
	v1Used := 0
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
		OnV1Used:              func() { v1Used++ },
	}
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	ts := time.Now().Unix()
	path := "/user"
	sig := computeHMAC("GET", path, "", secret, ts) // v1 canonical form
	req := httptest.NewRequest("GET", path, http.NoBody)
	req.Header.Set(headerSignature, sig)
	req.Header.Set(headerTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(headerKeyID, "key1")
	// No version header, no nonce => v1 path.
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, v1Used, "v1 deprecation observer should fire")
}

// TestMemoryReplayGuard_Concurrent exercises the guard under concurrent access to
// ensure exactly one caller "wins" a given key and the rest see a replay.
func TestMemoryReplayGuard_Concurrent(t *testing.T) {
	g := NewMemoryReplayGuard(0)
	const goroutines = 64
	var wg sync.WaitGroup
	var mu sync.Mutex
	firstSeen := 0
	replays := 0
	guardErrors := 0
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			replayed, err := g.SeenBefore("same-key", time.Minute)
			mu.Lock()
			if err != nil {
				guardErrors++
			}
			if replayed {
				replays++
			} else {
				firstSeen++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	assert.Zero(t, guardErrors)
	assert.Equal(t, 1, firstSeen, "exactly one goroutine should record the key first")
	assert.Equal(t, goroutines-1, replays, "the rest should be treated as replays")
}

// TestMemoryReplayGuard_TTLExpiry verifies an entry is forgotten after its TTL so the
// key can be used again (bounded memory + correctness after the window).
func TestMemoryReplayGuard_TTLExpiry(t *testing.T) {
	g := NewMemoryReplayGuard(0)
	// Freeze/advance a fake clock.
	now := time.Now()
	g.nowFn = func() time.Time { return now }

	assert.False(t, replaySeen(t, g, "k", 10*time.Millisecond))
	assert.True(t, replaySeen(t, g, "k", 10*time.Millisecond), "still within TTL => replay")

	// Advance beyond TTL and the gcEvery interval.
	now = now.Add(2 * time.Second)
	assert.False(t, replaySeen(t, g, "k", 10*time.Millisecond), "after TTL the key is fresh again")
}

// TestMemoryReplayGuard_MaxSizeFailsSafe ensures the bounded guard rejects new keys
// (fails safe) rather than growing unboundedly once the cap is hit.
func TestMemoryReplayGuard_MaxSizeFailsSafe(t *testing.T) {
	g := NewMemoryReplayGuard(2)
	assert.False(t, replaySeen(t, g, "a", time.Minute))
	assert.False(t, replaySeen(t, g, "b", time.Minute))
	// Cap reached; a brand-new key is treated as a replay (rejected).
	assert.True(t, replaySeen(t, g, "c", time.Minute))
	// An already-tracked key still reports replay.
	assert.True(t, replaySeen(t, g, "a", time.Minute))
}

func TestRedisReplayGuard(t *testing.T) {
	store := &replaySetNXResult{inserted: true}
	guard := newRedisReplayGuard(store)
	assert.False(t, replaySeen(t, guard, "key-id:nonce", 30*time.Second))
	assert.Equal(t, 30*time.Second, store.ttl)
	assert.Regexp(t, `^warden:hmac:replay:[0-9a-f]{64}$`, store.key)

	store.inserted = false
	assert.True(t, replaySeen(t, guard, "key-id:nonce", 30*time.Second))
}

func TestRedisReplayGuard_FailsClosed(t *testing.T) {
	store := &replaySetNXResult{err: errors.New("redis unavailable")}
	guard := newRedisReplayGuard(store)
	seen, err := guard.SeenBefore("key-id:nonce", time.Minute)
	assert.False(t, seen)
	require.Error(t, err)
	seen, err = newRedisReplayGuard(nil).SeenBefore("key-id:nonce", time.Minute)
	assert.False(t, seen)
	require.Error(t, err)
}
