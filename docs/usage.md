# 命令使用指南

本文档介绍 AtomGit CLI 各命令的常用参数和示例。安装方法请参阅[安装指南](installation.md)，认证与其他配置请参阅[配置指南](configuration.md)。

所有命令均可通过 `--help` 查看完整参数，例如：

```bash
ag pr --help
ag pr create --help
```

## 认证

```bash
# 浏览器 OAuth 登录并写入令牌文件
ag auth login
# 已登录时会提示无需重复登录；若要重新走浏览器：ag auth login --force
# 可为账号保存提交身份覆盖值
ag auth login --force --git-name "Alice" --git-email alice@example.com

# 无浏览器环境（沙箱、容器、CI）：从标准输入读取已有访问令牌登录
# 令牌会先经 AtomGit 用户接口验证，通过后才写入令牌文件
echo "$TOKEN" | ag auth login --with-token
ag auth login --with-token < token.txt

# 列出账号（不会输出 token）并切换活动账号及 Git identity
ag auth list
ag auth list --json
ag auth switch alice

# 默认同步当前仓库，也可覆盖 identity、修改全局配置或禁用同步
ag auth switch alice --git-name "Alice" --git-email alice@example.com
ag auth switch alice --global
ag auth switch alice --no-git

# 用 refresh_token 刷新 access_token（需之前登录响应里包含 refresh_token）
ag auth refresh

# 查看认证状态
ag auth status

# 显示当前 token
ag auth token

# 删除非活动账号、最后一个活动账号，或全部账号
ag auth logout
ag auth logout --account alice
ag auth logout --all
```

`auth status`、`auth token` 和 `auth refresh` 始终使用活动账号。首次登录的账号会自动成为活动账号；后续 `auth login --force` 只新增或更新账号，不会隐式切换，需使用 `auth switch` 显式选择。

`auth switch` 默认同时切换活动凭据并写入当前仓库的 `git config --local user.name/user.email`；只有 `--global` 才修改全局配置，`--no-git` 可明确禁用同步。API 邮箱缺失时 CLI 不会猜测邮箱，必须通过登录时的 `--git-email`、切换时的 `--git-email` 或 `--no-git` 明确处理。Git identity 写入失败不会切换活动凭据；凭据切换失败时会尝试恢复原 Git identity。该流程使用补偿式回滚降低半完成风险，但不承诺跨凭据文件和 Git 配置的完整原子性。

`auth logout` 默认删除活动账号。为避免删除动作隐式选择另一个账号并造成 Git identity 错配，当仍有其他账号时不能删除活动账号；应先执行 `auth switch <account>`，再用 `auth logout --account <old-account>` 删除原账号。`--account` 可直接删除非活动账号，`--all` 明确删除全部账号。access token、refresh token 不会写入 Git 配置、remote URL、账号列表或错误信息。

可选环境变量（覆盖默认 OAuth 应用）：`AG_OAUTH_CLIENT_ID`、`AG_OAUTH_CLIENT_SECRET`；若本机 **8765** 端口被占用，可设置 **`AG_OAUTH_REDIRECT_PORT`**（需与 AtomGit 应用配置的回调地址一致）。

## 仓库 (repo)

```bash
# 列出仓库（默认显示 30 条）
ag repo list

# 列出指定用户或组织的公开仓库
ag repo list alice
ag repo list hust-open-atom-club

# 指定最多列出100条仓库
ag repo list --limit 100

# 输出 JSON 数组
ag repo list --json

# 查看仓库详情
ag repo view
ag repo view owner/repo

# 输出 JSON 对象
ag repo view owner/repo --json

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


# 将 Fork 的默认分支与上游同步（不修改本地 Git 工作区）
ag repo sync
ag repo sync owner/fork


# 同步指定分支；强制覆盖分叉提交需确认或显式 --yes
ag repo sync owner/fork --branch develop
ag repo sync owner/fork --branch develop --force
ag repo sync owner/fork --branch develop --force --yes

# 删除仓库
ag repo delete --yes
ag repo delete owner/repo --yes
```

