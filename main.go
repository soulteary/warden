// Package main 是应用程序的入口点。
// 提供 HTTP 服务器、缓存管理、定时任务调度等功能。
package main

import (
	// 标准库
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	// 第三方库
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/cache"
	"soulteary.com/soulteary/warden/internal/cmd"
	"soulteary.com/soulteary/warden/internal/define"
	"soulteary.com/soulteary/warden/internal/errors"
	"soulteary.com/soulteary/warden/internal/logger"
	"soulteary.com/soulteary/warden/internal/metrics"
	"soulteary.com/soulteary/warden/internal/middleware"
	"soulteary.com/soulteary/warden/internal/parser"
	"soulteary.com/soulteary/warden/internal/router"
	"soulteary.com/soulteary/warden/internal/version"
	"soulteary.com/soulteary/warden/pkg/gocron"
)

const rulesFile = "./data.json"

// App 应用结构体，封装所有应用状态
type App struct {
	userCache           *cache.SafeUserCache    // 8 bytes pointer
	redisUserCache      *cache.RedisUserCache   // 8 bytes pointer
	redisClient         *redis.Client           // 8 bytes pointer
	rateLimiter         *middleware.RateLimiter // 8 bytes pointer
	log                 zerolog.Logger          // 24 bytes (interface)
	port                string                  // 16 bytes
	configURL           string                  // 16 bytes
	authorizationHeader string                  // 16 bytes
	appMode             string                  // 16 bytes
	apiKey              string                  // 16 bytes
	taskInterval        uint64                  // 8 bytes
}

// NewApp 创建新的应用实例
func NewApp(cfg *cmd.Config) (*App, error) {
	app := &App{
		port:                cfg.Port,
		configURL:           cfg.RemoteConfig,
		authorizationHeader: cfg.RemoteKey,
		appMode:             cfg.Mode,
		// #nosec G115 -- 转换是安全的，TaskInterval 是正数
		taskInterval: uint64(cfg.TaskInterval),
		apiKey:       cfg.APIKey,
		log:          logger.GetLogger(),
	}

	// 初始化 Redis 客户端（安全性改进）
	redisOptions := &redis.Options{Addr: cfg.Redis}
	if cfg.RedisPassword != "" {
		redisOptions.Password = cfg.RedisPassword
		// 安全检查：如果密码是通过命令行参数传递的，记录警告
		// 注意：这里无法直接判断密码来源，但可以通过环境变量检查来推断
		if os.Getenv("REDIS_PASSWORD") == "" && os.Getenv("REDIS_PASSWORD_FILE") == "" {
			app.log.Warn().Msg("⚠️  安全警告：Redis 密码通过命令行参数传递，建议使用环境变量 REDIS_PASSWORD 或 REDIS_PASSWORD_FILE")
		}
	}
	app.redisClient = redis.NewClient(redisOptions)

	// 验证 Redis 连接（带超时）
	ctx, cancel := context.WithTimeout(context.Background(), define.REDIS_CONNECTION_TIMEOUT)
	defer cancel()
	if err := app.redisClient.Ping(ctx).Err(); err != nil {
		return nil, errors.ErrRedisConnection.WithError(err)
	}
	app.log.Info().Str("redis", cfg.Redis).Msg("Redis 连接成功 ✓")

	// 初始化 HTTP 客户端（使用配置）
	parser.InitHTTPClient(cfg.HTTPTimeout, cfg.HTTPMaxIdleConns, cfg.HTTPInsecureTLS)
	if cfg.HTTPInsecureTLS {
		app.log.Warn().Msg("HTTP TLS 证书验证已禁用（仅用于开发环境）")
		// 在生产环境，强制启用 TLS 验证
		if cfg.Mode == "production" || cfg.Mode == "prod" {
			app.log.Fatal().Msg("生产环境不允许禁用 TLS 证书验证，程序退出")
		}
	}

	// 初始化缓存
	app.redisUserCache = cache.NewRedisUserCache(app.redisClient)
	app.userCache = cache.NewSafeUserCache()

	// 加载初始数据（多级降级）
	if err := app.loadInitialData(rulesFile); err != nil {
		app.log.Warn().Err(fmt.Errorf("加载初始数据失败: %w", err)).Msg("加载初始数据失败，使用空数据")
	}

	// 初始化缓存大小指标
	metrics.CacheSize.Set(float64(app.userCache.Len()))

	// 确保任务间隔不小于默认值
	if app.taskInterval < define.DEFAULT_TASK_INTERVAL {
		app.taskInterval = uint64(define.DEFAULT_TASK_INTERVAL)
	}

	// 初始化速率限制器（封装到 App 中，避免使用全局变量）
	app.rateLimiter = middleware.NewRateLimiter(define.DEFAULT_RATE_LIMIT, define.DEFAULT_RATE_LIMIT_WINDOW)

	return app, nil
}

