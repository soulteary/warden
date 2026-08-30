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
	health "github.com/soulteary/health-kit/v2"
	loggerkit "github.com/soulteary/logger-kit/v2"
	middlewarekit "github.com/soulteary/middleware-kit/v2"
	tracing "github.com/soulteary/tracing-kit"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/middleware"
	"github.com/soulteary/warden/internal/prommetrics"
	"github.com/soulteary/warden/internal/router"
	internal_tracing "github.com/soulteary/warden/internal/tracing"
)

// registerRoutes registers all HTTP routes
func registerRoutes(app *App) {
	trustedProxies := define.ParseTrustedProxyIPs(os.Getenv("TRUSTED_PROXY_IPS"))
	trustedProxyConfig := middlewarekit.NewTrustedProxyConfig(trustedProxies)

	i18nMiddleware := middleware.I18nMiddleware()
	errorHandlerMiddleware := middleware.ErrorHandlerMiddleware(app.environment)
	securityCfg := middlewarekit.StrictSecurityHeadersConfig()
	securityHeadersMiddleware := middlewarekit.SecurityHeadersStd(securityCfg)
	rateLimitMiddleware := middlewarekit.RateLimitStd(middlewarekit.RateLimitConfig{
		Limiter:            app.rateLimiter,
		TrustedProxyConfig: trustedProxyConfig,
		Logger:             logger.ZerologPtr(),
		SkipPaths:          define.SkipPathsHealthAndMetrics,
		OnLimitReached: func(key string) {
			prommetrics.RecordRateLimitHit(key)
		},
	})

	authBaseCfg := middlewarekit.DefaultAPIKeyConfig()
	authBaseCfg.APIKey = app.apiKey
	authBaseCfg.AuthScheme = "Bearer"
	authBaseCfg.TrustedProxyConfig = trustedProxyConfig
	authBaseCfg.Logger = logger.ZerologPtr()
	apiKeyMiddleware := middlewarekit.APIKeyAuthStd(authBaseCfg)
	// Service auth chain: mTLS (client cert) > HMAC > API Key.
	// Accept only authenticated v2 signatures by default. Operators can explicitly
	// enable legacy v1 during a bounded migration; usage is surfaced via a metric.
	hmacCfg := middleware.HMACConfig{
		Keys:                  app.hmacKeys,
		TimestampToleranceSec: app.hmacToleranceSec,
		AllowV1:               &app.hmacAllowV1,
		OnReplayRejected:      prommetrics.RecordHMACReplayRejected,
		OnV1Used: func() {
			prommetrics.RecordDeprecation("hmac_v1")
		},
	}
	if app.redisClient != nil {
		hmacCfg.ReplayGuard = middleware.NewRedisReplayGuard(app.redisClient)
	}
	authMiddleware := middleware.ServiceAuthChain(hmacCfg, apiKeyMiddleware)
	optionalAuthCfg := authBaseCfg
	optionalAuthCfg.AllowEmptyKey = true
	optionalAuthMiddleware := middlewarekit.APIKeyAuthStd(optionalAuthCfg)

	compressMiddleware := middlewarekit.CompressStd(middlewarekit.DefaultCompressConfig())
	bodyLimitCfg := middlewarekit.DefaultBodyLimitConfig()
	bodyLimitCfg.MaxSize = define.MAX_REQUEST_BODY_SIZE
	bodyLimitCfg.TrustedProxyConfig = trustedProxyConfig
	bodyLimitCfg.Logger = logger.ZerologPtr()
	bodyLimitMiddleware := middlewarekit.BodyLimitStd(bodyLimitCfg)
	ipAllowlistMiddleware := middleware.IPWhitelistMiddleware("")

	var tracingMiddleware func(http.Handler) http.Handler
	if tracing.IsEnabled() {
		tracingMiddleware = internal_tracing.Middleware
	}

	healthWhitelist := os.Getenv("HEALTH_CHECK_IP_WHITELIST")

	// Metrics exposure is configurable. By default metrics are exposed anonymously
	// (optionalAuthMiddleware) for scrape simplicity and expose only low-cardinality,
	// non-sensitive series. Operators can require authentication for /metrics by setting
	// WARDEN_METRICS_REQUIRE_AUTH=true, in which case the full service auth chain applies.
	metricsAuth := optionalAuthMiddleware
	if requireAuthEnv := strings.TrimSpace(os.Getenv("WARDEN_METRICS_REQUIRE_AUTH")); requireAuthEnv == "true" || requireAuthEnv == "1" {
		metricsAuth = authMiddleware
	}
	metricsHandler := i18nMiddleware(
		router.AccessLogMiddleware()(
			securityHeadersMiddleware(
				errorHandlerMiddleware(
					wrapWithTracingIfEnabled(tracingMiddleware,
						ipAllowlistMiddleware(metricsAuth(
							middleware.MetricsMiddleware(prommetrics.Handler()),
						)),
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
						ipAllowlistMiddleware(compressMiddleware(
							bodyLimitMiddleware(
								middleware.MetricsMiddleware(
									rateLimitMiddleware(
										authMiddleware(
											router.ProcessWithLogger(router.JSON(app.userCache, app.responseFields)),
										),
									),
								),
							),
						)),
					),
				),
			),
		),
	)
	http.Handle("/", mainHandler)
	http.Handle(define.PATH_DATA_JSON, mainHandler) // 与 GET / 行为一致，便于作为 data.json API 消费

	userHandler := i18nMiddleware(
		router.AccessLogMiddleware()(
			securityHeadersMiddleware(
				errorHandlerMiddleware(
					wrapWithTracingIfEnabled(tracingMiddleware,
						ipAllowlistMiddleware(compressMiddleware(
							bodyLimitMiddleware(
								middleware.MetricsMiddleware(
									rateLimitMiddleware(
										authMiddleware(
											router.ProcessWithLogger(router.GetUserByIdentifier(app.userCache, app.responseFields)),
										),
									),
								),
							),
						)),
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
						ipAllowlistMiddleware(compressMiddleware(
							bodyLimitMiddleware(
								middleware.MetricsMiddleware(
									rateLimitMiddleware(
										authMiddleware(
											router.ProcessWithLogger(router.GetLookup(app.userCache)),
										),
									),
								),
							),
						)),
					),
				),
			),
		),
	)
	http.Handle("/v1/lookup", lookupHandler)

	healthAggregator := setupHealthChecker(app.redisClient, app.userCache, app.snapshots, app.appMode, app.environment, app.redisEnabled, healthWhitelist)
	healthHandler := i18nMiddleware(
		router.AccessLogMiddleware()(
			securityHeadersMiddleware(
				errorHandlerMiddleware(
					wrapWithTracingIfEnabled(tracingMiddleware,
						ipAllowlistMiddleware(middleware.MetricsMiddleware(
							middlewarekit.NoCacheHeadersStd()(health.Handler(healthAggregator)),
						)),
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
						ipAllowlistMiddleware(middleware.MetricsMiddleware(
							authMiddleware(
								loggerkit.LevelHandler(loggerkit.LevelHandlerConfig{
									Logger: lkLog,
								}),
							),
						)),
					),
				),
			),
		),
	)
	http.Handle("/log/level", logLevelHandler)
}