`ag repo edit` 仅发送命令行中明确指定的字段，支持 `--name`、`--description`、`--default-branch` 和 `--visibility public|private`。`--public`、`--private` 是可见性的便利选项；它们与 `--visibility` 三者互斥。名称或可见性修改需要交互确认，可使用 `--yes` 跳过确认。成功后命令会显示更新后的仓库名称和浏览器 URL。

该命令不会修改仓库 URL 路径、所有者、主页、LFS、模块开关、合并策略，也不会接受后静默忽略 GitHub CLI 的其他仓库设置选项。

`ag repo sync` 仅更新 AtomGit 上的远端 Fork。命令会先验证仓库确为 Fork、上游存在且目标分支在两端都可读取；未指定 `--branch` 时使用 Fork 的默认分支。默认同步不会覆盖分叉提交，冲突时返回非零退出码。`--force` 可能覆盖 Fork 上的分叉提交，因此需要交互确认；仅在已审查目标后才应结合 `--yes` 使用。

### 仓库内容读取

```bash
# 读取默认分支上的文件内容
ag repo read-file owner/repo README.md
ag repo read-file owner/repo src/main.go --json

# 在指定分支、tag 或 commit 上读取文件
ag repo read-file owner/repo README.md --ref dev

# 列出目录内容（使用 . 表示仓库根目录）
ag repo read-dir owner/repo .
ag repo read-dir owner/repo src --ref v1.0.0 --json
```

`read-file` 输出解码后的文件文本；`read-dir` 每行输出一个条目（类型、大小、路径），字段中的反斜杠、制表符、换行和回车分别显示为 `\\`、`\t`、`\n` 和 `\r`。`--json` 输出稳定的 lowerCamelCase 字段：文件对象包含 `name`、`path`、`sha`、`size`、`encoding`、`content`（base64）和 `ref`；目录条目数组包含 `name`、`path`、`type`、`sha`、`size`，空目录输出 `[]`。

路径必须是仓库相对路径，不能以 `/` 开头或结尾，不能包含连续斜杠或 `.`/`..` 段（`read-dir .` 是唯一例外，映射到仓库根目录）。每个路径段独立转义。文件内容默认经过终端清理；如需保留原始字节，使用根级 `--raw-output`。这些命令只发送 GET 请求，不会修改仓库内容。

### 仓库协作者

```bash
# 列出协作者并查看有效权限及权限来源
ag repo collaborator list owner/repo --limit 50
ag repo collaborator view owner/repo octocat
ag repo collaborator view owner/repo octocat --json

# 添加、调整或移除直接协作者
ag repo collaborator add owner/repo octocat --permission push
ag repo collaborator edit owner/repo octocat --permission admin
ag repo collaborator edit owner/repo octocat --permission pull --yes
ag repo collaborator remove owner/repo octocat
ag repo collaborator remove owner/repo octocat --yes
```

AtomGit 内置协作者权限为 `pull`（参与者）、`push`（开发者）和 `admin`（仓库维护者）。`list` 和 `view` 会明确标记直接权限或权限来源；组织继承权限不能通过仓库级命令修改。降权和移除操作默认要求确认，可使用 `--yes` 跳过确认。

### 仓库 Webhook

```bash
# 列出和查看 Webhook（输出不包含 secret）
ag repo webhook list owner/repo --limit 50
ag repo webhook view owner/repo 42 --json

# 从环境变量、文件或标准输入安全读取 secret
ag repo webhook create owner/repo --url https://example.com/hook --events push,issues --secret-env WEBHOOK_SECRET
ag repo webhook create owner/repo --url https://example.com/hook --events merge-requests --secret-file ./webhook-secret
Get-Content ./webhook-secret | ag repo webhook edit owner/repo 42 --secret-stdin --encryption signature

# 替换事件、删除或发送真实测试请求
ag repo webhook edit owner/repo 42 --events push,tag-push,merge-requests
ag repo webhook edit owner/repo 42 --events none
ag repo webhook test owner/repo 42
ag repo webhook delete owner/repo 42 --yes
```

