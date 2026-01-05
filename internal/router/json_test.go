package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"soulteary.com/soulteary/warden/internal/cache"
	"soulteary.com/soulteary/warden/internal/define"
)

func TestJSON_Handler(t *testing.T) {
	// 准备测试数据
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13800138001", Mail: "test2@example.com"},
	}

	// 创建线程安全缓存并设置数据
	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)

	// 创建处理器
	handler := JSON(userCache)

	// 创建测试请求
	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	// 执行处理器
	handler(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code, "状态码应该是200")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "Content-Type应该是application/json")

	// 验证响应体
	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err, "响应体应该是有效的JSON")

	assert.Len(t, result, 2, "应该返回2条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
	assert.Equal(t, "test1@example.com", result[0].Mail)
}

func TestJSON_EmptyData(t *testing.T) {
	// 测试空数据
	emptyData := []define.AllowListUser{}
	userCache := cache.NewSafeUserCache()
	userCache.Set(emptyData)
	handler := JSON(userCache)

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
	// 测试单条记录
	singleData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "single@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(singleData)
	handler := JSON(userCache)

	req := httptest.NewRequest("GET", "/", nil)
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
	// 测试Unicode字符（邮箱不支持中文，使用有效的邮箱格式）
	unicodeData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(unicodeData)
	handler := JSON(userCache)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Len(t, result, 1)
	assert.Equal(t, "test@example.com", result[0].Mail, "应该正确处理数据")
}

func TestJSON_ContentType(t *testing.T) {
	// 测试Content-Type头
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	contentType := w.Header().Get("Content-Type")
	assert.Equal(t, "application/json", contentType, "Content-Type应该正确设置")
}

func TestJSON_StatusCode(t *testing.T) {
	// 测试状态码
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "状态码应该是200")
}

func TestJSON_MultipleRecords(t *testing.T) {
	// 测试多条记录
	multipleData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13800138001", Mail: "test2@example.com"},
		{Phone: "13800138002", Mail: "test3@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(multipleData)
	handler := JSON(userCache)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Len(t, result, 3, "应该返回3条记录")
}

func TestJSON_DataModification(t *testing.T) {
	// 测试数据修改后处理器仍然使用缓存中的原始数据（SafeUserCache 会创建副本）
	originalData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "original@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(originalData)
	handler := JSON(userCache)

	// 修改原始数据（不应该影响缓存中的数据）
	originalData[0].Mail = "modified@example.com"

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	// 应该使用缓存中的原始数据（因为 SafeUserCache 创建了副本）
	assert.Equal(t, "original@example.com", result[0].Mail)
}

func TestJSON_MethodNotAllowed(t *testing.T) {
	// 测试非 GET 方法应该返回 405
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)

	// 测试不允许的方法
	disallowedMethods := []string{"POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	for _, method := range disallowedMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			// 应该返回 405 Method Not Allowed
			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "方法%s应该返回405", method)
			assert.Contains(t, w.Body.String(), "Method not allowed", "响应体应该包含错误消息")
		})
	}
}

func TestJSON_Pagination(t *testing.T) {
	// 准备测试数据（10条记录）
	testData := make([]define.AllowListUser, 10)
	for i := 0; i < 10; i++ {
		testData[i] = define.AllowListUser{
			Phone: "1380013800" + strconv.Itoa(i),
			Mail:  "test" + strconv.Itoa(i) + "@example.com",
		}
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)

	t.Run("第一页", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=1&page_size=3", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		// 验证分页结构
		assert.Contains(t, result, "data")
		assert.Contains(t, result, "pagination")

		// 验证数据
		data, ok := result["data"].([]interface{})
		if !ok {
			t.Fatal("data 类型断言失败")
		}
		assert.Len(t, data, 3, "第一页应该返回3条记录")

		// 验证分页信息
		pagination, ok := result["pagination"].(map[string]interface{})
		require.True(t, ok, "pagination 类型断言失败")
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(3), pagination["page_size"])
		assert.Equal(t, float64(10), pagination["total"])
		assert.Equal(t, float64(4), pagination["total_pages"])
	})

	t.Run("最后一页", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=4&page_size=3", nil)
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
		req := httptest.NewRequest("GET", "/?page=100&page_size=3", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data := result["data"].([]interface{})
		assert.Empty(t, data, "超出范围的页面应该返回空数组")
	})

	t.Run("无效的分页参数-非数字", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=abc&page_size=3", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "无效的参数应该返回400")
		assert.Contains(t, w.Body.String(), "Invalid pagination parameters")
	})

	t.Run("无效的分页参数-负数", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=-1&page_size=3", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "负数参数应该返回400")
	})

	t.Run("无效的分页参数-超出最大页码", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=2000000&page_size=3", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "超出最大页码应该返回400")
	})

	t.Run("无效的分页参数-超出最大页面大小", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=1&page_size=2000", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "超出最大页面大小应该返回400")
	})

	t.Run("无效的分页参数-参数过长", func(t *testing.T) {
		// 创建一个超过20字符的参数
		longParam := strings.Repeat("1", 25)
		req := httptest.NewRequest("GET", "/?page="+longParam+"&page_size=3", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "参数过长应该返回400")
	})
}

func TestJSON_BackwardCompatibility(t *testing.T) {
	// 测试向后兼容性：没有分页参数时应该直接返回数组
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13800138001", Mail: "test2@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 应该直接返回数组，而不是分页对象
	var result []define.AllowListUser
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err, "没有分页参数时应该直接返回数组格式")

	assert.Len(t, result, 2, "应该返回所有数据")
}
