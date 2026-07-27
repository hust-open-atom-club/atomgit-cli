# 发布指南

本文档介绍 AtomGit CLI 的 GoReleaser 打包、npm 制品发布和 Nix package 维护流程。

## 发布打包

发布版使用 [GoReleaser](https://goreleaser.com/install/) 打包，tag 统一使用 `vX.Y.Z` 三段式 SemVer。先同步 npm 版本并提交改动，再在该提交上创建版本 tag：

```bash
VERSION=X.Y.Z
npm run version:npm -- "$VERSION"
git add package.json package-lock.json
git commit -m "chore: prepare v${VERSION}"
git tag "v${VERSION}"
make release VERSION="v${VERSION}"
```

`make release` 会检查工作区干净、tag 存在且指向当前 HEAD，然后在 `dist/vX.Y.Z/` 生成以下文件：

- 七个名称不变的普通归档；其中 Linux 支持 amd64、arm64 和 loong64，macOS 与 Windows 支持 amd64 和 arm64；二进制报告 `selfUpdate=true, source=release`。
- `package-managers/` 下 npm 的七个、Homebrew 的四个、Winget 的两个和 Scoop 的两个独立归档；文件名为 `ag_<os>_<arch>_<source>.<ext>`，二进制报告 `selfUpdate=false` 和对应来源。
- `package-managers/package-managers-manifest.json` 与只使用归档 basename 的 `package-managers-checksums.txt`。
- 已绑定当前 tag 的 `install.sh` 和 `install.ps1`。
- `npm/` 下的七个平台二进制包、一个主启动包和独立的 `npm/checksums.txt`。
- 仅覆盖 AtomGit Release 附件（七个归档和两个安装脚本）的根 `checksums.txt`。

仓库根目录的 `install.sh` 和 `install.ps1` 是不绑定具体版本的源码模板，使用 `__AG_RELEASE_TAG__` 占位符。发布构建只替换安装器中的绑定变量，生成的 Release 附件默认下载本次 tag；模板中的使用示例始终保持为 `latest` 和 `vX.Y.Z`，无需随版本手工修改。

npm 平台包从对应的 `_npm` 归档提取二进制；Homebrew formula、Winget manifest 和 Scoop bucket manifest 应分别消费匹配的 `_homebrew`、`_winget` 与 `_scoop` 归档及校验和。Nix 不生成 `_nix` 归档。

发布 npm 制品时，先发布 `npm/` 下七个名称带平台和架构的包；确认它们可用后，再发布 `atomgit-cli` 主包。主包和平台包必须使用相同版本。

上传 Release 附件前可校验所有制品：

```bash
# Linux
(cd dist/vX.Y.Z && sha256sum -c checksums.txt)

# macOS
(cd dist/vX.Y.Z && shasum -a 256 -c checksums.txt)
```

npm tarball 不作为 AtomGit Release 附件上传，可在发布到 npm registry 前单独校验：

```bash
(cd dist/vX.Y.Z/npm && shasum -a 256 -c checksums.txt)
```

包管理器归档使用独立校验文件：

```bash
(cd dist/vX.Y.Z/package-managers && shasum -a 256 -c package-managers-checksums.txt)
```

`scripts/build-release.sh` 始终使用 GoReleaser 的 `--skip=publish`，只在本地准备并验证制品，然后打印完整的附件 basename 清单。维护者按该清单手工上传七个普通归档、安装脚本、十五个包管理器归档、manifest 和包管理器校验文件；脚本不会创建 AtomGit Release 或上传附件。

未创建 tag 时，可使用 `make release-snapshot VERSION=vX.Y.Z` 进行本地试打包。Snapshot 允许脏工作区，其制品仅用于验证，不应上传到正式 Release。

底层 `scripts/build-release.sh` 也接受 `TAG`、`AG_RELEASE_SNAPSHOT=1` 和 `SOURCE_DATE_EPOCH` 环境变量。`SOURCE_DATE_EPOCH` 会同时固定二进制中的构建日期以及归档内文件的时间戳，用于生成可复现的发布制品；历史两段式 tag 仅保留给 snapshot 兼容。

## 维护 Nix package

仓库 flake 提供两种 package：

- `stable` 从对应版本的 AtomGit Release 源码归档构建，并固定源码 hash 和 `vendorHash`。
- `latest` 从当前 flake revision 的源码构建，使用独立的 `vendorHash`。

两个 package 都由 Nix 管理；支持发行来源字段的版本会报告 `selfUpdate=false, source=nix`。

`default` 和兼容名称 `ag` 都指向 `stable`。更新 Nix package 时，推荐先进入 flake 提供的开发环境：

当前 nixos-unstable 已停止支持 Intel macOS，因此 flake 仅为 `x86_64-darwin` 使用仍受维护的 `nixpkgs-26.05-darwin` input；其他平台继续使用 nixos-unstable。

```bash
# 默认更新 stable 的版本、Release 源码 hash 和 vendorHash
nix develop
./scripts/update-nix-package.sh vX.Y.Z

# 只更新当前源码对应的 latestVendorHash
./scripts/update-nix-package.sh --latest
```

也可以不进入开发环境直接运行，但需要预先安装 Nix 和 Git，并要求 `tar` 支持以 NUL 分隔的文件列表；Linux 上的 GNU tar 和 macOS 默认的 bsdtar 均受支持。脚本使用 `nix store prefetch-file` 计算 stable 源码 hash，并通过 `buildGoModule` 校验对应的 `vendorHash`。更新后会构建目标 package 并执行 `ag version --json`；验证失败时自动恢复原始 `flake.nix`，且不会提交、打标签或推送。
