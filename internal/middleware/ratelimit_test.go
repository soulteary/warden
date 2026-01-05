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

	// 前3个请求应该被允许
	for i := 0; i < 3; i++ {
		assert.True(t, rl.Allow(ip), "前3个请求应该被允许")
	}

	// 第4个请求应该被拒绝
	assert.False(t, rl.Allow(ip), "超过限制的请求应该被拒绝")
}

func TestRateLimiter_ResetAfterWindow(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)
	defer rl.Stop()

	ip := "127.0.0.1"

	// 使用所有配额
	assert.True(t, rl.Allow(ip))
	assert.True(t, rl.Allow(ip))
	assert.False(t, rl.Allow(ip), "应该超过限制")

	// 等待时间窗口过去
	time.Sleep(150 * time.Millisecond)

	// 应该可以再次请求
	assert.True(t, rl.Allow(ip), "时间窗口后应该可以再次请求")
}

func TestRateLimitMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rl := NewRateLimiter(2, 1*time.Second)
	defer rl.Stop()
	middleware := RateLimitMiddlewareWithLimiter(rl)
	wrappedHandler := middleware(handler)

	// 前2个请求应该成功
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "前2个请求应该成功")
	}

	// 第3个请求应该被限制
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "超过限制的请求应该返回 429")
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Second)
	defer rl.Stop()

	// 不同IP应该有独立的限制
	assert.True(t, rl.Allow("127.0.0.1"))
	assert.True(t, rl.Allow("127.0.0.1"))
	assert.False(t, rl.Allow("127.0.0.1"))

	// 不同IP应该可以继续请求
	assert.True(t, rl.Allow("192.168.1.1"))
	assert.True(t, rl.Allow("192.168.1.1"))
	assert.False(t, rl.Allow("192.168.1.1"))
}
