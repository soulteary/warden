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

// TestGetUserByIdentifier_ByPhone tests querying user by phone number
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

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "Content-Type should be application/json")

	var user define.AllowListUser
	err := json.NewDecoder(w.Body).Decode(&user)
	require.NoError(t, err, "Should be able to parse JSON")
	assert.Equal(t, "13800138000", user.Phone, "Phone number should match")
	assert.Equal(t, "test1@example.com", user.Mail, "Email should match")
}

// TestGetUserByIdentifier_ByMail tests querying user by email
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

// TestGetUserByIdentifier_ByUserID tests querying user by user ID
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

// TestGetUserByIdentifier_MissingIdentifier tests missing identifier
func TestGetUserByIdentifier_MissingIdentifier(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	handler := GetUserByIdentifier(userCache)

	req := httptest.NewRequest("GET", "/user", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400")
}

// TestGetUserByIdentifier_MultipleIdentifiers tests providing multiple identifiers
func TestGetUserByIdentifier_MultipleIdentifiers(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	handler := GetUserByIdentifier(userCache)

	req := httptest.NewRequest("GET", "/user?phone=13800138000&mail=test@example.com", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 (only one identifier should be provided)")
}

// TestGetUserByIdentifier_UserNotFound tests user not found
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

	assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404")
}

// TestGetUserByIdentifier_InvalidMethod tests invalid HTTP method
func TestGetUserByIdentifier_InvalidMethod(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	handler := GetUserByIdentifier(userCache)

	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/user?phone=13800138000", http.NoBody)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "Should return 405")
		})
	}
}

// TestGetUserByIdentifier_WithSpaces tests parameters with spaces
func TestGetUserByIdentifier_WithSpaces(t *testing.T) {
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com", UserID: "user1"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testUsers)

	handler := GetUserByIdentifier(userCache)

	// Test phone number with spaces (URL encoded space is %20)
	req := httptest.NewRequest("GET", "/user?phone=%2013800138000%20", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should be able to handle parameters with spaces")
}

// TestGetUserByIdentifier_JSONEncodingError tests JSON encoding error handling
func TestGetUserByIdentifier_JSONEncodingError(t *testing.T) {
	// This test is difficult to simulate because json.NewEncoder usually doesn't fail
	// But we can test basic error handling logic
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com", UserID: "user1"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testUsers)

	handler := GetUserByIdentifier(userCache)

	req := httptest.NewRequest("GET", "/user?phone=13800138000", http.NoBody)
	w := httptest.NewRecorder()

	handler(w, req)

	// Should succeed under normal circumstances
	assert.Equal(t, http.StatusOK, w.Code)
}
