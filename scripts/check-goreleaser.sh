#!/bin/sh
# Shared preflight for PATH and explicit executable overrides.
set -eu
ROOT=$(cd "$(dirname "$0")/.." && pwd)
expected=$(cat "$ROOT/.goreleaser-version")
case "$expected" in
  ''|*[!0-9.]*) echo "Invalid .goreleaser-version" >&2; exit 1 ;;
esac
tool=${1:-${GORELEASER:-goreleaser}}
if ! command -v "$tool" >/dev/null 2>&1; then
  echo "GoReleaser $expected is required; executable not found: $tool. See docs/releasing.md." >&2
  exit 1
fi
if ! output=$("$tool" --version 2>&1); then
  echo "Cannot query GoReleaser version: $tool. See docs/releasing.md." >&2
  exit 1
fi
actual=$(printf '%s\n' "$output" | sed -n 's/^GitVersion:[[:space:]]*//p')
if [ "$actual" != "$expected" ]; then
  echo "GoReleaser version mismatch: expected $expected; received ${actual:-unknown}. See docs/releasing.md." >&2
  exit 1
fi
printf 'GoReleaser %s verified\n' "$expected"