支持的事件为 `push`、`tag-push`、`issues`、`note` 和 `merge-requests`。Webhook secret 不支持命令行明文参数，只能通过 `--secret-env`、`--secret-file` 或 `--secret-stdin` 三选一提供，也不会出现在列表、详情、JSON 或错误响应中。`test` 会向真实目标发送请求，和删除操作一样默认要求确认。AtomGit 当前公开 API 仅在响应中提供 `active`，因此 CLI 将启用状态作为只读信息展示，不发送未公开的修改字段。

## 组织 (org)

```bash
# 列出当前账号所属的组织
ag org list

# 限制返回数量
ag org list --limit 100

# 输出固定字段的 JSON 数组
ag org list --json
```

## Branch

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

# 查看保护分支规则（输出会区分 exact 与 wildcard）
ag branch protection list owner/repo
ag branch protection view owner/repo main
ag branch protection view owner/repo "release/*"

# 创建保护规则；新规则必须同时指定 push 与 merge 权限
ag branch protection set owner/repo main --push admin --merge admin
ag branch protection set owner/repo main --push maintainer --merge admin
ag branch protection set owner/repo "release/*" --push "develop;alice" --merge admin

# 仅修改已有规则的推送权限；未指定的合并权限保持不变
ag branch protection set owner/repo main --push "" --yes

# 删除规则（默认显示当前规则并要求确认）
ag branch protection delete owner/repo "release/*"
ag branch protection delete owner/repo "release/*" --yes
```

保护规则的 `--push` 与 `--merge` 接受由英文分号分隔的 `develop`、`admin`、`maintainer` 或用户名；显式传入空字符串表示不允许任何人执行该操作。AtomGit 对精确分支规则的优先级高于匹配的 wildcard 规则。CLI 只管理官方 API 暴露的推送与合并白名单，不修改评审、流水线等其他保护设置。更新接口要求同时提交两类权限，因此 CLI 会先读取现有规则并保留未显式修改的一侧；若服务端返回无法无损表示的旧权限，命令会停止并要求显式提供该权限。

## Commit

```bash
# 列出仓库提交（默认显示 30 条）
ag commit list owner/repo
ag commit list owner/repo --limit 100

# 从指定 SHA 或分支名开始列出
ag commit list owner/repo --ref main
ag commit list owner/repo --ref abcdef1

# 只列出修改过指定文件的提交
ag commit list owner/repo --path src/main.go

# 按时间范围过滤（RFC 3339 格式）
ag commit list owner/repo --since 2024-11-08T16:25:44Z
ag commit list owner/repo --since 2024-11-08T16:25:44Z --until 2024-12-01T00:00:00Z

# 查看提交详情
ag commit view owner/repo abcdef1234567890abcdef1234567890abcdef12

# 以 JSON 输出或在浏览器中打开
ag commit list owner/repo --json
ag commit view owner/repo abcdef1 --json
ag commit view owner/repo abcdef1 --web

# 比较两个 commit、分支或 tag
ag commit compare main...feature
ag commit compare owner/repo release/1.0...feature/new-ui

# 输出固定字段的 JSON 对象
ag commit compare owner/repo main...feature --json

# 输出单个 commit 的 diff 或 email-style patch 文本
ag commit diff owner/repo <sha>
ag commit patch owner/repo <sha>

