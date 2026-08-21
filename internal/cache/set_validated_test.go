package cache

import (
	"sync"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
)

func TestSetValidated_RejectsConflictKeepsGood(t *testing.T) {
	define.SetUserIDStrategy(define.UserIDStrategyLegacy)
	define.SetRequireExplicitUserID(false)

	c := NewSafeUserCache()

	good := []define.AllowListUser{
		{Phone: "13800000001", Mail: "a@example.com"},
		{Phone: "13800000002", Mail: "b@example.com"},
	}
	if err := c.SetValidated(good, identity.Options{}); err != nil {
		t.Fatalf("unexpected error setting good data: %v", err)
	}
	if c.Len() != 2 {
		t.Fatalf("expected 2 users, got %d", c.Len())
	}
	goodHash := c.GetHash()

	// A conflicting set (duplicate phone) must be rejected and NOT overwrite the cache.
	bad := []define.AllowListUser{
		{Phone: "13800000009", Mail: "x@example.com"},
		{Phone: "13800000009", Mail: "y@example.com"},
	}
	if err := c.SetValidated(bad, identity.Options{}); err == nil {
		t.Fatalf("expected conflict error, got nil")
	}
	if c.Len() != 2 {
		t.Fatalf("cache was mutated on conflict: len=%d", c.Len())
	}
	if c.GetHash() != goodHash {
		t.Fatalf("cache hash changed on rejected set")
	}
}

func TestSetValidated_DeterministicOrder(t *testing.T) {
	define.SetUserIDStrategy(define.UserIDStrategyLegacy)
	define.SetRequireExplicitUserID(false)

	c1 := NewSafeUserCache()
	c2 := NewSafeUserCache()

	a := define.AllowListUser{Phone: "13800000003", Mail: "c@example.com", UserID: "id-c"}
	b := define.AllowListUser{Phone: "13800000001", Mail: "a@example.com", UserID: "id-a"}
	d := define.AllowListUser{Phone: "13800000002", Mail: "b@example.com", UserID: "id-b"}

	if err := c1.SetValidated([]define.AllowListUser{a, b, d}, identity.Options{}); err != nil {
		t.Fatalf("c1: %v", err)
	}
	if err := c2.SetValidated([]define.AllowListUser{d, a, b}, identity.Options{}); err != nil {
		t.Fatalf("c2: %v", err)
	}
	if c1.GetHash() != c2.GetHash() {
		t.Fatalf("hashes differ for reordered inputs: %s vs %s", c1.GetHash(), c2.GetHash())
	}
}

func TestSetValidated_ConcurrentReadsDuringSet(t *testing.T) {
	define.SetUserIDStrategy(define.UserIDStrategyLegacy)
	define.SetRequireExplicitUserID(false)

	c := NewSafeUserCache()
	users := []define.AllowListUser{{Phone: "13800000001", Mail: "a@example.com"}}
	_ = c.SetValidated(users, identity.Options{})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Get()
			_ = c.Len()
		}()
	}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.SetValidated(users, identity.Options{})
		}()
	}
	wg.Wait()
	if c.Len() != 1 {
		t.Fatalf("expected 1 user, got %d", c.Len())
	}
}
