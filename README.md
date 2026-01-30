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

在使用之前，需要配置访问令牌。创建文件 `~/.atomgit_personal_token.json`：

```json
{
  "access_token": "your-token-here",
  "user": "your-username"
}
```

## 命令

### 认证

```bash
# 查看认证状态
ag auth status

# 显示当前 token
ag auth token
```

### 仓库 (repo)

```bash
# 列出仓库
ag repo list

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
│       ├── pr/pr.go            # PR 命令
│       ├── issue/issue.go      # Issue 命令
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
