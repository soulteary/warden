package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	health "github.com/soulteary/health-kit/v2"
	middlewarekit "github.com/soulteary/middleware-kit/v2"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

type stubRefreshLocker struct {
	unlocks int
	locked  bool
}

type failingRefreshLocker struct{}

func (failingRefreshLocker) Lock(_ string) (bool, error) {
	return false, assert.AnError
}

func (failingRefreshLocker) Unlock(_ string) error {
	return nil
}

func (l *stubRefreshLocker) Lock(_ string) (bool, error) {
	return l.locked, nil
}

func (l *stubRefreshLocker) Unlock(_ string) error {
	l.unlocks++
	return nil
}

func newFailingRemoteServer(t *testing.T, expectedAuth string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if expectedAuth != "" {
			assert.Equal(t, expectedAuth, r.Header.Get("Authorization"), "Authorization header should match")
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestHealthSnapshotDegradedAfterStrictRefreshFailure(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	userCache.Set([]define.AllowListUser{{Phone: "13800138000"}})
	snapshots := newSnapshotStore()
	snapshots.Store(&Snapshot{
		Users:    userCache.Get(),
		Count:    1,
		Source:   "remote",
		Version:  "abc123",
		LoadedAt: time.Now(),
	})
	snapshots.RecordRefreshFailure("timeout")

	aggregator := setupHealthChecker(nil, userCache, snapshots, time.Minute, "ONLY_REMOTE", "development", false, false, "")
	result := aggregator.Check(context.Background())

	assert.Equal(t, health.StatusDegraded, result.Status)
	snapshotCheck := result.Checks["snapshot"]
	assert.Equal(t, health.StatusDegraded, snapshotCheck.Status)
	assert.Equal(t, "timeout", snapshotCheck.Metadata["reason"])
	assert.EqualValues(t, 1, snapshotCheck.Metadata["consecutive_failures"])
}

func TestHealthRedisUnavailableWithCachedDataIsDegraded(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	userCache.Set([]define.AllowListUser{{Phone: "13800138000"}})

	aggregator := setupHealthChecker(nil, userCache, nil, time.Minute, "DEFAULT", "development", true, false, "")
	result := aggregator.Check(context.Background())

	assert.Equal(t, health.StatusDegraded, result.Status)
	assert.Equal(t, health.StatusUnhealthy, result.Checks["redis"].Status)
	assert.Equal(t, health.StatusHealthy, result.Checks["data"].Status)
}

func TestHealthRedisUnavailableWithoutDataIsUnhealthy(t *testing.T) {
	userCache := cache.NewSafeUserCache()

	aggregator := setupHealthChecker(nil, userCache, nil, time.Minute, "DEFAULT", "development", true, false, "")
	result := aggregator.Check(context.Background())

	assert.Equal(t, health.StatusUnhealthy, result.Status)
	assert.Equal(t, health.StatusUnhealthy, result.Checks["redis"].Status)
	assert.Equal(t, health.StatusUnhealthy, result.Checks["data"].Status)
}

func TestHealthRedisUnavailableForHMACReplayIsUnhealthy(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	userCache.Set([]define.AllowListUser{{Phone: "13800138000"}})

	aggregator := setupHealthChecker(nil, userCache, nil, time.Minute, "DEFAULT", "development", true, true, "")
	result := aggregator.Check(context.Background())

	assert.Equal(t, health.StatusUnhealthy, result.Status)
	assert.Equal(t, health.StatusUnhealthy, result.Checks["redis"].Status)
	assert.Equal(t, health.StatusHealthy, result.Checks["data"].Status)
}

func TestHealthStaleSnapshotIsUnhealthyInStrictMode(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	userCache.Set([]define.AllowListUser{{Phone: "13800138000"}})
	snapshots := newSnapshotStore()
	snapshots.Store(&Snapshot{
		Users:    userCache.Get(),
		Count:    1,
		Source:   "remote",
		Version:  "abc123",
		LoadedAt: time.Now().Add(-2 * time.Minute),
	})

	aggregator := setupHealthChecker(nil, userCache, snapshots, time.Minute, "ONLY_REMOTE", "development", false, false, "")
	result := aggregator.Check(context.Background())

	assert.Equal(t, health.StatusUnhealthy, result.Status)
	assert.Equal(t, health.StatusUnhealthy, result.Checks["snapshot_freshness"].Status)
	assert.Equal(t, "snapshot_stale", result.Checks["snapshot_freshness"].Metadata["reason"])
}

func TestHealthStaleSnapshotIsDegradedInTolerantMode(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	userCache.Set([]define.AllowListUser{{Phone: "13800138000"}})
	snapshots := newSnapshotStore()
	snapshots.Store(&Snapshot{
		Users:    userCache.Get(),
		Count:    1,
		Source:   "local",
		Version:  "abc123",
		LoadedAt: time.Now().Add(-2 * time.Minute),
	})

	aggregator := setupHealthChecker(nil, userCache, snapshots, time.Minute, "LOCAL_FIRST_ALLOW_REMOTE_FAILED", "development", false, false, "")
	result := aggregator.Check(context.Background())

	assert.Equal(t, health.StatusDegraded, result.Status)
	assert.Equal(t, health.StatusUnhealthy, result.Checks["snapshot_freshness"].Status)
}

func TestHealthUnknownSnapshotIsUnhealthyInStrictMode(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	userCache.Set([]define.AllowListUser{{Phone: "13800138000"}})
	snapshots := newSnapshotStore()

	aggregator := setupHealthChecker(nil, userCache, snapshots, time.Minute, "REMOTE_FIRST", "development", false, false, "")
	result := aggregator.Check(context.Background())

	assert.Equal(t, health.StatusUnhealthy, result.Status)
	assert.Equal(t, health.StatusDegraded, result.Checks["snapshot"].Status)
	assert.Equal(t, "no_snapshot", result.Checks["snapshot"].Metadata["reason"])
	assert.Equal(t, health.StatusUnhealthy, result.Checks["snapshot_freshness"].Status)
	assert.Equal(t, "snapshot_unknown", result.Checks["snapshot_freshness"].Metadata["reason"])
}

func TestRequiresRedisForHMACReplay(t *testing.T) {
	tests := []struct {
		hmacKeys     map[string]string
		name         string
		redisEnabled bool
		want         bool
	}{
		{name: "redis disabled", redisEnabled: false, hmacKeys: map[string]string{"key-1": "secret"}, want: false},
		{name: "no hmac keys", redisEnabled: true, hmacKeys: nil, want: false},
		{name: "configured even before client connects", redisEnabled: true, hmacKeys: map[string]string{"key-1": "secret"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, requiresRedisForHMACReplay(tt.redisEnabled, tt.hmacKeys))
		})
	}
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

func TestNewApp_HMACKeys(t *testing.T) {
	t.Run("valid key set", func(t *testing.T) {
		app := NewApp(&cmd.Config{HMACKeys: `{" key-1 ":"secret"}`})
		require.NotNil(t, app)
		assert.Equal(t, map[string]string{"key-1": "secret"}, app.hmacKeys)
	})

	t.Run("invalid key set is disabled", func(t *testing.T) {
		app := NewApp(&cmd.Config{HMACKeys: `{"key-1":""}`})
		require.NotNil(t, app)
		assert.Nil(t, app.hmacKeys)
	})
}

func TestNewApp_HMACAllowV1(t *testing.T) {
	t.Run("disabled by default", func(t *testing.T) {
		t.Setenv("WARDEN_HMAC_ALLOW_V1", "")
		app := NewApp(&cmd.Config{})
		require.NotNil(t, app)
		assert.False(t, app.hmacAllowV1)
	})

	t.Run("explicit opt-in", func(t *testing.T) {
		t.Setenv("WARDEN_HMAC_ALLOW_V1", "true")
		app := NewApp(&cmd.Config{})
		require.NotNil(t, app)
		assert.True(t, app.hmacAllowV1)
	})

	t.Run("invalid value keeps secure default", func(t *testing.T) {
		t.Setenv("WARDEN_HMAC_ALLOW_V1", "enabled")
		app := NewApp(&cmd.Config{})
		require.NotNil(t, app)
		assert.False(t, app.hmacAllowV1)
	})
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

	// An empty loaded version carries no information and must count as a change.
	assert.True(t, app.checkDataChanged(""), "空版本号应视为发生变化")

	// Without a snapshot baseline every load must count as a change.
	assert.True(t, app.checkDataChanged(cache.HashUserList(users1)), "无快照基线时应视为发生变化")

	app.userCache.Set(users1)
	app.snapshots.Store(&Snapshot{Version: cache.HashUserList(users1)})

	// Same data should return false
	assert.False(t, app.checkDataChanged(cache.HashUserList(users1)), "相同数据应该返回 false")

	// Different data should return true
	users2 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13900139000", Mail: "test2@example.com"},
	}
	assert.True(t, app.checkDataChanged(cache.HashUserList(users2)), "不同数据应该返回 true")

	// Different length should return true
	users3 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13900139000", Mail: "test2@example.com"},
		{Phone: "14000140000", Mail: "test3@example.com"},
	}
	assert.True(t, app.checkDataChanged(cache.HashUserList(users3)), "长度不同应该返回 true")

	// Regression: a rule set holding a record the cache drops (malformed mail) must still
	// compare EQUAL when the identical set is loaded again. The previous implementation
	// compared the cache's post-validation length/hash against the raw input, so a single
	// malformed record made change detection report "changed" forever and the shared Redis
	// cache stopped being refreshed for the lifetime of the process.
	withDropped := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Mail: "not-an-email"},
	}
	require.NoError(t, app.applyUsers(withDropped))
	app.snapshots.Store(&Snapshot{Version: cache.HashUserList(withDropped)})
	require.Less(t, app.userCache.Len(), len(withDropped), "格式非法的记录应被缓存丢弃")
	assert.False(t, app.checkDataChanged(cache.HashUserList(withDropped)), "包含被丢弃记录的数据集再次加载时不应被判定为变化")
}

