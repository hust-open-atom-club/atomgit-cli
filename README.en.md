# AtomGit CLI (`ag`) English quickstart

AtomGit CLI is a command-line client for [AtomGit](https://atomgit.com/). It
supports macOS, Linux, and Windows and provides repository, Issue, pull
request, Actions, release, and account workflows from a terminal.

This page is a concise English onboarding path. The [Chinese usage guide](docs/usage.md)
remains the complete reference while more translations are maintained.

## Install

Choose one channel. Homebrew Core, WinGet, Scoop, Nix, and npm publish the
stable release; the project Homebrew tap and source builds track development
snapshots.

### npm (Node.js 18+)

```bash
npm install -g @hust-open-atom-club/atomgit-cli
```

### Homebrew

```bash
# Latest stable release (recommended)
brew install atomgit-cli

# Development snapshot from the project tap
brew install hust-open-atom-club/tap/atomgit-cli
```

Do not install both Homebrew formulas at the same time. See the
[installation guide](docs/installation.md#homebrew-安装) for switching and
upgrade details.

### Windows package managers

```powershell
winget install HUSTOpenAtomClub.AtomGitCLI

scoop bucket add hust-open-atom-club https://github.com/hust-open-atom-club/ScoopBucket
scoop install atomgit-cli
```

### Nix or Go

```bash
# Nix stable package from nixos-unstable
nix profile install nixpkgs#atomgit-cli

# Go 1.24.2+
go install atomgit.com/hust-open-atom-club/atomgit-cli/cmd/ag@latest
```

Release archives and source-build instructions are in the
[installation guide](docs/installation.md). Verify an installation with:

```bash
ag version
```

## Authenticate safely

### Browser OAuth (recommended)

```bash
ag auth login
```

The command opens AtomGit in a browser, validates the authorization, and
saves the account locally. Check the current account with:

```bash
ag auth status
```

### Existing token, stdin only

For a sandbox, container, or CI job without a browser, pass a PAT or OAuth
access token through stdin. Keep the value in an environment variable or a
protected secret store; do not put it in shell history, a command argument,
or a committed file.

```bash
printf '%s' "$ATOMGIT_TOKEN" | ag auth login --with-token
# A protected file is also accepted:
ag auth login --with-token < "$HOME/.config/atomgit/token.txt"
```

`--with-token` reads stdin to EOF, validates the token against AtomGit, and
only then saves it. PAT logins do not have a refresh token; log in again when
the PAT expires.

Credentials are stored in the per-user configuration directory, normally:

- Linux: `~/.config/ag-cli/token.json`
- macOS: `~/.config/ag-cli/token.json`
- Windows: `%USERPROFILE%\.config\ag-cli\token.json`

The file is private to the current user. Do not commit or share it. See the
[configuration guide](docs/configuration.md) for multi-account selection,
token rotation, and output-safety details.

## Repository inference

Many commands accept either an explicit `owner/repo` or no repository at all.
When omitted, `ag` inspects the current Git repository's AtomGit remote.
Explicit input always wins.

```bash
git clone https://atomgit.com/hust-open-atom-club/atomgit-cli.git
cd atomgit-cli

ag repo view
ag issue list
ag pr list
```

If a checkout has multiple remotes and no unambiguous AtomGit remote, pass
`owner/repo` explicitly. GitHub and unrelated GitLab remotes are not used for
inference.

## Inspect repositories, Issues, and PRs

```bash
# Repository details; --json is suitable for scripts
ag repo view owner/repo
ag repo view owner/repo --json

# List open items (defaults to 30)
ag issue list owner/repo --limit 50
ag pr list owner/repo --state open --limit 50

# Inspect one item
ag issue view owner/repo 42
ag pr view owner/repo 123
```

Use `--state open`, `closed`, or `all` where supported. `--json` emits
machine-readable output; keep normal output for humans and pipelines that do
not need a schema.

## Common read/write workflows

Create an Issue:

```bash
ag issue create owner/repo --title "Improve the documentation" --body "Details"
```

Create a branch and inspect its status:

```bash
ag branch create owner/repo docs/quickstart --ref main
ag branch view owner/repo docs/quickstart
```

Open a pull request from a pushed branch:

```bash
ag pr create owner/repo \
  --title "docs: improve quickstart" \
  --body "Summary and test notes" \
  --head "your-user:docs/quickstart" \
  --base main
```

Destructive and high-impact commands ask for confirmation by default. Use
`--yes` only in a deliberate, reviewed automation step. Run each command's
help before scripting it:

```bash
ag pr create --help
ag issue edit --help
ag repo delete --help
```

## Actions and JSON examples

```bash
ag workflow list owner/repo --json
ag run list owner/repo --status failed --json
```

Actions commands use AtomGit API v8. The general repository API uses API v5.
See the [AtomGit API documentation](https://docs.atomgit.com/docs/apis/) for
endpoint details.

## Get help and contribute

```bash
ag --help
ag repo --help
ag pr --help
```

- [Complete usage guide](docs/usage.md)
- [Installation guide](docs/installation.md)
- [Configuration guide](docs/configuration.md)
- [Credential and output-safety guidance](docs/configuration.md#认证)
- [AtomGit PAT security guidance](https://docs.gitcode.com/docs/help/home/user_center/security_management/user_pat/)
- [Contribution guide](CONTRIBUTING.md)
- [AtomGit API documentation](https://docs.atomgit.com/docs/apis/)
- [Releases](https://atomgit.com/hust-open-atom-club/atomgit-cli/releases)
- [Issue tracker](https://atomgit.com/hust-open-atom-club/atomgit-cli/issues)

### Maintenance rule

When a command, authentication flow, installation channel, credential path,
or output flag changes, update the matching section above in the same pull
request. Keep every example executable against the current `--help` output;
run the repository's Markdown/link checks before submitting documentation
changes.

## License

[Mulan Permissive Software License, Version 2](LICENSE)
