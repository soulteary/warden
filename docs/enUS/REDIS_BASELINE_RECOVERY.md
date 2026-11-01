# Recovering a missing V2 baseline

In `REDIS_SNAPSHOT_MODE=v2`, an instance without a known-good snapshot retries the shared V2 baseline after a source load or identity-validation failure. Startup and background refreshes use the same rule. A recovered snapshot retains its persisted source-success and publication times; only local restoration time advances. Background source-failure counters remain failures.

Recovery uses the existing schema, policy, content and effective-set checks. It does not publish the failed input, renew a snapshot, or replace an existing known-good generation, including a valid empty set. An expired baseline remains expired for freshness checks; cached query behavior is unchanged.

The additional failure-recovery path does not apply to legacy/shadow mode or ONLY_LOCAL source failures. Existing all-records-rejected handling remains unchanged. If Redis is still unavailable or its baseline is incompatible, the instance remains without a baseline and retries after a later failed source refresh.

This describes the default bootstrap policy. With explicit `REDIS_SNAPSHOT_RECOVERY=stale`, an expired local snapshot may follow an eligible fresh V2 snapshot after a source refresh fails; see [migration configuration](REDIS_SNAPSHOT_MIGRATION.md). Recovery preserves original source-success time and failure diagnostics.
