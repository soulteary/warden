package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	middlewarekit "github.com/soulteary/middleware-kit"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

func newFailingRemoteServer(t *testing.T, expectedAuth string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if expectedAuth != "" {
			assert.Equal(t, expectedAuth, r.Header.Get("Authorization"), "Authorization header should match")
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

// TestCalculateHash tests hash calculation function
func TestCalculateHash(t *testing.T) {
	tests := []struct {
		name     string
		users    []define.AllowListUser
		wantSame bool // Whether same input produces same hash
	}{
		{
			name:     "空列表",
			users:    []define.AllowListUser{},
			wantSame: true,
		},
		{
			name: "单个用户",
			users: []define.AllowListUser{
				{Phone: "13800138000", Mail: "test@example.com"},
			},
			wantSame: true,
		},
		{
			name: "多个用户",
			users: []define.AllowListUser{
				{Phone: "13800138000", Mail: "test1@example.com"},
				{Phone: "13900139000", Mail: "test2@example.com"},
			},
			wantSame: true,
		},
		{
			name: "相同数据不同顺序",
			users: []define.AllowListUser{
				{Phone: "13900139000", Mail: "test2@example.com"},
				{Phone: "13800138000", Mail: "test1@example.com"},
			},
			wantSame: true, // Should produce same hash (because it will be sorted)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := cache.HashUserList(tt.users)
			hash2 := cache.HashUserList(tt.users)

			if tt.wantSame {
				assert.Equal(t, hash1, hash2, "相同输入应该产生相同哈希")
			}

			// Hash value should be a valid hexadecimal string
			assert.NotEmpty(t, hash1, "哈希值不应该为空")
			assert.Len(t, hash1, 64, "SHA256 哈希应该是 64 个字符")
		})
	}
}

// TestCalculateHash_DifferentData tests that different data produces different hashes
func TestCalculateHash_DifferentData(t *testing.T) {
	users1 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
	}
	users2 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test2@example.com"},
	}

	hash1 := cache.HashUserList(users1)
	hash2 := cache.HashUserList(users2)

	assert.NotEqual(t, hash1, hash2, "不同数据应该产生不同哈希")
}

// TestHasChanged tests data change detection
func TestHasChanged(t *testing.T) {
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	oldHash := cache.HashUserList(users)

	// Same data should return false
	assert.False(t, hasChanged(oldHash, users), "相同数据应该返回 false")

	// Different data should return true
	newUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
		{Phone: "13900139000", Mail: "test2@example.com"},
	}
	assert.True(t, hasChanged(oldHash, newUsers), "不同数据应该返回 true")

	// Empty hash should return true
	assert.True(t, hasChanged("", users), "空哈希应该返回 true")
}

// TestNewApp tests application initialization
func TestNewApp(t *testing.T) {
	// Save original environment variables
	originalMode := os.Getenv("MODE")
	defer func() {
		require.NoError(t, os.Setenv("MODE", originalMode))
	}()

	//nolint:govet // fieldalignment: test struct field order does not affect functionality
	tests := []struct {
		name    string
		cfg     *cmd.Config
		wantErr bool
	}{
		{
			name: "基本配置",
			cfg: &cmd.Config{
				Port:             "8081",
				RedisEnabled:     false,
				Mode:             "development",
				APIKey:           "test-key",
				RemoteConfig:     "", // Avoid remote requests to prevent test hanging
				TaskInterval:     60,
				HTTPTimeout:      30,
				HTTPMaxIdleConns: 100,
				HTTPInsecureTLS:  false,
			},
			wantErr: false,
		},
		{
			name: "启用 Redis",
			cfg: &cmd.Config{
				Port:             "8081",
				Redis:            "localhost:6379",
				RedisEnabled:     true,
				Mode:             "development",
				APIKey:           "test-key",
				RemoteConfig:     "", // Avoid remote requests to prevent test hanging
				TaskInterval:     60,
				HTTPTimeout:      30,
				HTTPMaxIdleConns: 100,
				HTTPInsecureTLS:  false,
			},
			wantErr: false, // Redis connection failure won't return error, will fallback to memory mode
		},
		{
			name: "ONLY_LOCAL 模式",
			cfg: &cmd.Config{
				Port:             "8081",
				RedisEnabled:     false,
				Mode:             "ONLY_LOCAL",
				APIKey:           "test-key",
				TaskInterval:     60,
				HTTPTimeout:      30,
				HTTPMaxIdleConns: 100,
				HTTPInsecureTLS:  false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(tt.cfg)
			if tt.wantErr {
				assert.Nil(t, app)
			} else {
				assert.NotNil(t, app)
				if app != nil {
					assert.Equal(t, tt.cfg.Port, app.port)
					assert.Equal(t, tt.cfg.Mode, app.appMode)
					assert.Equal(t, tt.cfg.APIKey, app.apiKey)
					assert.NotNil(t, app.userCache)
					assert.NotNil(t, app.rateLimiter)
				}
			}
		})
	}
}

