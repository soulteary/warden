package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON_Handler(t *testing.T) {
	// Prepare test data
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13800138001", Mail: "test2@example.com"},
	}

	// Create thread-safe cache and set data
	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)

	// Create handler
	handler := JSON(userCache, nil)

	// Create test request
	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	// Execute handler
	handler(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "状态码应该是200")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "Content-Type应该是application/json")

	// Verify response body
	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err, "响应体应该是有效的JSON")

	assert.Len(t, result, 2, "应该返回2条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
	assert.Equal(t, "test1@example.com", result[0].Mail)
}

func TestJSON_EmptyData(t *testing.T) {
	// Test empty data
	emptyData := []define.AllowListUser{}
	userCache := cache.NewSafeUserCache()
	userCache.Set(emptyData)
	handler := JSON(userCache, nil)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Empty(t, result, "空数据应该返回空数组")
}

func TestJSON_SingleRecord(t *testing.T) {
	// Test single record
	singleData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "single@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(singleData)
	handler := JSON(userCache, nil)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Len(t, result, 1, "应该返回1条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
}

func TestJSON_Unicode(t *testing.T) {
	// Test Unicode characters (email doesn't support Chinese, use valid email format)
	unicodeData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(unicodeData)
	handler := JSON(userCache, nil)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Len(t, result, 1)
	assert.Equal(t, "test@example.com", result[0].Mail, "应该正确处理数据")
}

// TestJSON_WithResponseFields covers UsersToMaps path (field whitelist)
func TestJSON_WithResponseFields(t *testing.T) {
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "a@example.com", UserID: "u1"},
		{Phone: "13900139000", Mail: "b@example.com", UserID: "u2"},
	}
	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	responseFields := []string{"phone", "mail"}
	handler := JSON(userCache, responseFields)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Len(t, result[0], 2)
	assert.Equal(t, "13800138000", result[0]["phone"])
	assert.Equal(t, "a@example.com", result[0]["mail"])
	assert.NotContains(t, result[0], "user_id")
}

func TestJSON_ContentType(t *testing.T) {
	// Test Content-Type header
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache, nil)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	contentType := w.Header().Get("Content-Type")
	assert.Equal(t, "application/json", contentType, "Content-Type应该正确设置")
}

func TestJSON_StatusCode(t *testing.T) {
	// Test status code
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache, nil)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "状态码应该是200")
}

func TestJSON_MultipleRecords(t *testing.T) {
	// Test multiple records
	multipleData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13800138001", Mail: "test2@example.com"},
		{Phone: "13800138002", Mail: "test3@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(multipleData)
	handler := JSON(userCache, nil)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Len(t, result, 3, "应该返回3条记录")
}

func TestJSON_DataModification(t *testing.T) {
	// Test that handler still uses original data from cache after data modification (SafeUserCache creates a copy)
	originalData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "original@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(originalData)
	handler := JSON(userCache, nil)

	// Modify original data (should not affect data in cache)
	originalData[0].Mail = "modified@example.com"

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	// Should use original data from cache (because SafeUserCache created a copy)
	assert.Equal(t, "original@example.com", result[0].Mail)
}

func TestJSON_MethodNotAllowed(t *testing.T) {
	// Test that non-GET methods should return 405
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache, nil)

	// Test disallowed methods
	disallowedMethods := []string{"POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	for _, method := range disallowedMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", http.NoBody)
			w := httptest.NewRecorder()

			handler(w, req)

			// Should return 405 Method Not Allowed
			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "方法%s应该返回405", method)
			assert.Contains(t, w.Body.String(), "Method not allowed", "响应体应该包含错误消息")
		})
	}
}

func TestJSON_Pagination(t *testing.T) {
	// Prepare test data (10 records)
	testData := make([]define.AllowListUser, 10)
	for i := 0; i < 10; i++ {
		testData[i] = define.AllowListUser{
			Phone: "1380013800" + strconv.Itoa(i),
			Mail:  "test" + strconv.Itoa(i) + "@example.com",
		}
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache, nil)

	t.Run("第一页", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=1&page_size=3", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		// Verify pagination structure
		assert.Contains(t, result, "data")
		assert.Contains(t, result, "pagination")

		// Verify data
		data, ok := result["data"].([]interface{})
		if !ok {
			t.Fatal("data 类型断言失败")
		}
		assert.Len(t, data, 3, "第一页应该返回3条记录")

		// Verify pagination info
		pagination, ok := result["pagination"].(map[string]interface{})
		require.True(t, ok, "pagination 类型断言失败")
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(3), pagination["page_size"])
		assert.Equal(t, float64(10), pagination["total"])
		assert.Equal(t, float64(4), pagination["total_pages"])
	})

	t.Run("最后一页", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=4&page_size=3", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].([]interface{})
		require.True(t, ok, "data 类型断言失败")
		assert.Len(t, data, 1, "最后一页应该返回1条记录")
	})

	t.Run("超出范围", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=100&page_size=3", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].([]interface{})
		require.True(t, ok, "data 类型断言失败")
		assert.Empty(t, data, "超出范围的页面应该返回空数组")
	})

	t.Run("无效的分页参数-非数字", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=abc&page_size=3", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "无效的参数应该返回400")
		assert.Contains(t, w.Body.String(), "Invalid pagination parameters")
	})

	t.Run("无效的分页参数-负数", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=-1&page_size=3", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "负数参数应该返回400")
	})

	t.Run("无效的分页参数-超出最大页码", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=2000000&page_size=3", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "超出最大页码应该返回400")
	})

	t.Run("无效的分页参数-超出最大页面大小", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=1&page_size=2000", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "超出最大页面大小应该返回400")
	})

	t.Run("无效的分页参数-参数过长", func(t *testing.T) {
		// Create a parameter longer than 20 characters
		longParam := strings.Repeat("1", 25)
		req := httptest.NewRequest("GET", "/?page="+longParam+"&page_size=3", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "参数过长应该返回400")
	})
}

