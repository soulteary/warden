#!/usr/bin/env bash
# 报告各语言翻译文档相对 enUS 的行数差距。
#
# 这是一个信息性工具，不是 CI 门禁：deDE/frFR/itIT/jaJP/koKR 属于「尽力而为」级别，
# 允许滞后（见 docs/enUS/CONTRIBUTING.md 的翻译策略）。运行时字符串则由
# `go test ./locales/` 强制校验，那才是硬性门禁。
#
# 用法: scripts/check-docs-parity.sh [滞后百分比阈值，默认 20]

set -euo pipefail

THRESHOLD="${1:-20}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="$ROOT/docs"
REFERENCE="enUS"
LANGS=(zhCN deDE frFR itIT jaJP koKR)

printf '%-22s %8s' "文档" "$REFERENCE"
for lang in "${LANGS[@]}"; do printf ' %8s' "$lang"; done
printf '\n'

lagging=0
for doc in "$DOCS_DIR/$REFERENCE"/*.md; do
  name="$(basename "$doc")"
  ref_lines="$(wc -l < "$doc" | tr -d ' ')"
  printf '%-22s %8s' "$name" "$ref_lines"
  for lang in "${LANGS[@]}"; do
    target="$DOCS_DIR/$lang/$name"
    if [[ ! -f "$target" ]]; then
      printf ' %8s' "缺失"
      lagging=$((lagging + 1))
      continue
    fi
    lines="$(wc -l < "$target" | tr -d ' ')"
    gap=$(( (ref_lines - lines) * 100 / (ref_lines > 0 ? ref_lines : 1) ))
    if (( gap >= THRESHOLD )); then
      printf ' %6s%%↓' "$gap"
      lagging=$((lagging + 1))
    else
      printf ' %8s' "$lines"
    fi
  done
  printf '\n'
done

printf '\n滞后 >=%s%% 的文档数: %s\n' "$THRESHOLD" "$lagging"
printf '正本语言: enUS / zhCN。滞后文档顶部应带有翻译状态横幅。\n'
