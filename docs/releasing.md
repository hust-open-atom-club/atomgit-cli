# 发布指南

本文档介绍 AtomGit CLI 的 GoReleaser 打包、npm 制品发布、Homebrew tap 和 Nix package 维护流程。

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

- 七个名称不变的归档；其中 Linux 支持 amd64、arm64 和 loong64，macOS 与 Windows 支持 amd64 和 arm64。
- 已绑定当前 tag 的 `install.sh` 和 `install.ps1`。
- `npm/` 下的七个平台二进制包、一个主启动包和独立的 `npm/checksums.txt`。
- 仅覆盖 AtomGit Release 附件（七个归档和两个安装脚本）的根 `checksums.txt`。

仓库根目录的 `install.sh` 和 `install.ps1` 是不绑定具体版本的源码模板，使用 `__AG_RELEASE_TAG__` 占位符。发布构建只替换安装器中的绑定变量，生成的 Release 附件默认下载本次 tag；模板中的使用示例始终保持为 `latest` 和 `vX.Y.Z`，无需随版本手工修改。

npm 平台包复用对应操作系统和架构的普通 Release 归档。

AtomGit Release 只上传七个普通平台归档、两个安装脚本和根 `checksums.txt`，共十个项目附件。AtomGit 还会自动展示四个源码归档，因此保留 LoongArch64 支持的 Release 页面通常共显示十四个 artifacts。Homebrew、Scoop 和 WinGet 复用普通平台归档，不再发布按 distribution 重复打包的专用附件。

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

`scripts/build-release.sh` 始终使用 GoReleaser 的 `--skip=publish`，只在本地准备并验证制品，然后打印完整的附件 basename 清单。发布上传由下述 `make publish` 入口调用独立脚本完成；单独执行构建脚本不会创建 AtomGit Release 或上传附件。

# Linux
(cd dist/vX.Y.Z && sha256sum -c checksums.txt)

# macOS
(cd dist/vX.Y.Z && shasum -a 256 -c checksums.txt)
```

npm tarball 不作为 AtomGit Release 附件上传，可在发布到 npm registry 前单独校验：

```bash
(cd dist/vX.Y.Z/npm && shasum -a 256 -c checksums.txt)
```

`scripts/build-release.sh` 始终使用 GoReleaser 的 `--skip=publish`，只在本地准备并验证制品，然后打印完整的附件 basename 清单。发布上传由下述 `make publish` 入口调用独立脚本完成；单独执行构建脚本不会创建 AtomGit Release 或上传附件。

## 自动发布 AtomGit Release

正式 tag 所在提交准备好发布说明后，使用单一入口完成格式检查、Go 测试、构建、跨平台打包、附件上传和发布后回读验证：

```bash
make publish VERSION=vX.Y.Z NOTES_FILE=notes.md
```

默认仓库为 `hust-open-atom-club/atomgit-cli`。发布其他仓库、指定显示名称或创建预发布版本时可使用：

```bash
make publish \
  VERSION=vX.Y.Z \
  NOTES_FILE=notes.md \
  REPOSITORY=owner/repo \
  RELEASE_NAME="Version X.Y.Z" \
  PRERELEASE=1
```

发布入口要求 tag 存在、指向当前提交且工作区干净，并通过 `ag` 的现有认证配置访问 AtomGit。CI 中应从 secret 写入权限受限的临时 token 文件，格式和位置见[配置与认证](configuration.md)；不得把 PAT 写入仓库、制品或日志。

自动化流程会验证固定的十个项目附件、归档内容、安装脚本版本和 SHA-256，然后创建或安全补齐 Release，并确认 AtomGit 自动生成的四个源码归档存在。重复执行时，已有附件必须下载后与本地 checksum 一致才会跳过；目标提交冲突、未知附件、同名内容不一致或 API/上传失败都会停止。上传中断后可用相同 tag、发布说明和 `dist/<tag>/` 制品重新执行命令，脚本会列出尚未完成的附件。已发布且内容冲突的附件不会被自动覆盖；需要人工确认 Release 状态后再决定回滚。

发布完成前会再次校验 tag、目标提交、名称、说明、发布状态、全部附件下载 URL，并下载每个附件核对 SHA-256。Homebrew Formula 更新仍由独立流程处理。

未创建 tag 时，可使用 `make release-snapshot VERSION=vX.Y.Z` 进行本地试打包。Snapshot 允许脏工作区，其制品仅用于验证，不应上传到正式 Release。

底层 `scripts/build-release.sh` 也接受 `TAG`、`AG_RELEASE_SNAPSHOT=1` 和 `SOURCE_DATE_EPOCH` 环境变量。`SOURCE_DATE_EPOCH` 会同时固定二进制中的构建日期以及归档内文件的时间戳，用于生成可复现的发布制品；历史两段式 tag 仅保留给 snapshot 兼容。


## 发布到 npm registry

AtomGit Release 发布并验证完成后，再发布 `dist/vX.Y.Z/npm/` 中的 npm tarball。发布前确认当前 npm 账号有权发布 `@hust-open-atom-club` scope，并按 registry 的认证和双因素认证要求完成登录：

```bash
npm whoami


