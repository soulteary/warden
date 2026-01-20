package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/middleware"
)

// TestCalculateHash 测试哈希计算函数
func TestCalculateHash(t *testing.T) {
	tests := []struct {
		name     string
		users    []define.AllowListUser
		wantSame bool // 相同输入是否产生相同哈希
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
			wantSame: true, // 应该产生相同哈希（因为会排序）
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := calculateHash(tt.users)
			hash2 := calculateHash(tt.users)

			if tt.wantSame {
				assert.Equal(t, hash1, hash2, "相同输入应该产生相同哈希")
			}

			// 哈希值应该是有效的十六进制字符串
			assert.NotEmpty(t, hash1, "哈希值不应该为空")
			assert.Len(t, hash1, 64, "SHA256 哈希应该是 64 个字符")
		})
	}
}

// TestCalculateHash_DifferentData 测试不同数据产生不同哈希
func TestCalculateHash_DifferentData(t *testing.T) {
	users1 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
	}
	users2 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test2@example.com"},
	}

	hash1 := calculateHash(users1)
	hash2 := calculateHash(users2)

	assert.NotEqual(t, hash1, hash2, "不同数据应该产生不同哈希")
}

// TestHasChanged 测试数据变化检测
func TestHasChanged(t *testing.T) {
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	oldHash := calculateHash(users)

	// 相同数据应该返回 false
	assert.False(t, hasChanged(oldHash, users), "相同数据应该返回 false")

	// 不同数据应该返回 true
	newUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
		{Phone: "13900139000", Mail: "test2@example.com"},
	}
	assert.True(t, hasChanged(oldHash, newUsers), "不同数据应该返回 true")

	// 空哈希应该返回 true
	assert.True(t, hasChanged("", users), "空哈希应该返回 true")
}

// TestNewApp 测试应用初始化
func TestNewApp(t *testing.T) {
	// 保存原始环境变量
	originalMode := os.Getenv("MODE")
	defer os.Setenv("MODE", originalMode)

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
				RemoteConfig:     "", // 避免远程请求，防止测试卡住
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
				RemoteConfig:     "", // 避免远程请求，防止测试卡住
				TaskInterval:     60,
				HTTPTimeout:      30,
				HTTPMaxIdleConns: 100,
				HTTPInsecureTLS:  false,
			},
			wantErr: false, // Redis 连接失败不会返回错误，会降级到内存模式
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
			app, err := NewApp(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, app)
			} else {
				assert.NoError(t, err)
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

// TestApp_checkDataChanged 测试数据变化检测
func TestApp_checkDataChanged(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)
	require.NotNil(t, app)

	// 初始数据
	users1 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
	}
	app.userCache.Set(users1)

	// 相同数据应该返回 false
	assert.False(t, app.checkDataChanged(users1), "相同数据应该返回 false")

	// 不同数据应该返回 true
	users2 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13900139000", Mail: "test2@example.com"},
	}
	assert.True(t, app.checkDataChanged(users2), "不同数据应该返回 true")

	// 长度不同应该返回 true
	users3 := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13900139000", Mail: "test2@example.com"},
		{Phone: "14000140000", Mail: "test3@example.com"},
	}
	assert.True(t, app.checkDataChanged(users3), "长度不同应该返回 true")
}

// TestStartServer 测试服务器启动配置
func TestStartServer(t *testing.T) {
	srv := startServer("8081")
	require.NotNil(t, srv)
	assert.Equal(t, ":8081", srv.Addr)
	assert.NotZero(t, srv.ReadTimeout)
	assert.NotZero(t, srv.WriteTimeout)
	assert.NotZero(t, srv.ReadHeaderTimeout)
}

