package parser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromRemoteConfig_Success(t *testing.T) {
	// Create mock HTTP server
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "remote1@example.com"},
		{Phone: "13800138001", Mail: "remote2@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(testData); err != nil {
			t.Errorf("编码JSON失败: %v", err)
		}
	}))
	defer server.Close()

	// Test reading from remote config
	ctx := context.Background()
	result, err := FromRemoteConfig(ctx, server.URL, "")

	require.NoError(t, err, "应该成功读取远程配置")
	assert.Len(t, result, 2, "应该读取到2条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
	assert.Equal(t, "remote1@example.com", result[0].Mail)
}

func TestFromRemoteConfig_WithAuthorization(t *testing.T) {
	// Create mock server requiring authorization
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "auth@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(testData); err != nil {
			t.Errorf("编码JSON失败: %v", err)
		}
	}))
	defer server.Close()

	// Test request with authorization
	ctx := context.Background()
	result, err := FromRemoteConfig(ctx, server.URL, "Bearer test-token")

	require.NoError(t, err, "应该成功读取远程配置")
	assert.Len(t, result, 1, "应该读取到1条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
}

func TestFromRemoteConfig_InvalidURL(t *testing.T) {
	// Test invalid URL
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := FromRemoteConfig(ctx, "http://invalid-url-that-does-not-exist-12345.com/config", "")

	assert.Error(t, err, "无效URL应该返回错误")
	assert.Empty(t, result, "无效URL应该返回空切片")
}

func TestFromRemoteConfig_InvalidJSON(t *testing.T) {
	// Create server returning invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("invalid json")); err != nil {
			t.Errorf("写入响应失败: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	result, err := FromRemoteConfig(ctx, server.URL, "")

	assert.Error(t, err, "无效JSON应该返回错误")
	assert.Empty(t, result, "无效JSON应该返回空切片")
}

func TestFromRemoteConfig_EmptyResponse(t *testing.T) {
	// Create server returning empty response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("[]")); err != nil {
			t.Errorf("写入响应失败: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	result, err := FromRemoteConfig(ctx, server.URL, "")

	require.NoError(t, err, "空响应应该成功解析")
	assert.Empty(t, result, "空响应应该返回空切片")
}

func TestGetRules_DEFAULT(t *testing.T) {
	// Create temporary local file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "local.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0o600)
	require.NoError(t, err)

	// Create mock remote server
	remoteData := []define.AllowListUser{
		{Phone: "13800138001", Mail: "remote@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(remoteData); err != nil {
			t.Errorf("编码JSON失败: %v", err)
		}
	}))
	defer server.Close()

	// Test DEFAULT mode (should be equivalent to REMOTE_FIRST)
	ctx := context.Background()
	result := GetRules(ctx, testFile, server.URL, "", "DEFAULT")

	// Should contain both remote and local data (remote first, local supplement)
	assert.GreaterOrEqual(t, len(result), 1, "应该至少有一条记录")
}

func TestGetRules_ONLY_REMOTE(t *testing.T) {
	// Create mock remote server
	remoteData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "remote@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(remoteData); err != nil {
			t.Errorf("编码JSON失败: %v", err)
		}
	}))
	defer server.Close()

	// Test ONLY_REMOTE mode
	ctx := context.Background()
	result := GetRules(ctx, "", server.URL, "", "ONLY_REMOTE")

	assert.Len(t, result, 1, "应该只有远程数据")
	assert.Equal(t, "13800138000", result[0].Phone)
}

