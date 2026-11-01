package identity

import (
	"fmt"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
	"runtime"
	"testing"
)

func TestOwnedLargeCapacityRetainsOriginalBacking(t *testing.T) {
	storage := make([]define.AllowListUser, 2, 65536)
	storage[0] = define.AllowListUser{UserID: "z", Mail: "z@example.com", Scope: []string{"write"}}
	storage[1] = define.AllowListUser{UserID: "a", Mail: "a@example.com", Scope: []string{"read"}}
	base := &storage[0]
	result, err := ValidateAndIndexOwnedUsers(storage, Options{})
	require.NoError(t, err)
	require.Same(t, base, &result.Users[0], "capacity clipping must not be mistaken for a detached allocation")
	require.Equal(t, 2, cap(result.Users))
	require.Equal(t, "a", storage[0].UserID)
	require.Equal(t, "z", storage[1].UserID)
	storage[0].Name = "same backing"
	require.Equal(t, "same backing", result.Users[0].Name)
}

func TestIdentityScopeOwnershipBoundary(t *testing.T) {
	backing := []string{"read", "write", "spare"}
	rows := []define.AllowListUser{{UserID: "b", Mail: "b@example.com", Scope: backing[:2]}, {UserID: "a", Mail: "a@example.com", Scope: backing[1:2]}}
	for _, validate := range []func([]define.AllowListUser, Options) (Result, error){ValidateAndIndexUsers, ValidateAndIndexOwnedUsers} {
		input := append([]define.AllowListUser{}, rows...)
		result, err := validate(input, Options{})
		require.NoError(t, err)
		require.Same(t, &backing[1], &result.Users[0].Scope[0], "identity validation retains caller responsibility for nested ownership")
		require.Same(t, &backing[0], &result.Users[1].Scope[0])
		require.Equal(t, []string{"read", "write", "spare"}, backing)
	}
}

// A GC-after-allocation diagnostic, not an assertion about RSS or peak memory.
// The result keeps the entire original allocation alive even when cap(result)=len.
func BenchmarkOwnedBackingRetention(b *testing.B) {
	for _, capacity := range []int{16, 65536} {
		b.Run(fmt.Sprintf("len=16/cap=%d", capacity), func(b *testing.B) {
			var live float64
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				runtime.GC()
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				b.StartTimer()
				rows := make([]define.AllowListUser, 16, capacity)
				for j := range rows {
					rows[j] = define.AllowListUser{UserID: fmt.Sprint(16 - j), Mail: fmt.Sprintf("m%d@example.com", j)}
				}
				result, err := ValidateAndIndexOwnedUsers(rows, Options{})
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				runtime.GC()
				runtime.ReadMemStats(&after)
				live += float64(after.HeapAlloc) - float64(before.HeapAlloc)
				runtime.KeepAlive(result)
				b.StartTimer()
			}
			b.ReportMetric(live/float64(b.N), "live-B/result")
		})
	}
}
