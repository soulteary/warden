package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
)

func TestGetLookup_ByMail(t *testing.T) {
	testUsers := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com", UserID: "uid1", Status: "active"},
		{Phone: "", Mail: "emailonly@example.com", UserID: "uid2", Status: "active"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testUsers)

	handler := GetLookup(userCache)

	req := httptest.NewRequest("GET", "/v1/lookup?identifier=test1@example.com", http.NoBody)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp LookupResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "uid1", resp.UserID)
	assert.Equal(t, "active", resp.Status)
	assert.Equal(t, "test1@example.com", resp.Destination.Email)
	assert.Equal(t, "13800138000", resp.Destination.Phone)
	assert.Equal(t, "sms", resp.ChannelHint)
}

func TestGetLookup_EmailOnlyUser(t *testing.T) {
	testUsers := []define.AllowListUser{
		{Phone: "", Mail: "only@example.com", UserID: "uid-email-only", Status: "active"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testUsers)

	handler := GetLookup(userCache)

	req := httptest.NewRequest("GET", "/v1/lookup?identifier=only@example.com", http.NoBody)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp LookupResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "uid-email-only", resp.UserID)
	assert.Equal(t, "active", resp.Status)
	assert.Equal(t, "only@example.com", resp.Destination.Email)
	assert.Empty(t, resp.Destination.Phone)
	assert.Equal(t, "email", resp.ChannelHint)
}

func TestGetLookup_NotFound(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	userCache.Set([]define.AllowListUser{})

	handler := GetLookup(userCache)

	req := httptest.NewRequest("GET", "/v1/lookup?identifier=nobody@example.com", http.NoBody)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetLookup_MissingIdentifier(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	handler := GetLookup(userCache)

	req := httptest.NewRequest("GET", "/v1/lookup", http.NoBody)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