func TestGetRules_ONLY_LOCAL(t *testing.T) {
	// Create temporary local file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "local-only.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
		{Phone: "13800138001", Mail: "local2@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0o600)
	require.NoError(t, err)

	// Test ONLY_LOCAL mode
	ctx := context.Background()
	result := GetRules(ctx, testFile, "", "", "ONLY_LOCAL")

	assert.Len(t, result, 2, "应该只有本地数据")

	// Since map storage is used, order is uncertain, so check if data exists rather than checking order
	phones := make(map[string]bool)
	for _, r := range result {
		phones[r.Phone] = true
	}
	assert.True(t, phones["13800138000"], "应该包含13800138000")
	assert.True(t, phones["13800138001"], "应该包含13800138001")
}

func TestGetRules_REMOTE_FIRST(t *testing.T) {
	// Create temporary local file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "remote-first.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
		{Phone: "13800138002", Mail: "local-only@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0o600)
	require.NoError(t, err)

	// Create mock remote server
	remoteData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "remote@example.com"}, // Duplicate with local
		{Phone: "13800138001", Mail: "remote-only@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(remoteData); err != nil {
			t.Errorf("编码JSON失败: %v", err)
		}
	}))
	defer server.Close()

	// Test REMOTE_FIRST mode
	ctx := context.Background()
	result := GetRules(ctx, testFile, server.URL, "", "REMOTE_FIRST")

	// Should contain remote data (priority) and local-only data
	assert.GreaterOrEqual(t, len(result), 2, "应该至少包含远程数据")

	// Verify remote data exists
	hasRemote := false
	for _, r := range result {
		if r.Phone == "13800138001" {
			hasRemote = true
			break
		}
	}
	assert.True(t, hasRemote, "应该包含远程数据")
}

func TestGetRules_LOCAL_FIRST(t *testing.T) {
	// Create temporary local file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "local-first.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
		{Phone: "13800138002", Mail: "local-only@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0o600)
	require.NoError(t, err)

	// Create mock remote server
	remoteData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "remote@example.com"}, // Duplicate with local
		{Phone: "13800138001", Mail: "remote-only@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(remoteData); err != nil {
			t.Errorf("编码JSON失败: %v", err)
		}
	}))
	defer server.Close()

	// Test LOCAL_FIRST mode
	ctx := context.Background()
	result := GetRules(ctx, testFile, server.URL, "", "LOCAL_FIRST")

	// Should contain local data (priority) and remote-only data
	assert.GreaterOrEqual(t, len(result), 2, "应该至少包含本地数据")

	// Verify local data exists
	hasLocal := false
	for _, r := range result {
		if r.Phone == "13800138002" {
			hasLocal = true
			break
		}
	}
	assert.True(t, hasLocal, "应该包含本地数据")
}

func TestGetRules_InvalidMode(t *testing.T) {
	// Test invalid mode (should use default behavior REMOTE_FIRST)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid-mode.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0o600)
	require.NoError(t, err)

	// Test invalid mode with invalid remote URL
	// According to code logic, invalid mode will use REMOTE_FIRST mode, and allowSkipRemoteFailed=false
	// When remote fetch fails, will return empty result (because skipping remote failure is not allowed)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := GetRules(ctx, testFile, "http://invalid-url-that-does-not-exist-12345.com/config", "", "INVALID_MODE")

	// When remote config fails and skipping is not allowed, should return empty result
	// This is expected behavior, as REMOTE_FIRST mode returns empty when remote fails and skipping is not allowed
	assert.Empty(t, result, "无效模式使用REMOTE_FIRST，远程失败且不允许跳过时应返回空结果")
}

func TestGetRules_InvalidMode_WithAllowRemoteFailed(t *testing.T) {
	// Test invalid mode, but with scenario allowing remote failure
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid-mode-allow-failed.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0o600)
	require.NoError(t, err)

	// Test invalid mode, but with a remote URL that will fail
	// Since invalid mode uses REMOTE_FIRST and allowSkipRemoteFailed=false, remote failure will return empty
	// But we can test REMOTE_FIRST_ALLOW_REMOTE_FAILED mode to verify behavior when failure is allowed
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := GetRules(ctx, testFile, "http://invalid-url-that-does-not-exist-12345.com/config", "", "REMOTE_FIRST_ALLOW_REMOTE_FAILED")

	// When remote failure is allowed, should return local data
	assert.NotEmpty(t, result, "允许远程失败时应该返回本地数据")
	assert.Len(t, result, 1, "应该返回1条本地记录")
	assert.Equal(t, "13800138000", result[0].Phone)
}
