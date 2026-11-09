# Version command

This source adds `warden --version`: one JSON record from the existing
version-kit `Default()` API, followed by a newline. It includes the actual linker
`version`, the unmodified linker `commit` value (which may be abbreviated by the
build), and `build_date`, plus the Go runtime/platform/compiler
metadata supplied by that API. Development defaults remain `dev`/`unknown`; the
command never invents release values.

Use `--version` alone. Combining it with other arguments, positional arguments,
or any `--version=...` form is rejected without starting the application. A
separate `--version` token is reserved even when positioned as another option's
value. `-v` and
`-version` are not aliases. Arguments without a version request retain their
existing CLI behavior. Output failure exits nonzero and cannot fall through to
normal startup; partial output from a broken writer is not a successful record.

The command returns before the main startup banner, logger acquisition,
configuration parsing/loading, tracing, `NewApp`, Redis/source loading, scheduled
tasks or HTTP startup. Its branch reads no environment or configuration. Go
package initialization has already run, however: the existing logger init can
read `LOG_LEVEL`, and other package initializers remain unchanged. This is not a
claim that the whole process bypasses every environment read or logger init.

Tests exercise the real built program with explicit synthetic linker metadata,
a nonexistent synthetic config path, rejected mixed arguments and a broken
stdout descriptor. They do not establish a release identity or run a real
service. No dependency or ordinary application startup behavior changes.

The already released Warden v1.6.2 does **not** have this command. This source
change is not evidence that its existing image supports a metadata-only probe.
The reported commit field alone is not a guarantee of full source identity;
source-constrained signatures and image/source metadata remain separate evidence.