# 为 git apply、git am 或文件保存保留原始字节
ag --raw-output commit diff owner/repo <sha>
ag --raw-output commit patch owner/repo <sha>
```

`commit compare` 接受 commit SHA、分支名或 tag，比较参数必须使用 `<base>...<head>` 格式。省略仓库时会从当前 Git 仓库推断；包含 `/` 的 ref 会作为单个路径参数安全编码。

文本比较输出包含 base、merge base、提交列表和文件统计；文件行使用独立的元数据列标记二进制文件及服务端截断的文件，字段中的制表符、换行和反斜杠会转义。空比较仍会输出零提交、零文件；JSON 模式中的 `commits` 和 `files` 固定为空数组。`diff` 与 `patch` 不做 JSON 解码，默认仍遵循全局终端安全清理；保存为可直接处理的原始 diff/patch 时使用 `ag --raw-output commit diff ...` 或 `ag --raw-output commit patch ...`。接口文档声明的成功状态 `200` 会直接输出正文，其他状态会作为命令错误返回。

## Browse

在默认浏览器中打开仓库页面或指定资源：

```bash
# 打开当前仓库首页（需在 git 仓库内运行）
ag browse

# 打开指定仓库
ag browse --repo owner/repo

# 打开 Issue 或 PR
ag browse 42

# 打开文件（默认分支）
ag browse main.go

# 打开文件并定位到指定行
ag browse main.go:312
ag browse main.go:312-320
ag browse main.go:312..320

# 在指定分支上打开文件
ag browse --branch dev main.go:42

# 在指定 commit 上打开文件
ag browse --commit abc1234 main.go

# 打开 Releases 页面
ag browse --releases


# 打开 Actions 页面
ag browse --actions


# 打开 Wiki 页面
ag browse --wiki


# 打开 Settings 页面
ag browse --settings

# 只打印 URL，不打开浏览器
ag browse --no-browser
```

## Pull Request (pr)

```bash
# 列出 PR
ag pr list
ag pr list owner/repo
ag pr list owner/repo --state closed
ag pr list owner/repo --json

# 查看 PR
ag pr view 123
ag pr view owner/repo 123
ag pr view owner/repo 123 --json

# 在浏览器中打开 PR
ag pr view owner/repo 123 --web

# 修改 PR 标题或正文
ag pr edit owner/repo 123 --title "Updated title"
ag pr edit owner/repo 123 --body "Updated description"

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
ag pr create owner/repo --title "Fix bug" --body-file description.md --base main --head feature-branch
cat description.md | ag pr create owner/repo --title "Fix bug" --body-file - --base main --head feature-branch
ag pr create owner/repo --title "Fix bug" --head feature-branch \
  --assignee alice --reviewer bob --tester carol --label Bug --milestone v1.0

# 修改 PR 协作元数据
ag pr edit owner/repo 123 --add-assignee alice --remove-assignee bob
ag pr edit owner/repo 123 --add-reviewer carol --remove-reviewer dave
ag pr edit owner/repo 123 --add-tester erin --add-label "Priority:High"
ag pr edit owner/repo 123 --remove-label Bug --milestone v1.1
ag pr edit owner/repo 123 --milestone none

# 查看、关联或取消关联 PR 对应的 Issue
ag pr issues owner/repo 123
ag pr link-issues owner/repo 123 --issue 42 --issue 43
ag pr unlink-issues owner/repo 123 --issue 42

# 查看 PR 当前提交的 CI 检查；等待检查完成
ag pr checks owner/repo 123
ag pr checks owner/repo 123 --watch --interval 5s

# 关闭 PR
ag pr close owner/repo 123

# 重新打开 PR
ag pr reopen owner/repo 123

# 检出 PR 到本地
ag pr checkout owner/repo 42
ag pr checkout owner/repo 42 --branch review-fix
ag pr checkout owner/repo 42 --force
ag pr checkout owner/repo 42 --detach
ag pr checkout owner/repo 42 --recurse-submodules

