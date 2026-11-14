package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A request without X-Signature-Version is read as v2 when it carries a nonce
// and its signature is a v2 signature. A v1 signer that happens to send a
// nonce header is still accepted as v1, but only when v1 is allowed.
func TestHMACUnversionedRequestWithNonce(t *testing.T) {
	const keyID, shared = "gateway", "unversioned-key-material"
	now := time.Now().Unix()
	newChain := func(allowV1 bool, v1Used *int) http.Handler {
		cfg := HMACConfig{
			Keys:        map[string]string{keyID: shared},
			ReplayGuard: NewMemoryReplayGuard(0),
			AllowV1:     &allowV1,
			OnV1Used:    func() { *v1Used++ },
		}
		return HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	}
	serveOnce := func(h http.Handler, req *http.Request) int {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req.Clone(req.Context()))
		return rec.Code
	}
	v2WithoutVersion := func(nonce string) *http.Request {
		req := newV2Request(http.MethodGet, "/user?phone=13800138000", "", shared, keyID, now, nonce)
		req.Header.Del(headerVersion)
		return req
	}
	v1WithNonce := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/user?phone=13800138000", http.NoBody)
		req.Header.Set(headerSignature, computeHMAC(http.MethodGet, "/user?phone=13800138000", "", shared, now))
		req.Header.Set(headerTimestamp, strconv.FormatInt(now, 10))
		req.Header.Set(headerKeyID, keyID)
		req.Header.Set(headerNonce, "sent-by-a-v1-signer")
		return req
	}

	var v1Used int
	strict := newChain(false, &v1Used)
	// Accepted as v2, and the nonce is single use as for an explicit v2 request.
	req := v2WithoutVersion("unversioned-v2")
	require.Equal(t, http.StatusOK, serveOnce(strict, req))
	require.Equal(t, http.StatusUnauthorized, serveOnce(strict, req))
	// A v1 signature is refused while v1 is not allowed, nonce or not.
	require.Equal(t, http.StatusUnauthorized, serveOnce(strict, v1WithNonce()))
	require.Zero(t, v1Used)

	lenient := newChain(true, &v1Used)
	require.Equal(t, http.StatusOK, serveOnce(lenient, v2WithoutVersion("unversioned-v2-lenient")))
	require.Zero(t, v1Used, "a v2 signature must not be counted as v1 use")
	require.Equal(t, http.StatusOK, serveOnce(lenient, v1WithNonce()))
	require.Equal(t, 1, v1Used)
}
