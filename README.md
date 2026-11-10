# AtomGit CLI (`ag-cli`)

在终端中高效管理 AtomGit 仓库、Issue、Pull Request，查看 Actions 运行状态并发布 Release。
AtomGit CLI 提供丰富的命令行工具和通用 API 封装，便于集成到自动化脚本与 CI/CD 流程中。

[English](README.en.md) · [安装指南](docs/installation.md) · [使用指南](docs/usage.md) · [命令参考](docs/command-reference.md)

[![License](https://img.shields.io/badge/license-MulanPSL--2.0-blue.svg)](LICENSE)
[![npm](https://img.shields.io/npm/v/%40hust-open-atom-club%2Fatomgit-cli?logo=npm)](https://www.npmjs.com/package/@hust-open-atom-club/atomgit-cli)
[![Latest Release](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.atomgit.com%2Fapi%2Fv5%2Frepos%2Fhust-open-atom-club%2Fatomgit-cli%2Freleases%2Flatest&query=%24.tag_name&label=release)](https://atomgit.com/hust-open-atom-club/atomgit-cli/releases)
[![CI Status](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.atomgit.com%2Fapi%2Fv8%2Frepos%2Fhust-open-atom-club%2Fatomgit-cli%2Factions%2Fruns%3Fworkflow_name%3DCI%26branch%3Dmain%26per_page%3D1&query=%24.workflow_runs%5B0%5D.status&label=CI%20status)](https://atomgit.com/hust-open-atom-club/atomgit-cli/actions)

---

## 核心功能概览

| 类别 | 能力 |
| --- | --- |
| 仓库 | 列出、查看、创建、编辑、克隆、删除、复刻和同步仓库，管理仓库级推送规则 |
| 协作者 | 列出、查看、添加、修改和移除仓库协作者 |
| Webhook | 列出、查看、创建、编辑、删除和测试仓库 Webhook |
| 分支 | 列出、查看、创建和删除分支，管理分支保护规则 |
| Pull Request | 列出、查看、创建、编辑、关闭、重开、审查、合并和检出 PR，查看差异与检查结果，管理评论和关联 Issue |
| Issue | 列出、查看、创建、编辑、关闭和重开 Issue，管理标签和评论 |
| 标签 | 列出、创建、编辑和删除仓库标签 |
| 里程碑 | 列出、查看、创建、编辑、关闭、重开和删除里程碑 |
| Tag | 列出、创建和删除 Git tag，管理保护 tag 规则 |
| Release | 列出、查看、创建和编辑 Release，上传和下载附件 |
| Actions | 列出、校验和触发 workflow；查看 workflow 运行、job、日志与 artifact，并下载日志和 artifact |
| 组织 | 列出当前账号加入的组织 |
| 搜索 | 搜索仓库、用户和 Issue |
| Discussion | 列出仓库 Discussion |
| 通知 | 列出仓库通知并标记已读 |
| 认证与 SSH Key | OAuth 登录、刷新和切换账号，查看认证状态，管理 SSH 公钥 |
| API | 调用 AtomGit API v5，支持分页、JSON 请求体和 GET、POST、PATCH、PUT、DELETE 方法 |

---

> [!IMPORTANT]
> **重要变更**
>
> 命令行主命令名已由 `ag` 更改为 **`ag-cli`**，新版本不提供 `ag` 兼容入口。已有的配置与认证凭据保持兼容，升级后无需重新登录。
> 已有用户请参阅[迁移指南](docs/installation.md#从-ag-迁移到-ag-cli)，并及时更新自动化脚本及 CI 配置文件中的调用命令。各安装渠道的更新进度可能不同，升级后请确认命令名。

---

## 快速安装

选择常用的包管理器快速安装：

| 包管理器 | 适用环境 | 安装命令 |
| :--- | :--- | :--- |
| **[npm](docs/installation.md#npm-安装)** | macOS / Linux / Windows，需 Node.js 18+ | `npm install -g @hust-open-atom-club/atomgit-cli` |
| **[Homebrew](docs/installation.md#homebrew-安装)** | macOS / Linux | `brew install atomgit-cli` |
| **[WinGet](docs/installation.md#winget-安装)** | Windows | `winget install HUSTOpenAtomClub.AtomGitCLI` |

有关其他包管理器、安装脚本、手动与源码安装，以及升级、卸载和 Shell 自动补全的说明，请参阅[安装指南](docs/installation.md)。

---

## 快速上手

1. **验证安装**

   ```bash
   ag-cli version
   ```

2. **登录认证**
   通过浏览器完成交互式授权：
   ```bash
   ag-cli auth login
   ```

3. **查看仓库列表**

   ```bash
   ag-cli repo list
   ```

> [!TIP]
> **使用已有访问令牌登录**
>
> 如果已有个人访问令牌（Personal Access Token），可以通过标准输入传入 CLI 完成登录。此方式也适用于无浏览器的服务器和 CI/CD 容器。
> 将访问令牌保存到 `TOKEN` 变量并执行：
> ```bash
> echo "$TOKEN" | ag-cli auth login --with-token
> ```
> 详细的令牌权限要求与管理说明，请参阅[配置指南](docs/configuration.md)。

更多命令帮助可运行 `ag-cli --help` 或特定子命令帮助（如 `ag-cli pr create --help`）查看。Issue 管理、PR 评审和工作流查看等操作示例，请参阅[使用指南](docs/usage.md)。

---

## AI Agent Skills 支持

[AtomGit Skills](https://atomgit.com/hust-open-atom-club/atomgit-skills) 提供了由 `ag-cli` 驱动的 Codex Skills，
覆盖 Issue 管理、PR 处理、CLI 自动发布及 GitHub 镜像同步等工作流。如需安装与配置，请访问该开源仓库。

---

## 文档索引

- 📦 **[安装指南](docs/installation.md)** — 涵盖各平台安装、升级、卸载及 Shell 自动补全配置。
- ⚙️ **[配置指南](docs/configuration.md)** — 身份认证、凭据安全、输出安全与仓库上下文推断。
- 📖 **[使用指南](docs/usage.md)** — 提供常见开发场景的操作示例。
- 📑 **[命令参考](docs/command-reference.md)** — 详细的命令列表、参数与默认值说明（基于 Cobra 自动生成）。
- ❓ **[常见问题（FAQ）](docs/faq.md)** — 解答使用过程中的常见错误与疑问。
- 🤝 **[贡献指南](CONTRIBUTING.md)** — 包含[项目结构](docs/project-structure.md)与[版本发布流程](docs/releasing.md)。
- 🔗 **[OpenAPI 覆盖清单](docs/openapi-coverage.md)** — 核对与 AtomGit 官方 API 的对应关系、已实现范围及已知缺口。

> [!CAUTION]
> **安全漏洞报告**
>
> 如发现安全漏洞，请按[安全策略](SECURITY.md)进行非公开报告，请勿在公开 Issue 或 PR 中披露。

---

## 参考与致谢

AtomGit CLI 常规功能基于 **AtomGit API v5** (`https://api.atomgit.com/api/v5`) 构建，Actions 模块基于 **API v8** (`https://api.atomgit.com/api/v8`)。

- [AtomGit 官方 API 文档](https://docs.atomgit.com/docs/apis/)
- [GitHub CLI](https://cli.github.com/)（本项目的交互与架构设计参考）

---

## 开源许可

本项目采用 [**木兰宽松许可证第 2 版（Mulan PSL v2）**](LICENSE) 开源发布。

Copyright (c) 2026 HUST OpenAtom Club, AtomGit, and the AtomGit CLI contributors
