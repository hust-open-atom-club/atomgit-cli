# 配置指南

本文档介绍 AtomGit CLI 的认证、凭据文件、输出安全和当前仓库推断规则。

## 目录

- [认证](#认证)
- [输出安全](#输出安全)
- [API 限流重试](#api-限流重试)
- [当前仓库推断](#当前仓库推断)

## 认证

首次使用本工具前，需要选择以下任一方式配置访问令牌：

- 使用 OAuth 登录（推荐）：运行 `ag-cli auth login`，在浏览器中完成 AtomGit 授权。登录成功后，`ag-cli` 会自动将认证信息写入令牌文件。

  浏览器回调的 `state` 不匹配时，CLI 会拒绝该请求并继续等待本次登录的有效回调。

  > [!IMPORTANT]
  > 使用 Actions/流水线或项目讨论相关功能时，OAuth 应用需要提供 “Action” 和“项目讨论”的读写权限。运行 `ag-cli auth login --force` 会重新打开授权确认；如果授权页面未提供这两项权限，请使用已具备相应权限的访问令牌通过 `ag-cli auth login --with-token --force` 登录。

- 使用已有令牌登录（无浏览器环境）：将预先签发的访问令牌（PAT 或 OAuth access_token）通过标准输入传给 `ag-cli auth login --with-token`，CLI 会先调用 AtomGit 用户接口验证令牌并获取用户名，验证通过后才写入令牌文件：

  ```bash
  echo "$TOKEN" | ag-cli auth login --with-token
  ag-cli auth login --with-token < token.txt
  ```

  管道或重定向输入全程无交互提示；在终端直接运行时会出现一次隐藏输入的粘贴提示。该方式适合沙箱、容器、CI 等无法打开浏览器的环境。通过此方式登录的账号没有 `refresh_token`，`ag-cli auth refresh` 对其不可用，令牌过期后需重新执行本命令。

- 手动创建访问令牌：参考 [AtomGit 访问令牌（PAT）文档](https://docs.gitcode.com/docs/help/home/user_center/security_management/user_pat/)，依次进入「个人设置」->「访问令牌」->「新建访问令牌」，按需设置权限范围和到期时间，再将生成的 PAT 写入令牌文件。PAT 创建后只显示一次，请立即妥善保存，不要将其提交到代码仓库或分享给他人。

需要认证的命令在缺少凭据时会提示运行 `ag-cli auth login`，并返回非零退出码。该提示不展示凭据查找路径或原始底层错误。

常规命令加载凭据时，若令牌文件的 JSON 语法或字段类型错误，CLI 会提示检查文件格式；语法错误会包含字节位置。该诊断不展示原始 JSON 内容或解码器错误，修正文件后可重试命令。

需要访问不同部署或以不同用户身份执行单次命令时，也可以临时指定访问令牌和服务根地址：

```bash
ag-cli repo list --token "$TOKEN" --base-url "$BASE_URL"
```

`--token` 和 `--base-url` 只在当前命令进程的内存中生效，不会写入 `token.json`、`config.json` 或其他配置文件。`--base-url` 默认为 `https://api.atomgit.com`，不传该参数时使用同一地址；远程服务必须使用不含路径的 HTTPS 根地址，只有 `localhost`、`127.0.0.1` 和 `::1` 等回环地址允许使用 HTTP。普通命令自动使用 `/api/v5`，Actions 命令自动使用 `/api/v8`。需要当前用户名的命令会使用临时 token 调用所选服务的 `/api/v5/user`，不会混用本地持久化账户的用户名；`auth status --verify` 和 `doctor --live` 仍会只读检查本地凭据状态，但其在线请求同样优先使用显式传入的 token。不同进程可以使用不同参数。由于命令行参数可能被 shell 历史或系统进程列表记录，不需要临时覆盖时仍建议使用凭据文件。

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
| `refresh_token` | 否 | OAuth 刷新令牌，仅由 `ag-cli auth login` 获取，并供 `ag-cli auth refresh` 换取新的访问令牌。PAT 没有该字段。 |
| `expires_in` | 否 | OAuth 访问令牌从签发时刻起的有效秒数，由服务端返回。PAT 的有效期在创建 PAT 时设置，手动配置可省略。 |
| `created_at` | 否 | CLI 保存或刷新 OAuth 凭据时记录的 Unix 时间戳（秒），用于表示签发/保存时间。手动配置 PAT 时可省略。 |
| `token_type` | 否 | OAuth 服务返回的令牌类型；手动配置 PAT 时可省略。API 请求当前使用 `Bearer`。 |

`ag-cli auth login` 会将凭据保存为多账号格式。由于 CLI 仅支持 `atomgit.com`，账号以规范化后的用户名作为唯一标识：

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

本地已有凭据但指定账号未保存时，CLI 会提示用 `ag-cli auth list` 查看可用账号。账号选择失败与凭据缺失、凭据内容损坏分别报告；切换和删除规则见[认证命令说明](usage.md#认证)。

除只读的 `ag-cli auth status` 外，常规 `ag-cli auth` 子命令会检查令牌文件格式；`auth login` 在新凭据验证通过后才执行迁移。旧单账号文件会通过安全的原子写入自动升级为当前多账号格式，已有 `refresh_token` 等 OAuth 字段会保留；当前格式不会重复写入，未知版本会拒绝迁移。手动使用 PAT 时不要自行编造 `refresh_token`、`expires_in` 或 `created_at`。令牌文件使用 `0600` 权限，写入通过同目录临时文件原子替换；不要将它提交到版本控制。

别名配置文件 `config.json` 不存在时，按空别名表处理。读取或解码失败时，`ag-cli alias list` 会报告错误。命令尝试展开别名时遇到这两类故障，会输出警告并继续处理原始参数。

凭据保存失败时，CLI 会指出失败阶段和涉及的路径。若临时凭据文件也无法删除，诊断会提示该文件可能仍然存在。

凭据删除失败时，CLI 会指出失败的路径，并列出此前已经删除的凭据文件。此时注销未完成，剩余文件仍可能提供凭据；检查失败路径后再重试。无法确定凭据文件路径时会报告错误，不会按“已注销”处理。

别名配置保存失败时，CLI 会清理临时文件。清理也失败时，返回错误会同时保留保存失败和清理失败的原因。别名配置写入成功后若释放锁失败，会明确报告配置已保存及释放锁失败。

`ag-cli auth status [--json]` 只读检查当前活动凭据，不迁移旧格式、不修正文件权限，也不刷新或切换账号；配置损坏仍会输出状态结果并返回非零。加上 `--verify` 才会请求 v5 `/user` 验证身份（30 秒超时），不会据此推断其他资源的访问权限。状态输出完全省略令牌及其片段；字段和退出码见[认证命令说明](usage.md#认证)。

### 组织仓库归档的可选密码

`repo archive/unarchive` 通过既有访问令牌认证，默认不发送账户密码。部署需要额外密码时，`--with-password` 从交互终端隐藏读取；自动化使用 `--yes --password-stdin`，不要把密码放在命令行参数中。密码只保存在本次进程内存并放入该次 PUT 的 JSON 请求体，不写入配置、日志或仓库 URL。服务端业务消息和完整失败响应不会显示；归档写入不自动重试或跟随重定向，失败时应先核对远端状态。详见[归档命令说明](usage.md#仓库-repo)。

### 认证 HTTPS 克隆的凭据使用

`ag-cli repo clone` 在克隆 URL 使用 `https://` 且主机为一级 AtomGit 域名（`atomgit.com` 或其 `gitcode.com` 镜像）时，会自动把当前激活账号存储的访问令牌交给 git 完成认证，无需手动输入用户名或令牌：

- 认证方式为 HTTP Basic：登录用户名 + 访问令牌。令牌仅用于**本次 `git clone` 进程**，通过临时注入的环境变量式 Git 配置（`http.https://<主机>/.extraheader`）传递，因此**不会写入**克隆出的仓库的 remote URL（`.git/config` 中的地址保持干净），也不会出现在命令行参数中；
- 只有**已登录**（令牌文件中存在有效账号）时才会注入凭据；未登录的克隆保持匿名，公开仓库无需登录即可克隆；
- SSH 克隆（如 `git@atomgit.com:owner/repo.git`）、明文 `http://` 地址以及非 AtomGit/GitCode 主机（如 `github.com`）的克隆**不受影响**，保持原有匿名或用户自行配置的 Git 凭据行为；
- 访问令牌无效或无权访问目标仓库时，克隆会直接失败并提示认证错误，而**不会弹出交互式 Git 用户名/密码提示**（克隆进程设置了 `GIT_TERMINAL_PROMPT=0`）。令牌过期或失效时，可运行 `ag-cli auth refresh`（OAuth 登录）或重新执行 `ag-cli auth login --with-token`（PAT 登录）后重试。

### 为普通 Git HTTPS 操作配置 credential helper

`ag-cli repo clone` 会按上述方式自动为单次克隆提供凭据。如需让直接执行的 `git clone`、`git pull` 和 `git push` 等 HTTPS 操作复用当前活动账号，可在登录后运行：

```bash
ag-cli auth setup-git
```

该命令仅在全局 Git 配置中为 `https://atomgit.com` 注册 `ag-cli` credential helper，不会把访问令牌写入 Git 配置或 remote URL。Git 每次需要凭据时会从 `ag-cli` 的凭据文件读取当前活动账号；使用 `ag-cli auth switch` 切换账号后无需重新运行。SSH remote 不受此设置影响。

### 必需认证与可选认证

大多数命令需要已登录账号；缺少凭据时会提示运行 `ag-cli auth login`。凭据路径无法确定或文件访问失败时，会单独报告无法读取凭据，并在权限不足时提示检查文件及父目录的所有权和读取权限。

Unix 上读取凭据时会尝试收紧过宽的文件权限。修复失败时会报告无法保护凭据文件。若系统拒绝修改，请检查文件所有权及当前用户的权限。

符号链接、打开期间文件变化、Unix owner-read 校验失败等安全拒绝会报告凭据配置无效，并给出对应的检查建议。

常规命令加载凭据和只读检查（`ag-cli auth status`、`ag-cli doctor`）使用相同的文件限制。两者都只接受普通文件，并拒绝超过 1 MiB 的凭据文件。普通文件限制排除 FIFO、设备文件和目录。凭据文件超过 1 MiB 时，命令会报告凭据配置无效，并提示凭据文件大小限制为 1 MiB。

使用本地凭据时，存储版本、账号记录或活动账号引用无效会导致命令失败，并提示检查这些内容。主凭据文件内容无效时，查找在该文件处停止。

帮助和版本入口不需要读取凭据：`ag-cli --help`、`ag-cli help <command>`、子命令的 `-h`/`--help`、`ag-cli --version`、`ag-cli version` 和 `ag-cli version --json` 在凭据缺失、损坏或无法读取时仍可使用。

部分公开只读命令（例如 `ag-cli discussion list` 和 `ag-cli discussion view`）在尚未登录时会以匿名方式请求。凭据文件损坏、权限错误或其他存储故障**不会**降级为匿名访问。

## 输出安全

`ag-cli` 默认会将终端控制字符转换为可见转义文本，包括输出经管道转发时，以防止仓库、Issue、PR 或 Git 服务端返回的内容注入终端控制序列。确实需要为机器处理保留原始字节时，可显式使用全局参数 `--raw-output`，例如 `ag-cli --raw-output pr diff owner/repo 123`；请勿将未经检查的原始输出直接转发到终端。

`release view --json` 等通过共享 JSON 输出器生成的结构化输出会使用合法 JSON 转义保留控制字符（包括 DEL 的 `\u007f`），标准 JSON 解析器可还原字段内容，无需为了读取这些字段开启 `--raw-output`。

AtomGit API v5、Actions API v8、OAuth 与 Release 等领域客户端的非成功响应统一经过同一安全边界：错误正文最多保留 4 KiB，超出部分会显示为 `...`；明确省略敏感响应正文的端点仅输出清洗后的状态。错误上下文会转义终端控制字符和 Unicode 方向控制字符，并脱敏 Bearer 值及常见的 token、secret、password、authorization 字段。该安全边界独立于终端 writer，因此 `--raw-output` 不会关闭 API 错误脱敏。

普通 API v5、Actions API v8、外部许可证合规检查和通用 `api` 命令在顶层只展示固定诊断和 HTTP 状态码，不展开上述错误正文、状态文本或原始原因。HTTP 错误体读取失败作为附加诊断，不能将已收到的 HTTP 状态改判为取消或超时。底层错误链仍保留这些事实，`--raw-output` 不会切换为详细诊断模式。

`org create` 仅接受约定的 HTTP 200 成功响应；非预期的其他 2xx、3xx 和 5xx 响应均归为创建结果无法确认，不会自动重试或跟随重定向。创建结果无法确认时，顶层错误会保留 HTTP 状态或失败类型，并明确提示组织可能已经创建，先检查 AtomGit 再决定是否重试。此提示也会在 `--raw-output` 模式显示，且不展开原始响应正文或传输错误。

`org edit` 的更新结果无法确认时，同样保留安全的 HTTP 状态或失败类型，并提示组织资料可能已经改变、先检查 AtomGit 再重试；普通与 `--raw-output` 模式均不展开原始响应正文或传输错误。成功输出的提交字段未经额外查询验证，不代表完整远端资料。

`repo settings view/edit` 同样要求认证并复用仓库上下文解析。GET 和 PUT 均不跟随重定向，避免将设置请求发送到重定向目标；3xx 响应返回错误。编辑先展示目标与显式提交字段，取消不发送 API 请求；CLI 启动时仍可能读取凭据或修复其权限。PUT 不自动重试。结果不确定时，普通和 `--raw-output` 模式均显示安全的状态或失败类型及重试前核对提示，不展开原始响应正文。成功输出区分提交字段与返回设置，缺失值保留为 `null`，不声称完成独立回读。参阅[仓库通用设置](usage.md#仓库通用设置)。

`repo module edit` 对结果不确定的写入保留安全的 HTTP 状态或失败类型，提示模块设置可能已改变，应先在 AtomGit 核对再决定是否重试。默认和 `--raw-output` 模式均不展开服务端消息或传输错误正文；成功仅表示收到 `code=1/msg=success` 确认，不代表独立回读当前配置。

`run cancel/stop`、`run rerun` 和 `run retry` 默认确认后才发送 Actions 请求；`--yes` 跳过确认。写请求不重试、不跟随重定向，只接受 HTTP 200 且 `success=true`。显式任务 ID 的本地校验先于凭据加载；确认取消不发送网络请求，但 CLI 启动仍可能读取本地凭据。

## API 限流重试

普通 AtomGit API v5 和 Actions API v8 元数据 GET/HEAD 请求遇到 HTTP 429 时会进行有界重试，包括使用共享客户端的分页查询。每个逻辑请求默认最多发送 3 次 HTTP 请求，并具有包含请求执行和等待时间在内的 30 秒总预算；调用方设置的更短 context 截止时间会优先生效。服务端 `Retry-After` 支持非负秒数和 HTTP-date。该响应头缺失、非法或已经过期时，CLI 使用带抖动且不超过 2 秒的退避。

如果服务端要求的等待时间不小于剩余预算，CLI 不会缩短等待后提前重试，而是直接返回保留 `Retry-After` 的限流错误。重试前只会有界读取最多 4 KiB 的响应体以帮助连接复用，随后立即关闭响应体。等待提示写入 stderr，不会混入命令的 stdout、JSON 或 NDJSON 输出。

HTTP 429 重试不适用于 POST、PATCH、PUT、DELETE 等写请求，也不适用于日志、附件和 commit diff/patch 等流式下载；显式禁用重试的请求同样不会重试。分页只重试当前尚未成功处理的页面，不会重新请求或重复输出已完成的页面。如果后续页面最终失败，命令返回非零；此前已经输出的分页内容属于不完整结果，不应当作完整响应使用。

`ag-cli api --dry-run` 在启动阶段跳过凭据加载，不读取 token 文件，也不会迁移旧凭据或修正权限。它仅进行本地请求准备；指定 `--input` 时仍会读取该输入。直接调用不依赖别名配置，使用命令别名时仅按现有机制读取别名配置。预览默认隐藏所有请求值及未知名称，`--raw-output` 不会关闭脱敏；字段与边界见[API 请求预览](usage.md#api-请求预览)。

## 当前仓库推断

`issue`、`pr`、`tag`、`label`、`release`、`run`、`branch` 命令以及 `repo view`、`repo edit`、`repo fork`、`repo delete` 可以省略 `owner/repo`。省略时，`ag-cli` 会从当前 Git 仓库的 AtomGit remote 推断目标仓库；显式传入的 `owner/repo` 始终优先。

支持 `git@atomgit.com:owner/repo.git`、`ssh://git@atomgit.com/owner/repo.git` 和 `https://atomgit.com/owner/repo.git`。存在多个 remote 时，依次选择 `remote.pushDefault`、当前分支的 upstream remote、AtomGit `origin` 或唯一的 AtomGit remote。GitHub、GitLab 等其他服务的 remote 不会被识别为 AtomGit 仓库；无法唯一确定时，请显式传入 `owner/repo`。

## 只读健康检查

遇到配置损坏、认证失败或网络异常时，先运行 `ag-cli doctor --json`；需要验证服务端时运行 `ag-cli doctor owner/repo --live --json`。详见[健康检查](usage.md#健康检查-doctor)。运行 `doctor`（包括通过命令别名调用）会绕过普通启动时的凭据加载，以便报告原始配置问题，不会自动调整权限或迁移凭据。默认离线，联网探测仅在指定 `--live` 时执行，不上传诊断报告。

`ag-cli schema [<command> ...]` 查询静态命令说明，同样绕过凭据初始化，且不读取 token、不刷新或迁移配置、也不联网。
直接调用时不加载用户别名；查询目标只支持内置命令名及别名，不输出用户自定义配置。
认证配置损坏或尚未登录时也可使用。格式及兼容规则见[机器可读命令说明](usage.md#机器可读命令说明-schema)。
