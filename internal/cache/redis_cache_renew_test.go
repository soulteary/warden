package cache

import (
	"crypto/sha1" // #nosec G505 -- matches the digest under test
	"encoding/hex"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/define"
)

func sourceRecord(t *testing.T, users []define.AllowListUser, at time.Time) string {
	t.Helper()
	data, err := json.Marshal(users)
	require.NoError(t, err)
	sum := sha1.Sum(data) // #nosec G401 -- see the import
	ms := int64(0)
	if !at.IsZero() {
		ms = at.UnixMilli()
	}
	return hex.EncodeToString(sum[:]) + ":" + strconv.FormatInt(ms, 10)
}

func publishFixture(t *testing.T) (*RedisUserCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck // test cleanup
	return NewRedisUserCache(client), mr
}

func storedRecord(t *testing.T, mr *miniredis.Miniredis) string {
	t.Helper()
	record, err := mr.Get(REDIS_CACHE_SOURCE_KEY)
	require.NoError(t, err)
	return record
}

// A set from the current refresh renews the TTL of an identical stored value
// and writes anything else.
func TestPublishCurrentRuleSet(t *testing.T) {
	c, mr := publishFixture(t)
	users := []define.AllowListUser{{Phone: "13800138000", Mail: "a@example.com", UserID: "a", Status: "active", Scope: []string{"read"}}}
	first := time.Now().Add(-time.Hour).UTC()

	outcome, err := c.Publish(users, first, false)
	require.NoError(t, err)
	require.Equal(t, RuleSetWritten, outcome, "nothing is stored yet")
	require.Equal(t, sourceRecord(t, users, first), storedRecord(t, mr))
	require.Equal(t, REDIS_CACHE_TTL, mr.TTL(REDIS_CACHE_SOURCE_KEY))
	version, err := c.GetVersion()
	require.NoError(t, err)
	mr.FastForward(40 * time.Minute)
	require.Less(t, mr.TTL(REDIS_CACHE_KEY), 30*time.Minute)

	second := first.Add(40 * time.Minute)
	outcome, err = c.Publish(users, second, false)
	require.NoError(t, err)
	require.Equal(t, RuleSetRenewed, outcome)
	require.Equal(t, REDIS_CACHE_TTL, mr.TTL(REDIS_CACHE_KEY), "the TTL starts again")
	require.Equal(t, REDIS_CACHE_TTL, mr.TTL(REDIS_CACHE_SOURCE_KEY))
	require.Zero(t, mr.TTL(REDIS_CACHE_VERSION_KEY), "the version key stays persistent")
	require.Equal(t, sourceRecord(t, users, second), storedRecord(t, mr), "the later load of the same content is recorded")
	after, err := c.GetVersion()
	require.NoError(t, err)
	require.Equal(t, version, after, "the list was not written")
	stored, err := c.Get()
	require.NoError(t, err)
	require.Equal(t, users, stored)

	// Different content is written, whatever the record says, including an
	// empty set and a source time earlier than the recorded one.
	changed := []define.AllowListUser{{Phone: "13800138000", Mail: "a@example.com", UserID: "a", Status: "suspended", Scope: []string{"read"}}}
	outcome, err = c.Publish(changed, first, false)
	require.NoError(t, err)
	require.Equal(t, RuleSetWritten, outcome)
	require.Equal(t, sourceRecord(t, changed, first), storedRecord(t, mr))
	outcome, err = c.Publish([]define.AllowListUser{}, second, false)
	require.NoError(t, err)
	require.Equal(t, RuleSetWritten, outcome)
	outcome, err = c.Publish([]define.AllowListUser{}, second, false)
	require.NoError(t, err)
	require.Equal(t, RuleSetRenewed, outcome, "a stored empty rule set is renewed like any other")
	after, err = c.GetVersion()
	require.NoError(t, err)
	require.Equal(t, version+2, after)

	// A value written by someone else is replaced.
	require.NoError(t, mr.Set(REDIS_CACHE_KEY, `[{"phone":"13900139000","mail":"","user_id":"x","status":"active","scope":[],"role":""}]`))
	outcome, err = c.Publish(users, second, false)
	require.NoError(t, err)
	require.Equal(t, RuleSetWritten, outcome)

	require.NoError(t, c.Clear())
	require.False(t, mr.Exists(REDIS_CACHE_KEY))
	require.False(t, mr.Exists(REDIS_CACHE_SOURCE_KEY), "the record goes with the value")
}

