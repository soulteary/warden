package warden

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

// sdkErrorCodes lists every ErrCode* constant declared in errors.go.
// TestSDKErrorCodeTableMatchesDeclarations fails when a constant is added
// there and not here.
var sdkErrorCodes = map[string]string{
	"ErrCodeInvalidConfig":   ErrCodeInvalidConfig,
	"ErrCodeRequestFailed":   ErrCodeRequestFailed,
	"ErrCodeInvalidResponse": ErrCodeInvalidResponse,
	"ErrCodeUnauthorized":    ErrCodeUnauthorized,
	"ErrCodeNotFound":        ErrCodeNotFound,
	"ErrCodeServerError":     ErrCodeServerError,
	"ErrCodeSourceFreshness": ErrCodeSourceFreshness,
	"ErrCodeRateLimited":     ErrCodeRateLimited,
}

func TestLogErrorCodeUsesOnlySDKCategories(t *testing.T) {
	for name, code := range sdkErrorCodes {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, code, logErrorCode(NewError(code, "private-message", nil)))
			require.Equal(t, code, logErrorCode(fmt.Errorf("wrapped: %w", NewError(code, "private-message", nil))))
		})
	}
	require.Equal(t, "UNKNOWN", logErrorCode(errors.New("private-message")))
	require.Equal(t, "UNKNOWN", logErrorCode(NewError("private-code", "message", nil)))
	require.Equal(t, "UNKNOWN", logErrorCode(nil))
}

// TestSDKErrorCodeTableMatchesDeclarations reads the constant declarations in
// errors.go, so a code added there without a row in sdkErrorCodes, and so
// without a logErrorCode assertion, fails this test.
func TestSDKErrorCodeTableMatchesDeclarations(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "errors.go", nil, parser.SkipObjectResolution)
	require.NoError(t, err)
	declared := map[string]string{}
	for _, decl := range file.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.CONST {
			continue
		}
		for _, spec := range group.Specs {
			values, ok := spec.(*ast.ValueSpec)
			require.True(t, ok)
			for i, name := range values.Names {
				if !strings.HasPrefix(name.Name, "ErrCode") {
					continue
				}
				require.Len(t, values.Values, len(values.Names), "%s must have its own value", name.Name)
				literal, ok := values.Values[i].(*ast.BasicLit)
				require.True(t, ok, "%s must be a string literal", name.Name)
				value, err := strconv.Unquote(literal.Value)
				require.NoError(t, err)
				declared[name.Name] = value
			}
		}
	}
	require.NotEmpty(t, declared)
	require.Equal(t, declared, sdkErrorCodes)
}

func TestCheckUserInListLogsRateLimitedCode(t *testing.T) {
	logger := &mockLogger{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
		_, err := io.WriteString(w, "private-response-canary")
		assert.NoError(t, err)
	}))
	defer server.Close()
	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithLogger(logger))
	require.NoError(t, err)
	defer client.Close()

	require.False(t, client.CheckUserInList(context.Background(), "15555550123", "private@example.invalid"))

	logs := strings.Join(append(append(append(logger.debugs, logger.warns...), logger.errors...), logger.infos...), "\n")
	require.Contains(t, logs, "Failed to get user from Warden API: code=RATE_LIMITED (phone=")
	require.NotContains(t, logs, "code=UNKNOWN")
	require.Contains(t, logs, "Warden API returned non-200 status: 429")
	require.NotContains(t, logs, "private-response-canary")
	require.NotContains(t, logs, "15555550123")
	require.NotContains(t, logs, "private@example.invalid")
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
