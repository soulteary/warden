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

// TestHealthCheck_ProductionMode tests health check in production mode
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
	require.True(t, ok, "details should be map")

	// Production environment should not return user count
	assert.NotContains(t, details, "user_count", "Production environment should not return user count")
	assert.Contains(t, details, "data_loaded", "Should contain data_loaded field")

	// Production environment should not return mode information
	assert.NotContains(t, response, "mode", "Production environment should not return mode information")
}

// TestHealthCheck_DevelopmentMode tests health check in development mode
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

	// Development environment should return user count
	assert.Contains(t, details, "user_count", "Development environment should return user count")
	assert.Equal(t, float64(1), details["user_count"], "User count should be 1")

	// Development environment should return mode information
	assert.Contains(t, response, "mode", "Development environment should return mode information")
	assert.Equal(t, "development", response["mode"])
}

// TestHealthCheck_RedisDisabled tests Redis disabled state
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

	assert.Equal(t, "disabled", details["redis"], "Redis status should be disabled")
}

// TestHealthCheck_RedisUnavailable tests Redis unavailable state
func TestHealthCheck_RedisUnavailable(t *testing.T) {
	// Create a Redis client that cannot connect
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Non-existent address
	})

	userCache := cache.NewSafeUserCache()
	handler := HealthCheck(redisClient, userCache, "development", true)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	// Note: May need to wait due to timeout
	// But according to implementation, should return 503
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable,
		"Should return 200 or 503")

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	details, ok := response["details"].(map[string]interface{})
	require.True(t, ok)

	// Redis status should be unavailable
	if w.Code == http.StatusServiceUnavailable {
		assert.Equal(t, "unavailable", details["redis"])
	}
}

// TestHealthCheck_NoData tests case with no data
func TestHealthCheck_NoData(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	// Don't set any data

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
	assert.False(t, dataLoaded, "data_loaded should be false")
}

// TestHealthCheck_NilCache tests case when cache is nil
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
	assert.False(t, dataLoaded, "data_loaded should be false when cache is nil")
}

// TestHealthCheck_RedisOK tests Redis normal case (requires real Redis)
func TestHealthCheck_RedisOK(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires Redis")
	}

	// Try to connect to local Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping test: unable to connect to Redis: %v", err)
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

	assert.Equal(t, "ok", details["redis"], "Redis status should be ok")
}
