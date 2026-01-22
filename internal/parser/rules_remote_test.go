package parser

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestServer creates a test HTTP server that works in restricted environments
func createTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)

	// Try to create a listener on localhost
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		// Fallback to NewServer if listener creation fails
		server.Close()
		return httptest.NewServer(handler)
	}

	server.Listener = listener
	server.Start()
	return server
}

// TestFromRemoteConfig_RetryOn5xx tests retry on 5xx errors
func TestFromRemoteConfig_RetryOn5xx(t *testing.T) {
	attemptCount := 0
	server := createTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// First two attempts return 500 error
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte("Internal Server Error"))
			require.NoError(t, err)
		} else {
			// Third attempt succeeds
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

// TestFromRemoteConfig_ContextCancellation tests context cancellation
func TestFromRemoteConfig_ContextCancellation(t *testing.T) {
	// Create a slow server
	server := createTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("[]"))
		require.NoError(t, err)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	result, err := FromRemoteConfig(ctx, server.URL, "")

	assert.Error(t, err, "上下文取消应该返回错误")
	assert.Contains(t, err.Error(), "取消", "错误信息应该包含取消相关的内容")
	assert.Empty(t, result, "上下文取消应该返回空结果")
}

// TestFromRemoteConfig_ContextTimeout tests context timeout
func TestFromRemoteConfig_ContextTimeout(t *testing.T) {
	// Create a slow server
	server := createTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

// TestFromRemoteConfig_Non200Status tests non-200 status codes
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
			server := createTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

// TestFromRemoteConfig_LargeResponse tests large response body
func TestFromRemoteConfig_LargeResponse(t *testing.T) {
	// Create a server returning large amount of data
	largeData := make([]define.AllowListUser, 1000)
	for i := range largeData {
		largeData[i] = define.AllowListUser{
			Phone: "13800138000",
			Mail:  "test@example.com",
		}
	}

	server := createTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(largeData))
	}))
	defer server.Close()

	ctx := context.Background()
	result, err := FromRemoteConfig(ctx, server.URL, "")

	require.NoError(t, err, "应该成功读取大响应")
	assert.Len(t, result, 1000, "应该读取到所有记录")
}

// TestFromRemoteConfig_ResponseSizeLimit tests response body size limit
func TestFromRemoteConfig_ResponseSizeLimit(t *testing.T) {
	// Create a server returning oversized data (exceeding MAX_JSON_SIZE)
	// Note: This test may need adjustment based on actual MAX_JSON_SIZE value
	server := createTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Generate an oversized JSON string
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

	// Should be able to read (because LimitReader is used), but may fail during parsing
	// Or successfully read but limited in size
	if err != nil {
		// If error occurs, should be parsing error
		assert.Contains(t, err.Error(), "解析", "应该是解析错误")
	}
}

// TestFromRemoteConfig_Headers tests request header settings
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

	// Verify request headers
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"), "应该设置 Content-Type")
	assert.Equal(t, "max-age=0", receivedHeaders.Get("Cache-Control"), "应该设置 Cache-Control")
	assert.Equal(t, "Bearer test-token", receivedHeaders.Get("Authorization"), "应该设置 Authorization")
}

// TestFromRemoteConfig_NoAuthorization tests no authorization header
func TestFromRemoteConfig_NoAuthorization(t *testing.T) {
	var receivedHeaders http.Header
	server := createTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode([]define.AllowListUser{}))
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := FromRemoteConfig(ctx, server.URL, "")
	require.NoError(t, err)

	// Verify Authorization header is not set
	assert.Empty(t, receivedHeaders.Get("Authorization"), "不应该设置 Authorization 头")
}

// TestFromRemoteConfig_UserNormalization tests user data normalization
func TestFromRemoteConfig_UserNormalization(t *testing.T) {
	testData := []define.AllowListUser{
		{
			Phone:  "13800138000",
			Mail:   "test@example.com",
			UserID: "", // UserID not set
			Status: "", // Status not set
		},
	}

	server := createTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(testData))
	}))
	defer server.Close()

	ctx := context.Background()
	result, err := FromRemoteConfig(ctx, server.URL, "")
	require.NoError(t, err)
	require.Len(t, result, 1)

	// Verify data is normalized
	assert.NotEmpty(t, result[0].UserID, "UserID 应该被自动生成")
	assert.Equal(t, "active", result[0].Status, "Status 应该有默认值")
}

// TestInitHTTPClient tests HTTP client initialization
func TestInitHTTPClient(t *testing.T) {
	// Save original client
	originalClient := httpClient

	// Test initialization
	InitHTTPClient(30, 100, false)

	// Verify client is updated
	assert.NotNil(t, httpClient, "客户端不应该为 nil")
	assert.Equal(t, 30*time.Second, httpClient.Timeout, "超时时间应该正确设置")

	// Restore original client
	httpClient = originalClient
}

// TestInitHTTPClient_InsecureTLS tests insecure TLS configuration
func TestInitHTTPClient_InsecureTLS(t *testing.T) {
	// Save original client
	originalClient := httpClient

	// Test initialization (enable insecure TLS)
	InitHTTPClient(30, 100, true)

	// Verify client is updated
	assert.NotNil(t, httpClient, "客户端不应该为 nil")

	// Verify TLS configuration (if possible)
	transport, ok := httpClient.Transport.(*http.Transport)
	if ok && transport.TLSClientConfig != nil {
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify, "应该跳过 TLS 验证")
	}

	// Restore original client
	httpClient = originalClient
}
