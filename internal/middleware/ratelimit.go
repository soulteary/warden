package middleware

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	middlewarekit "github.com/soulteary/middleware-kit/v3"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

// Rate-limit scopes. They are the only values passed to RateLimitConfig.OnLimited,
// so they are safe to use as a metric label.
const (
	RateLimitScopeIP         = "ip"
	RateLimitScopeCredential = "credential"
)

// rateLimitEvictionDivisor is the share of the table freed when it is full: a
// tenth. Evicting one entry would leave it full and repeat the pass for every
// further new key.
const rateLimitEvictionDivisor = 10

// RateLimiter is an in-memory sliding-window limiter: no interval of one window
// contains more than rate accepted requests per key. It differs from the
// middleware-kit limiter in two ways the routes need: a reservation can be
// returned (Release), and a refusal reports when the next request will fit.
//
// Memory is bounded: at most maxKeys keys are tracked, each holding at most
// rate timestamps. When the table is full the least recently seen keys are
// evicted.
type RateLimiter struct {
	now     func() time.Time
	entries map[string]*rateLimitEntry
	stopCh  chan struct{}
	done    chan struct{}
	epoch   time.Time
	window  time.Duration
	rate    int
	maxKeys int
	mu      sync.Mutex
	stop    sync.Once
}

// rateLimitEntry holds the accepted requests of one key that are still inside
// the window, oldest first, as offsets from the limiter epoch. The offsets come
// from time.Time.Sub, which uses the monotonic clock, so a wall-clock step does
// not extend or shorten a window.
type rateLimitEntry struct {
	stamps   []time.Duration
	lastSeen time.Duration
}

// NewRateLimiter creates a limiter that accepts rate requests per window for
// each key and tracks at most maxKeys keys. Non-positive arguments fall back to
// the defaults. Stop must be called to end its cleanup goroutine.
func NewRateLimiter(rate int, window time.Duration, maxKeys int) *RateLimiter {
	if rate < 1 {
		rate = define.DEFAULT_RATE_LIMIT
	}
	if window <= 0 {
		window = define.DEFAULT_RATE_LIMIT_WINDOW
	}
	if maxKeys < 1 {
		maxKeys = define.MAX_VISITORS_MAP_SIZE
	}
	l := &RateLimiter{
		now:     time.Now,
		entries: make(map[string]*rateLimitEntry),
		stopCh:  make(chan struct{}),
		done:    make(chan struct{}),
		epoch:   time.Now(),
		window:  window,
		rate:    rate,
		maxKeys: maxKeys,
	}
	go l.cleanupLoop(define.RATE_LIMIT_CLEANUP_INTERVAL)
	return l
}

// Reservation identifies one request recorded by Reserve, so that Release can
// return that request and no other.
type Reservation struct {
	at time.Duration
}

// Allow records a request for key when the window has room for it. Otherwise
// it records nothing and returns the time until the oldest recorded request
// leaves the window.
func (l *RateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	_, allowed, retryAfter = l.Reserve(key)
	return allowed, retryAfter
}

// Reserve is Allow that also returns a handle for Release.
func (l *RateLimiter) Reserve(key string) (reservation Reservation, allowed bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	at := l.now().Sub(l.epoch)
	e, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= l.maxKeys {
			l.evict(at)
		}
		e = &rateLimitEntry{}
		l.entries[key] = e
	}
	e.lastSeen = at
	e.prune(at - l.window)
	if len(e.stamps) >= l.rate {
		return Reservation{}, false, e.stamps[0] + l.window - at
	}
	e.stamps = append(e.stamps, at)
	return Reservation{at: at}, true, 0
}

// Release returns the request recorded by the given reservation, so a request
// that was admitted provisionally does not count against the key. It removes
// that request's own record: requests complete in a different order than they
// were admitted, and removing the newest record instead could erase a later
// request that must stay counted. A reservation that has already left the
// window is ignored. Call Release at most once per reservation: two requests
// admitted at the same instant have equal records.
func (l *RateLimiter) Release(key string, reservation Reservation) {
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.entries[key]
	if !ok {
		return
	}
	// Records are ordered, and a release usually targets a recent one.
	for i := len(e.stamps) - 1; i >= 0 && e.stamps[i] >= reservation.at; i-- {
		if e.stamps[i] == reservation.at {
			e.stamps = append(e.stamps[:i], e.stamps[i+1:]...)
			break
		}
	}
	if len(e.stamps) == 0 {
		delete(l.entries, key)
	}
}

// Stop ends the cleanup goroutine. It is safe to call more than once.
func (l *RateLimiter) Stop() {
	l.stop.Do(func() {
		close(l.stopCh)
		<-l.done
	})
}

// prune drops the requests that have left the window.
func (e *rateLimitEntry) prune(cutoff time.Duration) {
	i := 0
	for i < len(e.stamps) && e.stamps[i] <= cutoff {
		i++
	}
	e.stamps = e.stamps[i:]
}

// evict makes room for a new key. Keys with no request left in the window are
// dropped first; if the table is still full, the least recently seen tenth
// goes. The caller holds the lock.
func (l *RateLimiter) evict(at time.Duration) {
	l.sweep(at)
	headroom := l.maxKeys / rateLimitEvictionDivisor
	if headroom < 1 {
		headroom = 1
	}
	remove := len(l.entries) - l.maxKeys + headroom
	if remove <= 0 {
		return
	}
	keys := make([]string, 0, len(l.entries))
	for key := range l.entries {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return l.entries[keys[i]].lastSeen < l.entries[keys[j]].lastSeen
	})
	for _, key := range keys[:remove] {
		delete(l.entries, key)
	}
}

