package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	health "github.com/soulteary/health-kit/v4"
	"github.com/stretchr/testify/require"
)

func TestCachedCheckerExpiresAndReportsChanges(t *testing.T) {
	calls := 0
	status := health.StatusHealthy
	inner := health.NewCheckerFunc("redis", func(context.Context) health.CheckResult {
		calls++
		return health.CheckResult{Name: "redis", Status: status}
	})
	now := time.Unix(1000, 0)
	checker := newCachedChecker(inner, 3*time.Second)
	checker.now = func() time.Time { return now }

	require.Equal(t, "redis", checker.Name())
	require.Equal(t, health.StatusHealthy, checker.Check(context.Background()).Status)
	status = health.StatusUnhealthy
	now = now.Add(2 * time.Second)
	require.Equal(t, health.StatusHealthy, checker.Check(context.Background()).Status, "within the window the cached result answers")
	require.Equal(t, 1, calls)

	now = now.Add(time.Second)
	require.Equal(t, health.StatusUnhealthy, checker.Check(context.Background()).Status, "an expired result is probed again")
	require.Equal(t, 2, calls)

	// A failed probe is reused for the shorter failure interval only.
	status = health.StatusHealthy
	now = now.Add(999 * time.Millisecond)
	require.Equal(t, health.StatusUnhealthy, checker.Check(context.Background()).Status)
	require.Equal(t, 2, calls)
	now = now.Add(time.Millisecond)
	require.Equal(t, health.StatusHealthy, checker.Check(context.Background()).Status, "recovery shows after the failure interval")
	require.Equal(t, 3, calls)
}

// While the dependency is down, callers that arrive together share one probe.
// Each of them used to run its own, one after the other, so the last caller
// waited for all the probes before it.
func TestCachedCheckerSharesOneFailingProbe(t *testing.T) {
	const callers = 20
	var calls atomic.Int64
	release := make(chan struct{})
	inner := health.NewCheckerFunc("redis", func(context.Context) health.CheckResult {
		calls.Add(1)
		<-release
		return health.CheckResult{Name: "redis", Status: health.StatusUnhealthy, Error: "connection refused"}
	})
	checker := newCachedChecker(inner, 3*time.Second)

	results := make(chan health.CheckResult, callers)
	var started sync.WaitGroup
	for i := 0; i < callers; i++ {
		started.Add(1)
		go func() {
			started.Done()
			results <- checker.Check(context.Background())
		}()
	}
	started.Wait()
	// Let the callers reach the probe before it finishes.
	require.Eventually(t, func() bool { return calls.Load() >= 1 }, time.Second, time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	close(release)

	for i := 0; i < callers; i++ {
		select {
		case result := <-results:
			require.Equal(t, health.StatusUnhealthy, result.Status)
			require.Equal(t, "connection refused", result.Error)
		case <-time.After(5 * time.Second):
			t.Fatal("a caller did not receive the shared result")
		}
	}
	require.EqualValues(t, 1, calls.Load(), "%d concurrent callers start one probe", callers)
}

// A caller that gives up neither cancels the shared probe nor decides its
// result: the next caller receives what the probe found.
func TestCachedCheckerIgnoresTheCancellationOfOneCaller(t *testing.T) {
	var calls atomic.Int64
	entered := make(chan struct{})
	release := make(chan struct{})
	inner := health.NewCheckerFunc("redis", func(ctx context.Context) health.CheckResult {
		calls.Add(1)
		close(entered)
		<-release
		if err := ctx.Err(); err != nil {
			return health.CheckResult{Name: "redis", Status: health.StatusUnhealthy, Error: err.Error()}
		}
		return health.CheckResult{Name: "redis", Status: health.StatusHealthy}
	})
	checker := newCachedChecker(inner, 3*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan health.CheckResult, 1)
	go func() { first <- checker.Check(ctx) }()
	<-entered
	cancel()
	select {
	case result := <-first:
		require.Equal(t, health.StatusUnhealthy, result.Status, "the caller that gave up reports its own cancellation")
		require.Equal(t, context.Canceled.Error(), result.Error)
		require.Equal(t, "redis", result.Name)
	case <-time.After(5 * time.Second):
		t.Fatal("a cancelled caller kept waiting for the probe")
	}

	second := make(chan health.CheckResult, 1)
	go func() { second <- checker.Check(context.Background()) }()
	close(release)
	require.Equal(t, health.StatusHealthy, (<-second).Status, "the probe was not cancelled with the first caller")
	require.Equal(t, health.StatusHealthy, checker.Check(context.Background()).Status, "and its result is the cached one")
	require.EqualValues(t, 1, calls.Load())
}

// The probe is bounded even though it does not follow the caller's context.
func TestCachedCheckerBoundsTheProbe(t *testing.T) {
	inner := health.NewCheckerFunc("redis", func(ctx context.Context) health.CheckResult {
		<-ctx.Done()
		return health.CheckResult{Name: "redis", Status: health.StatusUnhealthy, Error: ctx.Err().Error()}
	})
	checker := newCachedChecker(inner, time.Minute)
	checker.timeout = 20 * time.Millisecond

	result := checker.Check(context.Background())
	require.Equal(t, health.StatusUnhealthy, result.Status)
	require.Equal(t, context.DeadlineExceeded.Error(), result.Error)
}

// A panic in the wrapped checker is reported as a result: the probe runs in
// its own goroutine, where an unrecovered panic would end the process.
func TestCachedCheckerReportsAPanicAsUnhealthy(t *testing.T) {
	inner := health.NewCheckerFunc("redis", func(context.Context) health.CheckResult { panic("boom") })
	result := newCachedChecker(inner, 3*time.Second).Check(context.Background())
	require.Equal(t, health.StatusUnhealthy, result.Status)
	require.Contains(t, result.Error, "boom")
	require.Equal(t, "redis", result.Name)
}
