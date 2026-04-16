#!/bin/sh
# AtomGit CLI (ag) — 一键安装
# 仓库: https://gitcode.com/GitCode/ag-cli
#
# 用法:
#   curl -fsSL https://gitcode.com/xiaogang_o/ag-cli/releases/download/v0.2/install.sh | sh
#   AG_VERSION=v0.2 sh install.sh
#   AG_FROM_SOURCE=1 sh install.sh    # 从本仓库源码构建（需 git、Go）

set -eu

REPO_OWNER="${AG_REPO_OWNER:-GitCode}"
REPO_NAME="${AG_REPO_NAME:-ag-cli}"
BASE_URL="https://gitcode.com/${REPO_OWNER}/${REPO_NAME}"
# 默认下载该 tag 下的预编译包。仓库内请随最新 Release 更新；也可用 scripts/build-release.sh 生成 dist/<tag>/install.sh（已写入本次 TAG）再上传。
_BUNDLED_TAG="v0.2"
DEFAULT_VERSION="${AG_DEFAULT_VERSION:-${_BUNDLED_TAG}}"
VERSION="${AG_VERSION:-$DEFAULT_VERSION}"

# 与 GitCode 默认分支一致（当前为 master）
GIT_REF="${AG_GIT_REF:-master}"

die() {
  echo "install.sh: $*" >&2
  exit 1
}

detect_os() {
  os=$(uname -s)
  case "$os" in
  Linux) echo linux ;;
  Darwin) echo darwin ;;
  *) die "unsupported OS: $os" ;;
  esac
}

detect_arch() {
  arch=$(uname -m)
  case "$arch" in
  x86_64 | amd64) echo amd64 ;;
  aarch64 | arm64) echo arm64 ;;
  *) die "unsupported CPU: $arch" ;;
  esac
}

pick_install_dir() {
  if [ -n "${AG_INSTALL_DIR:-}" ]; then
    echo "$AG_INSTALL_DIR"
    return
  fi
  prefix="${AG_PREFIX:-}"
  if [ -z "$prefix" ]; then
    if [ -w /usr/local/bin ] 2>/dev/null; then
      prefix="/usr/local"
    else
      prefix="$HOME/.local"
    fi
  fi
  echo "$prefix/bin"
}

install_binary() {
  os=$(detect_os)
  arch=$(detect_arch)
  tmpdir=$(mktemp -d)
  trap "rm -rf \"$tmpdir\"" EXIT INT HUP

  tag="$VERSION"
  asset="ag_${os}_${arch}.tar.gz"
  url="${BASE_URL}/releases/download/${tag}/${asset}"

  echo "Downloading ${url} ..."
  if ! curl -fsSL "$url" -o "${tmpdir}/${asset}"; then
    die "无法下载预编译包（请确认已发布 ${tag} 且附件名为 ${asset}）。也可使用: AG_FROM_SOURCE=1 sh install.sh"
  fi

  (cd "$tmpdir" && tar -xzf "$asset")
  if [ ! -f "${tmpdir}/ag" ]; then
    die "压缩包内未找到名为 ag 的可执行文件"
  fi

  dest=$(pick_install_dir)
  mkdir -p "$dest"
  if [ ! -w "$dest" ] && command -v sudo >/dev/null 2>&1; then
    echo "Installing to $dest (sudo) ..."
    sudo install -m 0755 "${tmpdir}/ag" "${dest}/ag"
  else
    echo "Installing to $dest ..."
    install -m 0755 "${tmpdir}/ag" "${dest}/ag"
  fi

  case ":${PATH:-}:" in
  *:"$dest":*) ;;
  *)
    echo "请将目录加入 PATH，例如在 ~/.profile 中追加:"
    echo "  export PATH=\"$dest:\$PATH\""
    ;;
  esac

  "${dest}/ag" --help >/dev/null 2>&1 || true
  echo "ag 已安装: ${dest}/ag"
}

install_from_source() {
  command -v git >/dev/null 2>&1 || die "需要 git"
  command -v go >/dev/null 2>&1 || die "需要 Go 工具链"

  tmpdir=$(mktemp -d)
  trap "rm -rf \"$tmpdir\"" EXIT INT HUP

  echo "Cloning ${BASE_URL}.git (ref: ${GIT_REF}) ..."
  git clone --depth 1 --branch "$GIT_REF" "${BASE_URL}.git" "${tmpdir}/src"

  dest=$(pick_install_dir)
  mkdir -p "$dest"
  echo "Building ..."
  (cd "${tmpdir}/src" && go build -o "${tmpdir}/ag" ./cmd/ag)

  if [ ! -w "$dest" ] && command -v sudo >/dev/null 2>&1; then
    echo "Installing to $dest (sudo) ..."
    sudo install -m 0755 "${tmpdir}/ag" "${dest}/ag"
  else
    echo "Installing to $dest ..."
    install -m 0755 "${tmpdir}/ag" "${dest}/ag"
  fi

  echo "ag 已安装: ${dest}/ag"
}

main() {
  if [ -n "${AG_FROM_SOURCE:-}" ] && [ "$AG_FROM_SOURCE" != "0" ]; then
    install_from_source
  else
    install_binary
  fi
}

main "$@"
