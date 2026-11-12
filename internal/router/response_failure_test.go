package router

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

var errResponseWrite = errors.New("response disconnected")

type failedResponseWriter struct {
	header http.Header
	writes int
}

func (w *failedResponseWriter) Header() http.Header       { return w.header }
func (w *failedResponseWriter) WriteHeader(int)           {}
func (w *failedResponseWriter) Write([]byte) (int, error) { w.writes++; return 0, errResponseWrite }
func newFailedWriter() *failedResponseWriter              { return &failedResponseWriter{header: make(http.Header)} }

func TestJSONEncodingAndWriteFailures(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	require.Error(t, encodeJSONResponse(rec, req, make(chan int)))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.ErrorIs(t, encodeJSONResponse(newFailedWriter(), req, map[string]int{"value": 1}), errResponseWrite)
	writer := newFailedWriter()
	WriteJSONError(writer, 503, "unavailable")
	require.Positive(t, writer.writes)
	for _, size := range []int{1, define.SMALL_DATA_THRESHOLD, define.LARGE_DATA_THRESHOLD} {
		users := make([]define.AllowListUser, size)
		for i := range users {
			users[i].Mail = "member@example.com"
		}
		c := cache.NewSafeUserCache()
		c.Set(users)
		for _, url := range []string{"/", "/?page=1"} {
			w := newFailedWriter()
			JSON(c, []string{"mail"})(w, httptest.NewRequest(http.MethodGet, url, http.NoBody))
			require.Positive(t, w.writes)
		}
	}
	empty, total, pages := paginate(nil, 1, 10)
	require.Empty(t, empty)
	require.Zero(t, total)
	require.Zero(t, pages)
}

func TestLookupValidationAndWriteFailure(t *testing.T) {
	c := cache.NewSafeUserCache()
	c.Set([]define.AllowListUser{{Phone: "13800138000", Mail: "member@example.com", UserID: "member", Status: "active"}})
	handler := GetLookup(c, nil)
	for _, tc := range []struct {
		method, url string
		status      int
	}{
		{http.MethodPost, "/", 405}, {http.MethodGet, "/?identifier=" + strings.Repeat("a", define.MAX_IDENTIFIER_LENGTH+1), 400}, {http.MethodGet, "/?identifier=13900139000", 404},
	} {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest(tc.method, tc.url, http.NoBody))
		require.Equal(t, tc.status, w.Code)
	}
	w := newFailedWriter()
	handler(w, httptest.NewRequest(http.MethodGet, "/?identifier=13800138000", http.NoBody))
	require.Positive(t, w.writes)
	w = newFailedWriter()
	GetUserByIdentifier(c, nil)(w, httptest.NewRequest(http.MethodGet, "/?phone=13800138000", http.NoBody))
	require.Positive(t, w.writes)
}
