// Package middleware provides HTTP middleware functionality.
// Includes internationalization language detection middleware.
package middleware

import (
	// Standard library
	"net/http"
	"strings"

	// Internal packages
	"github.com/soulteary/warden/internal/i18n"
)

// I18nMiddleware creates internationalization language detection middleware
//
// This middleware detects user language from HTTP requests with the following priority:
// 1. Query parameter ?lang=xx
// 2. Accept-Language request header
// 3. Default language (English)
//
// Detected language will be stored in request context for subsequent processing.
//
// Returns:
//   - func(http.Handler) http.Handler: HTTP middleware function
func I18nMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Detect language
			lang := detectLanguage(r)

			// Set language in request context
			r = i18n.SetLanguageInContext(r, lang)

			// Continue processing request
			next.ServeHTTP(w, r)
		})
	}
}

// detectLanguage detects language from request
// Priority: query parameter > Accept-Language > default
func detectLanguage(r *http.Request) i18n.Language {
	// 1. Check query parameter
	if lang := r.URL.Query().Get("lang"); lang != "" {
		return i18n.NormalizeLanguage(lang)
	}

	// 2. Check Accept-Language header
	if acceptLang := r.Header.Get("Accept-Language"); acceptLang != "" {
		return parseAcceptLanguage(acceptLang)
	}

	// 3. Default to English
	return i18n.LangEN
}

// parseAcceptLanguage parses Accept-Language request header
// Supports format: en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7
func parseAcceptLanguage(acceptLang string) i18n.Language {
	// Remove spaces
	acceptLang = strings.ReplaceAll(acceptLang, " ", "")

	// Split language tags by comma
	langs := strings.Split(acceptLang, ",")

	// Iterate through language tags, find first supported language
	for _, langTag := range langs {
		// Remove quality value (q=0.9)
		langTag = strings.Split(langTag, ";")[0]
		langTag = strings.TrimSpace(langTag)

		// Normalize language code
		normalized := i18n.NormalizeLanguage(langTag)
		if normalized != i18n.LangEN || langTag == "en" || strings.HasPrefix(langTag, "en-") {
			// If normalized value is not default, or is indeed English, return that language
			return normalized
		}
	}

	// Default to English
	return i18n.LangEN
}

// GetLanguage gets language from request (helper function)
func GetLanguage(r *http.Request) i18n.Language {
	return i18n.GetLanguageFromContext(r)
}