# 查看 PR 的提交、文件变更和反应
ag pr commits owner/repo 42
ag pr commits owner/repo 42 --limit 50 --json
ag pr files owner/repo 42
ag pr files owner/repo 42 --json
ag pr reactions owner/repo 42
ag pr reactions owner/repo 42 --json
```

`pr commits`、`pr files` 和 `pr reactions` 都是只读命令，只发送 GET 请求。`pr commits` 支持 `--limit`（默认 30，必须为正整数）控制返回数量。文本模式每行输出一个条目摘要；JSON 模式输出稳定的 lowerCamelCase 数组，无结果时输出 `[]`。

`pr view --json` 在现有字段基础上新增 `assignees`、`approvalReviewers`、`testers`（均为字符串数组，空时为 `[]`）和 `milestone`（对象或 `null`）字段。`pr list --json` 的 schema 保持不变。

跨仓库创建 PR 时 `--head` 的写法请参阅[跨仓库 PR 示例](cross_repo_pr_demo.md)。

负责人（assignee）负责后续工作，批准审查人（approval reviewer）负责批准变更，测试人（tester）负责验证变更；三个 AtomGit 角色相互独立。用户账号、标签和里程碑会在修改 PR 前解析，标签和里程碑必须已存在。`pr edit` 只修改显式传入的字段，添加和移除参数可重复使用，也可用逗号一次传入多个值。

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

# 编辑评论（交互式编辑）
ag pr comment edit owner/repo 123 456
ag pr comment edit owner/repo 123 456 --body "Updated comment"

# 删除评论
ag pr comment delete owner/repo 123 456
ag pr comment delete owner/repo 123 456 --yes

# 回复评论（PR 特有）
ag pr comment reply owner/repo 123 456 --body "Thanks for the feedback!"
```

## Issue

```bash
# 列出 Issue
ag issue list
ag issue list owner/repo
ag issue list owner/repo --state all
ag issue list owner/repo --json

# 查看 Issue
ag issue view 42
ag issue view owner/repo 42
ag issue view owner/repo 42 --json

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

# 创建 Issue 时指派负责人
ag issue create owner/repo --title "Bug report" --assignee alice

# 修改已有 Issue 的负责人（需要确认，使用 --yes 跳过）
ag issue edit owner/repo 42 --assignee alice
ag issue edit owner/repo 42 --assignee alice --yes
ag issue edit owner/repo 42 --remove-assignee --yes

# 创建 Issue
ag issue create owner/repo --title "Bug report" --body "Description"
ag issue create owner/repo --title "Bug report" --body-file description.md
cat description.md | ag issue create owner/repo --title "Bug report" --body-file -

# 关闭 Issue
ag issue close owner/repo 42

# 重新打开 Issue
ag issue reopen owner/repo 42
```

`--assignee` 接受一个非空用户登录名。`--assignee` 和 `--remove-assignee` 互斥。创建 Issue 时设置负责人是纯新增操作，不需要确认；修改已有 Issue 的负责人（设置或清除）默认需要确认，确认提示输出到 stderr 以免混入正常命令输出，`--yes` 可跳过确认。

### Issue 关联 PR 与分支

```bash
# 列出与 Issue 关联的 Pull Request
ag issue prs owner/repo 42
ag issue prs owner/repo 42 --json

# 列出 Issue 的关联分支
ag issue branches owner/repo 42
ag issue branches owner/repo 42 --json

# 添加或移除关联分支（移除需要确认，--yes 跳过）
ag issue branches owner/repo 42 --add feature/fix
ag issue branches owner/repo 42 --add feature/fix --add feature/docs --yes
ag issue branches owner/repo 42 --remove old-branch --yes
ag issue branches owner/repo 42 --add new-feature --remove stale-feature --yes
```

`issue prs` 文本模式每行输出一个关联 PR（编号、标题、状态），JSON 模式输出稳定数组，无结果时输出 `[]`。

`issue branches` 不带 `--add`/`--remove` 时列出当前关联分支。`--add` 和 `--remove` 可重复使用，可以同时添加和移除不同分支。空名称、重复名称、同一分支同时出现在添加和移除集合中都会在认证前被拒绝。移除操作默认需要确认，确认提示输出到 stderr，`--yes` 跳过确认。

