#!/bin/sh
# AtomGit CLI (ag-cli) — one-line installer
# Repository: https://atomgit.com/hust-open-atom-club/atomgit-cli
#
# Usage:
#   curl -fsSL https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/download/latest/install.sh | sh
#   AG_VERSION=vX.Y.Z sh install.sh
#   AG_FROM_SOURCE=1 sh install.sh    # build from this repository source (requires git and Go)

set -eu

REPO_OWNER="${AG_REPO_OWNER:-hust-open-atom-club}"
REPO_NAME="${AG_REPO_NAME:-atomgit-cli}"
BASE_URL="https://atomgit.com/${REPO_OWNER}/${REPO_NAME}"
# Release builds replace this placeholder with the current tag. When running
# the source template directly, set AG_VERSION or use AG_FROM_SOURCE=1.
_BUNDLED_TAG="__AG_RELEASE_TAG__"
DEFAULT_VERSION="${AG_DEFAULT_VERSION:-${_BUNDLED_TAG}}"
VERSION="${AG_VERSION:-$DEFAULT_VERSION}"

# Matches the AtomGit default branch (currently main).
GIT_REF="${AG_GIT_REF:-main}"

die() {
  echo "install.sh: $*" >&2
  exit 1
}

detect_os() {
  os=$(uname -s)
  case "$os" in
  Linux) echo linux ;;
  Darwin) echo darwin ;;
  *) die "unsupported OS: $os" ;;
  esac
}

detect_arch() {
  arch=$(uname -m)
  case "$arch" in
  x86_64 | amd64) echo amd64 ;;
  aarch64 | arm64) echo arm64 ;;
  loongarch64 | loong64) echo loong64 ;;
  *) die "unsupported CPU: $arch" ;;
  esac
}

pick_install_dir() {
  if [ -n "${AG_INSTALL_DIR:-}" ]; then
    echo "$AG_INSTALL_DIR"
    return
  fi
  prefix="${AG_PREFIX:-}"
  if [ -z "$prefix" ]; then
    if [ -w /usr/local/bin ] 2>/dev/null; then
      prefix="/usr/local"
    else
      prefix="$HOME/.local"
    fi
  fi
  echo "$prefix/bin"
}

# expected_checksum prints the SHA-256 for $asset from checksums.txt. The file
# covers every archive and both installers, so replacing one attachment alone
# breaks the check.
expected_checksum() {
  awk -v want="$asset" '
    {
      gsub(/\r/, "")
      name = $2
      sub(/^\*/, "", name)
      sub(/^\.\//, "", name)
      if (name == want) { print tolower($1); exit }
    }
  ' "${tmpdir}/checksums.txt"
}

# sha256_of prints the SHA-256 of one file. macOS ships shasum instead of
# sha256sum; openssl is the last fallback. Each branch captures the tool output
# first: a pipeline would hide the hash tool's status behind awk.
sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    out=$(sha256sum "$1") || return 1
    printf '%s\n' "${out%% *}"
  elif command -v shasum >/dev/null 2>&1; then
    out=$(shasum -a 256 "$1") || return 1
    printf '%s\n' "${out%% *}"
  elif command -v openssl >/dev/null 2>&1; then
    out=$(openssl dgst -sha256 "$1") || return 1
    printf '%s\n' "${out##* }"
  else
    return 1
  fi
}

