# AtomGit CLI (ag)

AtomGit 命令行工具，参考 GitHub CLI (gh) 开发。

## 安装

### npm

需要 Node.js 18 或更高版本：

```bash
npm install --global @hust-open-atom-club/atomgit-cli
```

npm 会通过当前操作系统和 CPU 对应的可选依赖安装预编译二进制，不需要运行 `postinstall` 脚本。

### Homebrew

macOS 或 Linux 用户可以通过项目维护的 Homebrew tap 安装：

```bash
brew install hust-open-atom-club/tap/atomgit-cli
```


### WinGet

Windows 用户可以通过 Windows Package Manager 安装：

```powershell
winget install hust-open-atom-club.atomgit-cli
```

升级已安装的 AtomGit CLI：

```powershell
winget upgrade atomgit-cli
```

WinGet 安装的二进制由 WinGet 管理升级。

### Scoop

Windows 用户可以通过组织维护的 scoop bucket 安装：

```powershell
scoop install https://raw.githubusercontent.com/hust-open-atom-club/ScoopBucket/main/bucket/atomgit-cli.json
```

### Nix / NixOS

AtomGit CLI 已进入 `nixos-unstable`，包名为 `atomgit-cli`，安装后提供 `ag` 命令。

如果你的 Nix registry 或 flake input 已指向 `nixos-unstable`：

```bash
nix profile install nixpkgs#atomgit-cli
```

如果你使用的是稳定版 nixpkgs，可以显式从 `nixos-unstable` 安装：

```bash
nix profile install github:NixOS/nixpkgs/nixos-unstable#atomgit-cli
```

### Go

需要 Go 1.24.2 或更高版本。项目正式支持 macOS、Linux 和 Windows；其他 Go 目标平台的兼容性不作保证：

```bash
go install atomgit.com/hust-open-atom-club/atomgit-cli/cmd/ag@latest
```

**注意：** 通过 Go 模块代理安装时，`ag version` 能显示模块版本，但由于模块代理提供的源码包不包含 Git 历史，文本输出会省略无法获得的 commit 和构建时间；`ag version --json` 中对应字段为 `unknown`。这不影响 CLI 功能；如需完整的版本元数据，请使用 npm、Homebrew 或 Release 页面提供的预编译版本。

其他安装方式请参阅[完整安装指南](docs/installation.md)。

## 配置

首次使用时运行 `ag auth login` 完成 OAuth 登录。访问令牌、手动配置、输出安全和仓库推断规则请参阅[配置指南](docs/configuration.md)。

## 命令

完成认证后，可以通过 `ag` 管理 AtomGit 仓库、Issue、Pull Request、Release 和其他资源：

```bash
ag auth status
ag repo view
ag issue list
ag pr list
ag check-update
ag org list
ag --help
```

各命令的参数、行为说明和完整示例请参阅[命令使用指南](docs/usage.md)。

## 发布打包

GoReleaser、npm 制品和 Nix package 的发布与维护流程请参阅[发布指南](docs/releasing.md)。

## 项目结构

CLI 的主要执行流程为 `cmd/ag` → `internal/agcmd` → `pkg/cmd/root` → 各命令包。目录职责、命令包列表和扩展方式请参阅[项目结构说明](docs/project-structure.md)。

## API

常规仓库功能使用 AtomGit API v5：`https://api.atomgit.com/api/v5`。

Actions 运行检查使用独立的 AtomGit API v8：`https://api.atomgit.com/api/v8`。

## 参考

- [AtomGit API 文档](https://docs.atomgit.com/docs/apis/)
- [GitHub CLI](https://cli.github.com/)

## License

[木兰宽松许可证第2版](LICENSE) (Mulan Permissive Software License, Version 2)

Copyright (c) 2026 HUST OpenAtom Club, AtomGit, and the AtomGit CLI contributors