// loadInitialData 多级降级加载数据
func (app *App) loadInitialData(rulesFile string) error {
	// 1. 尝试从 Redis 缓存加载
	if cachedUsers, err := app.redisUserCache.Get(); err == nil && len(cachedUsers) > 0 {
		metrics.CacheHits.Inc() // 记录缓存命中
		app.log.Info().
			Int("count", len(cachedUsers)).
			Msg("从 Redis 缓存加载数据 ✓")
		app.userCache.Set(cachedUsers)
		return nil
	}
	metrics.CacheMisses.Inc() // 记录缓存未命中

	// 2. 尝试从远程 API 加载
	ctx, cancel := context.WithTimeout(context.Background(), define.DEFAULT_LOAD_DATA_TIMEOUT)
	defer cancel()
	users := parser.GetRules(ctx, rulesFile, app.configURL, app.authorizationHeader, app.appMode)
	if len(users) > 0 {
		app.log.Info().
			Int("count", len(users)).
			Msg("从远程 API 加载数据 ✓")
		app.userCache.Set(users)
		// 同时更新 Redis 缓存
		if err := app.redisUserCache.Set(users); err != nil {
			app.log.Warn().Err(err).Msg("更新 Redis 缓存失败")
		}
		return nil
	}

	// 3. 尝试从本地文件加载
	localUsers := parser.FromFile(rulesFile)
	if len(localUsers) > 0 {
		app.log.Info().
			Int("count", len(localUsers)).
			Msg("从本地文件加载数据 ✓")
		app.userCache.Set(localUsers)
		// 同时更新 Redis 缓存
		if err := app.redisUserCache.Set(localUsers); err != nil {
			app.log.Warn().Err(err).Msg("更新 Redis 缓存失败")
		}
		return nil
	}

	// 4. 都失败，使用空数据
	app.log.Warn().Msg("所有数据源都失败，使用空数据")
	return nil
}

// hasChanged 比较数据是否有变化（使用缓存的哈希值优化）
//
// 该函数通过比较缓存的哈希值来判断数据是否发生变化，用于优化缓存更新策略。
// 优先使用缓存的哈希值，避免重复计算。
//
// 参数:
//   - oldHash: 旧数据的缓存哈希值
//   - newUsers: 新的用户列表
//
// 返回:
//   - bool: true 表示数据有变化，false 表示数据未变化
//
// 注意:
//   - 该函数优先使用缓存的哈希值，避免重复计算
//   - 如果提供了缓存的哈希值，可以显著提高性能
func hasChanged(oldHash string, newUsers []define.AllowListUser) bool {
	// 计算新数据的哈希值
	newHash := calculateHash(newUsers)
	return oldHash != newHash
}

