#!/bin/sh
# 使用 GoReleaser 生成预编译包到 dist/<版本>/：
#   - Linux/macOS: ag_<os>_<arch>.tar.gz（包内可执行文件名为 ag）
#   - Windows:     ag_windows_<arch>.zip（包内为 ag.exe）
#   - SHA-256:     checksums.txt（覆盖六个归档和两个安装脚本）
#
# 用法:
#   ./scripts/build-release.sh              # 版本来自 git describe，或环境变量 TAG
#   TAG=v0.1.0 ./scripts/build-release.sh
#
# 环境变量:
#   TAG               版本标签或 git describe 版本 (如 v0.5.0、v0.5.0-2-gabc1234)
#   SOURCE_DATE_EPOCH 用于可复现构建的 Unix 时间戳
#   AG_VERIFY_ONLY=1  仅构建校验一个二进制，不产生发布归档
#   AG_RELEASE_SNAPSHOT=1 允许未打 tag 或脏工作区的本地试打包，不得用于正式发布
#   AG_RELEASE_OUT    发布输出根目录，默认 dist
#   GORELEASER        GoReleaser 可执行文件，默认 goreleaser

set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"
GORELEASER="${GORELEASER:-goreleaser}"

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

# 正式发布统一使用三段式 SemVer；历史两段式 tag 仅保留给
# git describe 和本地 snapshot 兼容。
validate_release_tag() {
  value=$1
  echo "$value" | grep -Eq \
    '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$' || return 1
  validate_version "$value"
}

if ! validate_version "$TAG"; then
  echo "错误: TAG 值 \"$TAG\" 不是有效的发布或 git describe 版本" >&2
  echo "请使用 v0.5.0 或 v0.5.0-2-gabc1234 等格式。" >&2
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
  BUILD_TIMESTAMP="$SOURCE_DATE_EPOCH"
else
  BUILD_TIMESTAMP=$(git log -1 --format=%ct)
fi
BUILD_DATE=$(format_epoch_utc "$BUILD_TIMESTAMP") || exit 1

# 共享 linker 标志，仅用于 AG_VERIFY_ONLY 的快速注入校验。
LINK_FLAGS="-s -w"
LINK_FLAGS="${LINK_FLAGS} -X 'atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Version=${TAG}'"
LINK_FLAGS="${LINK_FLAGS} -X 'atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Commit=${COMMIT}'"
LINK_FLAGS="${LINK_FLAGS} -X 'atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.BuildDate=${BUILD_DATE}'"

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

  flag_out=$("$tmpbin" --version 2>&1)
  echo "    --version 输出: $flag_out"
  if [ "$flag_out" != "$out" ]; then
    echo "错误: --version 输出与 version 子命令不一致" >&2
    echo "version 输出: $out" >&2
    echo "--version 输出: $flag_out" >&2
    exit 1
  fi

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

# 以下仅在完整发布构建时执行。GoReleaser 使用固定的
# dist/.goreleaser 作为临时目录，完成后再将可发布制品复制到版本目录。
if ! command -v "$GORELEASER" >/dev/null 2>&1; then
  echo "错误: 未找到 GoReleaser（命令: $GORELEASER）。" >&2
  echo "请先安装 GoReleaser：https://goreleaser.com/install/" >&2
  exit 1
fi

RELEASE_MODE=release
if [ "${AG_RELEASE_SNAPSHOT:-}" = "1" ]; then
  RELEASE_MODE=snapshot
else
  if ! validate_release_tag "$TAG"; then
    echo "错误: 正式发布标签必须使用 vX.Y.Z 格式（例如 v0.5.0）。" >&2
    echo "历史两段式 tag 仅可通过 AG_RELEASE_SNAPSHOT=1 进行本地试打包。" >&2
    exit 1
  fi

  # 正式发布制品必须可追溯到唯一、已提交的 tag。这些检查要在
  # 删除或创建任何输出目录之前完成。
  if [ -n "$(git status --porcelain --untracked-files=normal)" ]; then
    echo "错误: 正式发布要求 Git 工作区干净。" >&2
    echo "请先提交或移除未提交改动；仅本地试打包可设置 AG_RELEASE_SNAPSHOT=1。" >&2
    exit 1
  fi

  if ! TAG_COMMIT=$(git rev-parse -q --verify "refs/tags/${TAG}^{commit}"); then
    echo "错误: 正式发布标签 ${TAG} 不存在。" >&2
    echo "请先在当前提交创建该 tag，或使用 AG_RELEASE_SNAPSHOT=1 试打包。" >&2
    exit 1
  fi
  if [ "$TAG_COMMIT" != "$COMMIT" ]; then
    echo "错误: 标签 ${TAG} 未指向当前 HEAD。" >&2
    echo "tag 提交: $TAG_COMMIT" >&2
    echo "当前 HEAD: $COMMIT" >&2
    exit 1
  fi
