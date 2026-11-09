# Exact Warden v1.6.3 owned-attempt recovery

This workflow is separate from fresh-version publication. It can only address
the known empty draft `439146629` created during run `42620110898`, job
`129169750296`, whose only failed step was create-response parsing (23).
It cannot create or delete a Release, adopt a Release by tag, move a Git tag,
push/rebuild/resign an image, or overwrite an asset. The original failed run and
its partial image/draft are preserved. Do not rerun the fresh-only publisher.

The fixed source is `6fe54f24f78c28225882c2aae779e1020bb8f87e`, tree
`d98d9dad3f1b2ca916802301d8175da662246782`, and signed annotated tag object
`e83fd755541be594032279d14ce4c610bb449da7`. The existing index is
`sha256:9ac7db4184d922ddaadfa99f9e65a59e88641f08181f6eab83a57c19a9e3234c`.
The original actual build log records Go1.27.3/linux-amd64 and the single release
build timestamp `2026-11-09T12:51:58+0000`.

## Admission and dry plan

This candidate binds the exact Root-captured image qualification and recovery
capsule by two fixed SHA256 constants. The records attest only the independently
reviewed partial signed image and this exact attempt. They are not recovery
execution approval: reviewed main/CI, fresh safe plan, and every live admission
check remain required. Full release/assets/runtime qualification remains false.

After Root independently qualifies the exact partial image's raw two-platform
Merkle graph, source/version/date/User metadata and source-constrained keyless
signature (including a specific wrong-source negative), Root maps the actual
accepted evidence into `image-qualification.json`. The record refers to the
actual raw-graph manifest and actual cryptographic capture hashes. It is not a
boolean grant or crypto proof by JSON alone. Root binds its exact SHA256 and an
exact `capsule.json` SHA256 in reviewed source. The capsule names only this
attempt and an independently chosen nonce. There is no CLI override for either
hash, target, source, tag, image or draft ID.

Use the manual workflow from main after merge. `plan` has contents-write for owned draft visibility and
outputs current safe observations plus missing Root bindings; it performs no
build or remote write. The helper checks actual tag ref/object, source/tree,
original run/event/attempt/only-step23 failure, exact draft body/author/empty
assets, absent published tag and current version/digest index. It then calls
Cosign with the exact original workflow identity, issuer, source SHA, tag ref,
repository and push trigger. A successful read-only plan with unbound Root data
is not execution admission. `recover` requires the Root bindings before even
selecting/downloading the compiler. The helper rejects workflow reruns
(`GITHUB_RUN_ATTEMPT != 1`), nonmanual execution and unreviewed branch contexts.

Root must review the actual dry plan before invoking recovery. Same-version
concurrency serializes this workflow with fresh publication; it does not exclude
other administrators or registry writers. All checks are as-of observations,
not a server-side transaction or future immutability promise.

## Rebuild and SBOM

Only original source6fe is compiled. The helper verifies HEAD/tree, every tracked
working-file Git blob and executable bit, and absence of extra/ignored files.
This rejects Git replacement or assume-unchanged worktree masking. Source is
checked again after build. It is independent of the newer recovery-tool revision.

The compiler must report exactly `go version go1.27.3 linux/amd64`. Dependency
download/verification and all six fixed OS/architecture builds use a minimal
public-tool environment, explicit module cache, CGO0 and original linker
Version1.6.3/full Commit6fe/release BuildDate. No application configuration or
GH token is forwarded to compiler commands. A separate actual rebuild UTC and
recovery tool/workflow SHA/run ID are recorded.

The original standalone dist was not retained as a downloadable artifact. The
Docker build record is not that dist. Rebuilding the same source/compiler/linker
fields does **not** prove equality with original unpublished bytes: paths,
build IDs and tools can differ. The preserved BuildDate is the release identity
timestamp, not a claim that later compilation happened at that old clock time.
Actual new bytes receive actual new hashes; historical checksum rows are not
substituted for them.