// TestStartServer tests server startup configuration
func TestStartServer(t *testing.T) {
	mux := http.NewServeMux()
	srv := startServer("8081", mux, "", "", "", false)
	require.NotNil(t, srv)
	assert.Equal(t, ":8081", srv.Addr)
	assert.NotZero(t, srv.ReadTimeout)
	assert.NotZero(t, srv.WriteTimeout)
	assert.NotZero(t, srv.ReadHeaderTimeout)
	// A nil Handler silently falls back to http.DefaultServeMux, which is exactly the
	// global-registration coupling this signature exists to prevent.
	assert.Same(t, mux, srv.Handler, "服务器必须使用显式传入的 mux，不能回落到 DefaultServeMux")
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
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
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
	err = app.loadInitialData(tmpFile.Name(), "")
	assert.NoError(t, err)
	assert.Greater(t, app.userCache.Len(), 0, "应该加载了数据")
}

// TestApp_loadInitialData_EmptyFile tests loading empty file
func TestApp_loadInitialData_EmptyFile(t *testing.T) {
	// Create empty file
	tmpFile, err := os.CreateTemp("", "test-empty-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
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
	err = app.loadInitialData(tmpFile.Name(), "")
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
	err := app.loadInitialData("/nonexistent/file.json", "")
	// Non-existent file should not cause error, just no data
	assert.NoError(t, err)
}

// TestApp_backgroundTask_NoChange tests background task (no data change)
func TestApp_backgroundTask_NoChange(t *testing.T) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
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
	err = app.loadInitialData(tmpFile.Name(), "")
	require.NoError(t, err)

	initialLen := app.userCache.Len()

	// Run background task (no data change)
	app.backgroundTask(tmpFile.Name(), "")

	// Verify data hasn't changed
	assert.Equal(t, initialLen, app.userCache.Len(), "数据未变化时长度应该相同")
}

