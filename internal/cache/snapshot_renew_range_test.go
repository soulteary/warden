package cache

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/alicebob/miniredis/v2/server"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// The reply Redis 8 gives redis.pcall for a command the calling user's ACL
// does not permit.
const getrangeDenied = "ERR ACL failure in script: User restricted has no permissions to run the 'getrange' command"

// commandLog records the commands miniredis runs against one key, including
// those a script calls, and can refuse GETRANGE the way an ACL does.
type commandLog struct {
	refuse string
	seen   []string
	mu     sync.Mutex
}

func watchCommands(mr *miniredis.Miniredis, key, refuseGetrange string) *commandLog {
	log := &commandLog{refuse: refuseGetrange}
	mr.Server().SetPreHook(func(peer *server.Peer, cmd string, args ...string) bool {
		if len(args) == 0 || args[0] != key {
			return false
		}
		log.mu.Lock()
		defer log.mu.Unlock()
		log.seen = append(log.seen, cmd)
		if cmd == "GETRANGE" && log.refuse != "" {
			peer.WriteError(log.refuse)
			return true
		}
		return false
	})
	return log
}

func (l *commandLog) take() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	seen := strings.Join(l.seen, " ")
	l.seen = nil
	return seen
}

// A renewal of a published envelope reads only its beginning: the script does
// not GET the value, whose size is bounded only by MAX_ENVELOPE_JSON_SIZE.
func TestRenewReadsOnlyTheBeginningOfAPublishedEnvelope(t *testing.T) {
	store, mr := snapshotStoreFixture(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	published, err := lease.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	before, err := mr.Get(store.key)
	require.NoError(t, err)
	log := watchCommands(mr, store.key, "")

	require.NoError(t, lease.Renew(ctx, published.Revision))
	require.Equal(t, "STRLEN GETRANGE PEXPIRE", log.take())
	require.ErrorIs(t, lease.Renew(ctx, strings.Repeat("0", 32)), ErrSnapshotConflict)
	require.Equal(t, "STRLEN GETRANGE", log.take())

	// A value with another beginning is still read in full to be decoded, and
	// a value that states a later schema version is not.
	require.NoError(t, mr.Set(store.key, " "+before))
	require.NoError(t, lease.Renew(ctx, published.Revision))
	require.Equal(t, "STRLEN GETRANGE GET PEXPIRE", log.take())
	require.NoError(t, mr.Set(store.key, `{"schema_version":3,"records":[`))
	require.ErrorIs(t, lease.Renew(ctx, published.Revision), ErrSnapshotConflict)
	require.Equal(t, "STRLEN GETRANGE", log.take())

	// STRLEN is 0 for a missing key and for an empty value; GET tells them
	// apart, and GETRANGE, which answers both with an empty string, is skipped.
	require.NoError(t, mr.Set(store.key, ""))
	require.ErrorIs(t, lease.Renew(ctx, published.Revision), ErrSnapshotInvalid)
	require.Equal(t, "STRLEN GET", log.take())
	mr.Del(store.key)
	require.ErrorIs(t, lease.Renew(ctx, published.Revision), ErrSnapshotMissing)
	require.Equal(t, "STRLEN GET", log.take())
}

// A Redis user whose ACL does not permit GETRANGE gets an error reply from
// redis.pcall. The script then reads the value with GET and renews as it did
// before GETRANGE was used; the refusal does not abort the script.
func TestRenewFallsBackToGetWhenGetrangeIsRefused(t *testing.T) {
	for name, refusal := range map[string]string{
		"acl_denied":      getrangeDenied,
		"unknown_command": "ERR unknown command 'GETRANGE'",
	} {
		t.Run(name, func(t *testing.T) {
			store, mr := snapshotStoreFixture(t)
			ctx := context.Background()
			lease, err := store.Acquire(ctx, time.Minute)
			require.NoError(t, err)
			published, err := lease.Publish(ctx, sourceSnapshot(time.Now()))
			require.NoError(t, err)
			before, err := mr.Get(store.key)
			require.NoError(t, err)
			log := watchCommands(mr, store.key, refusal)

			mr.SetTTL(store.key, time.Second)
			require.NoError(t, lease.Renew(ctx, published.Revision))
			require.Equal(t, "STRLEN GETRANGE GET PEXPIRE", log.take())
			require.Equal(t, REDIS_CACHE_TTL, mr.TTL(store.key))
			require.ErrorIs(t, lease.Renew(ctx, strings.Repeat("0", 32)), ErrSnapshotConflict)
			require.Equal(t, "STRLEN GETRANGE GET", log.take(), "the value is read once, not again for the decode")
			require.NoError(t, mr.Set(store.key, " "+before))
			require.NoError(t, lease.Renew(ctx, published.Revision))
			require.Equal(t, "STRLEN GETRANGE GET PEXPIRE", log.take())
			after, err := mr.Get(store.key)
			require.NoError(t, err)
			require.Equal(t, " "+before, after)
			require.NoError(t, lease.Release(ctx))
			require.ErrorIs(t, lease.Renew(ctx, published.Revision), ErrSnapshotLeaseLost)
			require.Empty(t, log.take(), "the lease is checked before the value is read")
		})
	}
}

// Every stored value is classified the same way whether its beginning was
// read with GETRANGE or, after a refusal, with GET.
func TestRenewClassifiesTheSameAfterGetrangeIsRefused(t *testing.T) {
	store, mr := snapshotStoreFixture(t)
	log := watchCommands(mr, store.key, getrangeDenied)
	renewCases(t, store, store.client)
	require.Contains(t, log.take(), "GETRANGE GET")
}

// restrictedSnapshotStore returns a store for the same dataset whose Redis
// user may run the scripts and exactly the listed commands on the two keys.
func restrictedSnapshotStore(t *testing.T, store *RedisSnapshotStore, admin *redis.Client, commands ...string) (restricted *RedisSnapshotStore, user string) {
	t.Helper()
	ctx := context.Background()
	token, err := newSnapshotToken()
	require.NoError(t, err)
	user = "snapshot-it-" + token
	rule := []any{"ACL", "SETUSER", user, "reset", "on", ">integration-password", "~" + store.key, "~" + store.lock, "+evalsha", "+eval"}
	for _, command := range commands {
		rule = append(rule, "+"+command)
	}
	require.NoError(t, admin.Do(ctx, rule...).Err())
	t.Cleanup(func() { require.NoError(t, admin.Do(context.Background(), "ACL", "DELUSER", user).Err()) })
	client := redis.NewClient(&redis.Options{Addr: admin.Options().Addr, Username: user, Password: "integration-password", MaxRetries: -1, ContextTimeoutEnabled: true})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	restricted, err = NewRedisSnapshotStore(client, store.dataset, store.policy)
	require.NoError(t, err)
	return restricted, user
}

func getrangeDenials(t *testing.T, admin *redis.Client, user string) int64 {
	t.Helper()
	entries, err := admin.ACLLog(context.Background(), 128).Result()
	require.NoError(t, err)
	var count int64
	for _, entry := range entries {
		if entry.Username == user && entry.Object == "getrange" {
			count += entry.Count
		}
	}
	return count
}

// The commands the snapshot scripts called before GETRANGE was added are
// enough: Redis refuses GETRANGE inside the script, the script reads the value
// with GET, and every stored value is classified as it is for a user that may
// call GETRANGE.
func TestRedisSnapshotIntegrationRenewWithoutGetrangePermission(t *testing.T) {
	store, admin := realSnapshotStore(t)
	restricted, user := restrictedSnapshotStore(t, store, admin, "get", "strlen", "set", "del", "pexpire")
	ctx := context.Background()
	lease, err := restricted.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	published, err := lease.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	before, err := admin.Get(ctx, store.key).Result()
	require.NoError(t, err)
	require.NoError(t, admin.PExpire(ctx, store.key, time.Minute).Err())
	require.Zero(t, getrangeDenials(t, admin, user))

	require.NoError(t, lease.Renew(ctx, published.Revision))
	ttl, err := admin.PTTL(ctx, store.key).Result()
	require.NoError(t, err)
	require.Greater(t, ttl, REDIS_CACHE_TTL-time.Minute)
	require.ErrorIs(t, lease.Renew(ctx, strings.Repeat("0", 32)), ErrSnapshotConflict)
	after, err := admin.Get(ctx, store.key).Result()
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, int64(2), getrangeDenials(t, admin, user), "Redis refused GETRANGE in both renewals and the script went on")
	read, err := restricted.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, published, read)
	require.NoError(t, lease.Release(ctx))

	renewCases(t, restricted, admin)
}

// With GETRANGE permitted, the same user's renewals are not refused anything.
func TestRedisSnapshotIntegrationRenewWithGetrangePermission(t *testing.T) {
	store, admin := realSnapshotStore(t)
	restricted, user := restrictedSnapshotStore(t, store, admin, "get", "strlen", "set", "del", "pexpire", "getrange")
	renewCases(t, restricted, admin)
	require.Zero(t, getrangeDenials(t, admin, user))
}
