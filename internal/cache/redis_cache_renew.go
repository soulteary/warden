package cache

import (
	"context"
	"crypto/sha1" // #nosec G505 -- matches redis.sha1hex; a change detector, not a security control
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/redis/go-redis/v9"

	"github.com/soulteary/warden/internal/define"
)

// renewScript extends the TTL of the cached rule set when the stored value is
// exactly the expected one. It compares a digest computed inside Redis, so the
// value itself never crosses the network, and it runs atomically.
//
// KEYS[1] data key, KEYS[2] version key; ARGV[1] expected SHA-1 (hex),
// ARGV[2] TTL in milliseconds. Returns 1 when the TTL was renewed, 0 when the
// key is missing or holds something else.
var renewScript = redis.NewScript(`
local value = redis.call('GET', KEYS[1])
if value == false then
  return 0
end
if redis.sha1hex(value) ~= ARGV[1] then
  return 0
end
redis.call('PEXPIRE', KEYS[1], ARGV[2])
redis.call('PERSIST', KEYS[2])
return 1
`)

// RenewIfUnchanged extends the TTL of the shared cache when it already holds
// exactly users, and reports whether it did. When it returns false the caller
// must write the rule set with Set: the key is missing, or its content differs
// from users (another writer, a different data set, a tampered value).
//
// A refresh that finds the data unchanged used to rewrite the complete list
// only to renew its TTL. Checking the content first keeps the property that
// every refresh leaves Redis holding what this replica serves, without the
// write.
func (c *RedisUserCache) RenewIfUnchanged(users []define.AllowListUser) (bool, error) {
	if c.client == nil {
		return false, errors.New("redis cache unavailable")
	}
	// The same encoding Set uses for the stored value. If the two ever
	// diverge the digests differ and the caller falls back to Set.
	data, err := json.Marshal(users)
	if err != nil {
		return false, err
	}
	sum := sha1.Sum(data) // #nosec G401 -- see the import
	ctx, cancel := context.WithTimeout(context.Background(), REDIS_OPERATION_TIMEOUT)
	defer cancel()
	renewed, err := renewScript.Run(ctx, c.client, []string{REDIS_CACHE_KEY, REDIS_CACHE_VERSION_KEY},
		hex.EncodeToString(sum[:]), REDIS_CACHE_TTL.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return renewed == 1, nil
}
