// Package main is the entry point of the application.
// Provides HTTP server, cache management, scheduled task scheduling and other functionality.
package main

import (
	// Standard library
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	// Third-party libraries
	"github.com/redis/go-redis/v9"
	health "github.com/soulteary/health-kit"
	loggerkit "github.com/soulteary/logger-kit"
	rediskitclient "github.com/soulteary/redis-kit/client"
	secure "github.com/soulteary/secure-kit"

	// Middleware kit
	middlewarekit "github.com/soulteary/middleware-kit"

	// Internal packages
	"github.com/soulteary/tracing-kit"
	version "github.com/soulteary/version-kit"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/loader"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/metrics"
	"github.com/soulteary/warden/internal/middleware"
	"github.com/soulteary/warden/internal/router"
	internal_tracing "github.com/soulteary/warden/internal/tracing"
	"github.com/soulteary/warden/pkg/gocron"
)

const rulesFile = "./data.json"

// App application struct that encapsulates all application state
type App struct {
	userCache           *cache.SafeUserCache       // 8 bytes pointer
	redisUserCache      *cache.RedisUserCache      // 8 bytes pointer
	redisClient         *redis.Client              // 8 bytes pointer
	rateLimiter         *middlewarekit.RateLimiter // 8 bytes pointer
	rulesLoader         *loader.RulesLoader        // rules loader (parser-kit)
	log                 *loggerkit.Logger          // logger-kit instance
	port                string                     // 16 bytes
	configURL           string                     // 16 bytes
	authorizationHeader string                     // 16 bytes
	appMode             string                     // 16 bytes
	apiKey              string                     // 16 bytes
	taskInterval        uint64                     // 8 bytes
	redisEnabled        bool                       // 1 byte (padding to 8 bytes)
}

