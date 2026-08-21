package middleware

import (
	"sync"
	"time"
)

// ReplayGuard records recently-seen request nonces so a captured, still-in-window
// HMAC v2 request cannot be replayed. Implementations must be safe for concurrent
// use.
//
// IMPORTANT (multi-replica limitation): the bundled in-memory implementation only
// protects a single process. When Warden runs as multiple replicas behind a load
// balancer, a replayed request routed to a *different* replica will NOT be caught
// unless a shared store is plugged in. The signature timestamp tolerance still
// bounds the replay window in that case. Provide a distributed implementation
// (e.g. Redis SETNX with TTL) via SetReplayGuard for real multi-replica replay
// protection. This is a known, documented limitation — do not assume multi-replica
// replay is solved by the default guard.
type ReplayGuard interface {
	// SeenBefore atomically records the key and reports whether it had already been
	// recorded within its TTL window. It returns true when the key is a replay.
	// The ttl bounds how long the key must be remembered; callers pass the signature
	// timestamp tolerance so memory is bounded by the accepted skew window.
	SeenBefore(key string, ttl time.Duration) bool
}

// memoryReplayGuard is a single-node ReplayGuard backed by a map with lazy TTL
// eviction plus a periodic sweep. It is intentionally simple and lock-based; the
// per-request work is O(1) amortized.
//
//nolint:govet // fieldalignment: field order chosen for readability
type memoryReplayGuard struct {
	mu      sync.Mutex
	seen    map[string]time.Time // key -> expiry
	lastGC  time.Time
	gcEvery time.Duration
	maxSize int
	nowFn   func() time.Time
}

// NewMemoryReplayGuard returns a single-node in-memory ReplayGuard. maxSize caps
// the number of tracked keys to bound memory under abuse; when exceeded the guard
// fails safe by treating new keys as replays (rejecting) until the sweep drains
// expired entries. A non-positive maxSize disables the cap.
func NewMemoryReplayGuard(maxSize int) *memoryReplayGuard {
	return &memoryReplayGuard{
		seen:    make(map[string]time.Time),
		gcEvery: time.Second,
		maxSize: maxSize,
		nowFn:   time.Now,
	}
}

// SeenBefore implements ReplayGuard.
func (g *memoryReplayGuard) SeenBefore(key string, ttl time.Duration) bool {
	if ttl <= 0 {
		ttl = time.Minute
	}
	now := g.nowFn()
	g.mu.Lock()
	defer g.mu.Unlock()

	g.gcLocked(now)

	if exp, ok := g.seen[key]; ok && exp.After(now) {
		return true
	}
	if g.maxSize > 0 && len(g.seen) >= g.maxSize {
		// Bounded: refuse to grow unbounded under a flood of unique nonces. Treating
		// this as a replay (reject) fails safe rather than allowing unlimited memory.
		if _, ok := g.seen[key]; !ok {
			return true
		}
	}
	g.seen[key] = now.Add(ttl)
	return false
}

// gcLocked evicts expired entries at most once per gcEvery. Caller holds mu.
func (g *memoryReplayGuard) gcLocked(now time.Time) {
	if now.Sub(g.lastGC) < g.gcEvery {
		return
	}
	g.lastGC = now
	for k, exp := range g.seen {
		if !exp.After(now) {
			delete(g.seen, k)
		}
	}
}
