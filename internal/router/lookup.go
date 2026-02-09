// Package router provides HTTP routing functionality.
// Lookup handler for Stargate integration: GET /v1/lookup?identifier=xxx
package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"

	"github.com/soulteary/tracing-kit"
	"github.com/soulteary/warden/internal/auditlog"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/logger"
)

// LookupResponse is the response body for GET /v1/lookup (Stargate/Herald integration).
type LookupResponse struct {
	UserID         string      `json:"user_id"`
	Destination    Destination `json:"destination"`
	Status         string      `json:"status"`
	ChannelHint    string      `json:"channel_hint,omitempty"`    // "sms" or "email" for OTP
	Name           string      `json:"name,omitempty"`            // User display name (optional)
	DingtalkUserID string      `json:"dingtalk_userid,omitempty"` // DingTalk user ID for work notification (optional)
}

// Destination holds email and phone for OTP delivery.
type Destination struct {
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// GetLookup returns a handler for GET /v1/lookup?identifier=xxx.
// identifier is auto-detected: if it contains @ then mail; else try phone then user_id.
// Returns { user_id, destination: { email?, phone? }, status, channel_hint } for Stargate/Herald.
func GetLookup(userCache *cache.SafeUserCache) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, span := tracing.StartSpan(r.Context(), "warden.lookup")
		defer span.End()

		if r.Method != http.MethodGet {
			tracing.RecordError(span, errors.New("method not allowed"))
			logger.FromRequest(r).Warn().Str("method", r.Method).Msg(i18n.T(r, "log.unsupported_method"))
			WriteJSONError(w, http.StatusMethodNotAllowed, i18n.T(r, "http.method_not_allowed"))
			return
		}

		identifier := strings.TrimSpace(r.URL.Query().Get("identifier"))
		if identifier == "" {
			tracing.RecordError(span, fmt.Errorf("missing identifier"))
			logger.FromRequest(r).Warn().Msg(i18n.T(r, "log.missing_query_params"))
			WriteJSONError(w, http.StatusBadRequest, i18n.T(r, "error.missing_identifier"))
			return
		}

		if len(identifier) > define.MAX_IDENTIFIER_LENGTH {
			tracing.RecordError(span, fmt.Errorf("identifier too long"))
			logger.FromRequest(r).Warn().Msg(i18n.T(r, "error.invalid_identifier"))
			WriteJSONError(w, http.StatusBadRequest, i18n.T(r, "error.invalid_identifier"))
			return
		}

		span.SetAttributes(attribute.String("warden.lookup.identifier_len", fmt.Sprintf("%d", len(identifier))))

		var user define.AllowListUser
		var found bool
		if strings.Contains(identifier, "@") {
			user, found = userCache.GetByMail(identifier)
		} else {
			user, found = userCache.GetByPhone(identifier)
			if !found {
				user, found = userCache.GetByUserID(identifier)
			}
		}

		if !found {
			span.SetAttributes(attribute.Bool("warden.lookup.found", false))
			logger.FromRequest(r).Info().Str("identifier", logger.SanitizeEmail(identifier)).Msg(i18n.T(r, "log.user_not_found"))
			var sanitized string
			if strings.Contains(identifier, "@") {
				sanitized = logger.SanitizeEmail(identifier)
			} else {
				sanitized = logger.SanitizePhone(identifier)
			}
			auditlog.LogUserQuery(r.Context(), "", sanitized, "identifier", r.RemoteAddr, false, "user_not_found")
			WriteJSONError(w, http.StatusNotFound, i18n.T(r, "http.user_not_found"))
			return
		}

		span.SetAttributes(attribute.Bool("warden.lookup.found", true), attribute.String("warden.user.id", user.UserID))

		channelHint := "email"
		if strings.TrimSpace(user.Phone) != "" {
			channelHint = "sms"
		}

		resp := LookupResponse{
			UserID:         user.UserID,
			Status:         user.Status,
			ChannelHint:    channelHint,
			Name:           strings.TrimSpace(user.Name),
			DingtalkUserID: strings.TrimSpace(user.DingtalkUserID),
			Destination: Destination{
				Email: strings.TrimSpace(user.Mail),
				Phone: strings.TrimSpace(user.Phone),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			tracing.RecordError(span, err)
			logger.FromRequest(r).Error().Err(err).Msg(i18n.T(r, "error.json_encode_failed"))
			WriteJSONError(w, http.StatusInternalServerError, i18n.T(r, "http.internal_server_error"))
			return
		}

		logger.FromRequest(r).Info().
			Str("user_id", user.UserID).
			Str("channel_hint", channelHint).
			Msg(i18n.T(r, "log.user_query_success"))
		var sanitized string
		if strings.Contains(identifier, "@") {
			sanitized = logger.SanitizeEmail(identifier)
		} else {
			sanitized = logger.SanitizePhone(identifier)
		}
		auditlog.LogUserQuery(r.Context(), user.UserID, sanitized, "identifier", r.RemoteAddr, true, "")
	}
}
