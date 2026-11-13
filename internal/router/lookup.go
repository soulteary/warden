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

	tracing "github.com/soulteary/tracing-kit/v2"
	"github.com/soulteary/warden/internal/auditlog"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/middleware"
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
//
// If responseFields is non-empty it is applied as the same allowlist used by the
// list and user handlers: "mail" governs destination.email, "phone" governs
// destination.phone, and "name" and "dingtalk_userid" govern the fields of the
// same name. user_id, status, channel_hint and the destination object are part of
// the lookup contract and are always returned. channel_hint is derived from the
// destination that is actually returned, so it does not reveal a withheld phone.
func GetLookup(userCache cache.ViewSource, responseFields []string) func(http.ResponseWriter, *http.Request) {
	allowed := lookupFieldFilter(responseFields)

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

		view := userCache.ReadView()

		var user define.AllowListUser
		var found bool
		if strings.Contains(identifier, "@") {
			user, found = view.GetByMail(identifier)
		} else {
			user, found = view.GetByPhone(identifier)
			if !found {
				user, found = view.GetByUserID(identifier)
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
			auditlog.LogUserQuery(r.Context(), "", sanitized, "identifier", middleware.ClientIP(r), false, "user_not_found")
			WriteJSONError(w, http.StatusNotFound, i18n.T(r, "http.user_not_found"))
			return
		}

		span.SetAttributes(attribute.Bool("warden.lookup.found", true), attribute.String("warden.user.id", user.UserID))

		resp := LookupResponse{
			UserID: user.UserID,
			Status: user.Status,
		}
		if allowed("mail") {
			resp.Destination.Email = strings.TrimSpace(user.Mail)
		}
		if allowed("phone") {
			resp.Destination.Phone = strings.TrimSpace(user.Phone)
		}
		if allowed("name") {
			resp.Name = strings.TrimSpace(user.Name)
		}
		if allowed("dingtalk_userid") {
			resp.DingtalkUserID = strings.TrimSpace(user.DingtalkUserID)
		}

		channelHint := "email"
		if resp.Destination.Phone != "" {
			channelHint = "sms"
		}
		resp.ChannelHint = channelHint

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
		auditlog.LogUserQuery(r.Context(), user.UserID, sanitized, "identifier", middleware.ClientIP(r), true, "")
	}
}

// lookupFieldFilter reports whether an optional lookup field may be returned.
// An empty allowlist allows every field, matching UserToMap.
func lookupFieldFilter(responseFields []string) func(string) bool {
	if len(responseFields) == 0 {
		return func(string) bool { return true }
	}
	set := make(map[string]bool, len(responseFields))
	for _, f := range responseFields {
		set[f] = true
	}
	return func(field string) bool { return set[field] }
}
