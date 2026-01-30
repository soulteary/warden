// Package main - route registration and health check setup.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	health "github.com/soulteary/health-kit"
	loggerkit "github.com/soulteary/logger-kit"
	middlewarekit "github.com/soulteary/middleware-kit"
	tracing "github.com/soulteary/tracing-kit"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/metrics"
	"github.com/soulteary/warden/internal/middleware"
	"github.com/soulteary/warden/internal/router"
	internal_tracing "github.com/soulteary/warden/internal/tracing"
)

// registerRoutes registers all HTTP routes
func registerRoutes(app *App) {
	trustedProxies := define.ParseTrustedProxyIPs(os.Getenv("TRUSTED_PROXY_IPS"))
	trustedProxyConfig := middlewarekit.NewTrustedProxyConfig(trustedProxies)

	i18nMiddleware := middleware.I18nMiddleware()
	errorHandlerMiddleware := middleware.ErrorHandlerMiddleware(app.appMode)
	securityCfg := middlewarekit.StrictSecurityHeadersConfig()
	securityHeadersMiddleware := middlewarekit.SecurityHeadersStd(securityCfg)
	rateLimitMiddleware := middlewarekit.RateLimitStd(middlewarekit.RateLimitConfig{
		Limiter:            app.rateLimiter,
		TrustedProxyConfig: trustedProxyConfig,
		Logger:             logger.ZerologPtr(),
		SkipPaths:          define.SkipPathsHealthAndMetrics,
		OnLimitReached: func(key string) {
			metrics.RateLimitHits.WithLabelValues(key).Inc()
		},
	})

	authBaseCfg := middlewarekit.DefaultAPIKeyConfig()
	authBaseCfg.APIKey = app.apiKey
	authBaseCfg.AuthScheme = "Bearer"
	authBaseCfg.TrustedProxyConfig = trustedProxyConfig
	authBaseCfg.Logger = logger.ZerologPtr()
	authMiddleware := middlewarekit.APIKeyAuthStd(authBaseCfg)
	optionalAuthCfg := authBaseCfg
	optionalAuthCfg.AllowEmptyKey = true
	optionalAuthMiddleware := middlewarekit.APIKeyAuthStd(optionalAuthCfg)

	compressMiddleware := middlewarekit.CompressStd(middlewarekit.DefaultCompressConfig())
	bodyLimitCfg := middlewarekit.DefaultBodyLimitConfig()
	bodyLimitCfg.MaxSize = define.MAX_REQUEST_BODY_SIZE
	bodyLimitCfg.TrustedProxyConfig = trustedProxyConfig
	bodyLimitCfg.Logger = logger.ZerologPtr()
	bodyLimitMiddleware := middlewarekit.BodyLimitStd(bodyLimitCfg)

	var tracingMiddleware func(http.Handler) http.Handler
	if tracing.IsEnabled() {
		tracingMiddleware = internal_tracing.Middleware
	}

	healthWhitelist := os.Getenv("HEALTH_CHECK_IP_WHITELIST")

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

	lookupHandler := i18nMiddleware(
		router.AccessLogMiddleware()(
			securityHeadersMiddleware(
				errorHandlerMiddleware(
					wrapWithTracingIfEnabled(tracingMiddleware,
						compressMiddleware(
							bodyLimitMiddleware(
								middleware.MetricsMiddleware(
									rateLimitMiddleware(
										authMiddleware(
											router.ProcessWithLogger(router.GetLookup(app.userCache)),
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
	http.Handle("/v1/lookup", lookupHandler)

	healthAggregator := setupHealthChecker(app.redisClient, app.userCache, app.appMode, app.redisEnabled, healthWhitelist)
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

	http.Handle("/v1/users", mainHandler)
	http.Handle("/v1/user", userHandler)
	http.Handle("/v1/health", healthHandler)
	http.Handle("/v1/healthcheck", healthHandler)

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

	switch {
	case !redisEnabled:
		aggregator.AddChecker(health.NewDisabledChecker("redis").
			WithMessage("Redis is disabled"))
	case redisClient != nil:
		aggregator.AddChecker(health.NewRedisChecker(redisClient))
	default:
		aggregator.AddChecker(health.NewCustomChecker("redis", func(_ context.Context) error {
			return errors.New("client not initialized")
		}))
	}

	aggregator.AddChecker(health.NewCustomChecker("data", func(_ context.Context) error {
		if userCache == nil {
			return errors.New("cache not initialized")
		}
		if userCache.Len() == 0 {
			if isOnlyLocalMode {
				return nil
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
