# 安装指南

AtomGit CLI 支持 macOS、Linux 和 Windows，可通过 npm、Homebrew、WinGet、Scoop、Nix、AUR 或 Go 安装，也可以通过 AtomGit Release 自动或手动安装，或从源码构建。

## 目录

- [npm 安装](#npm-安装)
- [Homebrew 安装](#homebrew-安装)
- [WinGet 安装](#winget-安装)
- [Scoop 安装](#scoop-安装)
- [Nix / NixOS 安装](#nix--nixos-安装)
- [AUR / Arch Linux 安装](#aur--arch-linux-安装)
- [OpenKylin 安装](#openkylin-安装)
- [Go 安装](#go-安装)
- [AtomGit Release 安装](#atomgit-release-安装)
- [源码安装](#源码安装)
- [安装验证](#安装验证)

## npm 安装

npm 安装需要 Node.js 18 或更高版本。执行：

```bash
npm install -g @hust-open-atom-club/atomgit-cli
```

npm 主包通过 `optionalDependencies` 声明七个平台二进制包。npm 根据当前操作系统和 CPU 架构只安装匹配的包，整个过程不使用 `postinstall`，运行 `ag` 时也不会额外联网下载或写入包目录。

请勿使用 `--omit=optional` 安装；该选项会跳过平台二进制包，使 `ag` 无法启动。

| 操作系统 | 支持的处理器架构 |
| --- | --- |
| macOS | x64（Intel）、arm64（Apple Silicon） |
| Linux | x64 / amd64、arm64 / aarch64、loong64 / loongarch64 |
| Windows | x64 / amd64、arm64 |

升级全局 npm 安装的 AtomGit CLI，可直接让 `ag` 识别当前二进制来源并调用 npm：

```bash
ag update
```

发现新版本后，命令会让用户选择 `Update via npm` 或仅本次 `Skip`；直接回车或标准输入 EOF 时默认更新。选择更新时会先确认目标版本已经发布到官方 npm registry，更新后再执行 npm 生成的 `ag` 命令入口核对版本，而不只检查平台包中的二进制。Windows 上如果 npm 因正在运行的 `ag.exe` 被锁定而破坏入口，或者 npm 报告成功但入口仍是旧版本，命令会下载 AtomGit Release 中校验和匹配的 Windows 二进制修复该入口。修复后的入口不再由 npm 管理；以后可重新执行下面的 npm 安装命令恢复 npm 管理。

也可以手动升级：

```bash
npm update -g @hust-open-atom-club/atomgit-cli
```

卸载：

```bash
npm uninstall -g @hust-open-atom-club/atomgit-cli
```

## Homebrew 安装

macOS 或 Linux 用户可以通过 Homebrew Core 安装稳定版，也可以通过项目维护的 Homebrew Tap 跟踪开发快照。

### Homebrew Core（稳定版，推荐）

Homebrew Core 中的 `atomgit-cli` 跟随项目正式发布的稳定版本，适合日常使用和自动化环境。安装时执行：

```bash
brew install atomgit-cli
```

升级 Homebrew Core 版本，可直接让 `ag` 识别 Formula 来源、调用 Homebrew 并验证新版本：

```bash
ag update
```

发现新版本后可选择 `Update via Homebrew Core` 或仅本次 `Skip`；直接回车或标准输入 EOF 时默认更新。

也可以手动升级：

```bash
brew update
brew upgrade atomgit-cli
```

Homebrew Core Formula 使用固定的稳定版源码；目标平台存在 bottle 时，Homebrew 会优先安装 bottle，否则会按 Formula 从源码构建。项目发布新版本后，Homebrew Core 的版本可能需要等待 Formula 更新完成。

### 项目 Homebrew Tap（开发快照）

项目维护的 [`hust-open-atom-club/tap`](https://github.com/hust-open-atom-club/homebrew-tap) 跟踪 AtomGit `main` 分支的最新 commit，适合提前测试尚未进入稳定版的修复和功能。安装时使用完整 Formula 名称，以便与 Homebrew Core 明确区分：

```bash
brew install hust-open-atom-club/tap/atomgit-cli
```

也可以先显式添加 Tap，再安装：

```bash
brew tap hust-open-atom-club/tap
brew install hust-open-atom-club/tap/atomgit-cli
```

升级项目 Tap 版本仍使用完整 Formula 名称；当前 `ag update` 不处理项目 Tap，也不会将其当成 Homebrew Core：

```bash
brew update
brew upgrade hust-open-atom-club/tap/atomgit-cli
```

Tap Formula 会固定到一个不可变的 AtomGit commit，并从源码构建。自动化任务每小时检查一次 `main`；发现新 commit 后，只有 Formula 在 macOS 和 Linux 上通过测试，更新才会合并。其开发版本号包含稳定版基线、commit 时间、必要时使用的同秒碰撞计数器和缩写 SHA。

### Homebrew Core 与项目 Tap 的区别

| 项目 | Homebrew Core | 项目 Homebrew Tap |
| --- | --- | --- |
| 更新目标 | 最新正式稳定版 | AtomGit `main` 最新固定 commit |
| 更新节奏 | 随 Homebrew Core Formula 更新 | 每小时检查，通过 macOS/Linux 测试后更新 |
| 稳定性 | 推荐日常使用 | 开发快照，可能包含尚未发布的改动 |
| 安装命令 | `brew install atomgit-cli` | `brew install hust-open-atom-club/tap/atomgit-cli` |

两个渠道提供的 Formula 名称均为 `atomgit-cli`，Homebrew 不允许同时安装。若要从 Homebrew Core 切换到项目 Tap：

```bash
brew uninstall atomgit-cli
brew install hust-open-atom-club/tap/atomgit-cli
```

若要从项目 Tap 切换回 Homebrew Core：

```bash
brew uninstall hust-open-atom-club/tap/atomgit-cli
brew untap hust-open-atom-club/tap
brew install atomgit-cli
```

无论使用哪个渠道，已安装的二进制都由 Homebrew 管理。可以通过对应的完整 Formula 名称运行 `brew info`，确认准备安装或升级的版本与来源。

卸载：

```bash
brew uninstall atomgit-cli
```

## WinGet 安装

Windows 用户可以通过 Windows Package Manager 安装：

```powershell
winget install HUSTOpenAtomClub.AtomGitCLI
```

升级已安装的 AtomGit CLI：

```powershell
winget upgrade HUSTOpenAtomClub.AtomGitCLI
```

WinGet 安装的二进制由 WinGet 管理升级。

卸载：

```powershell
winget uninstall HUSTOpenAtomClub.AtomGitCLI
```

## Scoop 安装

Windows 用户可以通过组织维护的 scoop bucket 安装：

```powershell
scoop bucket add hust-open-atom-club https://github.com/hust-open-atom-club/ScoopBucket
scoop install atomgit-cli
```

升级已安装的 AtomGit CLI：

```powershell
scoop update atomgit-cli
```

卸载：

```powershell
scoop uninstall atomgit-cli
```

## Nix / NixOS 安装

AtomGit CLI 已进入 `nixos-unstable`，nixpkgs 包名为 `atomgit-cli`，安装后提供 `ag` 命令。

如果你的 Nix registry 或 flake input 已指向 `nixos-unstable`，可以直接安装：

```bash
nix profile install nixpkgs#atomgit-cli
```

如果你使用的是稳定版 nixpkgs，可以显式从 `nixos-unstable` 安装：

```bash
nix profile install github:NixOS/nixpkgs/nixos-unstable#atomgit-cli
```

也可以在不安装的情况下临时运行：

```bash
nix run github:NixOS/nixpkgs/nixos-unstable#atomgit-cli -- version
```

升级通过 Nix profile 安装的 AtomGit CLI：

```bash
nix profile upgrade atomgit-cli
```

卸载通过 Nix profile 安装的 AtomGit CLI：

```bash
nix profile remove atomgit-cli
```

如果 profile 中显示的名称不同，可先运行 `nix profile list` 确认名称，再将其传给 `nix profile upgrade` 或 `nix profile remove`。

如果通过 Home Manager 或 NixOS 配置安装，请从 `home.packages` 或 `environment.systemPackages` 中移除对应条目，然后重新应用配置。

### 使用 Home Manager 或 NixOS 安装

如果系统本身已经使用 `nixos-unstable`，可以直接把 `pkgs.atomgit-cli` 加入系统或用户环境：

```nix
{
  environment.systemPackages = [
    pkgs.atomgit-cli
  ];
}
```

Home Manager：

```nix
{
  home.packages = [
    pkgs.atomgit-cli
  ];
}
```

如果系统使用稳定版 nixpkgs，可以额外引入 `nixos-unstable`，只从 unstable 安装 AtomGit CLI。先在 flake inputs 中加入：

```nix
{
  inputs.nixpkgs-unstable.url = "github:NixOS/nixpkgs/nixos-unstable";
}
```

确保 NixOS 的 `specialArgs` 或 Home Manager 的 `extraSpecialArgs` 已传入 `inputs`，然后在 module 中使用：

```nix
{ pkgs, inputs, ... }:

let
  unstable = import inputs.nixpkgs-unstable {
    system = pkgs.stdenv.hostPlatform.system;
  };
in
{
  environment.systemPackages = [
    unstable.atomgit-cli
  ];
}
```

Home Manager 中同样可以将 `unstable.atomgit-cli` 加入 `home.packages`。

### 使用本仓库 flake

仓库仍提供支持 Linux 和 macOS（x86_64、aarch64）的 Nix flake，并包含两种 package：

- `stable`：从固定版本的 AtomGit Release 源码归档构建。
- `latest`：从当前 flake revision 的源码构建。

默认 package 和兼容名称 `ag` 均指向 `stable`。安装稳定版：

由于 nixos-unstable 已停止支持 Intel macOS，flake 对 `x86_64-darwin` 使用仍受维护的 `nixpkgs-26.05-darwin` input；其他平台继续使用 nixos-unstable。

```bash
nix profile install git+https://atomgit.com/hust-open-atom-club/atomgit-cli#ag
```

跟随仓库 revision 安装 latest：

```bash
nix profile install git+https://atomgit.com/hust-open-atom-club/atomgit-cli#latest
```

也可以在不安装的情况下直接运行任一 package：

```bash
nix run git+https://atomgit.com/hust-open-atom-club/atomgit-cli#stable -- version
nix run git+https://atomgit.com/hust-open-atom-club/atomgit-cli#latest -- version
```

以下是使用本仓库 stable package 的完整配置示例。`pkgs` 由 `nixpkgs` 为目标系统实例化，并通过 `pkgs.stdenv.hostPlatform.system` 选择对应的 AtomGit CLI package：

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    atomgit-cli = {
      url = "git+https://atomgit.com/hust-open-atom-club/atomgit-cli";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { nixpkgs, atomgit-cli, ... }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
      ag = atomgit-cli.packages.${pkgs.stdenv.hostPlatform.system}.stable;
    in
    {
      # Home Manager 模块
      homeManagerModules.default = {
        home.packages = [ ag ];
      };

      # NixOS 模块
      nixosModules.default = {
        environment.systemPackages = [ ag ];
      };
    };
}
```

## AUR / Arch Linux 安装

AUR 是用户仓库，并非 Arch 官方仓库；安装与使用按 AUR 规则自行承担风险。三个包由项目维护者 `moyigeek` 在 AUR 维护并随 Release 更新。如果使用 Arch Linux 或基于 Arch Linux 的发行版，可以直接使用 AUR 仓库中的包进行安装。AUR 中提供了三个包，可按需选择：

```bash
# 二进制安装（免编译，稳定版）
yay -S atomgit-cli-bin
# 从源码安装（稳定版）
yay -S atomgit-cli
# 开发版安装（跟随 main 分支最新提交）
yay -S atomgit-cli-git
```

三个包共用 `provides=('ag')` 并互相声明 `conflicts`，同时安装会冲突；支持的架构覆盖 `x86_64`、`aarch64` 和 `loong64`。

升级随 AUR 助手的常规更新一起完成，例如 `yay -Syu`。卸载使用 `yay -R atomgit-cli-bin`（替换为实际安装的包名）。

## Go 安装

Go 安装将自动下载源码包并进行编译，需要 Go 1.26.6 或更高版本：

```bash
go install atomgit.com/hust-open-atom-club/atomgit-cli/cmd/ag@latest
```

该方法将从 Go 模块代理下载该项目及其依赖的源码包，并在本机构建出二进制文件。项目正式支持并测试 macOS、Linux 和 Windows；其他 Go 目标平台可能能够编译，但不保证完整兼容。`ag auth login` 的浏览器 OAuth 流程仅支持上述三个操作系统，其他平台需要在功能可用的前提下手动配置 PAT。

升级通过 Go 安装的 AtomGit CLI 时，重新执行相同命令即可：

```bash
go install atomgit.com/hust-open-atom-club/atomgit-cli/cmd/ag@latest
```

**注意：** Go 模块代理提供的源码包不包含 `.git` 目录，因此这种安装方式构建的二进制只能可靠获得模块版本。执行 `ag version` 时，文本输出会省略无法获得的 commit 和构建时间；`ag version --json` 中对应字段为 `unknown`。这是预期行为，不影响 CLI 功能。如需同时包含版本、commit 和构建时间，请改用其他安装方法（如 npm、Homebrew），或通过 AtomGit Release 安装预编译版本。

Go 本身不记录通过 `go install` 安装的软件包。卸载时，删除 `GOBIN` 中的 `ag` 或 `ag.exe`；未设置 `GOBIN` 时，对应文件位于 `$(go env GOPATH)/bin`。

## OpenKylin 安装

AtomGit CLI 已正式进入 OpenKylin 软件仓库，支持 OpenKylin 2.0 SP2（nile-sp2）及 OpenKylin 3.0（huanghe）。最新版本首先进入 `proposed` 仓库，经测试验证后推送至 `release` 仓库。

### 从 release 仓库安装（稳定版）

```bash
sudo apt update
sudo apt install atomgit-cli
```

### 从 proposed 仓库安装（最新版）

如需安装 `proposed` 仓库中的最新版本，需先添加 proposed 源。OpenKylin 2.0 SP2（nile）使用 `nile.bedrock-proposed`，OpenKylin 3.0（huanghe）使用 `huanghe-proposed`。将对应源写入 `/etc/apt/sources.list.d/` 下的一个文件：

```bash
# OpenKylin 2.0 SP2 (nile)
echo "deb http://archive.build.openkylin.top/openkylin nile.bedrock-proposed main cross pty" | sudo tee /etc/apt/sources.list.d/openkylin-proposed.list

# OpenKylin 3.0 (huanghe)
echo "deb http://archive.build.openkylin.top/openkylin huanghe-proposed main cross pty" | sudo tee /etc/apt/sources.list.d/openkylin-proposed.list
```

然后更新并安装：

```bash
sudo apt update
sudo apt install atomgit-cli
```

安装完成后，如不再需要 proposed 源，可删除该文件以避免后续收到不稳定更新：

```bash
sudo rm /etc/apt/sources.list.d/openkylin-proposed.list
sudo apt update
```

也可以直接编辑 `/etc/apt/sources.list.d/openkylin-proposed.list`，注释对应行后执行 `sudo apt update`。

### 升级

```bash
sudo apt update
sudo apt upgrade atomgit-cli
```

### 卸载

```bash
sudo apt remove atomgit-cli
```

如需同时删除配置文件：

```bash
sudo apt purge atomgit-cli
```

## AtomGit Release 安装

### 自动安装

安装脚本会识别当前操作系统（支持 Linux、Windows 和 macOS）与处理器架构（Linux 支持 amd64、arm64 和 loong64，macOS 与 Windows 支持 amd64 和 arm64），下载匹配的预编译文件并安装 `ag`。

#### macOS 和 Linux

请在终端执行：

```bash
curl -fsSL "https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/download/latest/install.sh" | sh
```

脚本默认将 `ag` 安装到 `/usr/local/bin`；该目录不可写时，会改用 `~/.local/bin`。如果安装目录不在 `PATH` 中，脚本会输出相应的配置提示。

#### Windows

请在 Windows PowerShell 5.1 或 PowerShell 7+ 中执行：

```powershell
irm "https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/download/latest/install.ps1" | iex
```

`irm` 是 `Invoke-RestMethod` 的别名。若当前环境不支持该别名，可执行：

```powershell
Invoke-RestMethod -Uri "https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/download/latest/install.ps1" | Invoke-Expression
```

如果系统禁止运行脚本，可先为当前用户设置执行策略（只需执行一次）：

```powershell
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
```

脚本默认安装到 `%USERPROFILE%\.local\bin`，并将该目录加入当前用户的 `Path`。

`install.ps1` 保持纯 ASCII，以兼容 Windows PowerShell 5.1 在不同系统代码页下的脚本下载、保存和执行；安装过程中的提示信息使用英文。

### 手动安装

从 [Release 页面](https://atomgit.com/hust-open-atom-club/atomgit-cli/releases)下载与操作系统和处理器架构匹配的文件：

| 操作系统 | 处理器架构 | 安装包 |
| --- | --- | --- |
| macOS | Apple Silicon（arm64） | `ag_darwin_arm64.tar.gz` |
| macOS | Intel（amd64） | `ag_darwin_amd64.tar.gz` |
| Linux | arm64 / aarch64 | `ag_linux_arm64.tar.gz` |
| Linux | LoongArch64（loong64 / loongarch64） | `ag_linux_loong64.tar.gz` |
| Linux | x86-64 / amd64 | `ag_linux_amd64.tar.gz` |
| Windows | ARM64 | `ag_windows_arm64.zip` |
| Windows | 常见 64 位 PC（amd64） | `ag_windows_amd64.zip` |

macOS 和 Linux 可执行 `uname -m` 查看架构：`arm64` 或 `aarch64` 对应 arm64，`x86_64` 对应 amd64，Linux 上的 `loongarch64` 或 `loong64` 对应 loong64。Windows 可在 PowerShell 中执行 `$env:PROCESSOR_ARCHITECTURE`；`ARM64` 对应 arm64，`AMD64` 对应 amd64。

#### macOS 和 Linux

1. 解压下载的 `.tar.gz` 文件，得到可执行文件 `ag`。例如：

   ```bash
   tar -xzf ag_darwin_arm64.tar.gz
   ```

   请根据实际下载的文件名调整命令。

2. 将 `ag` 安装到已加入 `PATH` 的目录，并赋予执行权限。例如：

   ```bash
   mkdir -p "$HOME/.local/bin"
   install -m 0755 ag "$HOME/.local/bin/ag"
   ```

3. 如果 `~/.local/bin` 尚未加入 `PATH`，将下面一行添加到 shell 配置文件（例如 `~/.zshrc` 或 `~/.bashrc`），然后重新打开终端：

   ```bash
   export PATH="$HOME/.local/bin:$PATH"
   ```

#### Windows

1. 解压下载的 `.zip` 文件，得到 `ag.exe`。
2. 将 `ag.exe` 放入固定目录，例如 `C:\Users\<用户名>\.local\bin`。
3. 打开系统“环境变量”，编辑当前用户的 `Path`，添加上述目录。
4. 新开 PowerShell 或命令提示符。

## 源码安装

源码构建支持 macOS、Linux 和 Windows，需要安装：

- [Git](https://git-scm.com/downloads)
- [Go](https://go.dev/dl/) 1.26.6 或更高版本
- `make`（仅 macOS 和 Linux 使用 Makefile 时需要）

`go.mod` 的 `go` 行规定源码构建最低支持 Go 1.26.6，`toolchain` 行建议开发和正式
发布使用 Go 1.26.8。直接运行 Go 命令时可使用任意不低于最低版本的兼容工具链；普通
Make 目标会选择建议的精确版本。如果本机尚未缓存对应工具链，首次构建会按照
`GOPROXY` 下载，并通过 `GOSUMDB` 配置的校验和数据库验证，因此需要访问相应服务。
开始较长的构建前，可运行 `make go-version` 下载并确认正式发布工具链；维护者还应
运行 `make go-min-version test-min-go` 验证最低版本兼容性。

克隆仓库：

```bash
git clone https://atomgit.com/hust-open-atom-club/atomgit-cli.git
cd atomgit-cli
```

默认构建当前开发版本。如需构建特定发布版本，请将 `vX.Y.Z` 替换为对应的 Release tag：

```bash
git checkout vX.Y.Z
```

### macOS 和 Linux

构建到 `bin/ag`：

```bash
make build
```

也可以安装到 `GOBIN` 指定的目录；未设置 `GOBIN` 时，默认目录为 `$(go env GOPATH)/bin`：

```bash
make install
```

### Windows

在 PowerShell 中构建 `ag.exe`：

```powershell
go build -trimpath -o ag.exe ./cmd/ag
```

将生成的 `ag.exe` 放入已加入用户 `Path` 的目录。

也可以在 PowerShell 中执行：

```powershell
go install ./cmd/ag
```

可执行文件会安装到 `GOBIN` 指定的目录；未设置 `GOBIN` 时，默认目录为 `go env GOPATH` 所显示目录下的 `bin` 子目录。请确保该目录已加入 `PATH`。

## 安装验证

完成安装或构建后，新开一个终端并执行：

```bash
ag version
```
