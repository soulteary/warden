package loader

import (
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMergeOrderingWithoutDistinctUserIDs(t *testing.T) {
	users := mergeByMode([]define.AllowListUser{{Phone: "2", Mail: "z@example.com"}, {Phone: "1", Mail: "b@example.com"}}, []define.AllowListUser{{Mail: "a@example.com"}}, "DEFAULT")
	require.Len(t, users, 3)
	require.Equal(t, "1", users[0].Phone)
	require.Equal(t, "2", users[1].Phone)
	require.Equal(t, "a@example.com", users[2].Mail)
}
