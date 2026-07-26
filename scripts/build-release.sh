#!/bin/sh
# 使用 GoReleaser 生成预编译包到 dist/<版本>/：
#   - Linux/macOS: ag_<os>_<arch>.tar.gz（包内可执行文件名为 ag）
#   - Windows:     ag_windows_<arch>.zip（包内为 ag.exe）
#   - 包管理器:    package-managers/ag_<os>_<arch>_<source>.<ext>
#   - npm:         六个平台二进制子包和一个主启动包，以及独立的 npm/checksums.txt
#   - SHA-256:     根 checksums.txt 仅覆盖六个普通归档和两个安装脚本；
#                  package-managers-checksums.txt 仅覆盖 12 个 managed 归档
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

# 正式 SemVer tag 需要与 npm 元数据保持一致，避免发布旧包版本或
# 让 npm 安装器下载错误的 Release。
if validate_release_tag "$TAG"; then
  if ! command -v node >/dev/null 2>&1; then
    echo "错误: 校验 npm 包版本需要 Node.js 18 或更高版本。" >&2
    exit 1
  fi
  echo "==> 校验 npm 包版本: ${TAG#v}"
  node scripts/check-npm-version.js "${TAG#v}"
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
LINK_FLAGS="${LINK_FLAGS} -X 'atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Source=release'"

ORDINARY_ARCHIVES="
ag_darwin_amd64.tar.gz
ag_darwin_arm64.tar.gz
ag_linux_amd64.tar.gz
ag_linux_arm64.tar.gz
ag_windows_amd64.zip
ag_windows_arm64.zip
"

# Manifest order is source, OS, architecture. The checksum file uses the same
# stable order; filenames remain flat Release attachment basenames.
MANAGED_ARCHIVES="
ag_darwin_amd64_homebrew.tar.gz
ag_darwin_arm64_homebrew.tar.gz
ag_linux_amd64_homebrew.tar.gz
ag_linux_arm64_homebrew.tar.gz
ag_darwin_amd64_npm.tar.gz
ag_darwin_arm64_npm.tar.gz
ag_linux_amd64_npm.tar.gz
ag_linux_arm64_npm.tar.gz
ag_windows_amd64_npm.zip
ag_windows_arm64_npm.zip
ag_windows_amd64_winget.zip
ag_windows_arm64_winget.zip
"

sha256_file() {
  file=$1
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    echo "错误: 生成发布校验和需要 sha256sum 或 shasum。" >&2
    return 1
  fi
}

generate_checksum_file() {
  directory=$1
  output=$2
  shift 2
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$directory" && sha256sum "$@") >"$output"
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$directory" && shasum -a 256 "$@") >"$output"
  else
    echo "错误: 生成发布校验和需要 sha256sum 或 shasum。" >&2
    return 1
  fi
}

validate_archive_matrix() {
  directory=$1
  expected=$2
  actual=$(find "$directory" -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.zip' \) \
    -exec basename {} \; | LC_ALL=C sort)
  expected_sorted=$(printf '%s\n' $expected | LC_ALL=C sort)
  if [ "$actual" != "$expected_sorted" ]; then
    echo "错误: ${directory} 的归档矩阵不匹配。" >&2
    echo "期望:" >&2
    printf '%s\n' "$expected_sorted" >&2
    echo "实际:" >&2
    printf '%s\n' "$actual" >&2
    return 1
  fi
}

extract_archive_binary() {
  archive=$1
  destination=$2
  case "$archive" in
    *.zip)
      command -v unzip >/dev/null 2>&1 || {
        echo "错误: 验证 Windows 归档需要 unzip。" >&2
        return 1
      }
      unzip -p "$archive" ag.exe >"$destination"
      ;;
    *.tar.gz)
      tar -xOf "$archive" ag >"$destination"
      ;;
    *)
      echo "错误: 不支持的归档格式: $archive" >&2
      return 1
      ;;
  esac
  chmod +x "$destination"
}