AtomGit 关联分支接口使用整列表替换语义，因此 CLI 采用读-改-写流程：先 GET 当前列表，计算目标列表（保留现有顺序，追加新分支，移除指定分支），仅在目标列表与当前列表不同时发送一次 PUT。PUT 不会自动重试。如果目标列表与当前列表相同，则只发送 GET，不发送 PUT。

**并发注意**：由于接口没有提供条件写令牌或原子增删操作，如果在 GET 和 PUT 之间另一个客户端修改了关联列表，本次 PUT 可能覆盖其变更。CLI 只能保证保留 GET 快照中的关联，不能防止并发写入竞态。

#### Issue 评论

```bash

# 创建评论
ag issue comment create owner/repo 42 --body "I can reproduce this issue"
ag issue comment create owner/repo 42 --body-file details.md

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

# 编辑评论（交互式编辑）
ag issue comment edit owner/repo 42 789
ag issue comment edit owner/repo 42 789 --body "Updated information"

# 删除评论
ag issue comment delete owner/repo 42 789
ag issue comment delete owner/repo 42 789 --yes
```

## Tag

```bash
# 在当前 Git 仓库中列出标签
ag tag list

# 显式指定仓库
ag tag list owner/repo
ag tag list owner/repo --json

# 创建或删除标签
ag tag create v1.0.0 --ref main
ag tag delete v1.0.0
```

## 资源命令的 JSON 输出

`repo list/view`、`issue list/view`、`pr list/view`、`tag list` 和 `commit list/view/compare` 支持布尔参数 `--json`。list 命令输出完整 JSON 数组，view 与 compare 命令输出完整 JSON 对象；没有结果时 list 输出 `[]`。默认文本输出保持不变。

JSON 字段使用 lowerCamelCase，并由 CLI 显式定义，不会因为 AtomGit API 增加字段而自动改变。Issue 和 PR 的 `number` 始终是字符串，标签输出为名称数组，PR 的 `head` 和 `base` 输出分支名称。可选的服务端字段缺失时仍输出对应的零值，以保持固定结构。

`view --json` 与 `view --web` 互斥；JSON 模式只向标准输出写入一个 JSON 值，不混入浏览器提示或其他文本。原始字节流命令（例如 `ag pr diff`）不提供 JSON 包装。

## Label

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


## Milestone

Milestone 日期使用明确的 `YYYY-MM-DD` 格式。关闭 Milestone 会保留数据，删除则会永久移除并默认要求确认。AtomGit API 要求每次更新都包含 `title` 和 `due_on`，因此 edit、close 和 reopen 会先读取当前 Milestone 并原样保留这两个必填字段；其他未指定字段不会发送。

```bash

# 列出和查看 Milestone
ag milestone list owner/repo --state all --limit 50
ag milestone view owner/repo 12
ag milestone view owner/repo 12 --json


# 创建和修改 Milestone
ag milestone create owner/repo --title "Version 1.0" --description "Release scope" --due-on 2026-08-31
ag milestone edit owner/repo 12 --title "Version 1.1" --due-on 2026-09-30


# 关闭、重新打开或永久删除
ag milestone close owner/repo 12
ag milestone reopen owner/repo 12
ag milestone delete owner/repo 12
ag milestone delete owner/repo 12 --yes
```

## Actions 运行记录 (run)

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

## Actions 工作流管理 (workflow)

`ag workflow` 用于查看 AtomGit Actions 工作流列表以及手动触发工作流运行（`workflow_dispatch`）。

```bash
# 列出仓库下的工作流及其 ID
ag workflow list owner/repo
ag workflow list owner/repo --json

# 手动触发工作流运行（按工作流 ID 或名称/路径）
ag workflow run owner/repo 12345 --ref main
ag workflow run owner/repo ci.yml --ref feature-branch -f env=production -F debug=true
```