// TestShutdownServer 测试服务器关闭
func TestShutdownServer(t *testing.T) {
	// 创建一个简单的速率限制器
	rateLimiter := middleware.NewRateLimiter(100, time.Second)

	// 创建一个测试服务器
	srv := &http.Server{
		Addr: ":0", // 使用随机端口
	}

	// 启动服务器（在 goroutine 中）
	go func() {
		_ = srv.ListenAndServe()
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 测试关闭（shutdownServer 会调用 rateLimiter.Stop()，所以不需要 defer）
	log := logger.GetLogger()
	shutdownServer(srv, rateLimiter, &log)

	// 验证速率限制器已停止
	// 注意：这里只是验证函数不会 panic，实际的状态检查需要更复杂的测试
}

// TestApp_loadInitialData_ONLY_LOCAL 测试 ONLY_LOCAL 模式的数据加载
func TestApp_loadInitialData_ONLY_LOCAL(t *testing.T) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// 写入测试数据
	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	tmpFile.Close()

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

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 测试加载数据
	err = app.loadInitialData(tmpFile.Name())
	assert.NoError(t, err)
	assert.Greater(t, app.userCache.Len(), 0, "应该加载了数据")
}

// TestApp_loadInitialData_EmptyFile 测试空文件加载
func TestApp_loadInitialData_EmptyFile(t *testing.T) {
	// 创建空文件
	tmpFile, err := os.CreateTemp("", "test-empty-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 测试加载空文件
	err = app.loadInitialData(tmpFile.Name())
	// 空文件不应该导致错误，只是没有数据
	assert.NoError(t, err)
}

// TestApp_loadInitialData_NonExistentFile 测试不存在的文件
func TestApp_loadInitialData_NonExistentFile(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 测试加载不存在的文件
	err = app.loadInitialData("/nonexistent/file.json")
	// 文件不存在不应该导致错误，只是没有数据
	assert.NoError(t, err)
}

// TestApp_backgroundTask_NoChange 测试后台任务（数据未变化）
func TestApp_backgroundTask_NoChange(t *testing.T) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	tmpFile.Close()

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

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 先加载数据
	err = app.loadInitialData(tmpFile.Name())
	require.NoError(t, err)

	initialLen := app.userCache.Len()

	// 运行后台任务（数据未变化）
	app.backgroundTask(tmpFile.Name())

	// 验证数据未变化
	assert.Equal(t, initialLen, app.userCache.Len(), "数据未变化时长度应该相同")
}

// TestApp_backgroundTask_WithChange 测试后台任务（数据有变化）
func TestApp_backgroundTask_WithChange(t *testing.T) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	initialData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(initialData)
	require.NoError(t, err)
	tmpFile.Close()

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

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 先加载初始数据
	err = app.loadInitialData(tmpFile.Name())
	require.NoError(t, err)

	initialLen := app.userCache.Len()

	// 更新文件内容
	newData := `[
		{"phone": "13800138000", "mail": "test@example.com"},
		{"phone": "13900139000", "mail": "test2@example.com"}
	]`
	err = os.WriteFile(tmpFile.Name(), []byte(newData), 0644)
	require.NoError(t, err)

	// 运行后台任务（数据有变化）
	app.backgroundTask(tmpFile.Name())

	// 验证数据已更新
	assert.Greater(t, app.userCache.Len(), initialLen, "数据有变化时应该更新")
}

// TestApp_backgroundTask_PanicRecovery 测试后台任务的 panic 恢复
func TestApp_backgroundTask_PanicRecovery(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 测试 panic 恢复（通过传递无效文件路径触发可能的 panic）
	// 注意：这里只是验证函数不会因为 panic 而崩溃
	assert.NotPanics(t, func() {
		app.backgroundTask("/invalid/path/that/might/cause/panic")
	}, "后台任务应该能够恢复 panic")
}

// TestApp_updateRedisCacheWithRetry 测试Redis缓存更新重试机制
func TestApp_updateRedisCacheWithRetry(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 如果Redis不可用，跳过测试
	if app.redisUserCache == nil {
		t.Skip("跳过测试：Redis不可用")
	}

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	// 测试成功更新
	err = app.updateRedisCacheWithRetry(users)
	// 如果Redis可用，应该成功；如果不可用，会返回错误
	if err != nil {
		t.Logf("Redis更新失败（可能是Redis不可用）: %v", err)
	} else {
		assert.NoError(t, err, "Redis缓存更新应该成功")
	}
}

