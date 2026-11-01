package cache

import (
	"sync"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"github.com/stretchr/testify/require"
)

func TestPreparedRulesMatchesExistingValidation(t *testing.T) {
	for name, users := range map[string][]define.AllowListUser{
		"empty":          {},
		"valid":          {{Mail: "a@example.com", UserID: "a", Scope: []string{"read"}}},
		"partial":        {{Mail: "a@example.com", UserID: "a"}, {Mail: "not-an-email", UserID: "bad"}},
		"all_rejected":   {{Mail: "not-an-email", UserID: "bad"}},
		"no_identifiers": {{UserID: "empty"}},
	} {
		t.Run(name, func(t *testing.T) {
			validated, err := identity.ValidateAndIndexUsers(users, identity.Options{})
			require.NoError(t, err)
			legacy := newUserIndexes(validateUser)
			valid := make([]define.AllowListUser, 0, len(users))
			for i := range validated.Users {
				if primaryKeyForUser(validated.Users[i]) != "" {
					valid = append(valid, validated.Users[i])
				}
			}
			legacy.Set(valid)
			prepared, err := PrepareRules(users, identity.Options{})
			require.NoError(t, err)
			require.Equal(t, legacy.GetAll(), prepared.ReadView().Get())
			require.Equal(t, legacy.GetHash(), prepared.ReadView().GetHash())
			require.Equal(t, len(users), prepared.InputCount())
			require.Equal(t, HashUserList(users), prepared.InputVersion())
		})
	}
}

func TestPreparedRulesRejectsIdentityBeforePublication(t *testing.T) {
	c := NewSafeUserCache()
	c.Set([]define.AllowListUser{{Mail: "old@example.com", UserID: "old"}})
	before := c.ReadView()
	_, err := PrepareRules([]define.AllowListUser{{Mail: "invalid", UserID: "same"}, {Mail: "other-invalid", UserID: "same"}}, identity.Options{})
	require.ErrorIs(t, err, identity.ErrIdentityConflict)
	require.Same(t, before, c.ReadView())
	previous := define.RequireExplicitUserID()
	define.SetRequireExplicitUserID(true)
	t.Cleanup(func() { define.SetRequireExplicitUserID(previous) })
	_, err = PrepareRules([]define.AllowListUser{{Mail: "a@example.com"}}, identity.Options{RequireExplicitUserID: true})
	require.ErrorIs(t, err, identity.ErrMissingUserID)
}

func TestPreparedRulesOwnsScopesAndRetainedViews(t *testing.T) {
	input := []define.AllowListUser{{Phone: "13800138000", Mail: "a@example.com", UserID: "a", Scope: []string{"read"}}}
	prepared, err := PrepareRules(input, identity.Options{})
	require.NoError(t, err)
	input[0].Scope[0] = "write"
	c := NewSafeUserCache()
	c.ApplyPrepared(prepared)
	old := c.ReadView()
	oldVersion := old.GetHash()
	for _, read := range []func() (define.AllowListUser, bool){
		func() (define.AllowListUser, bool) { return old.GetByPhone("13800138000") },
		func() (define.AllowListUser, bool) { return old.GetByMail("a@example.com") },
		func() (define.AllowListUser, bool) { return old.GetByUserID("a") },
	} {
		u, found := read()
		require.True(t, found)
		require.Equal(t, []string{"read"}, u.Scope)
		u.Scope[0] = "changed"
	}
	old.Get()[0].Scope[0] = "changed"
	old.Iterate(func(u define.AllowListUser) bool { u.Scope[0] = "changed"; return true }) //nolint:gocritic // hugeParam: public callback signature
	c.Set([]define.AllowListUser{{Mail: "b@example.com", UserID: "b"}})
	require.Equal(t, "b", c.Get()[0].UserID)
	require.Equal(t, "a", old.Get()[0].UserID)
	require.Equal(t, []string{"read"}, old.Get()[0].Scope)
	require.Equal(t, oldVersion, old.GetHash())
}

func TestReadViewDoesNotMixGenerations(t *testing.T) {
	c := NewSafeUserCache()
	c.Set([]define.AllowListUser{{Mail: "a@example.com", UserID: "a"}})
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			c.Set([]define.AllowListUser{{Mail: "b@example.com", UserID: "b"}})
			c.Set([]define.AllowListUser{{Mail: "a@example.com", UserID: "a"}})
		}
	}()
	for i := 0; i < 100; i++ {
		view := c.ReadView()
		users := view.Get()
		_, found := view.GetByUserID(users[0].UserID)
		require.True(t, found)
		require.Equal(t, HashUserList(users), view.GetHash())
	}
	wg.Wait()
}
