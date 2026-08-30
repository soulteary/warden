// Package main is the entry point of the application.
// Provides HTTP server, cache management, scheduled task scheduling and other functionality.
package main

import (
	// Standard library
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	// Third-party libraries
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/redis/go-redis/v9"
	loggerkit "github.com/soulteary/logger-kit/v2"
	rediskitclient "github.com/soulteary/redis-kit/client"

	// Middleware kit
	middlewarekit "github.com/soulteary/middleware-kit/v2"

	// Internal packages
	"github.com/soulteary/tracing-kit"
	version "github.com/soulteary/version-kit/v2"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/identity"
	"github.com/soulteary/warden/internal/loader"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/prommetrics"
	"github.com/soulteary/warden/internal/remote"
	"github.com/soulteary/warden/pkg/gocron"
)

// App application struct that encapsulates all application state.
//
//nolint:govet // fieldalignment: keep field order for readability; optional size win would reorder pointers/strings/bools
type App struct {
	userCache            *cache.SafeUserCache
	redisUserCache       *cache.RedisUserCache
	redisClient          *redis.Client
	rateLimiter          *middlewarekit.RateLimiter
	rulesLoader          *loader.RulesLoader
	snapshots            *snapshotStore
	log                  *loggerkit.Logger
	port                 string
	configURL            string
	authorizationHeader  string
	appMode              string
	environment          string
	apiKey               string
	dataFile             string
	dataDir              string
	responseFields       []string
	taskInterval         uint64
	snapshotMaxAge       time.Duration
	redisEnabled         bool
	hmacKeys             map[string]string
	hmacToleranceSec     int
	hmacAllowV1          bool
	tlsCertFile          string
	tlsKeyFile           string
	tlsCAFile            string
	tlsRequireClientCert bool
}

// taskIntervalU64 converts task interval to uint64, clamping negative values to 0 to avoid overflow.
func taskIntervalU64(sec int) uint64 {
	if sec <= 0 {
		return 0
	}
	return uint64(sec)
}

