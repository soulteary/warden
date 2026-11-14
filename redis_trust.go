package main

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisForeignKeySample is how many keys one SCAN step asks for when looking
// for keys Warden did not write. One step keeps the check cheap; it is a
// sample, so a negative result does not prove the instance is dedicated.
const redisForeignKeySample = 200

// redisHoldsForeignKeys reports whether a sample of the Redis keyspace
// contains a key that is not Warden's. Every key Warden writes carries
// "warden:" in its name (the legacy cache, snapshots, refresh locks and
// replay nonces).
func redisHoldsForeignKeys(ctx context.Context, client redis.Cmdable) (bool, error) {
	keys, _, err := client.Scan(ctx, 0, "", redisForeignKeySample).Result()
	if err != nil {
		return false, err
	}
	for _, key := range keys {
		if !strings.Contains(key, "warden:") {
			return true, nil
		}
	}
	return false, nil
}

// warnIfRedisIsShared logs a warning when the Redis database also holds keys
// of another application. The cached rule set is adopted at startup without
// any proof of who wrote it, so every other writer to the instance can change
// the allow list. A failed check is ignored: it must not affect startup.
func (app *App) warnIfRedisIsShared() {
	if app.redisClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	shared, err := redisHoldsForeignKeys(ctx, app.redisClient)
	if err != nil || !shared {
		return
	}
	app.log.Warn().Msg("this Redis database holds keys that Warden did not write: the allow list cached under warden:users:cache is adopted at startup, so every application that can write to this database can change who is allowed; give Warden a dedicated Redis instance")
}
