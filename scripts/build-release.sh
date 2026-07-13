#!/bin/sh
# 生成预编译包到 dist/<版本>/：
#   - Linux/macOS: ag_<os>_<arch>.tar.gz（包内可执行文件名为 ag）
#   - Windows:     ag_windows_<arch>.zip（包内为 ag.exe）
#
# 用法:
#   ./scripts/build-release.sh              # 版本来自 git describe，或环境变量 TAG
#   TAG=v0.1.0 ./scripts/build-release.sh
#
# 环境变量:
#   TAG               版本标签或 git describe 版本 (如 v0.5、v0.5.0、v0.5-2-gabc1234)
#   SOURCE_DATE_EPOCH 用于可复现构建的 Unix 时间戳
#   AG_VERIFY_ONLY=1  仅构建校验一个二进制，不产生发布归档
#   AG_RELEASE_OUT    发布输出根目录，默认 dist

set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

# ---------------------------------------------------------------------------
# 标签解析与校验
# ---------------------------------------------------------------------------
TAG="${TAG:-}"
if [ -z "$TAG" ]; then
  TAG=$(git describe --tags --always --dirty 2>/dev/null) || TAG="dev"
fi

validate_version() {
  value=$1
  echo "$value" | grep -Eq '^[0-9a-f]+(-dirty)?$' && return 0
  echo "$value" | grep -Eq \
    '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*))?-[0-9]+-g[0-9a-f]+(-dirty)?$' && return 0
  echo "$value" | grep -Eq \
    '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*))?(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$' || return 1

  core=${value%%+*}
  case "$core" in
    *-*) prerelease=${core#*-} ;;
    *) return 0 ;;
  esac
  echo "$prerelease" | grep -Eq '(^|\.)0[0-9]+($|\.)' && return 1
  return 0
}

if ! validate_version "$TAG"; then
  echo "错误: TAG 值 \"$TAG\" 不是有效的发布或 git describe 版本" >&2
  echo "请使用 v0.5、v0.5.0 或 v0.5-2-gabc1234 等格式。" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# 构建元数据
# ---------------------------------------------------------------------------
COMMIT=$(git rev-parse HEAD)

# format_epoch_utc: 将 Unix 时间戳转换为 ISO 8601 UTC 字符串。
# 兼容 GNU date (-d @epoch) 和 BSD/macOS date (-r epoch)。
format_epoch_utc() {
  epoch="$1"
  out=$(date -u -d "@${epoch}" "+%Y-%m-%dT%H:%M:%SZ" 2>/dev/null) && echo "$out" && return 0
  out=$(date -u -r "${epoch}" "+%Y-%m-%dT%H:%M:%SZ" 2>/dev/null) && echo "$out" && return 0
  echo "错误: 无法格式化时间戳 ${epoch}。需要 GNU date 或 BSD date。" >&2
  return 1
}

if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
  BUILD_DATE=$(format_epoch_utc "$SOURCE_DATE_EPOCH") || exit 1
else
  COMMIT_TS=$(git log -1 --format=%ct)
  BUILD_DATE=$(format_epoch_utc "$COMMIT_TS") || exit 1
fi

# 共享 linker 标志（仅 linker 标志，不含 go build 选项）
LINK_FLAGS="-s -w"
LINK_FLAGS="${LINK_FLAGS} -X 'atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Version=${TAG}'"
LINK_FLAGS="${LINK_FLAGS} -X 'atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Commit=${COMMIT}'"
LINK_FLAGS="${LINK_FLAGS} -X 'atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.BuildDate=${BUILD_DATE}'"

# ---------------------------------------------------------------------------
# 发布构建函数
# ---------------------------------------------------------------------------
build_one() {
  goos=$1
  goarch=$2
  asset="ag_${goos}_${goarch}.tar.gz"
  tmpdir=$(mktemp -d)

  echo "构建 $goos/$goarch -> $asset"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -trimpath -ldflags="${LINK_FLAGS}" -o "${tmpdir}/ag" ./cmd/ag
  (cd "$tmpdir" && tar -czf "${OUT}/${asset}" ag)
  rm -rf "$tmpdir"
  ls -lh "${OUT}/${asset}"
  echo ""
}

build_windows() {
  goarch=$1
  asset="ag_windows_${goarch}.zip"
  tmpdir=$(mktemp -d)

  echo "构建 windows/$goarch -> $asset"
  GOOS=windows GOARCH="$goarch" CGO_ENABLED=0 \
    go build -trimpath -ldflags="${LINK_FLAGS}" -o "${tmpdir}/ag.exe" ./cmd/ag
  (cd "$tmpdir" && zip -q "${OUT}/${asset}" ag.exe)
  rm -rf "$tmpdir"
  ls -lh "${OUT}/${asset}"
  echo ""
}