verify_profile_metadata() {
  profile=$1
  archive=$2
  expected_self_update=$3
  expected_source=$4
  archive_os=$5
  archive_arch=$6

  verify_dir=$(mktemp -d)
  verify_binary="${verify_dir}/ag"
  host_os=$(go env GOHOSTOS)
  host_arch=$(go env GOHOSTARCH)
  if [ "$host_os" = "windows" ]; then
    verify_binary="${verify_binary}.exe"
  fi

  if [ "$archive_os" = "$host_os" ] && [ "$archive_arch" = "$host_arch" ]; then
    extract_archive_binary "$archive" "$verify_binary" || {
      rm -rf "$verify_dir"
      return 1
    }
    verification_kind="归档"
  else
    profile_flags="-s -w"
    profile_flags="${profile_flags} -X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Version=${TAG}"
    profile_flags="${profile_flags} -X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Commit=${COMMIT}"
    profile_flags="${profile_flags} -X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.BuildDate=${BUILD_DATE}"
    profile_flags="${profile_flags} -X atomgit.com/hust-open-atom-club/atomgit-cli/internal/version.Source=${expected_source}"
    GOOS="$host_os" GOARCH="$host_arch" CGO_ENABLED=0 \
      go build -trimpath -ldflags="$profile_flags" -o "$verify_binary" ./cmd/ag || {
        rm -rf "$verify_dir"
        echo "错误: 无法构建 ${profile} 宿主元数据 probe。" >&2
        return 1
      }
    verification_kind="宿主 probe"
  fi

  json_out=$("$verify_binary" version --json 2>&1) || {
    rm -rf "$verify_dir"
    echo "错误: ${profile} ${verification_kind}二进制无法执行: $json_out" >&2
    return 1
  }
  json_self_update=$(printf '%s' "$json_out" | sed -n 's/.*"selfUpdate": *\([^,}]*\).*/\1/p')
  json_source=$(printf '%s' "$json_out" | sed -n 's/.*"source": *"\([^"]*\)".*/\1/p')
  if [ "$json_self_update" != "$expected_self_update" ] || [ "$json_source" != "$expected_source" ]; then
    rm -rf "$verify_dir"
    echo "错误: ${profile} ${verification_kind}元数据为 ${json_self_update}/${json_source}，期望 ${expected_self_update}/${expected_source}" >&2
    return 1
  fi
  echo "==> ${profile} ${verification_kind}元数据通过: selfUpdate=${expected_self_update}, source=${expected_source}"

  rm -rf "$verify_dir"
}

generate_managed_manifest() {
  directory=$1
  manifest="${directory}/package-managers-manifest.json"
  tmp="${manifest}.tmp"
  {
    printf '{\n'
    printf '  "schemaVersion": 1,\n'
    printf '  "tag": "%s",\n' "$TAG"
    printf '  "artifacts": [\n'
    first=1
    for filename in $MANAGED_ARCHIVES; do
      stem=${filename%.tar.gz}
      stem=${stem%.zip}
      source=${stem##*_}
      target=${stem%_*}
      goarch=${target##*_}
      goos=${target#ag_}
      goos=${goos%_*}
      case "$filename" in
        *.zip) format=zip ;;
        *) format=tar.gz ;;
      esac
      digest=$(sha256_file "${directory}/${filename}") || exit 1
      if [ "$first" = "0" ]; then
        printf ',\n'
      fi
      first=0
      printf '    {\n'
      printf '      "source": "%s",\n' "$source"
      printf '      "selfUpdate": false,\n'
      printf '      "goos": "%s",\n' "$goos"
      printf '      "goarch": "%s",\n' "$goarch"
      printf '      "format": "%s",\n' "$format"
      printf '      "filename": "%s",\n' "$filename"
      printf '      "sha256": "%s"\n' "$digest"
      printf '    }'
    done
    printf '\n  ]\n'
    printf '}\n'
  } >"$tmp"
  mv "$tmp" "$manifest"
}

verify_managed_metadata() {
  directory=$1
  checksum_file="${directory}/package-managers-checksums.txt"
  manifest="${directory}/package-managers-manifest.json"
  expected_names=$(printf '%s\n' $MANAGED_ARCHIVES)
  checksum_names=$(awk '{print $2}' "$checksum_file")
  if [ "$checksum_names" != "$expected_names" ]; then
    echo "错误: managed checksum basename 顺序与声明矩阵不一致。" >&2
    return 1
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$directory" && sha256sum -c package-managers-checksums.txt >/dev/null)
  else
    (cd "$directory" && shasum -a 256 -c package-managers-checksums.txt >/dev/null)
  fi

  filename_count=$(grep -c '"filename":' "$manifest")
  disabled_count=$(grep -c '"selfUpdate": false' "$manifest")
  if [ "$filename_count" -ne 12 ] || [ "$disabled_count" -ne 12 ]; then
    echo "错误: managed manifest 必须恰好包含 12 个 selfUpdate=false 条目。" >&2
    return 1
  fi

  for filename in $MANAGED_ARCHIVES; do
    digest=$(sha256_file "${directory}/${filename}") || return 1
    grep -Fq "\"filename\": \"${filename}\"" "$manifest" || {
      echo "错误: managed manifest 缺少 ${filename}" >&2
      return 1
    }
    grep -Fq "\"sha256\": \"${digest}\"" "$manifest" || {
      echo "错误: managed manifest 中 ${filename} 的 SHA-256 不匹配。" >&2
      return 1
    }
  done
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
  json_self_update=$(echo "$json_out" | sed -n 's/.*"selfUpdate": *\([^,}]*\).*/\1/p')
  json_source=$(echo "$json_out" | sed -n 's/.*"source": *"\([^"]*\)".*/\1/p')

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
  if [ "$json_self_update" != "true" ] || [ "$json_source" != "release" ]; then
    echo "错误: JSON policy=${json_self_update}/${json_source}, 期望 true/release" >&2
    exit 1
  fi

  echo "==> 注入校验通过 (tag=$TAG, commit=$COMMIT, buildDate=$BUILD_DATE, selfUpdate=true, source=release)"
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
MANAGED_OUT="${OUT}/package-managers"
mkdir -p "$MANAGED_OUT"

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
for archive in $MANAGED_ARCHIVES; do
  cp "${STAGING}/${archive}" "$MANAGED_OUT/"
done

validate_archive_matrix "$OUT" "$ORDINARY_ARCHIVES"
validate_archive_matrix "$MANAGED_OUT" "$MANAGED_ARCHIVES"
generate_checksum_file "$MANAGED_OUT" "${MANAGED_OUT}/package-managers-checksums.txt" $MANAGED_ARCHIVES
generate_managed_manifest "$MANAGED_OUT"
verify_managed_metadata "$MANAGED_OUT"

verify_os=$(go env GOHOSTOS)
verify_arch=$(go env GOHOSTARCH)
case "${verify_os}/${verify_arch}" in
  linux/amd64|linux/arm64|darwin/amd64|darwin/arm64|windows/amd64|windows/arm64)
    ordinary_verify="${OUT}/ag_${verify_os}_${verify_arch}"
    npm_verify="${MANAGED_OUT}/ag_${verify_os}_${verify_arch}_npm"
    case "$verify_os" in
      windows)
        ordinary_verify="${ordinary_verify}.zip"
        npm_verify="${npm_verify}.zip"
        ;;
      *)
        ordinary_verify="${ordinary_verify}.tar.gz"
        npm_verify="${npm_verify}.tar.gz"
        ;;
    esac
    verify_profile_metadata ordinary "$ordinary_verify" true release "$verify_os" "$verify_arch"
    verify_profile_metadata npm "$npm_verify" false npm "$verify_os" "$verify_arch"
    ;;
  *)
    verify_profile_metadata ordinary "${OUT}/ag_linux_amd64.tar.gz" true release linux amd64
    verify_profile_metadata npm "${MANAGED_OUT}/ag_linux_amd64_npm.tar.gz" false npm linux amd64
    ;;
