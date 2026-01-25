// Package i18n provides internationalization support.
// Supports language detection from request context and multi-language text translation.
// This is an adapter layer for i18n-kit.
package i18n

import (
	"context"
	"fmt"
	"net/http"

	kit "github.com/soulteary/i18n-kit"
	"github.com/soulteary/warden/locales"
)

// Language is an alias for kit.Language for backward compatibility.
type Language = kit.Language

// Language constants for backward compatibility.
const (
	LangEN Language = kit.LangEN
	LangZH Language = kit.LangZH
	LangFR Language = kit.LangFR
	LangIT Language = kit.LangIT
	LangJA Language = kit.LangJA
	LangDE Language = kit.LangDE
	LangKO Language = kit.LangKO
)

// bundle is the translation bundle.
var bundle *kit.Bundle

func init() {
	bundle = kit.NewBundle(kit.LangEN)

	// Load translations from embedded files
	loadEmbeddedTranslations()
}

// loadEmbeddedTranslations loads all translation files from embedded FS.
func loadEmbeddedTranslations() {
	files := []struct {
		lang Language
		path string
	}{
		{kit.LangEN, "en.json"},
		{kit.LangZH, "zh.json"},
		{kit.LangFR, "fr.json"},
		{kit.LangIT, "it.json"},
		{kit.LangJA, "ja.json"},
		{kit.LangDE, "de.json"},
		{kit.LangKO, "ko.json"},
	}

	for _, f := range files {
		data, err := locales.FS.ReadFile(f.path)
		if err != nil {
			continue
		}
		if err := bundle.LoadJSON(f.lang, data); err != nil {
			continue
		}
	}
}

// GetBundle returns the translation bundle.
func GetBundle() *kit.Bundle {
	return bundle
}

// NormalizeLanguage normalizes language code.
// Delegates to i18n-kit's NormalizeLanguage.
func NormalizeLanguage(lang string) Language {
	return kit.NormalizeLanguage(lang)
}

// T returns the translated string for the given key from request context.
// If the key is not found, it returns the key itself.
func T(r *http.Request, key string) string {
	lang := kit.LanguageFromRequest(r)
	return bundle.GetTranslation(lang, key)
}

// Tf returns a formatted translated string from request context.
func Tf(r *http.Request, key string, args ...interface{}) string {
	return fmt.Sprintf(T(r, key), args...)
}

// TWithLang returns the translated string for the given key with specified language.
func TWithLang(lang Language, key string) string {
	return bundle.GetTranslation(lang, key)
}

// TfWithLang returns a formatted translated string with specified language.
func TfWithLang(lang Language, key string, args ...interface{}) string {
	return fmt.Sprintf(TWithLang(lang, key), args...)
}

// GetLanguageFromContext gets the language from the request context.
// Delegates to i18n-kit's LanguageFromRequest.
func GetLanguageFromContext(r *http.Request) Language {
	return kit.LanguageFromRequest(r)
}

// GetLanguageFromContextValue gets the language from context.Context (for scenarios without http.Request).
func GetLanguageFromContextValue(ctx context.Context) Language {
	return kit.LanguageFromContext(ctx)
}

// SetLanguageInContext sets the language in the request context.
// Delegates to i18n-kit's SetLanguageInRequest.
func SetLanguageInContext(r *http.Request, lang Language) *http.Request {
	return kit.SetLanguageInRequest(r, lang)
}
