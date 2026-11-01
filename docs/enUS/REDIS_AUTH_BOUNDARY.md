# Redis replay protection at startup

When Redis is enabled and HMAC keys are configured, HMAC v2 requires the shared Redis replay store. An unavailable or uninitialized Redis client rejects HMAC requests with the existing unauthorized response; it does not select process-local replay protection. The Redis health check remains critical for this configuration.

API key and verified mTLS authentication retain their existing behavior. Explicitly disabling Redis retains the single-process replay guard and its documented multi-replica limitation. Legacy HMAC v1, when explicitly enabled, retains its migration behavior and does not gain v2 replay protection.

The initial-connection failure case follows the same fail-closed rule as a Redis outage after startup. This changes the erroneous startup fallback, without changing signature formats or response payloads.
