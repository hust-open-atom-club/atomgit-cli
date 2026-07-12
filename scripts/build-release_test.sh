#!/bin/sh
# build-release.sh 测试套件
# 验证：validate_tag 严格性、date 可移植性、verify_injection 精确性、
#       无效 TAG 在 dist/ 创建前失败、AG_VERIFY_ONLY 不产生 dist/。
set -eu

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
BUILD_SCRIPT="$SCRIPT_DIR/build-release.sh"

# 提取 validate_tag 函数用于单元测试
validate_tag() {
  tag=$1
  echo "$tag" | grep -Eq \
    '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$' || return 1
  case "$tag" in
    *-*) prerelease=${tag#*-}; prerelease=${prerelease%%+*} ;;
    *) return 0 ;;
  esac
  old_ifs=$IFS
  IFS=.
  set -- $prerelease
  IFS=$old_ifs
  for identifier do
    case "$identifier" in
      0) ;;
      0[0-9]*) return 1 ;;
    esac
  done
}

passed=0
failed=0

pass() { passed=$((passed + 1)); }
fail() {
  echo "  失败: $*" >&2
  failed=$((failed + 1))
}

echo "=== 1. validate_tag 有效标签 ==="

valid_tags="
v0.5.0
v1.0.0
v0.0.1
v10.20.30
v1.2.3-beta.1
v1.2.3-beta
v1.2.3+build
v1.2.3+build.123
v1.2.3-beta.1+build.123
v1.2.3-rc.1.2.3+build.1.2.3
v0.5.0-alpha
v0.5.0-alpha.1
v1.2.3-0abc-def
"

for tag in $valid_tags; do
  if validate_tag "$tag"; then
    echo "  PASS: $tag"
    pass
  else
    fail "validate_tag 拒绝有效标签: $tag"
  fi
done

echo ""
echo "=== 2. validate_tag 无效标签 ==="

invalid_tags="
v1
v1.2
1.2.3
v1.2.3.
v1.2.3foo
v1.2.3.4
vv1.2.3
not-a-version
v01.02.03
v1.2.3-
v1.2.3+
v1.2.3-beta..1
v1.2.3-beta.
v1.2.3-01
v1.2.3-alpha.01
dev
"

for tag in $invalid_tags; do
  if validate_tag "$tag"; then
    fail "validate_tag 未拒绝无效标签: $tag"
  else
    echo "  PASS (拒绝): $tag"
    pass
  fi
done

echo ""
echo "=== 3. 无效 TAG 在 dist/ 创建前失败 ==="

# 确保没有遗留 dist/
rm -rf "$ROOT/dist"

# 用无效 TAG 运行构建脚本，应失败且不创建 dist/
if TAG="not-a-version" sh "$BUILD_SCRIPT" 2>/dev/null; then
  fail "无效 TAG 应导致脚本退出 1"
else
  if [ -d "$ROOT/dist" ]; then
    fail "无效 TAG 不应创建 dist/ 目录"
  else
    echo "  PASS: 无效 TAG 退出非零且未创建 dist/"
    pass
  fi
fi

rm -rf "$ROOT/dist"

echo ""
echo "=== 4. AG_VERIFY_ONLY 不创建 dist/ ==="

rm -rf "$ROOT/dist"
TAG="v0.5.0" AG_VERIFY_ONLY=1 sh "$BUILD_SCRIPT" >/dev/null 2>&1
if [ -d "$ROOT/dist" ]; then
  fail "AG_VERIFY_ONLY=1 不应创建 dist/"
else
  echo "  PASS: AG_VERIFY_ONLY=1 未创建 dist/"
  pass
fi
rm -rf "$ROOT/dist"

echo ""
echo "=== 5. format_epoch_utc 可移植性（冒烟） ==="

# 提取 format_epoch_utc 并测试
format_epoch_utc() {
  epoch="$1"
  out=$(date -u -d "@${epoch}" "+%Y-%m-%dT%H:%M:%SZ" 2>/dev/null) && echo "$out" && return 0
  out=$(date -u -r "${epoch}" "+%Y-%m-%dT%H:%M:%SZ" 2>/dev/null) && echo "$out" && return 0
  echo "错误: 无法格式化时间戳 ${epoch}。" >&2
  return 1
}

test_epoch=1783855619
result=$(format_epoch_utc "$test_epoch") || fail "format_epoch_utc ${test_epoch} 失败"
if [ -n "$result" ]; then
  echo "  PASS: format_epoch_utc($test_epoch) = $result"
  pass
fi

# 如果 SOURCE_DATE_EPOCH 在环境中已设置，验证其被正确使用
if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
  epoch="$SOURCE_DATE_EPOCH"
  result=$(format_epoch_utc "$epoch") || fail "format_epoch_utc SOURCE_DATE_EPOCH=$epoch 失败"
  echo "  PASS: format_epoch_utc(SOURCE_DATE_EPOCH=$epoch) = $result"
  pass
else
  echo "  SKIP: SOURCE_DATE_EPOCH 未设置，跳过环境验证"
fi

echo ""
echo "=== 6. verify_injection 精确值验证 ==="

# 完整端到端：构建并验证精确注入值
rm -rf "$ROOT/dist"
TAG="v0.5.0" AG_VERIFY_ONLY=1 sh "$BUILD_SCRIPT" 2>&1
echo "  PASS: AG_VERIFY_ONLY TAG=v0.5.0 精确校验通过"
pass
rm -rf "$ROOT/dist"

# 用预发布标签验证
rm -rf "$ROOT/dist"
TAG="v1.2.3-beta.1+build.42" AG_VERIFY_ONLY=1 sh "$BUILD_SCRIPT" 2>&1
echo "  PASS: AG_VERIFY_ONLY TAG=v1.2.3-beta.1+build.42 精确校验通过"
pass
rm -rf "$ROOT/dist"

echo ""
echo "========================================="
echo "结果: $passed 通过, $failed 失败"
echo "========================================="

if [ "$failed" -gt 0 ]; then
  exit 1
fi
