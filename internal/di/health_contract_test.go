package di

import (
	"net/http"
	"net/http/httptest"
	"testing"

	health "github.com/soulteary/health-kit/v4"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

func TestDependencyHealthReflectsCacheAndMissingRedis(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		enabled, cache, data bool
	}{
		{"missing cache", false, false, false}, {"empty cache", false, true, false}, {"loaded cache", false, true, true}, {"missing redis", true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &Dependencies{Config: &cmd.Config{RedisEnabled: tc.enabled}}
			if tc.cache {
				d.UserCache = cache.NewSafeUserCache()
			}
			if tc.data {
				d.UserCache.Set([]define.AllowListUser{{Mail: "member@example.com"}})
			}
			w := httptest.NewRecorder()
			health.Handler(d.createHealthAggregator()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", http.NoBody))
			if tc.data && !tc.enabled {
				require.Equal(t, 200, w.Code)
			} else {
				require.Equal(t, 503, w.Code)
			}
		})
	}
}
