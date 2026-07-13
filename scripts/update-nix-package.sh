#!/bin/sh
# 更新 flake.nix 中的版本与 vendorHash。
# 用法: ./scripts/update-nix-package.sh v0.6.0
#
# 原始 flake.nix 从不会包含 lib.fakeHash。
# 验证失败时 EXIT trap 自动恢复原始备份。
# 不会自动提交/打标签/推送。

set -eu

# ---- 事务回滚状态（EXIT trap 引用） ----
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

# ---- 版本校验: v?MAJOR.MINOR.PATCH[-(prerelease)][+(build)] ----
validate_version() {
  printf '%s' "$1" | grep -qE \
    '^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'
}

# ---- 统计 flake 中某类赋值行数（只匹配行首空白 + name = " 模式） ----
count_assignments() {
  grep -c "^[[:space:]]*${1} = \"" "$2" 2>/dev/null || printf '0'
}

# ---- 复制 Git 已跟踪源码，与 Nix 的本地 Git flake 输入语义保持一致 ----
copy_src() {
  (cd "$1" && git ls-files --cached -z \
    | tar --null --verbatim-files-from -T - -cf - \
    | tar -xf - -C "$2")
}

# ---- 从 nix 错误输出中解析 SRI sha256 hash（只匹配 "got:" 行，要求恰好一条） ----
parse_vendor_hash() {
  log="$1"
  count=$(grep -c 'got: *sha256-' "$log" 2>/dev/null || true)
  [ "$count" -eq 1 ] || die "expected exactly 1 'got:' hash in nix output, found $count"
  grep 'got: *sha256-' "$log" | sed 's/.*got: *\(sha256-[A-Za-z0-9+/=]\{43,\}\).*/\1/' | tr -d '\n'
}

