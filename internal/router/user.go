// Package router provides HTTP routing functionality.
// Includes user queries, JSON responses and other route handlers.
package router

import (
	// Standard library
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	// Third-party libraries
	"go.opentelemetry.io/otel/attribute"

	// Internal packages
	"github.com/soulteary/tracing-kit"
	"github.com/soulteary/warden/internal/auditlog"
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
		// Start span for user query
		_, span := tracing.StartSpan(r.Context(), "warden.get_user")
		defer span.End()

		// Validate request method, only allow GET
		if r.Method != http.MethodGet {
			tracing.RecordError(span, errors.New("method not allowed"))
			logger.FromRequest(r).Warn().
				Str("method", r.Method).
				Msg(i18n.T(r, "log.unsupported_method"))
			WriteJSONError(w, http.StatusMethodNotAllowed, i18n.T(r, "http.method_not_allowed"))
			return
		}

		// Get query parameters
		phone := strings.TrimSpace(r.URL.Query().Get("phone"))
		mail := strings.TrimSpace(r.URL.Query().Get("mail"))
		userID := strings.TrimSpace(r.URL.Query().Get("user_id"))

		// Set span attributes
		span.SetAttributes(
			attribute.String("warden.query.phone", logger.SanitizePhone(phone)),
			attribute.String("warden.query.mail", logger.SanitizeEmail(mail)),
			attribute.String("warden.query.user_id", userID),
		)

		// Validate at least one identifier is provided
		if phone == "" && mail == "" && userID == "" {
			tracing.RecordError(span, fmt.Errorf("missing identifier"))
			logger.FromRequest(r).Warn().
				Msg(i18n.T(r, "log.missing_query_params"))
			WriteJSONError(w, http.StatusBadRequest, i18n.T(r, "error.missing_identifier"))
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
			tracing.RecordError(span, fmt.Errorf("multiple identifiers provided"))
			logger.FromRequest(r).Warn().
				Msg(i18n.T(r, "log.multiple_query_params"))
			WriteJSONError(w, http.StatusBadRequest, i18n.T(r, "error.multiple_identifiers"))
			return
		}

		// Security: limit identifier length to prevent DoS and log/cache bloat
		if (phone != "" && len(phone) > define.MAX_IDENTIFIER_LENGTH) ||
			(mail != "" && len(mail) > define.MAX_IDENTIFIER_LENGTH) ||
			(userID != "" && len(userID) > define.MAX_IDENTIFIER_LENGTH) {
			tracing.RecordError(span, fmt.Errorf("identifier too long"))
			logger.FromRequest(r).Warn().Msg(i18n.T(r, "error.invalid_identifier"))
			WriteJSONError(w, http.StatusBadRequest, i18n.T(r, "error.invalid_identifier"))
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
			span.SetAttributes(attribute.Bool("warden.user.found", false))
			logger.FromRequest(r).Info().
				Str("phone", logger.SanitizePhone(phone)).
				Str("mail", logger.SanitizeEmail(mail)).
				Str("user_id", userID).
				Msg(i18n.T(r, "log.user_not_found"))

			// Audit log: user query failed (identifier sanitized to avoid PII in audit storage)
			identifier := phone
			identifierType := "phone"
			if mail != "" {
				identifier = mail
				identifierType = "mail"
			} else if userID != "" {
				identifier = userID
				identifierType = "user_id"
			}
			auditlog.LogUserQuery(r.Context(), "", sanitizeIdentifierForAudit(identifier, identifierType), identifierType, r.RemoteAddr, false, "user_not_found")

			WriteJSONError(w, http.StatusNotFound, i18n.T(r, "http.user_not_found"))
			return
		}

		// Set span attributes for found user
		span.SetAttributes(
			attribute.Bool("warden.user.found", true),
			attribute.String("warden.user.id", user.UserID),
		)

		// Return user information
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(user); err != nil {
			tracing.RecordError(span, err)
			logger.FromRequest(r).Error().
				Err(err).
				Msg(i18n.T(r, "error.json_encode_failed"))
			WriteJSONError(w, http.StatusInternalServerError, i18n.T(r, "http.internal_server_error"))
			return
		}

		logger.FromRequest(r).Info().
			Str("user_id", user.UserID).
			Str("phone", logger.SanitizePhone(user.Phone)).
			Str("mail", logger.SanitizeEmail(user.Mail)).
			Msg(i18n.T(r, "log.user_query_success"))

		// Audit log: user query success (identifier sanitized to avoid PII in audit storage)
		identifier := phone
		identifierType := "phone"
		if mail != "" {
			identifier = mail
			identifierType = "mail"
		} else if userID != "" {
			identifier = userID
			identifierType = "user_id"
		}
		auditlog.LogUserQuery(r.Context(), user.UserID, sanitizeIdentifierForAudit(identifier, identifierType), identifierType, r.RemoteAddr, true, "")
	}
}

// sanitizeIdentifierForAudit returns a sanitized identifier for audit log storage to avoid storing plain PII.
// phone and mail are masked; user_id is returned as-is (internal id, not direct PII).
func sanitizeIdentifierForAudit(identifier, identifierType string) string {
	switch identifierType {
	case "phone":
		return logger.SanitizePhone(identifier)
	case "mail":
		return logger.SanitizeEmail(identifier)
	default:
		return identifier
	}
}
