package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = "198.51.100.9:4711"
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	assert.Equal(t, "198.51.100.9", ClientIP(req), "forwarded headers are not read here")

	req.RemoteAddr = "not-an-address"
	assert.Equal(t, "not-an-address", ClientIP(req))

	assert.Equal(t, "203.0.113.7", ClientIP(WithClientIP(req, "203.0.113.7")))
}
