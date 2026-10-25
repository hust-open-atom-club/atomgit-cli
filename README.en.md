# AtomGit CLI (`ag-cli`)

Efficiently manage AtomGit repositories, issues, and pull requests, inspect Actions runs, and publish releases directly from your terminal. AtomGit CLI provides a rich set of command-line tools and general-purpose API wrappers for easy integration into automation scripts and CI/CD workflows.

[中文文档](README.md) · [Installation Guide](docs/installation.md) · [Usage Guide](docs/usage.md) · [Command Reference](docs/command-reference.md)

[![License](https://img.shields.io/badge/license-MulanPSL--2.0-blue.svg)](LICENSE)
[![npm](https://img.shields.io/npm/v/%40hust-open-atom-club%2Fatomgit-cli?logo=npm)](https://www.npmjs.com/package/@hust-open-atom-club/atomgit-cli)
[![Latest Release](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.atomgit.com%2Fapi%2Fv5%2Frepos%2Fhust-open-atom-club%2Fatomgit-cli%2Freleases%2Flatest&query=%24.tag_name&label=release)](https://atomgit.com/hust-open-atom-club/atomgit-cli/releases)
[![CI Status](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.atomgit.com%2Fapi%2Fv8%2Frepos%2Fhust-open-atom-club%2Fatomgit-cli%2Factions%2Fruns%3Fworkflow_name%3DCI%26branch%3Dmain%26per_page%3D1&query=%24.workflow_runs%5B0%5D.status&label=CI%20status)](https://atomgit.com/hust-open-atom-club/atomgit-cli/actions)

---

## Features

| Category | Capabilities |
| --- | --- |
| Repositories | List, view, create, edit, clone, delete, fork, and synchronize repositories; manage repository push rules |
| Collaborators | List, view, add, edit, and remove repository collaborators |
| Webhooks | List, view, create, edit, delete, and test repository webhooks |
| Branches | List, view, create, and delete branches; manage branch protection rules |
| Pull Requests | List, view, create, edit, close, reopen, review, merge, and check out PRs; inspect diffs and checks; manage comments and linked Issues |
| Issues | List, view, create, edit, close, and reopen Issues; manage labels and comments |
| Labels | List, create, edit, and delete repository labels |
| Milestones | List, view, create, edit, close, reopen, and delete milestones |
| Tags | List, create, and delete Git tags; manage protected tag rules |
| Releases | List, view, create, and edit Releases; upload and download assets |
| Actions | List, validate, and trigger workflows; inspect workflow runs, jobs, logs, and artifacts; download logs and artifacts |
| Organizations | List organizations joined by the current account |
| Search | Search repositories, users, and Issues |
| Discussions | List repository Discussions |
| Notifications | List repository notifications and mark them as read |
| Authentication and SSH keys | OAuth login, token refresh, account switching, authentication status, and SSH public-key management |
| API | Call AtomGit API v5 with pagination, JSON request bodies, and GET, POST, PATCH, PUT, and DELETE methods |

---

> [!IMPORTANT]
> **Breaking Change**
>
> The main command name has changed from `ag` to **`ag-cli`**. The new version does not provide an `ag` compatibility command. Existing configurations and credentials remain compatible, so re-authentication is not required after upgrading.
> Existing users should review the [Migration Guide](docs/installation.md#从-ag-迁移到-ag-cli) and promptly update command calls in automation scripts and CI configuration files. Update timing may vary by installation channel; verify the command name after upgrading.

---

## Installation

Choose your preferred package manager to install quickly:

| Package Manager | Environment | Installation Command |
| :--- | :--- | :--- |
| **[npm](docs/installation.md#npm-安装)** | macOS / Linux / Windows; Node.js 18+ required | `npm install -g @hust-open-atom-club/atomgit-cli` |
| **[Homebrew](docs/installation.md#homebrew-安装)** | macOS / Linux | `brew install atomgit-cli` |
| **[WinGet](docs/installation.md#winget-安装)** | Windows | `winget install HUSTOpenAtomClub.AtomGitCLI` |

For other package managers, installation scripts, manual downloads, source builds, upgrades, uninstallation, and shell completion, see the [Installation Guide](docs/installation.md).

---

## Getting Started

1. **Verify installation**

   ```bash
   ag-cli version
   ```

2. **Log in**
   Log in to AtomGit and follow the prompts to complete browser authorization:
   ```bash
   ag-cli auth login
   ```

3. **List repositories**

   ```bash
   ag-cli repo list
   ```

> [!TIP]
> **Log in with an existing access token**
>
> If you already have a Personal Access Token, you can pass it to the CLI through standard input to log in. This also works on headless servers and in CI/CD containers.
> Store the token in a `TOKEN` variable and run:
> ```bash
> echo "$TOKEN" | ag-cli auth login --with-token
> ```
> For details on token scopes, expiration, and credential management, see the [Configuration Guide](docs/configuration.md).

Use `ag-cli --help` to view a command overview, or run `ag-cli pr create --help` to view flags for a specific command. You can then refer to the [Usage Guide](docs/usage.md) to manage issues, review PRs, or view workflow runs.

---

## AI Agent Skills

[AtomGit Skills](https://atomgit.com/hust-open-atom-club/atomgit-skills) provides Codex Skills powered by `ag-cli`, covering workflows such as issue management, PR handling, automated CLI releases, and GitHub mirror synchronization. For installation and configuration, visit the repository.

---

## Documentation Index

| Topic | Start Here |
| :--- | :--- |
| Installation, upgrades, uninstallation, and shell completion configuration | [Installation Guide](docs/installation.md) |
| Authentication, credential management, output safety, and repository context inference | [Configuration Guide](docs/configuration.md) |
| Examples for common development scenarios | [Usage Guide](docs/usage.md) |
| Detailed command list, options, defaults, and aliases (generated from the Cobra command tree) | [Command Reference](docs/command-reference.md) |
| Troubleshooting common errors and questions | [FAQ](docs/faq.md) |
| Contributing Guide, including project structure and release process | [Contributing Guide](CONTRIBUTING.md), [Project Structure](docs/project-structure.md), [Release Guide](docs/releasing.md) |
| Correspondence with AtomGit's official APIs, implemented functionality, and known gaps | [OpenAPI Coverage & Responsibility Checklist](docs/openapi-coverage.md) |

> [!CAUTION]
> **Security Vulnerability Reporting**
>
> If you discover a security vulnerability, please report it privately according to the [Security Policy](SECURITY.md). Do not disclose sensitive information in public issues or PRs.

---

## References & Acknowledgements

AtomGit CLI standard features are built on **AtomGit API v5** (`https://api.atomgit.com/api/v5`), while the Actions module uses **API v8** (`https://api.atomgit.com/api/v8`).

- [AtomGit API Documentation](https://docs.atomgit.com/docs/apis/)
- [GitHub CLI](https://cli.github.com/) (interaction and architecture design reference for this project)

---

## License

This project is released under the [**Mulan Permissive Software License, Version 2 (Mulan PSL v2)**](LICENSE).

Copyright (c) 2026 HUST OpenAtom Club, AtomGit, and the AtomGit CLI contributors
