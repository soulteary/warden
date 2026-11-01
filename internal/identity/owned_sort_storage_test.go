package identity

import (
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOwnedSortReusesPrivateOuter(t *testing.T) {
	storage := make([]define.AllowListUser, 2, 5)
	storage[0] = define.AllowListUser{UserID: "z", Mail: "z@example.com", Scope: []string{"write"}}
	storage[1] = define.AllowListUser{UserID: "a", Mail: "a@example.com", Scope: []string{"read"}}
	firstScope, secondScope := &storage[0].Scope[0], &storage[1].Scope[0]
	base := &storage[0]
	result, err := ValidateAndIndexOwnedUsers(storage, Options{})
	require.NoError(t, err)
	require.Same(t, base, &result.Users[0])
	require.Equal(t, 2, cap(result.Users))
	require.Equal(t, "a", storage[0].UserID)
	require.Equal(t, "z", storage[1].UserID)
	require.Same(t, secondScope, &result.Users[0].Scope[0])
	require.Same(t, firstScope, &result.Users[1].Scope[0])
	require.Equal(t, []string{"read"}, result.Users[0].Scope)
	require.Equal(t, []string{"write"}, result.Users[1].Scope)
}