# ===================================================================
# 参数解析与校验
# ===================================================================
if [ $# -ne 1 ] || [ -z "$1" ]; then
  echo "用法: $0 <version>" >&2
  echo "示例: $0 v0.6.0" >&2
  exit 1
fi

INPUT_VERSION="$1"
validate_version "$INPUT_VERSION" || die "无效版本 \"$INPUT_VERSION\"（要求 v?MAJOR.MINOR.PATCH[-(prerelease)][+(build)]）"

NIX_VERSION=$(printf '%s' "$INPUT_VERSION" | sed 's/^v//')
INJECTED_VERSION="v${NIX_VERSION}"

# ---- 定位仓库根目录 ----
REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
FLAKE="${REPO_ROOT}/flake.nix"
[ -f "$FLAKE" ] || die "flake.nix not found: $FLAKE"

# ---- 校验原始 flake 恰好各有一处赋值 ----
VER_COUNT=$(count_assignments "version" "$FLAKE")
[ "$VER_COUNT" -eq 1 ] || die "flake.nix 应有恰好 1 处 version 赋值，实际 ${VER_COUNT} 处"
HASH_COUNT=$(count_assignments "vendorHash" "$FLAKE")
[ "$HASH_COUNT" -eq 1 ] || die "flake.nix 应有恰好 1 处 vendorHash 赋值，实际 ${HASH_COUNT} 处"

printf '%s\n' "==> ${REPO_ROOT} 版本 ${INPUT_VERSION} -> Nix ${NIX_VERSION}"

# ===================================================================
# 临时工作区：先复用现有 hash，仅在依赖变化时通过副本重新计算
# ===================================================================
TMPDIR=$(mktemp -d)
copy_src "$REPO_ROOT" "$TMPDIR"
TMP_FLAKE="${TMPDIR}/flake.nix"
[ -f "$TMP_FLAKE" ] || die "临时 flake 不存在"

# 校验复制的 flake 同样恰好各一处
TMP_VER_COUNT=$(count_assignments "version" "$TMP_FLAKE")
[ "$TMP_VER_COUNT" -eq 1 ] || die "临时 flake 应有恰好 1 处 version，实际 ${TMP_VER_COUNT} 处"
TMP_HASH_COUNT=$(count_assignments "vendorHash" "$TMP_FLAKE")
[ "$TMP_HASH_COUNT" -eq 1 ] || die "临时 flake 应有恰好 1 处 vendorHash，实际 ${TMP_HASH_COUNT} 处"

sed "s/^\([[:space:]]*version = \)\"[^\"]*\";/\1\"${NIX_VERSION}\";/" \
  "$TMP_FLAKE" > "${TMP_FLAKE}.tmp" && mv "${TMP_FLAKE}.tmp" "$TMP_FLAKE"

BUILD_LOG="${TMPDIR}/nix-build.log"
VENDOR_HASH=$(sed -n 's/^[[:space:]]*vendorHash = "\([^"]*\)";/\1/p' "$FLAKE")

printf '%s\n' "==> 在临时副本中验证现有 vendorHash..."
if (cd "$TMPDIR" && nix build --offline --no-link .#ag); then
  printf '%s\n' "==> Go 依赖未变化，复用现有 vendorHash: $VENDOR_HASH"
else
  sed "s/^\([[:space:]]*vendorHash = \)\"[^\"]*\";/\1pkgs.lib.fakeHash;/" \
    "$TMP_FLAKE" > "${TMP_FLAKE}.tmp" && mv "${TMP_FLAKE}.tmp" "$TMP_FLAKE"

  printf '%s\n' "==> Go 依赖已变化，正在临时副本中重新计算 vendorHash..."
  (cd "$TMPDIR" && nix build --no-link .#ag > "$BUILD_LOG" 2>&1) &
  BUILD_PID=$!
  while kill -0 "$BUILD_PID" 2>/dev/null; do
    printf '.'
    sleep 2
  done
  printf '\n'
  wait "$BUILD_PID" || true
  BUILD_PID=""

  VENDOR_HASH=$(parse_vendor_hash "$BUILD_LOG")
  printf '%s' "$VENDOR_HASH" | grep -qE '^sha256-[A-Za-z0-9+/=]{43,}$' \
    || die "解析出的 vendorHash 格式无效。原始 flake.nix 未修改。"
  printf '%s\n' "==> 真实 vendorHash: $VENDOR_HASH"
fi

# ===================================================================
# 原子更新原始 flake.nix（此后任何失败都由 EXIT trap 自动回滚）
# ===================================================================
BACKUP="${FLAKE}.bak.$$"
cp "$FLAKE" "$BACKUP"
ROLLBACK=1

TEMP_FLAKE="${FLAKE}.tmp.$$"
sed "s/^\([[:space:]]*version = \)\"[^\"]*\";/\1\"${NIX_VERSION}\";/" \
  "$FLAKE" > "$TEMP_FLAKE"
sed "s/^\([[:space:]]*vendorHash = \)\"[^\"]*\";/\1\"${VENDOR_HASH}\";/" \
  "$TEMP_FLAKE" > "${TEMP_FLAKE}.tmp" && mv "${TEMP_FLAKE}.tmp" "$TEMP_FLAKE"

mv "$TEMP_FLAKE" "$FLAKE"
printf '%s\n' "==> flake.nix 已原子更新"

# ===================================================================
# 验证 1: nix build
# ===================================================================
printf '%s\n' "==> 验证 nix build ..."
if ! (cd "$REPO_ROOT" && nix build --no-link .#ag 1>/dev/null 2>&1); then
  printf '%s\n' "nix build 验证失败，将自动回滚" >&2
  exit 1
fi

# ===================================================================
# 验证 2: ag version --json
# ===================================================================
printf '%s\n' "==> 验证 ag version --json ..."
BIN_TMP=$(mktemp -d)
if ! (cd "$REPO_ROOT" && nix build --out-link "${BIN_TMP}/result" .#ag 1>/dev/null 2>&1); then
  printf '%s\n' "无法构建验证用二进制，将自动回滚" >&2
  exit 1
fi
RESULT_LINK="${BIN_TMP}/result"

AG_BIN="${BIN_TMP}/result/bin/ag"
[ -x "$AG_BIN" ] || AG_BIN=$(find "${BIN_TMP}/result" -name ag -type f -perm -100 2>/dev/null | head -1)
[ -n "${AG_BIN:-}" ] && [ -x "$AG_BIN" ] || die "找不到 ag 二进制"

JSON_OUT=$("$AG_BIN" version --json 2>&1) || die "ag version --json 执行失败: $JSON_OUT"

json_version=$(printf '%s' "$JSON_OUT" | sed -n 's/.*"version": *"\([^"]*\)".*/\1/p')
json_commit=$(printf '%s' "$JSON_OUT"  | sed -n 's/.*"commit": *"\([^"]*\)".*/\1/p')
json_build_date=$(printf '%s' "$JSON_OUT" | sed -n 's/.*"buildDate": *"\([^"]*\)".*/\1/p')

if [ "$json_version" != "$INJECTED_VERSION" ]; then
  printf '%s\n' "版本不匹配: 期望 ${INJECTED_VERSION} 实际 ${json_version}，将自动回滚" >&2
  exit 1
fi
if [ -z "$json_commit" ] || [ "$json_commit" = "unknown" ]; then
  printf '%s\n' "警告: commit=${json_commit}" >&2
fi
if [ -z "$json_build_date" ] || [ "$json_build_date" = "unknown" ]; then
  printf '%s\n' "警告: buildDate=${json_build_date}" >&2
fi

printf '%s\n' "  version=${json_version} commit=${json_commit} buildDate=${json_build_date}"

# ===================================================================
# 全部验证通过
# ===================================================================
ROLLBACK=0
rm -f "$BACKUP"
BACKUP=""

printf '%s\n' ""
printf '%s\n' "==> 完成: ${NIX_VERSION}  vendorHash=${VENDOR_HASH}"
printf '%s\n' "==> flake.nix 已更新。未自动提交/打标签。"
