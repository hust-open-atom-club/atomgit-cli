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

## AI Agent Skills

[AtomGit Skills](https://atomgit.com/hust-open-atom-club/atomgit-skills) 提供由 `ag` 驱动的 Codex Skills，覆盖 Issue、Pull Request、CLI 发布和 GitHub 镜像工作流。安装方式和完整清单见该仓库。

## 安装

### npm

```bash
npm install -g @hust-open-atom-club/atomgit-cli
```

### Homebrew

```bash
brew install hust-open-atom-club/tap/atomgit-cli
```

### WinGet

```powershell
winget install HUSTOpenAtomClub.AtomGitCLI
```

### Scoop

```powershell
scoop bucket add hust-open-atom-club https://github.com/hust-open-atom-club/ScoopBucket
scoop install atomgit-cli
```

### Nix / NixOS

目前仅 `nixos-unstable` 提供 `atomgit-cli`，请确保 `nixpkgs` 指向 unstable；stable 尚未收录。

```bash
nix profile install nixpkgs#atomgit-cli
```

### Go

```bash
go install atomgit.com/hust-open-atom-club/atomgit-cli/cmd/ag@latest
```

安装要求、升级方式、AtomGit Release 和源码安装请参阅[完整安装指南](docs/installation.md)。

## 配置

首次使用时运行 `ag auth login` 完成 OAuth 登录。凭据、输出安全和仓库推断参阅[配置指南](docs/configuration.md)。

## 常用命令

```bash
ag repo view
ag repo collaborator list
ag repo webhook list
ag branch list owner/repo
ag pr list
ag issue list
ag label list
ag milestone list
ag tag list
ag release list
ag run list owner/repo
ag org list
ag search repositories atomgit
ag auth status
ag ssh-key list
ag api /user
ag check-update
ag --help
```

完整命令参阅[使用指南](docs/usage.md)。
安装、认证、使用和故障排查中的常见问题请参阅[常见问题（FAQ）](docs/faq.md)。

## 更多文档

- [发布指南](docs/releasing.md)
- [项目结构](docs/project-structure.md)

## API

常规仓库功能使用 AtomGit API v5：`https://api.atomgit.com/api/v5`。

Actions 运行检查使用独立的 AtomGit API v8：`https://api.atomgit.com/api/v8`。

### 参考

- [AtomGit API 文档](https://docs.atomgit.com/docs/apis/)
- [GitHub CLI](https://cli.github.com/)

## License

[木兰宽松许可证第2版](LICENSE) (Mulan Permissive Software License, Version 2)

Copyright (c) 2026 HUST OpenAtom Club, AtomGit, and the AtomGit CLI contributors

## Contributors

<a href="https://github.com/hust-open-atom-club/atomgit-cli/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=hust-open-atom-club/atomgit-cli" alt="AtomGit CLI contributors" />
</a>
