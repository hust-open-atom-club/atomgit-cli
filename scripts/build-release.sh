#!/bin/sh
# 生成预编译包到 dist/<版本>/：
#   - Linux/macOS: ag_<os>_<arch>.tar.gz（包内可执行文件名为 ag）
#   - Windows:     ag_windows_<arch>.zip（包内为 ag.exe）
#
# 用法:
#   ./scripts/build-release.sh              # 版本来自 git describe，或环境变量 TAG
#   TAG=v0.1.0 ./scripts/build-release.sh

set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

TAG="${TAG:-}"
if [ -z "$TAG" ]; then
  TAG=$(git describe --tags --always 2>/dev/null) || TAG="dev"
fi

OUT="${ROOT}/dist/${TAG}"
mkdir -p "$OUT"

echo "输出目录: $OUT"
echo "版本标识: $TAG"
echo ""

build_one() {
  goos=$1
  goarch=$2
  asset="ag_${goos}_${goarch}.tar.gz"
  tmpdir=$(mktemp -d)

  echo "构建 $goos/$goarch -> $asset"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${tmpdir}/ag" ./cmd/ag
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
  GOOS=windows GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${tmpdir}/ag.exe" ./cmd/ag
  (cd "$tmpdir" && zip -q "${OUT}/${asset}" ag.exe)
  rm -rf "$tmpdir"
  ls -lh "${OUT}/${asset}"
  echo ""
}

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

echo "完成。将 dist/${TAG}/ 下各 .tar.gz / .zip、install.sh 与 install.ps1 作为 GitCode Release「${TAG}」的附件上传即可。"
echo "（Windows 也可：PowerShell 执行 install.ps1，或下载 ag_windows_*.zip 手动解压并加入 PATH。）"
