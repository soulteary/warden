// Package middleware provides HTTP middleware functionality.
// Includes internationalization language detection middleware.
package middleware

import (
	"net/http"

	kit "github.com/soulteary/i18n-kit"
	"github.com/soulteary/warden/internal/i18n"
)

// I18nMiddleware creates internationalization language detection middleware.
//
// This middleware detects user language from HTTP requests with the following priority:
// 1. Query parameter ?lang=xx
// 2. Cookie
// 3. Accept-Language request header
// 4. Default language (English)
//
// Detected language will be stored in request context for subsequent processing.
//
// Returns:
//   - func(http.Handler) http.Handler: HTTP middleware function
func I18nMiddleware() func(http.Handler) http.Handler {
	return kit.StdMiddleware(kit.MiddlewareConfig{
		Bundle: i18n.GetBundle(),
	})
}

// GetLanguage gets language from request (helper function).
func GetLanguage(r *http.Request) i18n.Language {
	return kit.LanguageFromRequest(r)
}
