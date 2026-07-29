# 配置指南

本文档介绍 AtomGit CLI 的认证、凭据文件、输出安全和当前仓库推断规则。

## 认证

首次使用本工具前，需要选择以下任一方式配置访问令牌：

- 使用 OAuth 登录（推荐）：运行 `ag auth login`，在浏览器中完成 AtomGit 授权。登录成功后，`ag` 会自动将认证信息写入令牌文件。

- 手动创建访问令牌：参考 [AtomGit 访问令牌（PAT）文档](https://docs.gitcode.com/docs/help/home/user_center/security_management/user_pat/)，依次进入「个人设置」->「访问令牌」->「新建访问令牌」，按需设置权限范围和到期时间，再将生成的 PAT 写入令牌文件。PAT 创建后只显示一次，请立即妥善保存，不要将其提交到代码仓库或分享给他人。

令牌文件的默认路径因操作系统而异：

- Linux：`/home/<用户名>/.config/ag-cli/token.json`
- macOS：`/Users/<用户名>/.config/ag-cli/token.json`
- Windows：`C:\Users\<用户名>\.config\ag-cli\token.json`

手动配置单个 PAT 时，仍可使用兼容的旧格式：

```json
{
  "access_token": "your-personal-access-token",
  "user": "your-atomgit-login"
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
| `token_type` | 否 | OAuth 服务返回的令牌类型；手动配置 PAT 时可省略。API 请求当前使用 `Bearer`。 |

`ag auth login` 会将凭据保存为多账号格式。由于 CLI 仅支持 `atomgit.com`，账号以规范化后的用户名作为唯一标识：

```json
{
  "version": 2,
  "active": "alice",
  "accounts": [
    {
      "user": "alice",
      "access_token": "your-personal-access-token",
      "name": "Alice",
      "email": "alice@example.com",
      "token_type": "Bearer"
    }
  ]
}
```

执行任意 `ag auth` 子命令前，CLI 都会检查令牌文件格式。旧单账号文件会通过安全的原子写入自动升级为当前多账号格式，已有 `refresh_token` 等 OAuth 字段会保留；当前格式不会重复写入，未知版本会拒绝迁移。手动使用 PAT 时不要自行编造 `refresh_token`、`expires_in` 或 `created_at`。令牌文件使用 `0600` 权限，写入通过同目录临时文件原子替换；不要将它提交到版本控制。

## 输出安全

`ag` 默认会将终端控制字符转换为可见转义文本，包括输出经管道转发时，以防止仓库、Issue、PR 或 Git 服务端返回的内容注入终端控制序列。确实需要为机器处理保留原始字节时，可显式使用全局参数 `--raw-output`，例如 `ag --raw-output pr diff owner/repo 123`；请勿将未经检查的原始输出直接转发到终端。

## 当前仓库推断

`issue`、`pr`、`tag`、`label`、`release` 命令以及 `repo view`、`repo edit`、`repo fork`、`repo delete` 可以省略 `owner/repo`。省略时，`ag` 会从当前 Git 仓库的 AtomGit remote 推断目标仓库；显式传入的 `owner/repo` 始终优先。

支持 `git@atomgit.com:owner/repo.git`、`ssh://git@atomgit.com/owner/repo.git` 和 `https://atomgit.com/owner/repo.git`。存在多个 remote 时，依次选择 `remote.pushDefault`、当前分支的 upstream remote、AtomGit `origin` 或唯一的 AtomGit remote。GitHub、GitLab 等其他服务的 remote 不会被识别为 AtomGit 仓库；无法唯一确定时，请显式传入 `owner/repo`。
