package main

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/middleware"
)

const (
	rateLimitTestPath = "/user?user_id=member"
	rateLimitIPA      = "198.51.100.30:1234"
	rateLimitIPB      = "198.51.100.31:1234"
)

// rateLimitedApp returns an app that allows ipRate requests per window to a
// client IP and credentialRate to a credential. The window outlasts the test.
// The API key is "key" and the HMAC key "gateway" signs policySignedRequest.
func rateLimitedApp(t *testing.T, ipRate, credentialRate int) *App {
	t.Helper()
	app := responseMetadataApp(t)
	app.hmacKeys = map[string]string{"gateway": "secret"}
	app.hmacToleranceSec = 60
	limits := middleware.NewRateLimits(ipRate, credentialRate, time.Minute)
	t.Cleanup(limits.Stop)
	app.rateLimiter = limits
	return app
}

func serve(mux http.Handler, req *http.Request, remoteAddr string) *httptest.ResponseRecorder {
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func apiKeyRequest(path, key string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	return req
}

func TestAuthenticatedRequestsAreNotLimitedPerIP(t *testing.T) {
	mux := registerRoutes(rateLimitedApp(t, 2, 10))
	for i := 0; i < 10; i++ {
		require.Equal(t, http.StatusOK, serve(mux, apiKeyRequest(rateLimitTestPath, "key"), rateLimitIPA).Code, "request %d", i+1)
	}
	// They left the per-IP budget untouched: two failures are still answered
	// with 401 before the address is refused.
	for i := 0; i < 2; i++ {
		require.Equal(t, http.StatusUnauthorized, serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPA).Code)
	}
	require.Equal(t, http.StatusTooManyRequests, serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPA).Code)
}

func TestCredentialsHaveIndependentBudgets(t *testing.T) {
	mux := registerRoutes(rateLimitedApp(t, 100, 2))
	mtls := func(der string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, rateLimitTestPath, http.NoBody)
		req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{Raw: []byte(der)}}}
		return req
	}
	credentials := map[string]func(i int) *http.Request{
		"api key":       func(int) *http.Request { return apiKeyRequest(rateLimitTestPath, "key") },
		"hmac key":      func(i int) *http.Request { return policySignedRequest(rateLimitTestPath, t.Name()+strconv.Itoa(i)) },
		"certificate a": func(int) *http.Request { return mtls("certificate-a") },
		"certificate b": func(int) *http.Request { return mtls("certificate-b") },
	}
	// All from one address: every credential gets its full budget, and
	// exhausting one returns 429 for that credential only.
	for name, request := range credentials {
		require.Equal(t, http.StatusOK, serve(mux, request(0), rateLimitIPA).Code, name)
		require.Equal(t, http.StatusOK, serve(mux, request(1), rateLimitIPA).Code, name)
		rec := serve(mux, request(2), rateLimitIPA)
		require.Equal(t, http.StatusTooManyRequests, rec.Code, name)
		require.Equal(t, "Too Many Requests\n", rec.Body.String(), name)
		require.Equal(t, "60", rec.Header().Get("Retry-After"), name)
	}
}

func TestCredentialBudgetIsSharedAcrossIPs(t *testing.T) {
	mux := registerRoutes(rateLimitedApp(t, 100, 2))
	signed := func(i int) *http.Request { return policySignedRequest(rateLimitTestPath, t.Name()+strconv.Itoa(i)) }
	require.Equal(t, http.StatusOK, serve(mux, signed(1), rateLimitIPA).Code)
	require.Equal(t, http.StatusOK, serve(mux, signed(2), rateLimitIPB).Code)
	require.Equal(t, http.StatusTooManyRequests, serve(mux, signed(3), rateLimitIPA).Code)
	require.Equal(t, http.StatusTooManyRequests, serve(mux, signed(4), rateLimitIPB).Code)
	// Requests refused by the credential limit do not spend the per-IP budget.
	require.Equal(t, http.StatusUnauthorized, serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPA).Code)
}

// The API key is shared by every caller, so one caller must not be able to
// exhaust it for the others: its budget is kept per client address.
func TestAPIKeyBudgetIsPerClientAddress(t *testing.T) {
	mux := registerRoutes(rateLimitedApp(t, 100, 2))
	for i := 0; i < 2; i++ {
		require.Equal(t, http.StatusOK, serve(mux, apiKeyRequest(rateLimitTestPath, "key"), rateLimitIPA).Code)
	}
	require.Equal(t, http.StatusTooManyRequests, serve(mux, apiKeyRequest(rateLimitTestPath, "key"), rateLimitIPA).Code)
	for i := 0; i < 2; i++ {
		require.Equal(t, http.StatusOK, serve(mux, apiKeyRequest(rateLimitTestPath, "key"), rateLimitIPB).Code)
	}
	require.Equal(t, http.StatusTooManyRequests, serve(mux, apiKeyRequest(rateLimitTestPath, "key"), rateLimitIPB).Code)
}

