# AtomGit CLI command reference

> This file is generated from the Cobra command tree. Do not edit it manually.
> Regenerate with `go run ./scripts/generate-command-reference`.

> Only commands registered in the root command tree are emitted. The optional Cobra completion helper is not registered by this CLI and is intentionally omitted.

## Command index

- [ag](#ag) — AtomGit CLI
- [ag alias](#ag-alias) — Create command shortcuts
- [ag alias delete](#ag-alias-delete) — Delete an alias
- [ag alias list](#ag-alias-list) — List aliases
- [ag alias set](#ag-alias-set) — Create a shortcut for an ag command
- [ag api](#ag-api) — Make an authenticated AtomGit API request
- [ag auth](#ag-auth) — Authenticate with AtomGit
- [ag auth list](#ag-auth-list) — List saved AtomGit accounts
- [ag auth login](#ag-auth-login) — Log in with AtomGit OAuth (opens browser, saves token.json)
- [ag auth logout](#ag-auth-logout) — Remove the active or a selected stored account
- [ag auth refresh](#ag-auth-refresh) — Refresh the access token using the stored refresh_token
- [ag auth status](#ag-auth-status) — View authentication status
- [ag auth switch](#ag-auth-switch) — Switch the active account and synchronize Git identity
- [ag auth token](#ag-auth-token) — Print the authentication token
- [ag branch](#ag-branch) — Manage remote branches
- [ag branch create](#ag-branch-create) — Create a remote branch
- [ag branch delete](#ag-branch-delete) — Delete a remote branch
- [ag branch list](#ag-branch-list) — List remote branches
- [ag branch protection](#ag-branch-protection) — Manage protected branch rules
- [ag branch protection delete](#ag-branch-protection-delete) — Delete a protected branch rule
- [ag branch protection list](#ag-branch-protection-list) — List protected branch rules
- [ag branch protection set](#ag-branch-protection-set) — Create or update a protected branch rule
- [ag branch protection view](#ag-branch-protection-view) — View a protected branch rule
- [ag branch view](#ag-branch-view) — View a remote branch
- [ag browse](#ag-browse) — Open repositories, issues, pull requests, and more in the browser
- [ag check-update](#ag-check-update) — Check for a newer AtomGit CLI release
- [ag commit](#ag-commit) — Manage commits
- [ag commit compare](#ag-commit-compare) — Compare two commits, branches, or tags
- [ag commit diff](#ag-commit-diff) — Show a commit's diff
- [ag commit list](#ag-commit-list) — List commits
- [ag commit patch](#ag-commit-patch) — Show a commit's patch
- [ag commit view](#ag-commit-view) — View a commit
- [ag discussion](#ag-discussion) — View repository discussions
- [ag discussion list](#ag-discussion-list) — List repository discussions
- [ag discussion view](#ag-discussion-view) — View a repository discussion
- [ag issue](#ag-issue) — Manage issues
- [ag issue branches](#ag-issue-branches) — List or update related branches for an issue
- [ag issue close](#ag-issue-close) — Close an issue
- [ag issue comment](#ag-issue-comment) — Manage issue comments
- [ag issue comment create](#ag-issue-comment-create) — Create a comment on an issue
- [ag issue comment delete](#ag-issue-comment-delete) — Delete a comment on an issue
- [ag issue comment edit](#ag-issue-comment-edit) — Edit a comment on an issue
- [ag issue comment view](#ag-issue-comment-view) — View all comments on an issue
- [ag issue create](#ag-issue-create) — Create an issue
- [ag issue edit](#ag-issue-edit) — Edit an issue
- [ag issue label](#ag-issue-label) — Add or remove labels on an issue
- [ag issue list](#ag-issue-list) — List issues
- [ag issue prs](#ag-issue-prs) — List pull requests linked to an issue
- [ag issue reopen](#ag-issue-reopen) — Reopen an issue
- [ag issue view](#ag-issue-view) — View an issue
- [ag kanban](#ag-kanban) — View organization Kanban boards
- [ag kanban items](#ag-kanban-items) — List items on a Kanban board
- [ag kanban list](#ag-kanban-list) — List organization Kanban boards
- [ag kanban view](#ag-kanban-view) — View a Kanban board
- [ag label](#ag-label) — Manage repository labels
- [ag label create](#ag-label-create) — Create a repository label
- [ag label delete](#ag-label-delete) — Delete a repository label
- [ag label edit](#ag-label-edit) — Edit a repository label
- [ag label list](#ag-label-list) — List repository labels
- [ag license](#ag-license) — License compliance checking
- [ag license check](#ag-license-check) — Check license compliance
- [ag milestone](#ag-milestone) — Manage repository milestones
- [ag milestone close](#ag-milestone-close) — Close a repository milestone
- [ag milestone create](#ag-milestone-create) — Create a repository milestone
- [ag milestone delete](#ag-milestone-delete) — Delete a repository milestone
- [ag milestone edit](#ag-milestone-edit) — Edit a repository milestone
- [ag milestone list](#ag-milestone-list) — List repository milestones
- [ag milestone reopen](#ag-milestone-reopen) — Reopen a repository milestone
- [ag milestone view](#ag-milestone-view) — View a repository milestone
- [ag notification](#ag-notification) — Manage repository notifications
- [ag notification list](#ag-notification-list) — List repository notifications
- [ag notification mark-read](#ag-notification-mark-read) — Mark repository notifications as read
- [ag org](#ag-org) — Manage organizations
- [ag org list](#ag-org-list) — List organizations for the authenticated user
- [ag org members](#ag-org-members) — List organization members
- [ag org repos](#ag-org-repos) — List organization repositories
- [ag org view](#ag-org-view) — View an organization
- [ag pr](#ag-pr) — Manage pull requests
- [ag pr checkout](#ag-pr-checkout) — Check out a pull request locally
- [ag pr checks](#ag-pr-checks) — Show CI checks for a pull request's current head commit
- [ag pr close](#ag-pr-close) — Close a pull request
- [ag pr comment](#ag-pr-comment) — Manage pull request comments
- [ag pr comment create](#ag-pr-comment-create) — Create a comment on a pull request
- [ag pr comment delete](#ag-pr-comment-delete) — Delete a comment on a pull request
- [ag pr comment edit](#ag-pr-comment-edit) — Edit a comment on a pull request
- [ag pr comment reply](#ag-pr-comment-reply) — Reply to a comment thread on a pull request
- [ag pr comment view](#ag-pr-comment-view) — View all comments on a pull request
- [ag pr commits](#ag-pr-commits) — List commits in a pull request
- [ag pr create](#ag-pr-create) — Create a pull request
- [ag pr diff](#ag-pr-diff) — Show diff of a pull request
- [ag pr edit](#ag-pr-edit) — Edit a pull request
- [ag pr files](#ag-pr-files) — List files changed in a pull request
- [ag pr issues](#ag-pr-issues) — View linked issues of a pull request
- [ag pr link-issues](#ag-pr-link-issues) — Link issues to a pull request
- [ag pr list](#ag-pr-list) — List pull requests
- [ag pr merge](#ag-pr-merge) — Merge a pull request
- [ag pr reactions](#ag-pr-reactions) — List reactions on a pull request
- [ag pr reopen](#ag-pr-reopen) — Reopen a pull request
- [ag pr review](#ag-pr-review) — Approve a pull request review
- [ag pr unlink-issues](#ag-pr-unlink-issues) — Unlink issues from a pull request
- [ag pr view](#ag-pr-view) — View a pull request
- [ag release](#ag-release) — Manage repository releases
- [ag release create](#ag-release-create) — Create a release
- [ag release download](#ag-release-download) — Download an attachment from a release
- [ag release edit](#ag-release-edit) — Edit a release
- [ag release list](#ag-release-list) — List repository releases
- [ag release upload](#ag-release-upload) — Upload an attachment to a release
- [ag release view](#ag-release-view) — View a release by tag
- [ag repo](#ag-repo) — Manage repositories
- [ag repo clone](#ag-repo-clone) — Clone a repository
- [ag repo collaborator](#ag-repo-collaborator) — Manage repository collaborators
- [ag repo collaborator add](#ag-repo-collaborator-add) — Add a direct repository collaborator
- [ag repo collaborator edit](#ag-repo-collaborator-edit) — Update a direct repository collaborator's permission
- [ag repo collaborator list](#ag-repo-collaborator-list) — List repository collaborators
- [ag repo collaborator remove](#ag-repo-collaborator-remove) — Remove a direct repository collaborator
- [ag repo collaborator view](#ag-repo-collaborator-view) — View a repository collaborator's effective permission
- [ag repo content](#ag-repo-content) — Browse repository contents
- [ag repo content list](#ag-repo-content-list) — List a repository directory
- [ag repo content view](#ag-repo-content-view) — View a repository file
- [ag repo create](#ag-repo-create) — Create a new repository
- [ag repo delete](#ag-repo-delete) — Delete a repository
- [ag repo edit](#ag-repo-edit) — Edit repository settings
- [ag repo fork](#ag-repo-fork) — Fork a repository
- [ag repo fork list](#ag-repo-fork-list) — List forks of a repository
- [ag repo insights](#ag-repo-insights) — Inspect repository activity and statistics
- [ag repo insights contributors](#ag-repo-insights-contributors) — List repository contributor statistics
- [ag repo insights downloads](#ag-repo-insights-downloads) — Show repository download statistics
- [ag repo insights events](#ag-repo-insights-events) — List repository activity events
- [ag repo insights languages](#ag-repo-insights-languages) — Show repository language percentages
- [ag repo insights stargazers](#ag-repo-insights-stargazers) — List repository stargazers
- [ag repo insights watchers](#ag-repo-insights-watchers) — List repository watchers
- [ag repo list](#ag-repo-list) — List repositories
- [ag repo mirror](#ag-repo-mirror) — Inspect repository remote mirrors
- [ag repo mirror list](#ag-repo-mirror-list) — List configured push remote mirrors
- [ag repo mirror view](#ag-repo-mirror-view) — View repository remote mirror state
- [ag repo push-rule](#ag-repo-push-rule) — Manage repository push rules
- [ag repo push-rule edit](#ag-repo-push-rule-edit) — Edit repository push rules
- [ag repo push-rule view](#ag-repo-push-rule-view) — View repository push rules
- [ag repo read-dir](#ag-repo-read-dir) — List contents of a repository directory
- [ag repo read-file](#ag-repo-read-file) — Read a file from a repository
- [ag repo sync](#ag-repo-sync) — Synchronize a fork with its upstream repository
- [ag repo transfer](#ag-repo-transfer) — Transfer a repository to an organization
- [ag repo view](#ag-repo-view) — View a repository
- [ag repo webhook](#ag-repo-webhook) — Manage repository webhooks
- [ag repo webhook create](#ag-repo-webhook-create) — Create a repository webhook
- [ag repo webhook delete](#ag-repo-webhook-delete) — Delete a repository webhook
- [ag repo webhook edit](#ag-repo-webhook-edit) — Edit a repository webhook
- [ag repo webhook list](#ag-repo-webhook-list) — List repository webhooks
- [ag repo webhook test](#ag-repo-webhook-test) — Send a test payload to a repository webhook
- [ag repo webhook view](#ag-repo-webhook-view) — View a repository webhook
- [ag run](#ag-run) — Inspect AtomGit Actions workflow runs and artifacts
- [ag run artifact](#ag-run-artifact) — Inspect and manage workflow artifacts
- [ag run artifact delete](#ag-run-artifact-delete) — Delete a workflow artifact
- [ag run artifact view](#ag-run-artifact-view) — View artifact metadata without downloading the archive
- [ag run list](#ag-run-list) — List workflow runs
- [ag run step-log](#ag-run-step-log) — Fetch step-level logs for a workflow job
- [ag run view](#ag-run-view) — View a workflow run, jobs, logs, and artifacts
- [ag runner](#ag-runner) — Inspect AtomGit Actions host runners
- [ag runner list](#ag-runner-list) — List host runners configured for a repository
- [ag runner shared](#ag-runner-shared) — List host runners shared with a repository
- [ag search](#ag-search) — search atomgit
- [ag search issues](#ag-search-issues) — search issues
- [ag search repositories](#ag-search-repositories) — search repositories
- [ag search users](#ag-search-users) — search users
- [ag ssh-key](#ag-ssh-key) — Manage SSH keys
- [ag ssh-key add](#ag-ssh-key-add) — Add an SSH key to your AtomGit account
- [ag ssh-key delete](#ag-ssh-key-delete) — Delete an SSH key from your AtomGit account
- [ag ssh-key list](#ag-ssh-key-list) — List SSH keys registered with your AtomGit account
- [ag tag](#ag-tag) — Manage tags
- [ag tag create](#ag-tag-create) — Create a tag
- [ag tag delete](#ag-tag-delete) — Delete a tag
- [ag tag list](#ag-tag-list) — List tags
- [ag tag protection](#ag-tag-protection) — Manage protected tag rules
- [ag tag protection delete](#ag-tag-protection-delete) — Delete a protected tag rule
- [ag tag protection list](#ag-tag-protection-list) — List protected tag rules
- [ag tag protection set](#ag-tag-protection-set) — Create or update a protected tag rule
- [ag tag protection view](#ag-tag-protection-view) — View a protected tag rule
- [ag update](#ag-update) — Update AtomGit CLI to the latest stable release
- [ag user](#ag-user) — View AtomGit users, repositories, namespaces, and activity
- [ag user emails](#ag-user-emails) — List email addresses for the authenticated user
- [ag user events](#ag-user-events) — List personal activity events for a user
- [ag user namespaces](#ag-user-namespaces) — List namespaces for the authenticated user
- [ag user starred](#ag-user-starred) — List starred repositories for a user
- [ag user view](#ag-user-view) — View the current user or a public user profile
- [ag user watching](#ag-user-watching) — List watched repositories for a user
- [ag version](#ag-version) — Show version information
- [ag workflow](#ag-workflow) — Manage AtomGit Actions workflows
- [ag workflow list](#ag-workflow-list) — List workflows in a repository
- [ag workflow run](#ag-workflow-run) — Run a workflow
- [ag workflow validate](#ag-workflow-validate) — Validate a local workflow YAML file

## ag

Usage: `ag <command> <subcommand> [flags]`

AtomGit CLI

Work seamlessly with AtomGit from the command line.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | local |
| `--version` | Show version information | `false` | local |


## ag alias

Usage: `ag alias`

Create command shortcuts

Create, list, and delete command shortcuts (aliases) for "ag" commands.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag alias delete

Usage: `ag alias delete <alias>`

Delete an alias

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag alias list

Usage: `ag alias list`

List aliases

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag alias set

Usage: `ag alias set <alias> <expansion>...`

Create a shortcut for an ag command

Create a shortcut for an ag command.

Aliases are expanded at invocation time: the first non-flag argument of an ag
invocation is looked up and replaced with the expansion. Aliases never
override built-in commands, so names that conflict with a built-in command
are rejected, and the expansion must start with a known built-in command.

To include a literal space inside an expansion argument (for example a
Windows path), escape it with a backslash: C:\Program\ Files.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag alias set pl "pr list"
  ag alias set rv repo view
```


## ag api

Usage: `ag api <endpoint> [flags]`

Make an authenticated AtomGit API request

Make an authenticated request to a relative AtomGit API v5 endpoint.

GET is the default. Supported methods are GET, POST, PATCH, PUT, and DELETE.
Explicit non-GET requests may change remote resources; ag does not infer or
confirm the endpoint's effects. Redirects only retain credentials on the exact
AtomGit API origin. Paginated output is one compact JSON page per line.
Response bytes use terminal-safe output unless --raw-output is specified.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--input` | Read the raw request body from a file or - for stdin | `` | local |
| `--paginate` | Request all pages and emit compact JSON pages as NDJSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-H, --accept` | Set the Accept request header | `application/json` | local |
| `-X, --method` | HTTP method: GET, POST, PATCH, PUT, or DELETE | `GET` | local |
| `-f, --field` | Add a string field as key=value | `[]` | local |

### Example

```bash
ag api /user
  ag api /repos/owner/repo/issues --field state=open
  ag api /repos/owner/repo/issues --method POST --field title='New issue'
  ag api /repos/owner/repo/issues/42 --method PATCH --input update.json
  ag api /repos/owner/repo/issues --paginate
```


## ag auth

Usage: `ag auth <command>`

Authenticate with AtomGit

Manage authentication state for AtomGit.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag auth list

Usage: `ag auth list [flags]`

List saved AtomGit accounts

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output accounts as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag auth login

Usage: `ag auth login [flags]`

Log in with AtomGit OAuth (opens browser, saves token.json)

Opens a browser to authorize ag against atomgit.com, then writes
access_token and user to the XDG config path (see README). With --with-token,
skips the browser and reads an existing access token (PAT or OAuth token)
from standard input instead — useful in sandboxes, containers, and CI where
no browser is available:

    echo "$TOKEN" | ag auth login --with-token
    ag auth login --with-token < token.txt

Piped or redirected input is read to EOF without any prompt; in an
interactive terminal a single hidden prompt is shown. The token is validated
against the AtomGit user API before it is saved. If already logged in, skips
unless --force is set. The first saved account becomes active; later logins
do not change the active account.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--force` | Always authenticate again even if already logged in | `false` | local |
| `--git-email` | Override the Git user.email stored for this account | `` | local |
| `--git-name` | Override the Git user.name stored for this account | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--with-token` | Read an access token from standard input instead of browser OAuth | `false` | local |


## ag auth logout

Usage: `ag auth logout [flags]`

Remove the active or a selected stored account

Remove the active account or one selected with --account.
An active account can only be removed when it is the last saved account;
otherwise switch to another account first. Use --all to remove every account.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--account` | Account username to remove | `` | local |
| `--all` | Remove all saved accounts | `false` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag auth refresh

Usage: `ag auth refresh`

Refresh the access token using the stored refresh_token

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag auth status

Usage: `ag auth status`

View authentication status

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag auth switch

Usage: `ag auth switch <account> [flags]`

Switch the active account and synchronize Git identity

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--git-email` | Override Git user.email for this switch | `` | local |
| `--git-name` | Override Git user.name for this switch | `` | local |
| `--global` | Update global Git identity instead of the current repository | `false` | local |
| `--help` | Show help for command | `false` | inherited |
| `--no-git` | Do not update Git identity | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag auth token

Usage: `ag auth token`

Print the authentication token

Display the authentication token used for AtomGit API requests.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag branch

Usage: `ag branch`

Manage remote branches

List, view, create, delete, and protect AtomGit remote branches.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag branch list owner/repo
  ag branch view owner/repo main
  ag branch create owner/repo feature/foo --ref main
  ag branch delete owner/repo feature/foo
  ag branch protection list owner/repo
```


## ag branch create

Usage: `ag branch create [<owner>/<repo>] <branch> --ref <ref> [flags]`

Create a remote branch

Create a remote branch

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Source ref to create the branch from | `` | local |

### Example

```bash
ag branch create owner/repo feature/foo --ref main
```


## ag branch delete

Usage: `ag branch delete [<owner>/<repo>] <branch> [flags]`

Delete a remote branch

Delete a remote branch

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Confirm branch deletion without prompting | `false` | local |

### Example

```bash
ag branch delete owner/repo feature/foo
  ag branch delete owner/repo feature/foo --yes
```


## ag branch list

Usage: `ag branch list [<owner>/<repo>] [flags]`

List remote branches

List remote branches

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output branches as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of branches to list | `30` | local |

### Example

```bash
ag branch list owner/repo --limit 50
```


## ag branch protection

Usage: `ag branch protection`

Manage protected branch rules

List, view, create, update, and delete protected branch rules.

Rules may name an exact branch or contain an AtomGit wildcard pattern. Exact
rules take precedence over matching wildcard rules. AtomGit's API exposes only
push and merge allowlists; other web settings are not changed by this command.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag branch protection delete

Usage: `ag branch protection delete [<owner>/<repo>] <branch-or-pattern> [flags]`

Delete a protected branch rule

Delete a protected branch rule

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip deletion confirmation | `false` | local |


## ag branch protection list

Usage: `ag branch protection list [<owner>/<repo>] [flags]`

List protected branch rules

List protected branch rules

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of protected branch rules to list | `30` | local |

### Example

```bash
ag branch protection list owner/repo --limit 50
```


## ag branch protection set

Usage: `ag branch protection set [<owner>/<repo>] <branch-or-pattern> [flags]`

Create or update a protected branch rule

Create or update a protected branch rule.

Permission values are semicolon-separated role names or usernames. Supported
roles are develop, admin, and maintainer. An explicitly empty value denies the
operation to everyone. Existing rules preserve any permission whose flag is
omitted; new rules require both --push and --merge. Updating an existing rule
requires confirmation unless --yes is supplied. AtomGit requires every role or
user allowed to push to also be explicitly allowed to merge.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--merge` | Merge allowlist: develop, admin, maintainer, usernames, or empty | `` | local |
| `--push` | Push allowlist: develop, admin, maintainer, usernames, or empty | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation when updating an existing rule | `false` | local |

### Example

```bash
ag branch protection set owner/repo main --push admin --merge admin
  ag branch protection set owner/repo main --push maintainer --merge maintainer
  ag branch protection set owner/repo "release/*" --push "develop;alice" --merge "develop;alice"
  ag branch protection set owner/repo main --push "" --yes
```


## ag branch protection view

Usage: `ag branch protection view [<owner>/<repo>] <branch-or-pattern>`

View a protected branch rule

View a protected branch rule

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag branch view

Usage: `ag branch view [<owner>/<repo>] <branch>`

View a remote branch

View a remote branch

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag branch view owner/repo main
```


## ag browse

Usage: `ag browse [<number> | <path> | <commit-sha>] [flags]`

Open repositories, issues, pull requests, and more in the browser

Open repositories, issues, pull requests, and more in the browser

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-R, --repo` | Select another repository using the OWNER/REPO format | `` | local |
| `-a, --actions` | Open repository actions | `false` | local |
| `-b, --branch` | Select another branch by passing in the branch name | `` | local |
| `-c, --commit` | Select another commit by passing in the commit SHA, default is the last commit | `` | local |
| `-n, --no-browser` | Print destination URL instead of opening the browser | `false` | local |
| `-r, --releases` | Open repository releases | `false` | local |
| `-s, --settings` | Open repository settings | `false` | local |
| `-w, --wiki` | Open repository wiki | `false` | local |


## ag check-update

Usage: `ag check-update`

Check for a newer AtomGit CLI release

> Deprecated: use "ag update --check" instead

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag commit

Usage: `ag commit`

Manage commits

List, view, compare, and inspect repository commits.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag commit compare

Usage: `ag commit compare [<owner>/<repo>] <base>...<head> [flags]`

Compare two commits, branches, or tags

Compare two commits, branches, or tags

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output comparison as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag commit diff

Usage: `ag commit diff [<owner>/<repo>] <sha>`

Show a commit's diff

Show a commit's diff

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag commit list

Usage: `ag commit list [<owner>/<repo>] [flags]`

List commits

List commits

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output commits as JSON | `false` | local |
| `--path` | Only list commits that touch the given file path | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Commit SHA or branch name to start from | `` | local |
| `--since` | Only list commits after this time (RFC 3339, e.g. 2024-11-08T16:25:44Z) | `` | local |
| `--until` | Only list commits before this time (RFC 3339, e.g. 2024-11-08T16:25:44Z) | `` | local |
| `-L, --limit` | Maximum number of commits to list | `30` | local |


## ag commit patch

Usage: `ag commit patch [<owner>/<repo>] <sha>`

Show a commit's patch

Show a commit's patch

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag commit view

Usage: `ag commit view [<owner>/<repo>] <sha> [flags]`

View a commit

View a commit

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output commit as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-w, --web` | Open a commit in the browser | `false` | local |


## ag discussion

Usage: `ag discussion`

View repository discussions

List AtomGit discussions for a repository

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag discussion list

Usage: `ag discussion list [<owner>/<repo>] [flags]`

List repository discussions

List repository discussions

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output discussions as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of discussions to list | `30` | local |


## ag discussion view

Usage: `ag discussion view [<owner>/<repo>] <number> [flags]`

View a repository discussion

View a single repository discussion with its Markdown body.

With --comments the comment thread is fetched as well; comments that have
replies include their nested replies in server order. Hidden, deleted, and
empty bodies are shown as explicit placeholders instead of blank text.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--comments` | Also fetch and show the comment thread | `false` | local |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output the discussion as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag discussion view owner/repo 1
  ag discussion view owner/repo 1 --comments
  ag discussion view owner/repo 1 --comments --json
```


## ag issue

Usage: `ag issue`

Manage issues

Create, view, and manage issues.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag issue branches

Usage: `ag issue branches [<owner>/<repo>] <number> [flags]`

List or update related branches for an issue

List or update related branches for an issue.

With neither --add nor --remove, the command lists current related branch names.

Concurrent branch-name updates from other clients can be overwritten by this
command because the AtomGit API uses whole-list replacement. Review the current
list before mutating branches that other tools may also manage.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--add` | Branch names to add (repeatable) | `[]` | local |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output branch names as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--remove` | Branch names to remove (repeatable) | `[]` | local |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag issue branches owner/repo 42
  ag issue branches owner/repo 42 --json
  ag issue branches owner/repo 42 --add feature/x
  ag issue branches owner/repo 42 --remove main --yes
```


## ag issue close

Usage: `ag issue close [<owner>/<repo>] <number>`

Close an issue

Close an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag issue comment

Usage: `ag issue comment`

Manage issue comments

Create, view, edit, and delete issue comments.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag issue comment create

Usage: `ag issue comment create [<owner>/<repo>] <number> [flags]`

Create a comment on an issue

Create a comment on an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --body-file` | Read body text from file | `` | local |
| `-b, --body` | Comment body text | `` | local |


## ag issue comment delete

Usage: `ag issue comment delete [<owner>/<repo>] <number> <comment-id> [flags]`

Delete a comment on an issue

Delete a comment on an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |


## ag issue comment edit

Usage: `ag issue comment edit [<owner>/<repo>] <number> <comment-id> [flags]`

Edit a comment on an issue

Edit a comment on an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --body` | New comment body text | `` | local |


## ag issue comment view

Usage: `ag issue comment view [<owner>/<repo>] <number>`

View all comments on an issue

View all comments on an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag issue create

Usage: `ag issue create [<owner>/<repo>] [flags]`

Create an issue

Create an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--assignee` | Assign the issue to a user (login) | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --body-file` | Read issue body from file (use - for stdin) | `` | local |
| `-b, --body` | Issue body | `` | local |
| `-t, --title` | Issue title | `` | local |

### Example

```bash
ag issue create owner/repo --title "Bug report" --body "Description"
  ag issue create owner/repo --title "Bug report" --assignee alice
  ag issue create owner/repo --title "Bug report" --body-file description.md
  ag issue create owner/repo --title "Bug report" --body-file -
```


## ag issue edit

Usage: `ag issue edit [<owner>/<repo>] <number> [flags]`

Edit an issue

Edit an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--assignee` | Set the issue assignee (login) | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--remove-assignee` | Clear the issue assignee | `false` | local |
| `-F, --body-file` | Read the new issue body from a file | `` | local |
| `-b, --body` | New issue body | `` | local |
| `-t, --title` | New issue title | `` | local |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag issue edit owner/repo 42 --title "new title" --body "new body"
  ag issue edit owner/repo 42 --assignee alice
  ag issue edit owner/repo 42 --remove-assignee --yes
  ag issue edit owner/repo 42 --body-file description.md
```


## ag issue label

Usage: `ag issue label [<owner>/<repo>] <number> [<labels>] [flags]`

Add or remove labels on an issue

Add or remove labels on an issue.

Labels are comma-separated. Positional labels are treated as labels to add for
backward compatibility. Use --add or --remove to make the operation explicit.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--add` | Comma-separated labels to add | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--remove` | Comma-separated labels to remove | `` | local |

### Example

```bash
ag issue label owner/repo 42 "bug, help wanted"
  ag issue label owner/repo 42 --add "bug, help wanted"
  ag issue label owner/repo 42 --remove "priority/high"
```


## ag issue list

Usage: `ag issue list [<owner>/<repo>] [flags]`

List issues

List issues

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--assignee` | Filter by assignee: @me for issues assigned to you across all your repositories | `` | local |
| `--author` | Filter by author: @me for issues you created across all your repositories | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--involved` | Filter by involvement: @me for issues you created or are assigned to across all your repositories | `` | local |
| `--json` | Output issues as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of issues to list | `30` | local |
| `-s, --state` | Filter by state: open, closed, all | `open` | local |


## ag issue prs

Usage: `ag issue prs [<owner>/<repo>] <number> [flags]`

List pull requests linked to an issue

List pull requests linked to an issue.

Concurrent updates to linked pull requests are not reflected until the next
request.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output linked pull requests as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag issue prs owner/repo 42
  ag issue prs owner/repo 42 --json
```


## ag issue reopen

Usage: `ag issue reopen [<owner>/<repo>] <number>`

Reopen an issue

Reopen an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag issue view

Usage: `ag issue view [<owner>/<repo>] <number> [flags]`

View an issue

View an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output issue as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-w, --web` | Open an issue in the browser | `false` | local |


## ag kanban

Usage: `ag kanban`

View organization Kanban boards

List and inspect read-only organization Kanban boards and their Issue/Pull Request items.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag kanban list hust-open-atom-club
  ag kanban view hust-open-atom-club 1234567890
  ag kanban items hust-open-atom-club 1234567890 --json
```


## ag kanban items

Usage: `ag kanban items <owner> <kanban-id> [flags]`

List items on a Kanban board

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output Kanban items as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of Kanban items to list | `30` | local |

### Example

```bash
ag kanban items hust-open-atom-club 1234567890
  ag kanban items hust-open-atom-club 1234567890 --limit 50 --json
```


## ag kanban list

Usage: `ag kanban list <owner> [flags]`

List organization Kanban boards

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output Kanban boards as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of Kanban boards to list | `30` | local |

### Example

```bash
ag kanban list hust-open-atom-club
  ag kanban list hust-open-atom-club --limit 50 --json
```


## ag kanban view

Usage: `ag kanban view <owner> <kanban-id> [flags]`

View a Kanban board

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output the Kanban board as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag kanban view hust-open-atom-club 1234567890 --json
```


## ag label

Usage: `ag label`

Manage repository labels

List and manage repository labels.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag label create

Usage: `ag label create [<owner>/<repo>] [flags]`

Create a repository label

Create a repository label. AtomGit API v5 accepts a name and color for label creation.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--color` | Label color in #RGB or #RRGGBB format | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--name` | Label name | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag label create owner/repo --name bug --color "#ff0000"
```


## ag label delete

Usage: `ag label delete [<owner>/<repo>] <name> [flags]`

Delete a repository label

Delete a repository label from AtomGit.

By default, you will be prompted to confirm the deletion. Use --yes to skip
the confirmation prompt.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag label delete owner/repo obsolete
  ag label delete owner/repo obsolete --yes
```


## ag label edit

Usage: `ag label edit [<owner>/<repo>] <name> [flags]`

Edit a repository label

Edit a repository label. AtomGit API v5 accepts a new name and color for label updates.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--color` | New label color in #RGB or #RRGGBB format | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--name` | New label name | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag label edit owner/repo bug --name defect --color "#d73a4a"
```


## ag label list

Usage: `ag label list [<owner>/<repo>] [flags]`

List repository labels

List repository labels

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output labels as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of labels to list | `30` | local |

### Example

```bash
ag label list owner/repo --limit 50
```


## ag license

Usage: `ag license`

License compliance checking

Check license compliance using openEuler compliance service.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag license check

Usage: `ag license check <license>`

Check license compliance

Check if a license is compliant using openEuler compliance service.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag milestone

Usage: `ag milestone`

Manage repository milestones

List, view, create, update, close, reopen, and delete repository milestones.

AtomGit requires title and due_on on every milestone PATCH. Edit, close, and
reopen read the current milestone first and preserve those required fields.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag milestone close

Usage: `ag milestone close [<owner>/<repo>] <number>`

Close a repository milestone

Close a repository milestone

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag milestone create

Usage: `ag milestone create [<owner>/<repo>] [flags]`

Create a repository milestone

Create a repository milestone

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--due-on` | Due date in YYYY-MM-DD format | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-d, --description` | Milestone description | `` | local |
| `-t, --title` | Milestone title | `` | local |


## ag milestone delete

Usage: `ag milestone delete [<owner>/<repo>] <number> [flags]`

Delete a repository milestone

Permanently delete a milestone. This differs from closing it and requires confirmation unless --yes is used.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |


## ag milestone edit

Usage: `ag milestone edit [<owner>/<repo>] <number> [flags]`

Edit a repository milestone

Edit a repository milestone

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--due-on` | New due date in YYYY-MM-DD format | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-d, --description` | New milestone description; pass an empty value to clear | `` | local |
| `-t, --title` | New milestone title | `` | local |


## ag milestone list

Usage: `ag milestone list [<owner>/<repo>] [flags]`

List repository milestones

List repository milestones

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--direction` | Sort direction: asc, desc | `asc` | local |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output milestones as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--sort` | Sort milestones by created or due_on | `due_on` | local |
| `-L, --limit` | Maximum number of milestones to list | `30` | local |
| `-s, --state` | Filter by state: open, closed, all | `open` | local |

### Example

```bash
ag milestone list owner/repo --state all --limit 50
```


## ag milestone reopen

Usage: `ag milestone reopen [<owner>/<repo>] <number>`

Reopen a repository milestone

Reopen a repository milestone

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag milestone view

Usage: `ag milestone view [<owner>/<repo>] <number> [flags]`

View a repository milestone

View a repository milestone

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output milestone as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag notification

Usage: `ag notification`

Manage repository notifications

List repository notifications and mark them as read.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag notification list

Usage: `ag notification list [<owner>/<repo>] [flags]`

List repository notifications

List notifications for a repository, most recent first.

By default all notifications are listed; --unread keeps only unread ones.
--since and --before accept RFC 3339 timestamps (for example
2026-08-14T00:00:00+08:00). --type keeps only notifications of one type,
such as merge_requests_open or issue_open.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--before` | Only list notifications updated before this RFC 3339 timestamp | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output notifications as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--since` | Only list notifications updated at or after this RFC 3339 timestamp | `` | local |
| `--type` | Only list notifications of this type (for example merge_requests_open) | `` | local |
| `--unread` | Only list unread notifications | `false` | local |
| `-L, --limit` | Maximum number of notifications to list | `30` | local |

### Example

```bash
ag notification list owner/repo --limit 20
  ag notification list --unread --json
  ag notification list owner/repo --type issue_open --since 2026-08-01T00:00:00Z
```


## ag notification mark-read

Usage: `ag notification mark-read [<owner>/<repo>] [<notification-id>...] [flags]`

Mark repository notifications as read

Mark notifications for a repository as read.

Pass one or more notification IDs (as shown by "ag notification list") to
mark exactly those notifications, or pass --all to mark every unread
notification in the repository. --all asks for confirmation unless --yes is
supplied.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--all` | Mark every unread notification in the repository | `false` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip the --all confirmation prompt | `false` | local |

### Example

```bash
ag notification mark-read owner/repo 292ecbec857e4f27b426d66f2157938c
  ag notification mark-read --all --yes
```


## ag org

Usage: `ag org`

Manage organizations

List organizations associated with your AtomGit account and inspect organization details, members, and repositories.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag org list

Usage: `ag org list [flags]`

List organizations for the authenticated user

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output organizations as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of organizations to list | `30` | local |

### Example

```bash
ag org list
  ag org list --limit 100
  ag org list --json
```


## ag org members

Usage: `ag org members <org> [flags]`

List organization members

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output members as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of members to list | `30` | local |

### Example

```bash
ag org members my-organization
  ag org members my-organization --limit 100
  ag org members my-organization --json
```


## ag org repos

Usage: `ag org repos <org> [flags]`

List organization repositories

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output repositories as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of repositories to list | `30` | local |

### Example

```bash
ag org repos my-organization
  ag org repos my-organization --limit 100
  ag org repos my-organization --json
```


## ag org view

Usage: `ag org view <org> [flags]`

View an organization

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output organization as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag org view my-organization
  ag org view my-organization --json
```


## ag pr

Usage: `ag pr`

Manage pull requests

Create, view, and checkout pull requests.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag pr checkout

Usage: `ag pr checkout [<owner>/<repo>] <number> [flags]`

Check out a pull request locally

Check out the source branch of a pull request into a local branch.

When run inside a git repository, the owner and repository are inferred from
the current git remote. You can also specify them explicitly.

For same-repository PRs, the source branch is fetched from the matching remote.
For fork PRs, a temporary remote is added to fetch the source branch.

By default, the command will not overwrite uncommitted changes or existing local
branches. Use --force to override these safety checks.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--detach` | Check out PR in detached HEAD mode | `false` | local |
| `--force` | Force checkout, bypassing safety checks for dirty tree and branch conflicts | `false` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--recurse-submodules` | Update submodules after checkout | `false` | local |
| `-b, --branch` | Local branch name (default: PR head branch name) | `` | local |

### Example

```bash
# Check out PR #42, inferring the repository from git remote
  ag pr checkout 42

  # Check out PR #42 from a specific repository
  ag pr checkout owner/repo 42

  # Check out to a custom branch name
  ag pr checkout 42 --branch review-fix

  # Force checkout, discarding safety checks
  ag pr checkout 42 --force

  # Check out in detached HEAD mode and update submodules
  ag pr checkout 42 --detach --recurse-submodules
```


## ag pr checks

Usage: `ag pr checks [<owner>/<repo>] <number> [flags]`

Show CI checks for a pull request's current head commit

Show AtomGit Actions runs for a pull request's current head commit. This command does not infer or report required-check semantics.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-i, --interval` | Polling interval when using --watch | `10s` | local |
| `-w, --watch` | Watch checks until they reach a terminal state | `false` | local |

### Example

```bash
ag pr checks owner/repo 42
  ag pr checks 42 --watch
  ag pr checks owner/repo 42 --watch --interval 5s
```


## ag pr close

Usage: `ag pr close [<owner>/<repo>] <number>`

Close a pull request

Close a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag pr comment

Usage: `ag pr comment`

Manage pull request comments

Create, view, edit, delete, and reply to pull request comments.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag pr comment create

Usage: `ag pr comment create [<owner>/<repo>] <number> [flags]`

Create a comment on a pull request

Create a comment on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --body-file` | Read body text from file | `` | local |
| `-b, --body` | Comment body text | `` | local |


## ag pr comment delete

Usage: `ag pr comment delete [<owner>/<repo>] <number> <comment-id> [flags]`

Delete a comment on a pull request

Delete a comment on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |


## ag pr comment edit

Usage: `ag pr comment edit [<owner>/<repo>] <number> <comment-id> [flags]`

Edit a comment on a pull request

Edit a comment on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --body` | New comment body text | `` | local |


## ag pr comment reply

Usage: `ag pr comment reply [<owner>/<repo>] <number> <discussion-id> [flags]`

Reply to a comment thread on a pull request

Reply to a comment thread on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --body` | Reply body text | `` | local |


## ag pr comment view

Usage: `ag pr comment view [<owner>/<repo>] <number>`

View all comments on a pull request

View all comments on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag pr commits

Usage: `ag pr commits [<owner>/<repo>] <number> [flags]`

List commits in a pull request

List commits in a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output commits as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of commits to list | `30` | local |


## ag pr create

Usage: `ag pr create [<owner>/<repo>] [flags]`

Create a pull request

Create a pull request and optionally set its collaboration metadata.

Assignees own follow-up work, approval reviewers approve the change, and
testers verify it. These AtomGit roles are managed independently. Labels and
milestones must already exist in the repository.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--assignee` | Assignee login (repeat for multiple users) | `[]` | local |
| `--base` | Base branch (defaults to repository default) | `` | local |
| `--head` | Head branch | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--label` | Label name (repeat for multiple labels) | `[]` | local |
| `--milestone` | Milestone number or exact title | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--reviewer` | Approval reviewer login (repeat for multiple users) | `[]` | local |
| `--tester` | Tester login (repeat for multiple users) | `[]` | local |
| `-F, --body-file` | Read PR body from file (use - for stdin) | `` | local |
| `-b, --body` | PR body | `` | local |
| `-t, --title` | PR title | `` | local |

### Example

```bash
ag pr create owner/repo --title "Fix bug" --body "Description" --base main --head feature
  ag pr create owner/repo --title "Fix bug" --body-file description.md --base main --head feature
  ag pr create owner/repo --title "Fix bug" --body-file - --base main --head feature
```


## ag pr diff

Usage: `ag pr diff [<owner>/<repo>] <number>`

Show diff of a pull request

Show diff of a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag pr edit

Usage: `ag pr edit [<owner>/<repo>] <number> [flags]`

Edit a pull request

Edit a pull request and explicitly add or remove collaboration metadata.

Assignees, approval reviewers, and testers are distinct AtomGit roles.
Unspecified metadata is left unchanged; use --milestone none to clear the
current milestone.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--add-assignee` | Assignee login to add (repeat for multiple users) | `[]` | local |
| `--add-label` | Label name to add (repeat for multiple labels) | `[]` | local |
| `--add-reviewer` | Approval reviewer login to add (repeat for multiple users) | `[]` | local |
| `--add-tester` | Tester login to add (repeat for multiple users) | `[]` | local |
| `--help` | Show help for command | `false` | inherited |
| `--milestone` | Milestone number, exact title, or 'none' to clear | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--remove-assignee` | Assignee login to remove (repeat for multiple users) | `[]` | local |
| `--remove-label` | Label name to remove (repeat for multiple labels) | `[]` | local |
| `--remove-reviewer` | Approval reviewer login to remove (repeat for multiple users) | `[]` | local |
| `--remove-tester` | Tester login to remove (repeat for multiple users) | `[]` | local |
| `-b, --body` | New PR body | `` | local |
| `-t, --title` | New PR title | `` | local |


## ag pr files

Usage: `ag pr files [<owner>/<repo>] <number> [flags]`

List files changed in a pull request

List files changed in a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output files as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag pr issues

Usage: `ag pr issues [<owner>/<repo>] <pr_number>`

View linked issues of a pull request

View all issues linked to a pull request.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag pr link-issues

Usage: `ag pr link-issues [<owner>/<repo>] <pr_number> [flags]`

Link issues to a pull request

Link one or more issues to a pull request.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-i, --issue` | Issue number to link (can be specified multiple times) | `[]` | local |


## ag pr list

Usage: `ag pr list [<owner>/<repo>] [flags]`

List pull requests

List pull requests

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--assignee` | Filter by assignee: @me for PRs assigned to you across all your repositories | `` | local |
| `--author` | Filter by author: @me for PRs you created across all your repositories | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output pull requests as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--review-needed` | Filter by requested reviewer: @me for PRs that need your review across all your repositories | `` | local |
| `--review-requested` | Filter by requested approver: @me for PRs that need your approval across all your repositories | `` | local |
| `-L, --limit` | Maximum number of PRs to list | `30` | local |
| `-s, --state` | Filter by state: open, closed, all | `open` | local |


## ag pr merge

Usage: `ag pr merge [<owner>/<repo>] <number> [flags]`

Merge a pull request

Merge a pull request.

By default, ag creates a merge commit. Use --rebase to rebase the commits onto the base branch.


When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--admin` | Use administrator privileges to merge a pull request that does not meet requirements | `false` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --body` | Body text for the merge commit | `` | local |
| `-d, --delete-branch` | Delete the source branch after merge | `false` | local |
| `-r, --rebase` | Rebase the commits onto the base branch | `false` | local |
| `-s, --squash` | Squash the commits into one commit | `false` | local |
| `-t, --subject` | Subject text for the merge commit | `` | local |


## ag pr reactions

Usage: `ag pr reactions [<owner>/<repo>] <number> [flags]`

List reactions on a pull request

List reactions on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output reactions as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag pr reopen

Usage: `ag pr reopen [<owner>/<repo>] <number>`

Reopen a pull request

Reopen a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag pr review

Usage: `ag pr review <owner>/<repo> <number> [flags]`

Approve a pull request review

Approve a pull request using AtomGit's formal review API.

AtomGit currently exposes approval as the only review action. Use ag pr comment
create to leave an ordinary comment; request-changes reviews are not supported
by the public API.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--approve` | Approve the pull request | `false` | local |
| `--force` | Force approval as a repository administrator | `false` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag pr review owner/repo 42 --approve
  ag pr review owner/repo 42 --approve --force
```


## ag pr unlink-issues

Usage: `ag pr unlink-issues [<owner>/<repo>] <pr_number> [flags]`

Unlink issues from a pull request

Unlink one or more issues from a pull request.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-i, --issue` | Issue number to unlink (can be specified multiple times) | `[]` | local |


## ag pr view

Usage: `ag pr view [<owner>/<repo>] <number> [flags]`

View a pull request

View a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output pull request as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-w, --web` | Open a pull request in the browser | `false` | local |


## ag release

Usage: `ag release`

Manage repository releases

List, view, create, and edit repository releases on AtomGit, and upload or download release attachments.

The repository's automated release pipeline validates tags and artifacts, then
uses these primitives through "make publish VERSION=vX.Y.Z NOTES_FILE=notes.md".

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag release list owner/repo
  ag release view owner/repo v1.0.0
  ag release create owner/repo v1.0.0 --name "Version 1.0.0" --body "Release notes"
  ag release upload owner/repo v1.0.0 ./dist/app.tar.gz
```


## ag release create

Usage: `ag release create [<owner>/<repo>] <tag> [flags]`

Create a release

Create a release for a repository identified by its tag. A non-empty release body is required by the AtomGit API.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--prerelease` | Mark release as a prerelease (release_status=pre) | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--target` | Target commitish (branch or SHA) | `` | local |
| `-F, --body-file` | Path to file containing release body | `` | local |
| `-b, --body` | Release body text | `` | local |
| `-n, --name` | Release name (defaults to tag) | `` | local |

### Example

```bash
ag release create owner/repo v1.0.0 --name "First" --body "Initial release"
  ag release create owner/repo v1.0.0-rc --prerelease --body-file notes.md
```


## ag release download

Usage: `ag release download [<owner>/<repo>] <tag> <asset> [flags]`

Download an attachment from a release

Download a release attachment identified by its exact asset name to a local file.

The required -o/--output flag names the local destination path. When the
destination already exists the command fails unless --overwrite is given, in
which case the existing file is replaced. The attachment is streamed to a
temporary file in the destination directory and only installed at the target
path after the transfer completes; a transfer interruption leaves any existing
destination untouched.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--overwrite` | Replace an existing local file at --output | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--timeout` | Maximum attachment transfer time (0 disables the limit) | `30m0s` | local |
| `-o, --output` | Local file path to write the attachment to (required) | `` | local |

### Example

```bash
ag release download owner/repo v1.0.0 app.tar.gz -o ./dist/app.tar.gz
  ag release download owner/repo v1.0.0 app.tar.gz --output ./existing.tar.gz --overwrite
```


## ag release edit

Usage: `ag release edit [<owner>/<repo>] <tag> [flags]`

Edit a release

Edit an existing release identified by its tag.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--latest` | Set release status to latest (release_status=latest) | `false` | local |
| `--prerelease` | Set release status to prerelease (release_status=pre) | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --body-file` | Path to file containing new release body | `` | local |
| `-b, --body` | New release body text | `` | local |
| `-n, --name` | New release name | `` | local |

### Example

```bash
ag release edit owner/repo v1.0.0 --name "First Release"
  ag release edit owner/repo v1.0.0 --latest --body-file notes.md
  ag release edit owner/repo v1.0.0-rc --prerelease
```


## ag release list

Usage: `ag release list [<owner>/<repo>] [flags]`

List repository releases

List releases for a repository, ordered most recent first.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output releases as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of releases to list | `30` | local |

### Example

```bash
ag release list owner/repo --limit 50
```


## ag release upload

Usage: `ag release upload [<owner>/<repo>] <tag> <file> [flags]`

Upload an attachment to a release

Upload a local file as an attachment to an existing release identified by its tag.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--overwrite` | Delete an existing attachment with the same name before uploading | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--skip-existing` | Do nothing and report success if an attachment with the same name already exists | `false` | local |
| `--timeout` | Maximum attachment transfer time (0 disables the limit) | `30m0s` | local |
| `-n, --name` | Remote attachment name (defaults to the local file's base name) | `` | local |

### Example

```bash
ag release upload owner/repo v1.0.0 ./dist/app.tar.gz
  ag release upload owner/repo v1.0.0 ./build/app.zip --name app-v1.zip
  ag release upload owner/repo v1.0.0 ./new.tar.gz --overwrite
  ag release upload owner/repo v1.0.0 ./existing.tar.gz --skip-existing
```


## ag release view

Usage: `ag release view [<owner>/<repo>] <tag>`

View a release by tag

Show details of a single release identified by its tag.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag release view owner/repo v1.0.0
```


## ag repo

Usage: `ag repo`

Manage repositories

Create, clone, edit, fork, sync, transfer, view, browse contents, and manage repository collaborators and webhooks. Use `ag repo fork list` to inspect existing forks; `ag repo fork` creates a fork.

For repository-scoped commands, OWNER/REPO may be omitted and inferred from the current Git repository.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag repo clone

Usage: `ag repo clone <repository> [<directory>] [flags]`

Clone a repository

Clone a repository from AtomGit.

The repository argument can be:
- Full URL: https://atomgit.com/owner/repo
- Owner/repo format: owner/repo
- Just repo name (uses current user as owner)

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --branch` | Clone specific branch | `` | local |

### Example

```bash
# Clone using full URL
  ag repo clone https://atomgit.com/shinwell_hu/my-project

  # Clone using owner/repo format
  ag repo clone shinwell_hu/my-project

  # Clone to specific directory
  ag repo clone shinwell_hu/my-project my-project-local

  # Clone specific branch
  ag repo clone shinwell_hu/my-project --branch develop
```


## ag repo collaborator

Usage: `ag repo collaborator`

Manage repository collaborators

List, inspect, add, edit, and remove direct repository collaborators.

The built-in AtomGit permissions are pull, push, and admin. Organization-
inherited permissions are displayed but cannot be changed by these commands.

AtomGit API v5 exposes accepted collaborators only. Pending invitations are
not shown by list or view; remove attempts to revoke a possible pending
invitation when the user is not an accepted collaborator. In text mode, list
writes this API limitation as a note to stderr; JSON output contains only the
accepted-collaborator data returned by the API.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag repo collaborator add

Usage: `ag repo collaborator add [<owner>/<repo>] <username> [flags]`

Add a direct repository collaborator

Add a direct repository collaborator

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-p, --permission` | Permission: pull, push, or admin | `push` | local |

### Example

```bash
ag repo collaborator add owner/repo octocat --permission push
```


## ag repo collaborator edit

Usage: `ag repo collaborator edit [<owner>/<repo>] <username> [flags]`

Update a direct repository collaborator's permission

Update a direct repository collaborator's permission

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-p, --permission` | Permission: pull, push, or admin | `` | local |
| `-y, --yes` | Skip confirmation for permission reductions | `false` | local |

### Example

```bash
ag repo collaborator edit owner/repo octocat --permission pull
```


## ag repo collaborator list

Usage: `ag repo collaborator list [<owner>/<repo>] [flags]`

List repository collaborators

List accepted repository collaborators. Pending invitations are not exposed by AtomGit API v5 and cannot be included in this listing.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output collaborators as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of collaborators to list | `30` | local |

### Example

```bash
ag repo collaborator list owner/repo --limit 50
```


## ag repo collaborator remove

Usage: `ag repo collaborator remove [<owner>/<repo>] <username> [flags]`

Remove a direct repository collaborator

Remove an accepted direct collaborator or attempt to revoke a pending invitation.

Because AtomGit API v5 does not expose pending invitations, this command first
checks accepted collaborators. If the user is not found, it sends the delete
request as a possible invitation revocation. JSON output is not available for
this mutation.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag repo collaborator remove owner/repo octocat --yes
```


## ag repo collaborator view

Usage: `ag repo collaborator view [<owner>/<repo>] <username> [flags]`

View a repository collaborator's effective permission

View a repository collaborator's effective permission

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output collaborator as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag repo collaborator view owner/repo octocat
```


## ag repo content

Usage: `ag repo content`

Browse repository contents

List directories and view files without modifying repository contents.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag repo content list

Usage: `ag repo content list [<owner>/<repo>] [<path>] [flags]`

List a repository directory

List a repository directory without modifying it. With no arguments, the repository is inferred and its root is listed. A single argument is a path in the inferred repository; use OWNER/REPO . to list the root of an explicit repository.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output the complete API directory array as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Branch, tag, or commit identifier | `` | local |

### Example

```bash
ag repo content list
  ag repo content list docs
  ag repo content list owner/repo .
  ag repo content list owner/repo docs/guides --ref v1.0.0 --json
```


## ag repo content view

Usage: `ag repo content view [<owner>/<repo>] <path> [flags]`

View a repository file

View a repository file

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output the complete API file object as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Branch, tag, or commit identifier | `` | local |

### Example

```bash
ag repo content view README.md
  ag repo content view owner/repo src/main.go --ref dev
  ag repo content view owner/repo README.md --json
```


## ag repo create

Usage: `ag repo create <repo> | <owner>/<repo> [flags]`

Create a new repository

Create a new repository on AtomGit.

Use repo to create the repository under your current account, or owner/repo to
create it under an explicitly specified user or organization namespace.
Pass --public to make the repository public, or --private for private.
If neither is specified, it defaults to private.

Pass --clone to clone the repository locally after creation.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--private` | Make the repository private | `false` | local |
| `--public` | Make the repository public | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-c, --clone` | Clone the repository after creation | `false` | local |
| `-d, --description` | Description of the repository | `` | local |

### Example

```bash
# Create a new private repository under your account
  ag repo create my-project

  # Create a public repository and clone it
  ag repo create my-project --public --clone

  # Create a repository in an organization
  ag repo create my-org/my-project --public
```


## ag repo delete

Usage: `ag repo delete [<repository>] [flags]`

Delete a repository

Delete a repository from AtomGit.

This command permanently deletes a repository. This action cannot be undone.

By default, you will be prompted to confirm the deletion. Use --yes to skip
the confirmation prompt.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
# Delete a repository (with confirmation)
  ag repo delete my-project

  # Delete a repository without confirmation
  ag repo delete my-project --yes

  # Delete a repository in an organization
  ag repo delete my-org/my-project --yes
```


## ag repo edit

Usage: `ag repo edit [<owner>/<repo>] [flags]`

Edit repository settings

Edit supported repository metadata and visibility on AtomGit.

Only flags explicitly provided are sent to AtomGit; omitted settings remain
unchanged. Name and visibility updates require confirmation unless --yes is
used. --visibility, --public, and --private are mutually exclusive.

This command does not change the repository path, owner, homepage, LFS state,
module switches, merge policies, or other unsupported GitHub CLI settings.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--default-branch` | New default branch | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--name` | New repository name (does not change the repository path) | `` | local |
| `--private` | Make the repository private | `false` | local |
| `--public` | Make the repository public | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--visibility` | New visibility: public or private | `` | local |
| `-d, --description` | New repository description | `` | local |
| `-y, --yes` | Skip confirmation for name or visibility changes | `false` | local |

### Example

```bash
# Update the current Git repository
  ag repo edit --description "New description"

  # Update an explicitly selected repository
  ag repo edit owner/repo --description "New description"

  # Clear a description without changing other settings
  ag repo edit owner/repo --description ""

  # Update several settings
  ag repo edit owner/repo --name "New name" --default-branch main --visibility private

  # Skip confirmation for a visibility update
  ag repo edit owner/repo --public --yes
```


## ag repo fork

Usage: `ag repo fork [<owner>/<repo>] [flags]`

Fork a repository

Fork a repository on AtomGit.

Creates a fork of the specified repository under your account or an organization.

By default, the fork will have the same visibility as the original repository.
Use --private or --public to override.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--private` | Make the forked repository private | `false` | local |
| `--public` | Make the forked repository public | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-c, --clone` | Clone the forked repository | `false` | local |
| `-d, --description` | Description for the forked repository | `` | local |
| `-n, --name` | Name for the forked repository | `` | local |


## ag repo fork list

Usage: `ag repo fork list [<owner>/<repo>] [flags]`

List forks of a repository

List existing forks of a repository.

This is read-only and is separate from `ag repo fork`, which creates a new fork.
The repository can be supplied explicitly or inferred from the current Git repository.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output forks as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of forks to list | `30` | local |

### Example

```bash
ag repo fork list owner/repo
  ag repo fork list owner/repo --limit 100 --json
  ag repo fork list
```


## ag repo insights

Usage: `ag repo insights`

Inspect repository activity and statistics

Inspect read-only repository languages, contributor statistics, events, watchers, stargazers, and download statistics.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag repo insights contributors

Usage: `ag repo insights contributors [<owner>/<repo>] [flags]`

List repository contributor statistics

List repository contributor statistics

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output contributors as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of contributors to list | `30` | local |


## ag repo insights downloads

Usage: `ag repo insights downloads [<owner>/<repo>] [flags]`

Show repository download statistics

Show repository download statistics

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output download statistics as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag repo insights events

Usage: `ag repo insights events [<owner>/<repo>] [flags]`

List repository activity events

List repository activity events

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output events as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of events to list | `30` | local |


## ag repo insights languages

Usage: `ag repo insights languages [<owner>/<repo>] [flags]`

Show repository language percentages

Show repository language percentages

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output languages as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag repo insights stargazers

Usage: `ag repo insights stargazers [<owner>/<repo>] [flags]`

List repository stargazers

List repository stargazers

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output stargazers as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of stargazers to list | `30` | local |


## ag repo insights watchers

Usage: `ag repo insights watchers [<owner>/<repo>] [flags]`

List repository watchers

List repository watchers

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output watchers as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of watchers to list | `30` | local |


## ag repo list

Usage: `ag repo list [<owner>] [flags]`

List repositories

List repositories for the authenticated user, a specified user, or an organization. A specified owner is checked as a user first and retried as an organization only when the user is not found.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output repositories as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of repositories to list | `30` | local |

### Example

```bash
ag repo list
  ag repo list alice
  ag repo list my-organization --limit 100
  ag repo list alice --json
```


## ag repo mirror

Usage: `ag repo mirror`

Inspect repository remote mirrors

Inspect configured push mirrors and repository mirror state. These commands are read-only and do not synchronize local Git remotes.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag repo mirror list

Usage: `ag repo mirror list [<owner>/<repo>] [flags]`

List configured push remote mirrors

List configured push remote mirrors. Destinations and returned messages are sanitized before text or JSON output.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output push remote mirrors as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of push remote mirrors to list | `30` | local |

### Example

```bash
ag repo mirror list owner/repo
  ag repo mirror list owner/repo --limit 100 --json
  ag repo mirror list
```


## ag repo mirror view

Usage: `ag repo mirror view [<owner>/<repo>] [flags]`

View repository remote mirror state

View repository remote mirror state. Destinations and returned errors are sanitized before text or JSON output.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output repository remote mirror state as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag repo mirror view owner/repo
  ag repo mirror view --json
```


## ag repo push-rule

Usage: `ag repo push-rule`

Manage repository push rules

View and edit repository-wide push rules.

These rules cover signed commits, commit message validation, maximum file
size, owner exemptions, and force pushes. Branch and tag protection rules are
managed separately.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag repo push-rule edit

Usage: `ag repo push-rule edit [<owner>/<repo>] [flags]`

Edit repository push rules

Edit repository-wide push rules.

Only flags explicitly provided are sent to AtomGit; omitted settings remain
unchanged. Explicit false, empty-string, and zero values are preserved. All
updates require confirmation unless --yes is supplied.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--commit-message-regex` | Regular expression required for commit messages (empty disables it) | `` | local |
| `--deny-force-push` | Deny force pushes, including from administrators | `false` | local |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output the updated fields as JSON | `false` | local |
| `--max-file-size` | Maximum committed file size in MB (0 disables the limit) | `0` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--reject-not-signed-by-gpg` | Require commits to have verified GPG signatures | `false` | local |
| `--skip-rule-for-owner` | Exempt repository administrators from applicable push rules | `false` | local |
| `-y, --yes` | Skip update confirmation | `false` | local |

### Example

```bash
ag repo push-rule edit owner/repo --deny-force-push --yes
  ag repo push-rule edit --commit-message-regex '^(feat|fix): '
  ag repo push-rule edit owner/repo --reject-not-signed-by-gpg=false --max-file-size 0
```


## ag repo push-rule view

Usage: `ag repo push-rule view [<owner>/<repo>] [flags]`

View repository push rules

View repository push rules

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output push rules as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag repo push-rule view owner/repo
  ag repo push-rule view --json
```


## ag repo read-dir

Usage: `ag repo read-dir [<owner>/<repo>] <path> [flags]`

List contents of a repository directory

> Deprecated: use 'ag repo content list' instead

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output directory entries as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Branch, tag, or commit identifier | `` | local |


## ag repo read-file

Usage: `ag repo read-file [<owner>/<repo>] <path> [flags]`

Read a file from a repository

> Deprecated: use 'ag repo content view' instead

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output file content as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Branch, tag, or commit identifier | `` | local |


## ag repo sync

Usage: `ag repo sync [<owner>/<repo>] [flags]`

Synchronize a fork with its upstream repository

Synchronize a remote AtomGit fork branch with its upstream repository.

The repository must be a fork with an available upstream repository. The
repository default branch is used unless --branch is specified. This command
updates only the remote fork and does not modify the local Git working tree.

By default, AtomGit performs a non-forced synchronization and reports a
conflict instead of overwriting divergent commits. --force requires an
interactive confirmation unless --yes is also supplied.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --branch` | Branch to synchronize (defaults to the repository default branch) | `` | local |
| `-f, --force` | Overwrite divergent commits after confirmation | `false` | local |
| `-y, --yes` | Skip confirmation when --force is used | `false` | local |

### Example

```bash
# Synchronize the current repository's default branch
  ag repo sync

  # Synchronize an explicit branch of a fork
  ag repo sync owner/fork --branch develop

  # Force synchronization after interactive confirmation
  ag repo sync owner/fork --branch develop --force

  # Force synchronization non-interactively
  ag repo sync owner/fork --branch develop --force --yes
```


## ag repo transfer

Usage: `ag repo transfer [<owner>/<repo>] --to <organization> [flags]`

Transfer a repository to an organization

Transfer a repository to an AtomGit organization.

Repository transfer can change access, URLs, and automation. The command shows
the source and resolved destination organization and requires confirmation
unless --yes is supplied. The currently documented AtomGit transfer APIs only
describe organization destinations, so personal-user destinations are rejected.
After the transfer, the authoritative repository name and URL are read back from
AtomGit. Local Git remotes are never modified.

AtomGit requires the account password when the source repository is owned by
an organization. It is read without echo from an interactive terminal, or from
standard input when --password-stdin is combined with --yes.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--password-stdin` | Read the organization-transfer password from standard input (requires --yes) | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--to` | Destination organization namespace (required) | `` | local |
| `-y, --yes` | Skip the confirmation prompt | `false` | local |

### Example

```bash
ag repo transfer owner/repo --to target-organization
  ag repo transfer owner/repo --to target-organization --yes
  printf '%s\n' "$PASSWORD" | ag repo transfer source-organization/repo --to target-organization --yes --password-stdin
```


## ag repo view

Usage: `ag repo view [<owner>/<repo>] [flags]`

View a repository

View a repository

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output repository as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-w, --web` | Open a repository in the browser | `false` | local |


## ag repo webhook

Usage: `ag repo webhook`

Manage repository webhooks

List, inspect, create, edit, delete, and test repository webhooks.

Webhook secrets are accepted only from an environment variable, a file, or
standard input. They are never included in command output. The API exposes the
active state as read-only metadata, so this command does not send an
undocumented active field.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag repo webhook create

Usage: `ag repo webhook create [<owner>/<repo>] [flags]`

Create a repository webhook

Create a repository webhook

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--encryption` | Secret mode: password or signature | `` | local |
| `--events` | Comma-separated events: push, tag-push, issues, note, merge-requests | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--secret-env` | Read the webhook secret from this environment variable | `` | local |
| `--secret-file` | Read the webhook secret from a file | `` | local |
| `--secret-stdin` | Read the webhook secret from standard input | `false` | local |
| `--url` | Webhook target HTTP(S) URL | `` | local |

### Example

```bash
ag repo webhook create owner/repo --url https://example.com/hook --events push,issues --secret-env WEBHOOK_SECRET
```


## ag repo webhook delete

Usage: `ag repo webhook delete [<owner>/<repo>] <id> [flags]`

Delete a repository webhook

Delete a repository webhook

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag repo webhook delete owner/repo 42 --yes
```


## ag repo webhook edit

Usage: `ag repo webhook edit [<owner>/<repo>] <id> [flags]`

Edit a repository webhook

Edit a repository webhook

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--encryption` | Secret mode: password or signature | `` | local |
| `--events` | Replace events; use none to disable all events | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--secret-env` | Read the webhook secret from this environment variable | `` | local |
| `--secret-file` | Read the webhook secret from a file | `` | local |
| `--secret-stdin` | Read the webhook secret from standard input | `false` | local |
| `--url` | New webhook target HTTP(S) URL | `` | local |

### Example

```bash
ag repo webhook edit owner/repo 42 --events push,merge-requests
```


## ag repo webhook list

Usage: `ag repo webhook list [<owner>/<repo>] [flags]`

List repository webhooks

List repository webhooks

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output webhooks as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of webhooks to list | `30` | local |

### Example

```bash
ag repo webhook list owner/repo --limit 50
```


## ag repo webhook test

Usage: `ag repo webhook test [<owner>/<repo>] <id> [flags]`

Send a test payload to a repository webhook

Send a test payload to a repository webhook

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag repo webhook test owner/repo 42 --yes
```


## ag repo webhook view

Usage: `ag repo webhook view [<owner>/<repo>] <id> [flags]`

View a repository webhook

View a repository webhook

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output webhook as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag repo webhook view owner/repo 42
```


## ag run

Usage: `ag run`

Inspect AtomGit Actions workflow runs and artifacts

List and inspect AtomGit Actions workflow runs, jobs, logs, and artifacts.

Artifacts can also be deleted after confirmation. Workflow run dispatch,
rerun, cancel, and deletion operations are not supported.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag run artifact

Usage: `ag run artifact`

Inspect and manage workflow artifacts

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag run artifact delete

Usage: `ag run artifact delete [<owner>/<repo>] <artifact-id> [flags]`

Delete a workflow artifact

Delete an AtomGit Actions artifact after reading its metadata.

By default, the command displays the artifact details and asks for
confirmation. Deletion cannot be undone. Use --yes to skip the prompt.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip deletion confirmation | `false` | local |

### Example

```bash
ag run artifact delete owner/repo <artifact-id>
  ag run artifact delete <artifact-id> --yes
```


## ag run artifact view

Usage: `ag run artifact view [<owner>/<repo>] <artifact-id> [flags]`

View artifact metadata without downloading the archive

Display AtomGit Actions artifact metadata.

This command does not download the archive. Use ag run view --artifact to
download a zip from a specific workflow run.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output artifact metadata as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag run artifact view owner/repo <artifact-id>
  ag run artifact view <artifact-id> --json
```


## ag run list

Usage: `ag run list [<owner>/<repo>] [flags]`

List workflow runs

List workflow runs

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--actor` | Filter by triggering username | `` | local |
| `--end-time` | Filter runs ending at or before this Unix timestamp in milliseconds | `0` | local |
| `--event` | Filter by event: mr, push, manual | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output workflow runs as JSON | `false` | local |
| `--pr` | Filter by pull request number | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--start-time` | Filter runs starting at or after this Unix timestamp in milliseconds | `0` | local |
| `--workflow` | Filter by workflow ID | `` | local |
| `--workflow-name` | Filter by workflow name | `` | local |
| `-L, --limit` | Maximum number of runs to list | `30` | local |
| `-b, --branch` | Filter by head branch | `` | local |
| `-s, --status` | Filter by status: completed, running, failed, canceled, ignored, paused, suspend | `` | local |

### Example

```bash
ag run list owner/repo
  ag run list
  ag run list owner/repo --branch main --status failed
  ag run list owner/repo --event push --workflow-name CI --limit 50
```


## ag run step-log

Usage: `ag run step-log [<owner>/<repo>] <run-id> <job-id> <step-id> [flags]`

Fetch step-level logs for a workflow job

Retrieve paginated AtomGit Actions step logs and write them as text.

Use ag run view to discover step IDs. --output writes the complete log
atomically and refuses to replace an existing file unless --overwrite is set.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--output` | Write the complete log to a file instead of stdout | `` | local |
| `--overwrite` | Replace an existing --output destination | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag run step-log owner/repo <run-id> <job-id> <step-id>
  ag run step-log <run-id> <job-id> <step-id>
  ag run step-log owner/repo <run-id> <job-id> <step-id> --output step.log
  ag run step-log owner/repo <run-id> <job-id> <step-id> --output step.log --overwrite
```


## ag run view

Usage: `ag run view [<owner>/<repo>] <run-id> [flags]`

View a workflow run, jobs, logs, and artifacts

View a workflow run, jobs, logs, and artifacts

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--artifact` | Download a specific artifact as a zip archive | `` | local |
| `--artifact-file` | Artifact destination path (defaults to the artifact name) | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--log` | Write the selected job log text to stdout | `false` | local |
| `--log-file` | Download the selected job log archive to a file | `` | local |
| `--overwrite` | Replace an existing download destination | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-j, --job` | View a specific job | `` | local |

### Example

```bash
ag run view owner/repo 12345
  ag run view 12345
  ag run view owner/repo 12345 --job job-id
  ag run view owner/repo 12345 --job job-id --log
  ag run view owner/repo 12345 --job job-id --log-file job-logs.zip
  ag run view owner/repo 12345 --artifact artifact-id
  ag run view owner/repo 12345 --artifact artifact-id --artifact-file build.zip --overwrite
```


## ag runner

Usage: `ag runner`

Inspect AtomGit Actions host runners

List repository-specific and shared AtomGit Actions host runners. These commands are read-only.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag runner list owner/repo
  ag runner shared owner/repo --json
```


## ag runner list

Usage: `ag runner list [<owner>/<repo>] [flags]`

List host runners configured for a repository

List host runners configured for a repository. This command is read-only and does not change runner configuration.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output runners as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of runners to list (0 means all) | `0` | local |

### Example

```bash
ag runner list owner/repo
  ag runner list owner/repo --limit 25 --json
```


## ag runner shared

Usage: `ag runner shared [<owner>/<repo>] [flags]`

List host runners shared with a repository

List host runners shared with a repository. This command is read-only and does not change runner configuration.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output runners as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of runners to list (0 means all) | `0` | local |

### Example

```bash
ag runner shared owner/repo
  ag runner shared owner/repo --limit 25 --json
```


## ag search

Usage: `ag search`

search atomgit

Search AtomGit repositories, issues, and users. Pull request search is not supported.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag search issues

Usage: `ag search issues <query> [flags]`

search issues

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output results as JSON | `false` | local |
| `--order` | Sort order: asc or desc | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--repo` | Filter by repository path | `` | local |
| `--sort` | Sort by created_at or last_push_at | `` | local |
| `--state` | Filter by state: open or closed | `` | local |
| `-L, --limit` | Maximum number of results | `30` | local |


## ag search repositories

Usage: `ag search repositories <query> [flags]`

search repositories

Aliases: `repos`

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--fork` | Include forked repositories | `false` | local |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output results as JSON | `false` | local |
| `--language` | Filter by repository language | `` | local |
| `--order` | Sort order: asc or desc | `` | local |
| `--owner` | Filter by repository owner path | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--sort` | Sort by last_push_at, stars_count, or forks_count | `` | local |
| `-L, --limit` | Maximum number of results | `30` | local |


## ag search users

Usage: `ag search users <query> [flags]`

search users

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output results as JSON | `false` | local |
| `--order` | Sort order: asc or desc | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--sort` | Sort by joined_at | `` | local |
| `-L, --limit` | Maximum number of results | `30` | local |


## ag ssh-key

Usage: `ag ssh-key <command>`

Manage SSH keys

Manage SSH keys registered with your AtomGit account.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag ssh-key add

Usage: `ag ssh-key add [<key-file>] [flags]`

Add an SSH key to your AtomGit account

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-t, --title` | Title for the new key | `` | local |


## ag ssh-key delete

Usage: `ag ssh-key delete <id> [flags]`

Delete an SSH key from your AtomGit account

Delete an SSH key from your AtomGit account.

The target key is retrieved before deletion. By default, you will be prompted
to confirm the deletion. Use --yes to skip the confirmation prompt.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag ssh-key delete 123
  ag ssh-key delete 123 --yes
```


## ag ssh-key list

Usage: `ag ssh-key list [flags]`

List SSH keys registered with your AtomGit account

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--limit` | Maximum number of SSH keys to list | `1000` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag ssh-key list
  ag ssh-key list --limit 200
```


## ag tag

Usage: `ag tag`

Manage tags

List, create, and delete tags, and manage protected tag rules.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag tag create

Usage: `ag tag create [<owner>/<repo>] <tag_name> [flags]`

Create a tag

Create a tag

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Branch, tag, or commit SHA to create the tag from (required) | `` | local |
| `-m, --message` | Tag message | `` | local |


## ag tag delete

Usage: `ag tag delete [<owner>/<repo>] <tag_name> [flags]`

Delete a tag

Delete a tag from AtomGit.

By default, you will be prompted to confirm the deletion. Use --yes to skip
the confirmation prompt.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag tag delete owner/repo v1.0.0
  ag tag delete owner/repo v1.0.0 --yes
```


## ag tag list

Usage: `ag tag list [<owner>/<repo>] [flags]`

List tags

List tags

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output tags as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of tags to list | `30` | local |

### Example

```bash
ag tag list owner/repo --limit 50
```


## ag tag protection

Usage: `ag tag protection`

Manage protected tag rules

List, view, create, update, and delete protected tag rules.

Rules may name an exact tag or contain an AtomGit wildcard pattern. AtomGit's
API exposes only create/push access levels; other web settings are not changed
by this command.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag tag protection delete

Usage: `ag tag protection delete [<owner>/<repo>] <tag-or-pattern> [flags]`

Delete a protected tag rule

Delete a protected tag rule.

By default, the current repository and rule are shown and you will be prompted
to confirm. Use --yes to skip the confirmation prompt.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip deletion confirmation | `false` | local |

### Example

```bash
ag tag protection delete owner/repo "v*"
  ag tag protection delete owner/repo "v*" --yes
```


## ag tag protection list

Usage: `ag tag protection list [<owner>/<repo>] [flags]`

List protected tag rules

List protected tag rules

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output protected tag rules as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of protected tag rules to list | `30` | local |

### Example

```bash
ag tag protection list owner/repo --limit 50
```


## ag tag protection set

Usage: `ag tag protection set [<owner>/<repo>] <tag-or-pattern> [flags]`

Create or update a protected tag rule

Create or update a protected tag rule.

--create-access accepts none, developer, or maintainer. These map to AtomGit
create_access_level values 0, 30, and 40: nobody; Developer/Maintainer/Admin;
and Maintainer/Admin. Omitting the flag on create uses the server default
(maintainer). Existing rules keep the current access level when the flag is
omitted. Updating an existing rule requires confirmation unless --yes is
supplied.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--create-access` | Create access: none, developer, or maintainer | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-y, --yes` | Skip confirmation when updating an existing rule | `false` | local |

### Example

```bash
ag tag protection set owner/repo v1.0.0 --create-access maintainer
  ag tag protection set owner/repo "v*" --create-access developer
  ag tag protection set owner/repo v1.0.0
  ag tag protection set owner/repo v1.0.0 --create-access none --yes
```


## ag tag protection view

Usage: `ag tag protection view [<owner>/<repo>] <tag-or-pattern> [flags]`

View a protected tag rule

View a protected tag rule

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output the protected tag rule as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag update

Usage: `ag update [flags]`

Update AtomGit CLI to the latest stable release

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-c, --check` | Check for an update without installing it | `false` | local |


## ag user

Usage: `ag user`

View AtomGit users, repositories, namespaces, and activity

View AtomGit user profiles, email addresses, starred and watched repositories, authenticated namespaces, and personal activity events.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag user emails

Usage: `ag user emails [flags]`

List email addresses for the authenticated user

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output email addresses as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag user emails
  ag user emails --json
```


## ag user events

Usage: `ag user events [<username>] [flags]`

List personal activity events for a user

List personal activity events for a user. Without an explicit username, the authenticated user is used.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output events as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--year` | Filter events to the specified year (0 disables the filter) | `0` | local |
| `-L, --limit` | Maximum number of events to list | `30` | local |

### Example

```bash
ag user events
  ag user events alice
  ag user events alice --year 2026 --limit 50
  ag user events alice --json
```


## ag user namespaces

Usage: `ag user namespaces [flags]`

List namespaces for the authenticated user

List user and group namespaces visible to the authenticated user. The default mode is intrant.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output namespaces as JSON | `false` | local |
| `--mode` | Namespace source: intrant, project, or all | `intrant` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of namespaces to list | `30` | local |

### Example

```bash
ag user namespaces
  ag user namespaces --mode project --limit 100
  ag user namespaces --mode all --json
```


## ag user starred

Usage: `ag user starred [<username>] [flags]`

List starred repositories for a user

List starred repositories for a user. Without a username, the authenticated-user endpoint is used; an explicit username selects the public-user endpoint. Authentication is required for both endpoints.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output repositories as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of repositories to list | `30` | local |

### Example

```bash
ag user starred
  ag user starred alice --limit 100
  ag user starred alice --json
```


## ag user view

Usage: `ag user view [<login>] [flags]`

View the current user or a public user profile

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output the user profile as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-w, --web` | Open the user profile in the browser | `false` | local |

### Example

```bash
ag user view
  ag user view alice
  ag user view alice --json
  ag user view alice --web
```


## ag user watching

Usage: `ag user watching [<username>] [flags]`

List watched repositories for a user

List watched repositories for a user. Without a username, the authenticated-user endpoint is used; an explicit username selects the public-user endpoint. Authentication is required for both endpoints.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output repositories as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of repositories to list | `30` | local |

### Example

```bash
ag user watching
  ag user watching alice --limit 100
  ag user watching alice --json
```


## ag version

Usage: `ag version [flags]`

Show version information

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output version information as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |


## ag workflow

Usage: `ag workflow`

Manage AtomGit Actions workflows

List, validate, and run AtomGit Actions workflows.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag workflow list owner/repo
  ag workflow validate --file .gitcode/workflows/ci.yml
  ag workflow run owner/repo 12345 --ref main
  ag workflow run owner/repo ci.yml -f env=production
```


## ag workflow list

Usage: `ag workflow list [<owner>/<repo>] [flags]`

List workflows in a repository

List workflows in a repository

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output workflows as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag workflow list owner/repo
```


## ag workflow run

Usage: `ag workflow run [<owner>/<repo>] <workflow_id> [flags]`

Run a workflow

Manually trigger an AtomGit Actions workflow run (workflow_dispatch).

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

Aliases: `dispatch`

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--help` | Show help for command | `false` | inherited |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --field` | Add a string parameter in key=value format | `[]` | local |
| `-f, --raw-field` | Add a string parameter in key=value format | `[]` | local |
| `-r, --ref` | The git reference (branch or tag) to run the workflow on | `` | local |

### Example

```bash
ag workflow run owner/repo 12345 --ref main
  ag workflow run owner/repo ci.yml --ref feature-branch -f env=prod -f debug=true
```


## ag workflow validate

Usage: `ag workflow validate [<owner>/<repo>] [flags]`

Validate a local workflow YAML file

Validate a local AtomGit Actions workflow file against the documented v8 endpoint.

The file is not modified. HTTP 200 with valid=false is treated as a command
error so CI can fail on invalid YAML.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--file` | Path to a local workflow YAML file | `` | local |
| `--help` | Show help for command | `false` | inherited |
| `--json` | Output the validation response as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |

### Example

```bash
ag workflow validate --file .gitcode/workflows/ci.yml
  ag workflow validate owner/repo --file workflow.yml
  ag workflow validate owner/repo --file workflow.yml --json
```