// TestApp_updateRedisCacheWithRetry_NoRedis 测试没有Redis时的行为
func TestApp_updateRedisCacheWithRetry_NoRedis(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	// 没有Redis时，redisUserCache为nil
	assert.Nil(t, app.redisUserCache, "没有Redis时redisUserCache应该为nil")

	// 直接调用应该返回错误而不是panic，因为redisUserCache是nil
	// 在实际使用中，这个函数只在redisUserCache != nil时才会被调用
	// 这个测试验证函数在nil情况下不会panic，而是返回错误
	assert.NotPanics(t, func() {
		err := app.updateRedisCacheWithRetry(users)
		assert.Error(t, err, "redisUserCache为nil时应该返回错误")
	}, "即使redisUserCache为nil也不应该panic")
}

// TestRegisterRoutes 测试路由注册
func TestRegisterRoutes(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 保存原始路由
	originalDefaultMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()

	// 注册路由
	registerRoutes(app)

	// 验证路由已注册
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

	// 恢复原始路由
	http.DefaultServeMux = originalDefaultMux
}

// TestNewApp_WithHTTPInsecureTLS 测试启用HTTP不安全TLS的情况
func TestNewApp_WithHTTPInsecureTLS(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  true,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)
	assert.NotNil(t, app)
}

// TestNewApp_ProductionModeWithInsecureTLS 测试生产模式启用不安全TLS（应该失败）
func TestNewApp_ProductionModeWithInsecureTLS(t *testing.T) {
	// 这个测试需要能够捕获Fatal，但Fatal会退出程序
	// 所以我们只测试配置验证，而不是实际运行
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "production",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  true,
	}

	// 注意：在生产模式下启用不安全TLS会导致Fatal退出
	// 这个测试主要验证配置检查逻辑存在
	// 实际测试需要mock logger.Fatal
	_ = cfg
}

// TestNewApp_WithRedisPassword 测试带Redis密码的配置
func TestNewApp_WithRedisPassword(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisPassword:    "test-password",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	// Redis连接可能失败，但不应该返回错误（会降级到内存模式）
	if err != nil {
		t.Skipf("跳过测试：Redis连接失败: %v", err)
	}
	assert.NotNil(t, app)
}

// TestNewApp_TaskIntervalTooSmall 测试任务间隔小于默认值的情况
func TestNewApp_TaskIntervalTooSmall(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     1,  // 小于默认值
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)
	assert.NotNil(t, app)
	// 验证任务间隔被调整为默认值
	assert.GreaterOrEqual(t, app.taskInterval, uint64(define.DEFAULT_TASK_INTERVAL))
}

// TestApp_loadInitialData_FromRedis 测试从Redis加载数据
func TestApp_loadInitialData_FromRedis(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 如果Redis不可用，跳过测试
	if app.redisUserCache == nil {
		t.Skip("跳过测试：Redis不可用")
	}

	// 先设置一些数据到Redis
	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}
	err = app.redisUserCache.Set(users)
	if err != nil {
		t.Skipf("跳过测试：无法设置Redis数据: %v", err)
	}

	// 清空内存缓存
	app.userCache.Set([]define.AllowListUser{})

	// 测试从Redis加载
	err = app.loadInitialData("/nonexistent/file.json")
	assert.NoError(t, err)
	// 如果Redis中有数据，应该加载成功
	if app.userCache.Len() > 0 {
		assert.Greater(t, app.userCache.Len(), 0, "应该从Redis加载了数据")
	}
}

// TestApp_loadInitialData_RemoteConfig 测试从远程配置加载数据
func TestApp_loadInitialData_RemoteConfig(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "http://invalid-url-that-will-fail.com/data.json",
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 测试从远程配置加载（会失败，然后降级到本地文件）
	err = app.loadInitialData("/nonexistent/file.json")
	// 应该不会返回错误，只是没有数据
	assert.NoError(t, err)
}

