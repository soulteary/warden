package warden

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/router"
)

// A server whose RESPONSE_FIELDS leaves out status returns users without it.
// The check cannot answer then, and must say so: it used to report every user
// as denied.
func TestCheckReportsAResponseWithoutStatus(t *testing.T) {
	store := cache.NewSafeUserCache()
	store.Set([]define.AllowListUser{{Phone: "13800138000", Mail: "active@example.com", UserID: "u-active", Status: "active"}})
	mux := http.NewServeMux()
	mux.HandleFunc("/user", router.GetUserByIdentifier(store, []string{"phone", "mail", "name"}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	logs := &warnRecorder{}
	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithLogger(logs))
	require.NoError(t, err)
	ctx := context.Background()

	for _, identifiers := range [][2]string{{"13800138000", ""}, {"", "active@example.com"}, {"13000000000", "active@example.com"}} {
		allowed, checkErr := client.CheckUserInListWithError(ctx, identifiers[0], identifiers[1])
		require.False(t, allowed)
		var sdkErr *Error
		require.True(t, errors.As(checkErr, &sdkErr), "%v: %v", identifiers, checkErr)
		require.Equal(t, ErrCodeInvalidResponse, sdkErr.Code)
		require.Contains(t, sdkErr.Message, "RESPONSE_FIELDS")
	}

	// An unknown user is still an answer: the 404 does not depend on the fields.
	allowed, err := client.CheckUserInListWithError(ctx, "13000000000", "nobody@example.com")
	require.NoError(t, err)
	require.False(t, allowed)

	// CheckUserInList denies as before and the log names the cause.
	require.False(t, client.CheckUserInList(ctx, "13800138000", ""))
	logs.mu.Lock()
	logged := strings.Join(logs.warns, "\n")
	logs.mu.Unlock()
	require.Contains(t, logged, "no status field")

	// GetUserByIdentifier returns what the server sent.
	user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
	require.NoError(t, err)
	require.Equal(t, "active@example.com", user.Mail)
	require.Empty(t, user.Status)
	require.False(t, user.IsActive())
}

// A status that is present is used as it is, including an empty one.
func TestCheckAcceptsAnEmptyStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"phone":"13800138000","status":""}`)) //nolint:errcheck // test server
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
	require.NoError(t, err)

	allowed, err := client.CheckUserInListWithError(context.Background(), "13800138000", "")
	require.NoError(t, err)
	require.False(t, allowed)
}
