# 安装指南

AtomGit CLI 支持 macOS、Linux 和 Windows，可通过自动安装脚本、手动下载预编译文件或从源码构建进行安装。

## 自动化安装（推荐）

安装脚本会识别当前操作系统（支持 Linux，Windows 和 macOS）与处理器架构（支持 amd64 和 arm64），下载匹配的预编译文件并安装 `ag`。

### macOS 和 Linux

请在终端执行：

```bash
curl -fsSL "https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/latest/download/install.sh" | sh
```

脚本默认将 `ag` 安装到 `/usr/local/bin`；该目录不可写时，会改用 `~/.local/bin`。如果安装目录不在 `PATH` 中，脚本会输出相应的配置提示。

### Windows

请在 Windows PowerShell 5.1 或 PowerShell 7+ 中执行：

```powershell
irm "https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/latest/download/install.ps1" | iex
```

`irm` 是 `Invoke-RestMethod` 的别名。若当前环境不支持该别名，可执行：

```powershell
Invoke-RestMethod -Uri "https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/latest/download/install.ps1" | Invoke-Expression
```

如果系统禁止运行脚本，可先为当前用户设置执行策略（只需执行一次）：

```powershell
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
```

脚本默认安装到 `%USERPROFILE%\.local\bin`，并将该目录加入当前用户的 `Path`。

## 手动安装

从 [Release 页面](https://atomgit.com/hust-open-atom-club/atomgit-cli/releases)下载与操作系统和处理器架构匹配的文件：

| 操作系统 | 处理器架构 | 安装包 |
| --- | --- | --- |
| macOS | Apple Silicon（arm64） | `ag_darwin_arm64.tar.gz` |
| macOS | Intel（amd64） | `ag_darwin_amd64.tar.gz` |
| Linux | arm64 / aarch64 | `ag_linux_arm64.tar.gz` |
| Linux | x86-64 / amd64 | `ag_linux_amd64.tar.gz` |
| Windows | ARM64 | `ag_windows_arm64.zip` |
| Windows | 常见 64 位 PC（amd64） | `ag_windows_amd64.zip` |

macOS 和 Linux 可执行 `uname -m` 查看架构：`arm64` 或 `aarch64` 对应 arm64，`x86_64` 对应 amd64。Windows 可在 PowerShell 中执行 `$env:PROCESSOR_ARCHITECTURE`；`ARM64` 对应 arm64，`AMD64` 对应 amd64。

### macOS 和 Linux

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

### Windows

1. 解压下载的 `.zip` 文件，得到 `ag.exe`。
2. 将 `ag.exe` 放入固定目录，例如 `C:\Users\<用户名>\.local\bin`。
3. 打开系统“环境变量”，编辑当前用户的 `Path`，添加上述目录。
4. 新开 PowerShell 或命令提示符。

## 从源码构建

源码构建支持 macOS、Linux 和 Windows，需要安装：

- [Git](https://git-scm.com/downloads)
- [Go](https://go.dev/dl/) 1.24.2 或更高版本
- `make`（仅 macOS 和 Linux 使用 Makefile 时需要）

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

也可以安装到 `$(go env GOPATH)/bin`：

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

可执行文件会安装到 `go env GOPATH` 所显示目录下的 `bin` 子目录。请确保该目录已加入 `PATH`。

## 验证安装

完成安装或构建后，新开一个终端并执行：

```text
ag --help
```
