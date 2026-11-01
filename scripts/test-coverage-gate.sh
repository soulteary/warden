#!/bin/sh
# Fixtures cover the exact threshold and duplicate blocks across package runs.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
check() {
  expected=$1
  actual=0
  sh "$script_dir/check-coverage.sh" "$tmp/profile" > "$tmp/result" 2>&1 || actual=$?
  if [ "$actual" -ne "$expected" ]; then
    cat "$tmp/result" >&2
    echo "expected exit $expected, got $actual" >&2
    exit 1
  fi
}
for mode in set count atomic; do
  printf 'mode: %s\nx.go:1.1,2.1 9499 1\nx.go:3.1,4.1 501 0\n' "$mode" > "$tmp/profile"
  check 1
  printf 'mode: %s\nx.go:1.1,2.1 95 1\nx.go:3.1,4.1 5 0\n' "$mode" > "$tmp/profile"
  check 0
  printf 'mode: %s\nx.go:1.1,2.1 9501 1\nx.go:3.1,4.1 499 0\n' "$mode" > "$tmp/profile"
  check 0
done
printf 'mode: atomic\nx.go:1.1,2.1 95 0\nx.go:1.1,2.1 95 2\nx.go:3.1,4.1 5 0\n' > "$tmp/profile"
check 0
printf 'mode: atomic\nx.go:1.1,2.1 95 1\nx.go:1.1,2.1 95 0\nx.go:3.1,4.1 5 0\n' > "$tmp/profile"
check 0
for body in '' 'mode: invalid' 'mode: atomic' 'mode: atomic
x.go:1.1,2.1 bad 1' 'mode: atomic
x.go:1.1,2.1 1 -1' 'mode: atomic
x.go:1.1,2.1 1 1
x.go:1.1,2.1 2 1'; do
  printf '%s\n' "$body" > "$tmp/profile"
  check 2
done
echo 'Coverage gate fixtures passed.'
