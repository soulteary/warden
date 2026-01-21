// Package di provides dependency injection functionality.
// Encapsulates all application dependency components, provides unified initialization and cleanup interfaces.
package di

import (
	// Standard library
	"net/http"
	"time"

	// Third-party libraries
	"github.com/redis/go-redis/v9"
	rediskitclient "github.com/soulteary/redis-kit/client"

	// Internal packages
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/errors"
	"github.com/soulteary/warden/internal/middleware"
	"github.com/soulteary/warden/internal/router"
)

// Dependencies dependency container, encapsulates all application dependencies
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

// NewDependencies creates dependency container
// Initializes each component in dependency order
func NewDependencies(cfg *cmd.Config) (*Dependencies, error) {
	deps := &Dependencies{
		Config: cfg,
	}

	// 1. Initialize Redis client
	if err := deps.initRedis(); err != nil {
		return nil, errors.ErrRedisConnection.WithError(err)
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
		return errors.ErrRedisConnection.WithError(err)
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

// initRateLimiter initializes rate limiter
func (d *Dependencies) initRateLimiter() {
	d.RateLimiter = middleware.NewRateLimiter(define.DEFAULT_RATE_LIMIT, define.DEFAULT_RATE_LIMIT_WINDOW)
}

// initHandlers initializes HTTP handlers
func (d *Dependencies) initHandlers() {
	// Create rate limiting middleware
	rateLimitMiddleware := middleware.RateLimitMiddlewareWithLimiter(d.RateLimiter)

	// Main data interface handler
	d.MainHandler = middleware.CompressMiddleware(
		middleware.BodyLimitMiddleware(
			middleware.MetricsMiddleware(
				rateLimitMiddleware(
					router.ProcessWithLogger(router.JSON(d.UserCache)),
				),
			),
		),
	)

	// Health check handler
	d.HealthHandler = middleware.MetricsMiddleware(
		router.ProcessWithLogger(router.HealthCheck(d.RedisClient, d.UserCache, d.Config.Mode, d.Config.RedisEnabled)),
	)

	// Log level control handler
	d.LogLevelHandler = middleware.MetricsMiddleware(
		router.ProcessWithLogger(router.LogLevelHandler()),
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