# Linux
(cd dist/vX.Y.Z/npm && sha256sum -c checksums.txt)


# macOS
(cd dist/vX.Y.Z/npm && shasum -a 256 -c checksums.txt)
```

目录中应包含七个平台包和一个 `@hust-open-atom-club/atomgit-cli` 主启动包。逐个使用 `npm pack <tarball> --dry-run --json` 检查包名、版本和文件清单，并检查包内元数据的平台限制，确保八个包的版本均为 `X.Y.Z`，且主包的 `optionalDependencies` 精确引用同版本的七个平台包。

先依次发布七个平台包，每发布一个包都应等待精确版本能够从 registry 查询到，再继续发布下一个包：

```bash
npm publish <platform-package.tgz> --access public --ignore-scripts
npm view <platform-package-name>@X.Y.Z version
```

确认七个平台包全部可见后，最后发布主启动包并回读版本：

```bash
npm publish <atomgit-cli-package.tgz> --access public --ignore-scripts
npm view @hust-open-atom-club/atomgit-cli@X.Y.Z version
```

npm 版本不可覆盖。重新执行中断的发布流程时，先通过 `npm view <name>@X.Y.Z --json` 检查已存在包的名称、版本和 `dist.integrity` 或 `dist.shasum`；只有远端内容与本地 tarball 完全一致时才跳过，出现冲突或无法确认时应停止，不得尝试替换已发布版本。

全部包可见后，在唯一的系统临时目录中安装主包的精确版本，确认 npm 自动选择了当前操作系统和架构对应的平台包，并执行 `ag version` 核对 tag。验证完成后删除该临时目录。

npm tarball 只发布到 npm registry，不得作为 AtomGit Release 附件上传。


## 维护 Homebrew tap

Homebrew tap 位于 [hust-open-atom-club/homebrew-tap](https://github.com/hust-open-atom-club/homebrew-tap)，使用 GitHub Actions 自动维护。更新工作流每 4 个小时检测一次 AtomGit 最新稳定版本；发现新版本后，会下载 macOS 和 Linux 的 amd64/arm64 Release 归档，确认归档包含 `ag`，重新计算 SHA-256，并更新 [Formula/atomgit-cli.rb](https://github.com/hust-open-atom-club/homebrew-tap/blob/main/Formula/atomgit-cli.rb)。工作流只允许更新 Formula 文件，并在 macOS 和 Linux 测试通过后自动 squash 合并更新 PR。

如果距离版本发布超过 4 个小时仍未正确更新，请先[手动运行更新工作流](https://github.com/hust-open-atom-club/homebrew-tap/actions/workflows/update-formula.yml)。如果工作流仍然失败，请[发起一个 Issue](https://github.com/hust-open-atom-club/homebrew-tap/issues)，或者手动更新 Formula 中的版本号、四个平台归档 URL 和 SHA-256 后发起 PR。

## 维护 Nix package

仓库 flake 从 `nix/` 下的独立表达式提供两种 package：

- `stable` 从上游仓库的最新正式 AtomGit Release 源码归档构建，并固定版本、源码 hash 和 `vendorHash`。
- `latest` 直接从当前 flake revision 的源码构建，因此始终对应检出仓库的最新 commit；工作流只维护其 `vendorHash`。

两个 package 都由 Nix 管理；支持发行来源字段的版本会报告 `selfUpdate=false, source=nix`。`default` 和兼容名称 `ag` 都指向 `stable`。

`.gitcode/workflows/update-nix.yml` 每天在默认分支上运行，也支持手动触发。工作流从 AtomGit Release API 读取 stable 版本，然后使用 nixpkgs 的 `nix-update` 更新 stable 的版本、源码 hash 和 `vendorHash`，并刷新当前 commit 对应的 latest `vendorHash`；构建验证后在内容变化时直接提交到默认分支。工作流需要 `repository: write`，并在单次 `git push` 中使用自动生成的 `ATOMGIT_TOKEN`，不需要额外长期 token。

工作流 runner 通过清华 TUNA 镜像执行 Nix 单用户安装，并禁用安装器默认添加的官方 channel，再从 TUNA 的 `nixpkgs-unstable` channel 安装 `nix-update`；Nix binary cache 使用显式优先级，依次尝试 TUNA、SJTU、USTC、CERNET，最后回退到官方 cache。项目 flake 的 nixpkgs inputs 通过 CERNET 的 NJU Git 镜像进行浅克隆，避免动态镜像调度因 runner 线路而选择不可用节点；两个 inputs 分别跟踪 `nixos-unstable` 和 `nixpkgs-26.05-darwin`。

`nix-update --build` 的 Go 模块下载显式使用 `goproxy.cn`、阿里云和 direct 的故障转移链；代理之间使用 `|` 分隔，使连接超时等网络错误也会切换到下一个来源。

可在本地复现相同更新；开发环境已包含 `nix-update`：

```bash
nix develop

