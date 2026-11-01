package cache

import (
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
)

// PreparedRules owns a validated candidate that has not been published yet.
// Identity errors reject the whole input before per-record filtering is applied.
type PreparedRules struct {
	view         *ReadView
	inputVersion string
	input        []define.AllowListUser
}

// PrepareRules constructs the effective indexes without changing any live cache
// or logging per-record rejections. Empty-ruleset policy can inspect the candidate
// before deciding whether to publish it.
func PrepareRules(users []define.AllowListUser, opts identity.Options) (*PreparedRules, error) {
	// Detach the input and Scope once before allowing in-place normalization.
	res, err := identity.ValidateAndIndexOwnedUsers(cloneUsers(users), opts)
	if err != nil {
		return nil, err
	}
	view := buildReadView(res.Users, false)
	return &PreparedRules{view: view, input: res.Users, inputVersion: HashUserList(users)}, nil
}

// ReadView returns the immutable effective candidate.
func (p *PreparedRules) ReadView() *ReadView { return p.view }

// InputCount is the count before per-record format validation.
func (p *PreparedRules) InputCount() int { return len(p.input) }

// InputVersion describes the loaded input, independently of the effective set.
func (p *PreparedRules) InputVersion() string { return p.inputVersion }

// ApplyPrepared publishes an already prepared generation in one pointer swap.
// The candidate remains immutable and can be reused without revalidating users.
func (c *SafeUserCache) ApplyPrepared(p *PreparedRules) {
	reportSet(p.input, p.view)
	c.current.Store(p.view)
}
