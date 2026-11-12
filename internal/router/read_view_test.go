package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

type changingViewSource struct {
	first  *cache.ReadView
	second *cache.ReadView
	calls  int
}

func (s *changingViewSource) ReadView() *cache.ReadView {
	s.calls++
	if s.calls == 1 {
		return s.first
	}
	return s.second
}

func TestHandlersPinOneReadView(t *testing.T) {
	first := cache.NewSafeUserCache()
	first.Set([]define.AllowListUser{{Phone: "13800138000", Mail: "old@example.com", UserID: "member", Status: "active"}})
	second := cache.NewSafeUserCache()
	second.Set([]define.AllowListUser{{Phone: "13900139000", Mail: "new@example.com", UserID: "member", Status: "inactive"}})
	for _, tc := range []struct {
		name string
		path string
	}{
		{"lookup_phone_to_id", "/v1/lookup?identifier=member"},
		{"user", "/user?user_id=member"},
		{"list", "/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &changingViewSource{first: first.ReadView(), second: second.ReadView()}
			var handler func(http.ResponseWriter, *http.Request)
			switch tc.name {
			case "lookup_phone_to_id":
				handler = GetLookup(source, nil)
			case "user":
				handler = GetUserByIdentifier(source, nil)
			case "list":
				handler = JSON(source, nil)
			}
			rec := httptest.NewRecorder()
			handler(rec, httptest.NewRequest(http.MethodGet, tc.path, http.NoBody))
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, 1, source.calls, "phone-to-ID fallback must not acquire a second generation")
			switch tc.name {
			case "lookup_phone_to_id":
				var got LookupResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, "old@example.com", got.Destination.Email)
				require.Equal(t, "active", got.Status)
			case "list":
				var got []define.AllowListUser
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Len(t, got, 1)
				require.Equal(t, "old@example.com", got[0].Mail)
			default:
				var got define.AllowListUser
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, "old@example.com", got.Mail)
			}
		})
	}
}
