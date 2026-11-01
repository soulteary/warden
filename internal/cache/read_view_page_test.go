package cache

import (
	"math"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"github.com/stretchr/testify/require"
)

func TestReadViewPagesPreserveEffectiveOrderAndOwnership(t *testing.T) {
	users := []define.AllowListUser{
		{Mail: "z@example.com", UserID: "z", Scope: []string{"first"}},
		{Mail: "a@example.com", UserID: "a", Scope: []string{"read", "write"}},
		{Phone: "invalid", Mail: "bad@example.com", UserID: "invalid"},
		{Mail: "m@example.com", UserID: "m", Scope: []string{}},
		{Mail: "z@example.com", UserID: "replacement", Scope: []string{"last"}},
	}
	c := NewSafeUserCache()
	c.Set(users)
	view := c.ReadView()
	want := view.Get()
	require.Len(t, want, 3)
	for _, size := range []int{1, 2, 3, 10, math.MaxInt} {
		var combined []define.AllowListUser
		for page := 1; page <= len(want); page++ {
			rows, total, pages := view.Page(page, size)
			require.Equal(t, len(want), total)
			expectedPages := len(want) / size
			if len(want)%size != 0 {
				expectedPages++
			}
			require.Equal(t, expectedPages, pages)
			combined = append(combined, rows...)
		}
		require.Equal(t, want, combined)
	}
	rows, _, _ := view.Page(1, 2)
	rows[0].UserID = "changed"
	rows[0].Scope[0] = "changed"
	require.Equal(t, want, view.Get())
	again, _, _ := view.Page(1, 2)
	require.Equal(t, want[:2], again)
	c.Set([]define.AllowListUser{{Mail: "new@example.com", Scope: []string{"new"}}})
	old, _, _ := view.Page(1, 10)
	require.Equal(t, want, old)
}

func TestReadViewPageBoundsAndPreparedRules(t *testing.T) {
	empty := NewSafeUserCache().ReadView()
	rows, total, pages := empty.Page(1, 20)
	require.NotNil(t, rows)
	require.Empty(t, rows)
	require.Zero(t, total)
	require.Zero(t, pages)
	prepared, err := PrepareRules([]define.AllowListUser{{Mail: "b@example.com", UserID: "b"}, {Mail: "a@example.com", UserID: "a"}}, identity.Options{})
	require.NoError(t, err)
	view := prepared.ReadView()
	for _, tc := range []struct{ page, size int }{{0, 1}, {-1, 1}, {3, 1}, {math.MaxInt, 1}, {1, 0}, {1, -1}} {
		rows, total, pages = view.Page(tc.page, tc.size)
		require.NotNil(t, rows)
		require.Empty(t, rows)
		require.Equal(t, 2, total)
		if tc.size <= 0 {
			require.Zero(t, pages)
		} else {
			require.Equal(t, 2, pages)
		}
	}
	rows, total, pages = view.Page(1, math.MaxInt)
	require.Equal(t, view.Get(), rows)
	require.Equal(t, 2, total)
	require.Equal(t, 1, pages)
}

func TestReadViewPagesStayTogetherDuringReplacement(t *testing.T) {
	c := NewSafeUserCache()
	makeUsers := func(marker string) []define.AllowListUser {
		return []define.AllowListUser{
			{Mail: "a@example.com", UserID: "a", Name: marker, Scope: []string{marker}},
			{Mail: "b@example.com", UserID: "b", Name: marker, Scope: []string{marker}},
			{Mail: "c@example.com", UserID: "c", Name: marker, Scope: []string{marker}},
		}
	}
	c.Set(makeUsers("first"))
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			c.Set(makeUsers("replacement"))
		}
	}()
	defer func() { <-done }()
	for i := 0; i < 100; i++ {
		view := c.ReadView()
		first, total, pages := view.Page(1, 2)
		last, lastTotal, lastPages := view.Page(2, 2)
		require.Equal(t, 3, total)
		require.Equal(t, 2, pages)
		require.Equal(t, total, lastTotal)
		require.Equal(t, pages, lastPages)
		require.Len(t, first, 2)
		require.Len(t, last, 1)
		require.Equal(t, first[0].Name, last[0].Name)
		require.Equal(t, first[0].Scope, last[0].Scope)
	}
}
