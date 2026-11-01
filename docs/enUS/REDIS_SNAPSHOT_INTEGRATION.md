# Redis 8 snapshot integration tests

CI runs a separate Linux job against a fresh Redis 8 service:

```sh
WARDEN_REDIS_INTEGRATION_ADDR=localhost:6379 \
WARDEN_REDIS_INTEGRATION_REQUIRE_COLD=true \
go test -race -count=1 -v -run '^TestRedisSnapshotIntegration' ./internal/cache
```

Use a disposable Redis 8 instance with permission to create temporary ACL users. Without an explicit address, these tests skip in the ordinary unit-test run. With an address, connection/version/ACL failures fail the tests rather than skip. The cold-script option requires a fresh script cache; the tests never flush scripts or databases. Each test uses a random namespace, deletes only its own keys, and removes its temporary ACL user.

The suite covers NOSCRIPT fallback, atomic missing/empty reads, publication/recovery timestamps, byte-identical TTL renewal, lease contention/expiry, expired-owner writes and release, stale revisions, malformed/oversized/wrong-type values, script ACL denial and canceled reads. No application mode, snapshot schema or production behavior changes.