// NewApp creates a new application instance
func NewApp(cfg *cmd.Config) *App {
	app := &App{
		port:                cfg.Port,
		configURL:           cfg.RemoteConfig,
		authorizationHeader: cfg.RemoteKey,
		appMode:             cfg.Mode,
		// #nosec G115 -- conversion is safe, TaskInterval is positive
		taskInterval: uint64(cfg.TaskInterval),
		apiKey:       cfg.APIKey,
		redisEnabled: cfg.RedisEnabled,
		log:          logger.GetLoggerKit(),
	}

	if cfg.HTTPInsecureTLS {
		app.log.Warn().Msg(i18n.TWithLang(i18n.LangZH, "log.http_tls_disabled"))
		// In production environment, force TLS verification
		if cfg.Mode == "production" || cfg.Mode == "prod" {
			app.log.Fatal().Msg(i18n.TWithLang(i18n.LangZH, "log.prod_tls_required"))
		}
	}

	// Initialize cache (create memory cache first)
	app.userCache = cache.NewSafeUserCache()

	// Handle Redis initialization (optional)
	if cfg.RedisEnabled {
		// Initialize Redis client using redis-kit
		redisCfg := rediskitclient.DefaultConfig().WithAddr(cfg.Redis)
		if cfg.RedisPassword != "" {
			redisCfg = redisCfg.WithPassword(cfg.RedisPassword)
			// Security check: if password is passed via command line argument, log warning
			// Note: cannot directly determine password source here, but can infer from environment variable check
			if os.Getenv("REDIS_PASSWORD") == "" && os.Getenv("REDIS_PASSWORD_FILE") == "" {
				app.log.Warn().Msg(i18n.TWithLang(i18n.LangZH, "log.redis_password_warning"))
			}
		}

		var err error
		app.redisClient, err = rediskitclient.NewClient(redisCfg)
		if err != nil {
			// Redis connection failed, log warning and fallback to memory mode
			app.log.Warn().
				Err(err).
				Str("redis", cfg.Redis).
				Msg(i18n.TWithLang(i18n.LangZH, "log.redis_connection_failed_fallback"))
			app.redisClient = nil
			app.redisUserCache = nil
		} else {
			app.log.Info().Str("redis", cfg.Redis).Msg(i18n.TWithLang(i18n.LangZH, "log.redis_connected"))
			// Initialize Redis cache
			app.redisUserCache = cache.NewRedisUserCache(app.redisClient)
		}
	} else {
		// Redis is explicitly disabled
		app.log.Info().Msg(i18n.TWithLang(i18n.LangZH, "log.redis_disabled"))
		app.redisClient = nil
		app.redisUserCache = nil
	}

	// Rules loader (parser-kit, replaces internal parser)
	rulesLoader, err := loader.NewRulesLoader(cfg, app.appMode)
	if err != nil {
		app.log.Warn().Err(err).Msg(i18n.TWithLang(i18n.LangZH, "log.load_initial_data_failed"))
	} else {
		app.rulesLoader = rulesLoader
	}

	app.log.Debug().Str("mode", app.appMode).Msg(i18n.TWithLang(i18n.LangZH, "log.current_mode"))

	// Load initial data (multi-level fallback)
	if app.rulesLoader != nil {
		if err := app.loadInitialData(rulesFile); err != nil {
			app.log.Warn().Err(fmt.Errorf("加载初始数据失败: %w", err)).Msg(i18n.TWithLang(i18n.LangZH, "log.load_initial_data_failed"))
		}
	}

	// Initialize cache size metrics
	metrics.CacheSize.Set(float64(app.userCache.Len()))

	// Ensure task interval is not less than default value
	if app.taskInterval < define.DEFAULT_TASK_INTERVAL {
		app.taskInterval = uint64(define.DEFAULT_TASK_INTERVAL)
	}

	// Initialize rate limiter (using middleware-kit DefaultRateLimiterConfig + overrides)
	rateLimitCfg := middlewarekit.DefaultRateLimiterConfig()
	rateLimitCfg.Rate = define.DEFAULT_RATE_LIMIT
	rateLimitCfg.Window = define.DEFAULT_RATE_LIMIT_WINDOW
	rateLimitCfg.MaxVisitors = define.MAX_VISITORS_MAP_SIZE
	rateLimitCfg.MaxWhitelist = define.MAX_WHITELIST_SIZE
	rateLimitCfg.CleanupInterval = define.RATE_LIMIT_CLEANUP_INTERVAL
	app.rateLimiter = middlewarekit.NewRateLimiter(rateLimitCfg)

	return app
}

