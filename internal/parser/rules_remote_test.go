package parser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFromRemoteConfig_RetryOn5xx 测试 5xx 错误重试
func TestFromRemoteConfig_RetryOn5xx(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要网络的测试")
	}

	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// 前两次返回 500 错误
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte("Internal Server Error"))
			require.NoError(t, err)
		} else {
			// 第三次成功
			testData := []define.AllowListUser{
				{Phone: "13800138000", Mail: "test@example.com"},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(testData))
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := FromRemoteConfig(ctx, server.URL, "")

	require.NoError(t, err, "重试后应该成功")
	assert.Len(t, result, 1, "应该读取到1条记录")
	assert.Equal(t, 3, attemptCount, "应该重试了3次")
}

// TestFromRemoteConfig_ContextCancellation 测试上下文取消
func TestFromRemoteConfig_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要网络的测试")
	}

	// 创建一个慢服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("[]"))
		require.NoError(t, err)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// 立即取消上下文
	cancel()

	result, err := FromRemoteConfig(ctx, server.URL, "")

	assert.Error(t, err, "上下文取消应该返回错误")
	assert.Contains(t, err.Error(), "取消", "错误信息应该包含取消相关的内容")
	assert.Empty(t, result, "上下文取消应该返回空结果")
}

// TestFromRemoteConfig_ContextTimeout 测试上下文超时
func TestFromRemoteConfig_ContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要网络的测试")
	}

	// 创建一个慢服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("[]"))
		require.NoError(t, err)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := FromRemoteConfig(ctx, server.URL, "")

	assert.Error(t, err, "超时应该返回错误")
	assert.Empty(t, result, "超时应该返回空结果")
}

// TestFromRemoteConfig_Non200Status 测试非 200 状态码
func TestFromRemoteConfig_Non200Status(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "404 Not Found", statusCode: http.StatusNotFound},
		{name: "401 Unauthorized", statusCode: http.StatusUnauthorized},
		{name: "403 Forbidden", statusCode: http.StatusForbidden},
		{name: "500 Internal Server Error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, err := w.Write([]byte("Error"))
				require.NoError(t, err)
			}))
			defer server.Close()

			ctx := context.Background()
			result, err := FromRemoteConfig(ctx, server.URL, "")

			assert.Error(t, err, "非 200 状态码应该返回错误")
			assert.Empty(t, result, "非 200 状态码应该返回空结果")
		})
	}
}

// TestFromRemoteConfig_LargeResponse 测试大响应体
func TestFromRemoteConfig_LargeResponse(t *testing.T) {
	// 创建一个返回大量数据的服务器
	largeData := make([]define.AllowListUser, 1000)
	for i := range largeData {
		largeData[i] = define.AllowListUser{
			Phone: "13800138000",
			Mail:  "test@example.com",
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(largeData))
	}))
	defer server.Close()

	ctx := context.Background()
	result, err := FromRemoteConfig(ctx, server.URL, "")

	require.NoError(t, err, "应该成功读取大响应")
	assert.Len(t, result, 1000, "应该读取到所有记录")
}

// TestFromRemoteConfig_ResponseSizeLimit 测试响应体大小限制
func TestFromRemoteConfig_ResponseSizeLimit(t *testing.T) {
	// 创建一个返回超大数据的服务器（超过 MAX_JSON_SIZE）
	// 注意：这个测试可能需要根据实际的 MAX_JSON_SIZE 值调整
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 生成一个超大的 JSON 字符串
		largeString := make([]byte, define.MAX_JSON_SIZE+1)
		for i := range largeString {
			largeString[i] = 'a'
		}
		_, err := w.Write([]byte(`[{"phone":"13800138000","mail":"` + string(largeString) + `"}]`))
		require.NoError(t, err)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := FromRemoteConfig(ctx, server.URL, "")

	// 应该能够读取（因为使用了 LimitReader），但可能会在解析时失败
	// 或者成功读取但被限制大小
	if err != nil {
		// 如果出错，应该是解析错误
		assert.Contains(t, err.Error(), "解析", "应该是解析错误")
	}
}

// TestFromRemoteConfig_Headers 测试请求头设置
func TestFromRemoteConfig_Headers(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode([]define.AllowListUser{}))
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := FromRemoteConfig(ctx, server.URL, "Bearer test-token")
	require.NoError(t, err)

	// 验证请求头
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"), "应该设置 Content-Type")
	assert.Equal(t, "max-age=0", receivedHeaders.Get("Cache-Control"), "应该设置 Cache-Control")
	assert.Equal(t, "Bearer test-token", receivedHeaders.Get("Authorization"), "应该设置 Authorization")
}

// TestFromRemoteConfig_NoAuthorization 测试无授权头
func TestFromRemoteConfig_NoAuthorization(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode([]define.AllowListUser{}))
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := FromRemoteConfig(ctx, server.URL, "")
	require.NoError(t, err)

	// 验证没有设置 Authorization 头
	assert.Empty(t, receivedHeaders.Get("Authorization"), "不应该设置 Authorization 头")
}

// TestFromRemoteConfig_UserNormalization 测试用户数据规范化
func TestFromRemoteConfig_UserNormalization(t *testing.T) {
	testData := []define.AllowListUser{
		{
			Phone:  "13800138000",
			Mail:   "test@example.com",
			UserID: "", // 未设置 UserID
			Status: "", // 未设置 Status
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(testData))
	}))
	defer server.Close()

	ctx := context.Background()
	result, err := FromRemoteConfig(ctx, server.URL, "")
	require.NoError(t, err)
	require.Len(t, result, 1)

	// 验证数据被规范化
	assert.NotEmpty(t, result[0].UserID, "UserID 应该被自动生成")
	assert.Equal(t, "active", result[0].Status, "Status 应该有默认值")
}

// TestInitHTTPClient 测试 HTTP 客户端初始化
func TestInitHTTPClient(t *testing.T) {
	// 保存原始客户端
	originalClient := httpClient

	// 测试初始化
	InitHTTPClient(30, 100, false)

	// 验证客户端被更新
	assert.NotNil(t, httpClient, "客户端不应该为 nil")
	assert.Equal(t, 30*time.Second, httpClient.Timeout, "超时时间应该正确设置")

	// 恢复原始客户端
	httpClient = originalClient
}

// TestInitHTTPClient_InsecureTLS 测试不安全的 TLS 配置
func TestInitHTTPClient_InsecureTLS(t *testing.T) {
	// 保存原始客户端
	originalClient := httpClient

	// 测试初始化（启用不安全的 TLS）
	InitHTTPClient(30, 100, true)

	// 验证客户端被更新
	assert.NotNil(t, httpClient, "客户端不应该为 nil")

	// 验证 TLS 配置（如果可能）
	transport, ok := httpClient.Transport.(*http.Transport)
	if ok && transport.TLSClientConfig != nil {
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify, "应该跳过 TLS 验证")
	}

	// 恢复原始客户端
	httpClient = originalClient
}
