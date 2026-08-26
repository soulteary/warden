package di

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
)

func TestNewDependencies_Success(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	require.NoError(t, err)
	require.NotNil(t, deps)

	assert.Equal(t, cfg, deps.Config)
	assert.NotNil(t, deps.UserCache)
	assert.NotNil(t, deps.RateLimiter)
	assert.NotNil(t, deps.HTTPServer)
	assert.NotNil(t, deps.MainHandler)
	assert.NotNil(t, deps.HealthHandler)
	assert.NotNil(t, deps.LogLevelHandler)
}

func TestNewDependencies_WithRedis(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	// Test Redis connection failure scenario (should return error)
	deps, err := NewDependencies(cfg)
	// If Redis is unavailable, should return error
	if err != nil {
		assert.Error(t, err)
		assert.Nil(t, deps)
	} else {
		// Redis available case
		assert.NotNil(t, deps)
		if deps != nil {
			assert.NotNil(t, deps.RedisClient)
			assert.NotNil(t, deps.RedisUserCache)
		}
	}
}

func TestNewDependencies_WithRedisPassword(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisPassword:    "test-password",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	// Test Redis connection failure scenario (should return error)
	deps, err := NewDependencies(cfg)
	// If Redis is unavailable, should return error
	if err != nil {
		assert.Error(t, err)
		assert.Nil(t, deps)
	} else {
		// Redis available case
		assert.NotNil(t, deps)
		if deps != nil {
			assert.NotNil(t, deps.RedisClient)
		}
	}
}

func TestDependencies_Cleanup(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	require.NoError(t, err)
	require.NotNil(t, deps)

	// Test Cleanup doesn't panic
	assert.NotPanics(t, func() {
		deps.Cleanup()
	})

	// Can call Cleanup multiple times
	assert.NotPanics(t, func() {
		deps.Cleanup()
	})
}

func TestDependencies_Cleanup_WithRedis(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	// Skip test if Redis is unavailable
	if err != nil {
		t.Skipf("跳过测试：Redis不可用: %v", err)
	}

	require.NotNil(t, deps)

	// Test Cleanup doesn't panic
	assert.NotPanics(t, func() {
		deps.Cleanup()
	})
}

func TestDependencies_HTTPServer_Configuration(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "9090",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	require.NoError(t, err)
	require.NotNil(t, deps)

	// Verify HTTP server configuration
	assert.Equal(t, ":9090", deps.HTTPServer.Addr)
	assert.Equal(t, define.DEFAULT_TIMEOUT*time.Second, deps.HTTPServer.ReadHeaderTimeout)
	assert.Equal(t, define.DEFAULT_TIMEOUT*time.Second, deps.HTTPServer.ReadTimeout)
	assert.Equal(t, define.DEFAULT_TIMEOUT*time.Second, deps.HTTPServer.WriteTimeout)
	assert.Equal(t, define.IDLE_TIMEOUT, deps.HTTPServer.IdleTimeout)
	assert.Equal(t, define.MAX_HEADER_BYTES, deps.HTTPServer.MaxHeaderBytes)
}

func TestDependencies_Handlers_NotNil(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	require.NoError(t, err)
	require.NotNil(t, deps)

	// Verify all handlers are not nil
	assert.NotNil(t, deps.MainHandler)
	assert.NotNil(t, deps.HealthHandler)
	assert.NotNil(t, deps.LogLevelHandler)

	// Verify handlers can process requests (won't panic)
	req, err := http.NewRequest("GET", "/", http.NoBody)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	assert.NotPanics(t, func() {
		deps.MainHandler.ServeHTTP(rr, req)
	})
}

func TestDependencies_InitRedis_ConnectionTimeout(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "invalid-host:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	// Test invalid Redis address should return error
	deps, err := NewDependencies(cfg)
	assert.Error(t, err)
	assert.Nil(t, deps)
}

func TestDependencies_InitCache_WithoutRedis(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	require.NoError(t, err)
	require.NotNil(t, deps)

	// Note: NewDependencies doesn't initialize RedisClient when RedisEnabled=false
	// But initCache will create RedisUserCache (even if RedisClient is nil)
	// So RedisUserCache may not be nil, but its client field is nil
	assert.NotNil(t, deps.UserCache)
	// RedisUserCache may be created, but its client is nil
	if deps.RedisUserCache != nil {
		// If created, verify it exists (this is normal, as NewRedisUserCache accepts nil client)
		assert.NotNil(t, deps.RedisUserCache)
	}
}