// loadInitialData loads data with multi-level fallback (Redis → parser-kit Load/FromFile).
func (app *App) loadInitialData(rulesFile string) error {
	ctx, cancel := context.WithTimeout(context.Background(), define.DEFAULT_LOAD_DATA_TIMEOUT)
	defer cancel()

	// ONLY_LOCAL mode: only use local file, no remote requests
	app.log.Debug().Str("appMode", app.appMode).Msg(i18n.TWithLang(i18n.LangZH, "log.check_mode"))
	if strings.ToUpper(strings.TrimSpace(app.appMode)) == "ONLY_LOCAL" {
		app.log.Debug().Msg(i18n.TWithLang(i18n.LangZH, "log.only_local_detected"))
		localUsers, err := app.rulesLoader.FromFile(ctx, rulesFile)
		if err == nil && len(localUsers) > 0 {
			app.log.Info().
				Int("count", len(localUsers)).
				Msg(i18n.TWithLang(i18n.LangZH, "log.loaded_from_local_file"))
			app.userCache.Set(localUsers)
			if app.redisUserCache != nil {
				if err := app.redisUserCache.Set(localUsers); err != nil {
					app.log.Warn().Err(err).Msg(i18n.TWithLang(i18n.LangZH, "log.redis_cache_update_failed"))
				}
			}
			return nil
		}
		_, statErr := os.Stat(rulesFile)
		if errors.Is(statErr, os.ErrNotExist) {
			app.log.Warn().
				Str("data_file", rulesFile).
				Str("example_file", "data.example.json").
				Msg(i18n.TWithLang(i18n.LangZH, "log.data_file_not_found"))
			app.log.Info().Msg(i18n.TWithLang(i18n.LangZH, "log.only_local_requires_file"))
			app.log.Info().Msgf(i18n.TWithLang(i18n.LangZH, "log.create_data_file"), rulesFile, "data.example.json")
		}
		app.log.Warn().Msg(i18n.TWithLang(i18n.LangZH, "log.only_local_load_failed"))
		return nil
	}

	// 1. Try to load from Redis cache (if Redis is available)
	if app.redisUserCache != nil {
		if cachedUsers, err := app.redisUserCache.Get(); err == nil && len(cachedUsers) > 0 {
			metrics.CacheHits.Inc()
			app.log.Info().
				Int("count", len(cachedUsers)).
				Msg(i18n.TWithLang(i18n.LangZH, "log.loaded_from_redis"))
			app.userCache.Set(cachedUsers)
			return nil
		}
		metrics.CacheMisses.Inc()
	}

	// 2. Try to load from parser-kit (remote + local by mode)
	users, err := app.rulesLoader.Load(ctx, rulesFile, app.configURL, app.authorizationHeader)
	if err == nil && len(users) > 0 {
		app.log.Info().
			Int("count", len(users)).
			Msg(i18n.TWithLang(i18n.LangZH, "log.loaded_from_remote_api"))
		app.userCache.Set(users)
		if app.redisUserCache != nil {
			if err := app.redisUserCache.Set(users); err != nil {
				app.log.Warn().Err(err).Msg(i18n.TWithLang(i18n.LangZH, "log.redis_cache_update_failed"))
			}
		}
		return nil
	}

	// 3. All failed: notify user
	_, localFileErr := os.Stat(rulesFile)
	hasRemoteConfig := app.configURL != "" && app.configURL != define.DEFAULT_REMOTE_CONFIG
	if errors.Is(localFileErr, os.ErrNotExist) && !hasRemoteConfig {
		// Local file does not exist and no remote address configured, provide friendly prompt
		app.log.Warn().
			Str("data_file", rulesFile).
			Str("example_file", "data.example.json").
			Msg(i18n.TWithLang(i18n.LangZH, "log.data_file_not_found_no_remote"))
		app.log.Info().
			Msg(i18n.TWithLang(i18n.LangZH, "log.tip_actions"))
		app.log.Info().
			Msgf(i18n.TWithLang(i18n.LangZH, "log.create_data_file_or_config"), rulesFile, "data.example.json")
		app.log.Info().
			Msg(i18n.TWithLang(i18n.LangZH, "log.config_remote_param"))
		app.log.Info().
			Msg(i18n.TWithLang(i18n.LangZH, "log.config_remote_env"))
		app.log.Warn().Msg(i18n.TWithLang(i18n.LangZH, "log.using_empty_data"))
	} else {
		app.log.Warn().Msg(i18n.TWithLang(i18n.LangZH, "log.all_sources_failed"))
	}
	return nil
}

// hasChanged compares if data has changed (optimized using cached hash value)
//
// This function determines if data has changed by comparing cached hash values, used to optimize cache update strategy.
// Prioritizes using cached hash values to avoid redundant calculations.
//
// Parameters:
//   - oldHash: cached hash value of old data
//   - newUsers: new user list
//
// Returns:
//   - bool: true means data has changed, false means data unchanged
//
// Notes:
//   - This function prioritizes using cached hash values to avoid redundant calculations
//   - If cached hash value is provided, performance can be significantly improved
func hasChanged(oldHash string, newUsers []define.AllowListUser) bool {
	// Calculate hash value of new data
	newHash := calculateHash(newUsers)
	return oldHash != newHash
}

