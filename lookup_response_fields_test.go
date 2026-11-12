package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/define"
)

// TestRegisterRoutes_LookupHonorsResponseFields pins that the configured field
// allowlist reaches /v1/lookup, not only the list and user routes.
func TestRegisterRoutes_LookupHonorsResponseFields(t *testing.T) {
	app := responseMetadataApp(t)
	prepared, snap := preparedTestSnapshot(t, []define.AllowListUser{
		{Phone: "13800138000", Mail: "member@example.com", UserID: "member", Status: "active", Name: "Member"},
	})
	snap.SourceSucceededAt = time.Now().Add(-time.Minute).UTC()
	app.userCache.ApplyPrepared(prepared)
	app.snapshots.Store(snap)
	app.responseFields = []string{"user_id", "status", "mail"}

	mux := registerRoutes(app)
	req := httptest.NewRequest(http.MethodGet, "/v1/lookup?identifier=member", http.NoBody)
	req.Header.Set("X-API-Key", "key")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	assert.NotContains(t, rec.Body.String(), "13800138000")
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "member", got["user_id"])
	assert.Equal(t, "active", got["status"])
	assert.Equal(t, "email", got["channel_hint"])
	assert.Equal(t, map[string]interface{}{"email": "member@example.com"}, got["destination"])
	assert.NotContains(t, got, "name")
}
