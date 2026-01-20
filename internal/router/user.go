// Package router provides HTTP routing functionality.
// Includes user queries, JSON responses and other route handlers.
package router

import (
	// Standard library
	"encoding/json"
	"net/http"
	"strings"

	// Third-party libraries
	"github.com/rs/zerolog/hlog"

	// Internal packages
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/logger"
)

// GetUserByIdentifier queries a single user by identifier
//
// This function creates an HTTP handler for querying a single user by phone, mail, or user_id.
// Supports the following query methods:
// - GET /user?phone=<phone> - Query by phone number
// - GET /user?mail=<mail> - Query by email
// - GET /user?user_id=<user_id> - Query by user ID
//
// Parameters:
//   - userCache: user cache instance for retrieving user data
//
// Returns:
//   - func(http.ResponseWriter, *http.Request): HTTP request handler function
func GetUserByIdentifier(userCache *cache.SafeUserCache) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate request method, only allow GET
		if r.Method != http.MethodGet {
			hlog.FromRequest(r).Warn().
				Str("method", r.Method).
				Msg(i18n.T(r, "log.unsupported_method"))
			http.Error(w, i18n.T(r, "http.method_not_allowed"), http.StatusMethodNotAllowed)
			return
		}

		// Get query parameters
		phone := strings.TrimSpace(r.URL.Query().Get("phone"))
		mail := strings.TrimSpace(r.URL.Query().Get("mail"))
		userID := strings.TrimSpace(r.URL.Query().Get("user_id"))

		// Validate at least one identifier is provided
		if phone == "" && mail == "" && userID == "" {
			hlog.FromRequest(r).Warn().
				Msg(i18n.T(r, "log.missing_query_params"))
			http.Error(w, i18n.T(r, "error.missing_identifier"), http.StatusBadRequest)
			return
		}

		// Validate only one identifier is provided
		identifierCount := 0
		if phone != "" {
			identifierCount++
		}
		if mail != "" {
			identifierCount++
		}
		if userID != "" {
			identifierCount++
		}
		if identifierCount > 1 {
			hlog.FromRequest(r).Warn().
				Msg(i18n.T(r, "log.multiple_query_params"))
			http.Error(w, i18n.T(r, "error.multiple_identifiers"), http.StatusBadRequest)
			return
		}

		var user define.AllowListUser
		var found bool

		// Query user by identifier type
		switch {
		case phone != "":
			user, found = userCache.GetByPhone(phone)
		case mail != "":
			user, found = userCache.GetByMail(mail)
		case userID != "":
			user, found = userCache.GetByUserID(userID)
		}

		// If user not found, return 404
		if !found {
			hlog.FromRequest(r).Info().
				Str("phone", logger.SanitizePhone(phone)).
				Str("mail", logger.SanitizeEmail(mail)).
				Str("user_id", userID).
				Msg(i18n.T(r, "log.user_not_found"))
			http.Error(w, i18n.T(r, "http.user_not_found"), http.StatusNotFound)
			return
		}

		// Return user information
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(user); err != nil {
			hlog.FromRequest(r).Error().
				Err(err).
				Msg(i18n.T(r, "error.json_encode_failed"))
			http.Error(w, i18n.T(r, "http.internal_server_error"), http.StatusInternalServerError)
			return
		}

		hlog.FromRequest(r).Info().
			Str("user_id", user.UserID).
			Str("phone", logger.SanitizePhone(user.Phone)).
			Str("mail", logger.SanitizeEmail(user.Mail)).
			Msg(i18n.T(r, "log.user_query_success"))
	}
}
