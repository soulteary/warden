package middleware

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/serviceauth"
	"github.com/stretchr/testify/require"
)

func TestServicePoliciesEnforceAuthenticatedPermissions(t *testing.T) {
	const secret = "gateway-secret"
	cfg := HMACConfig{Keys: map[string]string{"gateway": secret, "unmapped": "other"}, ServicePolicies: map[string]serviceauth.Policy{"gateway": {Actions: []string{"directory:lookup"}}}, ReplayGuard: NewMemoryReplayGuard(100)}
	calls := 0
	handler := ServiceAuthChain(cfg, func(next http.Handler) http.Handler { return next })(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(200) }))
	for i, tc := range []struct {
		method, path, key, secret string
		status                    int
	}{
		{"GET", "/user", "gateway", secret, 200}, {"GET", "/v1/user", "gateway", secret, 200}, {"GET", "/v1/lookup", "gateway", secret, 200},
		{"GET", "/", "gateway", secret, 403}, {"GET", "/data.json", "gateway", secret, 403}, {"GET", "/v1/users", "gateway", secret, 403},
		{"GET", "/log/level", "gateway", secret, 403}, {"POST", "/log/level", "gateway", secret, 403}, {"PUT", "/log/level", "gateway", secret, 403},
		{"GET", "/metrics", "gateway", secret, 403}, {"POST", "/user", "gateway", secret, 403}, {"GET", "/unknown", "gateway", secret, 403},
		{"GET", "/user", "unmapped", "other", 403}, {"GET", "/user", "gateway", "wrong", 401},
	} {
		req := newV2Request(tc.method, tc.path, "", tc.secret, tc.key, time.Now().Unix(), "policy-"+strconv.Itoa(i))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, tc.status, rec.Code, tc.path)
	}
	require.Equal(t, 3, calls)
	for _, tc := range []string{"api", "cert", "missing-signature", "missing-time", "missing-key", "v1"} {
		req := newV2Request("GET", "/user", "", secret, "gateway", time.Now().Unix(), tc)
		switch tc {
		case "api":
			req = httptest.NewRequest("GET", "/user", http.NoBody)
			req.Header.Set("X-API-Key", "key")
		case "cert":
			req = httptest.NewRequest("GET", "/user", http.NoBody)
			req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{}}}
		case "missing-signature":
			req.Header.Del(headerSignature)
		case "missing-time":
			req.Header.Del(headerTimestamp)
		case "missing-key":
			req.Header.Del(headerKeyID)
		case "v1":
			req.Header.Set(headerVersion, signatureV1)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, 401, rec.Code, tc)
	}
	require.Equal(t, 3, calls)
}

func TestServicePoliciesAllowExplicitOperationsAndDenyEmptyPolicy(t *testing.T) {
	cfg := HMACConfig{Keys: map[string]string{"admin": "secret"}, ServicePolicies: map[string]serviceauth.Policy{"admin": {Actions: []string{"directory:list", "admin:loglevel", "metrics:read"}}}, ReplayGuard: NewMemoryReplayGuard(100)}
	handler := ServiceAuthChain(cfg, func(next http.Handler) http.Handler { return next })(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
	for i, path := range []string{"/", "/log/level", "/metrics"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, newV2Request("GET", path, "", "secret", "admin", time.Now().Unix(), strconv.Itoa(i)))
		require.Equal(t, 204, rec.Code)
	}
	cfg.ServicePolicies = map[string]serviceauth.Policy{}
	handler = ServiceAuthChain(cfg, func(next http.Handler) http.Handler { return next })(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { t.Fatal("empty policy reached handler") }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, newV2Request("GET", "/user", "", "secret", "admin", time.Now().Unix(), "empty"))
	require.Equal(t, 403, rec.Code)
}
