package cache

import (
	"fmt"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"github.com/stretchr/testify/require"
	"testing"
)

// Frozen preparation boundary from main f2913a5. The identity package separately
// compares validation and ordering against its frozen implementation.
func legacyPrepareCandidate(users []define.AllowListUser, opts identity.Options) (*PreparedRules, error) {
	res, err := identity.ValidateAndIndexUsers(cloneUsers(users), opts)
	if err != nil {
		return nil, err
	}
	return &PreparedRules{view: buildReadView(res.Users, false), input: res.Users, inputVersion: legacyHashUserList(users)}, nil
}

func candidateContractInputs() [][]define.AllowListUser {
	return [][]define.AllowListUser{nil, {},
		{{Mail: "b@example.com", UserID: "b", Scope: []string{"write", "read"}}, {Mail: "a@example.com", UserID: "a", Scope: nil}, {Mail: "c@example.com", UserID: "c", Scope: []string{}}},
		{{Mail: "derived@example.com", Scope: []string{"read"}}},
		{{Mail: "valid@example.com", UserID: "valid"}, {Mail: "invalid", UserID: "invalid"}},
		{{Mail: "invalid", UserID: "rejected"}, {UserID: "no-identifier"}},
		{{Phone: "13800000001", Mail: "first@example.com", UserID: "same"}, {Phone: "13800000001", Mail: "second@example.com", UserID: "same"}},
		{{Mail: " SAME@example.com ", UserID: "a"}, {Mail: "same@example.com", UserID: "b"}},
		{{UserID: "same", Phone: "same", Mail: "X@example.com", Name: "first"}, {UserID: "same", Phone: "same", Mail: "x@example.com", Name: "second"}},
	}
}

func TestCandidatePreparationLegacyDifferential(t *testing.T) {
	oldStrategy, oldExplicit := define.GetUserIDStrategy(), define.RequireExplicitUserID()
	t.Cleanup(func() { define.SetUserIDStrategy(oldStrategy); define.SetRequireExplicitUserID(oldExplicit) })
	for _, strategy := range []define.UserIDStrategy{define.UserIDStrategyLegacy, define.UserIDStrategySHA256_128} {
		define.SetUserIDStrategy(strategy)
		for _, explicit := range []bool{false, true} {
			define.SetRequireExplicitUserID(explicit)
			for _, report := range []bool{false, true} {
				for i, users := range candidateContractInputs() {
					t.Run(fmt.Sprintf("%s/explicit=%t/report=%t/case=%d", strategy, explicit, report, i), func(t *testing.T) {
						before := copyHashInput(users)
						opts := identity.Options{RequireExplicitUserID: explicit, ReportOnly: report}
						want, wantErr := legacyPrepareCandidate(users, opts)
						got, gotErr := PrepareRules(users, opts)
						require.Equal(t, wantErr, gotErr)
						require.Equal(t, before, users)
						if wantErr != nil {
							require.Nil(t, got)
							return
						}
						require.Equal(t, want.input, got.input)
						require.Equal(t, want.InputCount(), got.InputCount())
						require.Equal(t, want.InputVersion(), got.InputVersion())
						require.Equal(t, want.view.Get(), got.view.Get())
						require.Equal(t, want.view.GetHash(), got.view.GetHash())
					})
				}
			}
		}
	}
}

func TestCandidateOwnershipAcrossRepeatedPublication(t *testing.T) {
	input := []define.AllowListUser{{Mail: "member@example.com", UserID: "member", Scope: []string{"read", "write"}}}
	prepared, err := PrepareRules(input, identity.Options{})
	require.NoError(t, err)
	version, effective := prepared.InputVersion(), prepared.view.GetHash()
	// Keep the existing separate Scope buffers in the input candidate and indexes.
	require.NotSame(t, &prepared.input[0].Scope[0], &prepared.view.rows[0].Scope[0])
	input[0].Scope[0] = "caller changed"
	input[0].Mail = "changed@example.com"
	first, second := NewSafeUserCache(), NewSafeUserCache()
	first.ApplyPrepared(prepared)
	second.ApplyPrepared(prepared)
	old := first.ReadView()
	first.Get()[0].Scope[0] = "reader changed"
	second.Get()[0].UserID = "reader changed"
	first.Set([]define.AllowListUser{{Mail: "replacement@example.com", UserID: "replacement"}})
	require.Equal(t, "member", old.Get()[0].UserID)
	require.Equal(t, []string{"read", "write"}, second.Get()[0].Scope)
	first.ApplyPrepared(prepared)
	require.Equal(t, old.Get(), first.Get())
	require.Equal(t, old.Get(), second.Get())
	require.Equal(t, version, prepared.InputVersion())
	require.Equal(t, effective, first.GetHash())
	require.Equal(t, effective, second.GetHash())
}

func TestCandidateFailureKeepsPublishedViews(t *testing.T) {
	cache := NewSafeUserCache()
	cache.Set([]define.AllowListUser{{Mail: "old@example.com", UserID: "old", Scope: []string{"read"}}})
	before := cache.ReadView()
	users := []define.AllowListUser{{Mail: "first@example.com", UserID: "duplicate"}, {Mail: "second@example.com", UserID: "duplicate"}}
	input := copyHashInput(users)
	require.ErrorIs(t, cache.SetValidated(users, identity.Options{}), identity.ErrIdentityConflict)
	require.Same(t, before, cache.ReadView())
	require.Equal(t, input, users)
	require.Equal(t, []string{"read"}, before.Get()[0].Scope)
}