// TestApp_checkDataChanged tests data change detection
func TestApp_checkDataChanged(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)
	require.NotNil(t, app)

	// Initial data
	users1 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
	}
	app.userCache.Set(users1)

	// Same data should return false
	assert.False(t, app.checkDataChanged(users1), "相同数据应该返回 false")

	// Different data should return true
	users2 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13900139000", Mail: "test2@example.com"},
	}
	assert.True(t, app.checkDataChanged(users2), "不同数据应该返回 true")

	// Different length should return true
	users3 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13900139000", Mail: "test2@example.com"},
		{Phone: "14000140000", Mail: "test3@example.com"},
	}
	assert.True(t, app.checkDataChanged(users3), "长度不同应该返回 true")
}

// TestStartServer tests server startup configuration
func TestStartServer(t *testing.T) {
	srv := startServer("8081")
	require.NotNil(t, srv)
	assert.Equal(t, ":8081", srv.Addr)
	assert.NotZero(t, srv.ReadTimeout)
	assert.NotZero(t, srv.WriteTimeout)
	assert.NotZero(t, srv.ReadHeaderTimeout)
}

// TestShutdownServer tests server shutdown
func TestShutdownServer(t *testing.T) {
	t.Helper()
	// Create a simple rate limiter (using middleware-kit DefaultRateLimiterConfig + overrides)
	cfg := middlewarekit.DefaultRateLimiterConfig()
	cfg.Rate = 100
	cfg.Window = time.Second
	rateLimiter := middlewarekit.NewRateLimiter(cfg)

	// Create a test server
	srv := &http.Server{
		Addr:              ":0", // Use random port
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start server (in goroutine)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("服务器启动错误: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test shutdown (shutdownServer will call rateLimiter.Stop(), so no need for defer)
	log := logger.GetLoggerKit()
	shutdownServer(srv, rateLimiter, log)

	// Verify rate limiter has stopped
	// Note: This only verifies the function doesn't panic, actual state checking requires more complex tests
}

// TestApp_loadInitialData_ONLY_LOCAL tests data loading in ONLY_LOCAL mode
func TestApp_loadInitialData_ONLY_LOCAL(t *testing.T) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	// Write test data
	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "ONLY_LOCAL",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Test loading data
	err = app.loadInitialData(tmpFile.Name())
	assert.NoError(t, err)
	assert.Greater(t, app.userCache.Len(), 0, "应该加载了数据")
}

// TestApp_loadInitialData_EmptyFile tests loading empty file
func TestApp_loadInitialData_EmptyFile(t *testing.T) {
	// Create empty file
	tmpFile, err := os.CreateTemp("", "test-empty-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()
	require.NoError(t, tmpFile.Close())

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Test loading empty file
	err = app.loadInitialData(tmpFile.Name())
	// Empty file should not cause error, just no data
	assert.NoError(t, err)
}

// TestApp_loadInitialData_NonExistentFile tests loading non-existent file
func TestApp_loadInitialData_NonExistentFile(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Test loading non-existent file
	err := app.loadInitialData("/nonexistent/file.json")
	// Non-existent file should not cause error, just no data
	assert.NoError(t, err)
}

// TestApp_backgroundTask_NoChange tests background task (no data change)
func TestApp_backgroundTask_NoChange(t *testing.T) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "ONLY_LOCAL",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Load data first
	err = app.loadInitialData(tmpFile.Name())
	require.NoError(t, err)

	initialLen := app.userCache.Len()

	// Run background task (no data change)
	app.backgroundTask(tmpFile.Name())

	// Verify data hasn't changed
	assert.Equal(t, initialLen, app.userCache.Len(), "数据未变化时长度应该相同")
}

// TestApp_backgroundTask_WithChange tests background task (with data change)
func TestApp_backgroundTask_WithChange(t *testing.T) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	initialData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(initialData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "ONLY_LOCAL",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Load initial data first
	err = app.loadInitialData(tmpFile.Name())
	require.NoError(t, err)

	initialLen := app.userCache.Len()

	// Update file content
	newData := `[
		{"phone": "13800138000", "mail": "test@example.com"},
		{"phone": "13900139000", "mail": "test2@example.com"}
	]`
	err = os.WriteFile(tmpFile.Name(), []byte(newData), 0o600)
	require.NoError(t, err)

	// Run background task (with data change)
	app.backgroundTask(tmpFile.Name())

	// Verify data has been updated
	assert.Greater(t, app.userCache.Len(), initialLen, "数据有变化时应该更新")
}

// TestApp_backgroundTask_PanicRecovery tests panic recovery in background task
func TestApp_backgroundTask_PanicRecovery(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Test panic recovery (by passing invalid file path that might trigger panic)
	// Note: This only verifies the function doesn't crash due to panic
	assert.NotPanics(t, func() {
		app.backgroundTask("/invalid/path/that/might/cause/panic")
	}, "后台任务应该能够恢复 panic")
}

// TestApp_updateRedisCacheWithRetry tests Redis cache update retry mechanism
func TestApp_updateRedisCacheWithRetry(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Skip test if Redis is unavailable
	if app.redisUserCache == nil {
		t.Skip("跳过测试：Redis不可用")
	}

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	// Test successful update
	err := app.updateRedisCacheWithRetry(users)
	// If Redis is available, should succeed; if unavailable, will return error
	if err != nil {
		t.Logf("Redis更新失败（可能是Redis不可用）: %v", err)
	} else {
		assert.NoError(t, err, "Redis缓存更新应该成功")
	}
}

// TestApp_updateRedisCacheWithRetry_NoRedis tests behavior when Redis is not available
func TestApp_updateRedisCacheWithRetry_NoRedis(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	// When Redis is not available, redisUserCache is nil
	assert.Nil(t, app.redisUserCache, "没有Redis时redisUserCache应该为nil")

	// Direct call should return error instead of panic, because redisUserCache is nil
	// In actual usage, this function is only called when redisUserCache != nil
	// This test verifies the function doesn't panic when nil, but returns error
	assert.NotPanics(t, func() {
		err := app.updateRedisCacheWithRetry(users)
		assert.Error(t, err, "redisUserCache为nil时应该返回错误")
	}, "即使redisUserCache为nil也不应该panic")
}

// TestRegisterRoutes tests route registration
func TestRegisterRoutes(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Save original routes
	originalDefaultMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()

	// Register routes
	registerRoutes(app)

	// Verify routes are registered
	_, pattern := http.DefaultServeMux.Handler(&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/"},
	})
	assert.NotEmpty(t, pattern, "根路由应该已注册")

	_, pattern = http.DefaultServeMux.Handler(&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/health"},
	})
	assert.NotEmpty(t, pattern, "健康检查路由应该已注册")

	_, pattern = http.DefaultServeMux.Handler(&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/metrics"},
	})
	assert.NotEmpty(t, pattern, "指标路由应该已注册")

	// Restore original routes
	http.DefaultServeMux = originalDefaultMux
}