The pinned Anchore action scans `registry:...@9ac` with explicit Syftv1.51.1
(the original observed generator version). Dependency snapshots, artifact
upload and Release-asset auto-upload are all explicitly false, so generating
the SBOM cannot alter the draft ahead of admission. SPDX shape/hash checks do
not independently authenticate SBOM provenance; the reviewed fixed workflow's
scanner input/action/tool and actual output are the producer evidence.
The single SPDX scan is explicitly linux/amd64, matching the original runner
architecture; it is not a claim of separate SBOM coverage for both image
platforms. Regenerated SBOM bytes may differ from the original unretained document.

## Upload and unknown/partial outcomes

Before the first upload, the helper revalidates the Root bindings and all live
attempt/source/draft-empty/current-index/signature checks. It reads the six new
binaries and new SBOM, verifies the recorded binary hashes, and generates seven
checksum rows excluding checksums.txt itself. Only the eight literal filenames
are POSTed to `/releases/439146629/assets`. Each needs explicit201 and an exact
name/size/digest/uploaded response. There is no clobber, asset delete or skip-if-
exists branch. Exact source/index/signature and this invocation's eight returned
asset IDs are checked again before PATCHing only that same ID to draftfalse.
No packages-write or id-token-write permission is requested.

A before-request intent and each known successful upload metadata record are
published atomically without replacement. A lost response, unexpected status or
local result-write failure stops immediately; remote state may already have
changed. Safe failure reports retain expected filenames/size/digests and known
uploaded IDs, fixed stage/reason/status and unknown state. No exception text,
URLs, credentials, command argv or raw logs are exported. Raw local logs remain
private to the owned directory; per-phase names prevent plan/build/upload log
collisions. Timed-out owned builder/reader commands are not killed/restarted by
the helper and remain unknown. No partial-directory or role is automatically
reused.

The manual `inspect-partial` mode is read-only. It reports the fixed draft ID's
current bounded asset metadata and explicitly says ownership/execution
qualification and retry authorization were **not** inferred. Once any asset is
present, ordinary recovery's empty-draft check refuses. Root must reconcile the
actual API records against safe known/unknown upload facts and approve a new,
separately reviewed exact partial-state plan. This workflow contains no resume,
adoption or replacement mechanism. A timeout after final PATCH may already
mean published; inspect the actual ID before deciding anything further.

Only safe plan/build/upload reports are workflow artifacts. Compiled binaries,
private logs and readiness files are not broadly uploaded. Intended Release
assets are uploaded only through the gated fixed-ID phase. Current official
image signing binds the preserved index, not independently the rebuilt
standalone bytes. Full release acceptance, binary metadata probes, service,
Redis/session migration and deployment remain separate gates.

## Local controls and execution boundary

Run `python3 -I -B scripts/warden163_recovery/test_recover.py` for offline tests.
They use explicit synthetic qualification/hash/HTTP/crypto/command collaborators,
not real qualification or Go execution. They cover changed current bindings,
wrong index/signature, Root hash/shape failures, exact ID/eight uploads, lost ACK,
partial refusal, source-byte tampering despite assume-unchanged/ignored files,
atomic no-replace roles, fixed workflow/permissions/scanner settings, and phase
log isolation with a real inert Python child only. No Go, Docker, ELF, network,
GitHub POST or recovery was run by the author.

Native scanner target reference: https://oss.anchore.com/docs/reference/syft/cli/

Both job scopes explicitly grant `actions: read` solely to inspect the original
Actions run and jobs. Both jobs require contents-write for owned draft visibility;
no Actions write, packages write or OIDC permission is granted.

## Draft visibility and read-only plan

The first actual plan stopped at draft GET with HTTP403 after the Actions run
and jobs reads passed; no remote write was attempted. GitHub documents that
draft listings require push access, unlike public release reads:
https://docs.github.com/en/rest/releases/releases#list-releases .
The plan job therefore grants contents-write plus actions-read. This is token
capability, not permission for the helper to mutate: Client rejects every
non-GET method and every GET body before opening a request in plan, inspect-partial
and rebuild. Only the explicit upload phase can use POST/PATCH. This does not
change the independent published-tag404 check or interpret a draft as published
absence. Actual workflow-token/API compatibility remains a live plan gate.
