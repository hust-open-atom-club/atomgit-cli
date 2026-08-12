# 常见问题（FAQ）

本文档收集 AtomGit CLI（`ag`）使用过程中的常见问题。安装方法请参阅[安装指南](installation.md)，认证与其他配置请参阅[配置指南](configuration.md)，命令示例请参阅[命令使用指南](usage.md)。

## 安装

### `ag` 支持哪些操作系统和架构？

| 操作系统 | 支持的处理器架构 |
| --- | --- |
| macOS | x64（Intel）、arm64（Apple Silicon） |
| Linux | amd64、arm64 / aarch64、loong64 / loongarch64 |
| Windows | amd64、arm64 |

### 为什么不能用 apt / dnf / zypper / pacman（AUR）等发行版包管理器安装？

项目目前没有发布官方维护的 Debian/Ubuntu（apt）、Fedora（dnf）、openSUSE（zypper）、Arch Linux（AUR）、Alpine（apk）等发行版专用包。截至本文档撰写时，AUR、Debian/Ubuntu 和 Fedora 的官方仓库中均检索不到 `atomgit-cli` 包。

官方安装渠道包括：

- npm：`npm install -g @hust-open-atom-club/atomgit-cli`
- Homebrew tap：`brew install hust-open-atom-club/tap/atomgit-cli`（macOS 或 Linux 上均可使用）
- WinGet：`winget install HUSTOpenAtomClub.AtomGitCLI`（仅 Windows）
- Scoop：`scoop bucket add hust-open-atom-club https://github.com/hust-open-atom-club/ScoopBucket && scoop install atomgit-cli`（仅 Windows）
- Nix / NixOS：`nix profile install nixpkgs#atomgit-cli`（`nixos-unstable` 已收录）
- Go：`go install atomgit.com/hust-open-atom-club/atomgit-cli/cmd/ag@latest`（需 Go 1.24.2 或更高版本）
- AtomGit Release：使用 `install.sh` / `install.ps1` 自动安装，或从 [Release 页面](https://atomgit.com/hust-open-atom-club/atomgit-cli/releases)手动下载对应平台的归档

> [!NOTE]
> 如果你的发行版社区自行维护了 `atomgit-cli` 包，其更新节奏、完整性和安全性由该社区负责，与官方 Release 不一定同步。建议优先使用上述官方渠道。

### 如何选择安装方式？

各渠道的适用场景不同，可按需选择：

- **npm**：需要 Node.js 18 或更高版本，适合已在使用 Node 生态的开发者；主包根据当前操作系统和 CPU 架构自动安装预编译二进制。
- **Homebrew**：macOS 或 Linux 上通过项目维护的 tap 安装，由 Homebrew 管理升级。
- **WinGet / Scoop**：Windows 用户可直接使用系统包管理器安装与升级。
- **Nix / NixOS**：NixOS 用户或已使用 Nix 的系统，包已进入 `nixos-unstable`。
- **`go install`**：适合 Go 开发者；从模块代理构建，`ag version` 中 commit 和构建时间可能为 `unknown`。
- **AtomGit Release**：通过 `install.sh` / `install.ps1` 自动安装或手动下载归档，不依赖 Node.js、包管理器或 Go 工具链。

### npm 安装后 `ag` 无法启动？

npm 主包通过 `optionalDependencies` 声明平台二进制包，npm 会根据当前操作系统和 CPU 架构只安装匹配的包。如果安装时使用了 `--omit=optional`（或旧版 npm 的 `--no-optional`），平台二进制包会被跳过，`ag` 无法启动。请去掉该选项后重新安装：

```bash
npm install -g @hust-open-atom-club/atomgit-cli
```

npm 安装需要 Node.js 18 或更高版本。安装完成后执行 `ag version` 验证。

相关 Issue：[#46 npm postinstall](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/46)

### npm 提示找不到 `@hust-open-atom-club/atomgit-cli`？

请确认包名拼写正确。官方 npm 包名是 `@hust-open-atom-club/atomgit-cli`，不要误用 `@atomgit/atomgit-cli` 等名称（该包在 npm 上不存在，安装会报 404 / 找不到包）。安装命令：

```bash
npm install -g @hust-open-atom-club/atomgit-cli
```

相关 Issue：[#20 npm install -g @atomgit/atomgit-cli](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/20)

### 为什么 `go install` 的 `ag version` 显示 `unknown` 或缺少 commit？

Go 模块代理提供的源码包不包含 `.git` 目录，因此通过 `go install` 构建的二进制只能可靠获得模块版本：文本输出会省略无法获得的 commit 和构建时间，`ag version --json` 中对应字段为 `unknown`。这是预期行为，不影响 CLI 功能。如需完整的版本、commit 和构建时间元数据，请使用 npm、Homebrew 或 AtomGit Release 预编译版本。

相关 Issue：[#61 go install 安装的二进制缺少 commit 和构建时间信息](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/61)

### 从源码构建后 `ag version` 显示 `dev`？

通过 `make build` / `make install` 构建且未注入发布元数据时，版本默认值为 `dev`。如果 Go 构建信息包含模块版本、源码提交或提交时间，命令会使用这些信息替代或补充默认值；工作区存在未提交改动时，版本还会带有 dirty 标记。

### 如何升级？

- npm：`npm update -g @hust-open-atom-club/atomgit-cli`
- Homebrew：`brew update && brew upgrade atomgit-cli`
- WinGet：`winget upgrade HUSTOpenAtomClub.AtomGitCLI`
- Scoop：`scoop update atomgit-cli`
- Nix：`nix profile upgrade`（升级 profile 中使用未锁定 flake 引用安装的全部包）；NixOS 系统级安装则通过 `nixos-rebuild switch` 跟随系统升级

也可以先运行 `ag check-update` 查看是否有新版本，再使用对应的安装方式升级。

### 为什么不应混用不同的安装来源？

不同安装来源由各自的工具管理升级：npm 用 `npm update`、Homebrew 用 `brew upgrade`、WinGet 用 `winget upgrade`、Scoop 用 `scoop update`、Nix 用 `nix profile upgrade`。混用多个来源会在 `PATH` 中留下多份 `ag` 二进制，实际调用的是 `PATH` 顺序中最先出现的那一份，升级时可能改到"看不见"的另一份，导致 `ag version` 与预期不一致。建议固定使用一种安装来源，并始终用它升级。

### 安装到了哪个目录？提示目录不在 PATH？

`install.sh` 默认将 `ag` 安装到 `/usr/local/bin`；该目录不可写时，会改用 `~/.local/bin`。如果安装目录不在 `PATH` 中，脚本会输出相应的配置提示，按提示把目录加入 `PATH` 即可。Windows 的 `install.ps1` 默认安装到 `%USERPROFILE%\.local\bin`，并将该目录加入当前用户的 `Path`。

### 系统里已有其他叫 `ag` 的命令（如 the-silver-searcher）怎么办？

`ag` 是较常见的命令名，可能与 the-silver-searcher 等工具冲突。`PATH` 中先出现的可执行文件会被调用，可用 `which ag`（Windows 用 `where ag`）确认当前实际调用的是哪一个。需要区分时，可使用完整路径调用（如 `~/.local/bin/ag`），或为 AtomGit CLI 设置 shell 别名（如 `alias atg='ag'`）。

## 认证与配置

### 如何登录？

首次使用前先运行：

```bash
ag auth login
```

会在浏览器中完成 AtomGit OAuth 授权，并自动将认证信息写入令牌文件。如果已经登录，命令会提示 `Already logged in` 并跳过浏览器；需要重新走一遍浏览器授权时，可显式使用 `--force`：

```bash
ag auth login --force
```

也可以手动创建 PAT（个人访问令牌）并写入令牌文件，具体格式见[配置指南](configuration.md)。

### 提示 `not authenticated: run 'ag auth login'`？

表示当前没有可用的访问令牌。先执行 `ag auth login` 完成登录，或手动配置 PAT 后重试。

### 令牌文件在哪里？

- Linux：`/home/<用户名>/.config/ag-cli/token.json`
- macOS：`/Users/<用户名>/.config/ag-cli/token.json`
- Windows：`C:\Users\<用户名>\.config\ag-cli\token.json`

令牌文件使用 `0600` 权限并通过原子写入更新，请勿提交到版本控制或分享给他人。执行任意 `ag auth` 子命令时，CLI 会自动检查令牌文件格式：旧版单账号文件会通过安全的原子写入自动升级为当前多账号格式，已有的 `refresh_token` 等 OAuth 字段会保留；当前格式不会重复写入，未知版本会拒绝迁移。

相关 Issue：[#13 ag 检查 token.json 文件的权限](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/13)

### OAuth 登录时提示端口被占用？

登录流程默认在本机 **8765** 端口等待回调。如果该端口被占用，可设置环境变量 `AG_OAUTH_REDIRECT_PORT` 改用其他端口（需与 AtomGit 应用配置的回调地址一致）：

```bash
AG_OAUTH_REDIRECT_PORT=9000 ag auth login
```

### 如何管理多个账号？

首次登录的账号会自动成为活动账号；之后 `ag auth login --force` 只新增或更新账号，**不会**隐式切换活动账号，以免意外改变后续命令的认证身份。需要切换时：

```bash
ag auth list                # 列出所有账号
ag auth switch <account>    # 切换活动账号
```

相关 Issue：[#59 多账号认证与 Git 身份同步](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/59)

### 为什么 `ag auth switch` 会修改 Git 身份？

`auth switch` 默认会把账号对应的身份写入当前仓库的 `git config --local user.name/user.email`，确保该仓库的提交署名与活动账号一致；只有 `--global` 才修改全局配置，`--no-git` 可明确禁用同步。账号没有 Git 邮箱或名称时，必须通过 `--git-email` / `--git-name` 指定，或使用 `--no-git`。`--no-git` 与 `--global`、`--git-name`、`--git-email` 互斥；Git identity 写入失败不会切换活动凭据。

### 为什么 `ag auth logout` 不能直接删除活动账号？

为避免删除动作隐式选择另一个账号并造成 Git identity 错配，当仍有其他账号时不能删除活动账号。应先 `ag auth switch <account>` 切换到其他账号，再用 `ag auth logout --account <old-account>` 删除原账号。`ag auth logout --all` 可删除全部账号。

### 令牌泄露了怎么办？

删除本机令牌文件（或执行 `ag auth logout`）后，再到 AtomGit「个人设置」中撤销对应的 PAT / OAuth 授权。access token、refresh token 不会写入 Git 配置、remote URL、账号列表或错误信息。

## 使用

### 为什么有些命令可以省略 `owner/repo`？

`issue`、`pr`、`tag`、`label`、`release` 命令以及 `repo view`、`repo edit`、`repo fork`、`repo delete` 可以省略 `owner/repo`，此时 `ag` 会从当前 Git 仓库的 AtomGit remote 推断目标仓库；显式传入的 `owner/repo` 始终优先。CLI 只识别指向 AtomGit/GitCode 主机（`atomgit.com` 与 `gitcode.com`）的 remote（`git@atomgit.com:owner/repo.git`、`ssh://git@atomgit.com/...`、`https://atomgit.com/...`），因此 GitHub、GitLab 等其他服务的 remote 不会被当作 AtomGit 仓库，避免对错误仓库执行操作；无法唯一确定时，请显式传入 `owner/repo`。详见[配置指南](configuration.md)。

相关 Issue：[#23 从 Git remote 推断仓库](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/23)、[#52 支持 gitcode.com 主机](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/52)

### 输出里出现 `\x1b`、`\u001b` 之类的转义文本？

`ag` 默认会对输出做终端安全清理：把控制字符转换为可见的转义表示，防止仓库、Issue、PR 或 Git 服务端返回的内容注入终端控制序列（CWE-150），管道转发时同样生效。需要为机器处理保留原始字节时，可显式使用全局参数 `--raw-output`（如 `ag --raw-output pr diff owner/repo 123`）。请勿将未经检查的原始输出直接转发到终端。

### `ag check-update` 报错说版本不可比较？

`ag check-update` 按 SemVer 比较当前版本与最新稳定 Release。如果本地版本是 `dev`、dirty、提交哈希或其他无法比较的值，会返回清晰错误。这属于预期行为；从源码构建且未注入发布元数据时属于正常情况，改用官方预编译版本即可获得可比较的版本号。

### 如何查看某个命令的完整参数？

所有命令均支持 `--help`：

```bash
ag --help
ag pr --help
ag pr create --help
```

全局参数包括 `--help`、`--version`、`--raw-output` 等。

### 没有专用子命令时，如何使用 `ag api` 调用 API？

`ag api` 可以对 AtomGit API v5 的相对路径发起认证请求，默认使用 GET，支持 GET、POST、PATCH、PUT、DELETE：

```bash
ag api /user
ag api /repos/owner/repo/issues --field state=open
ag api /repos/owner/repo/issues --method POST --field title="New issue"
ag api /repos/owner/repo/issues --paginate
```

`--field key=value` 添加查询参数或请求字段，`--input` 从文件或 stdin（`-`）读取原始请求体，`--paginate` 拉取全部分页并以紧凑 JSON 页面逐行输出。`ag api` 需要登录；响应输出同样经过终端安全清理，需要原始字节时使用 `--raw-output`。

相关 Issue：[#28 添加通用 ag api 命令](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/28)

### `ag` 支持 GitHub / GitLab 吗？

不支持。`ag` 只与 AtomGit/GitCode 交互，仅识别 `atomgit.com` 与 `gitcode.com` 的 remote。GitHub 上的同类需求请使用 [GitHub CLI (gh)](https://cli.github.com/)。

## 故障排查

### 命令失败时退出码是多少？

顶层命令执行失败时返回非零退出码（当前为 1），错误信息写入 stderr。脚本中可以用退出码判断命令是否成功。

### 命令报网络 / API 错误？

`ag` 需要访问 `https://api.atomgit.com` 的 API v5 与 API v8（Actions 相关）接口。请检查网络连接、代理或防火墙设置。错误信息包含操作上下文并包装了底层错误，但不会泄露 access token、refresh token 等敏感信息。

### `ag run view` 报 job not found？

当指定的 Actions job 不存在（或 run 中根本没有该 job）时，`ag run view` 会返回类似 `job <id> not found (API returned an empty response)` 的错误。这属于预期行为：先执行 `ag run list` / `ag run view <run-id>` 确认 run 中实际的 job ID 后再查询。

相关 Issue：[#50 run view 对不存在的 job 报告 EOF](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues/50)

### 还有问题？

可以在 [AtomGit CLI 仓库](https://atomgit.com/hust-open-atom-club/atomgit-cli)发起 Issue 反馈，并提供 `ag version` 输出、操作系统与架构、复现步骤以及脱敏后的错误信息，便于快速定位问题。
