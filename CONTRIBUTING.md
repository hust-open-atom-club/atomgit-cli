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

   - 最低支持 Go 1.26.6，开发和正式发布建议使用 Go 1.26.8
   - 使用 `gofmt` 格式化修改的 Go 文件
   - 为新增或修复的行为添加测试
   - 保持与现有代码风格一致
   - 更新相关文档

4. **测试**

   ```bash
   # 格式化修改的 Go 文件
   gofmt -w path/to/changed.go

   # 下载并确认正式发布使用的 Go 版本
   make go-version

   # 使用最低支持的 Go 版本运行兼容性测试
   make go-min-version test-min-go

   # 运行测试和静态检查
   go test ./...
   make lint

   # 在临时目录中验证构建，避免在仓库中留下本地二进制
   (
     ag_build_dir="$(mktemp -d)"
     trap 'rm -rf "$ag_build_dir"' EXIT
     go build -o "$ag_build_dir/ag" ./cmd/ag
   )

   # 运行 CI 使用的 Linux 竞态检测
   make test-race

   # 按 macOS 和 Windows 的构建约束编译全部 Go 测试
   make test-platform-compile

   # 交叉编译 .goreleaser.yaml 中的全部发布目标
   make cross-build

   # 构建正式发布使用的二进制并执行固定版本的漏洞扫描
   make vulncheck

   # 使用假的 registry 和子进程注入运行 npm 测试
   npm ci --no-fund --no-audit
   npm test
   ```

   `make test-race` 在 Linux CI 中运行 `go test -race -count=1 ./...`。可以通过
   `RACE_PACKAGES` 覆盖包列表，但除非某个包存在已记录的竞态检测器兼容问题，CI
   应保持默认的全模块覆盖。该目标设置了 Go 文档支持的
   `GORACE=atexit_sleep_ms=0`：部分测试会将测试二进制重新作为假的 Git 进程执行，
   关闭竞态运行时退出前的一秒等待可以加快这些辅助进程，同时不会抑制竞态报告。
   参见 [Go 竞态检测器文档](https://go.dev/doc/articles/race_detector)。

   AtomGit 托管 Runner 文档目前只列出基于 Linux 的 Ubuntu 和 Euler 环境，因此
   CI 无法执行原生 macOS 或 Windows 测试。`make test-platform-compile` 对
   `darwin/amd64` 和 `windows/amd64` 使用 `go test -exec=true`：所有目标平台专用的
   生产代码和测试代码都必须成功编译、链接，但不会执行生成的测试二进制。在具备相应
   环境时，发布前仍需在 macOS 和 Windows 上原生运行 `go test ./...`。参见
   [AtomGit 托管 Runner 文档](https://docs.gitcode.com/docs/help/home/org_project/pipeline/runner-management/using-hosted-runners/)。

   `go.mod` 的 `go` 行规定源码构建所需的最低版本 Go 1.26.6，`toolchain` 行规定
   开发和正式发布建议使用的精确版本 Go 1.26.8。普通 Make 目标固定使用建议版本，
   `make test-min-go` 则专门验证最低版本兼容性。AtomGit 的 `setup-go` 当前只提供到
   Go 1.26.1，因此 CI 先将其作为引导命令，再下载并验证这两个版本。工具链下载
   遵循 `GOPROXY`，并通过 `GOSUMDB` 配置的校验和数据库验证。参见
   [AtomGit setup 工具支持列表](https://docs.gitcode.com/docs/help/home/org_project/pipeline/syntax-reference/setup-supported-tools/)
   和 [Go 工具链文档](https://go.dev/doc/toolchain)。

   `make vulncheck` 使用 Go 1.26.8 构建实际发布形态的 `ag` 二进制，再用固定版本的
   `govulncheck` 以 binary 模式扫描，并显式查询规范数据库 `https://vuln.go.dev`。
   发现可达漏洞、工具下载失败、数据库不可用或扫描器报错都会使检查失败；规范数据库
   中已经撤回的报告不计为漏洞。参见 [govulncheck 命令文档](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
   和 [Go 漏洞数据库规范](https://go.dev/doc/security/vuln/database)。

   npm 测试使用假的 registry 和子进程注入，不得发布软件包、读取真实 npm token，
   也不得访问真实 registry。

   修改命令参数或输出时，请执行对应命令的 `--help` 冒烟检查；纯文档修改至少运行 `git diff --check`。命令树或命令元数据发生变化时，请运行 `make docs-reference` 更新自动生成的 `docs/command-reference.md`，并运行 `make docs-reference-check` 确认没有文档漂移。

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