# stable：从 AtomGit 最新正式 Release 更新
stable_version=$(curl --fail --silent --show-error \
  https://api.atomgit.com/api/v5/repos/hust-open-atom-club/atomgit-cli/releases/latest \
  | jq -er '.tag_name | select(test("^v[0-9]+\\.[0-9]+\\.[0-9]+([+-].*)?$")) | sub("^v"; "")')
test -n "$stable_version"
nix-update stable --flake --version "$stable_version" --build

# latest：使用当前 flake revision，只刷新其 vendorHash
nix-update latest --flake --version=skip --build
rm -f result result-*
```

`nix-update --build` 会创建 Nix 的 `result` 结果链接；上述本地流程在完成后删除它，仓库也忽略 `result` 和 `result-*`。`nix-update` 会同时维护源码 hash 和 Go `vendorHash`。当前 nixos-unstable 已停止支持 Intel macOS，因此 flake 仅为 `x86_64-darwin` 使用仍受维护的 `nixpkgs-26.05-darwin` input；其他平台继续使用 nixos-unstable。
## 维护 WinGet

WinGet 清单托管在社区仓库 [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)，包 ID 为 `HUSTOpenAtomClub.AtomGitCLI`。每个版本在 `manifests/h/HUSTOpenAtomClub/AtomGitCLI/<version>/` 下包含三个 YAML 清单文件（主清单、installer 和 locale），其中 installer 清单固定各平台安装包的下载 URL 和 SHA-256。

> [!NOTE]
> WinGet 仓库审核需要时间。如果 WinGet 暂时获取不到最新版本，请在进行下面的操作前检查仓库的 [Pull Request](https://github.com/microsoft/winget-pkgs/pulls?q=is%3Apr+is%3Aopen+New+version%3A+HUSTOpenAtomClub.AtomGitCLI+version) 页面中是否存在已经提交但仍处于 Open 状态的合并请求，避免重复提交。

发布新版本后，使用 [Komac](https://github.com/russellbanks/Komac) 生成并提交清单更新：

```powershell
komac update HUSTOpenAtomClub.AtomGitCLI --version X.Y.Z --urls <Windows ARM64 归档 URL> <Windows AMD64 归档 URL>
```

Komac 将自动根据传入的 URL 下载包，计算 SHA-256 并更新清单。确认无误后，选择 `Submit` 即可自动向 microsoft/winget-pkgs 发起合并请求。

> [!NOTE]
> Komac 需要配置 Personal access tokens(classic) 才能正常发起合并请求，参阅 [Komac: GitHub Token Setup](https://github.com/russellbanks/Komac#github-token-setup).

发起合并请求后，前往对应页面同意 CLA 后等待合并即可。

## 维护 Scoop

Scoop bucket 位于 [hust-open-atom-club/ScoopBucket](https://github.com/hust-open-atom-club/ScoopBucket)，使用 Excavator GitHub Actions 工作流自动维护。Excavator 每 4 个小时检测一次新版本。如果检测到新版本，将自动更新清单中的版本号、下载链接和 SHA-256 并提交合并请求。

如果距离版本发布超过 4 个小时仍没能正确更新，请[发起一个 Issue](https://github.com/hust-open-atom-club/ScoopBucket/issues)，或者手动更新 [bucket/atomgit-cli.json](https://github.com/hust-open-atom-club/ScoopBucket/blob/main/bucket/atomgit-cli.json) 清单中的相应字段后发起合并请求。
