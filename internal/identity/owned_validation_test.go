package identity

import (
	"encoding/json"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOwnedValidationMatchesLegacy(t *testing.T) {
	oldStrategy, oldExplicit := define.GetUserIDStrategy(), define.RequireExplicitUserID()
	t.Cleanup(func() { define.SetUserIDStrategy(oldStrategy); define.SetRequireExplicitUserID(oldExplicit) })
	for _, strategy := range []define.UserIDStrategy{define.UserIDStrategyLegacy, define.UserIDStrategySHA256_128} {
		define.SetUserIDStrategy(strategy)
		for _, explicit := range []bool{false, true} {
			define.SetRequireExplicitUserID(explicit)
			for _, report := range []bool{false, true} {
				for _, users := range sortContractUsers() {
					opts := Options{RequireExplicitUserID: explicit, ReportOnly: report}
					before, err := json.Marshal(users)
					require.NoError(t, err)
					owned := make([]define.AllowListUser, len(users))
					copy(owned, users)
					want, wantErr := legacyValidateAndIndexUsers(users, opts)
					got, gotErr := ValidateAndIndexOwnedUsers(owned, opts)
					require.Equal(t, wantErr, gotErr)
					require.Equal(t, want, got)
					after, err := json.Marshal(users)
					require.NoError(t, err)
					require.Equal(t, before, after, "owned outer records must not change caller records or Scope contents")
				}
			}
		}
	}
}

func TestOwnedValidationNormalizesPrivateOuterOnError(t *testing.T) {
	users := []define.AllowListUser{{UserID: "duplicate", Mail: "first@example.com", Scope: []string{"read"}}, {UserID: "duplicate", Mail: "second@example.com"}}
	result, err := ValidateAndIndexOwnedUsers(users, Options{})
	require.ErrorIs(t, err, ErrIdentityConflict)
	require.Equal(t, Result{}, result)
	require.Equal(t, "inactive", users[0].Status)
	require.Equal(t, "inactive", users[1].Status)
	require.Equal(t, []string{"read"}, users[0].Scope)
	require.NotNil(t, users[1].Scope)
}