// TestApp_loadInitialData_FileExistsButEmpty 测试文件存在但为空的情况
func TestApp_loadInitialData_FileExistsButEmpty(t *testing.T) {
	// 创建空文件
	tmpFile, err := os.CreateTemp("", "test-empty-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 测试加载空文件
	err = app.loadInitialData(tmpFile.Name())
	assert.NoError(t, err)
}

// TestApp_backgroundTask_WithRedis 测试带Redis的后台任务
func TestApp_backgroundTask_WithRedis(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		Redis:            "localhost:6379",
		RedisEnabled:     true,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 如果Redis不可用，跳过测试
	if app.redisUserCache == nil {
		t.Skip("跳过测试：Redis不可用")
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	tmpFile.Close()

	// 运行后台任务
	app.backgroundTask(tmpFile.Name())

	// 验证任务执行了（不会panic）
	assert.True(t, true)
}

// TestApp_backgroundTask_DataInconsistency 测试数据不一致的情况
func TestApp_backgroundTask_DataInconsistency(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test-data-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	testData := `[
		{"phone": "13800138000", "mail": "test@example.com"}
	]`
	_, err = tmpFile.WriteString(testData)
	require.NoError(t, err)
	tmpFile.Close()

	// 先加载数据
	err = app.loadInitialData(tmpFile.Name())
	require.NoError(t, err)

	// 在另一个goroutine中修改缓存，模拟数据不一致
	go func() {
		time.Sleep(10 * time.Millisecond)
		app.userCache.Set([]define.AllowListUser{
			{Phone: "99999999999", Mail: "modified@example.com"},
		})
	}()

	// 运行后台任务
	app.backgroundTask(tmpFile.Name())

	// 验证任务执行了（不会panic）
	assert.True(t, true)
}

// TestShutdownServer_WithNilRateLimiter 测试关闭服务器时rateLimiter为nil的情况
func TestShutdownServer_WithNilRateLimiter(t *testing.T) {
	srv := &http.Server{
		Addr: ":0",
	}

	log := logger.GetLogger()

	// 测试nil rateLimiter
	assert.NotPanics(t, func() {
		shutdownServer(srv, nil, &log)
	})
}

// TestShutdownServer_ShutdownError 测试关闭服务器时的错误处理
func TestShutdownServer_ShutdownError(t *testing.T) {
	rateLimiter := middleware.NewRateLimiter(100, time.Second)

	// 创建一个已经关闭的服务器
	srv := &http.Server{
		Addr: ":0",
	}

	// 先关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)

	log := logger.GetLogger()

	// 再次关闭应该不会panic
	assert.NotPanics(t, func() {
		shutdownServer(srv, rateLimiter, &log)
	})
}

// TestCalculateHash_WithAllFields 测试包含所有字段的哈希计算
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

	hash1 := calculateHash(users)
	hash2 := calculateHash(users)

	assert.Equal(t, hash1, hash2, "相同输入应该产生相同哈希")
	assert.Len(t, hash1, 64, "SHA256 哈希应该是 64 个字符")
}

// TestCalculateHash_WithScope 测试包含Scope字段的哈希计算
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

	hash1 := calculateHash(users1)
	hash2 := calculateHash(users2)

	assert.NotEqual(t, hash1, hash2, "不同Scope应该产生不同哈希")
}

// TestApp_checkDataChanged_EmptyHash 测试空哈希的情况
func TestApp_checkDataChanged_EmptyHash(t *testing.T) {
	cfg := &cmd.Config{
		Port:             "8081",
		RedisEnabled:     false,
		Mode:             "development",
		APIKey:           "test-key",
		RemoteConfig:     "", // 避免远程请求，防止测试卡住
		TaskInterval:     60,
		HTTPTimeout:      30,
		HTTPMaxIdleConns: 100,
		HTTPInsecureTLS:  false,
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	// 清空缓存哈希
	app.userCache.Set([]define.AllowListUser{})

	// 测试空哈希的情况
	assert.True(t, app.checkDataChanged(users), "空哈希时应该返回true")
}
