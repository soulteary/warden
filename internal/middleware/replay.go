package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/soulteary/warden/internal/logger"
)

// ReplayGuard records recently-seen request nonces so a captured, still-in-window
// HMAC v2 request cannot be replayed. Implementations must be safe for concurrent
// use.
//
// IMPORTANT (multi-replica limitation): the bundled in-memory implementation only
// protects a single process. When Warden runs as multiple replicas behind a load
// balancer, a replayed request routed to a *different* replica will NOT be caught
// unless a shared store is plugged in. Use NewRedisReplayGuard when multiple Warden
// replicas share Redis. The signature timestamp tolerance still bounds the replay
// window when the in-memory implementation is used.
type ReplayGuard interface {
	// SeenBefore atomically records the key and reports whether it had already been
	// recorded within its TTL window. It returns true when the key is a replay and
	// returns an error when the backing store cannot determine replay state.
	// The ttl bounds how long the key must be remembered; callers pass the signature
	// timestamp tolerance so memory is bounded by the accepted skew window.
	SeenBefore(key string, ttl time.Duration) (bool, error)
}

type redisSetNX interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
}

// redisReplayGuard uses Redis SET NX with an expiry so nonce insertion is atomic
// across all Warden replicas sharing the same Redis instance. Store failures reject
// the request rather than silently disabling replay protection.
type redisReplayGuard struct {
	client  redisSetNX
	logOnce sync.Once
}

// NewRedisReplayGuard creates a distributed replay guard backed by Redis.
func NewRedisReplayGuard(client *redis.Client) ReplayGuard {
	return newRedisReplayGuard(client)
}

func newRedisReplayGuard(client redisSetNX) ReplayGuard {
	return &redisReplayGuard{client: client}
}

// SeenBefore implements ReplayGuard.
func (g *redisReplayGuard) SeenBefore(key string, ttl time.Duration) (bool, error) {
	if g == nil || g.client == nil {
		return false, errors.New("redis replay guard is not initialized")
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	digest := sha256.Sum256([]byte(key))
	redisKey := "warden:hmac:replay:" + hex.EncodeToString(digest[:])
	inserted, err := g.client.SetNX(context.Background(), redisKey, "1", ttl).Result()
	if err != nil {
		g.logOnce.Do(func() {
			logger.GetLoggerKit().Error().Err(err).Msg("hmac: Redis replay guard failed closed")
		})
		return false, err
	}
	return !inserted, nil
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
func (g *memoryReplayGuard) SeenBefore(key string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = time.Minute
	}
	now := g.nowFn()
	g.mu.Lock()
	defer g.mu.Unlock()

	g.gcLocked(now)

	if exp, ok := g.seen[key]; ok && exp.After(now) {
		return true, nil
	}
	if g.maxSize > 0 && len(g.seen) >= g.maxSize {
		// Bounded: refuse to grow unbounded under a flood of unique nonces. Treating
		// this as a replay (reject) fails safe rather than allowing unlimited memory.
		if _, ok := g.seen[key]; !ok {
			return true, nil
		}
	}
	g.seen[key] = now.Add(ttl)
	return false, nil
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
