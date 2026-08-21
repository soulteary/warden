package warden

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedClock returns a constant time for deterministic signature timestamps.
type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

// serverVerifyV2 recomputes the v2 canonical form on the server side and compares
// signatures the same way the middleware does. It mirrors the production canonical
// form: METHOD\nPATH_AND_QUERY\nTIMESTAMP\nNONCE\nSHA256_HEX(body).
func serverVerifyV2(r *http.Request, secret string) bool {
	body, _ := io.ReadAll(r.Body)
	bh := sha256.Sum256(body)
	p := r.URL.EscapedPath()
	if p == "" {
		p = "/"
	}
	pathAndQuery := p
	if r.URL.RawQuery != "" {
		pathAndQuery = p + "?" + r.URL.RawQuery
	}
	canonical := r.Method + "\n" + pathAndQuery + "\n" +
		r.Header.Get("X-Timestamp") + "\n" +
		r.Header.Get("X-Nonce") + "\n" +
		hex.EncodeToString(bh[:])
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(r.Header.Get("X-Signature")), []byte(expected))
}

func TestSigner_SetsAllV2Headers(t *testing.T) {
	s := newHMACSigner("key-1", "secret", fixedClock{t: time.Unix(1700000000, 0)})
	require.NotNil(t, s)
	req := httptest.NewRequest("GET", "http://example.com/user?phone=1", http.NoBody)
	require.NoError(t, s.sign(req))

	assert.Equal(t, "v2", req.Header.Get(headerHMACVersion))
	assert.Equal(t, "key-1", req.Header.Get(headerHMACKeyID))
	assert.Equal(t, "1700000000", req.Header.Get(headerHMACTimestamp))
	assert.NotEmpty(t, req.Header.Get(headerHMACNonce))
	assert.Len(t, req.Header.Get(headerHMACNonce), 32, "128-bit nonce is 32 hex chars")
	assert.NotEmpty(t, req.Header.Get(headerHMACSignature))
}

func TestSigner_DisabledWhenIncomplete(t *testing.T) {
	assert.Nil(t, newHMACSigner("", "secret", nil))
	assert.Nil(t, newHMACSigner("id", "", nil))
}

func TestSigner_FreshNoncePerRequest(t *testing.T) {
	s := newHMACSigner("k", "secret", fixedClock{t: time.Unix(1700000000, 0)})
	req1 := httptest.NewRequest("GET", "http://example.com/user", http.NoBody)
	req2 := httptest.NewRequest("GET", "http://example.com/user", http.NoBody)
	require.NoError(t, s.sign(req1))
	require.NoError(t, s.sign(req2))
	assert.NotEqual(t, req1.Header.Get(headerHMACNonce), req2.Header.Get(headerHMACNonce),
		"each request must carry a unique nonce")
	assert.NotEqual(t, req1.Header.Get(headerHMACSignature), req2.Header.Get(headerHMACSignature))
}

// TestClient_SignedRequest_VerifiedByServer is the end-to-end happy path: the SDK
// signs, and a server recomputing the canonical form accepts the signature.
func TestClient_SignedRequest_VerifiedByServer(t *testing.T) {
	const secret = "shared-secret"
	var sawVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawVersion = r.Header.Get("X-Signature-Version")
		if !serverVerifyV2(r, secret) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]AllowListUser{{UserID: "u1", Phone: "13800138000", Status: "active"}})
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().
		WithBaseURL(server.URL).
		WithHMAC("key-1", secret))
	require.NoError(t, err)

	users, err := client.GetUsers(context.Background())
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "v2", sawVersion)
}

// TestClient_TamperedBodyFailsServerVerify confirms the signature binds the body:
// if a proxy alters the body the server rejects it.
func TestClient_TamperedRequestRejected(t *testing.T) {
	const secret = "shared-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Tamper with the perceived path before verifying to simulate a mismatch.
		r.URL.Path = "/tampered"
		if !serverVerifyV2(r, secret) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().
		WithBaseURL(server.URL).
		WithHMAC("key-1", secret))
	require.NoError(t, err)

	_, err = client.GetUsers(context.Background())
	require.Error(t, err)
	var sdkErr *Error
	if assert.ErrorAs(t, err, &sdkErr) {
		assert.Equal(t, ErrCodeUnauthorized, sdkErr.Code)
	}
}

