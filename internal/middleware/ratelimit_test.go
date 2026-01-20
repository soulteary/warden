package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Second)
	defer rl.Stop()

	ip := "127.0.0.1"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		assert.True(t, rl.Allow(ip), "First 3 requests should be allowed")
	}

	// 4th request should be rejected
	assert.False(t, rl.Allow(ip), "Requests exceeding limit should be rejected")
}

func TestRateLimiter_ResetAfterWindow(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)
	defer rl.Stop()

	ip := "127.0.0.1"

	// Use all quota
	assert.True(t, rl.Allow(ip))
	assert.True(t, rl.Allow(ip))
	assert.False(t, rl.Allow(ip), "Should exceed limit")

	// Wait for time window to pass
	time.Sleep(150 * time.Millisecond)

	// Should be able to request again
	assert.True(t, rl.Allow(ip), "Should be able to request again after time window")
}

func TestRateLimitMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rl := NewRateLimiter(2, 1*time.Second)
	defer rl.Stop()
	middleware := RateLimitMiddlewareWithLimiter(rl)
	wrappedHandler := middleware(handler)

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", http.NoBody)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "First 2 requests should succeed")
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "Requests exceeding limit should return 429")
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Second)
	defer rl.Stop()

	// Different IPs should have independent limits
	assert.True(t, rl.Allow("127.0.0.1"))
	assert.True(t, rl.Allow("127.0.0.1"))
	assert.False(t, rl.Allow("127.0.0.1"))

	// Different IPs should be able to continue requesting
	assert.True(t, rl.Allow("192.168.1.1"))
	assert.True(t, rl.Allow("192.168.1.1"))
	assert.False(t, rl.Allow("192.168.1.1"))
}