// calculateHash 计算用户列表的 SHA256 哈希值
//
// 该函数用于检测用户数据是否发生变化，通过计算哈希值来比较数据内容。
// 实现细节：
// - 对数据进行排序（按 Phone 和 Mail）确保相同数据产生相同哈希
// - 使用 SHA256 算法计算哈希值
// - 对于空数据，返回固定哈希值以优化性能
//
// 参数:
//   - users: 要计算哈希的用户列表
//
// 返回:
//   - string: 十六进制编码的 SHA256 哈希值
//
// 副作用:
//   - 会创建输入数据的副本进行排序，不修改原始数据
//   - 对于大数据集，排序操作可能有性能开销
//
// 优化:
//   - 空数据直接返回固定哈希，避免不必要的计算
//   - 使用数据副本排序，保持原始数据不变
func calculateHash(users []define.AllowListUser) string {
	// 优化：空数据直接返回固定哈希
	if len(users) == 0 {
		h := sha256.New()
		h.Write([]byte("empty"))
		return hex.EncodeToString(h.Sum(nil))
	}

	// 先排序，确保相同数据产生相同哈希
	// 优化：如果数据量很大，可以考虑使用原地排序，但为了保持数据不变，使用副本
	sorted := make([]define.AllowListUser, len(users))
	copy(sorted, users)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Phone != sorted[j].Phone {
			return sorted[i].Phone < sorted[j].Phone
		}
		return sorted[i].Mail < sorted[j].Mail
	})

	// 计算哈希
	h := sha256.New()
	for _, user := range sorted {
		// 使用分隔符确保不同字段不会混淆
		h.Write([]byte(user.Phone + ":" + user.Mail + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// checkDataChanged 检查数据是否有变化
//
// 该函数通过比较缓存的哈希值和长度来判断数据是否发生变化。
// 优先使用缓存的哈希值，避免重复计算。
//
// 参数:
//   - newUsers: 新的用户列表
//
// 返回:
//   - bool: true 表示数据有变化，false 表示数据未变化
func (app *App) checkDataChanged(newUsers []define.AllowListUser) bool {
	oldHash := app.userCache.GetHash()
	oldLen := app.userCache.Len()

	if oldLen != len(newUsers) {
		return true
	}

	if oldHash != "" && !hasChanged(oldHash, newUsers) {
		return false
	}

	return true
}

// updateRedisCacheWithRetry 更新 Redis 缓存，带重试机制
//
// 该函数实现了带重试的 Redis 缓存更新逻辑，最多重试 define.REDIS_RETRY_MAX_RETRIES 次。
// 每次重试的延迟时间会递增。
//
// 参数:
//   - users: 要更新的用户列表
//
// 返回:
//   - error: 更新失败时返回错误，成功时返回 nil
func (app *App) updateRedisCacheWithRetry(users []define.AllowListUser) error {
	var lastErr error
	for attempt := 0; attempt < define.REDIS_RETRY_MAX_RETRIES; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * define.REDIS_RETRY_DELAY)
			app.log.Debug().
				Int("attempt", attempt+1).
				Msg("重试更新 Redis 缓存")
		}

		if err := app.redisUserCache.Set(users); err != nil {
			lastErr = err
			if attempt < define.REDIS_RETRY_MAX_RETRIES-1 {
				continue
			}
		} else {
			if cacheVersion, err := app.redisUserCache.GetVersion(); err == nil {
				app.log.Debug().
					Int64("version", cacheVersion).
					Msg("Redis 缓存已更新")
			}
			return nil
		}
	}

	return fmt.Errorf("更新 Redis 缓存失败（已重试 %d 次）: %w", define.REDIS_RETRY_MAX_RETRIES, lastErr)
}

