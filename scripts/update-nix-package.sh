#!/bin/sh
# 更新 flake.nix 中 stable 或 latest package 的固定输出哈希。
#
# 用法:
#   ./scripts/update-nix-package.sh vX.Y.Z
#   ./scripts/update-nix-package.sh --latest
#
# 默认模式更新 stable 的版本、发布源码哈希和 vendorHash。
# --latest 只更新当前源码对应的 latestVendorHash。
# 验证失败时 EXIT trap 自动恢复原始 flake.nix。
# 不会自动提交、打标签或推送。

set -eu

FLAKE=""
ROLLBACK=0
BACKUP=""
TMPDIR=""
BIN_TMP=""
RESULT_LINK=""
BUILD_PID=""

cleanup() {
    if [ -n "${BUILD_PID:-}" ]; then
        kill "$BUILD_PID" 2>/dev/null || true
        wait "$BUILD_PID" 2>/dev/null || true
    fi
    if [ "$ROLLBACK" = "1" ] && [ -n "${BACKUP:-}" ] && [ -f "$BACKUP" ]; then
        printf '%s\n' "==> 回滚: 恢复 ${FLAKE}" >&2
        cp "$BACKUP" "$FLAKE"
    fi
    rm -f "${BACKUP:-}"
    rm -rf "${TMPDIR:-}" "${BIN_TMP:-}"
    if [ -n "${RESULT_LINK:-}" ] && [ -L "$RESULT_LINK" ]; then
        rm -f "$RESULT_LINK"
    fi
}
trap cleanup EXIT

die() {
    printf '%s\n' "错误: $*" >&2
    exit 1
}

usage() {
    cat >&2 <<EOF
用法:
  $0 <version>   更新 stable，例如 $0 vX.Y.Z
  $0 --latest    只更新 latestVendorHash
EOF
}

validate_version() {
    printf '%s' "$1" | grep -qE \
        '^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'
}

count_assignments() {
    awk -v name="$1" '
    $0 ~ "^[[:space:]]*" name " = \"" { count++ }
    END { print count + 0 }
  ' "$2"
}

read_assignment() {
    sed -n "s/^[[:space:]]*${1} = \"\\([^\"]*\\)\";/\\1/p" "$2"
}

replace_assignment() {
    name="$1"
    value="$2"
    file="$3"
    escaped=$(printf '%s' "$value" | sed 's/[\\&@]/\\&/g')
    sed "s@^\\([[:space:]]*${name} = \\)\"[^\"]*\";@\\1\"${escaped}\";@" \
        "$file" >"${file}.tmp"
    mv "${file}.tmp" "$file"
}

replace_with_fake_hash() {
    name="$1"
    file="$2"
    sed "s@^\\([[:space:]]*${name} = \\)\"[^\"]*\";@\\1pkgs.lib.fakeHash;@" \
        "$file" >"${file}.tmp"
    mv "${file}.tmp" "$file"
}

# Include tracked files plus non-ignored untracked files. Each step is checked
# separately because POSIX sh has no portable pipefail option.
copy_src() {
    source_dir="$1"
    destination="$2"
    file_list="${destination}/.source-files"
    archive="${destination}/.source.tar"

    (cd "$source_dir" && git ls-files --cached --others --exclude-standard -z >"$file_list") ||
        return 1
    (cd "$source_dir" && tar --null -T "$file_list" -cf "$archive") ||
        return 1
    rm -f "$file_list"
    tar -xf "$archive" -C "$destination" || return 1
    rm -f "$archive"
}

parse_vendor_hash() {
    log="$1"
    count=$(grep -c 'got: *sha256-' "$log" 2>/dev/null || true)
    [ "$count" -eq 1 ] || die "expected exactly 1 'got:' hash in nix output, found $count"
    grep 'got: *sha256-' "$log" |
        sed 's/.*got: *\(sha256-[A-Za-z0-9+/=]\{43,\}\).*/\1/' |
        tr -d '\n'
}

parse_prefetch_hash() {
    printf '%s' "$1" |
        sed -n 's/.*"hash"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
}

