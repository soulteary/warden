package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	middlewarekit "github.com/soulteary/middleware-kit/v3"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
)

// The list handler calls WriteHeader before writing. Behind the compression
// middleware, a response over the compression threshold must still reach a
// standard HTTP client as decodable JSON.
func TestJSON_CompressedListIsDecodableByDefaultClient(t *testing.T) {
	users := make([]define.AllowListUser, 0, 200)
	for i := 0; i < 200; i++ {
		users = append(users, define.AllowListUser{
			Phone: fmt.Sprintf("138001%05d", i),
			Mail:  fmt.Sprintf("user%d@example.com", i),
		})
	}
	userCache := cache.NewSafeUserCache()
	userCache.Set(users)

	compress := middlewarekit.CompressStd(middlewarekit.DefaultCompressConfig())
	server := httptest.NewServer(compress(http.HandlerFunc(JSON(userCache, nil))))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, resp.Uncompressed, "the response should have been gzip-encoded and decoded by the client")

	var got []define.AllowListUser
	require.NoError(t, json.Unmarshal(body, &got), "body must be valid JSON after transport decoding")
	require.Len(t, got, len(users))
}
