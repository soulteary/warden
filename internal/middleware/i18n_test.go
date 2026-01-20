package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/warden/internal/i18n"
)

func TestDetectLanguagePriority(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/?lang=fr", http.NoBody)
	req.Header.Set("Accept-Language", "zh-CN, en;q=0.8")

	if got := detectLanguage(req); got != i18n.LangFR {
		t.Fatalf("expected query param to take priority, got %s", got)
	}
}

func TestDetectLanguageDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", http.NoBody)

	if got := detectLanguage(req); got != i18n.LangEN {
		t.Fatalf("expected default language LangEN, got %s", got)
	}
}

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		header   string
		expected i18n.Language
	}{
		{"zh-CN, en;q=0.8", i18n.LangZH},
		{"es-ES, fr-FR;q=0.9", i18n.LangFR},
		{"en-GB, zh;q=0.8", i18n.LangEN},
		{"  zh  , en ", i18n.LangZH},
	}

	for _, tt := range tests {
		if got := parseAcceptLanguage(tt.header); got != tt.expected {
			t.Fatalf("parseAcceptLanguage(%q) = %s, want %s", tt.header, got, tt.expected)
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
	req.Header.Set("Accept-Language", "es-ES, zh-CN;q=0.8")
	rec := httptest.NewRecorder()

	I18nMiddleware()(handler).ServeHTTP(rec, req)

	if gotLang != i18n.LangZH {
		t.Fatalf("expected LangZH from Accept-Language, got %s", gotLang)
	}
}