calculate_vendor_hash() {
    package="$1"
    hash_name="$2"
    current_hash="$3"
    flake_dir="$4"
    build_log="${flake_dir}/nix-build-${package}.log"

    printf '%s\n' "==> 验证 ${package} 的现有 vendorHash..." >&2
    if nix build --offline --no-link "path:${flake_dir}#${package}" 1>/dev/null 2>&1; then
        printf '%s\n' "==> Go 依赖未变化，复用 ${hash_name}: ${current_hash}" >&2
        VENDOR_HASH="$current_hash"
        return
    fi

    replace_with_fake_hash "$hash_name" "${flake_dir}/flake.nix"
    printf '%s\n' "==> 计算 ${hash_name}..." >&2
    nix build --no-link "path:${flake_dir}#${package}" >"$build_log" 2>&1 &
    BUILD_PID=$!
    while kill -0 "$BUILD_PID" 2>/dev/null; do
        printf '.' >&2
        sleep 2
    done
    printf '\n' >&2
    wait "$BUILD_PID" || true
    BUILD_PID=""

    VENDOR_HASH=$(parse_vendor_hash "$build_log")
    printf '%s' "$VENDOR_HASH" | grep -qE '^sha256-[A-Za-z0-9+/=]{43,}$' ||
        die "解析出的 ${hash_name} 格式无效。原始 flake.nix 未修改。"
    printf '%s\n' "==> ${hash_name}: ${VENDOR_HASH}" >&2
}

verify_package() {
    package="$1"
    expected_version="$2"

    printf '%s\n' "==> 验证 nix build .#${package} ..."
    if ! nix build --no-link "path:${REPO_ROOT}#${package}" 1>/dev/null 2>&1; then
        die "nix build .#${package} 验证失败"
    fi

    printf '%s\n' "==> 验证 ${package} 的 ag version --json ..."
    BIN_TMP=$(mktemp -d) || die "无法创建二进制验证临时目录"
    [ -n "$BIN_TMP" ] && [ -d "$BIN_TMP" ] || die "二进制验证临时目录无效"
    if ! nix build --out-link "${BIN_TMP}/result" "path:${REPO_ROOT}#${package}" 1>/dev/null 2>&1; then
        die "无法构建 ${package} 验证用二进制"
    fi
    RESULT_LINK="${BIN_TMP}/result"

    AG_BIN="${BIN_TMP}/result/bin/ag"
    [ -x "$AG_BIN" ] || AG_BIN=$(find "${BIN_TMP}/result" -name ag -type f -perm -100 2>/dev/null | head -1)
    [ -n "${AG_BIN:-}" ] && [ -x "$AG_BIN" ] || die "找不到 ag 二进制"

    JSON_OUT=$("$AG_BIN" version --json 2>&1) || die "ag version --json 执行失败: $JSON_OUT"
    json_version=$(printf '%s' "$JSON_OUT" | sed -n 's/.*"version": *"\([^"]*\)".*/\1/p')
    json_commit=$(printf '%s' "$JSON_OUT" | sed -n 's/.*"commit": *"\([^"]*\)".*/\1/p')
    json_build_date=$(printf '%s' "$JSON_OUT" | sed -n 's/.*"buildDate": *"\([^"]*\)".*/\1/p')

    if [ -n "$expected_version" ] && [ "$json_version" != "$expected_version" ]; then
        die "版本不匹配: 期望 ${expected_version} 实际 ${json_version}"
    fi
    printf '%s\n' "  version=${json_version} commit=${json_commit} buildDate=${json_build_date}"

    rm -f "$RESULT_LINK"
    RESULT_LINK=""
    rm -rf "$BIN_TMP"
    BIN_TMP=""
}