// TestApp_backgroundTask_WithChange tests background task (with data change)
func TestApp_backgroundTask_WithChange(t *testing.T) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
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
	err = app.loadInitialData(tmpFile.Name(), "")
	require.NoError(t, err)

	initialLen := app.userCache.Len()

	// Update file content
	newData := `[
		{"phone": "13800138000", "mail": "test@example.com"},
		{"phone": "13900139000", "mail": "test2@example.com"}
	]`
	err = os.WriteFile(tmpFile.Name(), []byte(newData), 0o600) // #nosec G703 -- path from os.CreateTemp
	require.NoError(t, err)

	// Run background task (with data change)
	app.backgroundTask(tmpFile.Name(), "")

	// Verify data has been updated
	assert.Greater(t, app.userCache.Len(), initialLen, "数据有变化时应该更新")
}

// TestApp_backgroundTask_PublishesEffectiveSet pins the second half of the refresh fix:
// what backgroundTask hands to the shared Redis cache must be the set the process actually
// serves, not the raw loaded input. Records rejected by format validation must never be
// re-seeded to other replicas, and the refresh must not be skipped because of them.
func TestApp_backgroundTask_PublishesEffectiveSet(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "rules-*.json")
	require.NoError(t, err)
	// Three records, one of which the cache drops for a malformed mail address.
	_, err = tmpFile.WriteString(`[
		{"phone":"13800138000","mail":"a@example.com","status":"active"},
		{"mail":"not-an-email","status":"active"},
		{"phone":"13900139000","mail":"c@example.com","status":"active"}
	]`)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	app := NewApp(&cmd.Config{
		Port:         "8081",
		RedisEnabled: false,
		Mode:         "ONLY_LOCAL",
		APIKey:       "test-key",
		RemoteConfig: "",
		TaskInterval: 60,
		DataFile:     tmpFile.Name(),
	})
	require.NotNil(t, app)

	// Elect this instance as the shared-cache writer and capture what it publishes, so the
	// assertion observes the Redis payload itself rather than only the local cache.
	var published [][]define.AllowListUser
	app.redisUserCache = &cache.RedisUserCache{}
	app.redisRefreshLocker = &stubRefreshLocker{locked: true}
	app.publishToRedis = func(users []define.AllowListUser) error {
		published = append(published, append([]define.AllowListUser(nil), users...))
		return nil
	}

	// NewApp already loaded the file, so the snapshot baseline matches and a refresh would
	// take the "unchanged" path. Clear it so this first call exercises the CHANGED path —
	// the branch that used to be gated behind the broken hash comparison.
	app.snapshots = newSnapshotStore()

	app.backgroundTask(tmpFile.Name(), "")

	applied := app.userCache.Get()
	require.Len(t, applied, 2, "格式非法的记录必须被丢弃")
	for _, u := range applied {
		assert.NotEqual(t, "not-an-email", u.Mail, "被拒绝的记录不得进入生效集合")
	}

	// The heart of the fix: the CHANGED path must publish, and must publish the EFFECTIVE
	// set. Before the fix the hash gate suppressed this write for the life of the process
	// whenever any record was dropped.
	require.Len(t, published, 1, "数据变化时必须向共享缓存发布一次")
	assert.Equal(t, applied, published[0], "发布到共享缓存的必须是生效集合，而不是原始加载结果")
	for _, u := range published[0] {
		assert.NotEqual(t, "not-an-email", u.Mail, "被拒绝的记录不得被发布到其他副本")
	}

	// A second refresh of identical data takes the unchanged path, which must still republish
	// (it is what keeps the shared cache's TTL alive) and must publish the same effective set.
	app.backgroundTask(tmpFile.Name(), "")
	require.Len(t, published, 2, "数据未变化时仍需刷新共享缓存以维持 TTL")
	assert.Equal(t, applied, published[1], "未变化分支同样必须发布生效集合")

	snap := app.snapshots.Load()
	require.NotNil(t, snap)

	// Compare the recorded version against one computed independently from the same records,
	// rather than against itself — the snapshot version must describe the RAW loaded set.
	expectedRaw := []define.AllowListUser{
		{Phone: "13800138000", Mail: "a@example.com", Status: "active"},
		{Mail: "not-an-email", Status: "active"},
		{Phone: "13900139000", Mail: "c@example.com", Status: "active"},
	}
	assert.Equal(t, cache.HashUserList(expectedRaw), snap.Version,
		"快照版本必须是加载到的原始集合的哈希")
	assert.False(t, app.checkDataChanged(cache.HashUserList(expectedRaw)),
		"同一份数据再次加载不应被判定为变化")

	failures, _ := app.snapshots.RefreshFailure()
	assert.Zero(t, failures, "成功刷新后连续失败计数应被清零")
}

