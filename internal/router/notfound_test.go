package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/i18n"
)

// TestNotFound covers the catch-all handler directly. It is exercised end-to-end from the
// main package too, but this is the handler that stops an unregistered path from reaching a
// data handler, so it deserves a test in its own package.
func TestNotFound(t *testing.T) {
	handler := NotFound()

	req := httptest.NewRequest(http.MethodGet, "/definitely-not-a-route", http.NoBody)
	resp := httptest.NewRecorder()
	handler(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body), "响应必须是 JSON")
	assert.Equal(t, i18n.TWithLang(i18n.LangEN, "error.not_found"), body["error"])
	assert.Len(t, body, 1, "404 响应体只应包含 error 字段")
}

// TestNotFound_Localized pins that the message follows the request language.
//
// The language comes from the request CONTEXT, which I18nMiddleware populates — not from
// the Accept-Language header the handler sees. That is why the fallback is mounted inside
// i18nMiddleware in registerRoutes: a 404 handler registered outside it would silently
// answer in English no matter what the client asked for, and nothing would fail.
func TestNotFound_Localized(t *testing.T) {
	tests := []struct {
		name string
		lang i18n.Language
	}{
		{"中文", i18n.LangZH},
		{"日本語", i18n.LangJA},
		{"Français", i18n.LangFR},
		{"한국어", i18n.LangKO},
		{"Deutsch", i18n.LangDE},
		{"Italiano", i18n.LangIT},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := i18n.SetLanguageInContext(
				httptest.NewRequest(http.MethodGet, "/nope", http.NoBody), tt.lang)
			resp := httptest.NewRecorder()
			NotFound()(resp, req)

			require.Equal(t, http.StatusNotFound, resp.Code)
			var body map[string]string
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
			assert.Equal(t, i18n.TWithLang(tt.lang, "error.not_found"), body["error"],
				"404 文案应跟随请求语言")
			assert.NotEqual(t, i18n.TWithLang(i18n.LangEN, "error.not_found"), body["error"],
				"不应回落到英文")
		})
	}
}

// TestNotFound_NeverLeaksRequestDetails guards the reason the handler exists: the root
// pattern used to serve the full allow list, so the replacement must return no data and
// must not echo attacker-controlled input back to the caller.
func TestNotFound_NeverLeaksRequestDetails(t *testing.T) {
	const probe = "warden-probe-2c7f<script>"

	req := httptest.NewRequest(http.MethodGet, "/"+probe+"?q="+probe, http.NoBody)
	resp := httptest.NewRecorder()
	NotFound()(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
	assert.NotContains(t, resp.Body.String(), probe, "响应不得回显请求路径或查询串")
	assert.NotContains(t, resp.Body.String(), "@", "响应不得包含任何用户数据")
}

// TestNotFound_AnyMethod confirms the fallback answers uniformly: the mux routes every
// method to it, so a non-GET request must get the same 404 rather than a 405 that would
// reveal which paths are real.
func TestNotFound_AnyMethod(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodHead} {
		req := httptest.NewRequest(method, "/nope", http.NoBody)
		resp := httptest.NewRecorder()
		NotFound()(resp, req)
		assert.Equal(t, http.StatusNotFound, resp.Code, "方法 %s 应同样返回 404", method)
	}
}
