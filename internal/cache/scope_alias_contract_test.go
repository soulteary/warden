package cache

import (
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPreparedScopesDetachOverlappingAndEmptyBacking(t *testing.T) {
	backing := []string{"read", "write", "spare"}
	empty := []string{"unused"}
	input := []define.AllowListUser{
		{UserID: "b", Mail: "b@example.com", Scope: backing[:2]},
		{UserID: "a", Mail: "a@example.com", Scope: backing[1:2]},
		{UserID: "c", Mail: "c@example.com", Scope: empty[:0]},
		{UserID: "d", Mail: "d@example.com", Scope: nil},
	}
	prepared, err := PrepareRules(input, identity.Options{})
	require.NoError(t, err)
	expected := prepared.view.Get()
	inputVersion, effectiveVersion := prepared.InputVersion(), prepared.view.GetHash()
	require.Equal(t, legacyHashUserList(input), inputVersion)
	require.Equal(t, legacyHashUserList(expected), effectiveVersion)
	for i := range prepared.input {
		if len(prepared.input[i].Scope) > 0 {
			require.NotSame(t, &prepared.input[i].Scope[0], &prepared.view.rows[i].Scope[0])
		}
	}
	backing[0] = "caller"
	backing[1] = "caller"
	empty[0] = "caller"
	prepared.input[0].Scope[0] = "candidate"
	require.Equal(t, expected, prepared.view.Get())
	require.NotNil(t, prepared.view.Get()[2].Scope)
	require.Empty(t, prepared.view.Get()[2].Scope)
	require.NotNil(t, prepared.view.Get()[3].Scope)
	// Distinct records no longer share the caller's overlapping Scope allocation.
	prepared.input[1].Scope[1] = "other candidate"
	require.Equal(t, []string{"candidate"}, prepared.input[0].Scope)
	c := NewSafeUserCache()
	c.ApplyPrepared(prepared)
	old := c.ReadView()
	page, _, _ := old.Page(1, 4)
	page[0].Scope[0] = "reader"
	got, ok := old.GetByMail("b@example.com")
	require.True(t, ok)
	got.Scope[0] = "lookup"
	c.ApplyPrepared(prepared)
	require.Equal(t, expected, c.Get())
	c.Set([]define.AllowListUser{{UserID: "replacement", Mail: "replacement@example.com"}})
	require.Equal(t, expected, old.Get())
	require.Equal(t, inputVersion, prepared.InputVersion())
	require.Equal(t, effectiveVersion, old.GetHash())
}

func TestRejectedOverlappingScopeInputDoesNotChangePublishedView(t *testing.T) {
	c := NewSafeUserCache()
	c.Set([]define.AllowListUser{{UserID: "old", Mail: "old@example.com", Scope: []string{"read"}}})
	old := c.ReadView()
	before := old.Get()
	version := old.GetHash()
	backing := []string{"read", "write"}
	input := []define.AllowListUser{{UserID: "same", Mail: "a@example.com", Scope: backing[:1]}, {UserID: "same", Mail: "b@example.com", Scope: backing[1:]}}
	require.ErrorIs(t, c.SetValidated(input, identity.Options{}), identity.ErrIdentityConflict)
	require.Equal(t, "", input[0].Status)
	require.Equal(t, []string{"read", "write"}, backing)
	require.Same(t, old, c.ReadView())
	backing[0] = "caller"
	require.Equal(t, before, c.Get())
	require.Equal(t, version, c.GetHash())
}
