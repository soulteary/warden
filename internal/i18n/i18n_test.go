package i18n

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextLanguage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", http.NoBody)

	if got := GetLanguageFromContext(req); got != LangEN {
		t.Fatalf("expected default language LangEN, got %s", got)
	}

	req = SetLanguageInContext(req, LangZH)
	if got := GetLanguageFromContext(req); got != LangZH {
		t.Fatalf("expected LangZH from request context, got %s", got)
	}

	if got := GetLanguageFromContextValue(req.Context()); got != LangZH {
		t.Fatalf("expected LangZH from context value, got %s", got)
	}

	if got := GetLanguageFromContext(nil); got != LangEN {
		t.Fatalf("expected LangEN for nil request, got %s", got)
	}
}

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected Language
	}{
		{"zh", LangZH},
		{"zh-cn", LangZH},
		{"fr_fr", LangFR},
		{" it-it ", LangIT},
		{"ja-jp", LangJA},
		{"de_de", LangDE},
		{"ko-kr", LangKO},
		{"en-us", LangEN},
		{"unknown", LangEN},
	}

	for _, tt := range tests {
		if got := NormalizeLanguage(tt.input); got != tt.expected {
			t.Fatalf("NormalizeLanguage(%q) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestTranslateFallbacks(t *testing.T) {
	key := "error.redis_connection_failed"
	if got := TWithLang(LangEN, key); got != "Redis connection failed" {
		t.Fatalf("expected English translation, got %q", got)
	}

	if got := TWithLang(Language("es"), key); got != "Redis connection failed" {
		t.Fatalf("expected English fallback translation, got %q", got)
	}

	unknownKey := "unknown.key"
	if got := TWithLang(LangEN, unknownKey); got != unknownKey {
		t.Fatalf("expected key fallback, got %q", got)
	}
}

func TestTranslateHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", http.NoBody)
	req = SetLanguageInContext(req, LangEN)

	if got := T(req, "error.redis_connection_failed"); got != "Redis connection failed" {
		t.Fatalf("expected T() to return English translation, got %q", got)
	}

	expected := "Invalid port number: 8080 (must be an integer between 1-65535)"
	if got := Tf(req, "validation.port_invalid", "8080"); got != expected {
		t.Fatalf("Tf() = %q, want %q", got, expected)
	}

	if got := TWithLang(LangEN, "error.redis_connection_failed"); got != "Redis connection failed" {
		t.Fatalf("expected TWithLang() to return English translation, got %q", got)
	}

	if got := TfWithLang(LangEN, "validation.port_invalid", "8080"); got != expected {
		t.Fatalf("TfWithLang() = %q, want %q", got, expected)
	}
}

func TestGetLanguageFromContextValueWithEmptyContext(t *testing.T) {
	// Test with empty context returns default language
	if got := GetLanguageFromContextValue(context.Background()); got != LangEN {
		t.Fatalf("expected LangEN when context has no language, got %s", got)
	}

	// Test with context.TODO returns default language
	if got := GetLanguageFromContextValue(context.TODO()); got != LangEN {
		t.Fatalf("expected LangEN when context is TODO, got %s", got)
	}
}