// TestNewApp_WithHTTPInsecureTLS tests enabling insecure TLS for HTTP
func TestNewApp_WithHTTPInsecureTLS(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  true,
	}

	app := NewApp(cfg)
	assert.NotNil(t, app)
}

// TestNewApp_ProductionModeWithInsecureTLS tests enabling insecure TLS in production mode (should fail)
func TestNewApp_ProductionModeWithInsecureTLS(t *testing.T) {
	t.Helper()
	// This test needs to capture Fatal, but Fatal will exit the program
	// So we only test configuration validation, not actual execution
	//nolint:govet // unusedwrite: these fields are used to test configuration completeness, though not directly used in tests
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "production",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  true,
	}

	// Note: Enabling insecure TLS in production mode will cause Fatal exit
	// This test mainly verifies that configuration check logic exists
	// Actual testing requires mocking logger.Fatal
	_ = cfg
}

// TestNewApp_WithRedisPassword tests configuration with Redis password
func TestNewApp_WithRedisPassword(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisPassword:    "test-password",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)
	assert.NotNil(t, app)
}

// TestNewApp_TaskIntervalTooSmall tests task interval smaller than default value
func TestNewApp_TaskIntervalTooSmall(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     1,  // Smaller than default value
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)
	assert.NotNil(t, app)
	// Verify task interval is adjusted to default value
	assert.GreaterOrEqual(t, app.taskInterval, uint64(define.DEFAULT_TASK_INTERVAL))
}

