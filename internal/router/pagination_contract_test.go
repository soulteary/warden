package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

// Compare the optimized handler with the prior full-copy pagination contract.
func TestPaginationMatchesFullCopyContract(t *testing.T) {
	for _, users := range [][]define.AllowListUser{nil, {
		{Mail: "z@example.com", UserID: "z", Scope: []string{"read"}},
		{Mail: "a@example.com", UserID: "a", Scope: []string{}},
		{Phone: "bad", Mail: "bad@example.com"},
		{Mail: "m@example.com", UserID: "m"},
		{Mail: "z@example.com", UserID: "replacement", Scope: []string{"write"}},
	}} {
		c := cache.NewSafeUserCache()
		c.Set(users)
		original := c.Get()
		for _, size := range []int{1, 2, 20} {
			for _, page := range []int{1, 2, 3, 1000000} {
				for _, fields := range [][]string{nil, {"user_id", "scope"}} {
					req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?page=%d&page_size=%d", page, size), http.NoBody)
					got := httptest.NewRecorder()
					JSON(c, fields)(got, req)
					rows, total, pages := paginate(original, page, size)
					payload := buildPaginatedResponse(rows, page, size, total, pages)
					if len(fields) > 0 {
						payload["data"] = UsersToMaps(rows, fields)
					}
					expected := httptest.NewRecorder()
					require.NoError(t, encodeJSONResponse(expected, req, payload))
					require.Equal(t, http.StatusOK, got.Code)
					require.Equal(t, expected.Body.String(), got.Body.String())
					require.Equal(t, "application/json", got.Header().Get("Content-Type"))
				}
			}
		}
	}
}
