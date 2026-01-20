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

// TestLogLevelHandler_GET 测试获取当前日志级别
func TestLogLevelHandler_GET(t *testing.T) {
	handler := LogLevelHandler()

	req := httptest.NewRequest("GET", "/log/level", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response, "level", "应该包含 level 字段")
	level, ok := response["level"].(string)
	require.True(t, ok, "level 应该是字符串")
	assert.NotEmpty(t, level, "level 不应该为空")
}

// TestLogLevelHandler_POST_ValidLevel 测试设置有效的日志级别
func TestLogLevelHandler_POST_ValidLevel(t *testing.T) {
	// 保存原始级别
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

			// 验证级别确实被设置
			currentLevel := zerolog.GlobalLevel()
			expectedLevel, _ := zerolog.ParseLevel(level)
			assert.Equal(t, expectedLevel, currentLevel, "日志级别应该被正确设置")
		})
	}
}

// TestLogLevelHandler_POST_InvalidLevel 测试设置无效的日志级别
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

	assert.Contains(t, response["error"], "无效的日志级别", "应该返回错误消息")
}

// TestLogLevelHandler_POST_InvalidJSON 测试无效的 JSON 请求体
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

	assert.Contains(t, response["error"], "无效的请求体", "应该返回错误消息")
}

// TestLogLevelHandler_POST_CaseInsensitive 测试大小写不敏感的日志级别
func TestLogLevelHandler_POST_CaseInsensitive(t *testing.T) {
	// 保存原始级别
	originalLevel := zerolog.GlobalLevel()
	defer logger.SetLevel(originalLevel)

	handler := LogLevelHandler()

	requestBody := map[string]string{
		"level": "DEBUG", // 大写
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

	assert.Equal(t, "debug", response["level"], "应该转换为小写")
}

// TestLogLevelHandler_InvalidMethod 测试无效的 HTTP 方法
func TestLogLevelHandler_InvalidMethod(t *testing.T) {
	handler := LogLevelHandler()

	methods := []string{"PUT", "DELETE", "PATCH", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/log/level", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "应该返回 405")

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			assert.Contains(t, response["error"], "不支持的方法", "应该返回错误消息")
		})
	}
}

// TestLogLevelHandler_POST_EmptyBody 测试空请求体
func TestLogLevelHandler_POST_EmptyBody(t *testing.T) {
	handler := LogLevelHandler()

	req := httptest.NewRequest("POST", "/log/level", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestLogLevelHandler_POST_MissingLevel 测试缺少 level 字段
func TestLogLevelHandler_POST_MissingLevel(t *testing.T) {
	handler := LogLevelHandler()

	// 测试空 JSON 对象（没有 level 字段）
	// 当 JSON 中没有 level 字段时，request.Level 会是空字符串
	requestBody := map[string]string{}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/log/level", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	// 空字符串的 level 应该返回 400
	assert.Equal(t, http.StatusBadRequest, w.Code, "空 level 字段应该返回 400")

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "不能为空", "应该返回错误消息")
}

// TestLogLevelHandler_POST_EmptyLevel 测试空字符串的 level
func TestLogLevelHandler_POST_EmptyLevel(t *testing.T) {
	handler := LogLevelHandler()

	// 测试 level 字段为空字符串
	requestBody := map[string]string{
		"level": "",
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/log/level", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	// 空字符串的 level 应该返回 400
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "日志级别不能为空", "应该返回错误消息")
}