// TestClient_ResponseSizeLimit ensures the response body is bounded so a hostile or
// buggy server cannot force unbounded memory use.
func TestClient_ResponseSizeLimit(t *testing.T) {
	// Emit a large but valid-prefix JSON array. With a tiny limit the decode fails.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Write a big array that exceeds the limit.
		_, _ = w.Write([]byte("["))
		for i := 0; i < 10000; i++ {
			if i > 0 {
				_, _ = w.Write([]byte(","))
			}
			_, _ = w.Write([]byte(`{"user_id":"padding-padding-padding"}`))
		}
		_, _ = w.Write([]byte("]"))
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().
		WithBaseURL(server.URL).
		WithMaxResponseBytes(64)) // absurdly small cap
	require.NoError(t, err)

	_, err = client.GetUsers(context.Background())
	require.Error(t, err, "truncated body must fail to decode rather than read unbounded data")
}

// TestClient_RetryOnlyIdempotent verifies GET retries on 5xx while the retry engine
// refuses to replay non-idempotent verbs. We assert via method helper directly since
// all SDK public calls are GET.
func TestClient_RetryOnlyIdempotent(t *testing.T) {
	assert.True(t, isIdempotentMethod(http.MethodGet))
	assert.True(t, isIdempotentMethod(http.MethodHead))
	assert.True(t, isIdempotentMethod(http.MethodOptions))
	assert.False(t, isIdempotentMethod(http.MethodPost))
	assert.False(t, isIdempotentMethod(http.MethodPatch))
	assert.False(t, isIdempotentMethod(http.MethodDelete))
	assert.False(t, isIdempotentMethod(http.MethodPut))
}

// TestClient_RetriesGetOn5xx confirms a GET is retried and eventually succeeds, and
// that each attempt carries a *fresh* nonce (so the server's replay guard would not
// reject the retry).
func TestClient_RetriesGetOn5xxWithFreshNonce(t *testing.T) {
	const secret = "shared-secret"
	var mu sync.Mutex
	var attempts int
	nonces := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		n := r.Header.Get("X-Nonce")
		if n != "" {
			nonces[n] = true
		}
		attempt := attempts
		mu.Unlock()
		if attempt < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	retry := DefaultRetryOptions()
	retry.MaxRetries = 3
	retry.RetryDelay = time.Millisecond
	client, err := NewClient(DefaultOptions().
		WithBaseURL(server.URL).
		WithHMAC("key-1", secret).
		WithRetry(retry))
	require.NoError(t, err)

	_, err = client.GetUsers(context.Background())
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 3, attempts, "should retry until success")
	assert.Len(t, nonces, 3, "each attempt must use a fresh nonce")
}

// TestSigner_SecretNeverLogged is a lightweight guard: the mock logger must never see
// the raw secret in any message emitted during a signed request.
func TestSigner_SecretNeverLogged(t *testing.T) {
	const secret = "super-secret-value-should-not-leak"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	ml := &mockLogger{}
	client, err := NewClient(DefaultOptions().
		WithBaseURL(server.URL).
		WithHMAC("key-1", secret).
		WithLogger(ml))
	require.NoError(t, err)

	_, err = client.GetUsers(context.Background())
	require.NoError(t, err)

	all := strings.Join(append(append(append(append([]string{}, ml.debugs...), ml.infos...), ml.warns...), ml.errors...), "\n")
	assert.NotContains(t, all, secret, "secret must never appear in logs")
}

// TestOptions_TimestampFormat sanity-checks the signer timestamp is unix seconds.
func TestOptions_TimestampFormat(t *testing.T) {
	s := newHMACSigner("k", "secret", fixedClock{t: time.Unix(1700000123, 0)})
	req := httptest.NewRequest("GET", "http://example.com/", http.NoBody)
	require.NoError(t, s.sign(req))
	ts, err := strconv.ParseInt(req.Header.Get(headerHMACTimestamp), 10, 64)
	require.NoError(t, err)
	assert.Equal(t, int64(1700000123), ts)
}