// calculateHash calculates SHA256 hash value of user list
//
// This function is used to detect if user data has changed by calculating hash values to compare data content.
// Implementation details:
// - Sorts data (by Phone and Mail) to ensure same data produces same hash
// - Uses SHA256 algorithm to calculate hash value
// - Returns fixed hash value for empty data to optimize performance
// - Includes all fields (Phone, Mail, UserID, Status, Scope, Role) to ensure accurate data change detection
//
// Parameters:
//   - users: user list to calculate hash for
//
// Returns:
//   - string: hexadecimal encoded SHA256 hash value
//
// Side effects:
//   - Creates a copy of input data for sorting, does not modify original data
//   - For large datasets, sorting operation may have performance overhead
//
// Optimizations:
//   - Empty data directly returns fixed hash to avoid unnecessary calculations
//   - Uses data copy for sorting to keep original data unchanged
func calculateHash(users []define.AllowListUser) string {
	// Optimization: empty data directly returns fixed hash
	if len(users) == 0 {
		return secure.GetSHA256Hash("empty")
	}

	// Sort first to ensure same data produces same hash
	// Optimization: if data volume is large, can consider in-place sorting, but uses copy to keep data unchanged
	sorted := make([]define.AllowListUser, len(users))
	copy(sorted, users)
	// Normalize user data to ensure consistency (generate user_id, set default values, etc.)
	for i := range sorted {
		sorted[i].Normalize()
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Phone != sorted[j].Phone {
			return sorted[i].Phone < sorted[j].Phone
		}
		return sorted[i].Mail < sorted[j].Mail
	})

	// Calculate hash (includes all fields to ensure accurate data change detection, consistent with cache.calculateHashInternal)
	var sb strings.Builder
	for _, user := range sorted {
		scopeStr := strings.Join(user.Scope, ",")
		sb.WriteString(user.Phone + ":" + user.Mail + ":" + user.UserID + ":" + user.Status + ":" + scopeStr + ":" + user.Role + "\n")
	}
	return secure.GetSHA256Hash(sb.String())
}

// checkDataChanged checks if data has changed
//
// This function determines if data has changed by comparing cached hash values and length.
// Prioritizes using cached hash values to avoid redundant calculations.
//
// Parameters:
//   - newUsers: new user list
//
// Returns:
//   - bool: true means data has changed, false means data unchanged
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

// updateRedisCacheWithRetry updates Redis cache with retry mechanism
//
// This function implements Redis cache update logic with retry, up to define.REDIS_RETRY_MAX_RETRIES times.
// Delay time increases with each retry.
//
// Parameters:
//   - users: user list to update
//
// Returns:
//   - error: returns error on update failure, nil on success
func (app *App) updateRedisCacheWithRetry(users []define.AllowListUser) error {
	// If Redis cache is unavailable, return error directly
	if app.redisUserCache == nil {
		return fmt.Errorf("redis cache unavailable")
	}

	var lastErr error
	for attempt := 0; attempt < define.REDIS_RETRY_MAX_RETRIES; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * define.REDIS_RETRY_DELAY)
			app.log.Debug().
				Int("attempt", attempt+1).
				Msg(i18n.TWithLang(i18n.LangZH, "log.retry_redis_cache"))
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
					Msg(i18n.TWithLang(i18n.LangZH, "log.redis_cache_updated"))
			}
			return nil
		}
	}

	return fmt.Errorf("failed to update Redis cache (retried %d times): %w", define.REDIS_RETRY_MAX_RETRIES, lastErr)
}

