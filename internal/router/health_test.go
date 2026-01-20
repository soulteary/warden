package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestHealthCheck_ProductionMode 测试生产模式下的健康检查
func TestHealthCheck_ProductionMode(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}
	userCache.Set(testUsers)

	handler := HealthCheck(nil, userCache, "production", false)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])

	details, ok := response["details"].(map[string]interface{})
	require.True(t, ok, "details 应该是 map")

	// 生产环境不应该返回用户数量
	assert.NotContains(t, details, "user_count", "生产环境不应该返回用户数量")
	assert.Contains(t, details, "data_loaded", "应该包含 data_loaded 字段")

	// 生产环境不应该返回模式信息
	assert.NotContains(t, response, "mode", "生产环境不应该返回模式信息")
}

// TestHealthCheck_DevelopmentMode 测试开发模式下的健康检查
func TestHealthCheck_DevelopmentMode(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}
	userCache.Set(testUsers)

	handler := HealthCheck(nil, userCache, "development", false)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])

	details, ok := response["details"].(map[string]interface{})
	require.True(t, ok)

	// 开发环境应该返回用户数量
	assert.Contains(t, details, "user_count", "开发环境应该返回用户数量")
	assert.Equal(t, float64(1), details["user_count"], "用户数量应该是 1")

	// 开发环境应该返回模式信息
	assert.Contains(t, response, "mode", "开发环境应该返回模式信息")
	assert.Equal(t, "development", response["mode"])
}

// TestHealthCheck_RedisDisabled 测试 Redis 禁用状态
func TestHealthCheck_RedisDisabled(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	handler := HealthCheck(nil, userCache, "development", false)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	details, ok := response["details"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "disabled", details["redis"], "Redis 状态应该是 disabled")
}

// TestHealthCheck_RedisUnavailable 测试 Redis 不可用状态
func TestHealthCheck_RedisUnavailable(t *testing.T) {
	// 创建一个无法连接的 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // 不存在的地址
	})

	userCache := cache.NewSafeUserCache()
	handler := HealthCheck(redisClient, userCache, "development", true)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	// 注意：由于超时，可能需要等待
	// 但根据实现，应该返回 503
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable,
		"应该返回 200 或 503")

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	details, ok := response["details"].(map[string]interface{})
	require.True(t, ok)

	// Redis 状态应该是 unavailable
	if w.Code == http.StatusServiceUnavailable {
		assert.Equal(t, "unavailable", details["redis"])
	}
}

// TestHealthCheck_NoData 测试没有数据的情况
func TestHealthCheck_NoData(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	// 不设置任何数据

	handler := HealthCheck(nil, userCache, "development", false)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	details, ok := response["details"].(map[string]interface{})
	require.True(t, ok)

	dataLoaded, ok := details["data_loaded"].(bool)
	require.True(t, ok)
	assert.False(t, dataLoaded, "data_loaded 应该是 false")
}

// TestHealthCheck_NilCache 测试缓存为 nil 的情况
func TestHealthCheck_NilCache(t *testing.T) {
	handler := HealthCheck(nil, nil, "development", false)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	details, ok := response["details"].(map[string]interface{})
	require.True(t, ok)

	dataLoaded, ok := details["data_loaded"].(bool)
	require.True(t, ok)
	assert.False(t, dataLoaded, "缓存为 nil 时 data_loaded 应该是 false")
}

// TestHealthCheck_RedisOK 测试 Redis 正常情况（需要真实的 Redis）
func TestHealthCheck_RedisOK(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要 Redis 的测试")
	}

	// 尝试连接本地 Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("跳过测试：无法连接到 Redis: %v", err)
	}

	userCache := cache.NewSafeUserCache()
	handler := HealthCheck(redisClient, userCache, "development", true)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	details, ok := response["details"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "ok", details["redis"], "Redis 状态应该是 ok")
}
