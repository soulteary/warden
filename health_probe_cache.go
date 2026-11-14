package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	health "github.com/soulteary/health-kit/v4"
)

// redisProbeCacheTTL is how long one healthy Redis probe result answers health
// requests. The health routes are unauthenticated and outside the rate
// limiters, so without it every request is a Redis round trip that any caller
// can trigger as often as it likes. A few seconds is far below the interval of
// an orchestrator probe, so a real state change is still reported promptly.
const redisProbeCacheTTL = 3 * time.Second

// redisProbeFailureTTL is how long a result that is not healthy answers health
// requests. It is shorter than redisProbeCacheTTL so that recovery shows
// within about a second, and long enough that requests arriving while Redis is
// down do not each start a probe that waits for a timeout.
const redisProbeFailureTTL = time.Second

// redisProbeTimeout bounds one shared probe. It equals the timeout of the
// health aggregator, the longest any caller waits for the result.
const redisProbeTimeout = 5 * time.Second

// cachedChecker answers from the last result of the wrapped checker while that
// result is fresh: ttl for a healthy result, failureTTL for any other.
//
// Callers that find no fresh result share one probe. The probe runs detached
// from the caller that started it: it is not cancelled when that caller gives
// up, so one caller's cancelled context cannot become the result every other
// caller receives. A caller whose own context ends stops waiting and reports
// that; the probe continues for the others.
//
//nolint:govet // fieldalignment: fields grouped by role (configuration, then guarded state)
type cachedChecker struct {
	inner      health.Checker
	now        func() time.Time
	ttl        time.Duration
	failureTTL time.Duration
	timeout    time.Duration

	mu       sync.Mutex
	result   health.CheckResult
	at       time.Time
	valid    bool
	inflight *probeCall
}

// probeCall is one probe in progress. result is set before done is closed.
type probeCall struct {
	done   chan struct{}
	result health.CheckResult
}

func newCachedChecker(inner health.Checker, ttl time.Duration) *cachedChecker {
	return &cachedChecker{inner: inner, now: time.Now, ttl: ttl, failureTTL: redisProbeFailureTTL, timeout: redisProbeTimeout}
}

// Name implements health.Checker.
func (c *cachedChecker) Name() string { return c.inner.Name() }

// Check implements health.Checker.
func (c *cachedChecker) Check(ctx context.Context) health.CheckResult {
	c.mu.Lock()
	if c.fresh() {
		result := c.result
		c.mu.Unlock()
		return result
	}
	call := c.inflight
	if call == nil {
		call = &probeCall{done: make(chan struct{})}
		c.inflight = call
		// The probe keeps the values of the context that started it (trace
		// identifiers) but not its cancellation or deadline.
		go c.probe(context.WithoutCancel(ctx), call)
	}
	c.mu.Unlock()

	select {
	case <-call.done:
		return call.result
	case <-ctx.Done():
		return health.CheckResult{Name: c.inner.Name(), Status: health.StatusUnhealthy, Error: ctx.Err().Error(), Timestamp: c.now()}
	}
}

// fresh reports whether the stored result may answer a request. The caller
// holds the lock.
func (c *cachedChecker) fresh() bool {
	if !c.valid {
		return false
	}
	ttl := c.failureTTL
	if c.result.Status == health.StatusHealthy {
		ttl = c.ttl
	}
	return c.now().Sub(c.at) < ttl
}

// probe runs the wrapped checker once, stores the result and releases the
// callers waiting for it.
func (c *cachedChecker) probe(ctx context.Context, call *probeCall) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	result := c.run(ctx)

	c.mu.Lock()
	c.result, c.at, c.valid = result, c.now(), true
	c.inflight = nil
	c.mu.Unlock()

	call.result = result
	close(call.done)
}

// run calls the wrapped checker. The probe has its own goroutine, outside the
// recovery of the health aggregator, so a panic is turned into a result here.
func (c *cachedChecker) run(ctx context.Context) (result health.CheckResult) {
	defer func() {
		if r := recover(); r != nil {
			result = health.CheckResult{Name: c.inner.Name(), Status: health.StatusUnhealthy, Error: fmt.Sprintf("health check panicked: %v", r), Timestamp: c.now()}
		}
	}()
	return c.inner.Check(ctx)
}
