package middleware

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeClock is a manually advanced clock for a RateLimiter.
type fakeClock struct {
	now time.Time
	mu  sync.Mutex
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// newTestLimiter returns a limiter with a one-minute window and a manual clock.
func newTestLimiter(t *testing.T, rate, maxKeys int) (*RateLimiter, *fakeClock) {
	t.Helper()
	l := NewRateLimiter(rate, time.Minute, maxKeys)
	t.Cleanup(l.Stop)
	clock := &fakeClock{now: l.epoch}
	l.now = clock.Now
	return l, clock
}

func TestRateLimiterSlidingWindow(t *testing.T) {
	l, clock := newTestLimiter(t, 2, 10)

	ok, _ := l.Allow("a")
	require.True(t, ok)
	clock.Advance(20 * time.Second)
	ok, _ = l.Allow("a")
	require.True(t, ok)

	// Full: the refusal reports when the oldest request leaves the window and
	// is not itself recorded.
	clock.Advance(10 * time.Second)
	ok, retryAfter := l.Allow("a")
	require.False(t, ok)
	require.Equal(t, 30*time.Second, retryAfter)
	ok, retryAfter = l.Allow("a")
	require.False(t, ok)
	require.Equal(t, 30*time.Second, retryAfter)

	// Keys are independent.
	ok, _ = l.Allow("b")
	require.True(t, ok)

	// One request fits once the first has left the window; the second is
	// still inside it, so no interval of one window holds more than two.
	clock.Advance(30 * time.Second)
	ok, _ = l.Allow("a")
	require.True(t, ok)
	ok, retryAfter = l.Allow("a")
	require.False(t, ok)
	require.Equal(t, 20*time.Second, retryAfter)
}

func TestRateLimiterRelease(t *testing.T) {
	l, _ := newTestLimiter(t, 2, 10)

	// Released requests do not count, however many there are.
	for i := 0; i < 10; i++ {
		ok, _ := l.Allow("a")
		require.True(t, ok)
		l.Release("a")
	}
	require.Empty(t, l.entries, "a key with nothing recorded is not kept")

	ok, _ := l.Allow("a")
	require.True(t, ok)
	ok, _ = l.Allow("a")
	require.True(t, ok)
	l.Release("a")
	ok, _ = l.Allow("a")
	require.True(t, ok)
	ok, _ = l.Allow("a")
	require.False(t, ok)

	l.Release("unknown")
}

func TestRateLimiterBoundsTrackedKeys(t *testing.T) {
	const maxKeys = 100
	l, clock := newTestLimiter(t, 3, maxKeys)

	for i := 0; i < 50*maxKeys; i++ {
		clock.Advance(time.Millisecond)
		ok, _ := l.Allow("198.51.100." + strconv.Itoa(i))
		require.True(t, ok)
		require.LessOrEqual(t, len(l.entries), maxKeys)
	}
	// The most recently seen key survived eviction and kept its count.
	last := "198.51.100." + strconv.Itoa(50*maxKeys-1)
	require.Contains(t, l.entries, last)
	require.Len(t, l.entries[last].stamps, 1)

	// Each key holds at most rate timestamps.
	for i := 0; i < 20; i++ {
		l.Allow(last)
	}
	require.Len(t, l.entries[last].stamps, 3)
	require.LessOrEqual(t, cap(l.entries[last].stamps), 8)
}

func TestRateLimiterEvictsExpiredKeysFirst(t *testing.T) {
	l, clock := newTestLimiter(t, 1, 4)
	for _, key := range []string{"a", "b", "c"} {
		l.Allow(key)
	}
	clock.Advance(2 * time.Minute)
	l.Allow("d")
	// The table is full; "a", "b" and "c" have nothing left in the window and
	// are dropped, the live key "d" is kept.
	l.Allow("e")
	require.Len(t, l.entries, 2)
	require.Contains(t, l.entries, "d")
	require.Contains(t, l.entries, "e")
}

func TestRateLimiterSweepAndStop(t *testing.T) {
	l, clock := newTestLimiter(t, 1, 10)
	l.Allow("a")
	clock.Advance(30 * time.Second)
	l.Allow("b")
	clock.Advance(31 * time.Second)

	l.mu.Lock()
	l.sweep(clock.Now().Sub(l.epoch))
	l.mu.Unlock()
	require.NotContains(t, l.entries, "a")
	require.Contains(t, l.entries, "b")

	l.Stop()
	l.Stop()
	select {
	case <-l.done:
	default:
		t.Fatal("cleanup goroutine still running after Stop")
	}
	var limits *RateLimits
	limits.Stop()
}

func TestRateLimiterDefaults(t *testing.T) {
	limits := NewRateLimits(0, 0, 0)
	t.Cleanup(limits.Stop)
	require.Equal(t, 60, limits.IP.rate)
	require.Equal(t, 600, limits.Credential.rate)
	require.Equal(t, time.Minute, limits.IP.window)
	require.Equal(t, time.Minute, limits.Credential.window)
	require.Equal(t, 10000, limits.IP.maxKeys)
}

func TestRateLimiterConcurrentUse(t *testing.T) {
	l, _ := newTestLimiter(t, 50, 8)
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if ok, _ := l.Allow("shared"); ok {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
				l.Allow("other-" + strconv.Itoa(g*100+i))
				l.Release("other-" + strconv.Itoa(g*100+i))
			}
		}(g)
	}
	wg.Wait()
	require.Equal(t, 50, allowed)
}