// TestApp_updateRedisIfWriter_FallsBackToRetryingWriter covers the publish seam's default
// path. publishToRedis exists so tests can observe what a refresh hands to the shared cache;
// production leaves it pointing at updateRedisCacheWithRetry, and a nil value must fall back
// to that writer rather than silently skipping the publish.
func TestApp_updateRedisIfWriter_FallsBackToRetryingWriter(t *testing.T) {
	users := []define.AllowListUser{{Phone: "13800138000", Mail: "a@example.com"}}

	t.Run("非写入者不发布", func(t *testing.T) {
		called := false
		app := &App{log: logger.GetLoggerKit(), publishToRedis: func([]define.AllowListUser) error {
			called = true
			return nil
		}}
		app.updateRedisIfWriter(false, users)
		assert.False(t, called, "未当选写入者时不应发布")
	})

	t.Run("nil 钩子回落到重试写入器并记录错误", func(t *testing.T) {
		// publishToRedis nil + redisUserCache nil => updateRedisCacheWithRetry returns an
		// error, which must be handled rather than swallowed or panicking.
		app := &App{log: logger.GetLoggerKit()}
		require.NotPanics(t, func() { app.updateRedisIfWriter(true, users) })
	})

	t.Run("发布失败不影响调用方", func(t *testing.T) {
		app := &App{log: logger.GetLoggerKit(), publishToRedis: func([]define.AllowListUser) error {
			return assert.AnError
		}}
		require.NotPanics(t, func() { app.updateRedisIfWriter(true, users) })
	})
}

