#!/bin/sh
# 生成与 install.sh 约定一致的预编译包：dist/<版本>/ag_<os>_<arch>.tar.gz（包内仅含可执行文件 ag）
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

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64

# 生成与本次 TAG 默认一致的 install.sh（避免 Release 附件里脚本仍指向旧版本）
ESC_TAG=$(printf '%s\n' "$TAG" | sed 's/[\/&]/\\&/g')
sed "s/^_BUNDLED_TAG=.*/_BUNDLED_TAG=\"${ESC_TAG}\"/" "$ROOT/install.sh" > "${OUT}/install.sh"
chmod +x "${OUT}/install.sh"
echo "已生成 ${OUT}/install.sh（默认 TAG=${TAG}）"
echo ""

echo "完成。将 dist/${TAG}/ 下各 .tar.gz 与 install.sh 作为 GitCode Release「${TAG}」的附件上传即可。"
