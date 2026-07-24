# AtomGit CLI (ag)

AtomGit 命令行工具，参考 GitHub CLI (gh) 开发。

## 安装

### npm

需要 Node.js 18 或更高版本：

```bash
npm install --global @hust-open-atom-club/atomgit-cli
ag version
```

npm 会通过当前操作系统和 CPU 对应的可选依赖安装预编译二进制，不需要运行 `postinstall` 脚本。

### Homebrew

macOS 或 Linux 用户可以通过项目维护的 Homebrew tap 安装：

```bash
brew tap hust-open-atom-club/tap
brew install atomgit-cli
ag version
```

也可以使用一条命令直接安装：

```bash
brew install hust-open-atom-club/tap/atomgit-cli
```

升级已安装的 AtomGit CLI：

```bash
brew update
brew upgrade atomgit-cli
```

### Nix / NixOS

AtomGit CLI 已进入 `nixos-unstable`，包名为 `atomgit-cli`，安装后提供 `ag` 命令。

如果你的 Nix registry 或 flake input 已指向 `nixos-unstable`：

```bash
nix profile install nixpkgs#atomgit-cli
ag version
```

如果你使用的是稳定版 nixpkgs，可以显式从 `nixos-unstable` 安装：

```bash
nix profile install github:NixOS/nixpkgs/nixos-unstable#atomgit-cli
```

临时运行：

```bash
nix run github:NixOS/nixpkgs/nixos-unstable#atomgit-cli -- version
```

