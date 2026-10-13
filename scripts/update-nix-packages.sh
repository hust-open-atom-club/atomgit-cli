#!/bin/sh
# Refresh nix/stable.nix and nix/latest.nix from the latest AtomGit release.
# Run inside the repository devShell so that nix-update, jq, curl, sed, git,
# and the Nix toolchain all come from the nixpkgs input pinned by the
# committed flake.lock instead of a moving channel or the runner image:
#
#   nix develop --no-update-lock-file . -c bash scripts/update-nix-packages.sh
set -eu

export GOPROXY="https://goproxy.cn"

lock_before=$(mktemp)
trap 'rm -f "$lock_before"' EXIT HUP INT TERM
cp flake.lock "$lock_before"

# Evaluation must never invent or refresh inputs on its own: a missing or
# stale lock is an error, not something this job silently repairs.
nix flake metadata --no-update-lock-file --json . >/dev/null

release_json=$(curl --fail --silent --show-error \
  https://api.atomgit.com/api/v5/repos/hust-open-atom-club/atomgit-cli/releases/latest)
stable_version=$(printf '%s' "$release_json" \
  | jq -er '.tag_name | select(test("^v[0-9]+\\.[0-9]+\\.[0-9]+([+-].*)?$")) | sub("^v"; "")')
stable_commit=$(printf '%s' "$release_json" \
  | jq -er '.target_commitish | select(test("^[0-9a-f]{40}$"))')
test -n "$stable_version"
test -n "$stable_commit"

commit_json=$(curl --fail --silent --show-error \
  "https://api.atomgit.com/api/v5/repos/hust-open-atom-club/atomgit-cli/commits/$stable_commit")
stable_build_date=$(printf '%s' "$commit_json" \
  | jq -er '(.commit.committer.date // .commit.author.date) | select(test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T"))')
test -n "$stable_build_date"

sed -i \
  -e "s|^  commit = \"[^\"]*\";|  commit = \"$stable_commit\";|" \
  -e "s|^  buildDate = \"[^\"]*\";|  buildDate = \"$stable_build_date\";|" \
  nix/stable.nix
test "$(sed -n 's/^  commit = "\([^"]*\)";/\1/p' nix/stable.nix)" = "$stable_commit"
test "$(sed -n 's/^  buildDate = "\([^"]*\)";/\1/p' nix/stable.nix)" = "$stable_build_date"

nix-update stable --flake --version "$stable_version" --build
nix-update latest --flake --version=skip --build

stable_out=$(nix build --no-link --print-out-paths .#stable | tail -n 1)
stable_binary="$stable_out/bin/ag-cli"
# The pinned v0.7.3 release still provides the old executable.
if [ ! -x "$stable_binary" ]; then stable_binary="$stable_out/bin/ag"; fi
version_json=$("$stable_binary" version --json)
printf '%s\n' "$version_json"
printf '%s' "$version_json" | jq -e \
  --arg version "v$stable_version" \
  --arg commit "$stable_commit" \
  --arg build_date "$stable_build_date" \
  '.version == $version and .commit == $commit and .buildDate == $build_date' \
  >/dev/null

rm -f result result-*

if ! cmp -s flake.lock "$lock_before"; then
  echo "flake.lock changed during the update; refusing to continue." >&2
  exit 1
fi