esac

case "$verify_os" in
  linux|darwin)
    case "$verify_arch" in
      amd64|arm64)
        verify_profile_metadata homebrew \
          "${MANAGED_OUT}/ag_${verify_os}_${verify_arch}_homebrew.tar.gz" \
          false homebrew "$verify_os" "$verify_arch"
        ;;
      *)
        verify_profile_metadata homebrew \
          "${MANAGED_OUT}/ag_linux_amd64_homebrew.tar.gz" \
          false homebrew linux amd64
        ;;
    esac
    ;;
  *)
    verify_profile_metadata homebrew \
      "${MANAGED_OUT}/ag_linux_amd64_homebrew.tar.gz" \
      false homebrew linux amd64
    ;;
esac

case "$verify_os/$verify_arch" in
  windows/amd64|windows/arm64)
    verify_profile_metadata winget \
      "${MANAGED_OUT}/ag_windows_${verify_arch}_winget.zip" \
      false winget windows "$verify_arch"
    ;;
  *)
    verify_profile_metadata winget \
      "${MANAGED_OUT}/ag_windows_amd64_winget.zip" \
      false winget windows amd64
    ;;
esac

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

echo "==> 生成 npm 平台包 ..."
node scripts/build-npm-packages.js "$OUT" "${TAG#v}"
echo ""

# GoReleaser 的全局 checksum 会包含 managed profiles，因此在最终布局中
# 明确重建普通根校验和，保持原有六归档加两个安装脚本的契约。
generate_checksum_file "$OUT" "${OUT}/checksums.txt" $ORDINARY_ARCHIVES install.sh install.ps1
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$OUT/npm" && sha256sum ./*.tgz) >"${OUT}/npm/checksums.txt"
else
  (cd "$OUT/npm" && shasum -a 256 ./*.tgz) >"${OUT}/npm/checksums.txt"
fi
echo "已生成普通归档与安装脚本校验文件 ${OUT}/checksums.txt"
echo "已生成 managed 归档校验文件 ${MANAGED_OUT}/package-managers-checksums.txt"
echo "已生成 managed manifest ${MANAGED_OUT}/package-managers-manifest.json"
echo "已生成 npm 包本地校验文件 ${OUT}/npm/checksums.txt"
echo ""

echo "AtomGit Release 手动上传 basename 清单:"
for attachment in $ORDINARY_ARCHIVES checksums.txt install.sh install.ps1; do
  echo "  $attachment"
done
for attachment in $MANAGED_ARCHIVES package-managers-manifest.json package-managers-checksums.txt; do
  echo "  $attachment"
done
echo ""

if [ "$RELEASE_MODE" = "snapshot" ]; then
  echo "试打包完成。制品位于 ${OUT}/，请勿将未校验的快照制品用于正式发布。"
else
  echo "完成。按以上 basename 清单手动上传 ${OUT}/ 和 ${MANAGED_OUT}/ 中的独立附件到 AtomGit Release「${TAG}」。"
  echo "npm 包位于 ${OUT}/npm/；发布时先发布六个平台包，再发布 atomgit-cli 主包。"
fi
echo "（Windows 也可：PowerShell 执行 install.ps1，或下载 ag_windows_*.zip 手动解压并加入 PATH。）"