其他安装方式请参阅[完整安装指南](https://atomgit.com/hust-open-atom-club/atomgit-cli/blob/main/docs/installation.md)。

### 从源码构建

```bash
# 构建到 bin/ag（Windows 为 bin/ag.exe）
make build

# 安装到 $GOPATH/bin
make install
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

### 输出安全

`ag` 默认会将终端控制字符转换为可见转义文本，包括输出经管道转发时，以防止仓库、Issue、PR 或 Git 服务端返回的内容注入终端控制序列。确实需要为机器处理保留原始字节时，可显式使用全局参数 `--raw-output`，例如 `ag --raw-output pr diff owner/repo 123`；请勿将未经检查的原始输出直接转发到终端。

### 当前仓库推断

`issue`、`pr`、`tag`、`label`、`release` 命令以及 `repo view`、`repo edit`、`repo fork`、`repo delete` 可以省略 `owner/repo`。省略时，`ag` 会从当前 Git 仓库的 AtomGit remote 推断目标仓库；显式传入的 `owner/repo` 始终优先。

支持 `git@atomgit.com:owner/repo.git`、`ssh://git@atomgit.com/owner/repo.git` 和 `https://atomgit.com/owner/repo.git`。存在多个 remote 时，依次选择 `remote.pushDefault`、当前分支的 upstream remote、AtomGit `origin` 或唯一的 AtomGit remote。GitHub、GitLab 等其他服务的 remote 不会被识别为 AtomGit 仓库；无法唯一确定时，请显式传入 `owner/repo`。

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
ag repo view
ag repo view owner/repo

# 在浏览器中打开仓库
ag repo view owner/repo --web

# 创建仓库
# 在当前用户账号下创建
ag repo create my-project --public

# 在指定个人或组织账号下创建
ag repo create owner/my-project --public --description "My project"

# 只更新描述；未指定的仓库设置保持不变
ag repo edit --description "Updated description"
ag repo edit owner/my-project --description "Updated description"

# 显式清空描述
ag repo edit owner/my-project --description ""

# 更新默认分支
ag repo edit owner/my-project --default-branch main

# 同时更新名称和可见性（名称、可见性变更默认需要确认）
ag repo edit owner/my-project --name "My Project" --visibility private

# 非交互式修改可见性
ag repo edit owner/my-project --public --yes
ag repo edit owner/my-project --private --yes

# 克隆仓库
ag repo clone owner/repo
ag repo clone owner/repo --branch dev

# Fork 仓库
ag repo fork
ag repo fork owner/repo
ag repo fork owner/repo --name my-fork --public

# 删除仓库
ag repo delete --yes
ag repo delete owner/repo --yes
```

`ag repo edit` 仅发送命令行中明确指定的字段，支持 `--name`、`--description`、`--default-branch` 和 `--visibility public|private`。`--public`、`--private` 是可见性的便利选项；它们与 `--visibility` 三者互斥。名称或可见性修改需要交互确认，可使用 `--yes` 跳过确认。成功后命令会显示更新后的仓库名称和浏览器 URL。

该命令不会修改仓库 URL 路径、所有者、主页、LFS、模块开关、合并策略，也不会接受后静默忽略 GitHub CLI 的其他仓库设置选项。

### Branch

```bash
# 列出远程分支（默认显示 30 条）
ag branch list owner/repo
ag branch list owner/repo --limit 100

# 查看远程分支详情
ag branch view owner/repo main
ag branch view owner/repo feature/foo

# 从指定 ref 创建远程分支
ag branch create owner/repo feature/foo --ref main

# 删除远程分支（默认需要确认；不会删除本地 Git 分支）
ag branch delete owner/repo feature/foo
ag branch delete owner/repo feature/foo --yes
```

### Browse

在默认浏览器中打开仓库页面或指定资源：

```bash
# 打开当前仓库首页（需在 git 仓库内运行）
ag browse

# 打开指定仓库
ag browse -R owner/repo

# 打开 Issue 或 PR
ag browse 42

# 打开文件（默认分支）
ag browse main.go

# 打开文件并定位到指定行
ag browse main.go:312
ag browse main.go:312-320
ag browse main.go:312..320

# 在指定分支上打开文件
ag browse -b dev main.go:42

# 在指定 commit 上打开文件
ag browse -c abc1234 main.go

# 打开 Releases 页面
ag browse -r

# 只打印 URL，不打开浏览器
ag browse -n
```

### Pull Request (pr)

```bash
# 列出 PR
ag pr list
ag pr list owner/repo
ag pr list owner/repo --state closed

# 查看 PR
ag pr view 123
ag pr view owner/repo 123

# 在浏览器中打开 PR
ag pr view owner/repo 123 --web

# 查看 PR diff
ag pr diff owner/repo 123

# 合并 PR
ag pr merge owner/repo 123
ag pr merge owner/repo 123 --rebase
ag pr merge owner/repo 123 --squash
ag pr merge owner/repo 123 --admin
ag pr merge owner/repo 123 --subject "Merge PR #123" --body "Merge details"
ag pr merge owner/repo 123 --delete-branch
ag pr merge owner/repo 123 --rebase --squash --admin --subject "Merge PR #123" --body "Merge details" --delete-branch

# 创建 PR
ag pr create owner/repo --title "Fix bug" --body "Description" --base main --head feature-branch

# 关闭 PR
ag pr close owner/repo 123

# 重新打开 PR
ag pr reopen owner/repo 123
```

#### PR 评审

AtomGit 当前公开的 review API 仅支持批准操作，不支持 request-changes 或带正文的正式评审评论。普通评论请使用 `ag pr comment create`。

```bash
# 批准 PR
ag pr review owner/repo 123 --approve

# 仓库管理员强制通过审查
ag pr review owner/repo 123 --approve --force
```

命令会在提交前确认 PR 仍处于打开状态，并阻止当前用户误评审自己创建的 PR。

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
ag issue list
ag issue list owner/repo
ag issue list owner/repo --state all

# 查看 Issue
ag issue view 42
ag issue view owner/repo 42

# 在浏览器中打开 Issue
ag issue view owner/repo 42 --web

# 添加 Issue 标签（使用逗号分隔多个标签）
ag issue label owner/repo 42 "bug, help wanted,priority/high"
ag issue label owner/repo 42 --add "bug, help wanted"

# 移除 Issue 标签
ag issue label owner/repo 42 --remove "priority/high"

# 修改 Issue 标题或正文
ag issue edit owner/repo 42 --title "Updated title"
ag issue edit owner/repo 42 --body "Updated description"
ag issue edit owner/repo 42 --body-file details.md

# 创建 Issue
ag issue create owner/repo --title "Bug report" --body "Description"

# 重新打开 Issue
ag issue reopen owner/repo 42
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

### Tag

```bash
# 在当前 Git 仓库中列出标签
ag tag list

# 显式指定仓库
ag tag list owner/repo

# 创建或删除标签
ag tag create v1.0.0 --ref main
ag tag delete v1.0.0
```

### Label

```bash
# 列出仓库标签（默认显示 30 条）
ag label list owner/repo
ag label list owner/repo --limit 50

# 创建标签
ag label create owner/repo --name bug --color "#ff0000"

# 修改标签名称或颜色
ag label edit owner/repo bug --name defect
ag label edit owner/repo defect --color "#d73a4a"

# 删除标签（默认要求确认）
ag label delete owner/repo obsolete
ag label delete owner/repo obsolete --yes
```

AtomGit API v5 的标签创建和修改接口支持 `name` 与 `color`。`label list` 会在接口返回时显示标签描述，但创建和修改命令不会发送 API 未公开支持的 `description` 字段。颜色必须使用 `#RGB` 或 `#RRGGBB` 格式。

### Actions 运行记录 (run)

`ag run` 目前只提供只读的运行检查能力，不会触发、重跑、取消或删除工作流运行。

```bash
# 列出运行记录（默认最多 30 条）
ag run list owner/repo

# 按分支、状态和触发事件过滤
ag run list owner/repo --branch main --status failed --event push

# 也可按触发人、PR、workflow 和毫秒时间戳过滤
ag run list owner/repo --actor alice --pr 42 --workflow-name CI --limit 50
ag run list owner/repo --start-time 1700000000000 --end-time 1700086400000

# 查看 run、jobs、steps、URL 和 artifacts
ag run view owner/repo <run-id>

# 查看指定 job 及其步骤
ag run view owner/repo <run-id> --job <job-id>

# 解包 AtomGit 返回的日志归档，并将各步骤日志文本输出到 stdout
ag run view owner/repo <run-id> --job <job-id> --log

# 流式下载原始 job 日志 ZIP；默认不覆盖已有文件
ag run view owner/repo <run-id> --job <job-id> --log-file job-logs.zip
ag run view owner/repo <run-id> --job <job-id> --log-file job-logs.zip --overwrite

# 下载指定 artifact。未指定文件名时使用 artifact 名称并添加 .zip
ag run view owner/repo <run-id> --artifact <artifact-id>
ag run view owner/repo <run-id> --artifact <artifact-id> --artifact-file build.zip --overwrite
```

`--log` 会先把 AtomGit 返回的日志 ZIP 流式写入临时文件，再逐项输出其中的日志文本；若服务端返回纯文本也会直接兼容。`--log-file` 保留服务端原始 ZIP。日志和 artifact 文件下载都会先写入目标目录中的临时文件，完整写入后再移动到目标路径。若目标已存在，必须显式使用 `--overwrite`。

### 通用 API 请求

`ag api` 向 AtomGit API v5 的相对路径发送认证请求，适合调用尚无专用命令的接口。默认方法为 GET；POST、PATCH、PUT 和 DELETE 必须用 `--method` 显式选择。通用命令不会推断接口影响，也不会在可能修改远程资源前要求确认。

```bash
# 基本 GET 请求
ag api /user

# GET 字段会追加为 URL 编码的查询参数
ag api /repos/owner/repo/issues --field state=open --field labels="help wanted"

# 非 GET 字段会编码为仅包含字符串值的 JSON 对象
ag api /repos/owner/repo/issues --method POST --field title="API-created issue"

# 从文件或标准输入原样读取请求体
ag api /repos/owner/repo/issues/42 --method PATCH --input update.json
printf '%s' '{"title":"stdin"}' | ag api /repos/owner/repo/issues --method POST --input -

# 逐页请求；每个完整 JSON 页面压缩为一行 NDJSON
ag api /repos/owner/repo/issues --paginate
```

端点必须是 API v5 下的相对路径；绝对 URL、`//host/path`、片段和越过 API 基址的路径会在读取凭据前被拒绝。认证信息仅通过 `Authorization` 请求头发送。同源重定向可保留认证；scheme、主机或有效端口变化后，当前及后续跳转均不会再携带认证信息。

`--field key=value` 在第一个 `=` 处分隔。GET 会保留已有查询值并按命令行顺序追加字段；其他支持的方法生成 JSON 对象，重复键以最后一个值为准。`--input` 与 `--field` 互斥，原始输入不会推断 `Content-Type`。`--accept` 默认是 `application/json`。

`--paginate` 仅支持无原始输入的 GET。默认从 `page=1&per_page=100` 开始；已有的正整数值会被保留。服务端提供一致的 `total_page` 响应头时据此停止；否则仅数组响应可通过空页或短页停止。后续页面失败时，已完成的 NDJSON 行会保留，失败页面不会产生部分输出。

成功响应（包括空响应和二进制响应）会直接写到标准输出，不增加标签或换行。终端控制字符默认仍会转换为可见转义；机器处理确需原始字节时使用 `ag --raw-output api ...`，不要将未经检查的原始输出直接转发到终端。

### Release

```bash
# 列出仓库 Release（默认最多 30 条）
ag release list
ag release list owner/repo --limit 50

# 按 tag 查看 Release 详情（附件列表、作者、时间、状态等）
ag release view v1.0.0
ag release view owner/repo v1.0.0

# 创建 Release；--target 指向提交 SHA，--prerelease 标记预发布
ag release create v1.0.0 --body "首批正式发布"
ag release create owner/repo v1.0.0 --name "Release 1.0" --body "首批正式发布"
ag release create owner/repo v1.0.0 --body-file ./CHANGELOG.md --target 0123abcd --prerelease

# 编辑 Release；只改变用户明确指定的内容，未指定的 name/body 会从当前 Release 回读并保持不变
ag release edit v1.0.0 --name "Release 1.0.1"
ag release edit owner/repo v1.0.0 --body-file ./docs/release-notes.md --latest
ag release edit owner/repo v1.1.0-rc --prerelease
ag release edit owner/repo v1.0.0 --name "Release 1.1" --body "热修复"

# 上传附件；远端附件名默认为本地文件名，可用 --name 指定
ag release upload v1.0.0 ./dist/app.tar.gz
ag release upload owner/repo v1.0.0 ./build/app.zip --name app-v1.zip
ag release upload owner/repo v1.0.0 ./existing.tar.gz --skip-existing
ag release upload owner/repo v1.0.0 ./new.tar.gz --overwrite

```

`ag release create` 必须通过 `--body` 或 `--body-file` 提供非空说明，这是 AtomGit 创建 Release API 的必填字段。`ag release edit` 只改变用户明确指定的内容（`--name`、`--body` 或 `--body-file`、`--latest`、`--prerelease`），未指定的 name/body 会从当前 Release 回读并保持不变；状态仅在显式 `--latest` 或 `--prerelease` 时改变。`--body` 与 `--body-file` 互斥。`--latest` 将该 Release 标记为仓库最新发布。

`ag release upload` 的远端附件名默认为本地文件名（`filepath.Base(file)`），可用 `--name` 指定。若远端已存在同名附件，默认会报错并提示选择 `--skip-existing` 或 `--overwrite`，**绝不静默覆盖**：

- `--skip-existing`：发现同名附件即报告成功并退出，**不会修改远端**，也不会执行删除、查询上传地址或上传操作。
- `--overwrite`：仅当远端唯一匹配且该附件 `type=attach`、ID 为正整数时，先成功取得上传地址，再删除旧附件并上传新文件；取得上传地址失败时旧附件保持不变。若删除响应中断，命令会重新读取 Release，仅在确认旧附件已经不存在时继续；若后续上传失败，错误信息会明确说明旧附件已被删除。对于 `type=source` 的源码归档、ID 非正、或存在多个同名匹配的情况，均会拒绝且不执行删除。

附件传输不会使用普通 API 请求的 30 秒总超时，而是默认限制为 30 分钟，可用 `--timeout` 调整，或用 `--timeout 0` 关闭总时限。非空文件只会在传输尚未开始（请求体零字节被读取）时自动重试一次；零字节文件或传输开始后的中断会返回非零退出码并提示远端状态可能不确定，不会盲目重放上传。

本组 `ag release` 命令是供手工或脚本调用的底层 Release 管理原语，不会自动执行版本 tag 校验、测试、跨平台构建、校验和生成或整套发布编排。tag 驱动的端到端自动发布由 Issue #18 跟踪。

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

# 查看 SSH keys（可通过 --limit 限制数量）
ag ssh-key list
ag ssh-key list --limit 200

# 删除 SSH key（默认要求确认）
ag ssh-key delete 123
ag ssh-key delete 123 --yes
```

### 搜索

```bash
# 搜索命令均需要先登录
ag auth login

# 搜索用户；可按注册时间排序
ag search users torvalds
ag search users torvalds --sort joined_at --order asc

# 搜索仓库；repositories 可简写为 repos
ag search repositories kernel --limit 20
ag search repos kernel --limit 20
ag search repos cli --owner hust-open-atom-club --language Go --sort stars_count --order desc
ag search repos cli --fork

# 搜索 Issue；repo 接收仓库路径，state 支持 open 或 closed
ag search issues "memory leak" --limit 50
ag search issues bug --repo hust-open-atom-club/atomgit-cli --state open --sort created_at --order desc

# 结构化输出
ag search users torvalds --json
```

支持的服务端过滤和排序选项：

- `search users`：`--sort joined_at`、`--order asc|desc`；
- `search repositories|repos`：`--owner`、`--language`、`--fork`、`--sort last_push_at|stars_count|forks_count`、`--order asc|desc`；
- `search issues`：`--repo`、`--state open|closed`、`--sort created_at|last_push_at`、`--order asc|desc`。

### 版本

```bash
# 查看版本信息
ag version

# 等价的根级参数
ag --version

# 机器可读的 JSON 输出
ag version --json
```

通过 `make build` 或 `make install` 从源码构建且未注入发布元数据时，版本默认值为 `dev`。如果 Go 构建信息包含模块版本、源码提交或提交时间，`ag version` 会使用这些信息替代或补充默认值；工作区存在未提交改动时，版本还会带有 dirty 标记。

## 发布打包

发布版使用 [GoReleaser](https://goreleaser.com/install/) 打包，tag 统一使用 `vX.Y.Z` 三段式 SemVer。正式发布前应先提交所有改动，并在当前 HEAD 创建版本 tag：

```bash
npm run version:npm -- 0.6.0
git tag v0.6.0
make release VERSION=v0.6.0
```

`make release` 会检查工作区干净、tag 存在且指向当前 HEAD，然后在 `dist/v0.6.0/` 生成以下文件：

- Linux 和 macOS 的 amd64/arm64 `.tar.gz` 归档。
- Windows 的 amd64/arm64 `.zip` 归档。
- 已绑定当前 tag 的 `install.sh` 和 `install.ps1`。
- `npm/` 下的六个平台二进制包、一个主启动包和独立的 `npm/checksums.txt`。
- 仅覆盖 AtomGit Release 附件（六个归档和两个安装脚本）的根 `checksums.txt`。

发布 npm 制品时，先发布 `npm/` 下六个名称带平台和架构的包；确认它们可用后，再发布 `atomgit-cli` 主包。主包和平台包必须使用相同版本。

上传 Release 附件前可校验所有制品：

```bash
# Linux
(cd dist/v0.6.0 && sha256sum -c checksums.txt)

# macOS
(cd dist/v0.6.0 && shasum -a 256 -c checksums.txt)
```

npm tarball 不作为 AtomGit Release 附件上传，可在发布到 npm registry 前单独校验：

```bash
(cd dist/v0.6.0/npm && shasum -a 256 -c checksums.txt)
```

未创建 tag 时，可使用 `make release-snapshot VERSION=v0.6.0` 进行本地试打包。Snapshot 允许脏工作区，其制品仅用于验证，不应上传到正式 Release。

底层 `scripts/build-release.sh` 也接受 `TAG`、`AG_RELEASE_SNAPSHOT=1` 和 `SOURCE_DATE_EPOCH` 环境变量。`SOURCE_DATE_EPOCH` 会同时固定二进制中的构建日期以及归档内文件的时间戳，用于生成可复现的发布制品；历史两段式 tag 仅保留给 snapshot 兼容。

### 维护 Nix package

仓库 flake 提供两种 package：

- `stable` 从对应版本的 AtomGit Release 源码归档构建，并固定源码 hash 和 `vendorHash`。
- `latest` 从当前 flake revision 的源码构建，使用独立的 `vendorHash`。

`default` 和兼容名称 `ag` 都指向 `stable`。更新 Nix package 时，推荐先进入 flake 提供的开发环境：

当前 nixos-unstable 已停止支持 Intel macOS，因此 flake 仅为 `x86_64-darwin` 使用仍受维护的 `nixpkgs-26.05-darwin` input；其他平台继续使用 nixos-unstable。

```bash
# 默认更新 stable 的版本、Release 源码 hash 和 vendorHash
nix develop
./scripts/update-nix-package.sh v0.6.0

# 只更新当前源码对应的 latestVendorHash
./scripts/update-nix-package.sh --latest
```

也可以不进入开发环境直接运行，但需要预先安装 Nix 和 Git，并要求 `tar` 支持以 NUL 分隔的文件列表；Linux 上的 GNU tar 和 macOS 默认的 bsdtar 均受支持。脚本使用 `nix store prefetch-file` 计算 stable 源码 hash，并通过 `buildGoModule` 校验对应的 `vendorHash`。更新后会构建目标 package 并执行 `ag version --json`；验证失败时自动恢复原始 `flake.nix`，且不会提交、打标签或推送。

## 项目结构

```
atomgit-cli/
├── .goreleaser.yaml            # GoReleaser 跨平台打包配置
├── Makefile                    # 构建、测试、安装和发布入口
├── install.sh                  # Linux/macOS 安装脚本
├── install.ps1                 # Windows 安装脚本
├── bin/ag.js                   # npm 主包的平台二进制启动器
├── scripts/build-npm-packages.js # 从 Release 归档生成七个 npm 包
├── cmd/ag/main.go              # 入口
├── internal/
│   ├── agcmd/cmd.go            # 核心命令处理
│   ├── config/config.go        # 配置管理
│   ├── version/version.go      # 版本元数据
│   └── api/
│       ├── client.go           # API 客户端
│       ├── types.go            # API v5 数据类型
│       └── actions/            # Actions API v8 客户端与类型
├── pkg/
│   ├── cmdutil/factory.go      # 命令工厂
│   └── cmd/
│       ├── root/root.go        # 根命令
│       ├── auth/auth.go        # 认证命令
│       ├── browse/browse.go    # Browse 命令
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
│       ├── label/              # 标签命令
│       │   └── label.go
│       ├── run/                # Actions 运行、job、日志和 artifact 查看
│       ├── license/            # License 命令
│       │   ├── license.go
│       │   └── check.go
│       ├── ssh-key/ssh_key.go  # SSH key 命令
│       └── version/version.go  # 版本命令
├── scripts/build-release.sh    # GoReleaser 打包包装脚本
└── go.mod
```

## API

常规仓库功能使用 AtomGit API v5：`https://api.atomgit.com/api/v5`。

Actions 运行检查使用独立的 AtomGit API v8：`https://api.atomgit.com/api/v8`。

## 参考

- [AtomGit API 文档](https://docs.atomgit.com/docs/apis/)
- [GitHub CLI](https://cli.github.com/)

## License

[木兰宽松许可证第2版](LICENSE) (Mulan Permissive Software License, Version 2)

Copyright (c) 2026 AtomGit CLI Contributors