func TestFailedAuthenticationIsLimitedPerIP(t *testing.T) {
	mux := registerRoutes(rateLimitedApp(t, 3, 100))
	failures := []func(i int) *http.Request{
		func(int) *http.Request { return apiKeyRequest(rateLimitTestPath, "") },
		func(int) *http.Request { return apiKeyRequest(rateLimitTestPath, "wrong") },
		func(i int) *http.Request {
			req := policySignedRequest(rateLimitTestPath, t.Name()+strconv.Itoa(i))
			req.Header.Set("X-Signature", "00")
			return req
		},
	}
	for i, request := range failures {
		require.Equal(t, http.StatusUnauthorized, serve(mux, request(i), rateLimitIPA).Code, "failure %d", i+1)
	}
	for i, request := range failures {
		rec := serve(mux, request(i), rateLimitIPA)
		require.Equal(t, http.StatusTooManyRequests, rec.Code, "refusal %d", i+1)
		require.Equal(t, "Too Many Requests\n", rec.Body.String())
		require.Equal(t, "60", rec.Header().Get("Retry-After"))
	}
	// Another address has its own budget.
	require.Equal(t, http.StatusUnauthorized, serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPB).Code)
	require.Equal(t, http.StatusOK, serve(mux, apiKeyRequest(rateLimitTestPath, "key"), rateLimitIPB).Code)
}

// An address that has spent its budget on failures is refused before the
// credential is evaluated. A valid credential therefore receives 429 as well:
// answering it would tell a guesser which credential is valid at an unlimited
// rate.
func TestExhaustedIPIsRefusedBeforeAuthentication(t *testing.T) {
	mux := registerRoutes(rateLimitedApp(t, 1, 100))
	require.Equal(t, http.StatusUnauthorized, serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPA).Code)

	require.Equal(t, http.StatusTooManyRequests, serve(mux, apiKeyRequest(rateLimitTestPath, "key"), rateLimitIPA).Code)

	// A valid signature refused here was not evaluated: its nonce is unused,
	// so the same signed request is accepted from another address. Had
	// authentication run, the replay guard would reject the second use.
	signed := policySignedRequest(rateLimitTestPath, t.Name())
	require.Equal(t, http.StatusTooManyRequests, serve(mux, signed, rateLimitIPA).Code)
	require.Equal(t, http.StatusOK, serve(mux, signed, rateLimitIPB).Code)
}

func TestServicePolicyDenialsAreLimitedPerIP(t *testing.T) {
	t.Setenv("WARDEN_SERVICE_POLICIES", `{"gateway":{"actions":["directory:lookup"]}}`)
	mux := registerRoutes(rateLimitedApp(t, 2, 3))
	// Allowed action: counted against the credential.
	for i := 0; i < 3; i++ {
		require.Equal(t, http.StatusOK, serve(mux, policySignedRequest(rateLimitTestPath, t.Name()+"ok"+strconv.Itoa(i)), rateLimitIPA).Code)
	}
	require.Equal(t, http.StatusTooManyRequests, serve(mux, policySignedRequest(rateLimitTestPath, t.Name()+"over"), rateLimitIPA).Code)
	// Denied action: 403 from the auth layer, counted against the address.
	for i := 0; i < 2; i++ {
		require.Equal(t, http.StatusForbidden, serve(mux, policySignedRequest("/v1/users", t.Name()+"denied"+strconv.Itoa(i)), rateLimitIPB).Code)
	}
	require.Equal(t, http.StatusTooManyRequests, serve(mux, policySignedRequest("/v1/users", t.Name()+"refused"), rateLimitIPB).Code)
}

func TestHealthRoutesAreNeverRateLimited(t *testing.T) {
	mux := registerRoutes(rateLimitedApp(t, 1, 1))
	require.Equal(t, http.StatusUnauthorized, serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPA).Code)
	require.Equal(t, http.StatusTooManyRequests, serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPA).Code)
	for _, path := range []string{define.PATH_HEALTH, define.PATH_HEALTHCHECK, define.PATH_V1_HEALTH, define.PATH_V1_HEALTHCHECK} {
		for i := 0; i < 5; i++ {
			require.Equal(t, http.StatusOK, serve(mux, apiKeyRequest(path, "key"), rateLimitIPA).Code, path)
		}
	}
}