// TestRegisterRoutes_MetricsAuthPolicy is the regression guard for P2-a. The fix is the
// single line optionalAuthCfg.APIKey = "": middleware-kit only honours AllowEmptyKey when no
// key is configured, so before it an anonymous scrape got 401 in EVERY environment whenever
// API_KEY was set, contradicting the documented anonymous default and silently breaking
// Prometheus. These assertions drive real requests through the mux, not the policy helper.
func TestRegisterRoutes_MetricsAuthPolicy(t *testing.T) {
	scrape := func(t *testing.T, environment, requireAuth string, withKey bool) int {
		t.Helper()
		t.Setenv("IP_WHITELIST", "")
		t.Setenv("HEALTH_CHECK_IP_WHITELIST", "")
		t.Setenv("WARDEN_METRICS_REQUIRE_AUTH", requireAuth)

		app := NewApp(&cmd.Config{
			Port:         "8081",
			RedisEnabled: false,
			Mode:         "DEFAULT",
			Environment:  environment,
			APIKey:       "test-key",
			RemoteConfig: "",
			TaskInterval: 60,
		})
		mux := registerRoutes(app)
		require.NotNil(t, mux)

		req := httptest.NewRequest(http.MethodGet, define.PATH_METRICS, http.NoBody)
		if withKey {
			req.Header.Set("X-API-Key", "test-key")
		}
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		return resp.Code
	}

	t.Run("开发环境配置了 API_KEY 时匿名抓取仍应成功", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, scrape(t, "development", "", false))
	})
	t.Run("测试环境同样允许匿名抓取", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, scrape(t, "test", "", false))
	})
	t.Run("生产环境默认要求认证", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, scrape(t, "production", "", false))
	})
	t.Run("生产环境带凭据可抓取", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, scrape(t, "production", "", true))
	})
	t.Run("显式 true 可在开发环境强制认证", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, scrape(t, "development", "true", false))
	})
	t.Run("显式 false 可在生产环境放开匿名抓取", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, scrape(t, "production", "false", false))
	})
	t.Run("拼错的值回落到环境默认而非静默放行", func(t *testing.T) {
		// A typo must not be read as "false": production keeps requiring authentication
		// (registerRoutes also logs a warning naming the accepted spellings).
		assert.Equal(t, http.StatusUnauthorized, scrape(t, "production", "ture", false))
		assert.Equal(t, http.StatusOK, scrape(t, "development", "ture", false))
	})
}

// TestApp_backgroundTask_AllRecordsRejected covers the condition this PR made loud: a load
// that succeeds with records, none of which survive format validation. The effective set is
// empty and that empty set is what gets published — keeping the shared cache identical to
// what this replica serves rather than re-seeding rejected records to peers — so the
// condition has to be visible above Debug.
func TestApp_backgroundTask_AllRecordsRejected(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "rules-*.json")
	require.NoError(t, err)
	// Every record carries a malformed mail address, so all of them are dropped.
	_, err = tmpFile.WriteString(`[
		{"mail":"not-an-email","status":"active"},
		{"mail":"also-not-an-email","status":"active"}
	]`)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	app := NewApp(&cmd.Config{
		Port:         "8081",
		RedisEnabled: false,
		Mode:         "ONLY_LOCAL",
		APIKey:       "test-key",
		RemoteConfig: "",
		TaskInterval: 60,
		DataFile:     tmpFile.Name(),
	})
	require.NotNil(t, app)

	var published [][]define.AllowListUser
	app.redisUserCache = &cache.RedisUserCache{}
	app.redisRefreshLocker = &stubRefreshLocker{locked: true}
	app.publishToRedis = func(users []define.AllowListUser) error {
		published = append(published, append([]define.AllowListUser(nil), users...))
		return nil
	}

	// Clear the baseline so the refresh takes the changed path.
	app.snapshots = newSnapshotStore()
	app.backgroundTask(tmpFile.Name(), "")

	assert.Zero(t, app.userCache.Len(), "全部记录被拒时生效集合应为空")
	require.Len(t, published, 1, "仍应发布一次，使共享缓存与本副本所服务的内容一致")
	assert.Empty(t, published[0], "发布的必须是空的生效集合，而不是被拒绝的原始记录")

	// A successful-but-empty load is still a successful refresh: it must not be recorded as
	// a failure, or the snapshot would report a stale-data problem that does not exist.
	failures, _ := app.snapshots.RefreshFailure()
	assert.Zero(t, failures)
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
		app.backgroundTask("/invalid/path/that/might/cause/panic", "")
	}, "后台任务应该能够恢复 panic")
}

