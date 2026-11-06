package warden

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type logBoundaryTransport func(*http.Request) (*http.Response, error)

func (f logBoundaryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSDKTransportFailureLogsOmitRawErrorsAndRetryURLs(t *testing.T) {
	for _, operation := range []string{"list", "pagination", "lookup", "membership"} {
		t.Run(operation, func(t *testing.T) {
			logger := &mockLogger{}
			sentinel := errors.New("private-response-canary")
			transport := &http.Transport{}
			transport.RegisterProtocol("http", logBoundaryTransport(func(*http.Request) (*http.Response, error) { return nil, sentinel }))
			retry := DefaultRetryOptions()
			retry.MaxRetries = 1
			retry.RetryDelay = time.Nanosecond
			retry.MaxRetryDelay = time.Nanosecond
			client, err := NewClient(DefaultOptions().WithBaseURL("http://directory.invalid").WithTransport(transport).WithLogger(logger).WithRetry(retry))
			require.NoError(t, err)
			defer client.Close()
			switch operation {
			case "list":
				_, err = client.GetUsers(context.Background())
			case "pagination":
				_, err = client.GetUsersPaginated(context.Background(), 1, 10)
			case "lookup":
				_, err = client.GetUserByIdentifier(context.Background(), "15555550123", "", "")
			case "membership":
				require.False(t, client.CheckUserInList(context.Background(), "", "private@example.invalid"))
			}
			if operation != "membership" {
				require.ErrorIs(t, err, sentinel)
			}
			logs := strings.Join(append(append(append(logger.debugs, logger.warns...), logger.errors...), logger.infos...), "\n")
			require.NotContains(t, logs, sentinel.Error())
			require.NotContains(t, logs, "15555550123")
			require.NotContains(t, logs, "private@example.invalid")
			require.Contains(t, logs, "code=REQUEST_FAILED")
			require.Contains(t, logs, "Retrying request")
		})
	}
}

func TestSDKResponseFailureLogsOmitBody(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusUnauthorized, http.StatusTeapot} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			logger := &mockLogger{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, err := io.WriteString(w, "private-response-canary")
				require.NoError(t, err)
			}))
			defer server.Close()
			client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithLogger(logger))
			require.NoError(t, err)
			defer client.Close()
			_, err = client.GetUserByIdentifier(context.Background(), "15555550123", "", "")
			require.Error(t, err)
			if status == http.StatusTeapot {
				require.Contains(t, err.Error(), "private-response-canary", "returned error remains compatible")
			}
			logs := strings.Join(append(append(logger.debugs, logger.warns...), logger.errors...), "\n")
			require.NotContains(t, logs, "private-response-canary")
			require.Contains(t, logs, fmt.Sprintf("status: %d", status))
		})
	}
}

func TestLogErrorCodeUsesOnlySDKCategories(t *testing.T) {
	for _, code := range []string{ErrCodeInvalidConfig, ErrCodeRequestFailed, ErrCodeInvalidResponse, ErrCodeUnauthorized, ErrCodeNotFound, ErrCodeServerError, ErrCodeSourceFreshness} {
		require.Equal(t, code, logErrorCode(fmt.Errorf("wrapped: %w", NewError(code, "private-message", nil))))
	}
	require.Equal(t, "UNKNOWN", logErrorCode(errors.New("private-message")))
	require.Equal(t, "UNKNOWN", logErrorCode(NewError("private-code", "message", nil)))
	require.Equal(t, "UNKNOWN", logErrorCode(nil))
}

type logBoundaryBody struct {
	io.Reader
	failure error
}

func (b logBoundaryBody) Close() error { return b.failure }

type logBoundaryReader struct{ failure error }

func (b logBoundaryReader) Read([]byte) (int, error) { return 0, b.failure }

func TestSDKBodyFailureLogsOmitRawErrors(t *testing.T) {
	for _, retryResponse := range []bool{false, true} {
		t.Run(fmt.Sprint(retryResponse), func(t *testing.T) {
			logger := &mockLogger{}
			sentinel := errors.New("private-body-error-canary")
			transport := &http.Transport{}
			transport.RegisterProtocol("http", logBoundaryTransport(func(*http.Request) (*http.Response, error) {
				status := http.StatusOK
				var reader io.Reader = logBoundaryReader{failure: sentinel}
				if retryResponse {
					status = http.StatusServiceUnavailable
					reader = strings.NewReader("ignored")
				}
				return &http.Response{StatusCode: status, Header: http.Header{}, Body: logBoundaryBody{Reader: reader, failure: sentinel}}, nil
			}))
			retry := DefaultRetryOptions()
			retry.MaxRetries = 1
			retry.RetryDelay = time.Nanosecond
			retry.MaxRetryDelay = time.Nanosecond
			client, err := NewClient(DefaultOptions().WithBaseURL("http://directory.invalid").WithTransport(transport).WithLogger(logger).WithRetry(retry))
			require.NoError(t, err)
			defer client.Close()
			_, err = client.GetUsers(context.Background())
			require.Error(t, err)
			if !retryResponse {
				require.ErrorIs(t, err, sentinel)
			}
			logs := strings.Join(append(append(logger.debugs, logger.warns...), logger.errors...), "\n")
			require.NotContains(t, logs, sentinel.Error())
		})
	}
}
