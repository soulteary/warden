# Redis connection recovery

An initial Redis connection failure no longer removes the configured backend for the lifetime of the process. The initial verification keeps redis-kit's existing timeout and connection settings. If it fails, Warden retains a fresh lazy client with the same settings, since redis-kit closes the failed verified client.

No additional Redis bootstrap or writer-election attempts run during that startup. Sources still load into memory as before. After startup, existing Redis commands reconnect on demand using go-redis's bounded connection/command retries; there is no separate retry loop, client swapping or local fallback for shared writes.

The legacy cache, V2 store, writer lease, health probe and Redis replay guard keep the same client identity across recovery. Redis-disabled mode and cache formats remain unchanged. In Redis-enabled HMAC deployments, unavailable Redis rejects HMAC v2 requests; independent API-key/mTLS authentication follows the existing chain.

Recovery does not establish new source-success evidence by itself. A V2 restore retains persisted source time; a successful source refresh follows the existing publication path.