// rateLimitTestChain builds limiter(auth(handler)) with an API key "key" and
// an HMAC key "svc". authCalls counts requests that reached authentication.
func rateLimitTestChain(t *testing.T, ipRate, credentialRate int, hits map[string]int) (handler http.Handler, authCalls *int) {
	t.Helper()
	limits := NewRateLimits(ipRate, credentialRate, time.Minute)
	t.Cleanup(limits.Stop)
	cfg := RateLimitConfig{Limits: limits, OnLimited: func(scope string) { hits[scope]++ }}
	calls := 0
	apiKey := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-API-Key") != "key" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	chain := ServiceAuthChain(HMACConfig{}, apiKey)
	auth := func(next http.Handler) http.Handler {
		inner := chain(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			inner.ServeHTTP(w, r)
		})
	}
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	return RateLimitAuthenticated(cfg, auth)(ok), &calls
}

func rateLimitTestRequest(remoteAddr, apiKey string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = remoteAddr
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	return req
}

func TestRateLimitAuthenticatedRefusesExhaustedIPBeforeAuth(t *testing.T) {
	hits := map[string]int{}
	handler, authCalls := rateLimitTestChain(t, 2, 5, hits)
	do := func(remoteAddr, apiKey string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, rateLimitTestRequest(remoteAddr, apiKey))
		return rec
	}

	require.Equal(t, http.StatusUnauthorized, do("198.51.100.1:1", "bad").Code)
	require.Equal(t, http.StatusUnauthorized, do("198.51.100.1:1", "").Code)
	require.Equal(t, 2, *authCalls)

	// Refused without evaluating the credential, valid or not.
	for _, key := range []string{"bad", "key"} {
		rec := do("198.51.100.1:1", key)
		require.Equal(t, http.StatusTooManyRequests, rec.Code)
		require.Equal(t, "Too Many Requests\n", rec.Body.String())
		require.Equal(t, "60", rec.Header().Get("Retry-After"))
	}
	require.Equal(t, 2, *authCalls, "authentication must not run for an exhausted address")
	require.Equal(t, map[string]int{RateLimitScopeIP: 2}, hits)

	// The credential works from another address and is counted on its own.
	for i := 0; i < 5; i++ {
		require.Equal(t, http.StatusOK, do("198.51.100.2:1", "key").Code)
	}
	rec := do("198.51.100.2:1", "key")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "60", rec.Header().Get("Retry-After"))
	require.Equal(t, map[string]int{RateLimitScopeIP: 2, RateLimitScopeCredential: 1}, hits)
	// Neither the successes nor the credential refusal spent that address's budget.
	require.Equal(t, http.StatusUnauthorized, do("198.51.100.2:1", "bad").Code)
}

// Concurrent failures cannot exceed the per-address rate: the unit is taken
// before authentication, not after it fails.
func TestRateLimitAuthenticatedBoundsConcurrentFailures(t *testing.T) {
	limits := NewRateLimits(5, 5, time.Minute)
	t.Cleanup(limits.Stop)
	var mu sync.Mutex
	evaluated := 0
	release := make(chan struct{})
	auth := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			evaluated++
			mu.Unlock()
			<-release
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
	}
	handler := RateLimitAuthenticated(RateLimitConfig{Limits: limits}, auth)(http.NotFoundHandler())

	var wg sync.WaitGroup
	codes := make(chan int, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, rateLimitTestRequest("198.51.100.3:1", "bad"))
			codes <- rec.Code
		}()
	}
	// 35 requests are refused while the 5 admitted ones are still in flight.
	refused := 0
	for refused < 35 {
		select {
		case code := <-codes:
			require.Equal(t, http.StatusTooManyRequests, code)
			refused++
		case <-time.After(5 * time.Second):
			close(release)
			wg.Wait()
			t.Fatalf("%d of 40 concurrent requests were refused before authentication, want 35", refused)
		}
	}
	close(release)
	wg.Wait()
	require.Equal(t, 5, evaluated)
}

func TestRateLimitByIPCountsEveryRequest(t *testing.T) {
	limits := NewRateLimits(2, 100, time.Minute)
	t.Cleanup(limits.Stop)
	handler := RateLimitByIP(RateLimitConfig{Limits: limits})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	codes := make([]int, 0, 4)
	for _, remoteAddr := range []string{"198.51.100.4:1", "198.51.100.4:2", "198.51.100.4:3", "198.51.100.5:1"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, rateLimitTestRequest(remoteAddr, "key"))
		codes = append(codes, rec.Code)
	}
	require.Equal(t, []int{200, 200, 429, 200}, codes)
}

func TestServiceAuthChainSetsPrincipal(t *testing.T) {
	passThrough := func(next http.Handler) http.Handler { return next }
	principal := func(cfg HMACConfig, req *http.Request) string {
		got := "not reached"
		handler := ServiceAuthChain(cfg, passThrough)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			got = PrincipalFromContext(r.Context())
		}))
		handler.ServeHTTP(httptest.NewRecorder(), req)
		return got
	}

	require.Equal(t, "api-key", principal(HMACConfig{}, rateLimitTestRequest("198.51.100.6:1", "any")))

	mtls := func(der string) *http.Request {
		req := rateLimitTestRequest("198.51.100.6:1", "")
		req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{Raw: []byte(der)}}}
		return req
	}
	certA, certB := principal(HMACConfig{}, mtls("a")), principal(HMACConfig{}, mtls("b"))
	require.Regexp(t, `^mtls:[0-9a-f]{64}$`, certA)
	require.NotEqual(t, certA, certB)
	require.Equal(t, certA, principal(HMACConfig{}, mtls("a")))

	require.Empty(t, PrincipalFromContext(rateLimitTestRequest("198.51.100.6:1", "").Context()))
}
