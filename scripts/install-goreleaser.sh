#!/bin/sh
# Install only a pinned upstream archive verified against reviewed local hashes.
set -eu
ROOT=$(cd "$(dirname "$0")/.." && pwd)
if [ "$#" -ne 1 ] || [ -z "$1" ]; then
  echo "Usage: $0 <destination-directory>" >&2
  exit 1
fi
version=$(cat "$ROOT/.goreleaser-version")
case "$version" in
  ''|*[!0-9.]*) echo "Invalid .goreleaser-version" >&2; exit 1 ;;
esac
platform=$(uname -s)
case "$platform" in Darwin|Linux) ;; *) echo "Unsupported build host: $platform" >&2; exit 1 ;; esac
arch=$(uname -m)
case "$arch" in arm64|aarch64) arch=arm64 ;; x86_64|amd64) arch=x86_64 ;; *) echo "Unsupported build architecture: $arch" >&2; exit 1 ;; esac
archive="goreleaser_${platform}_${arch}.tar.gz"
expected=$(awk -v name="$archive" '$2 == name { print $1 }' "$ROOT/scripts/goreleaser-checksums.txt")
if [ "${#expected}" -ne 64 ]; then
  echo "Missing or ambiguous pinned checksum for $archive" >&2
  exit 1
fi
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT HUP INT TERM
curl --fail --location --silent --show-error --proto '=https' --proto-redir '=https' \
  "https://github.com/goreleaser/goreleaser/releases/download/v$version/$archive" -o "$work/$archive"
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$work/$archive" | awk '{print $1}')
else
  actual=$(shasum -a 256 "$work/$archive" | awk '{print $1}')
fi
if [ "$actual" != "$expected" ]; then
  echo "Checksum mismatch for $archive; refusing to extract or execute it." >&2
  exit 1
fi
tar -xzf "$work/$archive" -C "$work" goreleaser
sh "$ROOT/scripts/check-goreleaser.sh" "$work/goreleaser"
mkdir -p "$1"
install -m 755 "$work/goreleaser" "$1/goreleaser"
printf 'Installed GoReleaser %s to %s/goreleaser\n' "$version" "$1"