// NewApp creates a new application instance
func NewApp(cfg *cmd.Config) *App {
	hmacAllowV1, hmacAllowV1Err := cmd.ParseHMACAllowV1(os.Getenv("WARDEN_HMAC_ALLOW_V1"))
	snapshotMaxAge, snapshotMaxAgeErr := cmd.ParseSnapshotMaxAge(os.Getenv("SNAPSHOT_MAX_AGE"), cfg.TaskInterval)
	app := &App{
		port:                 cfg.Port,
		configURL:            cfg.RemoteConfig,
		authorizationHeader:  cfg.RemoteKey,
		appMode:              cfg.Mode,
		environment:          cfg.Environment,
		dataFile:             cfg.DataFile,
		dataDir:              cfg.DataDir,
		responseFields:       cfg.ResponseFields,
		taskInterval:         taskIntervalU64(cfg.TaskInterval),
		snapshotMaxAge:       snapshotMaxAge,
		apiKey:               cfg.APIKey,
		redisEnabled:         cfg.RedisEnabled,
		log:                  logger.GetLoggerKit(),
		hmacToleranceSec:     cfg.HMACToleranceSec,
		hmacAllowV1:          hmacAllowV1,
		tlsCertFile:          cfg.TLSCertFile,
		tlsKeyFile:           cfg.TLSKeyFile,
		tlsCAFile:            cfg.TLSCAFile,
		tlsRequireClientCert: cfg.TLSRequireClientCert,
	}
	app.snapshots = newSnapshotStore()
	if snapshotMaxAgeErr != nil {
		// ValidateConfig rejects this during normal startup. Direct NewApp callers
		// retain the derived default rather than silently disabling stale detection.
		app.snapshotMaxAge = cmd.DefaultSnapshotMaxAge(cfg.TaskInterval)
		app.log.Warn().Err(snapshotMaxAgeErr).Msg("invalid SNAPSHOT_MAX_AGE; using derived default")
	}
	if hmacAllowV1Err != nil {
		// ValidateConfig rejects this during normal startup. Keep the fallback secure
		// for direct NewApp callers as a defense-in-depth measure.
		app.log.Warn().Err(hmacAllowV1Err).Msg("invalid WARDEN_HMAC_ALLOW_V1; keeping secure default")
	}
	// Surface use of the deprecated legacy encryption format as a metric (deduped).
	remote.LegacyEncryptionObserver = func() {
		prommetrics.RecordDeprecation("encryption_legacy")
	}
	if cfg.HMACKeys != "" {
		keys, err := cmd.ParseHMACKeys(cfg.HMACKeys)
		if err != nil {
			app.log.Warn().Err(err).Msg("WARDEN_HMAC_KEYS invalid, HMAC auth disabled")
		} else {
			app.hmacKeys = keys
		}
	}

	if cfg.HTTPInsecureTLS {
		app.log.Warn().Msg(i18n.TWithLang(i18n.LangZH, "log.http_tls_disabled"))
		// In production environment, force TLS verification. Production is determined by the
		// dedicated ENVIRONMENT (with legacy MODE=production migration), never the merge mode.
		if env, _ := config.ParseEnvironment(cfg.Environment); env.IsProduction() {
			app.log.Fatal().Msg(i18n.TWithLang(i18n.LangZH, "log.prod_tls_required"))
		}
	}

	// Emit a one-time deprecation warning when a legacy MODE=production/prod was used to
	// derive the environment, so operators migrate to ENVIRONMENT.
	if cfg.LegacyModeUsedForEnv {
		app.log.Warn().Msg(i18n.TWithLang(i18n.LangZH, "log.legacy_mode_env_deprecated"))
		prommetrics.RecordDeprecation("mode_legacy_env")
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

	// Apply identity configuration (user_id derivation strategy + explicit-id requirement)
	// BEFORE any data is loaded so Normalize/validation observe a consistent policy.
	if strategy, ok := define.ParseUserIDStrategy(cfg.UserIDStrategy); ok {
		define.SetUserIDStrategy(strategy)
	} else {
		app.log.Warn().
			Str("value", cfg.UserIDStrategy).
			Msg("Unknown USER_ID_STRATEGY; falling back to legacy")
		define.SetUserIDStrategy(define.UserIDStrategyLegacy)
	}
	define.SetRequireExplicitUserID(cfg.RequireExplicitUserID)

	// Load initial data (multi-level fallback)
	if app.rulesLoader != nil {
		if err := app.loadInitialData(cfg.DataFile, cfg.DataDir); err != nil {
			app.log.Warn().Err(err).Msg(i18n.TWithLang(i18n.LangZH, "log.load_initial_data_failed"))
		}
	}

	// Read-only identity diagnostic: report (masked) conflict / missing-id counts and the
	// active user_id strategy so operators can assess migration risk without failing startup.
	app.reportIdentityDiagnostics()

	// Initialize cache size metrics
	prommetrics.CacheSize.Set(float64(app.userCache.Len()))

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

// reportIdentityDiagnostics runs a read-only (report-only) identity validation over the
// currently-loaded rule set and logs masked conflict / missing-id counts plus the active
// user_id derivation strategy. It never mutates the cache and never fails startup; it only
// surfaces migration risk (e.g. sets that would be rejected once REQUIRE_EXPLICIT_USER_ID
// or a stricter strategy is enabled).
func (app *App) reportIdentityDiagnostics() {
	users := app.userCache.Get()
	if len(users) == 0 {
		return
	}
	res, err := identity.ValidateAndIndexUsers(users, identity.Options{
		RequireExplicitUserID: define.RequireExplicitUserID(),
		ReportOnly:            true,
	})
	if err != nil {
		// ReportOnly should not return an error; log defensively without leaking data.
		app.log.Warn().Msg(i18n.TWithLang(i18n.LangZH, "log.identity_diagnostic_failed"))
		return
	}
	ev := app.log.Info().
		Str("user_id_strategy", string(define.GetUserIDStrategy())).
		Bool("require_explicit_user_id", define.RequireExplicitUserID()).
		Int("conflicts", res.ConflictCount).
		Int("missing_user_id", res.MissingIDCount)
	if res.ConflictCount > 0 || res.MissingIDCount > 0 {
		ev.Msg(i18n.TWithLang(i18n.LangZH, "log.identity_diagnostic_risk"))
		return
	}
	ev.Msg(i18n.TWithLang(i18n.LangZH, "log.identity_diagnostic_ok"))
}

// applyUsers validates a rule set through centralized identity validation and, only on
// success, swaps it into the shared cache. On a uniqueness/missing-id conflict it keeps
// the current cache contents (last-known-good) and returns the typed error so callers can
// decide whether to fall back. This is the single choke point ensuring no partial/
// last-write-wins set ever enters the shared cache.
func (app *App) applyUsers(users []define.AllowListUser) error {
	opts := identity.Options{RequireExplicitUserID: define.RequireExplicitUserID()}
	if err := app.userCache.SetValidated(users, opts); err != nil {
		app.log.Warn().
			Err(err).
			Msg(i18n.TWithLang(i18n.LangZH, "log.identity_validation_failed"))
		return err
	}
	return nil
}

// loadInitialData loads data with multi-level fallback (Redis → parser-kit Load).
func (app *App) loadInitialData(rulesFile, dataDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), define.DEFAULT_LOAD_DATA_TIMEOUT)
	defer cancel()

	app.log.Debug().Str("appMode", app.appMode).Msg(i18n.TWithLang(i18n.LangZH, "log.check_mode"))
	if strings.ToUpper(strings.TrimSpace(app.appMode)) == "ONLY_LOCAL" {
		app.log.Debug().Msg(i18n.TWithLang(i18n.LangZH, "log.only_local_detected"))
		localRes := app.rulesLoader.LoadWithResult(ctx, rulesFile, dataDir, "", "")
		localUsers := localRes.Users
		if localRes.Err == nil && len(localUsers) > 0 {
			app.log.Info().
				Int("count", len(localUsers)).
				Msg(i18n.TWithLang(i18n.LangZH, "log.loaded_from_local_file"))
			if err := app.applyUsers(localUsers); err != nil {
				// Conflicting local set: do not overwrite good data; report and abort this load.
				return err
			}
			app.snapshots.Store(snapshotFromResult(&localRes))
			app.updateSnapshotMetrics()
			if app.redisUserCache != nil {
				if err := app.redisUserCache.Set(localUsers); err != nil {
					app.log.Warn().Err(err).Msg(i18n.TWithLang(i18n.LangZH, "log.redis_cache_update_failed"))
				}
			}
			return nil
		}
		if dataDir == "" {
			_, statErr := os.Stat(rulesFile)
			if errors.Is(statErr, os.ErrNotExist) {
				app.log.Warn().
					Str("data_file", rulesFile).
					Str("example_file", "data.example.json").
					Msg(i18n.TWithLang(i18n.LangZH, "log.data_file_not_found"))
				app.log.Info().Msg(i18n.TWithLang(i18n.LangZH, "log.only_local_requires_file"))
				app.log.Info().Msgf(i18n.TWithLang(i18n.LangZH, "log.create_data_file"), rulesFile, "data.example.json")
			}
		}
		app.log.Warn().Msg(i18n.TWithLang(i18n.LangZH, "log.only_local_load_failed"))
		return nil
	}

	// 1. Try to load from Redis cache (if Redis is available)
	if app.redisUserCache != nil {
		if cachedUsers, err := app.redisUserCache.Get(); err == nil && len(cachedUsers) > 0 {
			prommetrics.CacheHits.Inc()
			app.log.Info().
				Int("count", len(cachedUsers)).
				Msg(i18n.TWithLang(i18n.LangZH, "log.loaded_from_redis"))
			if err := app.applyUsers(cachedUsers); err != nil {
				// Redis held a conflicting set; fall through to remote/local sources.
				prommetrics.CacheMisses.Inc()
			} else {
				return nil
			}
		} else {
			prommetrics.CacheMisses.Inc()
		}
	}

	// 2. Try to load from parser-kit (remote + local by mode)
	res := app.rulesLoader.LoadWithResult(ctx, rulesFile, dataDir, app.configURL, app.authorizationHeader)
	users := res.Users
	if res.Err == nil && len(users) > 0 {
		app.log.Info().
			Int("count", len(users)).
			Str("source", string(res.Source)).
			Bool("degraded", res.Degraded).
			Msg(i18n.TWithLang(i18n.LangZH, "log.loaded_from_remote_api"))
		if err := app.applyUsers(users); err != nil {
			// Conflicting rule set: keep last-known-good and report all sources failed below.
			app.log.Warn().Err(err).Msg(i18n.TWithLang(i18n.LangZH, "log.all_sources_failed"))
			return nil
		}
		app.snapshots.Store(snapshotFromResult(&res))
		app.updateSnapshotMetrics()
		if res.Degraded {
			prommetrics.RemoteFallbackTotal.WithLabelValues(strings.ToUpper(strings.TrimSpace(app.appMode)), res.DegradedReason).Inc()
		}
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
	newHash := cache.HashUserList(newUsers)
	return oldHash != newHash
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
//   - Updates Prometheus metrics (prommetrics.BackgroundTaskTotal, prommetrics.BackgroundTaskDuration, etc.)
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
func (app *App) backgroundTask(rulesFile, dataDir string) {
	defer func() {
		if r := recover(); r != nil {
			prommetrics.BackgroundTaskErrors.Inc()
			app.log.Error().
				Interface("panic", r).
				Msg(i18n.TWithLang(i18n.LangZH, "log.background_task_panic"))
		}
	}()

	start := time.Now()

	if app.rulesLoader == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(define.DEFAULT_TIMEOUT*2)*time.Second)
	defer cancel()
	var res loader.LoadResult
	if strings.ToUpper(strings.TrimSpace(app.appMode)) == "ONLY_LOCAL" {
		res = app.rulesLoader.LoadWithResult(ctx, rulesFile, dataDir, "", "")
	} else {
		res = app.rulesLoader.LoadWithResult(ctx, rulesFile, dataDir, app.configURL, app.authorizationHeader)
	}
	if res.Err != nil {
		// Refresh failed: keep the last-known-good snapshot and cache untouched.
		reason := classifyRefreshReason(res.Err)
		failures := app.snapshots.RecordRefreshFailure(reason)
		app.updateSnapshotMetrics()
		prommetrics.RefreshFailuresTotal.WithLabelValues(reason).Inc()
		app.log.Warn().
			Err(res.Err).
			Str("reason", reason).
			Int64("consecutive_failures", failures).
			Msg(i18n.TWithLang(i18n.LangZH, "log.background_load_failed"))
		return
	}

	newUsers := res.Users
	if res.Degraded {
		prommetrics.RemoteFallbackTotal.WithLabelValues(strings.ToUpper(strings.TrimSpace(app.appMode)), res.DegradedReason).Inc()
	}

	// Check if data has changed
	if !app.checkDataChanged(newUsers) {
		// Even when unchanged, refresh snapshot metadata (age/source/degraded) so a
		// successful refresh clears the failure counter and updates degraded state.
		app.snapshots.Store(snapshotFromResult(&res))
		app.updateSnapshotMetrics()
		app.log.Debug().Msg(i18n.TWithLang(i18n.LangZH, "log.data_unchanged"))
		return
	}

	// Update memory cache and swap in the new immutable snapshot atomically. Validation
	// happens inside applyUsers; on a conflict we keep the last-known-good snapshot and
	// record a refresh failure instead of overwriting good data with a bad set.
	if err := app.applyUsers(newUsers); err != nil {
		failures := app.snapshots.RecordRefreshFailure("identity_conflict")
		app.updateSnapshotMetrics()
		prommetrics.RefreshFailuresTotal.WithLabelValues("identity_conflict").Inc()
		app.log.Warn().
			Err(err).
			Str("reason", "identity_conflict").
			Int64("consecutive_failures", failures).
			Msg(i18n.TWithLang(i18n.LangZH, "log.background_load_failed"))
		return
	}
	app.snapshots.Store(snapshotFromResult(&res))

	// Verify data consistency (optimistic locking strategy)
	currentHash := app.userCache.GetHash()
	newHash := cache.HashUserList(newUsers)
	if currentHash != "" && currentHash == newHash {
		// Data consistent, update Redis cache (if Redis is available)
		if app.redisUserCache != nil {
			if err := app.updateRedisCacheWithRetry(newUsers); err != nil {
				app.log.Warn().
					Err(err).
					Msg(i18n.TWithLang(i18n.LangZH, "log.redis_cache_failed_continue"))
				prommetrics.BackgroundTaskErrors.Inc()
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
	prommetrics.BackgroundTaskTotal.Inc()
	prommetrics.BackgroundTaskDuration.Observe(duration)
	prommetrics.CacheSize.Set(float64(app.userCache.Len()))
	app.updateSnapshotMetrics()

	app.log.Info().
		Int("count", len(newUsers)).
		Float64("duration", duration).
		Msg(i18n.TWithLang(i18n.LangZH, "log.background_update"))
}

// startServer starts HTTP server. When tlsCertFile and tlsKeyFile are set, TLS (and optional mTLS) is enabled;
// the caller must use ListenAndServeTLS(certFile, keyFile) instead of ListenAndServe().
func startServer(port, tlsCertFile, tlsKeyFile, tlsCAFile string, tlsRequireClientCert bool) *http.Server {
	srv := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: define.DEFAULT_TIMEOUT * time.Second,
		ReadTimeout:       define.DEFAULT_TIMEOUT * time.Second,
		WriteTimeout:      define.DEFAULT_TIMEOUT * time.Second,
		IdleTimeout:       define.IDLE_TIMEOUT,
		MaxHeaderBytes:    define.MAX_HEADER_BYTES,
	}
	if tlsCertFile != "" && tlsKeyFile != "" {
		tlsCfg, err := buildTLSConfig(tlsCertFile, tlsKeyFile, tlsCAFile, tlsRequireClientCert)
		if err != nil {
			logger.GetLoggerKit().Fatal().Err(err).Msg("Failed to build TLS config for mTLS")
		}
		srv.TLSConfig = tlsCfg
	}
	return srv
}

// buildTLSConfig builds tls.Config for server and optional client cert verification (mTLS).
func buildTLSConfig(certFile, keyFile, caFile string, requireClientCert bool) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load TLS cert/key: %w", err)
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	if caFile != "" {
		// #nosec G304 -- caFile is from trusted config (env WARDEN_TLS_CA), not user input
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read client CA: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("parse client CA failed")
		}
		cfg.ClientCAs = pool
		if requireClientCert {
			cfg.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			cfg.ClientAuth = tls.VerifyClientCertIfGiven
		}
	}
	return cfg, nil
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

// showBanner displays the startup banner with version
func showBanner() {
	pterm.DefaultBox.Println(
		putils.CenterText(
			"Warden\n" +
				"Allowlist & Authorization Service\n" +
				"Version: " + version.Version,
		),
	)
	time.Sleep(time.Millisecond) // Don't ask why, but this fixes the docker-compose log
}

func main() {
	// Display startup banner
	showBanner()

	log := logger.GetLoggerKit()

	// Parse configuration
	cfg, err := cmd.GetArgsWithError()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg(i18n.TWithLang(i18n.LangZH, "log.config_validation_failed_exit"))
	}

	// Validate configuration
	if err := cmd.ValidateConfig(cfg); err != nil {
		log.Fatal().
			Err(err).
			Msg(i18n.TWithLang(i18n.LangZH, "log.config_validation_failed_exit"))
	}

	// Load config from file if config file is specified (for tracing config)
	var tracingCfg *config.Config
	if cfgFile := cfg.ConfigFile; cfgFile != "" {
		if loadedCfg, err := config.ParseFromFile(cfgFile); err == nil {
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
	if err := scheduler.Every(app.taskInterval).Seconds().Lock().Do(app.backgroundTask, app.dataFile, app.dataDir); err != nil {
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

	// Start server (TLS/mTLS when cert and key are set)
	srv := startServer(app.port, app.tlsCertFile, app.tlsKeyFile, app.tlsCAFile, app.tlsRequireClientCert)
	app.log.Info().Msgf(i18n.TWithLang(i18n.LangZH, "log.service_listening"), app.port)
	go func() {
		var err error
		if app.tlsCertFile != "" && app.tlsKeyFile != "" {
			err = srv.ListenAndServeTLS(app.tlsCertFile, app.tlsKeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
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
