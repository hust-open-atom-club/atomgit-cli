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

在使用之前需要配置访问令牌。推荐先执行 **`ag auth login`**：会在浏览器中完成 AtomGit OAuth，并把 `access_token` 与 `user` 写入下面的 `token.json`（覆盖已有内容）。

也可手动创建 token 文件：

**推荐方式：遵循XDG规范**
```
$XDG_CONFIG_HOME/ag-cli/token.json
```
默认路径为 `~/.config/ag-cli/token.json`

**兼容旧方式的路径**：
```
~/.atomgit_personal_token.json
```

文件内容示例：
```json
{
  "access_token": "your-token-here",
  "user": "your-username",
  "refresh_token": "…",
  "expires_in": 3600,
  "created_at": 1710000000,
  "token_type": "Bearer"
}
```
`refresh_token` / `expires_in` / `created_at` 由 `ag auth login` 在 OAuth 返回时写入，供 `ag auth refresh` 使用；手动编辑时可省略。

## 命令

### 认证

```bash
# 浏览器 OAuth 登录并写入 ~/.config/ag-cli/token.json（或 $XDG_CONFIG_HOME/ag-cli/token.json）
ag auth login
# 已登录时会提示无需重复登录；若要重新走浏览器：ag auth login --force

# 用 refresh_token 刷新 access_token（需之前登录响应里包含 refresh_token）
ag auth refresh

# 查看认证状态
ag auth status

# 显示当前 token
ag auth token

# 删除本地保存的 token 与用户（XDG 下 token.json 及兼容路径）
ag auth logout
```

可选环境变量（覆盖默认 OAuth 应用）：`AG_OAUTH_CLIENT_ID`、`AG_OAUTH_CLIENT_SECRET`；若本机 **8765** 端口被占用，可设置 **`AG_OAUTH_REDIRECT_PORT`**（需与 AtomGit 应用配置的回调地址一致）。

### 仓库 (repo)

```bash
# 列出仓库（默认第1页，每页30条）
ag repo list

# 分页列出仓库（第2页，每页50条）
ag repo list 2 50

# 查看仓库详情
ag repo view owner/repo

# 创建仓库
ag repo create my-project --public --description "My project"

# 克隆仓库
ag repo clone owner/repo
ag repo clone owner/repo --branch develop

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
ag pr create owner/repo --title "Fix bug" --body "Description" --base master --head feature-branch

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

## 项目结构

```
ag-cli/
├── cmd/ag/main.go              # 入口
├── internal/
│   ├── agcmd/cmd.go            # 核心命令处理
│   ├── config/config.go        # 配置管理
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
│       └── ssh-key/ssh_key.go  # SSH key 命令
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
