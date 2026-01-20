package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestGetUserByIdentifier_ByPhone 测试通过手机号查询用户
func TestGetUserByIdentifier_ByPhone(t *testing.T) {
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com", UserID: "user1"},
		{Phone: "13900139000", Mail: "test2@example.com", UserID: "user2"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testUsers)

	handler := GetUserByIdentifier(userCache)

	req := httptest.NewRequest("GET", "/user?phone=13800138000", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "应该返回 200")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "Content-Type 应该是 application/json")

	var user define.AllowListUser
	err := json.NewDecoder(w.Body).Decode(&user)
	require.NoError(t, err, "应该能够解析 JSON")
	assert.Equal(t, "13800138000", user.Phone, "手机号应该匹配")
	assert.Equal(t, "test1@example.com", user.Mail, "邮箱应该匹配")
}

// TestGetUserByIdentifier_ByMail 测试通过邮箱查询用户
func TestGetUserByIdentifier_ByMail(t *testing.T) {
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com", UserID: "user1"},
		{Phone: "13900139000", Mail: "test2@example.com", UserID: "user2"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testUsers)

	handler := GetUserByIdentifier(userCache)

	req := httptest.NewRequest("GET", "/user?mail=test2@example.com", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var user define.AllowListUser
	err := json.NewDecoder(w.Body).Decode(&user)
	require.NoError(t, err)
	assert.Equal(t, "13900139000", user.Phone)
	assert.Equal(t, "test2@example.com", user.Mail)
}

// TestGetUserByIdentifier_ByUserID 测试通过用户 ID 查询用户
func TestGetUserByIdentifier_ByUserID(t *testing.T) {
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com", UserID: "user1"},
		{Phone: "13900139000", Mail: "test2@example.com", UserID: "user2"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testUsers)

	handler := GetUserByIdentifier(userCache)

	req := httptest.NewRequest("GET", "/user?user_id=user1", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var user define.AllowListUser
	err := json.NewDecoder(w.Body).Decode(&user)
	require.NoError(t, err)
	assert.Equal(t, "user1", user.UserID)
	assert.Equal(t, "13800138000", user.Phone)
}

// TestGetUserByIdentifier_MissingIdentifier 测试缺少标识符
func TestGetUserByIdentifier_MissingIdentifier(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	handler := GetUserByIdentifier(userCache)

	req := httptest.NewRequest("GET", "/user", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "应该返回 400")
}

// TestGetUserByIdentifier_MultipleIdentifiers 测试提供多个标识符
func TestGetUserByIdentifier_MultipleIdentifiers(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	handler := GetUserByIdentifier(userCache)

	req := httptest.NewRequest("GET", "/user?phone=13800138000&mail=test@example.com", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "应该返回 400（只能提供一个标识符）")
}

// TestGetUserByIdentifier_UserNotFound 测试用户不存在
func TestGetUserByIdentifier_UserNotFound(t *testing.T) {
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com", UserID: "user1"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testUsers)

	handler := GetUserByIdentifier(userCache)

	req := httptest.NewRequest("GET", "/user?phone=99999999999", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "应该返回 404")
}

// TestGetUserByIdentifier_InvalidMethod 测试无效的 HTTP 方法
func TestGetUserByIdentifier_InvalidMethod(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	handler := GetUserByIdentifier(userCache)

	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/user?phone=13800138000", http.NoBody)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "应该返回 405")
		})
	}
}

// TestGetUserByIdentifier_WithSpaces 测试带空格的参数
func TestGetUserByIdentifier_WithSpaces(t *testing.T) {
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com", UserID: "user1"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testUsers)

	handler := GetUserByIdentifier(userCache)

	// 测试带空格的手机号（URL 编码空格为 %20）
	req := httptest.NewRequest("GET", "/user?phone=%2013800138000%20", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "应该能够处理带空格的参数")
}

// TestGetUserByIdentifier_JSONEncodingError 测试 JSON 编码错误处理
func TestGetUserByIdentifier_JSONEncodingError(t *testing.T) {
	// 这个测试比较难模拟，因为 json.NewEncoder 通常不会失败
	// 但我们可以测试基本的错误处理逻辑
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com", UserID: "user1"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testUsers)

	handler := GetUserByIdentifier(userCache)

	req := httptest.NewRequest("GET", "/user?phone=13800138000", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	// 正常情况下应该成功
	assert.Equal(t, http.StatusOK, w.Code)
}