`ag workflow run` 可以按工作流 ID、名称或相对路径（basename）选择目标；名称或 basename 同时匹配多个工作流时会报错并要求改用精确的 workflow ID 或完整路径。`--ref` 缺省时使用仓库的默认分支（而非假定为 `main`），无法确定默认分支时需显式传入 `--ref`。`-f/--raw-field` 与 `-F/--field` 均为 `key=value` 格式，参数值会作为 `workflow_dispatch` 的 inputs 传递。


## 通用 API 请求

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

## Release

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

# 下载附件；-o/--output 必填，默认不覆盖已有文件
ag release download v1.0.0 app.tar.gz -o ./dist/app.tar.gz
ag release download owner/repo v1.0.0 app.tar.gz --output ./existing.tar.gz --overwrite
```

`ag release create` 必须通过 `--body` 或 `--body-file` 提供非空说明，这是 AtomGit 创建 Release API 的必填字段。`ag release edit` 只改变用户明确指定的内容（`--name`、`--body` 或 `--body-file`、`--latest`、`--prerelease`），未指定的 name/body 会从当前 Release 回读并保持不变；状态仅在显式 `--latest` 或 `--prerelease` 时改变。`--body` 与 `--body-file` 互斥。`--latest` 将该 Release 标记为仓库最新发布。

`ag release upload` 的远端附件名默认为本地文件名（`filepath.Base(file)`），可用 `--name` 指定。若远端已存在同名附件，默认会报错并提示选择 `--skip-existing` 或 `--overwrite`，**绝不静默覆盖**：

- `--skip-existing`：发现同名附件即报告成功并退出，**不会修改远端**，也不会执行删除、查询上传地址或上传操作。
- `--overwrite`：仅当远端唯一匹配且该附件 `type=attach`、ID 为正整数时，先成功取得上传地址，再删除旧附件并上传新文件；取得上传地址失败时旧附件保持不变。若删除响应中断，命令会重新读取 Release，仅在确认旧附件已经不存在时继续；若后续上传失败，错误信息会明确说明旧附件已被删除。对于 `type=source` 的源码归档、ID 非正、或存在多个同名匹配的情况，均会拒绝且不执行删除。

附件传输不会使用普通 API 请求的 30 秒总超时，而是默认限制为 30 分钟，可用 `--timeout` 调整，或用 `--timeout 0` 关闭总时限。非空文件只会在传输尚未开始（请求体零字节被读取）时自动重试一次；零字节文件或传输开始后的中断会返回非零退出码并提示远端状态可能不确定，不会盲目重放上传。

`ag release download` 的 `-o/--output` 为必填项，命令不会将二进制写入 stdout。默认若目标文件已存在会立即失败且不发任何请求，只有显式 `--overwrite` 才允许替换。下载体先写入目标目录中的临时文件，完整接收后才安装到目标路径；传输失败会保留已有目标文件且不残留临时文件。下载与上传一样默认限制为 30 分钟，可通过 `--timeout` 调整或用 `--timeout 0` 关闭总时限。

`ag run view --artifact` 下载的是 Actions workflow run 的 artifact，而 `ag release download` 下载的是仓库 Release 的附件，两者来源和标识不同：前者按 run 的 artifact ID 下载，后者按 Release tag 和附件名下载。

本组 `ag release` 命令是供手工或脚本调用的底层 Release 管理原语，不会自动执行版本 tag 校验、测试、跨平台构建、校验和生成或整套发布编排。tag 驱动的端到端自动发布由 Issue #18 跟踪。

## License

```bash
# 检查 license 合规性
ag license check MIT
ag license check Apache-2.0
ag license check GPL-3.0
```

## SSH Key

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

## 通知 (notification)

```bash
# 列出仓库通知（默认列出全部，按时间倒序）
ag notification list owner/repo

# 在当前 Git 仓库中直接使用
ag notification list

# 只看未读；--type 按 merge_requests_open、issue_open 等类型过滤
ag notification list owner/repo --unread
ag notification list owner/repo --type issue_open