func TestAnonymousRoutesAreLimitedPerIP(t *testing.T) {
	t.Setenv("WARDEN_METRICS_REQUIRE_AUTH", "")
	for _, path := range []string{define.PATH_METRICS, "/no-such-route"} {
		t.Run(path, func(t *testing.T) {
			want := http.StatusOK
			if path != define.PATH_METRICS {
				want = http.StatusNotFound
			}
			mux := registerRoutes(rateLimitedApp(t, 2, 100))
			// A credential on an anonymous route is not evaluated and does not
			// move the request to the credential budget.
			require.Equal(t, want, serve(mux, apiKeyRequest(path, ""), rateLimitIPA).Code)
			require.Equal(t, want, serve(mux, apiKeyRequest(path, "key"), rateLimitIPA).Code)
			rec := serve(mux, apiKeyRequest(path, "key"), rateLimitIPA)
			require.Equal(t, http.StatusTooManyRequests, rec.Code)
			require.Equal(t, "60", rec.Header().Get("Retry-After"))
			require.Equal(t, want, serve(mux, apiKeyRequest(path, ""), rateLimitIPB).Code)
			// The per-IP budget is one budget across routes.
			require.Equal(t, http.StatusTooManyRequests, serve(mux, apiKeyRequest(rateLimitTestPath, "key"), rateLimitIPA).Code)
		})
	}
}

func TestRateLimitedResponseInProductionKeepsJSONBody(t *testing.T) {
	app := rateLimitedApp(t, 1, 100)
	app.environment = "production"
	mux := registerRoutes(app)
	require.Equal(t, http.StatusUnauthorized, serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPA).Code)
	rec := serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPA)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.Equal(t, "60", rec.Header().Get("Retry-After"))
	require.JSONEq(t, `{"error":"Too many requests, please try again later"}`, rec.Body.String())
}

// With TRUSTED_PROXY_IPS=none a private peer cannot choose its client address:
// X-Forwarded-For neither satisfies IP_WHITELIST nor opens a new rate-limit
// budget. With the variable empty the same header is honoured.
func TestTrustedProxyNoneIgnoresForwardedHeaders(t *testing.T) {
	const privatePeer = "10.0.0.7:1234"
	forwarded := func(path, key, clientIP string) *http.Request {
		req := apiKeyRequest(path, key)
		req.Header.Set("X-Forwarded-For", clientIP)
		req.Header.Set("X-Real-IP", clientIP)
		return req
	}

	t.Run("IP_WHITELIST", func(t *testing.T) {
		app := rateLimitedApp(t, 100, 100)
		t.Setenv("IP_WHITELIST", "203.0.113.5")
		require.Equal(t, http.StatusOK, serve(registerRoutes(app), forwarded(rateLimitTestPath, "key", "203.0.113.5"), privatePeer).Code)
		t.Setenv("TRUSTED_PROXY_IPS", "None")
		require.Equal(t, http.StatusForbidden, serve(registerRoutes(app), forwarded(rateLimitTestPath, "key", "203.0.113.5"), privatePeer).Code)
	})

	t.Run("rate limit", func(t *testing.T) {
		app := rateLimitedApp(t, 1, 100)
		mux := registerRoutes(app)
		for i := 0; i < 3; i++ {
			require.Equal(t, http.StatusUnauthorized, serve(mux, forwarded(rateLimitTestPath, "wrong", "203.0.113."+strconv.Itoa(i)), privatePeer).Code)
		}
		// Each forged address had its own budget. Under "none" they all count
		// against the peer.
		t.Setenv("TRUSTED_PROXY_IPS", "none")
		mux = registerRoutes(app)
		require.Equal(t, http.StatusUnauthorized, serve(mux, forwarded(rateLimitTestPath, "wrong", "203.0.113.8"), privatePeer).Code)
		require.Equal(t, http.StatusTooManyRequests, serve(mux, forwarded(rateLimitTestPath, "wrong", "203.0.113.9"), privatePeer).Code)
	})
}

func TestRateLimitHitsAreCountedByScope(t *testing.T) {
	t.Setenv("WARDEN_METRICS_REQUIRE_AUTH", "")
	mux := registerRoutes(rateLimitedApp(t, 1, 1))
	// One refusal per scope: the credential limit on address A, the per-IP
	// limit on address B.
	require.Equal(t, http.StatusOK, serve(mux, apiKeyRequest(rateLimitTestPath, "key"), rateLimitIPA).Code)
	require.Equal(t, http.StatusTooManyRequests, serve(mux, apiKeyRequest(rateLimitTestPath, "key"), rateLimitIPA).Code)
	require.Equal(t, http.StatusUnauthorized, serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPB).Code)
	require.Equal(t, http.StatusTooManyRequests, serve(mux, apiKeyRequest(rateLimitTestPath, "wrong"), rateLimitIPB).Code)

	body := serve(mux, apiKeyRequest(define.PATH_METRICS, ""), "198.51.100.32:1234").Body.String()
	require.Regexp(t, `(?m)^warden_rate_limit_hits_total\{scope="credential"\} [1-9]`, body)
	require.Regexp(t, `(?m)^warden_rate_limit_hits_total\{scope="ip"\} [1-9]`, body)
	require.NotRegexp(t, `warden_rate_limit_hits_total\{scope="(?:[^ci]|c[^r]|i[^p])`, body, "only the two fixed scopes are exported")
}
