#!/bin/sh
# Compare statement counts without rounding the displayed percentage.
set -eu
if [ "$#" -ne 1 ]; then
  echo "usage: $0 coverage.out" >&2
  exit 2
fi
LC_ALL=C awk '
function invalid(message) { print "invalid coverage profile: " message > "/dev/stderr"; failed=1; exit 2 }
NR == 1 { if ($0 !~ /^mode: (set|count|atomic)$/) invalid("unsupported mode"); next }
{
  if (NF != 3 || $1 !~ /^.+:[0-9]+\.[0-9]+,[0-9]+\.[0-9]+$/ || $2 !~ /^[0-9]+$/ || $3 !~ /^[0-9]+$/) invalid("malformed block")
  key=$1
  if ((key in statements) && statements[key] != $2) invalid("inconsistent statement count")
  statements[key]=$2
  if ($3+0 > 0) hit[key]=1
}
END {
  if (failed) exit 2
  for (key in statements) { total+=statements[key]; if (hit[key]) covered+=statements[key] }
  if (total == 0) { print "invalid coverage profile: no statements" > "/dev/stderr"; exit 2 }
  printf "Statement coverage: %.0f/%.0f (%.6f%%); minimum: 95%%\n", covered,total,100*covered/total
  if (covered*100 < total*95) exit 1
}
' "$1"
