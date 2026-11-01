# Optional snapshot response metadata

Authenticated data clients may request `Warden-Snapshot-Metadata: v1`. Supported list, pagination, user and lookup responses then include:

- `Warden-Snapshot-Metadata: v1`
- `Warden-Snapshot-Version`: digest of the complete effective snapshot, not a body signature
- `Warden-Source-Succeeded-At`: accepted resolved-source validation time, when known
- `Warden-Snapshot-Valid-Until`: source time plus the configured snapshot maximum age, when known

Times use RFC3339Nano. Headers and rows pin the same immutable generation. Metadata is added inside authentication; ordinary requests keep their existing headers, bodies and status codes. Negotiated responses add `Vary: Warden-Snapshot-Metadata` and `Cache-Control: private`. Error statuses still retain their original meanings; a valid 404 can carry evidence for the snapshot used to determine absence.

Legacy Redis recovery has no source time and omits both time headers. A new response or restoration time is not a substitute. Accepted local fallback describes the resolved set, not proof of remote success. These headers do not enforce freshness on the server, change JSON, or provide an upstream-signed revision. Consumers must trust the service/transport and bound cached data by the original source deadline, not receipt time.
