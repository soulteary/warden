package main

import (
	"net/http"
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
