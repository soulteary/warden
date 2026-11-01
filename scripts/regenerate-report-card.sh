#!/bin/sh
# Run after goreportcard-action installs its pinned CLI and native tools.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
WARDEN_REAL_MISSPELL=$(command -v misspell)
export WARDEN_REAL_MISSPELL
PATH="$script_dir/report-tools:$PATH"
export PATH

# The English dictionary is not suitable for translated documentation.
# English docs retain a separate, failing spelling check.
for document in ./*.md; do
    case "$document" in
        ./README.*.md) continue ;;
    esac
    "$WARDEN_REAL_MISSPELL" -error "$document"
done
"$WARDEN_REAL_MISSPELL" -error docs/enUS pkg/warden/README.md example/README.md example/basic/README.md example/advanced/README.md .github/ISSUE_TEMPLATE/bug_report.md .github/ISSUE_TEMPLATE/feature_request.md .github/ISSUE_TEMPLATE/question.md .github/pull_request_template.md
sh "$script_dir/test-report-spelling.sh"
# The CLI skips this linter for repositories over its file limit.
ineffassign ./...
goreportcard-cli -d . -svg .github/goreportcard.svg -report .github/goreportcard-report.md -t 0
