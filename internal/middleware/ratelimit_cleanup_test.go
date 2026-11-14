package middleware

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The cleanup loop removes keys whose requests have all left the window even
// when no further request arrives, and ends when the limiter is stopped.
func TestRateLimiterCleanupLoopDropsIdleKeys(t *testing.T) {
	var clockMu sync.Mutex
	current := time.Unix(1_700_000_000, 0)
	limiter := &RateLimiter{
		now: func() time.Time {
			clockMu.Lock()
			defer clockMu.Unlock()
			return current
		},
		entries: make(map[string]*rateLimitEntry),
		stopCh:  make(chan struct{}),
		done:    make(chan struct{}),
		epoch:   current,
		window:  time.Minute,
		rate:    2,
		maxKeys: 10,
	}
	tracked := func() int {
		limiter.mu.Lock()
		defer limiter.mu.Unlock()
		return len(limiter.entries)
	}
	go limiter.cleanupLoop(time.Millisecond)

	allowed, _ := limiter.Allow("idle")
	require.True(t, allowed)
	allowed, _ = limiter.Allow("busy")
	require.True(t, allowed)

	// Inside the window both keys are kept, however often the loop runs.
	time.Sleep(20 * time.Millisecond)
	require.Equal(t, 2, tracked())

	// "busy" is seen again later; "idle" is not.
	clockMu.Lock()
	current = current.Add(50 * time.Second)
	clockMu.Unlock()
	allowed, _ = limiter.Allow("busy")
	require.True(t, allowed)
	clockMu.Lock()
	current = current.Add(20 * time.Second)
	clockMu.Unlock()

	require.Eventually(t, func() bool { return tracked() == 1 }, 5*time.Second, 5*time.Millisecond)
	limiter.mu.Lock()
	_, idleKept := limiter.entries["idle"]
	busy := limiter.entries["busy"]
	limiter.mu.Unlock()
	require.False(t, idleKept)
	require.NotNil(t, busy)
	require.Len(t, busy.stamps, 1, "the request that left the window is pruned, the later one kept")

	limiter.Stop()
	select {
	case <-limiter.done:
	default:
		t.Fatal("the cleanup loop did not end on Stop")
	}
}
