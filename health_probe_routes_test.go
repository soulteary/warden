package main

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// pingCounter counts the PING commands a client sends.
type pingCounter struct{ pings atomic.Int64 }

func (p *pingCounter) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) { return next(ctx, network, addr) }
}

func (p *pingCounter) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if strings.EqualFold(cmd.Name(), "ping") {
			p.pings.Add(1)
		}
		return next(ctx, cmd)
	}
}

func (p *pingCounter) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error { return next(ctx, cmds) }
}

// The health routes are unauthenticated and not rate limited, so a burst of
// health requests must not turn into a burst of Redis commands.
func TestHealthRequestsShareOneRedisProbe(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck // test cleanup
	counter := &pingCounter{}
	client.AddHook(counter)

	app := responseMetadataApp(t)
	app.redisClient = client
	app.redisEnabled = true
	mux := registerRoutes(app)

	for i := 0; i < 20; i++ {
		for _, path := range []string{"/health", "/healthcheck", "/v1/health"} {
			rec := serve(mux, apiKeyRequest(path, ""), "127.0.0.1:1234")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"redis"`)
		}
	}
	require.EqualValues(t, 1, counter.pings.Load(), "60 health requests within the cache window probe Redis once")
}