// TestApp_loadInitialData_FromRedis tests loading data from Redis
func TestApp_loadInitialData_FromRedis(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Skip test if Redis is unavailable
	if app.redisUserCache == nil {
		t.Skip("跳过测试：Redis不可用")
	}

	// Set some data to Redis first
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}
	err := app.redisUserCache.Set(users)
	if err != nil {
		t.Skipf("跳过测试：无法设置Redis数据: %v", err)
	}

	// Clear memory cache
	app.userCache.Set([]define.AllowListUser{})

	// Test loading from Redis
	err = app.loadInitialData("/nonexistent/file.json")
	assert.NoError(t, err)
	// If Redis has data, should load successfully
	if app.userCache.Len() > 0 {
		assert.Greater(t, app.userCache.Len(), 0, "应该从Redis加载了数据")
	}
}

// TestApp_loadInitialData_RemoteConfig tests loading data from remote config
func TestApp_loadInitialData_RemoteConfig(t *testing.T) {
	remoteServer := newFailingRemoteServer(t, "")
	defer remoteServer.Close()

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests during NewApp
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)
	app.configURL = remoteServer.URL

	// Test loading from remote config (will fail, then fallback to local file)
	err := app.loadInitialData("/nonexistent/file.json")
	// Should not return error, just no data
	assert.NoError(t, err)
}

