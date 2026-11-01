# Snapshot age metrics

The application `/metrics` handler calculates `warden_snapshot_age_seconds` at scrape time. It grows even if refresh scheduling stops. Existing metric names and gauge types remain unchanged; negative clock skew is clamped to zero. The age basis follows existing mode semantics: legacy/shadow use LoadedAt, while V2 uses persisted source-success time. Redis restoration or TTL renewal does not rejuvenate V2 age.

Each scrape captures one immutable snapshot root for age, degraded state and timestamps. Handlers retain independent sources, without replacing a global callback. The event-updated compatibility gauges remain available through the package registry, but the application handler replaces their exported samples with the current collection.

Additional gauges have no labels:
- `warden_snapshot_loaded_timestamp_seconds` and `warden_snapshot_age_known`
- `warden_snapshot_source_succeeded_timestamp_seconds` and `warden_snapshot_source_time_known`
- `warden_snapshot_published_timestamp_seconds` and `warden_snapshot_publication_time_known`
- `warden_snapshot_restored_timestamp_seconds` and `warden_snapshot_restoration_time_known`

Unknown times are zero with the corresponding known gauge set to zero. A legacy Redis restore cannot establish source-success evidence. For source age, use `time() - warden_snapshot_source_succeeded_timestamp_seconds` only where `warden_snapshot_source_time_known == 1`; do not treat an unknown timestamp as a fresh source. Metrics do not enforce freshness on data reads or prove the shared Redis key currently exists.