// backgroundTask is a background task that periodically updates cache data
//
// This function implements intelligent cache update strategy with the following features:
// - Data change detection: avoids unnecessary updates through hash comparison
// - Optimistic locking strategy: uses optimistic locking to ensure data consistency
// - Error recovery: includes panic recovery mechanism to prevent task crashes from affecting main program
// - Retry mechanism: automatically retries on Redis update failure
// - Metrics collection: records task execution time, error count and other metrics
//
// Parameters:
//   - rulesFile: local rules file path, as one of the data sources
//
// Side effects:
//   - Updates memory cache (app.userCache)
//   - Updates Redis cache (app.redisUserCache)
//   - Updates Prometheus metrics (metrics.BackgroundTaskTotal, metrics.BackgroundTaskDuration, etc.)
//   - Records logs (debug, info, warning levels)
//
// Error handling:
//   - If panic occurs, will catch and record error without affecting main program execution
//   - Redis update failure will retry, on final failure will log warning but continue using memory cache
//
// Performance optimizations:
//   - Performs data comparison outside lock to reduce lock holding time
//   - Uses hash values to quickly detect data changes
//   - Returns directly when data unchanged, skipping update operations
func (app *App) backgroundTask(rulesFile string) {
	// Add error recovery mechanism to prevent panic from crashing entire program
	defer func() {
		if r := recover(); r != nil {
			metrics.BackgroundTaskErrors.Inc()
			app.log.Error().
				Interface("panic", r).
				Msg(i18n.TWithLang(i18n.LangZH, "log.background_task_panic"))
		}
	}()

	start := time.Now()
	var newUsers []define.AllowListUser

	if app.rulesLoader == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(define.DEFAULT_TIMEOUT*2)*time.Second)
	defer cancel()
	var err error
	if strings.ToUpper(strings.TrimSpace(app.appMode)) == "ONLY_LOCAL" {
		newUsers, err = app.rulesLoader.FromFile(ctx, rulesFile)
	} else {
		newUsers, err = app.rulesLoader.Load(ctx, rulesFile, app.configURL, app.authorizationHeader)
	}
	if err != nil {
		app.log.Warn().Err(err).Msg(i18n.TWithLang(i18n.LangZH, "log.background_load_failed"))
		return
	}

	// Check if data has changed
	if !app.checkDataChanged(newUsers) {
		app.log.Debug().Msg(i18n.TWithLang(i18n.LangZH, "log.data_unchanged"))
		return
	}

	// Update memory cache
	app.userCache.Set(newUsers)

	// Verify data consistency (optimistic locking strategy)
	currentHash := app.userCache.GetHash()
	newHash := calculateHash(newUsers)
	if currentHash != "" && currentHash == newHash {
		// Data consistent, update Redis cache (if Redis is available)
		if app.redisUserCache != nil {
			if err := app.updateRedisCacheWithRetry(newUsers); err != nil {
				app.log.Warn().
					Err(err).
					Msg(i18n.TWithLang(i18n.LangZH, "log.redis_cache_failed_continue"))
				metrics.BackgroundTaskErrors.Inc()
			}
		}
	} else {
		currentLen := app.userCache.Len()
		app.log.Debug().
			Int("expected_count", len(newUsers)).
			Int("actual_count", currentLen).
			Msg(i18n.TWithLang(i18n.LangZH, "log.data_modified_during_update"))
	}

	// Update metrics
	duration := time.Since(start).Seconds()
	metrics.BackgroundTaskTotal.Inc()
	metrics.BackgroundTaskDuration.Observe(duration)
	metrics.CacheSize.Set(float64(app.userCache.Len()))

	app.log.Info().
		Int("count", len(newUsers)).
		Float64("duration", duration).
		Msg(i18n.TWithLang(i18n.LangZH, "log.background_update"))
}