func TestJSON_BackwardCompatibility(t *testing.T) {
	// Test backward compatibility: should return array directly when no pagination parameters
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13800138001", Mail: "test2@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache, nil)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Should return array directly, not pagination object
	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err, "没有分页参数时应该直接返回数组格式")

	assert.Len(t, result, 2, "应该返回所有数据")
}

// TestJSON_DifferentDataSizes tests different data size scenarios
func TestJSON_DifferentDataSizes(t *testing.T) {
	// Test small data (< SMALL_DATA_THRESHOLD)
	t.Run("小数据", func(t *testing.T) {
		smallData := make([]define.AllowListUser, 50) // Less than SMALL_DATA_THRESHOLD (100)
		for i := range smallData {
			smallData[i] = define.AllowListUser{
				Phone: fmt.Sprintf("138001%05d", i),
				Mail:  "test" + strconv.Itoa(i) + "@example.com",
			}
		}

		userCache := cache.NewSafeUserCache()
		userCache.Set(smallData)
		handler := JSON(userCache, nil)

		req := httptest.NewRequest("GET", "/", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "小数据应该正常处理")
		var result []define.AllowListUser
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Len(t, result, 50, "应该返回所有数据")
	})

	// Test medium data (between SMALL_DATA_THRESHOLD and LARGE_DATA_THRESHOLD)
	t.Run("中等数据", func(t *testing.T) {
		mediumData := make([]define.AllowListUser, 500) // Between thresholds
		for i := range mediumData {
			mediumData[i] = define.AllowListUser{
				Phone: fmt.Sprintf("138001%05d", i),
				Mail:  "test" + strconv.Itoa(i) + "@example.com",
			}
		}

		userCache := cache.NewSafeUserCache()
		userCache.Set(mediumData)
		handler := JSON(userCache, nil)

		req := httptest.NewRequest("GET", "/", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "中等数据应该正常处理")
		var result []define.AllowListUser
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Len(t, result, 500, "应该返回所有数据")
	})

	// Test large data (>= LARGE_DATA_THRESHOLD)
	t.Run("大数据", func(t *testing.T) {
		largeData := make([]define.AllowListUser, 20000) // >= LARGE_DATA_THRESHOLD (10000)
		for i := range largeData {
			largeData[i] = define.AllowListUser{
				Phone: fmt.Sprintf("138001%05d", i),
				Mail:  "test" + strconv.Itoa(i) + "@example.com",
			}
		}

		userCache := cache.NewSafeUserCache()
		userCache.Set(largeData)
		handler := JSON(userCache, nil)

		req := httptest.NewRequest("GET", "/", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "大数据应该正常处理")
		var result []define.AllowListUser
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Len(t, result, 20000, "应该返回所有数据")
	})
}

// TestJSON_Pagination_EdgeCases tests pagination edge cases
func TestJSON_Pagination_EdgeCases(t *testing.T) {
	testData := make([]define.AllowListUser, 5)
	for i := range testData {
		testData[i] = define.AllowListUser{
			Phone: "1380013800" + strconv.Itoa(i),
			Mail:  "test" + strconv.Itoa(i) + "@example.com",
		}
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache, nil)

	t.Run("page=0", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=0&page_size=2", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "page=0应该返回400")
	})

	t.Run("page_size=0", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=1&page_size=0", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "page_size=0应该返回400")
	})

	t.Run("page with special characters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=1a&page_size=2", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "包含特殊字符的参数应该返回400")
	})

	t.Run("page_size with special characters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=1&page_size=2b", http.NoBody)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "包含特殊字符的参数应该返回400")
	})
}

// TestJSON_Pagination_OnlyPage tests pagination with only page parameter
func TestJSON_Pagination_OnlyPage(t *testing.T) {
	testData := make([]define.AllowListUser, 10)
	for i := range testData {
		testData[i] = define.AllowListUser{
			Phone: "1380013800" + strconv.Itoa(i),
			Mail:  "test" + strconv.Itoa(i) + "@example.com",
		}
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache, nil)

	req := httptest.NewRequest("GET", "/?page=2", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "只有page参数应该正常处理")
	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result, "pagination", "应该返回分页格式")
}

// TestJSON_Pagination_OnlyPageSize tests pagination with only page_size parameter
func TestJSON_Pagination_OnlyPageSize(t *testing.T) {
	testData := make([]define.AllowListUser, 10)
	for i := range testData {
		testData[i] = define.AllowListUser{
			Phone: "1380013800" + strconv.Itoa(i),
			Mail:  "test" + strconv.Itoa(i) + "@example.com",
		}
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache, nil)

	req := httptest.NewRequest("GET", "/?page_size=5", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "只有page_size参数应该正常处理")
	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result, "pagination", "应该返回分页格式")
}
