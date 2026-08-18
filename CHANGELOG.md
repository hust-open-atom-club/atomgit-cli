atomgit-cli (v0.7.2) unstable; urgency=medium

  * Add command aliases, Actions workflow listing and dispatch, and token-based
    login through ag auth login --with-token.
  * Add commit listing, viewing, comparison, diff, and patch commands, plus
    repository content reading and owner-scoped repository listing.
  * Add issue collaboration details, pull request detail queries, discussion
    listing and viewing, and repository notification management.
  * Validate local arguments and repository context before authentication, keep
    authentication failures canonical, and harden API error handling.
  * Serialize alias configuration updates and make config writes atomic across
    Unix and Windows.
  * Add stable and latest Nix package definitions, expand release maintenance
    guidance, and correct the WinGet package identifier.
  * Improve Release creation status handling and command output safety for
    commit and discussion data.

 -- Dongliang Mu <dzm91@hust.edu.cn>  Wed, 19 Aug 2026 00:04:07 +0800

atomgit-cli (v0.7.1) unstable; urgency=medium

  * Add milestone management, repository fork synchronization, and direct
    browser shortcuts for Actions, wiki, and repository settings.
  * Add an explicit update-check command with installation-source-aware
    guidance.
  * Automate AtomGit Release publication while restoring the compact artifact
    contract of seven platform archives, two installers, and one checksum file.
  * Make installer templates version-neutral and document WinGet installation.
  * Split credential permission handling by platform and remove unreachable
    token-file repair logic.
  * Harden repository and API path validation, redact tokens consistently, and
    validate command arguments before authentication or configuration access.
  * Make unauthenticated command failures and repository visibility validation
    consistent across command families.

 -- Dongliang Mu <dzm91@hust.edu.cn>  Sun, 09 Aug 2026 19:48:53 +0800

atomgit-cli (v0.7.0) unstable; urgency=medium

  * Add branch and protected-branch management, repository editing, webhooks,
    collaborators, organization listing, labels, SSH keys, search, and generic
    authenticated API commands.
  * Add complete pull request workflows for checkout, merge, review, current
    head checks, collaboration metadata, reopening, and body-file input.
  * Add AtomGit Actions run inspection, browser and --web support, and
    structured JSON resource output.
  * Add Release metadata management and attachment upload/download, with
    source-aware self-update policy and isolated package-manager artifacts.
  * Infer repository context from Git remotes, recognize gitcode.com aliases,
    and support explicit multi-account authentication workflows.
  * Add Linux LoongArch64 Release and npm packages, split stable/latest Nix
    packages, and document Homebrew and Scoop installation.
  * Improve API client consistency, write-result identifiers and URLs, Actions
    empty responses, repository creation, CI reliability, and Windows test
    isolation.

 -- Dongliang Mu <dzm91@hust.edu.cn>  Tue, 28 Jul 2026 00:09:08 +0800

atomgit-cli (v0.6.0) unstable; urgency=medium

  * Add npm distribution through a launcher package and six platform-specific
    binary packages, with local checksum generation and release validation.
  * Add issue editing, issue label management, repository label listing, and
    richer issue display output.
  * Retry transient failures for idempotent API requests and accept successful
    empty PATCH responses.
  * Harden credential file permission and race checks, and sanitize untrusted
    terminal output at the CLI boundary.
  * Add the root --version flag and improve reproducible Nix and installer
    release workflows.

 -- Dongliang Mu <dzm91@hust.edu.cn>  Sun, 19 Jul 2026 00:37:42 +0800

atomgit-cli (v0.5.0) unstable; urgency=medium

  * Move the Go module, documentation, API endpoints, and installers to the
    AtomGit-hosted hust-open-atom-club/atomgit-cli repository.
  * Add `ag version` with text and JSON output, including version, commit, and
    build date metadata.
  * Improve repository commands with paginated listing, richer view output,
    explicit user and organization creation targets, and shared name parsing.
  * Resolve short clone names against the authenticated user and preserve
    requested descriptions when forking repositories.
  * Make `issue list` and `pr list` enforce `--limit`, and default new pull
    requests to the repository's default branch when `--base` is omitted.
  * Fix browser-based OAuth login on Windows so authorization URLs are passed
    to the browser without command-line truncation.
  * Add Makefile and Nix development workflows and expand automated coverage
    for commands, configuration, OAuth, API behavior, and release packaging.
  * Package reproducible Linux, macOS, and Windows amd64/arm64 archives with
    GoReleaser, validated release tags, snapshots, and complete checksums.

 -- Dongliang Mu <dzm91@hust.edu.cn>  Mon, 13 Jul 2026 15:12:29 +0800

atomgit-cli (v0.4) unstable; urgency=medium

  * Add display of pull request review comments and nested replies.
  * Add replies to pull request review discussions.
  * Refresh the macOS, Linux, and Windows installation scripts and release
    artifacts.

 -- xiaogang <xiaogang@csdn.net>  Sat, 11 Jul 2026 00:26:24 +0800

atomgit-cli (v0.3) unstable; urgency=medium

  * Add browser-based AtomGit OAuth login, token refresh, status, and logout.
  * Add `pr diff` with optional JSON output.
  * Support `owner:branch` head references for cross-repository pull requests.
  * Convert HTML tables in comments to Markdown and fix comment creation
    response decoding.
  * Add installation scripts and release archives for macOS, Linux, and
    Windows on amd64 and arm64.

 -- xiaogang <xiaogang@csdn.net>  Sat, 11 Jul 2026 00:26:23 +0800

atomgit-cli (v0.2) unstable; urgency=medium

  * Follow the XDG Base Directory specification when locating the token file,
    while retaining the legacy token path as a fallback.
  * Fix comment ID decoding for AtomGit API responses.

 -- Shinwell Hu <huxinwei@huawei.com>  Sat, 11 Jul 2026 00:25:46 +0800

atomgit-cli (v0.1) unstable; urgency=medium

  * Add the initial AtomGit CLI with repository, pull request, issue,
    authentication, and SSH key commands.
  * Add pull request and issue comments, pull request close, issue close, and
    pull request issue linking commands.
  * Add repository tag management and license compliance checks.
  * Support string and numeric issue and pull request numbers returned by the
    AtomGit API.
  * Fix AtomGit user ID decoding and remove generated binaries from the source
    repository.

 -- Shinwell Hu <huxinwei@huawei.com>  Mon, 2 Feb 2026 09:17:41 +0800