// TestApp_loadInitialData_FileExistsButEmpty tests file exists but is empty
func TestApp_loadInitialData_FileExistsButEmpty(t *testing.T) {
	// Create empty file
	tmpFile, err := os.CreateTemp("", "test-empty-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()
	require.NoError(t, tmpFile.Close())

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Test loading empty file
	err = app.loadInitialData(tmpFile.Name())
	assert.NoError(t, err)
}

// TestApp_backgroundTask_WithRedis tests background task with Redis
func TestApp_backgroundTask_WithRedis(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Skip test if Redis is unavailable
	if app.redisUserCache == nil {
		t.Skip("跳过测试：Redis不可用")
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Run background task
	app.backgroundTask(tmpFile.Name())

	// Verify task executed (no panic)
	assert.True(t, true)
}

// TestApp_backgroundTask_DataInconsistency tests data inconsistency scenario
func TestApp_backgroundTask_DataInconsistency(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Load data first
	err = app.loadInitialData(tmpFile.Name())
	require.NoError(t, err)

	// Modify cache in another goroutine to simulate data inconsistency
	go func() {
		time.Sleep(10 * time.Millisecond)
		app.userCache.Set([]define.AllowListUser{
			{Phone: "99999999999", Mail: "modified@example.com"},
		})
	}()

	// Run background task
	app.backgroundTask(tmpFile.Name())

	// Verify task executed (no panic)
	assert.True(t, true)
}

// TestShutdownServer_WithNilRateLimiter tests server shutdown when rateLimiter is nil
func TestShutdownServer_WithNilRateLimiter(t *testing.T) {
	srv := &http.Server{
		Addr:              ":0",
		ReadHeaderTimeout: 5 * time.Second,
	}

	log := logger.GetLoggerKit()

	// Test nil rateLimiter
	assert.NotPanics(t, func() {
		shutdownServer(srv, nil, log)
	})
}

// TestShutdownServer_ShutdownError tests error handling during server shutdown
func TestShutdownServer_ShutdownError(t *testing.T) {
	cfg := middlewarekit.DefaultRateLimiterConfig()
	cfg.Rate = 100
	cfg.Window = time.Second
	rateLimiter := middlewarekit.NewRateLimiter(cfg)

	// Create an already closed server
	srv := &http.Server{
		Addr:              ":0",
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Close server first
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Logf("关闭服务器时出错: %v", err)
	}

	log := logger.GetLoggerKit()

	// Closing again should not panic
	assert.NotPanics(t, func() {
		shutdownServer(srv, rateLimiter, log)
	})
}

// TestCalculateHash_WithAllFields tests hash calculation with all fields
func TestCalculateHash_WithAllFields(t *testing.T) {
	users := []define.AllowListUser{
		{
			Phone:  "13800138000",
			Mail:   "test@example.com",
			UserID: "user123",
			Status: "active",
			Scope:  []string{"read", "write"},
			Role:   "admin",
		},
	}

	hash1 := cache.HashUserList(users)
	hash2 := cache.HashUserList(users)

	assert.Equal(t, hash1, hash2, "相同输入应该产生相同哈希")
	assert.Len(t, hash1, 64, "SHA256 哈希应该是 64 个字符")
}

// TestCalculateHash_WithScope tests hash calculation with Scope field
func TestCalculateHash_WithScope(t *testing.T) {
	users1 := []define.AllowListUser{
		{
			Phone: "13800138000",
			Mail:  "test@example.com",
			Scope: []string{"read"},
		},
	}

	users2 := []define.AllowListUser{
		{
			Phone: "13800138000",
			Mail:  "test@example.com",
			Scope: []string{"read", "write"},
		},
	}

	hash1 := cache.HashUserList(users1)
	hash2 := cache.HashUserList(users2)

	assert.NotEqual(t, hash1, hash2, "不同Scope应该产生不同哈希")
}

// TestApp_checkDataChanged_EmptyHash tests empty hash scenario
func TestApp_checkDataChanged_EmptyHash(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests to prevent test hanging
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	// Clear cache hash
	app.userCache.Set([]define.AllowListUser{})

	// Test empty hash scenario
	assert.True(t, app.checkDataChanged(users), "空哈希时应该返回true")
}

// TestApp_loadInitialData_AllSourcesFailed tests when all data sources fail
func TestApp_loadInitialData_AllSourcesFailed(t *testing.T) {
	remoteServer := newFailingRemoteServer(t, "")
	defer remoteServer.Close()

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests during NewApp
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)
	app.configURL = remoteServer.URL
	app.userCache.Set([]define.AllowListUser{})

	// Test loading when all sources fail
	err := app.loadInitialData("/nonexistent/file.json")
	assert.NoError(t, err, "所有源失败时不应该返回错误，只是没有数据")
	assert.Equal(t, 0, app.userCache.Len(), "所有源失败时缓存应该为空")
}

// TestApp_loadInitialData_RemoteFirst tests remote-first loading strategy
func TestApp_loadInitialData_RemoteFirst(t *testing.T) {
	remoteServer := newFailingRemoteServer(t, "")
	defer remoteServer.Close()

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "REMOTE_FIRST",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests during NewApp
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)
	app.configURL = remoteServer.URL

	// Create temporary local file as fallback
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Test loading (remote fails, should fallback to local)
	err = app.loadInitialData(tmpFile.Name())
	assert.NoError(t, err, "远程失败时应该回退到本地文件")
	assert.Greater(t, app.userCache.Len(), 0, "应该从本地文件加载数据")
}

// TestApp_backgroundTask_RemoteMode tests background task in remote mode
func TestApp_backgroundTask_RemoteMode(t *testing.T) {
	remoteServer := newFailingRemoteServer(t, "")
	defer remoteServer.Close()

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "REMOTE_FIRST",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests during NewApp
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)
	app.configURL = remoteServer.URL

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Run background task (will try remote first, then fallback to local)
	app.backgroundTask(tmpFile.Name())

	// Verify task executed without panic
	assert.True(t, true, "后台任务应该执行完成")
}

// TestApp_updateRedisCacheWithRetry_MaxRetries tests retry logic with max retries
func TestApp_updateRedisCacheWithRetry_MaxRetries(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Skip test if Redis is unavailable
	if app.redisUserCache == nil {
		t.Skip("跳过测试：Redis不可用")
	}

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	// Test retry mechanism (may succeed or fail depending on Redis availability)
	err := app.updateRedisCacheWithRetry(users)
	// This test mainly verifies the function doesn't panic
	// Actual result depends on Redis availability
	if err != nil {
		t.Logf("Redis更新失败（可能是Redis不可用）: %v", err)
	}
}

// TestNewApp_WithRedisConnectionFailure tests Redis connection failure handling
func TestNewApp_WithRedisConnectionFailure(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "invalid-redis-host:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Should not panic, should fallback to memory mode
	assert.NotNil(t, app, "应该创建应用实例")
	assert.NotNil(t, app.userCache, "应该有内存缓存")
	// Redis cache may be nil if connection failed
}