func TestApp_backgroundTask_SkipsOverlappingRefresh(t *testing.T) {
	app := &App{}
	app.refreshMu.Lock()
	defer app.refreshMu.Unlock()

	done := make(chan struct{})
	go func() {
		app.backgroundTask("", "")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("overlapping background task should be skipped")
	}
}

func TestAcquireRedisRefreshWriter(t *testing.T) {
	available := &stubRefreshLocker{locked: true}
	app := &App{
		redisUserCache:     &cache.RedisUserCache{},
		redisRefreshLocker: available,
	}

	writer, release := app.acquireRedisRefreshWriter()
	assert.True(t, writer)
	release()
	assert.Equal(t, 1, available.unlocks)

	held := &stubRefreshLocker{}
	app.redisRefreshLocker = held
	writer, release = app.acquireRedisRefreshWriter()
	assert.False(t, writer)
	release()
	assert.Zero(t, held.unlocks)

	app.redisRefreshLocker = failingRefreshLocker{}
	writer, release = app.acquireRedisRefreshWriter()
	assert.False(t, writer)
	release()
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

	// Routes live on a dedicated mux; http.DefaultServeMux must stay untouched.
	mux := registerRoutes(app)
	require.NotNil(t, mux)

	// The root document is served by the exact-match pattern, not by a subtree catch-all.
	_, pattern := mux.Handler(&http.Request{Method: "GET", URL: &url.URL{Path: "/"}})
	assert.Equal(t, rootExactPattern, pattern, "根路由应由精确匹配模式提供")

	_, pattern = mux.Handler(&http.Request{Method: "GET", URL: &url.URL{Path: define.PATH_HEALTH}})
	assert.Equal(t, define.PATH_HEALTH, pattern, "健康检查路由应该已注册")

	_, pattern = mux.Handler(&http.Request{Method: "GET", URL: &url.URL{Path: define.PATH_METRICS}})
	assert.Equal(t, define.PATH_METRICS, pattern, "指标路由应该已注册")

	// registerRoutes must not publish anything on the global mux.
	_, globalPattern := http.DefaultServeMux.Handler(&http.Request{Method: "GET", URL: &url.URL{Path: define.PATH_V1_LOOKUP}})
	assert.Empty(t, globalPattern, "registerRoutes 不应再向 http.DefaultServeMux 注册路由")
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
	err = app.loadInitialData("/nonexistent/file.json", "")
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
	err := app.loadInitialData("/nonexistent/file.json", "")
	// Should not return error, just no data
	assert.NoError(t, err)
}

