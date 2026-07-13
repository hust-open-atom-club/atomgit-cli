# AGENTS.md

本文件为在本仓库中工作的自动化编码代理提供项目约定。除非更深层目录中存在更具体的 `AGENTS.md`，本文件适用于整个仓库。

## 项目概览

- 本项目是 AtomGit 命令行工具 `ag`，使用 Go 和 Cobra 开发。
- Go 模块：`atomgit.com/hust-open-atom-club/atomgit-cli`。
- 当前 `go.mod` 指定 Go 1.24.2；不要无故降低或升级 Go 版本及依赖。
- CLI 入口为 `cmd/ag/main.go`，执行流程为 `main` -> `internal/agcmd.Main` -> `pkg/cmd/root.NewCmdRoot`。
- AtomGit API 基址和通用 HTTP 请求逻辑位于 `internal/api/client.go`，API 数据结构位于 `internal/api/types.go`。

## 目录与职责

- `cmd/ag/`：可执行程序入口，保持精简。
- `internal/agcmd/`：程序初始化、根命令执行和退出码处理。
- `internal/api/`：AtomGit API v5 客户端与响应/请求类型。
- `internal/config/`：XDG 配置、凭据读取与写入。
- `internal/oauth/`：OAuth 登录流程。
- `pkg/cmdutil/`：注入到命令中的共享依赖（`Factory`）。
- `pkg/cmd/<name>/`：各 Cobra 命令及子命令实现。
- `.goreleaser.yaml`：Linux、macOS 和 Windows 的 amd64/arm64 发布打包配置。
- `scripts/build-release.sh`：GoReleaser 打包包装脚本，负责版本元数据和安装脚本，输出到被忽略的 `dist/`。
- `README.md`：用户安装、配置和命令用法；`CHANGELOG.md`：面向用户的变更记录。

## 开发约定

- 遵循现有 Cobra 结构：命令包提供 `NewCmdXxx`，子命令通过 `cmd.AddCommand` 注册，根级命令在 `pkg/cmd/root/root.go` 注册。
- 命令参数使用 Cobra 的 `Args` 校验；选项集中放在小型 `Options` 结构体中；业务错误从 `RunE` 返回，不要直接终止进程。
- 只有 `internal/agcmd`/`cmd/ag` 负责顶层错误输出和退出码。库代码不得调用 `os.Exit` 或 `log.Fatal`。
- 复用 `cmdutil.Factory` 获取配置等依赖。需要提高可测试性时，优先扩展依赖注入，不要在测试中访问真实 AtomGit 服务。
- API 路径、HTTP 方法和成功状态码应与 AtomGit API v5 一致。新增 API JSON 字段时，在 `internal/api/types.go` 使用明确的 `json` 标签。
- 复用现有的仓库名解析、配置路径和 API 客户端逻辑，避免在不同命令中复制规则。
- 错误信息应包含操作上下文，并用 `%w` 包装底层错误；不要泄露 access token、refresh token、client secret 或完整认证响应。
- 凭据文件必须保持 `0600` 权限，并继续兼容 XDG 主路径和旧版 token 路径。涉及认证或配置的测试应使用临时 HOME/XDG 目录。
- 用户可见文案目前以英文为主；修改同一命令时保持其现有语言和格式一致。
- 使用 `gofmt` 格式化所有改动的 Go 文件。避免与任务无关的重构、依赖升级和大范围格式变化。

## 测试与验证

提交前至少运行：

```bash
gofmt -w <修改的.go文件>
go test ./...
go build ./cmd/ag
```

也可用以下命令确认没有遗漏格式问题：

```bash
test -z "$(gofmt -l .)"
```

- 新增或修复行为时添加测试，优先使用表驱动测试和 `t.Run`，风格参考 `pkg/cmd/repo/repo_test.go`。
- 纯解析、校验和格式化逻辑应拆成小函数做单元测试。
- 测试不得依赖真实凭据、用户主目录、浏览器、固定端口或外部网络；使用 `t.TempDir`、环境变量隔离和 `httptest`。
- 修改命令标志或输出时，除单元测试外，至少执行相应的 `go run ./cmd/ag <command> --help` 冒烟检查。
- 发布流程发生变化时，使用 `make release-snapshot VERSION=vX.Y.Z` 验证 GoReleaser 的多平台制品。只有工作区干净、`vX.Y.Z` tag 存在且指向当前 HEAD 时，才运行 `make release VERSION=vX.Y.Z`。两个命令都会将制品生成到 `dist/`，不要提交这些制品。

## 文档与提交

- 新增、删除或改变用户可见命令、参数、配置路径时，同步更新 `README.md`。
- `CHANGELOG.md` 用于 Debian 打包。只在准备新的 Debian 包版本时，按 Debian changelog 格式新增对应版本条目；不要将日常功能提交或内部发布流程变化逐次记入。
- 发布构建变化需检查 `install.sh`、`install.ps1` 和 `scripts/build-release.sh` 的一致性。
- 提交信息遵循仓库已有的简洁约定，推荐 `feat:`、`fix:`、`docs:`、`refactor:`、`test:`、`chore:` 前缀。
- 不要提交 `dist/`、本地 `ag` 二进制、覆盖率文件、IDE 文件、token 文件或任何密钥。

## 改动前检查

- 先阅读目标命令及相邻命令，保持命名、输出和错误处理一致。
- 保留工作区中与当前任务无关的现有改动，不覆盖或回退用户修改。
- 对删除仓库、修改认证状态、发布制品等有副作用的操作，不要用真实账号做验证；以单元测试或 mock HTTP 服务验证。
