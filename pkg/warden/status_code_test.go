package warden

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// An error that reports a response status carries that status, and the error
// code of each status is the one released callers already compare.
func TestErrorCarriesResponseStatus(t *testing.T) {
	for _, tc := range []struct {
		code   string
		status int
	}{
		{ErrCodeUnauthorized, http.StatusUnauthorized},
		{ErrCodeRequestFailed, http.StatusForbidden},
		{ErrCodeNotFound, http.StatusNotFound},
		{ErrCodeRateLimited, http.StatusTooManyRequests},
		{ErrCodeServerError, http.StatusInternalServerError},
		{ErrCodeServerError, http.StatusBadGateway},
		{ErrCodeServerError, http.StatusServiceUnavailable},
		{ErrCodeRequestFailed, http.StatusGatewayTimeout},
		{ErrCodeRequestFailed, http.StatusBadRequest},
	} {
		client, err := NewClient(DefaultOptions().WithBaseURL(statusServer(t, tc.status, nil)).WithAPIKey("key"))
		require.NoError(t, err)
		ctx := context.Background()

		calls := map[string]func() error{
			"GetUserByIdentifier": func() error {
				_, callErr := client.GetUserByIdentifier(ctx, "13800138000", "", "")
				return callErr
			},
			"GetUsers": func() error {
				_, callErr := client.GetUsers(ctx)
				return callErr
			},
			"GetUsersPaginated": func() error {
				_, callErr := client.GetUsersPaginated(ctx, 1, 10)
				return callErr
			},
			"Lookup": func() error {
				_, callErr := client.Lookup(ctx, "13800138000")
				return callErr
			},
		}
		// A 404 is the answer "not on the list" for this call, not an error.
		if tc.status != http.StatusNotFound {
			calls["CheckUserInListWithError"] = func() error {
				_, callErr := client.CheckUserInListWithError(ctx, "13800138000", "")
				return callErr
			}
		}
		for name, call := range calls {
			var sdkErr *Error
			require.Truef(t, errors.As(call(), &sdkErr), "%s status %d", name, tc.status)
			require.Equalf(t, tc.code, sdkErr.Code, "%s status %d", name, tc.status)
			require.Equalf(t, tc.status, sdkErr.StatusCode, "%s status %d", name, tc.status)
		}
	}
}

// The status of the last response is reported when retries are used up.
func TestErrorCarriesStatusAfterRetries(t *testing.T) {
	opts := DefaultOptions().WithBaseURL(statusServer(t, http.StatusServiceUnavailable, nil))
	opts.Retry = DefaultRetryOptions()
	opts.Retry.MaxRetries = 2
	opts.Retry.RetryDelay = 0
	client, err := NewClient(opts)
	require.NoError(t, err)

	_, err = client.GetUserByIdentifier(context.Background(), "13800138000", "", "")
	var sdkErr *Error
	require.True(t, errors.As(err, &sdkErr))
	require.Equal(t, ErrCodeServerError, sdkErr.Code)
	require.Equal(t, http.StatusServiceUnavailable, sdkErr.StatusCode)
}

// Errors that do not report a response status leave StatusCode at zero.
func TestErrorWithoutResponseStatus(t *testing.T) {
	ctx := context.Background()
	require.Zero(t, NewError(ErrCodeRequestFailed, "message", nil).StatusCode)

	t.Run("no response", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		url := server.URL
		server.Close()
		client, err := NewClient(DefaultOptions().WithBaseURL(url))
		require.NoError(t, err)
		_, err = client.GetUserByIdentifier(ctx, "13800138000", "", "")
		var sdkErr *Error
		require.True(t, errors.As(err, &sdkErr))
		require.Equal(t, ErrCodeRequestFailed, sdkErr.Code)
		require.Zero(t, sdkErr.StatusCode)
	})

	t.Run("undecodable body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not json")) //nolint:errcheck // test server
		}))
		t.Cleanup(server.Close)
		client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
		require.NoError(t, err)
		_, err = client.GetUserByIdentifier(ctx, "13800138000", "", "")
		var sdkErr *Error
		require.True(t, errors.As(err, &sdkErr))
		require.Equal(t, ErrCodeInvalidResponse, sdkErr.Code)
		require.Zero(t, sdkErr.StatusCode)
	})

	t.Run("invalid argument", func(t *testing.T) {
		client, err := NewClient(DefaultOptions().WithBaseURL("http://127.0.0.1:1"))
		require.NoError(t, err)
		_, err = client.GetUserByIdentifier(ctx, "", "", "")
		var sdkErr *Error
		require.True(t, errors.As(err, &sdkErr))
		require.Equal(t, ErrCodeInvalidConfig, sdkErr.Code)
		require.Zero(t, sdkErr.StatusCode)
	})
}
