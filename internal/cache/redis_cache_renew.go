package cache

import (
	"context"
	"crypto/sha1" // #nosec G505 -- matches redis.sha1hex; a change detector, not a security control
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/soulteary/warden/internal/define"
)

// REDIS_CACHE_SOURCE_KEY records which stored rule set a source time belongs
// to and that time: "<SHA-1 of the stored value, hex>:<Unix milliseconds>".
// The array under REDIS_CACHE_KEY is unchanged, so releases that do not know
// this key read and write the rule set as before. The digest ties the record
// to one value: when such a release replaces the array, the record no longer
// matches and the source time of the stored set counts as unknown.
//
//nolint:revive // ALL_CAPS is the project convention
const REDIS_CACHE_SOURCE_KEY = REDIS_CACHE_KEY + ":source"

// RuleSetOutcome says what Publish did with the shared rule set.
type RuleSetOutcome int

const (
	// RuleSetKept means Redis holds a different rule set whose source load is
	// at least as recent as the retained set offered; nothing was changed.
	RuleSetKept RuleSetOutcome = iota
	// RuleSetRenewed means Redis already held exactly this rule set and only
	// its TTL was extended.
	RuleSetRenewed
	// RuleSetWritten means the rule set was stored.
	RuleSetWritten
)

// publishCheckScript decides, without transferring the rule set, whether a
// publication has to write it.
//
// KEYS[1] data key, KEYS[2] version key, KEYS[3] source key; ARGV[1] SHA-1
// (hex) of the offered value, ARGV[2] TTL in milliseconds, ARGV[3] source time
// of the offered set in Unix milliseconds, ARGV[4] '1' when the offered set is
// one the replica retained after a rejected refresh.
//
// Returns 1 when the stored value is the offered one and its TTL was renewed,
// 0 when a retained set is offered and the stored value is a different set
// recorded with the same or a later source time, and 2 when the caller must
// write: the key is missing, or holds a different value that is older or has
// no matching source record.
var publishCheckScript = redis.NewScript(`
local value = redis.call('GET', KEYS[1])
if value == false then
  return 2
end
local digest = redis.sha1hex(value)
local record = redis.call('GET', KEYS[3])
local recorded = nil
if record and string.sub(record, 1, 41) == digest .. ':' then
  recorded = tonumber(string.sub(record, 42))
end
local superseded = ARGV[4] == '1' and recorded ~= nil and recorded >= tonumber(ARGV[3])
if digest == ARGV[1] then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
  redis.call('PERSIST', KEYS[2])
  if superseded then
    redis.call('PEXPIRE', KEYS[3], ARGV[2])
  else
    redis.call('SET', KEYS[3], digest .. ':' .. ARGV[3], 'PX', ARGV[2])
  end
  return 1
end
if superseded then
  return 0
end
return 2
`)

// Publish makes the shared cache hold users, loaded from the sources at
// sourceTime, unless that would move it backwards.
//
// A set from the current refresh (retained false) is always published: when
// Redis already holds exactly this value only its TTL is renewed and its
// source time recorded, otherwise the value is written. Checking the content
// first avoids sending an unchanged list on every refresh.
//
// A set the replica kept after a rejected refresh (retained true) is not new
// information. It renews the TTL when Redis holds the same value, so the
// stored set does not expire while the sources stay unusable, and it is
// written when the key is missing. A different stored value is replaced only
// when its recorded source time is earlier than sourceTime, or when it has no
// matching record; the latter is a value written by a release that does not
// record source times, and is replaced as that release would replace it.
//
// The caller holds the refresh lock, so no other replica writes between the
// check and the write.
func (c *RedisUserCache) Publish(users []define.AllowListUser, sourceTime time.Time, retained bool) (RuleSetOutcome, error) {
	if c.client == nil {
		return RuleSetKept, errors.New("redis cache unavailable")
	}
	// The same encoding rediscache uses for the stored value, so the digests
	// agree with values written by Set and by other releases.
	data, err := json.Marshal(users)
	if err != nil {
		return RuleSetKept, err
	}
	sum := sha1.Sum(data) // #nosec G401 -- see the import
	digest := hex.EncodeToString(sum[:])
	var at int64
	if !sourceTime.IsZero() {
		at = sourceTime.UnixMilli()
	}
	flag := "0"
	if retained {
		flag = "1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), REDIS_OPERATION_TIMEOUT)
	defer cancel()
	keys := []string{REDIS_CACHE_KEY, REDIS_CACHE_VERSION_KEY, REDIS_CACHE_SOURCE_KEY}
	decision, err := publishCheckScript.Run(ctx, c.client, keys, digest, REDIS_CACHE_TTL.Milliseconds(), at, flag).Int()
	switch {
	case err != nil && retained:
		// Without the comparison a retained set could replace a newer one.
		return RuleSetKept, err
	case err == nil && decision == 1:
		return RuleSetRenewed, nil
	case err == nil && decision == 0:
		return RuleSetKept, nil
	}
	// One transaction, as rediscache.Set uses for the value and its version,
	// so the source record never describes a value other than the stored one.
	pipe := c.client.TxPipeline()
	pipe.Set(ctx, REDIS_CACHE_KEY, data, REDIS_CACHE_TTL)
	pipe.Incr(ctx, REDIS_CACHE_VERSION_KEY)
	pipe.Persist(ctx, REDIS_CACHE_VERSION_KEY)
	pipe.Set(ctx, REDIS_CACHE_SOURCE_KEY, digest+":"+strconv.FormatInt(at, 10), REDIS_CACHE_TTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return RuleSetKept, err
	}
	return RuleSetWritten, nil
}
