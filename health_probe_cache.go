package main

import (
	"context"
	"sync"
	"time"

	health "github.com/soulteary/health-kit/v4"
)

// redisProbeCacheTTL is how long one Redis probe result answers health
// requests. The health routes are unauthenticated and outside the rate
// limiters, so without it every request is a Redis round trip that any caller
// can trigger as often as it likes. A few seconds is far below the interval of
// an orchestrator probe, so a real state change is still reported promptly.
const redisProbeCacheTTL = 3 * time.Second

// cachedChecker answers from the last healthy result of the wrapped checker
// while that result is younger than ttl. Concurrent callers wait for one probe
// instead of each starting their own. A result that is not healthy is never
// reused: while the dependency is down every request probes again, so recovery
// is reported by the first request after it.
//
//nolint:govet // fieldalignment: fields grouped by role (configuration, then guarded state)
type cachedChecker struct {
	inner health.Checker
	now   func() time.Time
	ttl   time.Duration

	mu     sync.Mutex
	result health.CheckResult
	at     time.Time
	valid  bool
}

func newCachedChecker(inner health.Checker, ttl time.Duration) *cachedChecker {
	return &cachedChecker{inner: inner, now: time.Now, ttl: ttl}
}

// Name implements health.Checker.
func (c *cachedChecker) Name() string { return c.inner.Name() }

// Check implements health.Checker.
func (c *cachedChecker) Check(ctx context.Context) health.CheckResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid && c.now().Sub(c.at) < c.ttl {
		return c.result
	}
	c.result = c.inner.Check(ctx)
	c.at = c.now()
	c.valid = c.result.Status == health.StatusHealthy
	return c.result
}
