# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of AtomGit CLI
- Authentication commands: `auth status`, `auth token`
- Repository commands: `repo list`, `repo view`, `repo create`, `repo clone`, `repo fork`, `repo delete`
- Pull Request commands: `pr list`, `pr view`, `pr create`
- Issue commands: `issue list`, `issue view`, `issue create`
- SSH Key commands: `ssh-key add`
- Configuration management via `~/.atomgit_personal_token.json`
- HTTP API client for AtomGit API v5

### Features

#### Authentication
- Read access token from `~/.atomgit_personal_token.json`
- Display authentication status with masked token
- Show current token

#### Repository Management
- List all accessible repositories
- View repository details (stars, forks, issues, etc.)
- Create new repositories (public/private)
- Clone repositories with optional branch selection
- Fork repositories with custom name and visibility
- Delete repositories with confirmation prompt

#### Pull Request Management
- List PRs with state filtering (open/closed/all)
- View PR details including branch information
- Create new PRs with title, body, base and head branches

#### Issue Management
- List issues with state filtering
- View issue details
- Create new issues with title and body

#### SSH Key Management
- Add SSH keys to AtomGit account
- Support for reading key from file or stdin

### Technical Details
- Built with Go 1.21+
- Uses Cobra framework for CLI commands
- RESTful API client with standard library
- JSON-based configuration

## [0.1.0] - 2026-01-30

### Added
- Project initialization
- Basic command structure following gh-cli patterns
- Core API client implementation
- Initial set of commands for repository, PR, and issue management

[Unreleased]: https://gitcode.com/openeuler/ag-cli/compare/v0.1.0...HEAD
[0.1.0]: https://gitcode.com/openeuler/ag-cli/releases/tag/v0.1.0