// registerRoutes registers all HTTP routes
func registerRoutes(app *App) {
	// Create trusted proxy config (from TRUSTED_PROXY_IPS env, parsed via define.ParseTrustedProxyIPs)
	trustedProxies := define.ParseTrustedProxyIPs(os.Getenv("TRUSTED_PROXY_IPS"))
	trustedProxyConfig := middlewarekit.NewTrustedProxyConfig(trustedProxies)

	// Create base middleware (using middleware-kit)
	i18nMiddleware := middleware.I18nMiddleware()
	errorHandlerMiddleware := middleware.ErrorHandlerMiddleware(app.appMode)

	// Security headers middleware (using middleware-kit with strict config)
	securityCfg := middlewarekit.StrictSecurityHeadersConfig()
	securityHeadersMiddleware := middlewarekit.SecurityHeadersStd(securityCfg)

	// Rate limit middleware (using middleware-kit, skip health/metrics paths)
	rateLimitMiddleware := middlewarekit.RateLimitStd(middlewarekit.RateLimitConfig{
		Limiter:            app.rateLimiter,
		TrustedProxyConfig: trustedProxyConfig,
		Logger:             logger.ZerologPtr(),
		SkipPaths:          define.SkipPathsHealthAndMetrics,
		OnLimitReached: func(key string) {
			metrics.RateLimitHits.WithLabelValues(key).Inc()
		},
	})

	// Auth middleware (using middleware-kit DefaultAPIKeyConfig + overrides)
	authBaseCfg := middlewarekit.DefaultAPIKeyConfig()
	authBaseCfg.APIKey = app.apiKey
	authBaseCfg.AuthScheme = "Bearer"
	authBaseCfg.TrustedProxyConfig = trustedProxyConfig
	authBaseCfg.Logger = logger.ZerologPtr()
	authMiddleware := middlewarekit.APIKeyAuthStd(authBaseCfg)

	// Optional auth for metrics: same as base but allow empty key when no API key configured
	optionalAuthCfg := authBaseCfg
	optionalAuthCfg.AllowEmptyKey = true
	optionalAuthMiddleware := middlewarekit.APIKeyAuthStd(optionalAuthCfg)

	// Compress middleware (using middleware-kit)
	compressMiddleware := middlewarekit.CompressStd(middlewarekit.DefaultCompressConfig())

	// Body limit middleware (using middleware-kit DefaultBodyLimitConfig + overrides)
	bodyLimitCfg := middlewarekit.DefaultBodyLimitConfig()
	bodyLimitCfg.MaxSize = define.MAX_REQUEST_BODY_SIZE
	bodyLimitCfg.TrustedProxyConfig = trustedProxyConfig
	bodyLimitCfg.Logger = logger.ZerologPtr()
	bodyLimitMiddleware := middlewarekit.BodyLimitStd(bodyLimitCfg)

	// Tracing middleware (if enabled)
	var tracingMiddleware func(http.Handler) http.Handler
	if tracing.IsEnabled() {
		tracingMiddleware = internal_tracing.Middleware
	}

	// Health check endpoint IP whitelist (read from environment variable)
	healthWhitelist := os.Getenv("HEALTH_CHECK_IP_WHITELIST")

	// Register Prometheus metrics endpoint (optional authentication)
	// i18n middleware placed at outermost layer to ensure all requests can detect language
	metricsHandler := i18nMiddleware(
		router.AccessLogMiddleware()(
			securityHeadersMiddleware(
				errorHandlerMiddleware(
					wrapWithTracingIfEnabled(tracingMiddleware,
						optionalAuthMiddleware(
							middleware.MetricsMiddleware(metrics.Handler()),
						),
					),
				),
			),
		),
	)
	http.Handle(define.PATH_METRICS, metricsHandler)

	// Register main data interface (requires authentication)
	// i18n middleware placed at outermost layer to ensure all requests can detect language
	mainHandler := i18nMiddleware(
		router.AccessLogMiddleware()(
			securityHeadersMiddleware(
				errorHandlerMiddleware(
					wrapWithTracingIfEnabled(tracingMiddleware,
						compressMiddleware(
							bodyLimitMiddleware(
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
				),
			),
		),
	)
	http.Handle("/", mainHandler)

	// Register user query interface (requires authentication)
	// i18n middleware placed at outermost layer to ensure all requests can detect language
	userHandler := i18nMiddleware(
		router.AccessLogMiddleware()(
			securityHeadersMiddleware(
				errorHandlerMiddleware(
					wrapWithTracingIfEnabled(tracingMiddleware,
						compressMiddleware(
							bodyLimitMiddleware(
								middleware.MetricsMiddleware(
									rateLimitMiddleware(
										authMiddleware(
											router.ProcessWithLogger(router.GetUserByIdentifier(app.userCache)),
										),
									),
								),
							),
						),
					),
				),
			),
		),
	)
	http.Handle("/user", userHandler)

	// Register health check endpoint (IP whitelist protection, limits information leakage)
	// Setup health checker using health-kit
	healthAggregator := setupHealthChecker(app.redisClient, app.userCache, app.appMode, app.redisEnabled, healthWhitelist)
	// i18n middleware placed at outermost layer to ensure all requests can detect language
	healthHandler := i18nMiddleware(
		router.AccessLogMiddleware()(
			securityHeadersMiddleware(
				errorHandlerMiddleware(
					wrapWithTracingIfEnabled(tracingMiddleware,
						middleware.MetricsMiddleware(
							middlewarekit.NoCacheHeadersStd()(health.Handler(healthAggregator)),
						),
					),
				),
			),
		),
	)
	http.Handle(define.PATH_HEALTH, healthHandler)
	http.Handle(define.PATH_HEALTHCHECK, healthHandler)

	// Register log level control endpoint using logger-kit (requires authentication)
	// i18n middleware placed at outermost layer to ensure all requests can detect language
	lkLog := logger.GetLoggerKit()
	logLevelHandler := i18nMiddleware(
		router.AccessLogMiddleware()(
			securityHeadersMiddleware(
				errorHandlerMiddleware(
					wrapWithTracingIfEnabled(tracingMiddleware,
						middleware.MetricsMiddleware(
							authMiddleware(
								loggerkit.LevelHandler(loggerkit.LevelHandlerConfig{
									Logger: lkLog,
								}),
							),
						),
					),
				),
			),
		),
	)
	http.Handle("/log/level", logLevelHandler)
}

// setupHealthChecker creates a health check aggregator with all dependencies
func setupHealthChecker(redisClient *redis.Client, userCache *cache.SafeUserCache, appMode string, redisEnabled bool, ipWhitelist string) *health.Aggregator {
	isProduction := appMode == "production" || appMode == "prod"
	isOnlyLocalMode := strings.ToUpper(strings.TrimSpace(appMode)) == "ONLY_LOCAL"

	// Parse IP whitelist
	var ipList []string
	if ipWhitelist != "" {
		for _, ip := range strings.Split(ipWhitelist, ",") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				ipList = append(ipList, ip)
			}
		}
	}

	healthConfig := health.DefaultConfig().
		WithServiceName("warden").
		WithTimeout(5 * time.Second).
		WithIPWhitelist(ipList).
		WithDetails(!isProduction).
		WithChecks(!isProduction)

	aggregator := health.NewAggregator(healthConfig)

	// Redis health check
	switch {
	case !redisEnabled:
		aggregator.AddChecker(health.NewDisabledChecker("redis").
			WithMessage("Redis is disabled"))
	case redisClient != nil:
		aggregator.AddChecker(health.NewRedisChecker(redisClient))
	default:
		// Redis enabled but client is nil (connection failed)
		aggregator.AddChecker(health.NewCustomChecker("redis", func(_ context.Context) error {
			return errors.New("client not initialized")
		}))
	}

	// Data loading check
	aggregator.AddChecker(health.NewCustomChecker("data", func(_ context.Context) error {
		if userCache == nil {
			return errors.New("cache not initialized")
		}
		if userCache.Len() == 0 {
			// In ONLY_LOCAL mode, empty data is acceptable warning
			if isOnlyLocalMode {
				return nil // Return ok for ONLY_LOCAL mode
			}
			return errors.New("no data loaded yet")
		}
		return nil
	}))

	return aggregator
}

// wrapWithTracingIfEnabled wraps handler with tracing middleware if enabled
func wrapWithTracingIfEnabled(tracingMiddleware func(http.Handler) http.Handler, handler http.Handler) http.Handler {
	if tracingMiddleware != nil {
		return tracingMiddleware(handler)
	}
	return handler
}

// startServer starts HTTP server
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

// shutdownServer gracefully shuts down the server
func shutdownServer(srv *http.Server, rateLimiter *middlewarekit.RateLimiter, log *loggerkit.Logger) {
	// Stop rate limiter
	if rateLimiter != nil {
		rateLimiter.Stop()
	}

	// Gracefully shutdown HTTP server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), define.SHUTDOWN_TIMEOUT)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Info().Err(fmt.Errorf("程序强制关闭: %w", err)).Msg(i18n.TWithLang(i18n.LangZH, "log.forced_shutdown"))
	}
}