func TestDependencies_InitRateLimiter(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	require.NoError(t, err)
	require.NotNil(t, deps)

	// Verify rate limiter is initialized
	assert.NotNil(t, deps.RateLimiter)
}

func TestDependencies_Cleanup_NilFields(t *testing.T) {
	// Test Cleanup handling nil fields
	deps := &Dependencies{
		RateLimiter: nil,
		RedisClient: nil,
	}

	// Should not panic
	assert.NotPanics(t, func() {
		deps.Cleanup()
	})
}

func TestDependencies_Cleanup_RedisCloseError(t *testing.T) {
	// Create an already closed Redis client to test error handling
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	// Skip test if Redis is unavailable
	if err != nil {
		t.Skipf("跳过测试：Redis不可用: %v", err)
	}

	require.NotNil(t, deps)

	// Close Redis client first
	if deps.RedisClient != nil {
		if err := deps.RedisClient.Close(); err != nil {
			t.Logf("关闭Redis客户端时出错: %v", err)
		}
	}

	// Calling Cleanup again should not panic
	assert.NotPanics(t, func() {
		deps.Cleanup()
	})
}

func TestDependencies_InitHandlers_WithRedis(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	// Skip test if Redis is unavailable
	if err != nil {
		t.Skipf("跳过测试：Redis不可用: %v", err)
	}

	require.NotNil(t, deps)

	// Verify handlers are initialized
	assert.NotNil(t, deps.MainHandler)
	assert.NotNil(t, deps.HealthHandler)
	assert.NotNil(t, deps.LogLevelHandler)
}

func TestDependencies_InitHTTPServer_CustomPort(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "12345",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	require.NoError(t, err)
	require.NotNil(t, deps)

	// Verify port configuration is correct
	assert.Equal(t, ":12345", deps.HTTPServer.Addr)
}

// Test error handling of initRedis method
func TestDependencies_InitRedis_InvalidAddress(t *testing.T) {
	d := &Dependencies{
		Config: &cmd.Config{
			Redis:        "invalid-address:99999",
			RedisEnabled: true,
		},
	}

	err := d.initRedis()
	assert.Error(t, err)
}

// Test initCache method
func TestDependencies_InitCache(t *testing.T) {
	d := &Dependencies{
		Config:      &cmd.Config{},
		RedisClient: nil, // No Redis client
	}

	d.initCache()
	assert.NotNil(t, d.UserCache)
	// Note: initCache will try to create RedisUserCache, even if RedisClient is nil
	// But RedisUserCache creation requires a valid RedisClient
	// According to implementation, if RedisClient is nil, RedisUserCache may be nil or creation may fail
}

// Test initCache method (with Redis)
func TestDependencies_InitCache_WithRedis(t *testing.T) {
	// Create a mock Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("跳过测试：Redis不可用: %v", err)
	}

	d := &Dependencies{
		Config:      &cmd.Config{},
		RedisClient: client,
	}

	d.initCache()
	assert.NotNil(t, d.UserCache)
	assert.NotNil(t, d.RedisUserCache)
}

// Test initRateLimiter method (direct call)
func TestDependencies_InitRateLimiter_Direct(t *testing.T) {
	d := &Dependencies{
		Config: &cmd.Config{},
	}

	d.initRateLimiter()
	assert.NotNil(t, d.RateLimiter)
}

// Test initHandlers method
func TestDependencies_InitHandlers(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	deps, err := NewDependencies(cfg)
	require.NoError(t, err)

	// Re-initialize handlers to test initHandlers method
	deps.initHandlers()
	assert.NotNil(t, deps.MainHandler)
	assert.NotNil(t, deps.HealthHandler)
	assert.NotNil(t, deps.LogLevelHandler)
}

// Test initHTTPServer method
func TestDependencies_InitHTTPServer(t *testing.T) {
	d := &Dependencies{
		Config: &cmd.Config{
			Port: "9999",
		},
	}

	d.initHTTPServer()
	assert.NotNil(t, d.HTTPServer)
	assert.Equal(t, ":9999", d.HTTPServer.Addr)
}
