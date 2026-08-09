# AtomGit CLI (ag)

[![License](https://img.shields.io/badge/license-MulanPSL--2.0-blue.svg)](LICENSE)
[![npm](https://img.shields.io/npm/v/%40hust-open-atom-club%2Fatomgit-cli?logo=npm)](https://www.npmjs.com/package/@hust-open-atom-club/atomgit-cli)
[![Latest Release](https://img.shields.io/github/v/release/hust-open-atom-club/atomgit-cli?display_name=tag)](https://atomgit.com/hust-open-atom-club/atomgit-cli/releases)
[![Build](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.atomgit.com%2Fapi%2Fv8%2Frepos%2Fhust-open-atom-club%2Fatomgit-cli%2Factions%2Fruns%3Fworkflow_name%3DCI%26branch%3Dmain%26per_page%3D1&query=%24.workflow_runs%5B0%5D.status&label=build)](https://atomgit.com/hust-open-atom-club/atomgit-cli/actions)
[![Homebrew](https://img.shields.io/badge/homebrew-tap-FBB040?logo=homebrew&logoColor=white)](https://github.com/hust-open-atom-club/homebrew-tap)
[![Go Reference](https://pkg.go.dev/badge/atomgit.com/hust-open-atom-club/atomgit-cli.svg)](https://pkg.go.dev/atomgit.com/hust-open-atom-club/atomgit-cli)
[![GoReleaser](https://img.shields.io/badge/powered_by-GoReleaser-69D7E4?logo=goreleaser&logoColor=white)](https://goreleaser.com/)

AtomGit 命令行工具，参考 GitHub CLI (gh) 开发。

## 功能

| 类别 | 能力 |
| --- | --- |
| 📦 仓库 | 列出、查看、创建、编辑、克隆、删除、复刻和同步仓库 |
| 👥 协作者 | 列出、查看、添加、修改和移除仓库协作者 |
| 🔔 Webhook | 列出、查看、创建、编辑、删除和测试仓库 Webhook |
| 🌿 分支 | 列出、查看、创建和删除分支，管理分支保护规则 |
| 🔀 Pull Request | 列出、查看、创建、编辑、关闭、重开、审查、合并和检出 PR，查看差异与检查结果，管理评论和关联 Issue |
| 🐛 Issue | 列出、查看、创建、编辑、关闭和重开 Issue，管理标签和评论 |
| 🔖 标签 | 列出、创建、编辑和删除仓库标签 |
| 🎯 里程碑 | 列出、查看、创建、编辑、关闭、重开和删除里程碑 |
| 🏷️ Tag | 列出、创建和删除 Git tag |
| 🚀 Release | 列出、查看、创建和编辑 Release，上传和下载附件 |
| ⚙️ Actions | 列出和查看 workflow 运行、job、日志与 artifact，并下载日志和 artifact |
| 🏢 组织 | 列出当前账号加入的组织 |
| 🔍 搜索 | 搜索仓库、用户和 Issue |
| 🔐 认证与 SSH Key | OAuth 登录、刷新和切换账号，查看认证状态，管理 SSH 公钥 |
| 🌐 API | 调用 AtomGit API v5，支持分页、JSON 请求体和 GET、POST、PATCH、PUT、DELETE 方法 |

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
winget install HUSTOpenAtomClub.AtomGitCLI
```

### Scoop

Windows 用户可以通过组织维护的 scoop bucket 安装：

```powershell
scoop bucket add hust-open-atom-club https://github.com/hust-open-atom-club/ScoopBucket
scoop install atomgit-cli
```

### Nix / NixOS

AtomGit CLI 已进入 `nixos-unstable`，包名为 `atomgit-cli`，安装后提供 `ag` 命令。

如果你的 Nix registry 或 flake input 已指向 `nixos-unstable`：

```bash
nix profile install nixpkgs#atomgit-cli
```

### Go

需要 Go 1.24.2 或更高版本。项目正式支持 macOS、Linux 和 Windows；其他 Go 目标平台的兼容性不作保证：

```bash
go install atomgit.com/hust-open-atom-club/atomgit-cli/cmd/ag@latest
```

**注意：** 通过 Go 模块代理安装时，`ag version` 能显示模块版本，但由于模块代理提供的源码包不包含 Git 历史，文本输出会省略无法获得的 commit 和构建时间；`ag version --json` 中对应字段为 `unknown`。这不影响 CLI 功能；如需完整的版本元数据，请使用 npm、Homebrew 或 Release 页面提供的预编译版本。

升级命令和其他安装方式请参阅[完整安装指南](docs/installation.md)。

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

## Contributors

<a href="https://github.com/hust-open-atom-club/atomgit-cli/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=hust-open-atom-club/atomgit-cli" alt="AtomGit CLI contributors" />
</a>
