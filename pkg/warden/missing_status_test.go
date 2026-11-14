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

// The warning about a response without status is logged once per client, not
// once per lookup; later occurrences go to the debug level.
func TestMissingStatusWarningIsLoggedOncePerClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"phone":"13800138000","mail":"active@example.com"}`)) //nolint:errcheck // test server
	}))
	t.Cleanup(server.Close)
	ctx := context.Background()
	countWarnings := func(logs *levelRecorder) (warns, debugs int) {
		logs.mu.Lock()
		defer logs.mu.Unlock()
		for _, msg := range logs.warns {
			if strings.Contains(msg, "no status field") {
				warns++
			}
		}
		for _, msg := range logs.debugs {
			if strings.Contains(msg, "no status field") {
				debugs++
			}
		}
		return warns, debugs
	}

	logs := &levelRecorder{}
	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithLogger(logs))
	require.NoError(t, err)
	for range 5 {
		user, getErr := client.GetUserByIdentifier(ctx, "13800138000", "", "")
		require.NoError(t, getErr)
		require.False(t, user.IsActive())
	}
	// The error result does not depend on whether the warning was logged.
	_, checkErr := client.CheckUserInListWithError(ctx, "13800138000", "")
	var sdkErr *Error
	require.True(t, errors.As(checkErr, &sdkErr))
	require.Equal(t, ErrCodeInvalidResponse, sdkErr.Code)
	warns, debugs := countWarnings(logs)
	require.Equal(t, 1, warns)
	require.Equal(t, 5, debugs)

	// A second client warns for itself.
	otherLogs := &levelRecorder{}
	other, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithLogger(otherLogs))
	require.NoError(t, err)
	_, err = other.GetUserByIdentifier(ctx, "13800138000", "", "")
	require.NoError(t, err)
	warns, debugs = countWarnings(otherLogs)
	require.Equal(t, 1, warns)
	require.Zero(t, debugs)
}

type levelRecorder struct {
	debugs []string
	warnRecorder
}

func (r *levelRecorder) Debug(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.debugs = append(r.debugs, msg)
}
