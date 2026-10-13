#!/bin/sh

set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

GO=${GO:-go}
GOVULNCHECK_VERSION=${GOVULNCHECK_VERSION:-v1.7.0}
GOVULNDB=${GOVULNDB:-https://vuln.go.dev}
VULNCHECK_TARGETS=${VULNCHECK_TARGETS:-linux/amd64 linux/arm64 linux/loong64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}
VULNCHECK_PACKAGE_TARGETS=${VULNCHECK_PACKAGE_TARGETS:-linux/amd64 darwin/amd64 windows/amd64}
VULNCHECK_BUILD=${VULNCHECK_BUILD:-1}
RELEASE_GO_TOOLCHAIN=${RELEASE_GO_TOOLCHAIN:-$(sed -n 's/^toolchain[[:space:]][[:space:]]*//p' go.mod)}

work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' 0 HUP INT TERM

if [ -n "${VULNCHECK_BINARY_DIR:-}" ]; then
	binary_dir=$VULNCHECK_BINARY_DIR
	mkdir -p "$binary_dir"
else
	binary_dir=$work_dir/binaries
	mkdir -p "$binary_dir"
fi

case "$VULNCHECK_BUILD" in
	0|1) ;;
	*)
		echo "VULNCHECK_BUILD must be 0 or 1" >&2
		exit 2
		;;
esac

case "$RELEASE_GO_TOOLCHAIN" in
	go[0-9]*.[0-9]*.[0-9]*) ;;
	*)
		echo "Invalid release Go toolchain: ${RELEASE_GO_TOOLCHAIN:-empty}" >&2
		exit 1
		;;
esac

scanner=${GOVULNCHECK:-}
if [ -z "$scanner" ]; then
	tool_dir=$work_dir/tools
	mkdir -p "$tool_dir"
	if ! GOBIN="$tool_dir" "$GO" install "golang.org/x/vuln/cmd/govulncheck@$GOVULNCHECK_VERSION"; then
		echo "Failed to install govulncheck $GOVULNCHECK_VERSION" >&2
		exit 1
	fi
	scanner=$tool_dir/govulncheck
fi

if ! scanner_version=$("$scanner" -version 2>&1); then
	echo "Failed to query govulncheck version" >&2
	printf '%s\n' "$scanner_version" >&2
	exit 1
fi
printf '%s\n' "$scanner_version"
case "$scanner_version" in
	*"Scanner: govulncheck@$GOVULNCHECK_VERSION"*) ;;
	*)
		echo "Expected govulncheck $GOVULNCHECK_VERSION" >&2
		exit 1
		;;
esac

target_parts() {
	target=$1
	case "$target" in
		*/*) ;;
		*)
			echo "Invalid vulnerability target: $target" >&2
			return 2
			;;
	esac
	target_goos=${target%/*}
	target_goarch=${target#*/}
	case "$target_goos/$target_goarch" in
		*[!a-z0-9_/-]*|/*|*/|*/*/*)
			echo "Invalid vulnerability target: $target" >&2
			return 2
			;;
	esac
}

binary_path() {
	target_parts "$1"
	printf '%s/ag-%s-%s\n' "$binary_dir" "$target_goos" "$target_goarch"
}

if [ "$VULNCHECK_BUILD" = 1 ]; then
	for target in $VULNCHECK_TARGETS; do
		target_parts "$target"
		binary=$(binary_path "$target")
		echo "=== Build vulnerability target: $target ==="
		GOOS="$target_goos" GOARCH="$target_goarch" CGO_ENABLED=0 \
			"$GO" build -trimpath -o "$binary" ./cmd/ag-cli
	done
else
	for target in $VULNCHECK_TARGETS; do
		binary=$(binary_path "$target")
		if [ ! -f "$binary" ]; then
			echo "Missing prebuilt vulnerability target: $binary" >&2
			exit 1
		fi
	done
fi

for target in $VULNCHECK_TARGETS; do
	binary=$(binary_path "$target")
	if ! version_output=$("$GO" version "$binary" 2>&1); then
		echo "Failed to read Go version from $target binary" >&2
		printf '%s\n' "$version_output" >&2
		exit 1
	fi
	printf '%s\n' "$version_output"
	case "$version_output" in
		*": $RELEASE_GO_TOOLCHAIN") ;;
		*)
			echo "Expected $target binary built with $RELEASE_GO_TOOLCHAIN" >&2
			exit 1
			;;
	esac
done

run_informational_scan() {
	level=$1
	target=$2
	shift 2
	target_parts "$target"
	echo "=== $level-level inventory: $target ==="
	set +e
	GOOS="$target_goos" GOARCH="$target_goarch" CGO_ENABLED=0 \
		"$scanner" -db="$GOVULNDB" -scan="$level" "$@"
	status=$?
	set -e
	case "$status" in
		0)
			echo "No $level-level findings for $target."
			;;
		3)
			echo "$level-level findings detected for $target (triage only)."
			;;
		*)
			echo "$level-level vulnerability scan failed for $target with exit code $status" >&2
			return "$status"
			;;
	esac
}

# Module findings are independent of GOOS/GOARCH. They are informational: only
# a reachable symbol in a release binary blocks the gate.
run_informational_scan module linux/amd64 -C ./cmd/ag-cli

# This repository has OS-specific Go files but no architecture-specific Go
# files. One representative architecture per release OS therefore exposes the
# package-only delta without repeating identical package scans for every arch.
for target in $VULNCHECK_PACKAGE_TARGETS; do
	run_informational_scan package "$target" -C ./cmd/ag-cli .
done

reachable_findings=0
for target in $VULNCHECK_TARGETS; do
	binary=$(binary_path "$target")
	target_parts "$target"
	echo "=== Reachable binary scan: $target ==="
	set +e
	GOOS="$target_goos" GOARCH="$target_goarch" CGO_ENABLED=0 \
		"$scanner" -db="$GOVULNDB" -mode=binary "$binary"
	status=$?
	set -e
	case "$status" in
		0)
			echo "No reachable vulnerabilities for $target."
			;;
		3)
			echo "Reachable vulnerabilities detected for $target." >&2
			reachable_findings=1
			;;
		*)
			echo "Reachable vulnerability scan failed for $target with exit code $status" >&2
			exit "$status"
			;;
	esac
done

if [ "$reachable_findings" -ne 0 ]; then
	echo "Reachable vulnerabilities were found in one or more release targets." >&2
	exit 3
fi

echo "All release targets passed the reachable vulnerability gate."