# ---------------------------------------------------------------------------
# 注入校验（可单独调用）
# ---------------------------------------------------------------------------
verify_injection() {
  tmpbin=$(mktemp)
  trap 'rm -f "$tmpbin"' EXIT

  verify_os=$(go env GOHOSTOS)
  verify_arch=$(go env GOHOSTARCH)
  echo "==> 校验版本注入: 构建 ${verify_os}/${verify_arch} 测试二进制 ..."
  GOOS="$verify_os" GOARCH="$verify_arch" CGO_ENABLED=0 \
    go build -trimpath -ldflags="${LINK_FLAGS}" -o "$tmpbin" ./cmd/ag

  # 文本输出 — 至少包含 TAG
  out=$("$tmpbin" version 2>&1)
  echo "    文本输出: $out"
  case "$out" in
    *"$TAG"*) ;;
    *)
      echo "错误: 二进制未报告预期的 TAG=\"$TAG\"" >&2
      echo "实际输出: $out" >&2
      exit 1
      ;;
  esac

  # JSON 输出 — 精确比对三个字段
  json_out=$("$tmpbin" version --json 2>&1)
  echo "    JSON 输出: $json_out"

  json_version=$(echo "$json_out" | sed -n 's/.*"version": *"\([^"]*\)".*/\1/p')
  json_commit=$(echo "$json_out"  | sed -n 's/.*"commit": *"\([^"]*\)".*/\1/p')
  json_build_date=$(echo "$json_out" | sed -n 's/.*"buildDate": *"\([^"]*\)".*/\1/p')

  if [ "$json_version" != "$TAG" ]; then
    echo "错误: JSON version=\"$json_version\", 期望 \"$TAG\"" >&2
    exit 1
  fi
  if [ "$json_commit" != "$COMMIT" ]; then
    echo "错误: JSON commit=\"$json_commit\", 期望 \"$COMMIT\"" >&2
    exit 1
  fi
  if [ "$json_build_date" != "$BUILD_DATE" ]; then
    echo "错误: JSON buildDate=\"$json_build_date\", 期望 \"$BUILD_DATE\"" >&2
    exit 1
  fi

  echo "==> 注入校验通过 (tag=$TAG, commit=$COMMIT, buildDate=$BUILD_DATE)"
  echo ""
  rm -f "$tmpbin"
  trap - EXIT
}

# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------

# 测试/校验模式：TAG 已在前面校验，现在仅构建并检查一个二进制，不创建 dist/。
if [ "${AG_VERIFY_ONLY:-}" = "1" ]; then
  verify_injection
  echo "校验完成。未创建发布归档。"
  exit 0
fi

# 以下仅在完整发布构建时执行
OUT_ROOT="${AG_RELEASE_OUT:-${ROOT}/dist}"
OUT="${OUT_ROOT}/${TAG}"
mkdir -p "$OUT"

echo "输出目录: $OUT"
echo "版本标识: $TAG"
echo "提交:      $COMMIT"
echo "构建日期:  $BUILD_DATE"
echo ""

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64

build_windows amd64
build_windows arm64

# 生成与本次 TAG 默认一致的 install.sh / install.ps1（避免 Release 附件里脚本仍指向旧版本）
ESC_TAG=$(printf '%s\n' "$TAG" | sed 's/[\/&]/\\&/g')
sed "s/^_BUNDLED_TAG=.*/_BUNDLED_TAG=\"${ESC_TAG}\"/" "$ROOT/install.sh" > "${OUT}/install.sh"
chmod +x "${OUT}/install.sh"
echo "已生成 ${OUT}/install.sh（默认 TAG=${TAG}）"
sed "s/^\$BundledTag = '.*'/\$BundledTag = '${ESC_TAG}'/" "$ROOT/install.ps1" > "${OUT}/install.ps1"
echo "已生成 ${OUT}/install.ps1（默认 TAG=${TAG}）"
echo ""

echo "完成。将 dist/${TAG}/ 下各 .tar.gz / .zip、install.sh 与 install.ps1 作为 AtomGit Release「${TAG}」的附件上传即可。"
echo "（Windows 也可：PowerShell 执行 install.ps1，或下载 ag_windows_*.zip 手动解压并加入 PATH。）"