fi

OUT_ROOT="${AG_RELEASE_OUT:-${ROOT}/dist}"
OUT="${OUT_ROOT}/${TAG}"
STAGING="${ROOT}/dist/.goreleaser"
rm -rf "$OUT"
mkdir -p "$OUT"

echo "输出目录: $OUT"
echo "版本标识: $TAG"
echo "提交:      $COMMIT"
echo "构建日期:  $BUILD_DATE"
echo "构建模式:  $RELEASE_MODE"
echo ""

echo "==> 校验 GoReleaser 配置 ..."
"$GORELEASER" check

if [ "$RELEASE_MODE" = "snapshot" ]; then
  # 快照模式只在本地打包。AG_VERSION 保留项目已有的 v0.5 等
  # 标签格式；快照内部版本去掉 v 前缀。
  AG_VERSION="$TAG" \
  AG_BUILD_DATE="$BUILD_DATE" \
  AG_BUILD_TIMESTAMP="$BUILD_TIMESTAMP" \
  AG_SNAPSHOT_VERSION="${TAG#v}" \
    "$GORELEASER" release --snapshot --clean
else
  # AtomGit 附件仍由后续流程上传；这里使用正常 release 流程执行
  # Git/tag 校验，并显式跳过 GoReleaser 的托管平台发布阶段。
  AG_VERSION="$TAG" \
  AG_BUILD_DATE="$BUILD_DATE" \
  AG_BUILD_TIMESTAMP="$BUILD_TIMESTAMP" \
  GORELEASER_CURRENT_TAG="$TAG" \
    "$GORELEASER" release --clean --skip=publish
fi

cp "$STAGING"/ag_linux_amd64.tar.gz "$OUT/"
cp "$STAGING"/ag_linux_arm64.tar.gz "$OUT/"
cp "$STAGING"/ag_darwin_amd64.tar.gz "$OUT/"
cp "$STAGING"/ag_darwin_arm64.tar.gz "$OUT/"
cp "$STAGING"/ag_windows_amd64.zip "$OUT/"
cp "$STAGING"/ag_windows_arm64.zip "$OUT/"
cp "$STAGING"/checksums.txt "$OUT/"
rm -rf "$STAGING"

# 生成与本次 TAG 一致的 install.sh / install.ps1，包括默认版本和文件头用法示例。
ESC_TAG=$(printf '%s\n' "$TAG" | sed 's/[\/&]/\\&/g')
sed \
  -e "s@/releases/download/v[^/]*/install.sh@/releases/download/${ESC_TAG}/install.sh@" \
  -e "s/^#   AG_VERSION=v[^ ]* sh install.sh$/#   AG_VERSION=${ESC_TAG} sh install.sh/" \
  -e "s/^_BUNDLED_TAG=.*/_BUNDLED_TAG=\"${ESC_TAG}\"/" \
  "$ROOT/install.sh" > "${OUT}/install.sh"
chmod +x "${OUT}/install.sh"
echo "已生成 ${OUT}/install.sh（默认 TAG=${TAG}）"
sed \
  -e "s@/releases/download/v[^/]*/install.ps1@/releases/download/${ESC_TAG}/install.ps1@" \
  -e "s@^#   \$env:AG_VERSION = .*@#   \$env:AG_VERSION = \"${ESC_TAG}\"; .\\\\install.ps1@" \
  -e "s/^\$BundledTag = '.*'/\$BundledTag = '${ESC_TAG}'/" \
  "$ROOT/install.ps1" > "${OUT}/install.ps1"
echo "已生成 ${OUT}/install.ps1（默认 TAG=${TAG}）"

# GoReleaser 的校验和只包含它生成的归档。安装脚本由本包装脚本
# 在打包后生成，因此将它们的 SHA-256 追加到最终 checksums.txt。
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$OUT" && sha256sum install.sh install.ps1) >> "${OUT}/checksums.txt"
elif command -v shasum >/dev/null 2>&1; then
  (cd "$OUT" && shasum -a 256 install.sh install.ps1) >> "${OUT}/checksums.txt"
else
  echo "错误: 生成安装脚本校验和需要 sha256sum 或 shasum。" >&2
  exit 1
fi
echo "已将 install.sh 和 install.ps1 加入 ${OUT}/checksums.txt"
echo ""

if [ "$RELEASE_MODE" = "snapshot" ]; then
  echo "试打包完成。制品位于 ${OUT}/，请勿将未校验的快照制品用于正式发布。"
else
  echo "完成。将 ${OUT}/ 下各 .tar.gz / .zip、checksums.txt、install.sh 与 install.ps1 作为 AtomGit Release「${TAG}」的附件上传即可。"
fi
echo "（Windows 也可：PowerShell 执行 install.ps1，或下载 ag_windows_*.zip 手动解压并加入 PATH。）"
