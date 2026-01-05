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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"soulteary.com/soulteary/warden/internal/define"
)

func TestFromRemoteConfig_Success(t *testing.T) {
	// 创建模拟HTTP服务器
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

	// 测试从远程配置读取
	ctx := context.Background()
	result, err := FromRemoteConfig(ctx, server.URL, "")

	require.NoError(t, err, "应该成功读取远程配置")
	assert.Len(t, result, 2, "应该读取到2条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
	assert.Equal(t, "remote1@example.com", result[0].Mail)
}

func TestFromRemoteConfig_WithAuthorization(t *testing.T) {
	// 创建需要授权的模拟服务器
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
		_ = json.NewEncoder(w).Encode(testData)
	}))
	defer server.Close()

	// 测试带授权的请求
	ctx := context.Background()
	result, err := FromRemoteConfig(ctx, server.URL, "Bearer test-token")

	require.NoError(t, err, "应该成功读取远程配置")
	assert.Len(t, result, 1, "应该读取到1条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
}

func TestFromRemoteConfig_InvalidURL(t *testing.T) {
	// 测试无效URL
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := FromRemoteConfig(ctx, "http://invalid-url-that-does-not-exist-12345.com/config", "")

	assert.Error(t, err, "无效URL应该返回错误")
	assert.Empty(t, result, "无效URL应该返回空切片")
}

func TestFromRemoteConfig_InvalidJSON(t *testing.T) {
	// 创建返回无效JSON的服务器
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
	// 创建返回空响应的服务器
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
	// 创建临时本地文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "local.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0600)
	require.NoError(t, err)

	// 创建模拟远程服务器
	remoteData := []define.AllowListUser{
		{Phone: "13800138001", Mail: "remote@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(remoteData)
	}))
	defer server.Close()

	// 测试DEFAULT模式（应该等同于REMOTE_FIRST）
	ctx := context.Background()
	result := GetRules(ctx, testFile, server.URL, "", "DEFAULT")

	// 应该包含远程和本地的数据（远程优先，本地补充）
	assert.GreaterOrEqual(t, len(result), 1, "应该至少有一条记录")
}

func TestGetRules_ONLY_REMOTE(t *testing.T) {
	// 创建模拟远程服务器
	remoteData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "remote@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(remoteData)
	}))
	defer server.Close()

	// 测试ONLY_REMOTE模式
	ctx := context.Background()
	result := GetRules(ctx, "", server.URL, "", "ONLY_REMOTE")

	assert.Len(t, result, 1, "应该只有远程数据")
	assert.Equal(t, "13800138000", result[0].Phone)
}

func TestGetRules_ONLY_LOCAL(t *testing.T) {
	// 创建临时本地文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "local-only.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
		{Phone: "13800138001", Mail: "local2@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0600)
	require.NoError(t, err)

	// 测试ONLY_LOCAL模式
	ctx := context.Background()
	result := GetRules(ctx, testFile, "", "", "ONLY_LOCAL")

	assert.Len(t, result, 2, "应该只有本地数据")

	// 由于使用map存储，顺序不确定，所以检查数据是否存在而不是检查顺序
	phones := make(map[string]bool)
	for _, r := range result {
		phones[r.Phone] = true
	}
	assert.True(t, phones["13800138000"], "应该包含13800138000")
	assert.True(t, phones["13800138001"], "应该包含13800138001")
}

func TestGetRules_REMOTE_FIRST(t *testing.T) {
	// 创建临时本地文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "remote-first.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
		{Phone: "13800138002", Mail: "local-only@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0600)
	require.NoError(t, err)

	// 创建模拟远程服务器
	remoteData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "remote@example.com"}, // 与本地重复
		{Phone: "13800138001", Mail: "remote-only@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(remoteData)
	}))
	defer server.Close()

	// 测试REMOTE_FIRST模式
	ctx := context.Background()
	result := GetRules(ctx, testFile, server.URL, "", "REMOTE_FIRST")

	// 应该包含远程数据（优先）和本地独有的数据
	assert.GreaterOrEqual(t, len(result), 2, "应该至少包含远程数据")

	// 验证远程数据存在
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
	// 创建临时本地文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "local-first.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
		{Phone: "13800138002", Mail: "local-only@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0600)
	require.NoError(t, err)

	// 创建模拟远程服务器
	remoteData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "remote@example.com"}, // 与本地重复
		{Phone: "13800138001", Mail: "remote-only@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(remoteData)
	}))
	defer server.Close()

	// 测试LOCAL_FIRST模式
	ctx := context.Background()
	result := GetRules(ctx, testFile, server.URL, "", "LOCAL_FIRST")

	// 应该包含本地数据（优先）和远程独有的数据
	assert.GreaterOrEqual(t, len(result), 2, "应该至少包含本地数据")

	// 验证本地数据存在
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
	// 测试无效模式（应该使用默认行为REMOTE_FIRST）
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid-mode.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0600)
	require.NoError(t, err)

	// 测试无效模式，使用无效的远程URL
	// 根据代码逻辑，无效模式会使用REMOTE_FIRST模式，且allowSkipRemoteFailed=false
	// 当远程获取失败时，会返回空结果（因为不允许跳过远程失败）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := GetRules(ctx, testFile, "http://invalid-url-that-does-not-exist-12345.com/config", "", "INVALID_MODE")

	// 当远程配置失败且不允许跳过时，应该返回空结果
	// 这是预期的行为，因为REMOTE_FIRST模式在远程失败且不允许跳过时会返回空
	assert.Empty(t, result, "无效模式使用REMOTE_FIRST，远程失败且不允许跳过时应返回空结果")
}

func TestGetRules_InvalidMode_WithAllowRemoteFailed(t *testing.T) {
	// 测试无效模式，但使用允许远程失败的场景
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid-mode-allow-failed.json")

	localData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "local@example.com"},
	}

	data, err := json.Marshal(localData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0600)
	require.NoError(t, err)

	// 测试无效模式，但使用一个会失败的远程URL
	// 由于无效模式使用REMOTE_FIRST且allowSkipRemoteFailed=false，远程失败会返回空
	// 但我们可以测试REMOTE_FIRST_ALLOW_REMOTE_FAILED模式来验证允许失败时的行为
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := GetRules(ctx, testFile, "http://invalid-url-that-does-not-exist-12345.com/config", "", "REMOTE_FIRST_ALLOW_REMOTE_FAILED")

	// 当允许远程失败时，应该返回本地数据
	assert.NotEmpty(t, result, "允许远程失败时应该返回本地数据")
	assert.Len(t, result, 1, "应该返回1条本地记录")
	assert.Equal(t, "13800138000", result[0].Phone)
}
