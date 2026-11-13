package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	loggerkit "github.com/soulteary/logger-kit/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
)

// A derived user ID is an unsalted digest of the phone or mail, so the
// handlers must not write it to the log in full.
func TestHandlersDoNotLogTheFullUserID(t *testing.T) {
	const userID = "0123456789abcdef"
	users := cache.NewSafeUserCache()
	users.Set([]define.AllowListUser{{Phone: "13800138000", Mail: "member@example.com", UserID: userID, Status: "active"}})

	handlers := map[string]func(http.ResponseWriter, *http.Request){
		"/user?user_id=" + userID:                GetUserByIdentifier(users, nil),
		"/user?phone=13800138000":                GetUserByIdentifier(users, nil),
		"/user?user_id=fedcba9876543210":         GetUserByIdentifier(users, nil),
		"/v1/lookup?identifier=" + userID:        GetLookup(users, nil),
		"/v1/lookup?identifier=fedcba9876543210": GetLookup(users, nil),
	}
	for target, handler := range handlers {
		t.Run(target, func(t *testing.T) {
			var out bytes.Buffer
			log := loggerkit.New(loggerkit.Config{Level: loggerkit.DebugLevel, Format: loggerkit.FormatJSON, Output: &out})
			req := loggerkit.SetLoggerInRequest(httptest.NewRequest(http.MethodGet, target, http.NoBody), log)
			handler(httptest.NewRecorder(), req)

			require.NotEmpty(t, out.String(), "the handler logs the query")
			assert.NotContains(t, out.String(), userID)
			assert.NotContains(t, out.String(), "fedcba9876543210")
		})
	}
}
