package middleware

import (
	"net/http"

	loggerkit "github.com/soulteary/logger-kit/v3"

	"github.com/soulteary/warden/internal/auditlog"
)

// AuditLogLevel records every request that tries to change the log level in
// the audit trail: who asked (the authenticated principal and client address),
// the level before and after, and whether the change was accepted. Reads (GET)
// are not recorded. It must run inside the authentication middleware so the
// principal is known.
func AuditLogLevel(log *loggerkit.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		before := log.GetLevel().String()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		auditlog.LogLevelChange(r.Context(), PrincipalFromContext(r.Context()), before, log.GetLevel().String(),
			ClientIP(r), r.UserAgent(), rw.statusCode == http.StatusOK)
	})
}