// A retained set is compared with what is stored before it is written.
func TestPublishRetainedRuleSet(t *testing.T) {
	older := []define.AllowListUser{{Mail: "kept@example.com"}, {Mail: "removed@example.com"}}
	newer := []define.AllowListUser{{Mail: "kept@example.com"}}
	base := time.Now().Add(-time.Hour).UTC()
	stored := func(t *testing.T, c *RedisUserCache) []define.AllowListUser {
		t.Helper()
		users, err := c.Get()
		require.NoError(t, err)
		return users
	}

	t.Run("missing key is written", func(t *testing.T) {
		c, mr := publishFixture(t)
		outcome, err := c.Publish(older, base, true)
		require.NoError(t, err)
		require.Equal(t, RuleSetWritten, outcome)
		require.Equal(t, older, stored(t, c))
		require.Equal(t, sourceRecord(t, older, base), storedRecord(t, mr))
	})
	t.Run("same value is renewed and its record kept", func(t *testing.T) {
		c, mr := publishFixture(t)
		_, err := c.Publish(newer, base.Add(time.Minute), false)
		require.NoError(t, err)
		mr.FastForward(50 * time.Minute)
		outcome, err := c.Publish(newer, base, true)
		require.NoError(t, err)
		require.Equal(t, RuleSetRenewed, outcome)
		require.Equal(t, REDIS_CACHE_TTL, mr.TTL(REDIS_CACHE_KEY))
		require.Equal(t, REDIS_CACHE_TTL, mr.TTL(REDIS_CACHE_SOURCE_KEY))
		require.Equal(t, sourceRecord(t, newer, base.Add(time.Minute)), storedRecord(t, mr), "an earlier load of the same content does not move the record back")
		outcome, err = c.Publish(newer, base.Add(2*time.Minute), true)
		require.NoError(t, err)
		require.Equal(t, RuleSetRenewed, outcome)
		require.Equal(t, sourceRecord(t, newer, base.Add(2*time.Minute)), storedRecord(t, mr), "a later load of the same content is recorded")
	})
	t.Run("different value with the same or a later source time is kept", func(t *testing.T) {
		for _, storedAt := range []time.Time{base, base.Add(time.Minute)} {
			c, mr := publishFixture(t)
			_, err := c.Publish(newer, storedAt, false)
			require.NoError(t, err)
			mr.FastForward(50 * time.Minute)
			version, err := c.GetVersion()
			require.NoError(t, err)
			outcome, err := c.Publish(older, base, true)
			require.NoError(t, err)
			require.Equal(t, RuleSetKept, outcome)
			require.Equal(t, newer, stored(t, c))
			require.Equal(t, sourceRecord(t, newer, storedAt), storedRecord(t, mr))
			require.Equal(t, 10*time.Minute, mr.TTL(REDIS_CACHE_KEY), "a set this replica does not hold is not renewed")
			after, err := c.GetVersion()
			require.NoError(t, err)
			require.Equal(t, version, after)
		}
	})
	t.Run("different value with an earlier source time is replaced", func(t *testing.T) {
		c, mr := publishFixture(t)
		_, err := c.Publish(older, base, false)
		require.NoError(t, err)
		outcome, err := c.Publish(newer, base.Add(time.Minute), true)
		require.NoError(t, err)
		require.Equal(t, RuleSetWritten, outcome)
		require.Equal(t, newer, stored(t, c))
		require.Equal(t, sourceRecord(t, newer, base.Add(time.Minute)), storedRecord(t, mr))
	})
	// A release that does not record source times leaves no record, or one
	// that describes the value it replaced. Either way the stored set has no
	// known source time and the retained set is written, as that release does.
	t.Run("value without a record is replaced", func(t *testing.T) {
		c, mr := publishFixture(t)
		require.NoError(t, c.Set(newer))
		outcome, err := c.Publish(older, base, true)
		require.NoError(t, err)
		require.Equal(t, RuleSetWritten, outcome)
		require.Equal(t, sourceRecord(t, older, base), storedRecord(t, mr))
	})
	t.Run("value with a record for another value is replaced", func(t *testing.T) {
		for _, record := range []string{sourceRecord(t, older, base.Add(time.Hour)), "not a record", ""} {
			c, mr := publishFixture(t)
			_, err := c.Publish(older, base.Add(time.Hour), false)
			require.NoError(t, err)
			require.NoError(t, c.Set(newer))
			require.NoError(t, mr.Set(REDIS_CACHE_SOURCE_KEY, record))
			outcome, err := c.Publish(older, base, true)
			require.NoError(t, err)
			require.Equal(t, RuleSetWritten, outcome)
			require.Equal(t, older, stored(t, c))
		}
	})
	t.Run("record without a readable time is treated as absent", func(t *testing.T) {
		c, mr := publishFixture(t)
		_, err := c.Publish(newer, base.Add(time.Hour), false)
		require.NoError(t, err)
		record := storedRecord(t, mr)
		require.NoError(t, mr.Set(REDIS_CACHE_SOURCE_KEY, record[:41]+"soon"))
		outcome, err := c.Publish(older, base, true)
		require.NoError(t, err)
		require.Equal(t, RuleSetWritten, outcome)
	})
	t.Run("same value without a record gets one", func(t *testing.T) {
		c, mr := publishFixture(t)
		require.NoError(t, c.Set(newer))
		outcome, err := c.Publish(newer, base, true)
		require.NoError(t, err)
		require.Equal(t, RuleSetRenewed, outcome)
		require.Equal(t, sourceRecord(t, newer, base), storedRecord(t, mr))
	})
	t.Run("set without a source time", func(t *testing.T) {
		c, mr := publishFixture(t)
		outcome, err := c.Publish(older, time.Time{}, true)
		require.NoError(t, err)
		require.Equal(t, RuleSetWritten, outcome)
		require.Equal(t, sourceRecord(t, older, time.Time{}), storedRecord(t, mr))
		outcome, err = c.Publish(newer, time.Time{}, true)
		require.NoError(t, err)
		require.Equal(t, RuleSetKept, outcome, "neither set has evidence of being later")
		outcome, err = c.Publish(newer, base, true)
		require.NoError(t, err)
		require.Equal(t, RuleSetWritten, outcome)
	})
}

func TestPublishFailures(t *testing.T) {
	users := []define.AllowListUser{{Mail: "a@example.com"}}
	_, err := (&RedisUserCache{}).Publish(users, time.Now(), false)
	require.Error(t, err)

	// A retained set is not written when the comparison cannot be made.
	c, mr := publishFixture(t)
	_, err = c.Publish(users, time.Now().Add(time.Minute), false)
	require.NoError(t, err)
	before, err := mr.Get(REDIS_CACHE_KEY)
	require.NoError(t, err)
	mr.SetError("LOADING Redis is loading the dataset in memory")
	_, err = c.Publish([]define.AllowListUser{}, time.Now(), true)
	require.Error(t, err)
	_, err = c.Publish([]define.AllowListUser{}, time.Now(), false)
	require.Error(t, err, "the write of a current set fails as well")
	require.Error(t, c.Clear())
	mr.SetError("")
	after, err := mr.Get(REDIS_CACHE_KEY)
	require.NoError(t, err)
	require.Equal(t, before, after)
}