if [ $# -ne 1 ] || [ -z "$1" ]; then
    usage
    exit 1
fi

MODE="stable"
if [ "$1" = "--latest" ]; then
    MODE="latest"
else
    validate_version "$1" ||
        die "无效版本 \"$1\"（要求 v?MAJOR.MINOR.PATCH[-(prerelease)][+(build)]）"
fi

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
FLAKE="${REPO_ROOT}/flake.nix"
[ -f "$FLAKE" ] || die "flake.nix not found: $FLAKE"

for name in stableVersion stableCommit stableBuildDate stableSrcHash stableVendorHash latestVendorHash; do
    count=$(count_assignments "$name" "$FLAKE")
    [ "$count" -eq 1 ] || die "flake.nix 应有恰好 1 处 ${name} 字符串赋值，实际 ${count} 处"
done

TMPDIR=$(mktemp -d) || die "无法创建临时工作目录"
[ -n "$TMPDIR" ] && [ -d "$TMPDIR" ] || die "临时工作目录无效"
copy_src "$REPO_ROOT" "$TMPDIR" || die "复制源码到临时工作目录失败"
TMP_FLAKE="${TMPDIR}/flake.nix"
[ -f "$TMP_FLAKE" ] || die "临时 flake 不存在"

if [ "$MODE" = "stable" ]; then
    NIX_VERSION=$(printf '%s' "$1" | sed 's/^v//')
    TAG="v${NIX_VERSION}"
    git -C "$REPO_ROOT" rev-parse --verify "${TAG}^{commit}" 1>/dev/null 2>&1 ||
        die "本地不存在 tag ${TAG}"

    STABLE_COMMIT=$(git -C "$REPO_ROOT" rev-list -n 1 "$TAG")
    STABLE_BUILD_DATE=$(git -C "$REPO_ROOT" show -s --format=%cI "$TAG")
    SOURCE_URL="https://raw.atomgit.com/hust-open-atom-club/atomgit-cli/archive/refs/heads/${TAG}.tar.gz"

    printf '%s\n' "==> stable ${TAG}: 预取 Release 源码..."
    PREFETCH_JSON=$(nix store prefetch-file --json "$SOURCE_URL")
    STABLE_SRC_HASH=$(parse_prefetch_hash "$PREFETCH_JSON")
    printf '%s' "$STABLE_SRC_HASH" | grep -qE '^sha256-[A-Za-z0-9+/=]{43,}$' ||
        die "无法从 nix store prefetch-file 输出解析 stableSrcHash"
    printf '%s\n' "==> stableSrcHash: ${STABLE_SRC_HASH}"

    replace_assignment stableVersion "$NIX_VERSION" "$TMP_FLAKE"
    replace_assignment stableCommit "$STABLE_COMMIT" "$TMP_FLAKE"
    replace_assignment stableBuildDate "$STABLE_BUILD_DATE" "$TMP_FLAKE"
    replace_assignment stableSrcHash "$STABLE_SRC_HASH" "$TMP_FLAKE"

    CURRENT_HASH=$(read_assignment stableVendorHash "$FLAKE")
    calculate_vendor_hash stable stableVendorHash "$CURRENT_HASH" "$TMPDIR"

    BACKUP="${FLAKE}.bak.$$"
    cp "$FLAKE" "$BACKUP"
    ROLLBACK=1

    replace_assignment stableVersion "$NIX_VERSION" "$FLAKE"
    replace_assignment stableCommit "$STABLE_COMMIT" "$FLAKE"
    replace_assignment stableBuildDate "$STABLE_BUILD_DATE" "$FLAKE"
    replace_assignment stableSrcHash "$STABLE_SRC_HASH" "$FLAKE"
    replace_assignment stableVendorHash "$VENDOR_HASH" "$FLAKE"

    verify_package stable "$TAG"
    printf '%s\n' "==> stable 完成: ${TAG}"
    printf '%s\n' "    srcHash=${STABLE_SRC_HASH}"
    printf '%s\n' "    vendorHash=${VENDOR_HASH}"
else
    CURRENT_HASH=$(read_assignment latestVendorHash "$FLAKE")
    calculate_vendor_hash latest latestVendorHash "$CURRENT_HASH" "$TMPDIR"

    BACKUP="${FLAKE}.bak.$$"
    cp "$FLAKE" "$BACKUP"
    ROLLBACK=1

    replace_assignment latestVendorHash "$VENDOR_HASH" "$FLAKE"
    verify_package latest ""
    printf '%s\n' "==> latest 完成: vendorHash=${VENDOR_HASH}"
fi

ROLLBACK=0
rm -f "$BACKUP"
BACKUP=""

printf '%s\n' "==> flake.nix 已更新。未自动提交、打标签或推送。"
