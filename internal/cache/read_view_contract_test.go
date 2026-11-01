package cache

import (
	"errors"
	"fmt"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/identity"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

// Frozen from main 496eeec. Preserve result ownership and callback behavior.
func legacyReadViewGet(v *ReadView) []define.AllowListUser {
	rows := v.cache.GetAll()
	for i := range rows {
		rows[i] = cloneUser(rows[i])
	}
	return rows
}
func legacyReadViewIterate(v *ReadView, fn func(define.AllowListUser) bool) {
	v.cache.Iterate(func(u define.AllowListUser) bool { return fn(cloneUser(u)) })
} //nolint:gocritic // hugeParam: preserve the existing callback signature.

func readContractFixtures() [][]define.AllowListUser {
	return [][]define.AllowListUser{nil, {},
		{{Mail: "one@example.com", UserID: "one", Scope: nil}},
		{{Mail: "b@example.com", UserID: "b", Scope: []string{"write", "read"}}, {Mail: "a@example.com", UserID: "a", Scope: []string{}}},
		{{Mail: "same@example.com", UserID: "first", Scope: []string{"read"}}, {Mail: "same@example.com", UserID: "last", Scope: []string{"write"}}, {Mail: "invalid", UserID: "bad"}},
	}
}

func TestReadViewGetLegacyContract(t *testing.T) {
	initial := NewSafeUserCache()
	require.NotNil(t, initial.Get())
	require.Empty(t, initial.Get())
	for i, users := range readContractFixtures() {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c := NewSafeUserCache()
			c.Set(users)
			v := c.ReadView()
			version := v.GetHash()
			want := legacyReadViewGet(v)
			got := v.Get()
			require.NotNil(t, got)
			require.Equal(t, want, got)
			if len(got) > 0 {
				got[0].Name = "changed"
				if len(got[0].Scope) > 0 {
					got[0].Scope[0] = "changed"
				}
			}
			require.Equal(t, want, v.Get())
			require.Equal(t, want, c.GetReadOnly())
			require.Equal(t, version, v.GetHash())
		})
	}
	prepared, err := PrepareRules([]define.AllowListUser{{Mail: "b@example.com", UserID: "b"}, {Mail: "a@example.com", UserID: "a", Scope: []string{"read"}}}, identity.Options{})
	require.NoError(t, err)
	require.Equal(t, legacyReadViewGet(prepared.ReadView()), prepared.ReadView().Get())
}

func readContractTrace(v *ReadView, c *SafeUserCache, stop int, legacy bool) []define.AllowListUser {
	var trace []define.AllowListUser
	fn := func(u define.AllowListUser) bool {
		trace = append(trace, cloneUser(u))
		if len(u.Scope) > 0 {
			u.Scope[0] = "changed"
		}
		_ = v.Get()
		c.Set([]define.AllowListUser{{Mail: "replacement@example.com", UserID: "replacement"}})
		return len(trace) < stop
	} //nolint:gocritic // hugeParam: public callback signature.
	if legacy {
		legacyReadViewIterate(v, fn)
	} else {
		v.Iterate(fn)
	}
	return trace
}

func TestReadViewIterateLegacyContract(t *testing.T) {
	for _, users := range readContractFixtures() {
		c := NewSafeUserCache()
		c.Set(users)
		v := c.ReadView()
		before := v.Get()
		version := v.GetHash()
		for _, stop := range []int{0, 1, 2, 100} {
			require.Equal(t, readContractTrace(v, c, stop, true), readContractTrace(v, c, stop, false))
			require.Equal(t, before, v.Get())
			require.Equal(t, version, v.GetHash())
		}
	}
}

func readContractPanic(fn func()) (value any) { defer func() { value = recover() }(); fn(); return nil }
func TestReadViewIterationPanicAndNilCallback(t *testing.T) {
	empty := NewSafeUserCache().ReadView()
	require.NotPanics(t, func() { empty.Iterate(nil) })
	require.NotPanics(t, func() { legacyReadViewIterate(empty, nil) })
	c := NewSafeUserCache()
	c.Set([]define.AllowListUser{{Mail: "member@example.com", UserID: "member", Scope: []string{"read"}}})
	v := c.ReadView()
	require.IsType(t, readContractPanic(func() { legacyReadViewIterate(v, nil) }), readContractPanic(func() { v.Iterate(nil) }))
	sentinel := errors.New("callback failed")
	called := 0
	fn := func(define.AllowListUser) bool { called++; panic(sentinel) }
	require.PanicsWithValue(t, sentinel, func() { v.Iterate(fn) })
	require.Equal(t, 1, called)
	require.Equal(t, []string{"read"}, v.Get()[0].Scope)
}

func TestReadViewIterationPinsGenerationDuringReplacement(t *testing.T) {
	c := NewSafeUserCache()
	c.Set(candidateBenchmarkUsers(100, 0))
	pinned := c.ReadView()
	want := pinned.Get()
	var writers sync.WaitGroup
	writers.Add(1)
	go func() {
		defer writers.Done()
		for i := 0; i < 100; i++ {
			c.Set(candidateBenchmarkUsers(100, i+1))
		}
	}()
	defer writers.Wait()
	for i := 0; i < 50; i++ {
		var got []define.AllowListUser
		pinned.Iterate(func(u define.AllowListUser) bool { got = append(got, u); return true })
		require.Equal(t, want, got)
	} //nolint:gocritic // hugeParam: public callback signature.
}

var readViewBenchmarkRows []define.AllowListUser
var readViewBenchmarkUser define.AllowListUser

func BenchmarkReadViewReads(b *testing.B) {
	for _, count := range []int{0, 1, 1000, 10000} {
		c := NewSafeUserCache()
		c.Set(candidateBenchmarkUsers(count, 0))
		view := c.ReadView()
		if view.Len() != count {
			b.Fatal("fixture lost rows")
		}
		b.Run(fmt.Sprintf("rows=%d", count), func(b *testing.B) {
			b.Run("Get", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					readViewBenchmarkRows = view.Get()
				}
			})
			for _, limit := range []struct {
				name string
				stop int
			}{{"first", 1}, {"twenty", 20}, {"all", count}} {
				b.Run("Iterate/"+limit.name, func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						visited := 0
						view.Iterate(func(u define.AllowListUser) bool { readViewBenchmarkUser = u; visited++; return visited < limit.stop })
					}
				})
			} //nolint:gocritic // hugeParam: public callback signature.
		})
	}
}
