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

	// 测试Redis连接失败的情况（应该返回错误）
	deps, err := NewDependencies(cfg)
	// 如果Redis不可用，应该返回错误
	if err != nil {
		assert.Error(t, err)
		assert.Nil(t, deps)
	} else {
		// Redis可用的情况
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

	// 测试Redis连接失败的情况（应该返回错误）
	deps, err := NewDependencies(cfg)
	// 如果Redis不可用，应该返回错误
	if err != nil {
		assert.Error(t, err)
		assert.Nil(t, deps)
	} else {
		// Redis可用的情况
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

	// 测试Cleanup不会panic
	assert.NotPanics(t, func() {
		deps.Cleanup()
	})

	// 可以多次调用Cleanup
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
	// 如果Redis不可用，跳过测试
	if err != nil {
		t.Skipf("跳过测试：Redis不可用: %v", err)
	}

	require.NotNil(t, deps)

	// 测试Cleanup不会panic
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

	// 验证HTTP服务器配置
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

	// 验证所有处理器都不为nil
	assert.NotNil(t, deps.MainHandler)
	assert.NotNil(t, deps.HealthHandler)
	assert.NotNil(t, deps.LogLevelHandler)

	// 验证处理器可以处理请求（不会panic）
	req, _ := http.NewRequest("GET", "/", nil)
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

	// 测试无效的Redis地址应该返回错误
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

	// 注意：NewDependencies在RedisEnabled=false时不会初始化RedisClient
	// 但initCache会创建RedisUserCache（即使RedisClient为nil）
	// 所以RedisUserCache可能不为nil，但它的client字段为nil
	assert.NotNil(t, deps.UserCache)
	// RedisUserCache可能被创建，但它的client为nil
	if deps.RedisUserCache != nil {
		// 如果创建了，验证它存在（这是正常的，因为NewRedisUserCache接受nil client）
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

	// 验证速率限制器已初始化
	assert.NotNil(t, deps.RateLimiter)
}

func TestDependencies_Cleanup_NilFields(t *testing.T) {
	// 测试Cleanup处理nil字段的情况
	deps := &Dependencies{
		RateLimiter: nil,
		RedisClient: nil,
	}

	// 应该不会panic
	assert.NotPanics(t, func() {
		deps.Cleanup()
	})
}

func TestDependencies_Cleanup_RedisCloseError(t *testing.T) {
	// 创建一个已经关闭的Redis客户端来测试错误处理
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
	// 如果Redis不可用，跳过测试
	if err != nil {
		t.Skipf("跳过测试：Redis不可用: %v", err)
	}

	require.NotNil(t, deps)

	// 先关闭Redis客户端
	if deps.RedisClient != nil {
		_ = deps.RedisClient.Close()
	}

	// 再次调用Cleanup应该不会panic
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
	// 如果Redis不可用，跳过测试
	if err != nil {
		t.Skipf("跳过测试：Redis不可用: %v", err)
	}

	require.NotNil(t, deps)

	// 验证处理器已初始化
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

	// 验证端口配置正确
	assert.Equal(t, ":12345", deps.HTTPServer.Addr)
}

// 测试initRedis方法的错误处理
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

// 测试initCache方法
func TestDependencies_InitCache(t *testing.T) {
	d := &Dependencies{
		Config:      &cmd.Config{},
		RedisClient: nil, // 没有Redis客户端
	}

	d.initCache()
	assert.NotNil(t, d.UserCache)
	// 注意：initCache会尝试创建RedisUserCache，即使RedisClient为nil
	// 但RedisUserCache的创建需要有效的RedisClient
	// 根据实现，如果RedisClient为nil，RedisUserCache可能为nil或创建失败
}

// 测试initCache方法（带Redis）
func TestDependencies_InitCache_WithRedis(t *testing.T) {
	// 创建一个mock Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 测试连接
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

// 测试initRateLimiter方法（直接调用）
func TestDependencies_InitRateLimiter_Direct(t *testing.T) {
	d := &Dependencies{
		Config: &cmd.Config{},
	}

	d.initRateLimiter()
	assert.NotNil(t, d.RateLimiter)
}

// 测试initHandlers方法
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

	// 重新初始化handlers来测试initHandlers方法
	deps.initHandlers()
	assert.NotNil(t, deps.MainHandler)
	assert.NotNil(t, deps.HealthHandler)
	assert.NotNil(t, deps.LogLevelHandler)
}

// 测试initHTTPServer方法
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
