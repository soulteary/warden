package cache

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/define"
)

func TestRenewIfUnchanged(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck // test cleanup
	c := NewRedisUserCache(client)
	users := []define.AllowListUser{{Phone: "13800138000", Mail: "a@example.com", UserID: "a", Status: "active", Scope: []string{"read"}}}

	renewed, err := c.RenewIfUnchanged(users)
	require.NoError(t, err)
	require.False(t, renewed, "nothing is stored yet")

	require.NoError(t, c.Set(users))
	version, err := c.GetVersion()
	require.NoError(t, err)
	mr.FastForward(40 * time.Minute)
	require.Less(t, mr.TTL(REDIS_CACHE_KEY), 30*time.Minute)

	renewed, err = c.RenewIfUnchanged(users)
	require.NoError(t, err)
	require.True(t, renewed)
	require.Equal(t, REDIS_CACHE_TTL, mr.TTL(REDIS_CACHE_KEY), "the TTL starts again")
	require.Zero(t, mr.TTL(REDIS_CACHE_VERSION_KEY), "the version key stays persistent")
	after, err := c.GetVersion()
	require.NoError(t, err)
	require.Equal(t, version, after, "nothing was written")
	stored, err := c.Get()
	require.NoError(t, err)
	require.Equal(t, users, stored)

	// Different content must be written, not renewed.
	changed := []define.AllowListUser{{Phone: "13800138000", Mail: "a@example.com", UserID: "a", Status: "suspended", Scope: []string{"read"}}}
	renewed, err = c.RenewIfUnchanged(changed)
	require.NoError(t, err)
	require.False(t, renewed)
	renewed, err = c.RenewIfUnchanged([]define.AllowListUser{})
	require.NoError(t, err)
	require.False(t, renewed)

	// A value written by someone else is not renewed either.
	require.NoError(t, mr.Set(REDIS_CACHE_KEY, `[{"phone":"13900139000","mail":"","user_id":"x","status":"active","scope":[],"role":""}]`))
	renewed, err = c.RenewIfUnchanged(users)
	require.NoError(t, err)
	require.False(t, renewed)

	// An empty rule set that is stored is renewed like any other.
	require.NoError(t, c.Set([]define.AllowListUser{}))
	renewed, err = c.RenewIfUnchanged([]define.AllowListUser{})
	require.NoError(t, err)
	require.True(t, renewed)

	mr.Close()
	_, err = c.RenewIfUnchanged(users)
	require.Error(t, err)

	_, err = (&RedisUserCache{}).RenewIfUnchanged(users)
	require.Error(t, err)
}
