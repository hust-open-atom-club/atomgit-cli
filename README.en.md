# AtomGit CLI (ag)

[![License](https://img.shields.io/badge/license-MulanPSL--2.0-blue.svg)](LICENSE)
[![npm](https://img.shields.io/npm/v/%40hust-open-atom-club%2Fatomgit-cli?logo=npm)](https://www.npmjs.com/package/@hust-open-atom-club/atomgit-cli)
[![Latest Release](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.atomgit.com%2Fapi%2Fv5%2Frepos%2Fhust-open-atom-club%2Fatomgit-cli%2Freleases%2Flatest&query=%24.tag_name&label=release)](https://atomgit.com/hust-open-atom-club/atomgit-cli/releases)
[![CI Status](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.atomgit.com%2Fapi%2Fv8%2Frepos%2Fhust-open-atom-club%2Fatomgit-cli%2Factions%2Fruns%3Fworkflow_name%3DCI%26branch%3Dmain%26per_page%3D1&query=%24.workflow_runs%5B0%5D.status&label=CI%20status)](https://atomgit.com/hust-open-atom-club/atomgit-cli/actions)
[![Homebrew](https://img.shields.io/badge/homebrew-core-FBB040?logo=homebrew&logoColor=white)](https://formulae.brew.sh/formula/atomgit-cli)
[![Go Reference](https://pkg.go.dev/badge/atomgit.com/hust-open-atom-club/atomgit-cli.svg)](https://pkg.go.dev/atomgit.com/hust-open-atom-club/atomgit-cli)
[![GoReleaser](https://img.shields.io/badge/powered_by-GoReleaser-69D7E4?logo=goreleaser&logoColor=white)](https://goreleaser.com/)

A command-line client for AtomGit, developed with reference to GitHub CLI (`gh`).

中文说明：[README.md](README.md)

## Features

| Category | Capabilities |
| --- | --- |
| 📦 Repositories | List, view, create, edit, clone, delete, fork, and synchronize repositories; manage repository push rules |
| 👥 Collaborators | List, view, add, edit, and remove repository collaborators |
| 🔔 Webhooks | List, view, create, edit, delete, and test repository webhooks |
| 🌿 Branches | List, view, create, and delete branches; manage branch protection rules |
| 🔀 Pull Requests | List, view, create, edit, close, reopen, review, merge, and check out PRs; inspect diffs and checks; manage comments and linked Issues |
| 🐛 Issues | List, view, create, edit, close, and reopen Issues; manage labels and comments |
| 🔖 Labels | List, create, edit, and delete repository labels |
| 🎯 Milestones | List, view, create, edit, close, reopen, and delete milestones |
| 🏷️ Tags | List, create, and delete Git tags; manage protected tag rules |
| 🚀 Releases | List, view, create, and edit Releases; upload and download assets |
| ⚙️ Actions | List, validate, and trigger workflows; inspect workflow runs, jobs, logs, and artifacts; download logs and artifacts |
| 🏢 Organizations | List organizations joined by the current account |
| 🔍 Search | Search repositories, users, and Issues |
| 💬 Discussions | List repository Discussions |
| 🔔 Notifications | List repository notifications and mark them as read |
| 🔐 Authentication and SSH keys | OAuth login, token refresh, account switching, authentication status, and SSH public-key management |
| 🌐 API | Call AtomGit API v5 with pagination, JSON request bodies, and GET, POST, PATCH, PUT, and DELETE methods |

## AI Agent Skills

[AtomGit Skills](https://atomgit.com/hust-open-atom-club/atomgit-skills) provides Codex Skills powered by `ag` for Issue, Pull Request, CLI release, and GitHub mirroring workflows. See that repository for installation instructions and the complete list.

## Installation

### npm

```bash
npm install -g @hust-open-atom-club/atomgit-cli
```

### Homebrew

Install the latest stable release from Homebrew Core (recommended):

```bash
brew install atomgit-cli
```

To track a development snapshot from the latest commit on AtomGit `main`, use the project-maintained Homebrew tap instead:

```bash
brew install hust-open-atom-club/tap/atomgit-cli
```

Homebrew Core provides stable releases, while the project tap provides development snapshots. Both formulas use the same name and cannot be installed together. See the [complete installation guide](docs/installation.md#homebrew-安装) for differences and switching instructions.

### WinGet

```powershell
winget install HUSTOpenAtomClub.AtomGitCLI
```

### Scoop

```powershell
scoop bucket add hust-open-atom-club https://github.com/hust-open-atom-club/ScoopBucket
scoop install atomgit-cli
```

### Nix / NixOS

`atomgit-cli` is currently available only in `nixos-unstable`. Make sure `nixpkgs` points to unstable; stable channels do not include it yet.

```bash
nix profile install nixpkgs#atomgit-cli
```

### AUR / Arch Linux

`atomgit-cli` is available in the Arch User Repository (AUR). Use `yay` or `paru` to choose a stable source package, a stable prebuilt binary, or a development package.

```bash
# Stable release built from source
yay -S atomgit-cli
# Stable prebuilt binary
yay -S atomgit-cli-bin
# Development version tracking the latest commit on main
yay -S atomgit-cli-git
```

### Go

```bash
go install atomgit.com/hust-open-atom-club/atomgit-cli/cmd/ag@latest
```

See the [complete installation guide](docs/installation.md) for requirements, upgrades, AtomGit Releases, and source installation.

## Configuration

Run `ag auth login` for the initial OAuth login. In an environment without a browser, such as a sandbox, container, or CI job, use `echo "$TOKEN" | ag auth login --with-token` to authenticate with an existing access token. See the [configuration guide](docs/configuration.md) for credentials, output safety, and repository inference.

## Usage

Run `ag --help` for a command overview or `ag <command> --help` for command-specific options. See the [usage guide](docs/usage.md) for complete examples and explanations.

See the [FAQ](docs/faq.md) for common installation, authentication, usage, and troubleshooting questions.

## More Documentation

- [Release guide](docs/releasing.md)
- [Project structure](docs/project-structure.md)

## API

General repository features use AtomGit API v5: `https://api.atomgit.com/api/v5`.

Actions run checks use the separate AtomGit API v8: `https://api.atomgit.com/api/v8`.

### References

- [AtomGit API documentation](https://docs.atomgit.com/docs/apis/)
- [GitHub CLI](https://cli.github.com/)

## License

[Mulan Permissive Software License, Version 2](LICENSE)

Copyright (c) 2026 HUST OpenAtom Club, AtomGit, and the AtomGit CLI contributors