// backgroundTask 后台任务，定期更新缓存数据
//
// 该函数实现了智能的缓存更新策略，包括以下特性：
// - 数据变化检测：通过哈希比较避免不必要的更新
// - 乐观锁策略：使用乐观锁确保数据一致性
// - 错误恢复：包含 panic 恢复机制，防止任务崩溃影响主程序
// - 重试机制：Redis 更新失败时自动重试
// - 指标收集：记录任务执行时间、错误次数等指标
//
// 参数:
//   - rulesFile: 本地规则文件路径，作为数据源之一
//
// 副作用:
//   - 更新内存缓存（app.userCache）
//   - 更新 Redis 缓存（app.redisUserCache）
//   - 更新 Prometheus 指标（metrics.BackgroundTaskTotal、metrics.BackgroundTaskDuration 等）
//   - 记录日志（调试、信息、警告级别）
//
// 错误处理:
//   - 如果发生 panic，会捕获并记录错误，不影响主程序运行
//   - Redis 更新失败会重试，最终失败时记录警告但继续使用内存缓存
//
// 性能优化:
//   - 在锁外进行数据比较，减少锁持有时间
//   - 使用哈希值快速检测数据变化
//   - 数据未变化时直接返回，跳过更新操作
func (app *App) backgroundTask(rulesFile string) {
	// 添加错误恢复机制，防止 panic 导致整个程序崩溃
	defer func() {
		if r := recover(); r != nil {
			metrics.BackgroundTaskErrors.Inc()
			app.log.Error().
				Interface("panic", r).
				Msg("后台任务发生 panic，已恢复")
		}
	}()

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(define.DEFAULT_TIMEOUT*2)*time.Second)
	defer cancel()
	newUsers := parser.GetRules(ctx, rulesFile, app.configURL, app.authorizationHeader, app.appMode)

	// 检查数据是否有变化
	if !app.checkDataChanged(newUsers) {
		app.log.Debug().Msg("数据未变化，跳过更新")
		return
	}

	// 更新内存缓存
	app.userCache.Set(newUsers)

	// 验证数据一致性（乐观锁策略）
	currentHash := app.userCache.GetHash()
	newHash := calculateHash(newUsers)
	if currentHash != "" && currentHash == newHash {
		// 数据一致，更新 Redis 缓存
		if err := app.updateRedisCacheWithRetry(newUsers); err != nil {
			app.log.Warn().
				Err(err).
				Msg("更新 Redis 缓存失败，继续使用内存缓存")
			metrics.BackgroundTaskErrors.Inc()
		}
	} else {
		currentLen := app.userCache.Len()
		app.log.Debug().
			Int("expected_count", len(newUsers)).
			Int("actual_count", currentLen).
			Msg("数据在更新过程中被修改，跳过 Redis 更新")
	}

	// 更新指标
	duration := time.Since(start).Seconds()
	metrics.BackgroundTaskTotal.Inc()
	metrics.BackgroundTaskDuration.Observe(duration)
	metrics.CacheSize.Set(float64(app.userCache.Len()))

	app.log.Info().
		Int("count", len(newUsers)).
		Float64("duration", duration).
		Msg("后台更新数据 📦")
}

// registerRoutes 注册所有 HTTP 路由
func registerRoutes(app *App) {
	// 创建基础中间件
	securityHeadersMiddleware := middleware.SecurityHeadersMiddleware
	errorHandlerMiddleware := middleware.ErrorHandlerMiddleware(app.appMode)
	rateLimitMiddleware := middleware.RateLimitMiddlewareWithLimiter(app.rateLimiter)
	authMiddleware := middleware.AuthMiddleware(app.apiKey)

	// 健康检查端点 IP 白名单（从环境变量读取）
	healthWhitelist := os.Getenv("HEALTH_CHECK_IP_WHITELIST")
	healthIPWhitelist := middleware.IPWhitelistMiddleware(healthWhitelist)

	// 注册 Prometheus metrics 端点（可选认证）
	metricsHandler := securityHeadersMiddleware(
		errorHandlerMiddleware(
			middleware.OptionalAuthMiddleware(app.apiKey)(
				middleware.MetricsMiddleware(metrics.Handler()),
			),
		),
	)
	http.Handle("/metrics", metricsHandler)

	// 注册主数据接口（需要认证）
	mainHandler := securityHeadersMiddleware(
		errorHandlerMiddleware(
			middleware.CompressMiddleware(
				middleware.BodyLimitMiddleware(
					middleware.MetricsMiddleware(
						rateLimitMiddleware(
							authMiddleware(
								router.ProcessWithLogger(router.JSON(app.userCache)),
							),
						),
					),
				),
			),
		),
	)
	http.Handle("/", mainHandler)

	// 注册健康检查端点（IP 白名单保护，限制信息泄露）
	healthHandler := securityHeadersMiddleware(
		errorHandlerMiddleware(
			healthIPWhitelist(
				middleware.MetricsMiddleware(
					router.ProcessWithLogger(router.HealthCheck(app.redisClient, app.userCache, app.appMode)),
				),
			),
		),
	)
	http.Handle("/health", healthHandler)
	http.Handle("/healthcheck", healthHandler)

	// 注册日志级别控制端点（需要认证）
	logLevelHandler := securityHeadersMiddleware(
		errorHandlerMiddleware(
			middleware.MetricsMiddleware(
				authMiddleware(
					router.ProcessWithLogger(router.LogLevelHandler()),
				),
			),
		),
	)
	http.Handle("/log/level", logLevelHandler)
}