# verify_archive refuses to continue unless checksums.txt lists $asset and the
# downloaded bytes match. The check fails closed: installing an unverified
# archive would leave HTTPS as the only defence against a tampered release.
verify_archive() {
  checksums_url="${BASE_URL}/releases/download/${tag}/checksums.txt"
  if ! curl -fsSL "$checksums_url" -o "${tmpdir}/checksums.txt"; then
    die "Failed to download checksums.txt from ${checksums_url}; refusing to install"
  fi

  expected=$(expected_checksum) || die "Failed to parse checksums.txt at ${tmpdir}/checksums.txt"
  if [ -z "$expected" ]; then
    die "checksums.txt does not contain an entry for ${asset}; refusing to install"
  fi
  if [ "${#expected}" -ne 64 ]; then
    die "Invalid checksum entry for ${asset} in checksums.txt; refusing to install"
  fi
  case "$expected" in
  *[!0-9a-f]*) die "Invalid checksum entry for ${asset} in checksums.txt; refusing to install" ;;
  esac

  actual=$(sha256_of "${tmpdir}/${asset}") ||
    die "Failed to compute SHA-256 for ${asset}; install sha256sum, shasum, or openssl"
  if [ -z "$actual" ]; then
    die "Failed to compute SHA-256 for ${asset}; install sha256sum, shasum, or openssl"
  fi

  if [ "$actual" != "$expected" ]; then
    die "Checksum mismatch for ${asset} (expected ${expected}, got ${actual}); refusing to install"
  fi
  echo "Checksum verified: ${asset}"
}

install_binary() {
  if [ "$VERSION" = "__AG_RELEASE_TAG__" ]; then
    die "The source installer template is not bound to a release. Set AG_VERSION=vX.Y.Z, or use AG_FROM_SOURCE=1 to install from source"
  fi

  os=$(detect_os)
  arch=$(detect_arch)
  tmpdir=$(mktemp -d)
  trap "rm -rf \"$tmpdir\"" EXIT INT HUP

  tag="$VERSION"
  asset="ag_${os}_${arch}.tar.gz"
  url="${BASE_URL}/releases/download/${tag}/${asset}"

  echo "Downloading ${url} ..."
  if ! curl -fsSL "$url" -o "${tmpdir}/${asset}"; then
    die "Failed to download the prebuilt archive. Verify that ${tag} is published with an asset named ${asset}. You can also use: AG_FROM_SOURCE=1 sh install.sh"
  fi

  verify_archive

  (cd "$tmpdir" && tar -xzf "$asset")
  if [ ! -f "${tmpdir}/ag-cli" ]; then
    die "ag-cli was not found in the archive"
  fi

  dest=$(pick_install_dir)
  mkdir -p "$dest"
  if [ ! -w "$dest" ] && command -v sudo >/dev/null 2>&1; then
    echo "Installing to $dest (sudo) ..."
    sudo install -m 0755 "${tmpdir}/ag-cli" "${dest}/ag-cli"
  else
    echo "Installing to $dest ..."
    install -m 0755 "${tmpdir}/ag-cli" "${dest}/ag-cli"
  fi

  case ":${PATH:-}:" in
  *:"$dest":*) ;;
  *)
    echo "Add this directory to PATH, for example by appending to ~/.profile:"
    echo "  export PATH=\"$dest:\$PATH\""
    ;;
  esac

  "${dest}/ag-cli" --help >/dev/null 2>&1 || true
  echo "ag-cli installed: ${dest}/ag-cli"
}

install_from_source() {
  command -v git >/dev/null 2>&1 || die "git is required"
  command -v go >/dev/null 2>&1 || die "Go toolchain is required"

  tmpdir=$(mktemp -d)
  trap "rm -rf \"$tmpdir\"" EXIT INT HUP

  echo "Cloning ${BASE_URL}.git (ref: ${GIT_REF}) ..."
  git clone --depth 1 --branch "$GIT_REF" "${BASE_URL}.git" "${tmpdir}/src"

  dest=$(pick_install_dir)
  mkdir -p "$dest"
  echo "Building ..."
  (cd "${tmpdir}/src" && go build -o "${tmpdir}/ag-cli" ./cmd/ag-cli)

  if [ ! -w "$dest" ] && command -v sudo >/dev/null 2>&1; then
    echo "Installing to $dest (sudo) ..."
    sudo install -m 0755 "${tmpdir}/ag-cli" "${dest}/ag-cli"
  else
    echo "Installing to $dest ..."
    install -m 0755 "${tmpdir}/ag-cli" "${dest}/ag-cli"
  fi

  echo "ag-cli installed: ${dest}/ag-cli"
}

main() {
  if [ -n "${AG_FROM_SOURCE:-}" ] && [ "$AG_FROM_SOURCE" != "0" ]; then
    install_from_source
  else
    install_binary
  fi
}

main "$@"
