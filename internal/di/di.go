// Package di 提供了依赖注入功能。
// 封装应用的所有依赖组件，提供统一的初始化和清理接口。
package di

import (
	// 标准库
	"context"
	"net/http"
	"time"

	// 第三方库
	"github.com/redis/go-redis/v9"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/cache"
	"soulteary.com/soulteary/warden/internal/cmd"
	"soulteary.com/soulteary/warden/internal/define"
	"soulteary.com/soulteary/warden/internal/errors"
	"soulteary.com/soulteary/warden/internal/middleware"
	"soulteary.com/soulteary/warden/internal/router"
)

// Dependencies 依赖容器，封装所有应用依赖
type Dependencies struct {
	Config          *cmd.Config
	RedisClient     *redis.Client
	UserCache       *cache.SafeUserCache
	RedisUserCache  *cache.RedisUserCache
	RateLimiter     *middleware.RateLimiter
	HTTPServer      *http.Server
	MainHandler     http.Handler
	HealthHandler   http.Handler
	LogLevelHandler http.Handler
}

// NewDependencies 创建依赖容器
// 按照依赖关系顺序初始化各个组件
func NewDependencies(cfg *cmd.Config) (*Dependencies, error) {
	deps := &Dependencies{
		Config: cfg,
	}

	// 1. 初始化 Redis 客户端
	if err := deps.initRedis(); err != nil {
		return nil, errors.ErrRedisConnection.WithError(err)
	}

	// 2. 初始化缓存
	deps.initCache()

	// 3. 初始化速率限制器
	deps.initRateLimiter()

	// 4. 初始化 HTTP 处理器
	deps.initHandlers()

	// 5. 初始化 HTTP 服务器
	deps.initHTTPServer()

	return deps, nil
}

// initRedis 初始化 Redis 客户端
func (d *Dependencies) initRedis() error {
	redisOptions := &redis.Options{Addr: d.Config.Redis}
	if d.Config.RedisPassword != "" {
		redisOptions.Password = d.Config.RedisPassword
	}

	d.RedisClient = redis.NewClient(redisOptions)

	// 验证 Redis 连接（带超时）
	ctx, cancel := context.WithTimeout(context.Background(), define.RedisConnectionTimeout)
	defer cancel()
	if err := d.RedisClient.Ping(ctx).Err(); err != nil {
		return errors.ErrRedisConnection.WithError(err)
	}

	return nil
}

// initCache 初始化缓存
func (d *Dependencies) initCache() {
	d.RedisUserCache = cache.NewRedisUserCache(d.RedisClient)
	d.UserCache = cache.NewSafeUserCache()
}

// initRateLimiter 初始化速率限制器
func (d *Dependencies) initRateLimiter() {
	d.RateLimiter = middleware.NewRateLimiter(define.DefaultRateLimit, define.DefaultRateLimitWindow)
}

// initHandlers 初始化 HTTP 处理器
func (d *Dependencies) initHandlers() {
	// 创建速率限制中间件
	rateLimitMiddleware := middleware.RateLimitMiddlewareWithLimiter(d.RateLimiter)

	// 主数据接口处理器
	d.MainHandler = middleware.CompressMiddleware(
		middleware.BodyLimitMiddleware(
			middleware.MetricsMiddleware(
				rateLimitMiddleware(
					router.ProcessWithLogger(router.JSON(d.UserCache)),
				),
			),
		),
	)

	// 健康检查处理器
	d.HealthHandler = middleware.MetricsMiddleware(
		router.ProcessWithLogger(router.HealthCheck(d.RedisClient, d.UserCache, d.Config.Mode)),
	)

	// 日志级别控制处理器
	d.LogLevelHandler = middleware.MetricsMiddleware(
		router.ProcessWithLogger(router.LogLevelHandler()),
	)
}

// initHTTPServer 初始化 HTTP 服务器
func (d *Dependencies) initHTTPServer() {
	d.HTTPServer = &http.Server{
		Addr:              ":" + d.Config.Port,
		ReadHeaderTimeout: define.DEFAULT_TIMEOUT * time.Second,
		ReadTimeout:       define.DEFAULT_TIMEOUT * time.Second,
		WriteTimeout:      define.DEFAULT_TIMEOUT * time.Second,
		IdleTimeout:       define.IdleTimeout,
		MaxHeaderBytes:    define.MaxHeaderBytes,
	}
}

// Cleanup 清理资源
func (d *Dependencies) Cleanup() {
	if d.RateLimiter != nil {
		d.RateLimiter.Stop()
	}
	if d.RedisClient != nil {
		_ = d.RedisClient.Close()
	}
}
