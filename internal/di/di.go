// Package di provides dependency injection functionality.
// Encapsulates all application dependency components, provides unified initialization and cleanup interfaces.
package di

import (
	// Standard library
	"context"
	"errors"
	"net/http"
	"time"

	// Third-party libraries
	"github.com/redis/go-redis/v9"
	health "github.com/soulteary/health-kit"
	loggerkit "github.com/soulteary/logger-kit"
	rediskitclient "github.com/soulteary/redis-kit/client"

	// Middleware kit
	middlewarekit "github.com/soulteary/middleware-kit"

	// Internal packages
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	internalerrors "github.com/soulteary/warden/internal/errors"
	"github.com/soulteary/warden/internal/middleware"
	"github.com/soulteary/warden/internal/router"
)

// Dependencies dependency container, encapsulates all application dependencies
type Dependencies struct {
	Config          *cmd.Config
	RedisClient     *redis.Client
	UserCache       *cache.SafeUserCache
	RedisUserCache  *cache.RedisUserCache
	RateLimiter     *middlewarekit.RateLimiter
	HTTPServer      *http.Server
	MainHandler     http.Handler
	HealthHandler   http.Handler
	LogLevelHandler http.Handler
}

// NewDependencies creates dependency container
// Initializes each component in dependency order
func NewDependencies(cfg *cmd.Config) (*Dependencies, error) {
	deps := &Dependencies{
		Config: cfg,
	}

	// 1. Initialize Redis client
	if err := deps.initRedis(); err != nil {
		return nil, internalerrors.ErrRedisConnection.WithError(err)
	}

	// 2. Initialize cache
	deps.initCache()

	// 3. Initialize rate limiter
	deps.initRateLimiter()

	// 4. Initialize HTTP handlers
	deps.initHandlers()

	// 5. Initialize HTTP server
	deps.initHTTPServer()

	return deps, nil
}

// initRedis initializes Redis client
func (d *Dependencies) initRedis() error {
	// If Redis is disabled, skip initialization
	if !d.Config.RedisEnabled {
		d.RedisClient = nil
		return nil
	}

	// Initialize Redis client using redis-kit
	redisCfg := rediskitclient.DefaultConfig().WithAddr(d.Config.Redis)
	if d.Config.RedisPassword != "" {
		redisCfg = redisCfg.WithPassword(d.Config.RedisPassword)
	}

	var err error
	d.RedisClient, err = rediskitclient.NewClient(redisCfg)
	if err != nil {
		return internalerrors.ErrRedisConnection.WithError(err)
	}

	return nil
}

// initCache initializes cache
func (d *Dependencies) initCache() {
	// Only create RedisUserCache if Redis client exists
	if d.RedisClient != nil {
		d.RedisUserCache = cache.NewRedisUserCache(d.RedisClient)
	} else {
		d.RedisUserCache = nil
	}
	d.UserCache = cache.NewSafeUserCache()
}

// initRateLimiter initializes rate limiter (using middleware-kit DefaultRateLimiterConfig + overrides)
func (d *Dependencies) initRateLimiter() {
	cfg := middlewarekit.DefaultRateLimiterConfig()
	cfg.Rate = define.DEFAULT_RATE_LIMIT
	cfg.Window = define.DEFAULT_RATE_LIMIT_WINDOW
	cfg.MaxVisitors = define.MAX_VISITORS_MAP_SIZE
	cfg.MaxWhitelist = define.MAX_WHITELIST_SIZE
	cfg.CleanupInterval = define.RATE_LIMIT_CLEANUP_INTERVAL
	d.RateLimiter = middlewarekit.NewRateLimiter(cfg)
}

