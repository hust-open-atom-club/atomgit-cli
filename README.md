# AtomGit CLI (ag)

AtomGit 命令行工具，参考 GitHub CLI (gh) 开发。

## 安装

```bash
# 从源码构建
go build ./cmd/ag

# 安装到 $GOPATH/bin
go install ./cmd/ag
```

## 配置

首次使用本工具前，需要选择以下任一方式配置访问令牌：

- 使用 OAuth 登录（推荐）：运行 `ag auth login`，在浏览器中完成 AtomGit 授权。登录成功后，`ag` 会自动将认证信息写入令牌文件。

- 手动创建访问令牌：参考 [AtomGit 访问令牌（PAT）文档](https://docs.gitcode.com/docs/help/home/user_center/security_management/user_pat/)，依次进入「个人设置」->「访问令牌」->「新建访问令牌」，按需设置权限范围和到期时间，再将生成的 PAT 写入令牌文件。PAT 创建后只显示一次，请立即妥善保存，不要将其提交到代码仓库或分享给他人。

令牌文件的默认路径因操作系统而异：

- Linux：`/home/<用户名>/.config/ag-cli/token.json`
- macOS：`/Users/<用户名>/.config/ag-cli/token.json`
- Windows：`C:\Users\<用户名>\.config\ag-cli\token.json`

手动配置 PAT 时，文件内容至少包括：

```json
{
  "access_token": "your-personal-access-token",
  "user": "your-atomgit-login",
  "token_type": "Bearer"
}
```

配置文件字段说明：

| 字段 | 是否必填 | 说明 |
| --- | --- | --- |
| `access_token` | 是 | 用于调用 AtomGit API 的访问令牌。手动配置时填写刚创建的 PAT；请勿泄露或提交到版本控制。 |
| `user` | 是 | AtomGit 登录用户名（账号标识），不是昵称或邮箱。 |
| `refresh_token` | 否 | OAuth 刷新令牌，仅由 `ag auth login` 获取，并供 `ag auth refresh` 换取新的访问令牌。PAT 没有该字段。 |
| `expires_in` | 否 | OAuth 访问令牌从签发时刻起的有效秒数，由服务端返回。PAT 的有效期在创建 PAT 时设置，手动配置可省略。 |
| `created_at` | 否 | CLI 保存或刷新 OAuth 凭据时记录的 Unix 时间戳（秒），用于表示签发/保存时间。手动配置 PAT 时可省略。 |
| `token_type` | 是 | 令牌认证类型。当前 PAT 和 OAuth 访问令牌均使用 `Bearer`。 |

`ag auth login` 会自动写入上述 OAuth 字段；手动使用 PAT 时不要自行编造 `refresh_token`、`expires_in` 或 `created_at`。请确保配置文件仅允许当前用户读取和写入令牌文件。

## 命令

### 认证

```bash
# 浏览器 OAuth 登录并写入令牌文件
ag auth login
# 已登录时会提示无需重复登录；若要重新走浏览器：ag auth login --force

# 用 refresh_token 刷新 access_token（需之前登录响应里包含 refresh_token）
ag auth refresh

# 查看认证状态
ag auth status

# 显示当前 token
ag auth token

# 删除本地令牌文件
ag auth logout
```

可选环境变量（覆盖默认 OAuth 应用）：`AG_OAUTH_CLIENT_ID`、`AG_OAUTH_CLIENT_SECRET`；若本机 **8765** 端口被占用，可设置 **`AG_OAUTH_REDIRECT_PORT`**（需与 AtomGit 应用配置的回调地址一致）。

### 仓库 (repo)

```bash
# 列出仓库（默认显示 30 条）
ag repo list

# 指定最多列出100条仓库
ag repo list --limit 100

# 查看仓库详情
ag repo view owner/repo

# 创建仓库
# 在当前用户账号下创建
ag repo create my-project --public

# 在指定个人或组织账号下创建
ag repo create owner/my-project --public --description "My project"

# 克隆仓库
ag repo clone owner/repo
ag repo clone owner/repo --branch dev

# Fork 仓库
ag repo fork owner/repo
ag repo fork owner/repo --name my-fork --public

# 删除仓库
ag repo delete owner/repo --yes
```

### Pull Request (pr)

```bash
# 列出 PR
ag pr list owner/repo
ag pr list owner/repo --state closed

# 查看 PR
ag pr view owner/repo 123

# 创建 PR
ag pr create owner/repo --title "Fix bug" --body "Description" --base main --head feature-branch

# 关闭 PR
ag pr close owner/repo 123
```

#### PR 评论

```bash
# 创建评论
ag pr comment create owner/repo 123 --body "LGTM!"
ag pr comment create owner/repo 123 --body-file review.md

# 查看所有评论（树形结构显示）
ag pr comment view owner/repo 123

# 编辑评论（交互式编辑）
ag pr comment edit owner/repo 123 456
ag pr comment edit owner/repo 123 456 --body "Updated comment"

# 删除评论
ag pr comment delete owner/repo 123 456
ag pr comment delete owner/repo 123 456 --yes

# 回复评论（PR 特有）
ag pr comment reply owner/repo 123 456 --body "Thanks for the feedback!"
```

### Issue

```bash
# 列出 Issue
ag issue list owner/repo
ag issue list owner/repo --state all

# 查看 Issue
ag issue view owner/repo 42

# 创建 Issue
ag issue create owner/repo --title "Bug report" --body "Description"
```

#### Issue 评论

```bash
# 创建评论
ag issue comment create owner/repo 42 --body "I can reproduce this issue"
ag issue comment create owner/repo 42 --body-file details.md

# 查看所有评论
ag issue comment view owner/repo 42

# 编辑评论（交互式编辑）
ag issue comment edit owner/repo 42 789
ag issue comment edit owner/repo 42 789 --body "Updated information"

# 删除评论
ag issue comment delete owner/repo 42 789
ag issue comment delete owner/repo 42 789 --yes
```

### License

```bash
# 检查 license 合规性
ag license check MIT
ag license check Apache-2.0
ag license check GPL-3.0
```

### SSH Key

```bash
# 添加 SSH key
ag ssh-key add ~/.ssh/id_rsa.pub --title "My Laptop"
cat ~/.ssh/id_rsa.pub | ag ssh-key add --title "My Laptop"
```

### 版本

```bash
# 查看版本信息
ag version

# 机器可读的 JSON 输出
ag version --json
```

文本输出包含版本、源码提交和构建日期，例如：

```text
ag version v0.5.0 (commit: abc1234, built: 2026-07-12T00:00:00Z)
```

JSON 输出包含固定的 `version`、`commit` 和 `buildDate` 字段：

```json
{
  "version": "v0.5.0",
  "commit": "abc1234",
  "buildDate": "2026-07-12T00:00:00Z"
}
```

从源码构建或执行 `go install` 且未注入发布元数据时，版本默认值为 `dev`。如果 Go 构建信息包含模块版本、源码提交或提交时间，`ag version` 会使用这些信息替代或补充默认值；工作区存在未提交改动时，版本还会带有 dirty 标记。

发布版二进制文件通过 `scripts/build-release.sh` 构建，使用 `TAG` 环境变量（如 `TAG=v0.5.0`）注入语义版本标签。发布版构建还支持 `SOURCE_DATE_EPOCH`，以生成可复现的构建日期。

## 项目结构

```
atomgit-cli/
├── cmd/ag/main.go              # 入口
├── internal/
│   ├── agcmd/cmd.go            # 核心命令处理
│   ├── config/config.go        # 配置管理
│   ├── version/version.go      # 版本元数据
│   └── api/
│       ├── client.go           # API 客户端
│       └── types.go            # 数据类型
├── pkg/
│   ├── cmdutil/factory.go      # 命令工厂
│   └── cmd/
│       ├── root/root.go        # 根命令
│       ├── auth/auth.go        # 认证命令
│       ├── repo/               # 仓库命令
│       │   ├── repo.go
│       │   ├── create.go
│       │   ├── clone.go
│       │   ├── delete.go
│       │   └── fork.go
│       ├── pr/                 # PR 命令
│       │   ├── pr.go
│       │   └── comment/        # PR 评论命令
│       ├── issue/              # Issue 命令
│       │   ├── issue.go
│       │   └── comment/        # Issue 评论命令
│       ├── license/            # License 命令
│       │   ├── license.go
│       │   └── check.go
│       ├── ssh-key/ssh_key.go  # SSH key 命令
│       └── version/version.go  # 版本命令
└── go.mod
```

## API

使用 AtomGit API v5: `https://api.atomgit.com/api/v5`

## 参考

- [AtomGit API 文档](https://docs.atomgit.com/docs/apis/)
- [GitHub CLI](https://cli.github.com/)

## License

[木兰宽松许可证第2版](LICENSE) (Mulan Permissive Software License, Version 2)

Copyright (c) 2026 AtomGit CLI Contributors
