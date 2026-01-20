// Package router provides HTTP routing functionality.
// Includes request logging, JSON responses, health checks and other route handlers.
package router

import (
	// Standard library
	"context"
	"encoding/json"
	"net/http"
	"time"

	// Third-party libraries
	"github.com/redis/go-redis/v9"

	// Internal packages
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/logger"
)

// HealthCheck returns health check handler
// Checks Redis connection status and whether data is loaded
// appMode controls response detail level: production environment ("production") hides detailed information, development environment shows detailed information
// redisEnabled indicates whether Redis is explicitly enabled (used to distinguish between disabled and unavailable states)
func HealthCheck(redisClient *redis.Client, userCache *cache.SafeUserCache, appMode string, redisEnabled bool) http.HandlerFunc {
	// Determine if it's production environment
	isProduction := appMode == "production" || appMode == "prod"

	return func(w http.ResponseWriter, _ *http.Request) {
		status := "ok"
		code := http.StatusOK
		details := make(map[string]interface{})

		// Check Redis connection status
		switch {
		case !redisEnabled:
			// Redis is explicitly disabled
			details["redis"] = "disabled"
		case redisClient != nil:
			// Redis is enabled, check connection status
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := redisClient.Ping(ctx).Err(); err != nil {
				status = "redis_unavailable"
				code = http.StatusServiceUnavailable
				details["redis"] = "unavailable"
				// Production environment does not return detailed error information to avoid leaking sensitive information
				if !isProduction {
					// Only return detailed error information in non-production environment
					details["redis_error"] = err.Error()
				}
			} else {
				details["redis"] = "ok"
			}
		default:
			// Redis client is nil (may be fallback state after connection failure)
			status = "redis_unavailable"
			code = http.StatusServiceUnavailable
			details["redis"] = "unavailable"
		}

		// Check if data is loaded
		if userCache != nil {
			userCount := userCache.Len()
			details["data_loaded"] = userCount > 0
			// Production environment hides specific user count, only returns whether data is loaded
			if isProduction {
				// Production environment: only return boolean value, not specific count
				details["data_loaded"] = userCount > 0
			} else {
				// Development environment: return detailed information
				details["data_loaded"] = userCount > 0
				details["user_count"] = userCount
			}
			if userCount == 0 {
				// Data not loaded does not affect health status, but recorded in details
				if !isProduction {
					// Only return warning information in non-production environment
					details["data_warning"] = "no data loaded yet"
				}
			}
		} else {
			details["data_loaded"] = false
			if !isProduction {
				// Only return warning information in non-production environment
				details["data_warning"] = "cache not initialized"
			}
		}

		response := map[string]interface{}{
			"status":  status,
			"details": details,
		}

		// Production environment does not return mode information
		if !isProduction {
			response["mode"] = appMode
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log := logger.GetLogger()
			// Note: health check may not have request context, use default language
			log.Error().
				Err(err).
				Msg(i18n.TWithLang(i18n.LangEN, "log.health_check_encode_failed"))
			// If status code has already been written, cannot modify it, only log error
		}
	}
}
