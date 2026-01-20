package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/logger"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestLogLevelHandler_GET tests getting current log level
func TestLogLevelHandler_GET(t *testing.T) {
	handler := LogLevelHandler()

	req := httptest.NewRequest("GET", "/log/level", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response, "level", "Should contain level field")
	level, ok := response["level"].(string)
	require.True(t, ok, "level should be string")
	assert.NotEmpty(t, level, "level should not be empty")
}

// TestLogLevelHandler_POST_ValidLevel tests setting valid log level
func TestLogLevelHandler_POST_ValidLevel(t *testing.T) {
	// Save original level
	originalLevel := zerolog.GlobalLevel()
	defer logger.SetLevel(originalLevel)

	handler := LogLevelHandler()

	levels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			requestBody := map[string]string{
				"level": level,
			}
			body, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/log/level", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			assert.Equal(t, "日志级别已更新", response["message"])
			assert.Equal(t, level, response["level"])

			// Verify level is actually set
			currentLevel := zerolog.GlobalLevel()
			expectedLevel, err := zerolog.ParseLevel(level)
			require.NoError(t, err, "Should be able to parse log level")
			assert.Equal(t, expectedLevel, currentLevel, "Log level should be set correctly")
		})
	}
}

// TestLogLevelHandler_POST_InvalidLevel tests setting invalid log level
func TestLogLevelHandler_POST_InvalidLevel(t *testing.T) {
	handler := LogLevelHandler()

	requestBody := map[string]string{
		"level": "invalid_level",
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/log/level", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "无效的日志级别", "Should return error message")
}

// TestLogLevelHandler_POST_InvalidJSON tests invalid JSON request body
func TestLogLevelHandler_POST_InvalidJSON(t *testing.T) {
	handler := LogLevelHandler()

	req := httptest.NewRequest("POST", "/log/level", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "无效的请求体", "Should return error message")
}

// TestLogLevelHandler_POST_CaseInsensitive tests case-insensitive log level
func TestLogLevelHandler_POST_CaseInsensitive(t *testing.T) {
	// Save original level
	originalLevel := zerolog.GlobalLevel()
	defer logger.SetLevel(originalLevel)

	handler := LogLevelHandler()

	requestBody := map[string]string{
		"level": "DEBUG", // Uppercase
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/log/level", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "debug", response["level"], "Should convert to lowercase")
}

// TestLogLevelHandler_InvalidMethod tests invalid HTTP method
func TestLogLevelHandler_InvalidMethod(t *testing.T) {
	handler := LogLevelHandler()

	methods := []string{"PUT", "DELETE", "PATCH", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/log/level", http.NoBody)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "Should return 405")

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			assert.Contains(t, response["error"], "不支持的方法", "Should return error message")
		})
	}
}

// TestLogLevelHandler_POST_EmptyBody tests empty request body
func TestLogLevelHandler_POST_EmptyBody(t *testing.T) {
	handler := LogLevelHandler()

	req := httptest.NewRequest("POST", "/log/level", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestLogLevelHandler_POST_MissingLevel tests missing level field
func TestLogLevelHandler_POST_MissingLevel(t *testing.T) {
	handler := LogLevelHandler()

	// Test empty JSON object (no level field)
	// When there's no level field in JSON, request.Level will be empty string
	requestBody := map[string]string{}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/log/level", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	// Empty string level should return 400
	assert.Equal(t, http.StatusBadRequest, w.Code, "Empty level field should return 400")

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "不能为空", "Should return error message")
}

// TestLogLevelHandler_POST_EmptyLevel tests empty string level
func TestLogLevelHandler_POST_EmptyLevel(t *testing.T) {
	handler := LogLevelHandler()

	// Test level field as empty string
	requestBody := map[string]string{
		"level": "",
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/log/level", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	// Empty string level should return 400
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "日志级别不能为空", "Should return error message")
}
