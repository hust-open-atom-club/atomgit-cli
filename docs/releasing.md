# 发布指南

本文档介绍 AtomGit CLI 的 GoReleaser 打包、npm 制品发布，以及 Homebrew tap、Nix package、WinGet、Scoop 和 AUR package 的维护流程。

## 目录

- [发布打包](#发布打包)
- [CI 验证时机](#ci-验证时机)
- [自动发布 AtomGit Release](#自动发布-atomgit-release)
- [发布到 npm registry](#发布到-npm-registry)
- [维护 Homebrew tap](#维护-homebrew-tap)
- [维护 Nix package](#维护-nix-package)
- [维护 WinGet](#维护-winget)
- [维护 Scoop](#维护-scoop)
- [维护 OpenKylin package](#维护-openkylin-package)
- [维护 AUR package](#维护-aur-package)

## CI 验证时机

通用 `.gitcode/workflows/ci.yml` 只监听 `main` 分支的 push，不在 Pull Request 或
其他分支 push 时运行。完整测试、竞态检测、发布目标交叉编译、平台测试编译、漏洞
扫描和 npm 测试因此针对已经进入 `main` 的精确提交执行。PR 作者需在合并前完成与
改动相符的本地验证并在 PR 描述中记录结果；PR 不应依赖一个不会启动的通用 CI
必需检查。合并后应确认 `main` 的运行成功，失败时停止后续发布并优先修复或回退。

Nix 更新使用独立的 `.gitcode/workflows/update-nix.yml`。该 workflow 保留
`main`、`test`、`nix-update` 分支 push 和手动触发入口，并在自身流程中构建、验证
Nix package；非 `main` 的维护分支不依赖通用 CI。Nix 更新提交进入 `main` 后，仍会
由通用 CI 对该精确提交运行完整矩阵。

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

`go.mod` 的 `go` 行规定源码构建最低支持 Go 1.26.6，`toolchain` 行规定开发和正式
发布建议使用 Go 1.26.8。`scripts/build-release.sh` 会读取、设置并回读 `toolchain`
中的精确版本，再让 GoReleaser 继承相同的 `GOTOOLCHAIN`；如果无法下载、验证或
执行该工具链，发布会在生成制品前停止。

`make vulncheck` 会使用同一版本构建 `CGO_ENABLED=0` 的实际 `ag` 二进制，并用固定
版本的 `govulncheck` 以 binary 模式查询 `https://vuln.go.dev`。可达漏洞、工具下载
失败、数据库不可用或扫描器错误都会使门禁失败；规范数据库中已经撤回的报告不计为
漏洞。更新最低支持版本时修改 `go.mod` 的 `go` 行，并运行
`make go-min-version test-min-go`；更新正式发布版本时修改 `toolchain` 行，并重新运行 `make go-version`、
`make vulncheck` 和下文的发布验证。

`make release` 会检查工作区干净、tag 存在且指向当前 HEAD，然后在 `dist/vX.Y.Z/` 生成以下文件：

- 七个名称不变的归档；其中 Linux 支持 amd64、arm64 和 loong64，macOS 与 Windows 支持 amd64 和 arm64。
- 已绑定当前 tag 的 `install.sh` 和 `install.ps1`。
- `npm/` 下的七个平台二进制包、一个主启动包和独立的 `npm/checksums.txt`。
- 仅覆盖 AtomGit Release 附件（七个归档和两个安装脚本）的根 `checksums.txt`。

仓库根目录的 `install.sh` 和 `install.ps1` 是不绑定具体版本的源码模板，使用 `__AG_RELEASE_TAG__` 占位符。发布构建只替换安装器中的绑定变量，生成的 Release 附件默认下载本次 tag；模板中的使用示例始终保持为 `latest` 和 `vX.Y.Z`，无需随版本手工修改。

`install.ps1` 必须保持纯 ASCII，确保 Windows PowerShell 5.1 在无 `charset` 的 HTTP 响应以及使用系统 ANSI 代码页读取本地脚本时都能正确解析。发布制品校验会拒绝包含非 ASCII 字节的 PowerShell 安装器。

npm 平台包复用对应操作系统和架构的普通 Release 归档。

AtomGit Release 只上传七个普通平台归档、两个安装脚本和根 `checksums.txt`，共十个项目附件。AtomGit 还会自动展示四个源码归档，因此保留 LoongArch64 支持的 Release 页面通常共显示十四个 artifacts。Homebrew、Scoop 和 WinGet 复用普通平台归档，不再发布按 distribution 重复打包的专用附件。

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

AtomGit Release 发布并验证完成后，再处理 `dist/vX.Y.Z/npm/` 中的七个平台包和一个主启动包。仓库只保留 `npm run publish:npm` 这一套安全入口；不要绕过它逐个运行 `npm publish`。

发布前使用不访问 registry 的校验模式：

```bash
npm run publish:npm -- vX.Y.Z dist/vX.Y.Z/npm --dry-run
```

脚本会检查 `checksums.txt`、tarball 名称和内容、完整的受支持平台集合、包名、版本、`os`/`cpu` 元数据、Unix 可执行位，以及主包 `optionalDependencies` 对七个平台包的精确引用。任何缺失、额外包或 checksum 冲突都会在 registry 请求之前失败。

### AtomGit Actions：暂存后人工审批

AtomGit Actions 当前不在 npm Trusted Publishing 支持的 CI 平台列表中，因此默认入口使用 [staged publishing](https://docs.npmjs.com/staged-publishing)：自动化只执行 `npm stage publish`，包不会立即公开；维护者随后通过 2FA 审批每个 stage。该流程要求 Node.js 22.14.0+、npm 11.15.0+，并要求目标包已经存在于 npm registry。

在 CI secret 中配置具备 scope 写权限的 granular access token 即可，token 不需要也不应启用 Bypass 2FA。通过临时 npm user config 或 `NODE_AUTH_TOKEN` 提供凭据，不得把 token 写入仓库、制品、命令参数或日志。包的发布访问设置可以继续保持要求 2FA 的安全策略。

校验通过后执行默认暂存流程：

```bash
npm run publish:npm -- vX.Y.Z dist/vX.Y.Z/npm
```

脚本先查询已公开版本和现有 stage：

- 已公开且 integrity/shasum 与本地 tarball 一致的包会安全跳过。
- 已暂存且 shasum 一致的包会复用原 stage ID，适合在自动化中断后直接重试。
- 已公开或已暂存但内容不一致时立即失败，不会尝试替换 npm 的不可变版本。
- 缺失的平台包按稳定顺序暂存；只有七个平台版本全部公开后，下一次运行才会暂存主包。输出会列出当前阶段的待审批 stage ID。

维护者先检查 stage，再依次审批七个平台包：

```bash
npm stage view <platform-stage-id>
npm stage download <platform-stage-id>
npm stage approve <platform-stage-id>
```

`npm stage approve` 会交互式要求 2FA。批准平台包后重新运行发布命令；脚本确认七个平台版本均已公开且与本地 tarball 一致后，才会创建主包 stage：

```bash
npm run publish:npm -- vX.Y.Z dist/vX.Y.Z/npm
npm stage view <main-stage-id>
npm stage approve <main-stage-id>
```

主包审批完成后再次运行相同的 `npm run publish:npm` 命令。脚本会验证八个远端版本的 integrity/shasum，在隔离临时目录安装主包的精确版本，并通过 `npm exec --prefix` 调用 npm 生成的 `.bin/ag`（Windows 为 `ag.cmd`）执行 `ag version --json`；只有实际命令入口可执行、输出与本次 `vX.Y.Z` tag 一致，且 JSON 只包含 `version`、`commit` 和 `buildDate` 时才报告发布完成。

staged publishing 不能创建从未在 npm registry 发布过的新包。新增平台包的首个版本应在 npm 支持的 Trusted Publishing CI 中完成，不能通过降低包的 2FA 或 token 安全设置绕过限制。

### Trusted Publishing

如果后续将最终发布迁移到 GitHub Actions、GitLab.com shared runners 或 CircleCI cloud，可在 npm 配置对应的 [Trusted Publisher](https://docs.npmjs.com/trusted-publishers/)，使用 Node.js 22.14.0+ 和 npm 11.5.1+，并显式运行直接发布模式：

```bash
npm run publish:npm -- vX.Y.Z dist/vX.Y.Z/npm --publish
```

直接模式仅用于受支持的 OIDC 工作流，不应回退到长期 Bypass 2FA token。npm CLI 会自动交换短期 OIDC 凭据；GitHub Actions 和 GitLab.com 会自动生成 provenance。正式启用前需要为八个包分别配置相同的受信任工作流和允许的操作。

发布中断后必须继续使用完全相同的 version 和制品目录。不要重新打包或替换已经暂存或公开的 tarball；出现 integrity/shasum 冲突时应停止并查明制品来源。npm tarball 只发布到 npm registry，不得作为 AtomGit Release 附件上传。


## 维护 Homebrew tap

项目 Homebrew Tap 位于 [hust-open-atom-club/homebrew-tap](https://github.com/hust-open-atom-club/homebrew-tap)，提供跟踪 AtomGit `main` 最新 commit 的开发快照；正式稳定版由 Homebrew Core 提供。Tap 的 GitHub Actions 每小时检查一次 `main`，发现新 commit 后更新 [Formula/atomgit-cli.rb](https://github.com/hust-open-atom-club/homebrew-tap/blob/main/Formula/atomgit-cli.rb) 中固定的 40 位 revision 和开发版本号，并从该不可变 commit 的源码构建。工作流只允许更新 Formula 文件；Formula 在 macOS 和 Linux 上测试通过后，更新 PR 才会自动 squash 合并。

如果 `main` 更新超过一个工作流周期后 Tap 仍未跟进，请先[手动运行更新工作流](https://github.com/hust-open-atom-club/homebrew-tap/actions/workflows/update-formula.yml)。如果工作流仍然失败，请[发起一个 Issue](https://github.com/hust-open-atom-club/homebrew-tap/issues)，或者手动更新 Formula 的 revision 和开发版本号并发起 PR；不要改为复用正式 Release 归档。安装、升级和在 Homebrew Core 与项目 Tap 之间切换的方式见[安装指南](installation.md#homebrew-安装)。

## 维护 Nix package

仓库 flake 从 `nix/` 下的独立表达式提供两种 package：

- `stable` 从上游仓库的最新正式 AtomGit Release 源码归档构建，并固定版本、源码 hash 和 `vendorHash`。
- `latest` 直接从当前 flake revision 的源码构建，因此始终对应检出仓库的最新 commit；工作流只维护其 `vendorHash`。

两个 package 都由 Nix 管理；共享构建参数暂时保留 `Source=nix` 注入，以便仍基于旧版源码的 `stable` 正确报告由 Nix 管理。源码移除发行来源字段后，该兼容参数不会改变版本输出。`default` 和兼容名称 `ag` 都指向 `stable`。

Nix package 使用 `go` 行声明的最低版本约束，并由锁定的 nixpkgs input 提供实际编译器；它不要求与官方 Release 使用的建议工具链补丁版本完全一致。更新 flake inputs 时仍需确认所有支持平台提供的 Go 版本不低于 1.26.6。

`.gitcode/workflows/update-nix.yml` 在 `main`、`test`、`nix-update` 分支 push 时运行，也支持手动触发。工作流从 AtomGit Release API 读取 stable 版本，然后使用 nixpkgs 的 `nix-update` 更新 stable 的版本、源码 hash 和 `vendorHash`，并刷新当前 commit 对应的 latest `vendorHash`；它通过 `nix-update --build` 和 stable 二进制版本元数据回读完成自身验证，再在内容变化时使用配置的 `NIX_UPDATE_TOKEN` 通过 Contents API 写回当前目标分支。工作流需要 `repository: write`；token 的安全加固与工具链固定由 Issue #123 跟踪。

工作流 runner 优先通过校园网联合镜像站（CERNET）执行 Nix 单用户安装，并禁用安装器默认添加的官方 channel，再从 CERNET 的 `nixpkgs-unstable` channel 安装 `nix-update`；Nix binary cache 按优先级依次尝试 CERNET、清华 TUNA、SJTU、USTC，最后回退到官方 cache。项目 flake 的 nixpkgs inputs 是例外，仍固定使用 NJU Git 镜像；两个 inputs 分别跟踪 `nixos-unstable` 和 `nixpkgs-26.05-darwin`。

`nix-update --build` 的 Go 模块下载仅使用 `goproxy.cn` Go module proxy。

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

## 维护 OpenKylin package

以下介绍从上游发布新版本到下游 OpenKylin 仓库的完整维护流程。

### 仓库结构

OpenKylin 使用 Git 仓库管理打包文件，遵循特定的分支规范：

- **`upstream/` 标签**：指向上游发布的原始源码（如 `upstream/0.7.0`），用于生成 `.orig.tar.gz`。
- **`packaging/openkylin/<series>` 分支**：存放 `debian/` 打包目录，采用 quilt 格式，所有对上游的修改以 patch 形式保存， `<series>` 对应 OpenKylin 的版本代号：
  - `nile-sp2` —— OpenKylin 2.0 SP2
  - `huanghe` —— OpenKylin 3.0
- **`openkylin/<series>` 分支**：开发分支，包含 `debian/` 目录，源码包格式为 native。

### 打包步骤

1. **拉取上游新版本并创建 `upstream/` 标签**：

   下游仓库使用 `upstream/X.Y.Z` 命名（无 `v` 前缀）指向上游 `vX.Y.Z` 标签，供 CI 生成 `.orig.tar.gz`：

   ```bash
   git fetch upstream --tags
   git tag upstream/X.Y.Z vX.Y.Z
   ```

2. **合并上游新版本到打包分支**：

   打包分支应通过**合并**而非重置来引入上游新版本，以保留已有的打包提交历史。OpenKylin 官方指南同样要求新上游版本合并进 Debian 分支：

   ```bash
   # 切换到打包分支（以 OpenKylin 3.0 Huanghe 为例）
   git checkout packaging/openkylin/huanghe
   # 合并新的上游标签，保留打包分支中已有的 debian/ 目录
   git merge upstream/X.Y.Z
   ```

   若合并产生冲突，仅需解决源码冲突；`debian/` 目录应保留打包分支当前版本。合并完成后工作区即包含新上游源码与现有 `debian/` 目录。

3. **更新 `debian/changelog`**：

   ```bash
   dch -v X.Y.Z-okN --distribution huanghe "New upstream release X.Y.Z"
   ```

   其中 `okN` 为 OpenKylin 的打包修订号，`distribution` 为目标发行版系列（如 `huanghe`、`nile-sp2`）。

4. **测试构建**：

   ```bash
   # 确保构建依赖已安装
   sudo apt build-dep .
   # 在本地进行测试构建
   debuild -us -uc -b
   ```

   确保二进制包能正常生成。注意：测试构建生成的文件绝对不应该被提交到 Git 仓库，提交并推送前需确保工作区干净。

5. **提交并推送分支与 `upstream/` 标签**：

   ```bash
   git add debian
   git commit -m "Packaging vX.Y.Z for huanghe"
   # 推送打包分支
   git push origin packaging/openkylin/huanghe
   # 推送新的 upstream 标签，供远端 CI 生成 .orig.tar.gz
   git push origin upstream/X.Y.Z
   ```

### CI 与发布流程

推送后，OpenKylin 的 CI 系统（Jenkins + gbp-bot）会自动触发：

1. **源码包构建**：CI 从打包分支生成 `.dsc` 和 `.orig.tar.gz`，并上传至 OKBS（OpenKylin 编译平台）。
2. **进入 `proposed` 仓库**：构建成功的源码包会进入对应发行版的 `proposed` 仓库，供测试验证。
3. **发布到 `release` 仓库**：经测试验证后，由社区维护者手动或自动将其从 `proposed` 迁移至 `release` 仓库，正式面向所有用户。

## 维护 AUR package

AUR（Arch User Repository）上维护了三个包，均由维护者 `moyigeek`（`moyi@openatom.club`）负责：

- [atomgit-cli](https://aur.archlinux.org/packages/atomgit-cli)：稳定版，从上游源码构建。
- [atomgit-cli-bin](https://aur.archlinux.org/packages/atomgit-cli-bin)：稳定版，直接使用 Release 预编译二进制。
- [atomgit-cli-git](https://aur.archlinux.org/packages/atomgit-cli-git)：开发版，跟随上游 `main` 分支最新提交。

三个包共用 `provides=('ag')`，并互相声明 `conflicts`，避免同时安装冲突。arch 覆盖 `x86_64`、`aarch64`、`loong64`。

### 稳定版（atomgit-cli / atomgit-cli-bin）

新版本发布后，更新对应 AUR 仓库的 PKGBUILD：

- **atomgit-cli**：将 `pkgver` 更新为新版本号，并**必须**把 `_commit` 更新为新 tag `v${pkgver}` 指向的 commit，同时确认源码 `git+...#tag=v${pkgver}` 已指向同一 tag。PKGBUILD 会无条件把 `_commit` 注入 `internal/version.Commit`；若仅更新 `pkgver` 而漏更 `_commit`，新 tag 的源码仍可正常构建，但 `ag version` 会继续报告上一个版本的 SHA。注意：本地 AUR 仓库（如 `~/atomgit-cli`）不包含上游 tag，在其中执行 `git rev-list -n 1 "v${pkgver}"` 会报 `unknown revision`（exit 128）。请改用以下任一方式在新 tag 指向的上游仓库中取得目标 commit 后再写入 `_commit`：
  ```bash
  # 方式 A：远程查询（可在任意目录执行，无需克隆）
  git ls-remote https://atomgit.com/hust-open-atom-club/atomgit-cli.git "refs/tags/v${pkgver}" \
    | awk '{print $1}'
  # 方式 B：在上游 atomgit-cli 检出中执行
  git rev-list -n 1 "v${pkgver}"
  ```
- **atomgit-cli-bin**：将 `pkgver` 更新为新版本号，并更新各架构归档的下载 URL 和对应的 `_sha256sums_*`。

维护流程：

```bash
# 在本地 AUR 仓库中
cd atomgit-cli      # 或 atomgit-cli-bin
# 编辑 PKGBUILD 更新版本号
makepkg --printsrcinfo > .SRCINFO   # 重新生成 .SRCINFO
git add PKGBUILD .SRCINFO
git commit -m "atomgit-cli X.Y.Z-1" # 包名替换为对应 pkgname
git push                            # 推送到 AUR
```

> [!NOTE]
> `-pkgrel`（包发布修订号）：当上游版本号不变、但打包方式有改动时递增（如 `0.7.2-1` → `0.7.2-2`）。版本号本身更新时通常重置回 `-1`。

### 开发版（atomgit-cli-git）

跟随 `main` 分支，无需随每个版本手动更新。PKGBUILD 的 `pkgver()` 函数用 `git describe --long --tags` 生成规范版本号（如 `0.7.2.r27.gdb67692`），并在 `build()` 中注入对应 commit 与构建日期。只有当上游仓库或打包配置有结构性改动时才需要更新提交。

### 校验

推送前应在本机用 `makepkg -f` 实际构建一次，确认能产出 `.pkg.tar.*` 包且 `ag version` 显示的版本符合预期。对于 `atomgit-cli`，还需精确核对 `ag version` 输出的 commit 与新 tag `v${pkgver}` 指向的 commit 完全一致。注意本地 AUR 仓库不包含上游 tag，不能在其中查询；请使用 `git ls-remote https://atomgit.com/hust-open-atom-club/atomgit-cli.git "refs/tags/v${pkgver}" | awk '{print $1}'`（任意目录），或在上游 atomgit-cli 检出中执行 `git rev-list -n 1 "v${pkgver}"` 取得参照值。若 `ag version` 仍显示上一个版本的 SHA，说明 `_commit` 未同步更新，必须修正后重新构建、再次校验通过，再推送 AUR。
