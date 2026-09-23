#!/bin/sh
# Publish updated nix/stable.nix and nix/latest.nix through the AtomGit
# Contents API. The access token is read from the NIX_UPDATE_TOKEN
# environment variable and is never placed in a URL, a command-line
# argument, a file left behind in the workspace, or printed output. Error
# diagnostics are bounded, control-stripped, and redacted.
#
# Expected environment:
#   ATOMGIT_REPOSITORY  e.g. hust-open-atom-club/atomgit-cli (runner-provided)
#   ATOMGIT_REF_NAME    target branch for the commits (runner-provided)
#   NIX_UPDATE_TOKEN    repository secret with contents write access
set -eu

api_base=https://api.atomgit.com/api/v5
repository=${ATOMGIT_REPOSITORY:-}
branch=${ATOMGIT_REF_NAME:-}
token=${NIX_UPDATE_TOKEN:-}

# Validate context with explicit checks: ${VAR:?} exit codes differ between
# /bin/sh implementations (dash exits 2, bash exits 1).
if [ -z "$repository" ]; then
  echo "ATOMGIT_REPOSITORY is not set." >&2
  exit 1
fi

# Only character classes that cannot break out of the JSON string built
# below are accepted for the branch name.
case "$branch" in
  ''|*[!A-Za-z0-9._/-]*)
    echo "Refusing to publish to unexpected branch name '$branch'." >&2
    exit 1
    ;;
esac

redact_token() {
  T=$token awk '
    {
      t = ENVIRON["T"]
      n = length(t)
      s = $0
      out = ""
      while ((i = index(s, t)) > 0) {
        out = out substr(s, 1, i - 1) "<redacted>"
        s = substr(s, i + n)
      }
      print out s
    }'
}

summarize_body() {
  # $1: response body file. Bounded excerpt, no control characters, token
  # redacted. Raw bodies are never printed in full.
  body_file=$1
  if [ ! -s "$body_file" ]; then
    printf '%s\n' 'response body is empty'
    return
  fi
  # Do not truncate before redaction: a token crossing the input boundary
  # would become an unmatchable prefix that could be printed in the excerpt.
  tr -d '\000-\037\177' < "$body_file" | redact_token | cut -c1-300
}

if git diff --quiet -- nix/stable.nix nix/latest.nix; then
  echo "Nix packages are already current."
  exit 0
fi

if [ -z "$token" ]; then
  echo "NIX_UPDATE_TOKEN is not set; refusing to publish updates." >&2
  exit 1
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT HUP INT TERM

for path in nix/stable.nix nix/latest.nix; do
  if git diff --quiet -- "$path"; then
    echo "$path is already current."
    continue
  fi

  blob_sha=$(git rev-parse "HEAD:$path")
  # Every interpolated value is restricted to a JSON-safe alphabet: base64
  # content, a hex blob SHA, the validated branch name, and a fixed message.
  request_body=$(printf '{"content":"%s","sha":"%s","branch":"%s","message":"chore: update Nix packages"}' \
    "$(base64 < "$path" | tr -d '\n')" \
    "$blob_sha" \
    "$branch")

  # The authorization header is passed to curl through a temporary file so
  # the token never shows up in a process command line either.
  header_file="$work/authorization-header"
  response_file="$work/contents-api-response"
  printf 'Authorization: Bearer %s\n' "$token" > "$header_file"

  set +e
  http_status=$(curl --silent --show-error \
    --output "$response_file" \
    --write-out '%{http_code}' \
    --request PUT \
    --header @"$header_file" \
    --header 'Accept: application/json' \
    --header 'Content-Type: application/json' \
    --data "$request_body" \
    "$api_base/repos/$repository/contents/$path")
  curl_exit=$?
  set -e
  rm -f "$header_file"

  echo "$path: curl exit code $curl_exit, HTTP status $http_status"
  if [ "$curl_exit" -ne 0 ]; then
    exit 1
  fi
  case "$http_status" in
    200)
      echo "$path: updated on $branch"
      ;;
    *)
      echo "$path: API rejected the update; response summary follows."
      summarize_body "$response_file"
      exit 1
      ;;
  esac
done