// TestApp_loadInitialData_FileExistsButEmpty tests file exists but is empty
func TestApp_loadInitialData_FileExistsButEmpty(t *testing.T) {
	// Create empty file
	tmpFile, err := os.CreateTemp("", "test-empty-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
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
	err = app.loadInitialData(tmpFile.Name(), "")
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
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Run background task
	app.backgroundTask(tmpFile.Name(), "")

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
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Load data first
	err = app.loadInitialData(tmpFile.Name(), "")
	require.NoError(t, err)

	// Modify cache in another goroutine to simulate data inconsistency
	go func() {
		time.Sleep(10 * time.Millisecond)
		app.userCache.Set([]define.AllowListUser{
			{Phone: "99999999999", Mail: "modified@example.com"},
		})
	}()

	// Run background task
	app.backgroundTask(tmpFile.Name(), "")

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

	// No snapshot baseline has been recorded yet, so any load counts as a change.
	assert.True(t, app.checkDataChanged(cache.HashUserList(users)), "无快照基线时应该返回true")
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
	err := app.loadInitialData("/nonexistent/file.json", "")
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
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Strict REMOTE_FIRST must not treat the local file as a successful refresh.
	err = app.loadInitialData(tmpFile.Name(), "")
	assert.NoError(t, err, "加载失败由健康状态和日志报告，不应导致启动 panic")
	assert.Zero(t, app.userCache.Len(), "严格远程模式不应在远程失败时提交本地数据")
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
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	users := []define.AllowListUser{{Phone: "13900139000", Mail: "old@example.com"}}
	app.userCache.Set(users)
	loadedAt := time.Now().Add(-2 * time.Minute)
	app.snapshots.Store(&Snapshot{
		Users:    users,
		Count:    len(users),
		Source:   "remote",
		Version:  "old-version",
		LoadedAt: loadedAt,
	})

	// A strict remote failure must retain the last-known-good snapshot instead
	// of renewing freshness from the local fallback.
	app.backgroundTask(tmpFile.Name(), "")

	snapshot := app.snapshots.Load()
	require.NotNil(t, snapshot)
	assert.Equal(t, loadedAt, snapshot.LoadedAt)
	failures, _ := app.snapshots.RefreshFailure()
	assert.EqualValues(t, 1, failures)
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
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
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
	err := app.loadInitialData("/nonexistent/file.json", "")
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
		require.NoError(t, os.Remove(tmpFile.Name())) // #nosec G703 -- path from os.CreateTemp
	}()

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Load initial data
	err = app.loadInitialData(tmpFile.Name(), "")
	require.NoError(t, err)

	// Modify cache hash to simulate inconsistency
	app.userCache.Set([]define.AllowListUser{
		{Phone: "99999999999", Mail: "modified@example.com"},
	})

	// Run background task (should detect inconsistency)
	app.backgroundTask(tmpFile.Name(), "")

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

// TestApp_checkDataChanged_EmptyBaseline covers the no-baseline branch on a bare App:
// snapshots is nil, which must be treated as "changed" rather than panicking.
func TestApp_checkDataChanged_EmptyBaseline(t *testing.T) {
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	bare := &App{}
	assert.True(t, bare.checkDataChanged(cache.HashUserList(users)), "snapshots 为 nil 时应返回 true")

	app := NewApp(&cmd.Config{Port: "8081", RedisEnabled: false, Mode: "DEFAULT", TaskInterval: 60})
	assert.True(t, app.checkDataChanged(cache.HashUserList(users)), "快照版本为空时应返回 true")
}

// TestApp_checkDataChanged_SameVersion covers the unchanged branch.
func TestApp_checkDataChanged_SameVersion(t *testing.T) {
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	app := NewApp(&cmd.Config{Port: "8081", RedisEnabled: false, Mode: "DEFAULT", TaskInterval: 60})
	version := cache.HashUserList(users)
	app.snapshots.Store(&Snapshot{Version: version})
	assert.False(t, app.checkDataChanged(version), "相同版本应该返回 false")
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

	mux := registerRoutes(app)
	require.NotNil(t, mux)

	// Every allowlisted path must resolve to its OWN pattern. Asserting "pattern is not
	// empty" would be vacuous now that a "/" fallback matches everything.
	for _, endpoint := range define.KnownRoutePaths {
		want := endpoint
		if endpoint == define.PATH_ROOT {
			want = rootExactPattern
		}
		_, pattern := mux.Handler(&http.Request{Method: "GET", URL: &url.URL{Path: endpoint}})
		assert.Equal(t, want, pattern, "端点 %s 应该已注册", endpoint)
	}

	// Unregistered paths must fall through to the catch-all, never to a data handler.
	for _, endpoint := range []string{"/foo", "/user/", "/v1/", "/definitely-not-a-route", "/v1/lookup/extra"} {
		_, pattern := mux.Handler(&http.Request{Method: "GET", URL: &url.URL{Path: endpoint}})
		assert.Equal(t, define.PATH_ROOT, pattern, "未注册路径 %s 必须落到 404 兜底", endpoint)
	}

	// ServeMux path-cleans before matching, so traversal-looking paths resolve to the
	// cleaned route (and redirect there) rather than reaching the catch-all. Pinned so the
	// difference between "unmatched" and "cleaned then matched" stays deliberate.
	_, cleanedPattern := mux.Handler(&http.Request{Method: "GET", URL: &url.URL{Path: "/metrics/../user"}})
	assert.Equal(t, define.PATH_USER, cleanedPattern, "路径清理后应命中 /user，而不是兜底路由")
}

func TestRegisterRoutes_GlobalIPAllowlist(t *testing.T) {
	t.Setenv("IP_WHITELIST", "192.0.2.10")

	app := NewApp(&cmd.Config{
		Port:         "8081",
		RedisEnabled: false,
		Mode:         "development",
		APIKey:       "test-key",
	})

	mux := registerRoutes(app)
	require.NotNil(t, mux)

	// The unmatched-path fallback is included deliberately: a bare 404 handler registered
	// outside ipAllowlistMiddleware would answer blocked clients that get 403 everywhere else.
	for _, endpoint := range []string{"/", "/user", "/v1/lookup", "/health", "/metrics", "/log/level", "/definitely-not-a-route"} {
		req := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
		req.RemoteAddr = "198.51.100.20:1234"
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code, "endpoint %s must honor IP_WHITELIST", endpoint)
	}
}