// startServer 启动 HTTP 服务器
func startServer(port string) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: define.DEFAULT_TIMEOUT * time.Second,
		ReadTimeout:       define.DEFAULT_TIMEOUT * time.Second,
		WriteTimeout:      define.DEFAULT_TIMEOUT * time.Second,
		IdleTimeout:       define.IDLE_TIMEOUT,
		MaxHeaderBytes:    define.MAX_HEADER_BYTES,
	}
}

// shutdownServer 优雅关闭服务器
func shutdownServer(srv *http.Server, rateLimiter *middleware.RateLimiter, log *zerolog.Logger) {
	// 停止速率限制器
	if rateLimiter != nil {
		rateLimiter.Stop()
	}

	// 优雅关闭 HTTP 服务器
	shutdownCtx, cancel := context.WithTimeout(context.Background(), define.SHUTDOWN_TIMEOUT)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Info().Err(fmt.Errorf("程序强制关闭: %w", err)).Msg("程序强制关闭")
	}
}

func main() {
	log := logger.GetLogger()

	// 解析配置
	cfg := cmd.GetArgs()

	// 验证配置
	if err := cmd.ValidateConfig(cfg); err != nil {
		log.Fatal().
			Err(err).
			Msg("配置验证失败，程序退出")
	}

	// 初始化应用
	app, err := NewApp(cfg)
	if err != nil {
		log.Fatal().
			Err(errors.ErrAppInit.WithError(err)).
			Msg("应用初始化失败，程序退出")
	}

	// 注册路由
	registerRoutes(app)

	// 设置信号处理
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app.log.Info().Msgf("程序版本：%s, 构建时间：%s, 代码版本：%s", version.Version, version.BuildDate, version.Commit)

	// 启动定时任务调度器
	gocron.SetLocker(&cache.Locker{Cache: app.redisClient})
	scheduler := gocron.NewScheduler()
	schedulerStopped := scheduler.Start()
	defer func() {
		close(schedulerStopped)
		scheduler.Clear()
		app.log.Info().Msg("定时任务调度器已关闭")
	}()
	if err := scheduler.Every(app.taskInterval).Seconds().Lock().Do(app.backgroundTask, rulesFile); err != nil {
		// 在退出前先清理资源（defer 会在函数返回时执行，但 log.Fatal 会立即退出）
		// 所以需要手动清理
		close(schedulerStopped)
		scheduler.Clear()
		stop()
		//nolint:gocritic // exitAfterDefer: 需要在错误时立即退出，已手动清理资源
		log.Fatal().
			Err(err).
			Msg("定时任务调度器初始化失败，程序退出")
	}

	// 启动服务器
	srv := startServer(app.port)
	app.log.Info().Msgf("服务监听端口：%s", app.port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.log.Fatal().
				Err(err).
				Msgf("程序启动出错: %s", err)
		}
	}()

	app.log.Info().Msg("程序已启动完毕 🚀")
	<-ctx.Done()

	stop()
	app.log.Info().Msg("程序正在关闭中，如需立即结束请按 CTRL+C")

	// 优雅关闭
	shutdownServer(srv, app.rateLimiter, &app.log)

	app.log.Info().Msg("期待与你的再次相遇 ❤️")
}
