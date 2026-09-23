#!/bin/sh
# Install a pinned Nix release after verifying the published tarball digest.
# The version and SHA-256 below are the review anchors: never download the
# expected digest at run time. To bump, fetch the new versioned tarball and
# its published digest from https://releases.nixos.org/nix/<version>/,
# compare them, and update both values in one reviewed change.
set -eu

version=2.34.8
system=x86_64-linux
expected_sha256=2c2e146b80834fe0ca201b51deeb939405b4f18e8d2071bf80b10f8123c50464

case "$(uname -s).$(uname -m)" in
  Linux.x86_64) ;;
  *)
    echo "Unsupported platform for the pinned Nix installer: $(uname -s) $(uname -m)" >&2
    exit 1
    ;;
esac

archive="nix-$version-$system.tar.xz"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT HUP INT TERM

curl --fail --location --silent --show-error \
  --proto '=https' --proto-redir '=https' --tlsv1.2 \
  "https://releases.nixos.org/nix/nix-$version/$archive" -o "$work/$archive"

actual_sha256=$(sha256sum "$work/$archive" | awk '{print $1}')
if [ "$actual_sha256" != "$expected_sha256" ]; then
  echo "Checksum mismatch for $archive; refusing to extract or execute it." >&2
  echo "Expected $expected_sha256, got $actual_sha256." >&2
  exit 1
fi

# Only after the digest check passes may the archive be unpacked and its
# installer executed. --no-channel-add keeps the mutable nixpkgs-unstable
# channel out of the profile; all tools come from the repository flake.lock.
tar -xJf "$work/$archive" -C "$work"
sh "$work/nix-$version-$system/install" --no-daemon --no-channel-add

printf 'Installed Nix %s into the user profile\n' "$version"