// setupHealthChecker creates a health check aggregator with all dependencies
func setupHealthChecker(redisClient *redis.Client, userCache *cache.SafeUserCache, snapshots *snapshotStore, appMode, environment string, redisEnabled bool, ipWhitelist string) *health.Aggregator {
	// Production hardening (hide details/checks) keys off the deployment ENVIRONMENT,
	// never off the data merge mode. isOnlyLocalMode still uses the merge mode.
	env, _ := config.ParseEnvironment(environment)
	isProduction := env.IsProduction()
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

	// Redis is non-critical while the in-memory cache can still serve a valid data set.
	// The critical data check keeps readiness fail-closed before any data is available.
	healthConfig := health.DefaultConfig().
		WithServiceName("warden").
		WithTimeout(5 * time.Second).
		WithIPWhitelist(ipList).
		WithDetails(!isProduction).
		WithChecks(!isProduction).
		WithCriticalChecks([]string{"data"})

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

	// Snapshot provenance checker. Serving a last-known-good snapshot after a
	// remote refresh failure is reported as "degraded" (still functional, HTTP
	// 200) rather than unhealthy, carrying only a stable, non-sensitive reason
	// code plus low-cardinality provenance metadata (source, short version
	// digest, loaded-at). Raw errors, URLs, and keys are never surfaced here.
	if snapshots != nil {
		aggregator.AddChecker(health.NewCheckerFunc("snapshot", func(_ context.Context) health.CheckResult {
			snap := snapshots.Load()
			failures, refreshReason := snapshots.RefreshFailure()
			res := health.CheckResult{
				Name:      "snapshot",
				Status:    health.StatusHealthy,
				Timestamp: time.Now(),
			}
			if snap == nil {
				res.Status = health.StatusDegraded
				res.Message = "no snapshot loaded"
				res.Metadata = map[string]any{"reason": "no_snapshot"}
				return res
			}
			meta := map[string]any{
				"source": string(snap.Source),
			}
			if snap.Version != "" {
				meta["version"] = snap.Version
			}
			if !snap.LoadedAt.IsZero() {
				meta["loaded_at"] = snap.LoadedAt.UTC().Format(time.RFC3339)
			}
			if failures > 0 {
				if refreshReason == "" {
					refreshReason = "unknown"
				}
				meta["reason"] = refreshReason
				meta["consecutive_failures"] = failures
				res.Status = health.StatusDegraded
				res.Message = "serving last-known-good snapshot after refresh failure"
			} else if snap.Degraded {
				reason := snap.DegradedReason
				if reason == "" {
					reason = "unknown"
				}
				meta["reason"] = reason
				res.Status = health.StatusDegraded
				res.Message = "serving last-known-good snapshot"
			}
			res.Metadata = meta
			return res
		}))
	}

	return aggregator
}

// wrapWithTracingIfEnabled wraps handler with tracing middleware if enabled
func wrapWithTracingIfEnabled(tracingMiddleware func(http.Handler) http.Handler, handler http.Handler) http.Handler {
	if tracingMiddleware != nil {
		return tracingMiddleware(handler)
	}
	return handler
}
