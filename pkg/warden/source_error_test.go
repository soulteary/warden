package warden

import (
	"errors"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNegotiatedSource503Classification(t *testing.T) {
	for _, bound := range []bool{false, true} {
		client, err := NewClient(DefaultOptions().WithBaseURL("https://example.com"))
		require.NoError(t, err)
		t.Cleanup(client.Close)
		if bound {
			client.sourceMaxAge = time.Minute
		}
		for _, tc := range []struct {
			want         error
			name, reason string
		}{
			{ErrSourceStale, "expired", "expired"}, {ErrSourceUnknown, "unknown", "unknown"}, {ErrSourceFuture, "future", "future"},
			{nil, "unsupported", "other"}, {nil, "empty", ""},
		} {
			t.Run(tc.name+map[bool]string{true: "/bound", false: "/default"}[bound], func(t *testing.T) {
				response := &http.Response{StatusCode: 503, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("unavailable"))}
				response.Header.Set(HeaderSnapshotMetadata, "v1")
				response.Header.Set(HeaderSourceError, tc.reason)
				_, err := client.checkDataResponse(response)
				var sdkErr *Error
				require.ErrorAs(t, err, &sdkErr)
				if bound && tc.want != nil {
					require.Equal(t, ErrCodeSourceFreshness, sdkErr.Code)
					require.ErrorIs(t, err, tc.want)
					var original *Error
					require.True(t, errors.As(errors.Unwrap(sdkErr), &original))
					require.Equal(t, ErrCodeServerError, original.Code)
				} else {
					require.Equal(t, ErrCodeServerError, sdkErr.Code)
				}
			})
		}
	}
}

func TestMalformedSourceErrorMarkersRemainServerErrors(t *testing.T) {
	client, err := NewClient(DefaultOptions().WithBaseURL("https://example.com").WithSourceFreshness(time.Minute, 0))
	require.NoError(t, err)
	t.Cleanup(client.Close)
	for _, change := range []func(http.Header){
		func(h http.Header) { h.Del(HeaderSnapshotMetadata) },
		func(h http.Header) { h.Set(HeaderSnapshotMetadata, "v2") },
		func(h http.Header) { h.Add(HeaderSnapshotMetadata, "v1") },
		func(h http.Header) { h.Add(HeaderSourceError, "expired") },
		func(h http.Header) { h.Set(HeaderSourceError, strings.Repeat("x", 129)) },
		func(h http.Header) { h.Set(HeaderSourceError, "Expired") },
	} {
		response := &http.Response{StatusCode: 503, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("source expired"))}
		response.Header.Set(HeaderSnapshotMetadata, "v1")
		response.Header.Set(HeaderSourceError, "expired")
		change(response.Header)
		_, err := client.checkDataResponse(response)
		var sdkErr *Error
		require.ErrorAs(t, err, &sdkErr)
		require.Equal(t, ErrCodeServerError, sdkErr.Code)
	}
	for _, status := range []int{401, 404, 500, 502} {
		response := &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("unavailable"))}
		response.Header.Set(HeaderSnapshotMetadata, "v1")
		response.Header.Set(HeaderSourceError, "expired")
		_, err := client.checkDataResponse(response)
		// 404 still requires its ordinary success/absence evidence, never the 503 marker.
		if status == 404 {
			require.ErrorIs(t, err, ErrSourceUnknown)
		} else {
			require.NotErrorIs(t, err, ErrSourceStale)
		}
	}
}
