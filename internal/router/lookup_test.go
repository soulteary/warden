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

	handler := GetLookup(userCache, nil)

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

	handler := GetLookup(userCache, nil)

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

	handler := GetLookup(userCache, nil)

	req := httptest.NewRequest("GET", "/v1/lookup?identifier=nobody@example.com", http.NoBody)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetLookup_MissingIdentifier(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	handler := GetLookup(userCache, nil)

	req := httptest.NewRequest("GET", "/v1/lookup", http.NoBody)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetLookup_ResponseFieldsAllowlist(t *testing.T) {
	userCache := cache.NewSafeUserCache()
	userCache.Set([]define.AllowListUser{{
		Phone: "13800138000", Mail: "test1@example.com", UserID: "uid1", Status: "active",
		Name: "Alice", DingtalkUserID: "ding-1",
	}})

	lookup := func(t *testing.T, fields []string) (LookupResponse, map[string]interface{}) {
		t.Helper()
		handler := GetLookup(userCache, fields)
		req := httptest.NewRequest("GET", "/v1/lookup?identifier=uid1", http.NoBody)
		w := httptest.NewRecorder()
		handler(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp LookupResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		var raw map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
		return resp, raw
	}

	t.Run("empty allowlist returns every field", func(t *testing.T) {
		resp, _ := lookup(t, nil)
		assert.Equal(t, "uid1", resp.UserID)
		assert.Equal(t, "active", resp.Status)
		assert.Equal(t, "sms", resp.ChannelHint)
		assert.Equal(t, "test1@example.com", resp.Destination.Email)
		assert.Equal(t, "13800138000", resp.Destination.Phone)
		assert.Equal(t, "Alice", resp.Name)
		assert.Equal(t, "ding-1", resp.DingtalkUserID)
	})

	t.Run("structural fields survive an allowlist that omits them", func(t *testing.T) {
		resp, raw := lookup(t, []string{"name"})
		assert.Equal(t, "uid1", resp.UserID)
		assert.Equal(t, "active", resp.Status)
		assert.Equal(t, "email", resp.ChannelHint)
		assert.Equal(t, "Alice", resp.Name)
		assert.Contains(t, raw, "destination")
		assert.Empty(t, raw["destination"])
		assert.NotContains(t, raw, "dingtalk_userid")
	})

	t.Run("phone and mail are withheld unless allowed", func(t *testing.T) {
		resp, raw := lookup(t, []string{"user_id", "status", "mail"})
		assert.Equal(t, "test1@example.com", resp.Destination.Email)
		assert.Empty(t, resp.Destination.Phone)
		assert.Equal(t, "email", resp.ChannelHint, "the hint must not reveal a withheld phone")
		assert.NotContains(t, raw, "name")
		assert.NotContains(t, raw, "dingtalk_userid")

		resp, _ = lookup(t, []string{"phone"})
		assert.Equal(t, "13800138000", resp.Destination.Phone)
		assert.Empty(t, resp.Destination.Email)
		assert.Equal(t, "sms", resp.ChannelHint)
	})
}
