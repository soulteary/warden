package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	loggerkit "github.com/soulteary/logger-kit/v3"
	"github.com/stretchr/testify/require"
)

// In production the error rewriter logs the detail it removes from the
// response. Client errors are routine and must not be logged as errors.
func TestProductionErrorDetailLogLevel(t *testing.T) {
	respond := map[string]func(w http.ResponseWriter, status int){
		"json": func(w http.ResponseWriter, status int) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"detail"}`)) //nolint:errcheck // test handler
		},
		"text": func(w http.ResponseWriter, status int) { http.Error(w, "detail", status) },
	}
	for status, wantLevel := range map[int]string{
		http.StatusBadRequest:          "info",
		http.StatusUnauthorized:        "info",
		http.StatusNotFound:            "info",
		http.StatusTooManyRequests:     "info",
		http.StatusInternalServerError: "error",
		http.StatusServiceUnavailable:  "error",
	} {
		for kind, write := range respond {
			var out bytes.Buffer
			log := loggerkit.New(loggerkit.Config{Level: loggerkit.DebugLevel, Format: loggerkit.FormatJSON, Output: &out})
			handler := ErrorHandlerMiddleware("production")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				write(w, status)
			}))
			req := loggerkit.SetLoggerInRequest(httptest.NewRequest(http.MethodGet, "/user", http.NoBody), log)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			require.Equal(t, status, rec.Code)
			require.NotContains(t, rec.Body.String(), "detail", "the response detail stays hidden")

			var lines []map[string]any
			scanner := bufio.NewScanner(&out)
			for scanner.Scan() {
				var line map[string]any
				require.NoError(t, json.Unmarshal(scanner.Bytes(), &line))
				lines = append(lines, line)
			}
			require.Len(t, lines, 1, "%d %s", status, kind)
			require.Equal(t, wantLevel, lines[0]["level"], "%d %s", status, kind)
			require.EqualValues(t, status, lines[0]["status_code"])
		}
	}
}
