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
