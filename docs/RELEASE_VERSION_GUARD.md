# Fresh version publication

The release workflow accepts a new `vX.Y.Z` tag or a manual request for an
already existing Git tag whose Release and exact GHCR version image do not yet
exist. It never deletes or adopts an existing Release, replaces an asset, or
uses an administrator/force override. It does not run automatically on main.

The same-tag workflow concurrency group covers both triggers and does not cancel
an in-progress run. Guard code is checked out from `github.workflow_sha` into a
separate tools directory; build source is checked out from the selected tag.
The checked-out full commit must equal the tag's resolved commit (and the push
commit for tag pushes). Manual publication of an older tag therefore uses the
current workflow's guard, not a possibly absent historical helper. Both
checkouts disable persisted credentials.

Before builds, and again immediately before registry publication, the Python
preflight requires an explicit GitHub published-tag 404, a complete bounded
**authenticated** release inventory with no matching published or draft release,
and an exact GHCR version manifest 404 with `MANIFEST_UNKNOWN`. The workflow's
contents-write token sees its drafts; the inventory budget is 1,000 entries and
fails closed if exhausted. Authentication failures, denied/rate-limited requests,
redirects, invalid or oversized bodies, unexpected registry errors and network
failures are not absence. Ambient HTTP proxies are disabled; HTTPS validation
remains enabled. GitHub credentials go only to its API; the anonymous pull token
goes only to the fixed GHCR manifest endpoint. Errors expose no server body,
URL or credential.

Final release publication uses **POST /releases**, not create-or-update.
The included HTTP response is consumed through a 1 MiB bounded strict parser.
The parser accepts the real `gh api --include` rendering: its status line ends
in LF while headers and the header/body separator use CRLF. It parses each
header line without rewriting body bytes; uniform LF/CRLF responses remain
supported. Only explicit HTTP **201**, valid JSON content type and an unambiguous bound
draft record authorize its ID. HTTP 200/202/redirect responses are refused even
when `gh` exits zero and supplies a plausible existing release body. A
collision fails and cannot fall back to update. Only the newly returned draft
ID receives the fixed eight asset uploads; each is a POST without replacement.
Only this ID is then finalized. GitHub documents `target_commitish` as ignored
when the tag exists, so that response field is not treated as source identity.
The local exact tag/commit comparison and explicit image revision label carry
that pipeline binding. One full source revision and one build timestamp are
passed to both standalone binaries and Docker build arguments; this is not a
promise that independently compiled executable bytes will be identical.

If any upload or later step fails, the new draft/image may remain. There is no
automatic deletion, retry or adoption of that partial version. Re-running the
same version will refuse existing state; a separately reviewed recovery decision
or a new version is required. Floating aliases (`major.minor`, `major`, `latest`)
retain their existing rolling semantics; they are not immutable version tags.

This is a **pipeline guard**, not a server-side registry lock. A different actor
can race the manifest read and push; same-workflow concurrency does not lock
external writers. Git tag protection and registry immutability, when desired,
are separate administrative controls. An API response with `immutable:false`
provides no future deletion/overwrite lock. No such server protection is claimed
by this change. No release, registry mutation, Docker build or application was
executed by the local tests.

Run the offline protocol controls and the installed-CLI loopback renderer regression with:

```sh
python3 -I -B scripts/test_release_version_guard.py
python3 -I -B scripts/test_release_gh_envelope.py
```

They use synthetic HTTP replies and a local inert `gh` process to exercise the
actual publication Bash. They prove guard decisions, create-collision refusal,
fixed-ID upload/finalization and failure propagation, not real GitHub/GHCR
publication. The second test starts only an owned ephemeral loopback HTTP server
and invokes the installed `gh` against that fixed local endpoint. It uses an
empty `GH_CONFIG_DIR`, a synthetic token, no inherited application environment,
and no real GitHub or registry request. Missing `gh` or a forbidden local bind
fails rather than skipping the renderer test. It checks actual mixed-line 201
acceptance and actual CLI exit-zero 200 refusal. The path-scoped CI job also runs these controls; PyYAML must be
available in its hosted Python environment or the job fails.

Official protocol references:

- https://docs.github.com/en/rest/releases/releases#create-a-release
- https://docs.github.com/en/rest/releases/releases#list-releases
- https://docs.github.com/en/rest/releases/releases#get-a-release-by-tag-name

## Bounded create-response diagnosis

A refusal prints `release create response refused: <fixed-reason>` and exits
nonzero. Reasons are limited to response/header budgets, header framing/syntax,
HTTP status, media type, JSON syntax/duplicates/nonfinite values, record shape,
record binding or unknown. No body, URL, headers, exception message or token is
printed. A parser refusal may occur after GitHub created a draft: preserve that
state and investigate; it is not permission to retry, delete or adopt it.

The formatter seam is defined by upstream GitHub CLI `v2.103.3`
[`processResponse` and `printHeaders`](https://github.com/cli/cli/blob/v2.103.3/pkg/cmd/api/api.go).
`fmt.Fprintln` writes the status LF and `printHeaders` writes header CRLF.
The runner image recorded for the first run,
[`ubuntu24/20261101.360`](https://github.com/actions/runner-images/blob/ubuntu24/20261101.360/images/ubuntu/Ubuntu2404-Readme.md),
lists GitHub CLI 2.103.3. Actual local tests use the independently installed CLI
and do not claim to execute that hosted runner version or replay a retained
original create response. The original raw response was not captured.
