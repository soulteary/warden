package main

import (
	"context"
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

	// A failed probe is not reused: recovery shows on the next request.
	status = health.StatusHealthy
	require.Equal(t, health.StatusHealthy, checker.Check(context.Background()).Status)
	require.Equal(t, 3, calls)
}