# 按更新时间过滤（RFC 3339 时间戳）
ag notification list owner/repo --since 2026-08-01T00:00:00Z --before 2026-08-15T00:00:00Z

# 限制数量与结构化输出
ag notification list owner/repo --limit 20
ag notification list owner/repo --json

# 标记指定通知为已读（ID 来自 `ag notification list`）
ag notification mark-read owner/repo 292ecbec857e4f27b426d66f2157938c

# 标记仓库内全部未读通知为已读（默认要求确认，--yes 跳过）
ag notification mark-read owner/repo --all
ag notification mark-read owner/repo --all --yes
```

文本输出每行依次为未读状态（`unread`/`read`）、类型、更新时间、标题和链接；没有通知时输出 `No notifications found`，`--json` 输出 `[]`。`--type` 由 CLI 在本地过滤，`--limit` 在过滤后的结果上生效。

## 搜索

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

## 讨论

```bash
# 列举讨论的具体命令
ag discussion list
ag discussion list hust-open-atom-club/atomgit-cli
ag discussion list hust-open-atom-club/atomgit-cli --limit 30
ag discussion list hust-open-atom-club/atomgit-cli --json
```

公开仓库的 Discussion 可匿名读取；登录后会携带访问令牌，以便访问账号有权限查看的仓库。

## 版本

```bash
# 查看版本信息
ag version

# 等价的根级参数
ag --version

# 机器可读的 JSON 输出
ag version --json
```

通过 `make build` 或 `make install` 从源码构建且未注入发布元数据时，版本默认值为 `dev`。如果 Go 构建信息包含模块版本、源码提交或提交时间，命令会使用这些信息替代或补充默认值；工作区存在未提交改动时，版本还会带有 dirty 标记。

通过 `go install ...@latest` 从模块代理安装时，模块版本仍然可用，但由于源码包不包含 Git 历史，文本输出会省略无法获得的 commit 和构建时间，JSON 输出则将对应字段保留为 `unknown`。

## 检查 CLI 更新

```bash
ag check-update
```

`ag check-update` 公开查询 `hust-open-atom-club/atomgit-cli` 的稳定 Release，并按 SemVer 比较当前版本。命令不需要登录或仓库上下文，不下载制品，也不会修改当前安装。输出包含当前版本、最新稳定 Release 和比较状态；`dev`、dirty、提交哈希或其他无法比较的本地版本会返回清晰错误。

## 命令别名 (alias)

为常用命令创建快捷方式。别名在调用时展开：`ag` 的第一个非 flag 参数会在别名表中查找，命中后替换为对应的展开内容再执行。

```bash
# 设置别名（展开内容含空格时用引号让 shell 视为一个参数）
ag alias set pl "pr list"
ag alias set rv repo view
ag alias set open-prs "pr list --state open"

# 列出所有别名
ag alias list

# 删除别名
ag alias delete pl
```

- 别名存储在 `~/.config/ag-cli/config.json`（或 `$XDG_CONFIG_HOME/ag-cli/config.json`），文件权限为 `0600`；该文件只存放别名数据，凭据保存在独立的 `token.json` 中，两者不共用。
- 别名不会覆盖内置命令：`alias set` 会直接拒绝与内置命令同名的别名名（内置命令始终优先）。
- 展开内容的第一个命令必须是已有内置命令，`alias set` 会校验并拒绝未知命令或指向其他别名的展开（别名只展开一次）。
- 展开内容不能以 `!` 开头（不支持 shell 风格别名），也不能为空。
- 展开内容不支持 shell 引号解析，引号会被当作普通字符；单个参数内的空格需用 `\ ` 转义，例如 Windows 路径 `C:\Program\ Files` 会作为单个参数传递（这与 GitHub CLI 用 shlex 解析的规则不同）。
- 别名配置文件损坏时，`ag` 会在 stderr 输出警告并忽略别名继续执行，不会阻塞其他命令。
