package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/warden/internal/i18n"
)

func TestDetectLanguagePriority(t *testing.T) {
	// Test that query param takes priority over Accept-Language header
	var gotLang i18n.Language

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLang = GetLanguage(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/?lang=fr", http.NoBody)
	req.Header.Set("Accept-Language", "zh-CN, en;q=0.8")
	rec := httptest.NewRecorder()

	I18nMiddleware()(handler).ServeHTTP(rec, req)

	if gotLang != i18n.LangFR {
		t.Fatalf("expected query param to take priority, got %s", gotLang)
	}
}

func TestDetectLanguageDefault(t *testing.T) {
	// Test default language when no language hints are provided
	var gotLang i18n.Language

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLang = GetLanguage(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", http.NoBody)
	rec := httptest.NewRecorder()

	I18nMiddleware()(handler).ServeHTTP(rec, req)

	if gotLang != i18n.LangEN {
		t.Fatalf("expected default language LangEN, got %s", gotLang)
	}
}

func TestParseAcceptLanguage(t *testing.T) {
	// Test Accept-Language header parsing through middleware
	// Note: i18n-kit parses Accept-Language from left to right, picking the first supported language
	tests := []struct {
		header   string
		expected i18n.Language
	}{
		{"zh-CN, en;q=0.8", i18n.LangZH},
		{"fr-FR, es-ES;q=0.9", i18n.LangFR},
		{"en-GB, zh;q=0.8", i18n.LangEN},
		{"zh, en", i18n.LangZH},
	}

	for _, tt := range tests {
		var gotLang i18n.Language

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotLang = GetLanguage(r)
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "http://example.com/", http.NoBody)
		req.Header.Set("Accept-Language", tt.header)
		rec := httptest.NewRecorder()

		I18nMiddleware()(handler).ServeHTTP(rec, req)

		if gotLang != tt.expected {
			t.Fatalf("Accept-Language %q: got %s, want %s", tt.header, gotLang, tt.expected)
		}
	}
}

func TestI18nMiddlewareSetsLanguage(t *testing.T) {
	var gotLang i18n.Language

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLang = GetLanguage(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/?lang=ja", http.NoBody)
	rec := httptest.NewRecorder()

	I18nMiddleware()(handler).ServeHTTP(rec, req)

	if gotLang != i18n.LangJA {
		t.Fatalf("expected LangJA from middleware, got %s", gotLang)
	}
}

func TestI18nMiddlewareUsesAcceptLanguage(t *testing.T) {
	var gotLang i18n.Language

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLang = GetLanguage(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", http.NoBody)
	// i18n-kit picks the first supported language from Accept-Language
	req.Header.Set("Accept-Language", "zh-CN, es-ES;q=0.8")
	rec := httptest.NewRecorder()

	I18nMiddleware()(handler).ServeHTTP(rec, req)

	if gotLang != i18n.LangZH {
		t.Fatalf("expected LangZH from Accept-Language, got %s", gotLang)
	}
}