// sweep drops every key with no request left in the window. The caller holds
// the lock.
func (l *RateLimiter) sweep(at time.Duration) {
	for key, e := range l.entries {
		e.prune(at - l.window)
		if len(e.stamps) == 0 {
			delete(l.entries, key)
		}
	}
}

func (l *RateLimiter) cleanupLoop(interval time.Duration) {
	defer close(l.done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.mu.Lock()
			l.sweep(l.now().Sub(l.epoch))
			l.mu.Unlock()
		case <-l.stopCh:
			return
		}
	}
}

// RateLimits holds the two limiters the routes use. IP is keyed by client
// address and counts requests that did not authenticate; Credential is keyed by
// the authenticated principal.
type RateLimits struct {
	IP         *RateLimiter
	Credential *RateLimiter
}

// NewRateLimits creates both limiters with a shared window.
func NewRateLimits(ipRate, credentialRate int, window time.Duration) *RateLimits {
	if credentialRate < 1 {
		credentialRate = define.DEFAULT_RATE_LIMIT_AUTHENTICATED
	}
	return &RateLimits{
		IP:         NewRateLimiter(ipRate, window, define.MAX_VISITORS_MAP_SIZE),
		Credential: NewRateLimiter(credentialRate, window, define.MAX_VISITORS_MAP_SIZE),
	}
}

// Stop stops both limiters. It is safe on a nil receiver.
func (s *RateLimits) Stop() {
	if s == nil {
		return
	}
	s.IP.Stop()
	s.Credential.Stop()
}

// RateLimitConfig configures the rate-limit middleware.
type RateLimitConfig struct {
	Limits             *RateLimits
	TrustedProxyConfig *middlewarekit.TrustedProxyConfig
	// OnLimited, when set, is called with the scope of each refused request.
	OnLimited func(scope string)
}

// RateLimitByIP limits every request per client address. It is for routes that
// do not authenticate.
func RateLimitByIP(cfg RateLimitConfig) func(http.Handler) http.Handler {
	return RateLimitAuthenticated(cfg, func(next http.Handler) http.Handler { return next })
}

// RateLimitAuthenticated wraps an authentication middleware so that failed and
// successful requests are counted separately.
//
// A request first takes one unit of its client address's budget; an address
// with no budget left is refused with 429 before auth runs. When auth passes
// the request on with a principal (see WithPrincipal), the unit is returned and
// the request is counted against that principal instead (per client address
// for the shared API key). When auth rejects the request, or passes it on
// without a principal, the unit stays spent.
//
// Taking the unit before auth, not after a failure, keeps the bound strict
// under concurrency: at most the per-address rate of credential checks can fail
// or be in flight per window, however many connections an address opens.
func RateLimitAuthenticated(cfg RateLimitConfig, auth func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := middlewarekit.GetClientIP(r, cfg.TrustedProxyConfig)
			reservation, ok, retryAfter := cfg.Limits.IP.Reserve(clientIP)
			if !ok {
				cfg.reject(w, r, RateLimitScopeIP, clientIP, retryAfter)
				return
			}
			// Handlers record this address in the audit trail.
			r = WithClientIP(r, clientIP)
			auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				principal := PrincipalFromContext(r.Context())
				if principal == "" {
					next.ServeHTTP(w, r)
					return
				}
				cfg.Limits.IP.Release(clientIP, reservation)
				// The API key is one shared secret and identifies no caller,
				// so its budget is kept per client address. A single budget
				// for every caller of the key would let one caller exhaust it
				// for all the others.
				if principal == principalAPIKey {
					principal += "|" + clientIP
				}
				if ok, retryAfter := cfg.Limits.Credential.Allow(principal); !ok {
					cfg.reject(w, r, RateLimitScopeCredential, principal, retryAfter)
					return
				}
				next.ServeHTTP(w, r)
			})).ServeHTTP(w, r)
		})
	}
}

// reject writes the 429 response. The status and body match what the
// middleware-kit limiter sent; Retry-After is the whole number of seconds,
// rounded up, until the limiter accepts another request for the key.
func (cfg RateLimitConfig) reject(w http.ResponseWriter, r *http.Request, scope, key string, retryAfter time.Duration) {
	if cfg.OnLimited != nil {
		cfg.OnLimited(scope)
	}
	logger.FromRequest(r).Warn().
		Str("scope", scope).
		Str("key", key).
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Msg("Rate limit exceeded")
	seconds := int64(math.Ceil(retryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
	http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
}

type principalContextKey struct{}

// WithPrincipal returns a copy of r that carries the identifier of the
// authenticated principal. The identifier must not be a secret: it is used as a
// rate-limit key and written to the log when that limit is reached.
func WithPrincipal(r *http.Request, principal string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), principalContextKey{}, principal))
}

// PrincipalFromContext returns the identifier set by WithPrincipal, or "" when
// the request did not authenticate with an identifiable credential.
func PrincipalFromContext(ctx context.Context) string {
	if principal, ok := ctx.Value(principalContextKey{}).(string); ok {
		return principal
	}
	return ""
}
