package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	loggerkit "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
)

func BenchmarkJSONPagination(b *testing.B) {
	quiet := loggerkit.New(loggerkit.Config{Level: loggerkit.ErrorLevel})
	for _, count := range []int{1000, 10000} {
		users := make([]define.AllowListUser, count)
		for i := range users {
			users[i] = define.AllowListUser{Mail: fmt.Sprintf("member%d@example.com", i), UserID: fmt.Sprintf("member%d", i), Status: "active", Scope: []string{"read", "write"}}
		}
		c := cache.NewSafeUserCache()
		c.Set(users)
		if c.Len() != count {
			b.Fatalf("fixture retained %d of %d rows", c.Len(), count)
		}
		for _, page := range []int{1, count / 20} {
			b.Run(fmt.Sprintf("rows=%d/page=%d", count, page), func(b *testing.B) {
				handler := JSON(c, nil)
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?page=%d&page_size=20", page), http.NoBody)
				req = loggerkit.SetLoggerInRequest(req, quiet)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					handler(httptest.NewRecorder(), req)
				}
			})
		}
	}
}
