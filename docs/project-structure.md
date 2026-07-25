# 项目结构

本文档介绍 AtomGit CLI 的代码组织、主要执行流程和各目录职责。命令的使用方法请参阅[命令使用指南](usage.md)，安装方法请参阅[安装指南](installation.md)。

## 执行流程

CLI 的主要执行路径如下：

```text
cmd/ag/main.go
  └── internal/agcmd.Main
      └── pkg/cmd/root.NewCmdRoot
          └── pkg/cmd/<name>.NewCmdXxx
              ├── internal/api
              ├── internal/config
              └── pkg/cmdutil
```

- `cmd/ag` 只负责可执行程序入口。
- `internal/agcmd` 初始化根命令、执行命令并处理顶层错误和退出码。
- `pkg/cmd/root` 创建 Cobra 根命令并注册各子命令。
- `pkg/cmd/<name>` 实现具体命令、参数校验和输出。
- `internal/api`、`internal/config` 等内部包提供底层能力。

## 目录说明

```text
atomgit-cli/
├── cmd/
│   └── ag/                     # ag 可执行程序入口
├── internal/
│   ├── agcmd/                  # 命令初始化、执行与退出码处理
│   ├── api/                    # AtomGit API 客户端、类型和分页逻辑
│   │   ├── actions/            # Actions API v8 客户端与类型
│   │   └── testdata/           # API 测试响应样本
│   ├── browser/                # 打开系统浏览器
│   ├── config/                 # XDG 配置与凭据读写
│   ├── oauth/                  # OAuth 登录流程
│   └── version/                # 版本、提交和构建时间元数据
├── pkg/
│   ├── cmdutil/                # 命令共享依赖和通用辅助逻辑
│   └── cmd/                    # Cobra 根命令与所有子命令
│       ├── api/                # 通用 API 请求
│       ├── auth/               # 登录、刷新和凭据管理
│       ├── branch/             # 分支管理
│       ├── browse/             # 在浏览器中打开 AtomGit 资源
│       ├── issue/              # Issue 与 Issue 评论
│       ├── label/              # 标签管理
│       ├── license/            # License 检查
│       ├── org/                # 组织查询
│       ├── pr/                 # Pull Request、审查与评论
│       ├── release/            # Release 与附件管理
│       ├── repo/               # 仓库管理
│       ├── root/               # 根命令和全局参数
│       ├── run/                # Actions run、job、日志与 artifact
│       ├── search/             # 用户、仓库和 Issue 搜索
│       ├── ssh-key/            # SSH Key 管理
│       ├── tag/                # Git tag 管理
│       └── version/            # 版本输出命令
├── bin/
│   └── ag.js                   # npm 主包的平台二进制启动器
├── docs/
│   ├── configuration.md        # 认证、凭据和运行配置
│   ├── cross_repo_pr_demo.md   # 跨仓库 PR 示例
│   ├── installation.md         # 完整安装指南
│   ├── project-structure.md    # 本文档
│   ├── releasing.md            # 发布打包与 Nix package 维护
│   └── usage.md                # 命令参数和使用示例
├── scripts/
│   ├── build-release.sh        # GoReleaser 打包包装脚本
│   ├── build-npm-packages.js   # 生成 npm 主包与平台包
│   ├── check-npm-version.js    # 校验发布版本与 npm 版本
│   ├── set-npm-version.js      # 同步 npm 包版本
│   └── update-nix-package.sh   # 更新并验证 Nix package
├── test/                       # npm 平台包集成测试
├── .goreleaser.yaml            # 跨平台发布打包配置
├── flake.nix                   # Nix package 和开发环境
├── install.sh                  # Linux/macOS Release 安装脚本
├── install.ps1                 # Windows Release 安装脚本
├── Makefile                    # 构建、测试、安装和发布入口
├── package.json                # npm 主包元数据
└── go.mod                      # Go 模块与 Go 版本声明
```

`dist/` 和 `node_modules/` 是本地生成目录，不属于源代码，也不应提交。

## 命令组织

每组根级命令位于独立的 `pkg/cmd/<name>` 包中，通常由 `NewCmdXxx` 创建 Cobra 命令，并在 `pkg/cmd/root/root.go` 中注册。包含多个操作的命令可继续拆分文件或子包，例如：

- `pkg/cmd/issue/comment`：Issue 评论的创建、查看、编辑和删除。
- `pkg/cmd/pr/comment`：Pull Request 评论以及回复。
- `pkg/cmd/release`：Release 的创建、编辑、查看、上传和下载。

命令通过 `pkg/cmdutil.Factory` 获取配置等共享依赖。需要访问 AtomGit API 时，各命令包使用小型接口描述所需方法，再由 `internal/api` 的客户端实现，便于在测试中注入替代实现。

## API 与配置

- 常规仓库功能使用 `internal/api` 中的 AtomGit API v5 客户端。
- Actions 运行记录使用 `internal/api/actions` 中的 API v8 客户端。
- API 请求和响应结构集中定义，字段使用明确的 JSON 标签。
- 用户配置和凭据由 `internal/config` 管理，并兼容 XDG 主路径与旧版 token 路径。
- OAuth 浏览器登录流程位于 `internal/oauth`，系统浏览器调用封装在 `internal/browser`。

## 测试

Go 测试文件与被测包放在同一目录，API 响应样本放在 `internal/api/testdata`；`test/` 包含 npm 平台包的集成测试。命令测试通过依赖注入、临时目录和模拟 HTTP 服务运行，不依赖真实 AtomGit 凭据或外部网络。

新增命令或修改目录职责时，应同步更新本文档和项目 [README](../README.md) 中的入口说明。
