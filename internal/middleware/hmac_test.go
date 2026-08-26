package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// computeHMAC returns signature and timestamp for the given method, path, body, secret.
func computeHMAC(method, path, body, secret string, ts int64) string {
	pathWithQuery := path
	h := sha256.Sum256([]byte(body))
	bodyHash := hex.EncodeToString(h[:])
	message := method + pathWithQuery + strconv.FormatInt(ts, 10) + bodyHash
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestHMACAuth_NoHeaders_PassesThrough(t *testing.T) {
	cfg := HMACConfig{
		Keys:                  map[string]string{"k1": "secret1"},
		TimestampToleranceSec: 60,
	}
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/user", http.NoBody)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHMACAuth_EmptyKeys_NoHeaders_PassesThrough(t *testing.T) {
	cfg := HMACConfig{Keys: nil, TimestampToleranceSec: 60}
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/user", http.NoBody)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHMACAuth_ValidSignature_Returns200(t *testing.T) {
	secret := "test-secret"
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
	}
	ts := time.Now().Unix()
	path := "/user"
	sig := computeHMAC("GET", path, "", secret, ts)
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", path, http.NoBody)
	req.Header.Set(headerSignature, sig)
	req.Header.Set(headerTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(headerKeyID, "key1")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHMACAuth_ValidSignature_WithBody_Returns200(t *testing.T) {
	secret := "test-secret"
	body := `{"foo":"bar"}`
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
	}
	ts := time.Now().Unix()
	path := "/post"
	sig := computeHMAC("POST", path, body, secret, ts)
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		got, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, body, string(got))
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("POST", path, bytes.NewReader([]byte(body)))
	req.Header.Set(headerSignature, sig)
	req.Header.Set(headerTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(headerKeyID, "key1")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHMACAuth_InvalidSignature_Returns401(t *testing.T) {
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": "secret"},
		TimestampToleranceSec: 60,
	}
	ts := time.Now().Unix()
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/user", http.NoBody)
	req.Header.Set(headerSignature, "wrong-signature")
	req.Header.Set(headerTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(headerKeyID, "key1")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHMACAuth_UnknownKeyID_Returns401(t *testing.T) {
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": "secret"},
		TimestampToleranceSec: 60,
	}
	ts := time.Now().Unix()
	sig := computeHMAC("GET", "/user", "", "secret", ts)
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/user", http.NoBody)
	req.Header.Set(headerSignature, sig)
	req.Header.Set(headerTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(headerKeyID, "unknown-key")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHMACAuth_ExpiredTimestamp_Returns401(t *testing.T) {
	secret := "secret"
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
	}
	ts := time.Now().Unix() - 120 // 2 minutes ago
	sig := computeHMAC("GET", "/user", "", secret, ts)
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/user", http.NoBody)
	req.Header.Set(headerSignature, sig)
	req.Header.Set(headerTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(headerKeyID, "key1")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHMACAuth_InvalidTimestamp_Returns401(t *testing.T) {
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": "secret"},
		TimestampToleranceSec: 60,
	}
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/user", http.NoBody)
	req.Header.Set(headerSignature, "any")
	req.Header.Set(headerTimestamp, "not-a-number")
	req.Header.Set(headerKeyID, "key1")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHMACAuth_PathWithQuery(t *testing.T) {
	secret := "s"
	cfg := HMACConfig{
		Keys:                  map[string]string{"k": secret},
		TimestampToleranceSec: 60,
	}
	ts := time.Now().Unix()
	path := "/user?phone=13800138000"
	sig := computeHMAC("GET", path, "", secret, ts)
	nextCalled := false
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", path, http.NoBody)
	req.Header.Set(headerSignature, sig)
	req.Header.Set(headerTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(headerKeyID, "k")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServiceAuthChain_NoTLSNoHMAC_CallsAPIKeyMiddleware(t *testing.T) {
	apiKeyCalled := false
	apiKeyMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKeyCalled = true
			next.ServeHTTP(w, r)
		})
	}
	chain := ServiceAuthChain(HMACConfig{Keys: map[string]string{"k": "s"}, TimestampToleranceSec: 60}, apiKeyMw)
	nextCalled := false
	chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", http.NoBody))
	assert.True(t, apiKeyCalled)
	assert.True(t, nextCalled)
}

func TestServiceAuthChain_ValidHMAC_SkipsAPIKey(t *testing.T) {
	secret := "s"
	cfg := HMACConfig{
		Keys:                  map[string]string{"k": secret},
		TimestampToleranceSec: 60,
	}
	ts := time.Now().Unix()
	sig := computeHMAC("GET", "/", "", secret, ts)
	apiKeyCalled := false
	apiKeyMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKeyCalled = true
			next.ServeHTTP(w, r)
		})
	}
	chain := ServiceAuthChain(cfg, apiKeyMw)
	nextCalled := false
	h := chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set(headerSignature, sig)
	req.Header.Set(headerTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(headerKeyID, "k")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.False(t, apiKeyCalled, "HMAC valid should skip API Key middleware")
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServiceAuthChain_InvalidHMAC_Returns401(t *testing.T) {
	apiKeyMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
	chain := ServiceAuthChain(HMACConfig{
		Keys:                  map[string]string{"k": "s"},
		TimestampToleranceSec: 60,
	}, apiKeyMw)
	nextCalled := false
	h := chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set(headerSignature, "bad")
	req.Header.Set(headerTimestamp, strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set(headerKeyID, "k")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestVerifyHMAC_Valid(t *testing.T) {
	secret := "x"
	cfg := HMACConfig{
		Keys:                  map[string]string{"id": secret},
		TimestampToleranceSec: 60,
	}
	ts := time.Now().Unix()
	path := "/user"
	sig := computeHMAC("GET", path, "", secret, ts)
	req := httptest.NewRequest("GET", path, http.NoBody)
	req.Header.Set(headerTimestamp, strconv.FormatInt(ts, 10))
	ok := verifyHMAC(cfg, req, sig, strconv.FormatInt(ts, 10), "id")
	require.True(t, ok)
	// Body should be restored for downstream
	_, err := io.ReadAll(req.Body)
	assert.NoError(t, err)
}

func TestVerifyHMAC_InvalidKeyID(t *testing.T) {
	cfg := HMACConfig{Keys: map[string]string{"a": "s"}, TimestampToleranceSec: 60}
	req := httptest.NewRequest("GET", "/", http.NoBody)
	ok := verifyHMAC(cfg, req, "sig", strconv.FormatInt(time.Now().Unix(), 10), "unknown")
	assert.False(t, ok)
}
