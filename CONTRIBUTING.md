# Contributing to AtomGit CLI

首先，感谢你考虑为 AtomGit CLI 做出贡献！

## 如何贡献

### 报告问题

如果你发现了 bug 或有功能建议，请通过以下方式提交：

1. **Bug 报告**：请提供以下信息
   - 使用的操作系统和架构
   - `ag version` 输出及安装方式
   - Go 版本（仅源码构建时，运行 `go version`）
   - 复现步骤
   - 期望行为 vs 实际行为
   - 脱敏后的错误日志（如果有）

2. **功能建议**：请描述
   - 功能的用例
   - 期望的行为
   - 可能的实现方案（可选）

### 提交代码

1. **Fork 仓库**

   先在 AtomGit 上 Fork 本仓库。如果已有 Fork，请先通过 AtomGit 页面完成同步，然后克隆自己的 Fork：

   ```bash
   git clone https://atomgit.com/YOUR_USERNAME/atomgit-cli.git
   cd atomgit-cli
   ```

2. **创建分支**

   ```bash
   git switch -c feat/your-feature-name
   # 或
   git switch -c fix/bug-description
   ```

3. **开发规范**

   - 使用 Go 1.24.2 或更高版本
   - 使用 `gofmt` 格式化修改的 Go 文件
   - 为新增或修复的行为添加测试
   - 保持与现有代码风格一致
   - 更新相关文档

4. **测试**

   ```bash
   # 格式化修改的 Go 文件
   gofmt -w path/to/changed.go

   # 运行测试和静态检查
   go test ./...
   make lint

   # 在临时目录中验证构建，避免在仓库中留下本地二进制
   (
     ag_build_dir="$(mktemp -d)"
     trap 'rm -rf "$ag_build_dir"' EXIT
     go build -o "$ag_build_dir/ag" ./cmd/ag
   )
   ```

   修改命令参数或输出时，请执行对应命令的 `--help` 冒烟检查；纯文档修改至少运行 `git diff --check`。

5. **提交更改**

   ```bash
   git status --short
   git add path/to/changed-file
   git diff --cached --check
   git commit -m "feat: add new feature description"
   git push -u origin HEAD
   ```

   请只暂存本次贡献相关的文件，不要提交 `dist/`、本地 `ag` 二进制、覆盖率文件、IDE 文件或任何凭据。

   提交信息格式：
   - `feat:` 新功能
   - `fix:` 修复 bug
   - `docs:` 文档更新
   - `style:` 代码格式（不影响功能）
   - `refactor:` 代码重构
   - `test:` 测试相关
   - `chore:` 构建过程或辅助工具的变动

6. **创建 Pull Request**

   从个人 Fork 的开发分支向本仓库的 `main` 分支创建 Pull Request：

   - 描述更改的内容和原因
   - 关联相关的 issue（如果有）
   - 说明执行过的测试及结果
   - 确保 CI 检查通过

7. **请求 AI 代码评审**

   如果仓库已安装 AtomCode AI Review 插件并完成授权，可在 Pull Request 评论区单独发送 `/ai review`，对当前改动执行完整评审并刷新评审评论。更多信息参阅 [AtomGit AI Code Review 文档](https://docs.atomgit.com/en/docs/help/home/org_project/codereview/ai_codereview/)。

8. **获取 PR 改动总结**

   在已启用 AI 代码评审的 Pull Request 评论区单独发送 `/ai summary`，可获取本次 PR 的改动总结，而不会重新执行代码评审。

## 开发指南

### 项目结构

以下仅列出主要目录，完整说明参阅[项目结构](docs/project-structure.md)：

```text
atomgit-cli/
├── cmd/ag/                 # 可执行程序入口
├── internal/               # API、配置、认证和版本等内部实现
│   ├── agcmd/              # 根命令执行与退出码处理
│   ├── api/                # AtomGit API v5 客户端
│   │   └── actions/        # Actions API v8 客户端
│   ├── config/             # XDG 配置与凭据读写
│   ├── browser/            # 打开系统浏览器
│   ├── git/                # Git remote 与仓库信息解析
│   ├── oauth/              # OAuth 登录流程
│   └── version/            # 版本与构建元数据
├── pkg/
│   ├── cmd/                # Cobra 根命令与子命令
│   └── cmdutil/            # 命令共享依赖和辅助逻辑
├── docs/                   # 用户与维护者文档
├── scripts/                # 构建和发布脚本
├── nix/                    # Nix package
└── test/                   # npm 包集成测试
```

### 添加新命令

参考现有命令的实现模式：

1. 在 `pkg/cmd/<command>/` 中提供 `NewCmdXxx`，并使用 Cobra 的参数校验和 `RunE`
2. 通过 `cmdutil.Factory` 获取共享依赖，为业务逻辑添加单元测试
3. 在 `pkg/cmd/root/root.go` 中注册根级命令
4. 新增或改变用户可见命令、参数和输出时更新 `docs/usage.md`
5. 只有影响 README 快速入口时才同步修改 README；目录职责变化时更新 `docs/project-structure.md`

`CHANGELOG.md` 用于 Debian 打包，仅在准备新的 Debian 包版本时更新，不记录日常功能改动。

### API 客户端

- 常规仓库功能使用 `internal/api` 中的 AtomGit API v5 客户端
- Actions run、job、日志和 artifact 使用 `internal/api/actions` 中的 API v8 客户端
- 请求和响应类型应使用明确的 `json` 标签，并与对应 API 的路径、HTTP 方法和成功状态码保持一致
- 通过小型接口和 `cmdutil.Factory` 注入依赖；测试使用模拟 HTTP 服务，不依赖真实凭据或外部网络

### 代码风格

- 使用 `gofmt` 格式化代码
- 遵循 [Effective Go](https://go.dev/doc/effective_go)
- 为导出的标识符和需要解释的复杂逻辑添加注释
- 错误信息应包含操作上下文，并使用 `%w` 包装底层错误
- 业务错误从 Cobra 的 `RunE` 返回，不要在库代码中调用 `os.Exit` 或 `log.Fatal`
- 不要在日志、错误或测试数据中泄露 access token、refresh token、client secret 等凭据

## 行为准则

- 尊重所有参与者
- 接受建设性的批评
- 关注对社区最有利的事情
- 对其他社区成员表示同理心

## 许可证

通过贡献代码，你同意你的贡献将在 [木兰宽松许可证第2版](LICENSE) 下发布。

## 联系方式

如有问题，可以通过以下方式联系：

- 在 [AtomGit CLI Issues](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues) 中提交问题

再次感谢你的贡献！