// initHandlers initializes HTTP handlers
func (d *Dependencies) initHandlers() {
	// Create rate limiting middleware (using middleware-kit, skip health/metrics paths)
	rateLimitMiddleware := middlewarekit.RateLimitStd(middlewarekit.RateLimitConfig{
		Limiter:   d.RateLimiter,
		SkipPaths: define.SkipPathsHealthAndMetrics,
	})

	// Compress middleware (using middleware-kit)
	compressMiddleware := middlewarekit.CompressStd(middlewarekit.DefaultCompressConfig())

	// Body limit middleware (using middleware-kit DefaultBodyLimitConfig + override)
	bodyLimitCfg := middlewarekit.DefaultBodyLimitConfig()
	bodyLimitCfg.MaxSize = define.MAX_REQUEST_BODY_SIZE
	bodyLimitMiddleware := middlewarekit.BodyLimitStd(bodyLimitCfg)

	// Main data interface handler
	d.MainHandler = compressMiddleware(
		bodyLimitMiddleware(
			middleware.MetricsMiddleware(
				rateLimitMiddleware(
					router.ProcessWithLogger(router.JSON(d.UserCache)),
				),
			),
		),
	)

	// Health check handler (SecurityHeaders + NoCache aligned with main's health chain)
	healthAggregator := d.createHealthAggregator()
	securityHeaders := middlewarekit.SecurityHeadersStd(middlewarekit.StrictSecurityHeadersConfig())
	noCache := middlewarekit.NoCacheHeadersStd()
	d.HealthHandler = securityHeaders(
		noCache(
			middleware.MetricsMiddleware(health.Handler(healthAggregator)),
		),
	)

	// Log level control handler
	d.LogLevelHandler = middleware.MetricsMiddleware(
		router.ProcessWithLogger(loggerkit.LevelHandlerFunc(loggerkit.DefaultLevelHandlerConfig())),
	)
}

// initHTTPServer initializes HTTP server
func (d *Dependencies) initHTTPServer() {
	d.HTTPServer = &http.Server{
		Addr:              ":" + d.Config.Port,
		ReadHeaderTimeout: define.DEFAULT_TIMEOUT * time.Second,
		ReadTimeout:       define.DEFAULT_TIMEOUT * time.Second,
		WriteTimeout:      define.DEFAULT_TIMEOUT * time.Second,
		IdleTimeout:       define.IDLE_TIMEOUT,
		MaxHeaderBytes:    define.MAX_HEADER_BYTES,
	}
}

// createHealthAggregator creates a health check aggregator with all dependencies
func (d *Dependencies) createHealthAggregator() *health.Aggregator {
	isProduction := d.Config.Mode == "production" || d.Config.Mode == "prod"

	healthConfig := health.DefaultConfig().
		WithServiceName("warden").
		WithTimeout(5 * time.Second).
		WithDetails(!isProduction).
		WithChecks(!isProduction)

	aggregator := health.NewAggregator(healthConfig)

	// Redis health check
	switch {
	case !d.Config.RedisEnabled:
		aggregator.AddChecker(health.NewDisabledChecker("redis").
			WithMessage("Redis is disabled"))
	case d.RedisClient != nil:
		aggregator.AddChecker(health.NewRedisChecker(d.RedisClient))
	default:
		// Redis enabled but client is nil (connection failed)
		aggregator.AddChecker(health.NewCustomChecker("redis", func(_ context.Context) error {
			return errors.New("client not initialized")
		}))
	}

	// Data loading check
	aggregator.AddChecker(health.NewCustomChecker("data", func(_ context.Context) error {
		if d.UserCache == nil {
			return errors.New("cache not initialized")
		}
		if d.UserCache.Len() == 0 {
			return errors.New("no data loaded yet")
		}
		return nil
	}))

	return aggregator
}

// Cleanup cleans up resources
func (d *Dependencies) Cleanup() {
	if d.RateLimiter != nil {
		d.RateLimiter.Stop()
	}
	if d.RedisClient != nil {
		if err := rediskitclient.Close(d.RedisClient); err != nil {
			// Log error but don't affect cleanup process
			_ = err // Explicitly ignore error
		}
	}
}
