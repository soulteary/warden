package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingReader reports how many bytes were consumed from an endless body.
type countingReader struct {
	read int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	c.read += int64(len(p))
	return len(p), nil
}

func TestHMAC_OversizedBodyIsRejectedWithoutBufferingIt(t *testing.T) {
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

	// A known key ID with an arbitrary signature is enough to reach the body read.
	// The body is endless and has no declared length, as with chunked encoding.
	req := newV2Request("GET", "/log/level", "", secret, "key1", time.Now().Unix(), "nonce-oversized")
	body := &countingReader{}
	req.Body = io.NopCloser(body)
	req.ContentLength = -1

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.LessOrEqual(t, body.read, int64(maxSignedBodyBytes)+int64(512),
		"the pre-authentication read must stop just past the limit")
}

func TestHMAC_DeclaredOversizedBodyIsRejectedBeforeReading(t *testing.T) {
	secret := "v2-secret"
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
		ReplayGuard:           NewMemoryReplayGuard(0),
	}
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	oversized := strings.Repeat("a", maxSignedBodyBytes+1)
	req := newV2Request("POST", "/log/level", oversized, secret, "key1", time.Now().Unix(), "nonce-declared")
	body := &countingReader{}
	req.Body = io.NopCloser(body)

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Zero(t, body.read)
}

func TestHMAC_BodyAtLimitIsVerifiedAndRestored(t *testing.T) {
	secret := "v2-secret"
	cfg := HMACConfig{
		Keys:                  map[string]string{"key1": secret},
		TimestampToleranceSec: 60,
		ReplayGuard:           NewMemoryReplayGuard(0),
	}
	atLimit := strings.Repeat("a", maxSignedBodyBytes)
	var downstream string
	mw := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		downstream = string(data)
		w.WriteHeader(http.StatusOK)
	}))

	req := newV2Request("POST", "/log/level", atLimit, secret, "key1", time.Now().Unix(), "nonce-at-limit")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, atLimit, downstream)
}
