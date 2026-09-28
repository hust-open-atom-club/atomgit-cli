# 命令使用指南

本文档介绍 AtomGit CLI 各命令的常用参数和示例。安装方法请参阅[安装指南](installation.md)，认证与其他配置请参阅[配置指南](configuration.md)。完整的命令树、参数和别名索引参阅[命令参考](command-reference.md)。

需要了解 API 覆盖边界、尚未实现的功能和维护归属时，请参阅 [OpenAPI 覆盖与责任清单](openapi-coverage.md)。

## 目录

- [健康检查 (doctor)](#健康检查-doctor)
- [机器可读命令说明 (schema)](#机器可读命令说明-schema)
- [认证](#认证)
- [仓库 (repo)](#仓库-repo)
- [组织 (org)](#组织-org)
- [用户 (user)](#用户-user)
- [Branch](#branch)
- [Commit](#commit)
- [Browse](#browse)
- [Pull Request (pr)](#pull-request-pr)
- [Issue](#issue)
- [Tag](#tag)
- [资源命令的 JSON 输出](#资源命令的-json-输出)
- [Label](#label)
- [Milestone](#milestone)
- [Actions 运行记录 (run)](#actions-运行记录-run)
- [Actions 工作流管理 (workflow)](#actions-工作流管理-workflow)
- [Actions Runner 查询 (runner)](#actions-runner-查询-runner)
- [通用 API 请求](#通用-api-请求)
- [Release](#release)
- [License](#license)
- [SSH Key](#ssh-key)
- [通知 (notification)](#通知-notification)
- [搜索](#搜索)
- [讨论](#讨论)
- [版本](#版本)
- [更新 CLI](#更新-cli)
- [命令别名 (alias)](#命令别名-alias)

所有命令均可通过 `--help` 查看完整参数，例如：

```bash
ag pr --help
ag pr create --help
```

## 机器可读命令说明 (schema)

```bash
ag schema                    # 当前 CLI 的公开命令目录
ag schema ag                 # 根命令详情
ag schema pr create          # 单个命令的参数和能力说明
ag schema ag pr create       # 也可直接使用目录返回的完整路径
ag schema api
ag schema pr comment create  # 嵌套命令
```

默认输出 JSON，无需 `--json`。查询不会执行目标命令，不需要登录或网络，
不读取 token、不迁移或修改凭据；损坏的认证配置也不阻断查询。
它适用于 Agent/脚本按需确认本机 CLI 的用法，减少 skills 中重复维护的参数表。
查询路径只接受完整命令名或内置别名；用户配置的动态别名不收录、不解析。
路径可带 `ag` 前缀，目录中的 `commands[].path` 按空格拆分后可直接作为查询参数。
`ag schema ag` 返回根命令详情；不传路径的 `ag schema` 仍返回命令目录。
未知路径或隐藏命令返回非零退出码，stdout 不输出部分结果。

这是**版本化命令描述格式，不是 JSON Schema**，也不是完整的参数验证器：

- 顶层 `formatVersion` 当前为 `1`；`cliVersion` 包含 CLI 的 `version`、`commit`、`buildDate`。
- 无参数时，`commands` 给出按规范路径排序的目录（包含根命令）；指定路径时，`command` 给出详情。
  别名查询返回规范路径，不生成重复目录项。公开的弃用入口保留 `deprecated` 提示。
  隐藏命令及其子树、隐藏 flags 均排除；运行时 Cobra 添加的公开 help/completion 辅助入口会收录。
- 详情包括 `usage`、描述、示例、子命令和公开 `flags`。flag 的 `type` 为 pflag 类型名，
  `default` 为声明时的字符串表示（例如布尔值 `"false"`、列表 `"[]"`），不会读取运行时值或解析动态默认值。
  `noOptDefault` 表示省略参数值时的取值；`inherited` 和 `definedOn` 说明继承关系。
- `required` 只表达已声明的必填规则，`requiredSource` 区分 `cobra`、显式 `annotation` 和 `none`。
  `flagGroups` 导出 Cobra 声明的 `mutuallyExclusive`、`requiredTogether`、`atLeastOneRequired` 约束；
  含隐藏 flag 的组不导出。没有声明不代表业务逻辑没有其他约束。
- `positionals` 来自命令旁的显式注解，包含 `minCount`、`maxCount`、描述；`maxCount: -1` 表示没有数量上限，
  `positionals: null` 表示未描述。首批补充 `pr create`、`api`、`pr comment create` 和 `schema`，其余可逐步补充。
- `capabilities.output` 可描述 `text`、`json` 或原始响应 `raw`；`effects` 可描述 `none`、`read`、`write`、`conditional`。
  未注解时均为 `undescribed`，不会从命令名称、HTTP 方法或存在 `--json` 推断。
  API 权限、业务响应结构当前也标记为 `undescribed`。
- `validation: "partial"` 和 `undescribed` 明确说明覆盖限制；业务校验说明可见 `notes`。
  schema 不替代实际命令校验、权限检查，也不授予执行写操作的授权。

同一格式版本内允许新增字段及能力描述，消费者应忽略未知字段，并处理 `null`/`undescribed`；
删除字段、改变字段类型或已有语义时提升 `formatVersion`。命令增删和参数变化随 CLI 版本变化，
不应把另一版本 CLI 的描述直接用于本机。Agent 宜按任务查询单个命令；旧版明确不支持 `schema` 时再回退到 `--help`。

## 健康检查 (doctor)

```bash
ag doctor
ag doctor --json
ag doctor --live
ag doctor owner/repo --live --json
```

默认仅检查本地 Git 可用性、别名配置、凭据格式/活动账号、Unix 凭据权限、已知到期时间和仓库上下文，不联网。`--live` 在总计 30 秒的超时内执行只读请求：API v5 连通性和当前用户认证，以及有仓库参数或可推断仓库时的仓库访问、Actions 工作流列表和 discussion 列表。显式仓库优先于 Git 推断。

结果为 `pass`、`warn`、`fail` 或 `skip`，每项包含稳定的检查 ID、说明及必要的下一步建议。JSON 包含 `version`、`platform`、`live`、`checks` 和 `ok`；出现 `fail` 时仍输出完整报告并返回非零退出码，仅有警告或跳过时返回零。未登录属于警告，缺少仓库上下文时跳过仓库相关探测。未启用联网检查不代表令牌已通过服务端认证。 `connectivity` 只表示网络可达，`service` 单独报告 API v5 服务状态：即使未登录，404、429、5xx 等错误也会使检查失败并返回非零退出码；401/403 则跳过服务状态判定，由认证检查报告访问被拒绝（未登录时认证检查也跳过）。

`doctor`（包括通过命令别名调用）不修改凭据权限、不迁移配置、不刷新令牌，也不会自动修复问题。Windows ACL 暂不检查；使用权限不安全的 Unix 凭据时跳过认证探测。HTTP 403 仅表示访问被拒绝，不能据此确定具体 OAuth scope；HTTP 404 也可能是资源不可见。只读探测成功不保证写权限可用。报告不输出令牌、账号名称、原始 Git remote URL、代理地址、HTTP 响应正文或原始底层错误。

已有凭据被服务端以 HTTP 401 拒绝时，运行 `ag auth login --force` 重新认证；普通 `ag auth login` 会因本地已有凭据而跳过登录。

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

# 查看本地认证状态（不联网、不输出令牌）
ag auth status
ag auth status --json

# 只读在线验证当前令牌对应的身份
ag auth status --verify
ag auth status --verify --json

# 让 Git HTTPS 操作安全复用 ag 当前活动账号的令牌
ag auth setup-git

# 显示当前 token
ag auth token

# 删除非活动账号、最后一个活动账号，或全部账号
ag auth logout
ag auth logout --account alice
ag auth logout --all
```

`auth status`、`auth token` 和 `auth refresh` 始终使用活动账号。首次登录的账号会自动成为活动账号；后续 `auth login --force` 只新增或更新账号，不会隐式切换，需使用 `auth switch` 显式选择。

`auth status` 默认只确认本地凭据及活动账号配置完整，不证明令牌有效。`--verify` 使用认证后的 v5 `GET /user`，支持取消，并限制在 30 秒内；只接受原接口的直接响应，不跟随重定向。验证成功仅说明该用户接口确认了身份，不代表拥有其他资源或服务的权限。账号比较不区分大小写；身份不一致时报告两侧账号，不自动切换或改写配置。

`--json` 始终输出单个对象，成功和运行时失败使用相同字段：

```json
{
  "ok": true,
  "host": "atomgit.com",
  "credentialsPresent": true,
  "localAccount": "alice",
  "localStatus": "configured",
  "verification": {
    "requested": false,
    "performed": false,
    "status": "not_requested",
    "account": null,
    "httpStatus": null
  },
  "message": "Local credentials are configured; online validity has not been verified."
}
```

- `ok` 为布尔值；本地状态正常且未请求验证，或请求的验证通过时为 `true`，退出码为 0；其他运行时结果为 `false`，退出码为 1，诊断写入 stderr。
- `credentialsPresent` 为 `true`（已读到完整的活动凭据）、`false`（没有凭据文件）或 `null`（配置不完整、损坏或无法读取，不能确认）。`localAccount` 为账号字符串，无法确定时为 `null`；`localStatus` 为 `configured`、`missing` 或 `invalid`。
- `verification.requested` 表示是否传入 `--verify`；`performed` 表示是否已尝试 API 请求，不表示验证成功。无可用本地凭据时跳过请求。
- `verification.status` 为 `not_requested`、`skipped`、`verified`、`identity_mismatch`、`unauthorized`（401）、`forbidden`（403）、`server_error`（5xx）、`http_error`（其他非预期 HTTP 状态）、`network_error`、`timeout`、`canceled`、`invalid_response` 或 `client_error`。403 和网络错误不等同于令牌过期。
- `verification.account` 为服务端确认的账号字符串；未取得有效身份时为 `null`。`httpStatus` 在取得有效身份或 HTTP 错误状态时为整数，否则为 `null`。无效 JSON、空身份等成功响应也会导致验证失败。
- `message` 是面向人的说明，脚本应判断上述状态字段。账号字段如包含已知凭据则显示 `[redacted]`；文本、JSON 及错误输出均不显示 token、token 片段或 refresh token。确需读取 token 时使用 `ag auth token`。

状态查询不登录、刷新、迁移凭据、修正权限或修改活动账号；旧版凭据路径仍可只读查询。新增 JSON 字段时消费者应忽略不认识的字段。

`auth setup-git` 会为 `https://atomgit.com` 写入全局、主机限定的 Git credential helper。之后，Git HTTPS 操作会调用 `ag auth git-credential`，从 `ag auth switch` 选中的活动账号读取用户名和访问令牌；令牌本身不会写入 Git 配置。该设置不影响 SSH remote，也不会为其他主机提供凭据。

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

# 查看仓库级推送规则
ag repo push-rule view
ag repo push-rule view owner/my-project
ag repo push-rule view owner/my-project --json

# 更新推送规则；默认显示目标仓库和变更字段并要求确认
ag repo push-rule edit owner/my-project --deny-force-push
ag repo push-rule edit owner/my-project --reject-not-signed-by-gpg --max-file-size 50

# 非交互式更新，或显式关闭/清空规则
ag repo push-rule edit owner/my-project --deny-force-push --yes
ag repo push-rule edit owner/my-project --reject-not-signed-by-gpg=false --yes
ag repo push-rule edit owner/my-project --commit-message-regex "" --max-file-size 0 --yes

# 只读查看推送远程镜像列表和仓库远程镜像状态
ag repo mirror list
ag repo mirror list owner/my-project --limit 100 --json
ag repo mirror view
ag repo mirror view owner/my-project --json

# 克隆仓库
ag repo clone owner/repo
ag repo clone owner/repo --branch dev

# Fork 仓库
ag repo fork
ag repo fork owner/repo
ag repo fork owner/repo --name my-fork --public

# 列出现有 Fork（只读；默认最多 30 条）
ag repo fork list owner/repo
ag repo fork list owner/repo --limit 100 --json
ag repo fork list


# 将 Fork 的默认分支与上游同步（不修改本地 Git 工作区）
ag repo sync
ag repo sync owner/fork


# 同步指定分支；强制覆盖分叉提交需确认或显式 --yes
ag repo sync owner/fork --branch develop
ag repo sync owner/fork --branch develop --force
ag repo sync owner/fork --branch develop --force --yes

# 将仓库转移到其他组织命名空间（默认需要确认）
ag repo transfer owner/repo --to target-organization
ag repo transfer owner/repo --to target-organization --yes

# 组织所属仓库需要账户密码；非交互使用时从标准输入安全读取
printf '%s\n' "$PASSWORD" | ag repo transfer source-organization/repo --to target-organization --yes --password-stdin

# 删除仓库
ag repo delete --yes
ag repo delete owner/repo --yes
```

`ag repo edit` 仅发送命令行中明确指定的字段，支持 `--name`、`--description`、`--default-branch` 和 `--visibility public|private`。`--public`、`--private` 是可见性的便利选项；它们与 `--visibility` 三者互斥。名称或可见性修改需要交互确认，可使用 `--yes` 跳过确认。成功后命令会显示更新后的仓库名称和浏览器 URL。

该命令不会修改仓库 URL 路径、所有者、主页、LFS、模块开关、合并策略，也不会接受后静默忽略 GitHub CLI 的其他仓库设置选项。

#### 仓库策略设置

```bash
# 默认查看全部三类设置，也可以只查看一类
ag repo policy view owner/repo
ag repo policy view --section permission --json
ag repo policy view owner/repo --section code-review
ag repo policy view owner/repo --section pull-request --json

# 权限模式：1 继承模式，2 独立模式；改变成员权限来源前请确认影响
ag repo policy edit owner/repo --section permission --mode 2
# 代码审查使用用户名，不是用户 ID；空字符串清空列表，0 保留为显式值
ag repo policy edit owner/repo --section code-review --assignees alice,bob --testers-number 0
ag repo policy edit owner/repo --section code-review --testers "" --yes
# PR 设置使用对应的独立字段；显式 false 不会被省略
ag repo policy edit owner/repo --section pull-request --can-force-merge=false --yes
ag repo policy edit owner/repo --section pull-request --merge-method ff --yes --json
# 禁用“合并后关闭已关联的 Issue”选项；传 false 可重新启用
ag repo policy edit owner/repo --section pull-request --forbidden-pr-related-issue-closed --yes
```

`view` 默认读取全部设置；指定 `--section permission|code-review|pull-request` 时只输出该类。权限模式来自 `GET /repos/{owner}/{repo}/transition`。PR 设置来自 `GET /repos/{owner}/{repo}/pull_request_settings`；代码审查查看是该响应中审批人、测试人及最低人数的投影，不调用未文档化的 `GET /reviewer`。响应中的人员信息保留 API 返回的对象，不能将其当作编辑接口的用户名字符串。

`edit` 必须指定一个 section 和至少一个该类字段，拒绝混用其他类参数。权限模式仅接受 `--mode 1|2`；代码审查支持 `--assignees`、`--testers`（逗号分隔用户名）及非负的 `--assignees-number`、`--testers-number`。PR 参数对应官方更新接口字段，将下划线改为连字符；完整列表见 `ag repo policy edit --help` 和[命令参考](command-reference.md)。其中 `--approval-approver-ids`/`--approval-tester-ids` 接受用户 ID 而非用户名；`--approval-required-reviewers` 为 0–5，其他人数为非负整数；`--merge-method` 为 `merge`、`rebase_merge` 或 `ff`，`--merged-commit-author` 为 `merged_by` 或 `created_by`。

每次更新仅发送显式参数，保留 `false`、`0` 和允许字段中的空字符串，不读取后回写整份设置。所有修改默认显示仓库、section 和变更字段并要求确认，`--yes` 跳过确认；提示写入 stderr，取消不会发出修改请求。三类编辑分别使用 `PUT /transition`、`PUT /reviewer`、`PUT /pull_request_settings`，只接受文档约定的 HTTP 200 及有效响应，不自动重试写入。网络失败时需先用 `view` 核实当前状态，再决定是否重试。

查看 JSON 使用固定的 `repository` 和 `settings`（以 section 为键）结构；未返回的可选设置显示为 `null`，不冒充 `false` 或 `0`。PR 响应中的布尔值、`0/1` 及布尔字符串统一显示为 `true/false`，更新响应也按布尔语义核对。`--forbidden-pr-related-issue-closed` 控制是否禁用“合并后关闭已关联的 Issue”选项；禁用后，`--close-issue-when-mr-merged` 的默认选择设置无效。编辑 JSON 为 `repository`、`section`、`changed_fields`，记录成功请求中发送的字段，不代表完整设置或额外在线回读。pull-request 更新会核对响应中返回的对应标量设置；与请求不一致时返回非零退出码，不输出成功结果。此时可能已有部分设置生效，请先查看策略再决定是否重试。接口未返回的字段和无法对应到审批人员对象的 ID 字段仅获请求成功确认，不代表逐字段验证通过。审批人员数组必须由 JSON 对象组成。接口权限不足、设置不可用或响应格式错误会返回非零退出码。仓库策略与 push-rule、分支/标签保护仍是独立命令。

接口依据：[仓库 API 字段说明](https://docs.gitcode.com/en/docs/repos/)、[权限模式更新](https://docs.gitcode.com/docs/apis/put-api-v-5-repos-owner-repo-transition/)、[代码审查更新](https://docs.gitcode.com/en/docs/apis/put-api-v-5-repos-owner-repo-reviewer/)、[PR 设置更新](https://docs.gitcode.com/docs/apis/put-api-v-5-repos-owner-repo-pull-request-settings/)。测试使用合成响应，不代表真实仓库写入已验证。

`ag repo push-rule view` 展示签名提交要求、提交信息正则、单文件大小限制、管理员豁免和强推限制。`ag repo push-rule edit` 只发送命令行中明确指定的字段，并保留显式的 `false`、空字符串和 `0`；未指定的远端规则保持不变。所有更新默认需要确认，可使用 `--yes` 跳过。仓库级推送规则与分支、标签保护规则相互独立。

`ag repo mirror list` 通过 `/push_remote_mirrors` 分页列出 AtomGit 上配置的推送镜像，`ag repo mirror view` 通过 `/repo_remote_mirror` 查看仓库镜像状态。两个命令都只发送 GET 请求，不会创建、修改或触发镜像同步，也不会同步或修改本地 Git remote。文本和 JSON 输出会删除镜像 URL 中的 userinfo、查询参数和 fragment，并对返回的错误或消息中的嵌入 URL 做同样处理；未由 API 返回的状态、时间和错误字段不会被补造。

`ag repo sync` 仅更新 AtomGit 上的远端 Fork。命令会先验证仓库确为 Fork、上游存在且目标分支在两端都可读取；未指定 `--branch` 时使用 Fork 的默认分支。默认同步不会覆盖分叉提交，冲突时返回非零退出码。`--force` 可能覆盖 Fork 上的分叉提交，因此需要交互确认；仅在已审查目标后才应结合 `--yes` 使用。

`ag repo transfer` 会改变仓库所有者，并可能影响仓库 URL、访问权限和自动化配置。当前公开的 AtomGit 转移接口仅描述组织目标，因此命令会先从当前账户可见的命名空间中解析 `--to`，并在确认或 POST 前拒绝个人用户、不可见目标和未知命名空间类型。命令随后显示源仓库和已解析的目标组织并要求确认，`--yes` 只跳过确认；组织所属仓库仍需输入 AtomGit 账户密码。交互终端中密码会隐藏输入，非交互场景必须结合 `--yes --password-stdin` 从标准输入读取。POST 返回后命令会读取目标仓库，只有仓库 ID 与源仓库一致，且完整名称和浏览器 URL 得到确认时才报告成功；如果 POST 未返回可判定的 HTTP 结果，命令会尝试目标回读，无法确认时将明确报告转移可能已完成但最终状态未知。该命令**不会修改本地 Git remote**，转移完成后请根据输出的新 URL 手动检查并更新本地 remote。

`ag repo fork list` 通过 `GET /repos/{owner}/{repo}/forks` 只读列出已有 Fork，支持仓库推断、分页、`--limit` 和 `--json`；不会创建或修改任何仓库。

### 仓库洞察

```bash
# 语言占比和贡献者统计
ag repo insights languages owner/repo
ag repo insights contributors owner/repo --limit 50

# 仓库事件、关注者和收藏者；省略仓库时从当前 Git 仓库推断
ag repo insights events --limit 100
ag repo insights watchers --limit 100 --json
ag repo insights stargazers owner/repo --json

# 下载统计
ag repo insights downloads owner/repo
ag repo insights downloads owner/repo --json
```

`ag repo insights` 下的命令都只发送 GET 请求。它们分别读取 `/languages`、`/contributors/statistic`、`/events`、`/subscribers`、`/stargazers` 和 `/download_statistics`。`events`、`watchers`、`stargazers` 会跨页读取并在达到 `--limit` 后停止；贡献者统计接口不分页，因此 `contributors --limit` 会对完整响应按提交数降序排列后截取。四个列表命令的默认上限均为 30。

文本输出中，语言按占比降序排列，贡献者按提交数降序排列，下载明细按日期降序排列；相同数值使用名称作为稳定的次级排序条件。空响应不会报错。每个子命令都支持 `--json`，并输出固定字段名：语言和列表结果为数组，下载结果包含 `periodDownloads`、`historyDownloads` 和 `details`。

### 仓库内容读取

```bash
# 列出当前仓库默认分支的根目录或嵌套目录
ag repo content list
ag repo content list docs/guides

# 列出显式仓库的根目录或指定 ref 下的嵌套目录
ag repo content list owner/repo .
ag repo content list owner/repo src --ref v1.0.0 --json

# 读取默认分支或指定 ref 上的文件
ag repo content view README.md
ag repo content view owner/repo src/main.go --ref dev
ag repo content view owner/repo README.md --json
```

`content list` 每行输出一个目录条目的类型、路径和对象 ID；`content view` 输出解码后的文件字节。字段中的反斜杠、制表符、换行和回车分别显示为 `\\`、`\t`、`\n` 和 `\r`。`--json` 保留 AtomGit API 返回的完整目录数组或文件对象，包括文件的 Base64 编码 `content` 和 `_links` 等元数据；空目录输出 `[]`。

路径必须是仓库相对路径，不能以 `/` 开头或结尾，不能包含连续斜杠或 `.`/`..` 段（`content list` 的 `.` 是唯一例外，表示仓库根目录）。不带参数的 `content list` 会推断当前仓库并列出根目录；单个参数始终作为推断仓库内的路径，因此列出显式仓库根目录时应使用 `owner/repo .`。每个路径段独立转义。文件内容默认经过终端清理；如需保留原始字节，使用根级 `--raw-output`。这些命令只发送 GET 请求，不会修改仓库内容。

原有的 `ag repo read-file` 和 `ag repo read-dir` 已弃用，但会继续保留以兼容已有脚本；请分别迁移到 `ag repo content view` 和 `ag repo content list`。

### 仓库协作者

```bash
# 列出协作者并查看有效权限及权限来源
ag repo collaborator list owner/repo --limit 50
ag repo collaborator view owner/repo octocat
ag repo collaborator view owner/repo octocat --json

# 添加、调整或移除直接协作者；移除命令也会尝试撤销待处理邀请
ag repo collaborator add owner/repo octocat --permission push
ag repo collaborator edit owner/repo octocat --permission admin
ag repo collaborator edit owner/repo octocat --permission pull --yes
ag repo collaborator remove owner/repo octocat
ag repo collaborator remove owner/repo octocat --yes
```

AtomGit 内置协作者权限为 `pull`（参与者）、`push`（开发者）和 `admin`（仓库维护者）。`list` 和 `view` 只显示已接受的协作者，并会明确标记直接权限或权限来源；AtomGit API v5 不公开待处理邀请，因此这两个命令无法列出或查看待处理邀请。组织继承权限不能通过仓库级命令修改。

`remove` 对已接受的直接协作者执行移除；如果用户不在已接受列表中，则会尝试撤销可能存在的待处理邀请。由于 API 不公开邀请状态，找不到邀请时命令会返回相应错误。降权和移除操作默认要求确认，可使用 `--yes` 跳过确认。文本模式下，关于 API 不公开待处理邀请的提示写入 stderr；这些变更命令不提供 `--json` 输出。

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

# 查看组织详情
ag org view hust-open-atom-club
ag org view hust-open-atom-club --json

# 列出组织成员和角色
ag org members hust-open-atom-club
ag org members hust-open-atom-club --limit 100 --json

# 列出组织仓库及其可见性、默认分支和活跃度信息
ag org repos hust-open-atom-club
ag org repos hust-open-atom-club --limit 100 --json

# 列出和查看组织级 Actions Runner Group
ag org runner-group list hust-open-atom-club
ag org runner-group view hust-open-atom-club <group-id>

# 查看 Runner Group 关联的主机 Runner、Kubernetes Runner Set 和可用仓库
ag org runner-group runners hust-open-atom-club <group-id> --limit 100
ag org runner-group runner-sets hust-open-atom-club <group-id> --json
ag org runner-group namespaces hust-open-atom-club <group-id> --json
```

`view` 输出组织路径、名称、可见性、描述和公开 URL。`members` 与 `repos` 支持分页，并在认证和网络请求前校验 `--limit`；空结果的 JSON 输出为 `[]`。仓库输出还包含描述、默认分支、主要语言、Star、Fork 和更新时间。

`ag org runner-group` 使用 Actions API v8，只读检查组织级 Runner Group；它与 `ag runner list/shared` 查看仓库直接配置或共享的主机 Runner 不同。`list`、`runners`、`runner-sets` 和 `namespaces` 默认最多返回 30 条并要求 `--limit` 为正数；文本输出包含组织和 Group ID，JSON 保留对应官方响应的固定 snake_case 字段。权限不足、Group 不存在或分页响应不完整时命令会明确失败，不会把部分数据当作完整结果。`namespaces` 会先查询 Group 详情，确认其存在且可访问，再读取共享仓库；不存在时返回非零退出码，存在但没有共享仓库时仍正常返回空列表。

## 组织看板 (kanban)

```bash
# 列出组织看板（默认最多 30 个）
ag kanban list hust-open-atom-club
ag kanban list hust-open-atom-club --limit 100 --json

# 查看看板详情
ag kanban view hust-open-atom-club 1234567890
ag kanban view hust-open-atom-club 1234567890 --json

# 查看看板中的 Issue 和 Pull Request
ag kanban items hust-open-atom-club 1234567890
ag kanban items hust-open-atom-club 1234567890 --limit 100 --json
```

这些命令只读访问组织看板。`list` 使用组织看板列表接口，`view` 使用看板详情接口，`items` 使用看板内容接口；看板 ID 和 `--limit` 会在认证与网络请求前校验。文本输出会区分 Issue 与 Pull Request，并在服务端返回时显示看板列状态。

## 用户 (user)

```bash
# 查看当前认证用户的资料
ag user view

# 查看公开用户资料（无需登录）
ag user view alice

# 输出 JSON
ag user view alice --json

# 在浏览器中打开资料页
ag user view alice --web

# 更新当前认证账号的资料（只发送显式提供的字段）
ag user edit --nickname "Alice" --company "Example Inc."

# 用空字符串清除字段，并以 JSON 输出更新后的资料
ag user edit --description "" --json

# 列出当前认证账号的邮件地址（会输出隐私信息）
ag user emails

# 输出稳定的 JSON 数组
ag user emails --json

# 列出当前认证账号参与的命名空间（默认 mode 为 intrant）
ag user namespaces

# 按项目关联或全部来源查询，并限制返回数量
ag user namespaces --mode project --limit 100
ag user namespaces --mode all

# 输出固定字段的 JSON 数组
ag user namespaces --json

# 列出当前认证账号的个人动态
ag user events

# 列出指定用户的个人动态，按年份过滤并限制总条数
ag user events alice --year 2026 --limit 50

# 输出稳定的 JSON 数组
ag user events alice --json

# 列出当前认证账号 star 的仓库
ag user starred
ag user starred --limit 100 --json

# 列出指定用户 star 的仓库
ag user starred alice

# 列出当前认证账号 watch 的仓库
ag user watching

# 列出指定用户 watch 的仓库
ag user watching alice --limit 100 --json
```

`ag user view --json` 输出稳定的 JSON 对象，字段始终齐全（空字符串、零计数和空数组也会输出，便于自动化区分"值为零/空"与"字段缺失"）：

```json
{
  "id": "686baf25160adc265d6cbdec",
  "login": "mudongliang",
  "name": "mudongliang",
  "email": "dzm91@hust.edu.cn",
  "url": "https://atomgit.com/mudongliang",
  "type": "User",
  "bio": "华中科技大学网络空间安全学院慕冬亮",
  "company": "",
  "website": "https://mudongliang.github.io/",
  "location": "",
  "followers": 2,
  "following": 1,
  "topLanguages": ["Go", "Shell", "PowerShell", "Markdown"]
}
```

字段说明：`id`/`login`/`name`/`email`/`url`/`type` 为用户基本信息；`bio`/`company`/`website`/`location` 为个人资料文本；`followers`/`following` 为关注计数；`topLanguages` 为仓库语言列表（缺失时输出 `[]`）。

`ag user edit` 需要认证，可通过 `--avatar`、`--nickname`、`--company`、`--description`、`--email`、`--github-account`、`--website` 和 `--location` 更新当前账号资料。命令只发送显式提供的字段，因此省略的字段保持不变；显式传入空字符串可清除支持的字段。默认文本输出账号和资料页 URL，`--json` 输出 API 返回的完整更新结果。该命令不上传头像文件、不验证邮箱，也不修改登录名。

`ag user emails` 需要认证，并且只会在明确调用时将当前账号的邮件地址输出到标准输出。文本模式显示邮件地址和状态；`--json` 输出固定 `email`、`state` 字段的数组。没有邮件地址时，文本模式输出 `No email addresses found.`，JSON 模式输出 `[]`。

`ag user namespaces` 需要认证，用于列出当前账号通过成员关系或项目关联可见的用户及群组命名空间。`--mode` 支持 `intrant`（默认值）、`project` 和 `all`；文本模式显示路径、名称、类型和 URL，`--json` 输出固定的 `id`、`path`、`name`、`url`、`type` 字段。`ag org list` 仍只列出组织，不受此命令影响。

`ag user events` 需要认证，用于列出用户个人动态。省略用户名时默认使用当前认证账号，显式用户名优先。`--year` 用于按年份过滤（`0` 表示不过滤，合法范围为 `1970`–`9999`）；`--limit` 控制跨游标页返回的总条数（默认 `30`，必须为正整数）。命令会跟随 API 返回的 `next` 游标继续请求，直到满足条数或没有更多游标，并检测重复游标以避免无限翻页。文本模式按日期从新到旧输出 `DATE`、`ACTION`、`PROJECT`、`TITLE` 四列；`--json` 输出稳定的数组（不是 API 按日期分组的对象），每个元素包含 `date`、`action`、`actionName`、`authorId`、`authorUsername`、`authorName`、`authorUrl`、`createdAt`、`projectId`、`projectName`、`targetId`、`targetIid`、`targetTitle`、`targetType`、`targetTypeFormat` 字段。没有动态时，文本模式输出 `No events found.`，JSON 模式输出 `[]`。

`ag user starred [<username>]` 和 `ag user watching [<username>]` 需要认证，分别列出 star 和 watch 的仓库集合。省略用户名时使用认证用户端点；显式用户名使用对应的公开用户端点，且用户名会作为单个 URL 路径段转义。两个命令的 `--limit` 均控制跨页返回的总仓库数（默认 `30`，必须为正整数）。文本模式输出无歧义的完整仓库名和 URL；`--json` 输出固定的 `id`、`fullName`、`url` 字段数组。空集合在文本模式下输出明确提示，在 JSON 模式下输出 `[]`。这些命令只读，不会 star、取消 star、watch 或取消 watch 仓库，也不会改变 `ag repo list` 的所有权及成员关系语义。

## Branch

```bash
# 列出远程分支（默认显示 30 条）
ag branch list owner/repo
ag branch list owner/repo --limit 100
ag branch list owner/repo --json
ag branch list --limit 100

# 查看远程分支详情
ag branch view owner/repo main
ag branch view owner/repo feature/foo
ag branch view main

# 从指定 ref 创建远程分支
ag branch create owner/repo feature/foo --ref main
ag branch create feature/foo --ref main

# 删除远程分支（默认需要确认；不会删除本地 Git 分支）
ag branch delete owner/repo feature/foo
ag branch delete owner/repo feature/foo --yes
ag branch delete feature/foo --yes

# 查看保护分支规则（输出会区分 exact 与 wildcard）
ag branch protection list owner/repo --limit 30
ag branch protection view owner/repo main
ag branch protection view owner/repo "release/*"
ag branch protection list
ag branch protection view main

# 创建保护规则；新规则必须同时指定 push 与 merge 权限
ag branch protection set owner/repo main --push admin --merge admin
ag branch protection set owner/repo main --push maintainer --merge maintainer
ag branch protection set owner/repo "release/*" --push "develop;alice" --merge "develop;alice"

# 仅修改已有规则的推送权限；未指定的合并权限保持不变
ag branch protection set owner/repo main --push "" --yes

# 删除规则（默认显示当前规则并要求确认）
ag branch protection delete owner/repo "release/*"
ag branch protection delete owner/repo "release/*" --yes
```

保护规则的 `--push` 与 `--merge` 接受由英文分号分隔的 `develop`、`admin`、`maintainer` 或用户名；显式传入空字符串表示不允许任何人执行该操作。AtomGit 要求拥有推送权限的角色或用户也在 `--merge` 中显式允许。AtomGit 对精确分支规则的优先级高于匹配的 wildcard 规则。CLI 只管理官方 API 暴露的推送与合并白名单，不修改评审、流水线等其他保护设置。更新接口要求同时提交两类权限，因此 CLI 会先读取现有规则并保留未显式修改的一侧；若服务端返回无法无损表示的旧权限，命令会停止并要求显式提供该权限。branch 命令省略 `owner/repo` 时使用当前 Git 仓库推断结果，显式参数始终优先。

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

### Commit 评论

commit 评论独立于 Issue 评论和 PR 评论，使用专门的 commit 评论接口；`view`、`edit`、`delete` 作用于仓库级评论 ID，传入 Issue/PR 评论的 ID 会被拒绝。评论 ID 按不透明字符串处理：`create` 返回的 ID（官方接口示例为 `12312sadsa` 这类非纯数字字符串）可直接传给 `view`、`edit`、`delete`，命令要求 ID 非空、由安全的路径字符（字母、数字、`-`、`.`、`_`、`~`）组成，并拒绝完整值 `.` 和 `..`，以免被规范化成当前或父路径。

```bash
# 列出某个 commit（完整或短 SHA、分支名）下的评论（默认 30 条）
ag commit comment list owner/repo abcdef1
ag commit comment list owner/repo abcdef1 --limit 100
ag commit comment list owner/repo abcdef1 --json

# 查看仓库级评论 ID 的详情
ag commit comment view owner/repo 12345
ag commit comment view owner/repo 12345 --json

# 在指定 commit 上创建评论，正文支持多行文本
ag commit comment create owner/repo abcdef1 --body "LGTM"
ag commit comment create owner/repo abcdef1 --body-file notes.md
cat notes.md | ag commit comment create owner/repo abcdef1 --body-file -

# 编辑自己的评论（只提交正文字段）
ag commit comment edit owner/repo 12345 --body "updated text"

# 删除自己的评论，需确认，--yes 跳过
ag commit comment delete owner/repo 12345
ag commit comment delete owner/repo 12345 --yes
```

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

# 列出我创建的 / 分配给我的 / 需要我批准的 / 需要我评审的 PR（跨所有仓库，不能与 owner/repo 同用）
ag pr list --author "@me"
ag pr list --assignee "@me"
ag pr list --review-requested "@me"
ag pr list --review-needed "@me"

# 查看 PR
ag pr view 123
ag pr view owner/repo 123
ag pr view owner/repo 123 --json

# 在浏览器中打开 PR
ag pr view owner/repo 123 --web

# 修改 PR 标题或正文
ag pr edit owner/repo 123 --title "Updated title"
ag pr edit owner/repo 123 --body "Updated description"
ag pr edit owner/repo 123 --body-file description.md
cat description.md | ag pr edit owner/repo 123 --body-file -

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
ag pr create owner/repo --title "Fix bug" --body "Description" --base main --head feature-branch --draft
ag pr create owner/repo --title "Fix bug" --body-file description.md --base main --head feature-branch
cat description.md | ag pr create owner/repo --title "Fix bug" --body-file - --base main --head feature-branch
ag pr create owner/repo --title "Fix bug" --head feature-branch \
  --assignee alice --reviewer bob --tester carol --label Bug --milestone v1.0
ag pr create owner/repo --title "Fix bug" --head feature-branch --prune-branch

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

# 查看 PR 的提交、文件变更、反应、操作日志和修改历史
ag pr commits owner/repo 42
ag pr commits owner/repo 42 --limit 50 --json
ag pr files owner/repo 42
ag pr files owner/repo 42 --json
ag pr reactions owner/repo 42
ag pr reactions owner/repo 42 --limit 50 --json
ag pr activity owner/repo 42
ag pr activity owner/repo 42 --limit 50 --json
ag pr history owner/repo 42
ag pr history owner/repo 42 --limit 50 --json
```

`pr commits`、`pr files`、`pr reactions`、`pr activity` 和 `pr history` 都是只读命令，只发送 GET 请求。`pr commits`、`pr reactions`、`pr activity` 和 `pr history` 支持 `--limit`（默认 30，必须为正整数）控制返回数量；其中 `pr commits`、`pr reactions` 和 `pr activity` 会对支持分页的服务端接口逐页拉取，而 `pr history` 对应的 `modify_history` 接口不支持分页，会先取回全部记录再在本地截断到 `--limit`。文本模式每行输出一个条目摘要，无结果时输出一行提示；JSON 模式输出稳定的 lowerCamelCase 数组，无结果时输出 `[]`。

`pr reactions --json` 的 `id` 统一输出为字符串（也兼容服务端返回的数字 ID），并提供 `emoji` 和 `emojiName` 字段。保留 `content` 和 `createdAt`：若响应未提供 `content`，依次使用 `emoji_name`、`emoji`；当前接口不提供时间戳，`createdAt` 为 `""`，文本输出省略时间。文本显示表态名称、表情和操作人。

`pr list` 的 `--author`、`--assignee`、`--review-requested` 和 `--review-needed` 目前只支持 `@me`：通过授权用户接口（`/api/v5/user/pulls`）按登录账号跨所有仓库过滤，分别对应服务端 `scope` 的 `created_by_me`（我创建的）、`assigned_to_me`（分配给我的）、`need_my_approve`（需要我批准的）和 `need_my_review`（需要我评审的）。这四个参数彼此互斥，也不能与显式 `owner/repo` 参数同用；不带这些参数时按仓库列出，行为不变。`--state`、`--limit` 和 `--json` 在两种模式下均可使用。跨仓库模式的文本输出会在每行开头附带 `owner/repo` 前缀（如 `owner/repo #1 标题 [open]`），因为编号只在各自仓库内唯一；`--json` 输出可通过每条记录的 `url` 字段区分仓库，schema 保持不变。

`pr view --json` 在现有字段基础上新增 `assignees`、`approvalReviewers`、`testers`（均为字符串数组，空时为 `[]`）和 `milestone`（对象或 `null`）字段。`pr list --json` 的 schema 保持不变。两个命令的 `merged` 字段会综合 AtomGit 响应中的 `merged`、`state` 和 `merged_at` 判断，避免 API 省略 `merged` 时把已合并 PR 错报为 `false`。

跨仓库创建 PR 时 `--head` 的写法请参阅[跨仓库 PR 示例](cross_repo_pr_demo.md)。

负责人（assignee）负责后续工作，批准审查人（approval reviewer）负责批准变更，测试人（tester）负责验证变更；三个 AtomGit 角色相互独立。用户账号、标签和里程碑会在修改 PR 前解析，标签和里程碑必须已存在。`pr edit` 只修改显式传入的字段；`--body-file -` 从标准输入读取正文，`--body` 与 `--body-file` 互斥，显式传入空正文会清空现有正文。添加和移除参数可重复使用，也可用逗号一次传入多个值。

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

# 列出我创建的 / 分配给我的 / 与我相关的 Issue（跨所有仓库，不能与 owner/repo 同用）
ag issue list --author "@me"
ag issue list --assignee "@me"
ag issue list --involved "@me"

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

`issue list` 的 `--author`、`--assignee` 和 `--involved` 目前只支持 `@me`：通过授权用户接口（`/api/v5/user/issues`）按登录账号跨所有仓库过滤，分别对应服务端 `filter` 的 `created`（我创建的）、`assigned`（分配给我的）和 `all`（创建或分配给我的）。这三个参数彼此互斥，也不能与显式 `owner/repo` 参数同用；不带这些参数时按仓库列出，行为不变。`--state`、`--limit` 和 `--json` 在两种模式下均可使用。跨仓库模式的文本输出会在每行开头附带 `owner/repo` 前缀（如 `owner/repo #1 标题 [open]`），因为编号只在各自仓库内唯一；`--json` 输出可通过每条记录的 `url` 字段区分仓库，schema 保持不变。

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

### Issue 活动、修改历史与表态

```bash
ag issue activity owner/repo 42 --limit 50
ag issue history owner/repo 42 --json
ag issue reactions 42 --limit 100 --json
```

三个命令均只读，支持显式仓库或 Git remote 推断；Issue 编号和 `--limit` 必须为正整数，默认最多输出 30 条。空结果的 JSON 为 `[]`，文本显示空状态提示。输出顺序保留服务端顺序，不额外按时间排序。 如果返回列表含空元素或无效记录标识，命令报错并停止，不输出部分结果或虚假的零值记录。

- `activity` 显示操作者、操作类型、Issue 编号、创建时间及操作描述。
- `history` 显示修改者（缺失时回退到创建者）、创建/修改/删除状态、Issue 编号、时间及内容。JSON 分别保留 `author` 与 `updatedBy`，以及 `created`、`deleted` 标记。
- `reactions` 显示用户、表态名称及 emoji；接口未提供时间字段，不生成虚假时间。

[操作日志接口](https://docs.atomgit.com/docs/apis/get-api-v-5-repos-owner-issues-number-operate-logs/)和[修改历史接口](https://docs.atomgit.com/docs/apis/get-api-v-5-repos-owner-repo-issues-number-modify-history/)未声明分页参数，因此单次读取列表后应用 `--limit`，该参数不限制服务端响应大小。[表态接口](https://docs.atomgit.com/docs/apis/get-api-v-5-repos-owner-repo-issues-number-user-reactions/)支持 `page/per_page`，按页获取直到达到限制或列表结束。

JSON 字段固定：activity 为 `id/author/action/content/createdAt/updatedAt/issueId/title/body/head/base`；history 为 `id/author/updatedBy/content/createdAt/updatedAt/created/deleted`；reactions 为 `id/author/emoji/emojiName`。activity 的 `id` 为整数，history 和 reactions 的 `id` 为不透明字符串；`author`、`updatedBy` 为登录名。activity 的 `head/base` 保留关联 PR 的 `ref`、`sha`、`repo`（`path/name`）及 `assigner`（`login/name`）；未提供的分支、仓库或指派人为 `null`，未提供的 `body` 为 `""`。JSON 保留正文中的换行，文本表格将空白折叠为单行，并继续经过默认终端控制字符清理。

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

## Tag

```bash
# 在当前 Git 仓库中列出标签（默认显示 30 条）
ag tag list

# 显式指定仓库或限制返回数量
ag tag list owner/repo
ag tag list owner/repo --limit 100
ag tag list owner/repo --json

# 创建或删除标签
ag tag create v1.0.0 --ref main
# 删除标签（默认要求确认；自动化场景请使用 --yes）
ag tag delete v1.0.0
ag tag delete v1.0.0 --yes

# 查看保护 tag 规则（输出会区分 exact 与 wildcard）
ag tag protection list owner/repo --limit 30
ag tag protection view owner/repo v1.0.0
ag tag protection view owner/repo "v*"
ag tag protection list --json
ag tag protection view v1.0.0 --json

# 创建保护规则；省略 --create-access 时使用服务端默认 maintainer
ag tag protection set owner/repo v1.0.0 --create-access maintainer
ag tag protection set owner/repo "v*" --create-access developer
ag tag protection set owner/repo v1.0.0

# 将已有规则改为不允许任何人推送；更新默认要求确认
ag tag protection set owner/repo v1.0.0 --create-access none --yes

# 删除规则（默认显示当前仓库和规则并要求确认）
ag tag protection delete owner/repo "v*"
ag tag protection delete owner/repo "v*" --yes
```

`tag create` 必须显式传入非空的 `--ref`，值可以是 branch、tag 或 commit SHA。

`ag tag delete` 默认会显示目标仓库和标签名并要求确认；可使用 `--yes`（或 `-y`）跳过确认提示。

保护 tag 规则的 `--create-access` 接受 `none`、`developer` 或 `maintainer`，分别对应 AtomGit `create_access_level` 的 `0`（不允许任何人推送）、`30`（Developer / Maintainer / Admin）和 `40`（Maintainer / Admin）。创建时省略该标志则不发送 `create_access_level`，由服务端使用默认值 `40`（Maintainer / Admin）。CLI 只管理官方 API 暴露的创建/推送权限，不修改其他保护设置。更新接口要求同时提交规则名和权限，因此更新已有规则时若省略 `--create-access`，CLI 会先读取现有规则并保留当前权限；若服务端返回无法识别的权限值，命令会停止并要求显式提供 `--create-access`。更新或删除已有规则时默认显示仓库和当前规则并要求确认，可用 `--yes` 跳过。tag 命令省略 `owner/repo` 时使用当前 Git 仓库推断结果，显式参数始终优先。

## 资源命令的 JSON 输出

`repo list/view`、`issue list/view`、`pr list/view`、`tag list`、`tag protection list/view`、`branch list`、`label list`、`release list/view`、`run list` 和 `commit list/view/compare` 支持布尔参数 `--json`。list 命令输出完整 JSON 数组，view 与 compare 命令输出完整 JSON 对象；没有结果时 list 输出 `[]`。默认文本输出保持不变。

JSON 字段使用 lowerCamelCase，并由 CLI 显式定义，不会因为 AtomGit API 增加字段而自动改变。Issue 和 PR 的 `number` 始终是字符串，标签输出为名称数组，PR 的 `head` 和 `base` 输出分支名称。可选的服务端字段缺失时仍输出对应的零值，以保持固定结构。

`view --json` 与 `view --web` 互斥；JSON 模式只向标准输出写入一个 JSON 值，不混入浏览器提示或其他文本。原始字节流命令（例如 `ag pr diff`）不提供 JSON 包装。

## Label

```bash
# 列出仓库标签（默认显示 30 条）
ag label list owner/repo
ag label list owner/repo --limit 50
ag label list owner/repo --json

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

`ag run` 提供工作流运行检查能力，并支持删除单个 artifact；不会触发、重跑、取消或删除工作流运行。

```bash
# 列出运行记录（默认最多 30 条）
ag run list owner/repo
ag run list owner/repo --json
ag run list

# 按分支、状态和触发事件过滤
ag run list owner/repo --branch main --status failed --event push

# 也可按触发人、PR、workflow 和毫秒时间戳过滤
ag run list owner/repo --actor alice --pr 42 --workflow-name CI --limit 50
ag run list owner/repo --start-time 1700000000000 --end-time 1700086400000

# 查看 run、jobs、steps、URL 和 artifacts；步骤行包含 step ID
ag run view owner/repo <run-id>
ag run view <run-id>

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

# 按 step ID 分页拉取步骤日志；默认输出到 stdout
ag run step-log owner/repo <run-id> <job-id> <step-id>
ag run step-log owner/repo <run-id> <job-id> <step-id> --output step.log
ag run step-log owner/repo <run-id> <job-id> <step-id> --output step.log --overwrite

# 查看 artifact 元数据，不下载归档
ag run artifact view owner/repo <artifact-id>
ag run artifact view <artifact-id>
ag run artifact view owner/repo <artifact-id> --json

# 删除 artifact；默认先显示元数据并要求确认
ag run artifact delete owner/repo <artifact-id>
ag run artifact delete <artifact-id> --yes
```

`--log` 会先把 AtomGit 返回的日志 ZIP 流式写入临时文件，再逐项输出其中的日志文本；若服务端返回纯文本也会直接兼容。`--log-file` 保留服务端原始 ZIP。`ag run view --artifact` 下载的是 artifact 归档，而 `ag run artifact view` 只读取元数据。`ag run artifact delete` 会先读取 artifact 元数据并显示仓库、ID、名称、workflow run ID 和过期时间，只有输入 `y` 或 `yes` 才会继续；`--yes` 可跳过确认，但不会跳过元数据读取。artifact 删除后无法恢复。日志、step-log `--output` 和 artifact 文件下载都会先写入目标目录中的临时文件，完整写入后再移动到目标路径。若目标已存在，必须显式使用 `--overwrite`。

## Actions 工作流管理 (workflow)

`ag workflow` 用于查看 AtomGit Actions 工作流列表、校验本地 workflow YAML，以及手动触发工作流运行（`workflow_dispatch`）。

```bash
# 列出仓库下的工作流及其 ID
ag workflow list owner/repo
ag workflow list owner/repo --json

# 校验本地 workflow 文件，不修改远程工作流
ag workflow validate --file .gitcode/workflows/ci.yml
ag workflow validate owner/repo --file workflow.yml --json

# 手动触发工作流运行（按工作流 ID 或名称/路径）
ag workflow run owner/repo 12345 --ref main
ag workflow run owner/repo ci.yml --ref feature-branch -f env=production -F debug=true
```

`ag workflow validate` 把本地文件作为 `base64_content` 发给 Actions API v8。HTTP 200 且 `valid=false` 时命令仍以非零状态退出，方便 CI 拦截无效 YAML；`--json` 会把完整响应写到 stdout。`ag workflow run` 可以按工作流 ID、名称或相对路径（basename）选择目标；名称或 basename 同时匹配多个工作流时会报错并要求改用精确的 workflow ID 或完整路径。`--ref` 缺省时使用仓库的默认分支（而非假定为 `main`），无法确定默认分支时需显式传入 `--ref`。`-f/--raw-field` 与 `-F/--field` 均为 `key=value` 格式，参数值会作为 `workflow_dispatch` 的 inputs 传递。

## Actions Runner 查询 (runner)

`ag runner` 只读查询仓库专属和共享给仓库的主机 Runner，不会修改 Runner 配置。默认跟随 API 分页返回全部结果；使用 `--limit` 可限制返回数量，`--json` 输出稳定的机器可读结构。

```bash
# 查询仓库专属 Runner
ag runner list owner/repo
ag runner list owner/repo --limit 20 --json

# 查询分享给仓库的 Runner
ag runner shared owner/repo
ag runner shared owner/repo --json

# 在当前 Git 仓库中推断 owner/repo
ag runner list
```

文本输出会区分 `repository` 与 `shared` 来源，并显示 ID、名称、状态、busy/online（服务端未返回时显示 `-`）、平台、操作系统和标签。命令检测到分页重复或 API 总数与实际返回不一致时会失败，避免把不完整列表当成完整结果。


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

# 仅本地预览，不发送请求，不需要登录
ag api /repos/owner/repo/issues --method POST --field title=example --dry-run
ag api /repos/owner/repo/issues/42 --method PATCH --input update.json --dry-run
ag api /repos/owner/repo/issues --paginate --dry-run
```

端点必须是 API v5 下的相对路径；绝对 URL、`//host/path`、片段和越过 API 基址的路径会在读取凭据前被拒绝。认证信息仅通过 `Authorization` 请求头发送。同源重定向可保留认证；scheme、主机或有效端口变化后，当前及后续跳转均不会再携带认证信息。

`--field key=value` 在第一个 `=` 处分隔。GET 会保留已有查询值并按命令行顺序追加字段；其他支持的方法生成 JSON 对象，重复键以最后一个值为准。`--input` 与 `--field` 互斥，原始输入不会推断 `Content-Type`。`--accept` 默认是 `application/json`。

`--paginate` 仅支持无原始输入的 GET。默认从 `page=1&per_page=100` 开始；已有的正整数值会被保留。服务端提供一致的 `total_page` 响应头时据此停止；否则仅数组响应可通过空页或短页停止。后续页面失败时，已完成的 NDJSON 行会保留，失败页面不会产生部分输出。

成功响应（包括空响应和二进制响应）会直接写到标准输出，不增加标签或换行。终端控制字符默认仍会转换为可见转义；机器处理确需原始字节时使用 `ag --raw-output api ...`，不要将未经检查的原始输出直接转发到终端。

### API 请求预览

`--dry-run` 复用真实模式的请求准备与校验，成功时只输出一个 JSON 对象和结尾换行；失败返回非零且不输出成功对象。它不读取令牌、不初始化或修改凭据，也不发起网络请求、刷新令牌或检查更新。未登录、凭据文件损坏时仍可使用；显式指定的 `--input` 文件或 stdin 会被读取，但不会写回。

预览中的 `dryRun: true`、`executed: false` 表示**仅完成本地准备，尚未执行请求**。预览不验证远端权限、资源存在性或服务端业务规则，也不授权后续写操作；实际执行需另行移除 `--dry-run`。`--dry-run=false` 保留原有真实请求行为。

预览 JSON 的字段固定保留：

| 字段 | 含义 |
| --- | --- |
| `schemaVersion` | 当前预览结构版本，固定为 `1` |
| `dryRun` / `executed` | 固定为 `true` / `false` |
| `method` | 最终 HTTP 方法，已转为大写 |
| `apiVersion` / `host` / `basePath` | 从实际请求基址取得；当前为 `v5` / `api.atomgit.com` / `/api/v5` |
| `path` | 规范化后的相对路径，不含查询字符串；未知片段以 `[redacted]` 替代 |
| `query` | 查询参数数组，按原始名称排序；每项含 `name`、`type: "string"`、`count`（同名值数量），不含值；无参数时为 `[]` |
| `accept` | 默认 `application/json` 原样展示；自定义值为 `[redacted]` |
| `body` | `source` 为 `none` / `fields` / `file` / `stdin`；`contentType` 是实际 HTTP 类型（未设置时为 `""`），`byteLength` 为请求体字节数 |
| `body.type` / `body.count` | 无请求体为 `absent`，非 JSON 为 `opaque`，否则为 JSON 类型（object/array/string/number/boolean/null）；count 为顶层对象字段数或数组元素数，其他类型为 `0` |
| `body.fields` / `body.truncated` | 顶层对象最多 50 项字段的名称与 JSON 类型，按原始名称排序，超出时 truncated 为 `true`；其他类型 fields 为 `[]`，不展开嵌套内容 |
| `pagination` | 固定含 `enabled`、`firstPage`、`perPage`、`strategy`；启用时两个数值字段均为字符串 `"[redacted]"`，包括默认值；未启用时分别为 `false`、`null`、`null`、`"none"` |

脱敏采用固定名称白名单，仅保留常见路由词和字段名（例如 `repos`、`issues`、`title`、`token`），其余名称与路径片段均为 `[redacted]`。因此 `/repos/owner/repo/issues/42` 显示为 `/repos/[redacted]/[redacted]/issues/[redacted]`。所有字段值、查询值、正文标量、嵌套内容、输入文件路径及认证信息都不展示；不根据某个字段“看起来安全”而输出其值。请求体字节数与结构数量仍可见。请求准备错误也会省略可能带有输入内容的底层详情。全局 `--raw-output` 不会关闭这些脱敏规则。

选项解析会在第一个错误处停止，可能尚未读到 `--dry-run`；因此 `ag api` 在预览和真实模式下均省略非法选项的原始名称和值，并提示查看 `ag api --help`。

`--paginate --dry-run` 只预览首个请求，将 `firstPage` / `perPage` 统一脱敏为 `"[redacted]"`，不展示通过 URL 查询参数或 `--field` 提供的原始数值。`strategy` 为 `total_page-or-short-array`，表示实际执行时依据服务端 `total_page` 或数组短页停止。实际请求仍使用准备阶段校验后的分页数值；预览没有远端总页数或后续请求结果，所有原有分页和输入互斥限制继续生效。

## Release

```bash
# 列出仓库 Release（默认最多 30 条）
ag release list
ag release list owner/repo --limit 50
ag release list owner/repo --json

# 按 tag 查看 Release 详情（附件列表、作者、时间、状态等）
ag release view v1.0.0
ag release view owner/repo v1.0.0
ag release view owner/repo v1.0.0 --json

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

`ag release view --json` 将与文本模式相同的一次查询结果输出为单个 JSON 对象，以换行结尾，不混入文本标签。查询失败时返回非零退出码，不输出成功对象。未指定 `--json` 时保留原有文本输出。

详情 JSON 的前八个字段与 `ag release list --json` 的每个列表项保持一致：

| 字段 | 类型 | 含义 |
| --- | --- | --- |
| `tag` | string | Release 的 tag |
| `name` | string | Release 名称 |
| `status` | string | `prerelease`、`latest` 或 `release`；预发布优先，其次为最新发布，其余状态归为 `release` |
| `draft` | boolean | API 返回的草稿标志 |
| `prerelease` | boolean | API 返回的预发布标志；服务端状态为预发布时，即使此标志为 `false`，`status` 仍为 `prerelease` |
| `targetCommitish` | string | Release 的目标提交或分支 |
| `createdAt` | string | API 返回的创建时间字符串 |
| `author` | string | 作者的登录名，不回退到显示名称 |
| `body` | string | 完整 Release 正文，换行、引号和控制字符通过 JSON 转义保留 |
| `assets` | array | API 返回的附件和源码归档，每项包含下表中的字段 |

| 附件字段 | 类型 | 含义 |
| --- | --- | --- |
| `id` | number | API 返回的附件 ID |
| `name` | string | 附件名称 |
| `type` | string | 附件类型，例如 `attach` 或 `source` |
| `browserDownloadUrl` | string | 附件下载 URL |

以上字段均固定保留，不因空值省略。缺失或为 `null` 的字符串输出 `""`，布尔值输出 `false`，附件 ID 输出 `0`；缺少作者登录名时 `author` 为 `""`；无附件（包括 API 未返回或返回 `null`）时 `assets` 为 `[]`，不会输出 `null`。`status` 始终按上述规则计算。JSON 仅包含这些 CLI 字段，不透传完整 API 响应。

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

# 查看单个讨论的标题、状态与 Markdown 正文
ag discussion view hust-open-atom-club/atomgit-cli 1

# 连同评论串一起输出；有回复的评论会嵌套展示其回复
ag discussion view hust-open-atom-club/atomgit-cli 1 --comments
ag discussion view hust-open-atom-club/atomgit-cli 1 --comments --json
```

公开仓库的 Discussion 可匿名读取；登录后会携带访问令牌，以便访问账号有权限查看的仓库。

`discussion view` 的 `<number>` 必须是正整数，在认证之前就会校验。加 `--comments` 时会额外请求评论串，评论按服务端顺序排列，仅 `reply_total` 大于 0 的评论才继续拉取嵌套回复。已删除、已隐藏或正文为空的条目分别显示为 `[deleted]`、`[hidden]`、`[no content]`。JSON 输出中 `comments` 字段仅在指定 `--comments` 时出现：请求了但没有评论时输出 `[]`。

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

文本输出包含版本号以及可用的 commit 和构建时间；JSON 输出固定包含 `version`、`commit` 和 `buildDate` 三个字段。

## 更新 CLI

```bash
# 检查最新稳定版，不安装
ag update --check

# 兼容旧脚本；已弃用，行为等同于 ag update --check
ag check-update

# 检查并更新；当前自动支持全局 npm 和 Homebrew Core 安装
ag update
```

`ag update` 公开查询 `hust-open-atom-club/atomgit-cli` 的稳定 Release，并按 SemVer 比较当前版本。命令不需要登录或仓库上下文。当前版本和最新 Release 都以带 `v` 前缀的 `vX.Y.Z` 格式输出。`--check` 只输出当前版本、最新稳定 Release 和比较状态，不识别安装来源、不调用包管理器，也不修改当前安装。为兼容已有脚本，弃用的 `ag check-update` 暂时保留，并转发到同一只读检查逻辑。

不带 `--check` 且发现新版本时，命令根据当前实际运行的 `ag` 二进制路径识别安装来源，然后提供两个选择：`Update via npm` 或 `Update via Homebrew Core` 会调用对应包管理器；`Skip` 只跳过本次运行。首次输入直接回车或尚未输入内容时到达 EOF 会默认更新；无效答案后到达 EOF 则返回错误，不会调用包管理器。

用户选择更新后，全局 npm 安装会先确认精确目标版本已经发布到官方 npm registry，再执行安装并通过 npm 生成的真实命令入口核对版本；Homebrew Core 安装会依次执行 `brew update` 和 `brew upgrade atomgit-cli`，随后运行新二进制核对版本。这样不会把 npm 启动器损坏或“Formula 尚未更新”等情况误报为升级成功。

Windows 上全局 npm 更新可能无法覆盖正在运行的 `ag.exe`。如果 npm 破坏了命令入口，或者 npm 报告成功但入口仍是旧版本，`ag update` 会下载 AtomGit Release 的 Windows 归档、使用 `checksums.txt` 校验 SHA-256，并以可回滚方式将 npm 命令入口修复为独立的 `ag.exe`。如果 npm 报错但旧入口仍可用，命令会保留原入口并报告 npm 错误。修复发生时会明确提示该入口不再由 npm 管理；重新执行 npm 全局安装可恢复 npm 管理。

项目 Homebrew Tap 暂不属于 `ag update` 支持范围，也不会被当成 Homebrew Core 自动升级。Tap、WinGet、Scoop、Nix、AUR、Go、Release 安装器、源码或无法识别的安装当前都不会被修改。`dev`、dirty、提交哈希或其他无法比较的本地版本会在安装来源识别和包管理器调用之前返回清晰错误。

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
