// Package router provides HTTP routing functionality.
// Includes request logging, JSON responses, health checks and other route handlers.
package router

import (
	// Standard library
	"encoding/json"
	"net/http"
	"strings"

	// Third-party libraries
	"github.com/rs/zerolog"

	// Internal packages
	"github.com/soulteary/warden/internal/auditlog"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/logger"
)

// LogLevelHandler handles HTTP endpoint for log level adjustment
// Supports GET (query current level) and POST (set new level)
func LogLevelHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			// Get current log level
			currentLevel := zerolog.GlobalLevel()
			response := map[string]interface{}{
				"level": currentLevel.String(),
			}
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log := logger.GetLogger()
				log.Error().Err(err).Msg("Failed to encode log level response")
			}

		case http.MethodPost:
			// Set new log level (requires authentication)
			var request struct {
				Level string `json:"level"`
			}

			log := logger.GetLogger()

			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(map[string]string{
					"error": i18n.T(r, "log_level.invalid_request_body"),
				}); err != nil {
					log.Error().Err(err).Msg("Failed to encode error response")
				}
				return
			}

			levelText := strings.TrimSpace(request.Level)
			// Validate level field is not empty
			if levelText == "" {
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(map[string]string{
					"error": i18n.T(r, "log_level.empty"),
				}); err != nil {
					log.Error().Err(err).Msg("Failed to encode error response")
				}
				return
			}

			level, err := zerolog.ParseLevel(strings.ToLower(levelText))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(map[string]string{
					"error": i18n.T(r, "log_level.invalid"),
				}); err != nil {
					log.Error().Err(err).Msg("Failed to encode error response")
				}
				return
			}

			// Record log level modification operation (security audit)
			oldLevel := zerolog.GlobalLevel()
			logger.SetLevel(level)

			// Record security event: log level was modified via audit-kit
			auditlog.LogConfigChange(r.Context(), "log_level", oldLevel.String(), level.String(), r.RemoteAddr, r.UserAgent())

			log.Info().
				Str("old_level", oldLevel.String()).
				Str("new_level", level.String()).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Msg("Log level modified (security audit)")

			response := map[string]interface{}{
				"message": i18n.T(r, "log_level.updated"),
				"level":   level.String(),
			}
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Error().Err(err).Msg("Failed to encode log level response")
			}

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			if err := json.NewEncoder(w).Encode(map[string]string{
				"error": i18n.T(r, "log_level.method_not_allowed"),
			}); err != nil {
				log := logger.GetLogger()
				log.Error().Err(err).Msg("Failed to encode error response")
			}
		}
	}
}