func main() {
	log := logger.GetLoggerKit()

	// Parse configuration
	cfg := cmd.GetArgs()

	// Validate configuration
	if err := cmd.ValidateConfig(cfg); err != nil {
		log.Fatal().
			Err(err).
			Msg(i18n.TWithLang(i18n.LangZH, "log.config_validation_failed_exit"))
	}

	// Load config from file if config file is specified (for tracing config)
	var tracingCfg *config.Config
	if cfgFile := os.Getenv("CONFIG_FILE"); cfgFile != "" {
		if loadedCfg, err := config.LoadFromFile(cfgFile); err == nil {
			tracingCfg = loadedCfg
		}
	}

	// Initialize OpenTelemetry tracing if enabled
	var tracerProvider interface{ Shutdown(context.Context) error }
	if tracingCfg != nil && tracingCfg.Tracing.Enabled && tracingCfg.Tracing.Endpoint != "" {
		tp, err := tracing.InitTracer(
			"warden",
			version.Version,
			tracingCfg.Tracing.Endpoint,
		)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize OpenTelemetry tracing")
		} else {
			tracerProvider = tp
			log.Info().Msg("OpenTelemetry tracing initialized")
		}
	} else if otlpEnabled := os.Getenv("OTLP_ENABLED"); otlpEnabled != "" && (otlpEnabled == "true" || otlpEnabled == "1") {
		otlpEndpoint := os.Getenv("OTLP_ENDPOINT")
		if otlpEndpoint != "" {
			tp, err := tracing.InitTracer(
				"warden",
				version.Version,
				otlpEndpoint,
			)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to initialize OpenTelemetry tracing")
			} else {
				tracerProvider = tp
				log.Info().Msg("OpenTelemetry tracing initialized")
			}
		}
	}

	// Initialize application
	app := NewApp(cfg)

	// Register routes
	registerRoutes(app)

	// Set up signal handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		stop()
		// Shutdown tracer if initialized
		if tracerProvider != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
				log.Warn().Err(err).Msg("Failed to shutdown tracer")
			}
		}
	}()

	app.log.Info().Msgf(i18n.TWithLang(i18n.LangZH, "log.app_version"), version.Version, version.BuildDate, version.Commit)

	// Start scheduled task scheduler
	// Select lock implementation based on Redis availability
	gocron.SetLocker(&cache.Locker{Cache: app.redisClient})
	scheduler := gocron.NewScheduler()
	schedulerStopped := scheduler.Start()
	defer func() {
		close(schedulerStopped)
		scheduler.Clear()
		app.log.Info().Msg(i18n.TWithLang(i18n.LangZH, "log.scheduler_closed"))
	}()
	if err := scheduler.Every(app.taskInterval).Seconds().Lock().Do(app.backgroundTask, rulesFile); err != nil {
		// Clean up resources before exiting (defer executes on function return, but log.Fatal exits immediately)
		// So need to manually clean up
		close(schedulerStopped)
		scheduler.Clear()
		stop()
		//nolint:gocritic // exitAfterDefer: need to exit immediately on error, resources manually cleaned up
		log.Fatal().
			Err(err).
			Msg(i18n.TWithLang(i18n.LangZH, "log.scheduler_init_failed"))
	}

	// Start server
	srv := startServer(app.port)
	app.log.Info().Msgf(i18n.TWithLang(i18n.LangZH, "log.service_listening"), app.port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.log.Fatal().
				Err(err).
				Msgf(i18n.TWithLang(i18n.LangZH, "log.startup_error"), err)
		}
	}()

	app.log.Info().Msg(i18n.TWithLang(i18n.LangZH, "log.app_started"))
	<-ctx.Done()

	stop()
	app.log.Info().Msg(i18n.TWithLang(i18n.LangZH, "log.shutting_down"))

	// Graceful shutdown
	shutdownServer(srv, app.rateLimiter, app.log)

	app.log.Info().Msg(i18n.TWithLang(i18n.LangZH, "log.goodbye"))
}
