package warden

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type responseReadFailure struct{ closed bool }

func (*responseReadFailure) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (r *responseReadFailure) Close() error           { r.closed = true; return errors.New("close failed") }

func TestSDKRequestConstructionAndIdentifierFailures(t *testing.T) {
	_, err := NewClient(nil)
	require.Error(t, err)
	client, err := NewClient(DefaultOptions().WithBaseURL("http://example.com"))
	require.NoError(t, err)
	defer client.Close()
	_, err = client.GetUserByIdentifier(context.Background(), "", "", "")
	require.Error(t, err)
	_, err = client.GetUserByIdentifier(context.Background(), "phone", "mail", "")
	require.Error(t, err)
	require.False(t, client.CheckUserInList(context.Background(), "", ""))
	client.baseURL = "http://bad\nurl"
	_, err = client.GetUsers(context.Background())
	require.Error(t, err)
	_, err = client.GetUsersPaginated(context.Background(), 1, 10)
	require.Error(t, err)
	client.baseURL = "http://example.com"
	//nolint:staticcheck // Exercise request construction failure with a nil context.
	_, err = client.GetUsersPaginated(nil, 1, 10)
	require.Error(t, err)
	//nolint:staticcheck // Exercise request construction failure with a nil context.
	_, err = client.GetUserByIdentifier(nil, "13800138000", "", "")
	require.Error(t, err)
}

func TestSDKDataEndpointsRejectMalformedResponses(t *testing.T) {
	for _, status := range []int{200, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, writeErr := io.WriteString(w, "invalid")
				require.NoError(t, writeErr)
			}))
			defer server.Close()
			opts := DefaultOptions().WithBaseURL(server.URL)
			opts.Retry = DefaultRetryOptions()
			opts.Retry.MaxRetries = 0
			c, err := NewClient(opts)
			require.NoError(t, err)
			defer c.Close()
			_, err = c.GetUsersPaginated(context.Background(), 1, 10)
			require.Error(t, err)
			_, err = c.GetUserByIdentifier(context.Background(), "13800138000", "", "")
			require.Error(t, err)
			require.False(t, c.CheckUserInList(context.Background(), "13800138000", ""))
		})
	}
}

func TestSDKDataEndpointsPropagateCancelledRequests(t *testing.T) {
	c, err := NewClient(DefaultOptions().WithBaseURL("http://127.0.0.1:1"))
	require.NoError(t, err)
	defer c.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = c.GetUsersPaginated(ctx, 1, 10)
	require.Error(t, err)
	_, err = c.GetUserByIdentifier(ctx, "13800138000", "", "")
	require.Error(t, err)
}

func TestSDKResponseReadAndCloseFailures(t *testing.T) {
	c, err := NewClient(DefaultOptions().WithBaseURL("http://example.com"))
	require.NoError(t, err)
	defer c.Close()
	require.NoError(t, c.applyResponseLimit(nil))
	require.NoError(t, c.applyResponseLimit(&http.Response{}))
	c.maxResponseBytes = 0
	body := &responseReadFailure{}
	require.ErrorIs(t, c.applyResponseLimit(&http.Response{Body: body}), io.ErrUnexpectedEOF)
	require.True(t, body.closed)
	err = c.checkResponseStatus(&http.Response{StatusCode: 503, Body: &responseReadFailure{}})
	require.Error(t, err)
	require.NoError(t, c.applyResponseLimit(&http.Response{Body: io.NopCloser(strings.NewReader("[]"))}))
	c.retry = nil
	require.Zero(t, c.calculateRetryDelay(0))
	c.retry = &RetryOptions{RetryDelay: time.Second, BackoffMultiplier: 2}
	require.Equal(t, time.Second, c.calculateRetryDelay(-1))
	c.retry.RetryDelay = 0
	require.Zero(t, c.calculateRetryDelay(1))
}

func TestSDKRequestSigningAndRetryExhaustion(t *testing.T) {
	opts := DefaultOptions().WithBaseURL("http://127.0.0.1:1")
	opts.Retry = DefaultRetryOptions()
	opts.Retry.MaxRetries = 1
	opts.Retry.RetryDelay = time.Nanosecond
	c, err := NewClient(opts)
	require.NoError(t, err)
	defer c.Close()
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:1", http.NoBody)
	require.NoError(t, err)
	_, err = c.doRequestWithRetry(context.Background(), req)
	require.Error(t, err)
	c.signer = newHMACSigner("key", "secret", "", nil)
	req.GetBody = func() (io.ReadCloser, error) { return nil, io.ErrUnexpectedEOF }
	req.Body = io.NopCloser(strings.NewReader("body"))
	_, err = c.doRequestWithRetry(context.Background(), req)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	c.signer = nil
	c.retry.MaxRetries = -1
	_, err = c.doRequestWithRetry(context.Background(), req)
	require.Error(t, err)
}