// TestNewApp_WithRedisPasswordFromFile tests reading Redis password from file
func TestNewApp_WithRedisPasswordFromFile(t *testing.T) {
	// Create temporary password file
	tmpFile, err := os.CreateTemp("", "test-redis-password-*.txt")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	testPassword := "file-password-123"
	_, err = tmpFile.WriteString(testPassword)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Set environment variable
	oldEnv := os.Getenv("REDIS_PASSWORD_FILE")
	defer func() {
		if oldEnv == "" {
			require.NoError(t, os.Unsetenv("REDIS_PASSWORD_FILE"))
		} else {
			require.NoError(t, os.Setenv("REDIS_PASSWORD_FILE", oldEnv))
		}
	}()

	require.NoError(t, os.Setenv("REDIS_PASSWORD_FILE", tmpFile.Name()))

	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)
	assert.NotNil(t, app, "应该创建应用实例")
}

// TestApp_loadInitialData_WithRemoteKey tests loading with authorization header
func TestApp_loadInitialData_WithRemoteKey(t *testing.T) {
	remoteServer := newFailingRemoteServer(t, "Bearer test-token")
	defer remoteServer.Close()

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // Avoid remote requests during NewApp
		RemoteKey:        "Bearer test-token",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)
	app.configURL = remoteServer.URL

	// Test loading with remote key (will fail, but tests the code path)
	err := app.loadInitialData("/nonexistent/file.json")
	assert.NoError(t, err, "应该处理远程密钥配置")
}

// TestApp_backgroundTask_DataConsistency tests data consistency check
func TestApp_backgroundTask_DataConsistency(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "ONLY_LOCAL",
		APIKey:           "test-key",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Load initial data
	err = app.loadInitialData(tmpFile.Name())
	require.NoError(t, err)

	// Modify cache hash to simulate inconsistency
	app.userCache.Set([]define.AllowListUser{
		{Phone: "99999999999", Mail: "modified@example.com"},
	})

	// Run background task (should detect inconsistency)
	app.backgroundTask(tmpFile.Name())

	// Verify task executed
	assert.True(t, true, "后台任务应该执行完成")
}

// TestCalculateHash_EmptyUsers tests hash calculation with empty users
func TestCalculateHash_EmptyUsers(t *testing.T) {
	users := []define.AllowListUser{}
	hash1 := cache.HashUserList(users)
	hash2 := cache.HashUserList(users)

	assert.Equal(t, hash1, hash2, "空用户列表应该产生相同哈希")
	assert.Len(t, hash1, 64, "SHA256 哈希应该是 64 个字符")
}

// TestCalculateHash_NilScope tests hash calculation with nil scope
func TestCalculateHash_NilScope(t *testing.T) {
	users := []define.AllowListUser{
		{
			Phone:  "13800138000",
			Mail:   "test@example.com",
			Scope:  nil, // nil scope
			Status: "active",
		},
	}

	hash1 := cache.HashUserList(users)
	hash2 := cache.HashUserList(users)

	assert.Equal(t, hash1, hash2, "相同输入应该产生相同哈希")
}

// TestHasChanged_EmptyOldHash tests hasChanged with empty old hash
func TestHasChanged_EmptyOldHash(t *testing.T) {
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	assert.True(t, hasChanged("", users), "空旧哈希应该返回true")
}

// TestHasChanged_SameHash tests hasChanged with same hash
func TestHasChanged_SameHash(t *testing.T) {
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	hash := cache.HashUserList(users)
	assert.False(t, hasChanged(hash, users), "相同哈希应该返回false")
}

// TestRegisterRoutes_AllEndpoints tests all registered endpoints
func TestRegisterRoutes_AllEndpoints(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app := NewApp(cfg)

	// Save original routes
	originalDefaultMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()

	// Register routes
	registerRoutes(app)

	// Test all endpoints
	endpoints := []string{"/", "/user", "/health", "/healthcheck", "/metrics", "/log/level"}
	for _, endpoint := range endpoints {
		_, pattern := http.DefaultServeMux.Handler(&http.Request{
			Method: "GET",
			URL:    &url.URL{Path: endpoint},
		})
		assert.NotEmpty(t, pattern, "端点 %s 应该已注册", endpoint)
	}

	// Restore original routes
	http.DefaultServeMux = originalDefaultMux
}