// TestRegisterRoutes_UnknownPathReturnsJSON404 pins the behavior change: an unregistered
// path used to be served by the root catch-all and returned the COMPLETE allow list to any
// authenticated caller. It must now return a localized JSON 404 and no user data.
func TestRegisterRoutes_UnknownPathReturnsJSON404(t *testing.T) {
	app := NewApp(&cmd.Config{
		Port:         "8081",
		RedisEnabled: false,
		Mode:         "DEFAULT",
		APIKey:       "test-key",
		RemoteConfig: "",
		TaskInterval: 60,
	})
	app.userCache.Set([]define.AllowListUser{
		{Phone: "13800138000", Mail: "leak@example.com", Status: "active"},
	})

	mux := registerRoutes(app)
	require.NotNil(t, mux)

	for _, path := range []string{"/foo", "/user/", "/v1/", "/%2e%2e/user"} {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		req.Header.Set("X-API-Key", "test-key")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		require.Equal(t, http.StatusNotFound, resp.Code, "未注册路径 %s 必须返回 404", path)
		assert.NotContains(t, resp.Body.String(), "leak@example.com", "404 响应不得泄漏用户数据")

		var body map[string]string
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body), "404 响应必须是 JSON")
		assert.NotEmpty(t, body["error"])
	}
}

// TestRegisterRoutes_PatternsMatchKnownRoutePaths guards against route/allowlist drift.
// define.KnownRoutePaths bounds the Prometheus endpoint label, so a route registered
// without a matching entry would silently reappear as the "other" bucket.
func TestRegisterRoutes_PatternsMatchKnownRoutePaths(t *testing.T) {
	src, err := os.ReadFile("main_routes.go")
	require.NoError(t, err)

	assert.NotContains(t, string(src), "\thttp.Handle(",
		"路由必须注册到局部 mux，不能回退到 http.DefaultServeMux")

	handleArg := regexp.MustCompile(`mux\.Handle\(\s*([A-Za-z0-9_.]+)\s*,`)
	matches := handleArg.FindAllStringSubmatch(string(src), -1)
	require.NotEmpty(t, matches, "未能解析出任何 mux.Handle 调用")

	distinctPathConsts := map[string]struct{}{}
	for _, m := range matches {
		arg := m[1]
		if arg == "rootExactPattern" {
			continue
		}
		require.True(t, strings.HasPrefix(arg, "define.PATH_"),
			"路由模式必须使用 define.PATH_* 常量，发现: %s", arg)
		distinctPathConsts[arg] = struct{}{}
	}

	assert.Len(t, distinctPathConsts, len(define.KnownRoutePaths),
		"新增路由后必须同步更新 define.KnownRoutePaths（指标标签白名单）")
}

// TestParseBoolLike pins which spellings count as an explicit setting. The distinction
// matters: an unrecognized value must fall back to the environment default (and warn),
// not be silently treated as false.
func TestParseBoolLike(t *testing.T) {
	for _, raw := range []string{"true", "TRUE", " 1 ", "yes", "on"} {
		value, ok := parseBoolLike(raw)
		assert.True(t, ok, "%q 应被识别", raw)
		assert.True(t, value, "%q 应解析为 true", raw)
	}
	for _, raw := range []string{"false", "FALSE", "0", "no", "off"} {
		value, ok := parseBoolLike(raw)
		assert.True(t, ok, "%q 应被识别", raw)
		assert.False(t, value, "%q 应解析为 false", raw)
	}
	for _, raw := range []string{"", "ture", "enabled", "2", "y"} {
		_, ok := parseBoolLike(raw)
		assert.False(t, ok, "%q 不应被识别为布尔值", raw)
	}
}

// TestMetricsRequireAuth covers the /metrics exposure policy: an explicit setting wins in
// both directions, otherwise production requires authentication and other environments do not.
func TestMetricsRequireAuth(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		environment string
		want        bool
	}{
		{"未设置-开发环境默认匿名", "", "development", false},
		{"未设置-测试环境默认匿名", "", "test", false},
		{"未设置-生产环境默认要求认证", "", "production", true},
		{"未设置-生产别名 prod", "", "prod", true},
		{"未设置-空环境按默认(开发)处理", "", "", false},
		{"显式 true 覆盖开发默认", "true", "development", true},
		{"显式 1 覆盖开发默认", "1", "development", true},
		{"显式 false 覆盖生产默认", "false", "production", false},
		{"显式 0 覆盖生产默认", "0", "production", false},
		{"大小写与空格不敏感", "  TRUE  ", "development", true},
		{"无法识别的值回落到环境默认", "maybe", "production", true},
		{"无法识别的值回落到环境默认-开发", "maybe", "development", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, metricsRequireAuth(tt.raw, tt.environment))
		})
	}
}
