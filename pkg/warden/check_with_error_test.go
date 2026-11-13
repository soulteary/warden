package warden

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/router"
)

// realHandlers serves the Warden handlers the SDK calls, backed by users.
func realHandlers(t *testing.T, users []define.AllowListUser) string {
	t.Helper()
	store := cache.NewSafeUserCache()
	store.Set(users)
	mux := http.NewServeMux()
	mux.HandleFunc("/user", router.GetUserByIdentifier(store, nil))
	mux.HandleFunc("/v1/lookup", router.GetLookup(store, nil))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server.URL
}

func TestCheckUserInListWithError(t *testing.T) {
	url := realHandlers(t, []define.AllowListUser{
		{Phone: "13800138000", Mail: "active@example.com", UserID: "u-active", Status: "active"},
		{Phone: "13900139000", Mail: "suspended@example.com", UserID: "u-suspended", Status: "suspended"},
		{Mail: "mailonly@example.com", UserID: "u-mail", Status: "active"},
	})
	client, err := NewClient(DefaultOptions().WithBaseURL(url))
	require.NoError(t, err)
	ctx := context.Background()

	for name, tc := range map[string]struct {
		phone, mail string
		want        bool
	}{
		"active by phone":                  {"13800138000", "", true},
		"active by mail":                   {"", "Active@Example.com", true},
		"unknown phone falls back to mail": {"13000000000", "mailonly@example.com", true},
		"unknown phone and unknown mail":   {"13000000000", "nobody@example.com", false},
		"unknown phone":                    {"13000000000", "", false},
		"unknown mail":                     {"", "nobody@example.com", false},
		"suspended by phone":               {"13900139000", "", false},
		"suspended phone ignores the mail": {"13900139000", "active@example.com", false},
		"suspended by mail":                {"", "suspended@example.com", false},
	} {
		got, checkErr := client.CheckUserInListWithError(ctx, tc.phone, tc.mail)
		require.NoError(t, checkErr, name)
		require.Equal(t, tc.want, got, name)
		require.Equal(t, client.CheckUserInList(ctx, tc.phone, tc.mail), got, "same answer as CheckUserInList: %s", name)
	}

	_, err = client.CheckUserInListWithError(ctx, " ", "")
	var sdkErr *Error
	require.True(t, errors.As(err, &sdkErr))
	require.Equal(t, ErrCodeInvalidConfig, sdkErr.Code)
}

// An outage, a rejected credential and a rate limit are errors, not "denied".
func TestCheckUserInListWithErrorReportsFailures(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		header http.Header
		code   string
		status int
	}{
		"unauthorized": {nil, ErrCodeUnauthorized, http.StatusUnauthorized},
		"rate limited": {http.Header{"Retry-After": []string{"42"}}, ErrCodeRateLimited, http.StatusTooManyRequests},
		"server error": {nil, ErrCodeServerError, http.StatusServiceUnavailable},
	} {
		client, err := NewClient(DefaultOptions().WithBaseURL(statusServer(t, tc.status, tc.header)).WithAPIKey("key"))
		require.NoError(t, err, name)

		allowed, err := client.CheckUserInListWithError(ctx, "13800138000", "user@example.com")
		require.False(t, allowed, name)
		var sdkErr *Error
		require.True(t, errors.As(err, &sdkErr), name)
		require.Equal(t, tc.code, sdkErr.Code, name)
		if tc.code == ErrCodeRateLimited {
			require.Equal(t, 42*time.Second, sdkErr.RetryAfter)
		} else {
			require.Zero(t, sdkErr.RetryAfter, name)
		}
		require.False(t, client.CheckUserInList(ctx, "13800138000", "user@example.com"), name)
	}

	// Warden unreachable.
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	client, err := NewClient(DefaultOptions().WithBaseURL(url))
	require.NoError(t, err)
	allowed, err := client.CheckUserInListWithError(ctx, "", "user@example.com")
	require.False(t, allowed)
	var sdkErr *Error
	require.True(t, errors.As(err, &sdkErr))
	require.Equal(t, ErrCodeRequestFailed, sdkErr.Code)
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	require.Equal(t, 17*time.Second, parseRetryAfter(" 17 ", now))
	require.Equal(t, 90*time.Second, parseRetryAfter(now.Add(90*time.Second).Format(http.TimeFormat), now))
	for _, value := range []string{"", "-5", "soon", now.Add(-time.Minute).Format(http.TimeFormat)} {
		require.Zero(t, parseRetryAfter(value, now), value)
	}
}

func TestLookupAgainstTheLookupHandler(t *testing.T) {
	url := realHandlers(t, []define.AllowListUser{
		{Phone: "13800138000", Mail: "active@example.com", UserID: "u-active", Status: "active", Name: "Ada"},
		{Mail: "mailonly@example.com", UserID: "u-mail", Status: "inactive"},
	})
	client, err := NewClient(DefaultOptions().WithBaseURL(url + "/"))
	require.NoError(t, err)
	ctx := context.Background()

	for _, identifier := range []string{"13800138000", "active@example.com", "u-active"} {
		result, lookupErr := client.Lookup(ctx, identifier)
		require.NoError(t, lookupErr, identifier)
		require.Equal(t, "u-active", result.UserID)
		require.True(t, result.IsActive())
		require.Equal(t, "13800138000", result.Destination.Phone)
		require.Equal(t, "active@example.com", result.Destination.Email)
		require.Equal(t, "sms", result.ChannelHint)
		require.Equal(t, "Ada", result.Name)
	}

	result, err := client.Lookup(ctx, "mailonly@example.com")
	require.NoError(t, err)
	require.False(t, result.IsActive())
	require.Equal(t, "email", result.ChannelHint)
	require.Empty(t, result.Destination.Phone)

	_, err = client.Lookup(ctx, "nobody@example.com")
	var sdkErr *Error
	require.True(t, errors.As(err, &sdkErr))
	require.Equal(t, ErrCodeNotFound, sdkErr.Code)

	_, err = client.Lookup(ctx, "  ")
	require.True(t, errors.As(err, &sdkErr))
	require.Equal(t, ErrCodeInvalidConfig, sdkErr.Code)
}
