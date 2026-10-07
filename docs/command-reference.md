# AtomGit CLI command reference

> This file is generated from the Cobra command tree. Do not edit it manually.
> Regenerate with `go run ./scripts/generate-command-reference`.

> Only commands registered in the root command tree are emitted. The optional Cobra completion helper is not registered by this CLI and is intentionally omitted.

## Command index

- [ag-cli](#ag-cli) — AtomGit CLI
- [ag-cli alias](#ag-cli-alias) — Create command shortcuts
- [ag-cli alias delete](#ag-cli-alias-delete) — Delete an alias
- [ag-cli alias list](#ag-cli-alias-list) — List aliases
- [ag-cli alias set](#ag-cli-alias-set) — Create a shortcut for an ag-cli command
- [ag-cli api](#ag-cli-api) — Make an authenticated AtomGit API request
- [ag-cli auth](#ag-cli-auth) — Authenticate with AtomGit
- [ag-cli auth git-credential](#ag-cli-auth-git-credential) — Implement the Git credential helper protocol
- [ag-cli auth list](#ag-cli-auth-list) — List saved AtomGit accounts
- [ag-cli auth login](#ag-cli-auth-login) — Log in with AtomGit OAuth (opens browser, saves token.json)
- [ag-cli auth logout](#ag-cli-auth-logout) — Remove the active or a selected stored account
- [ag-cli auth refresh](#ag-cli-auth-refresh) — Refresh the access token using the stored refresh_token
- [ag-cli auth setup-git](#ag-cli-auth-setup-git) — Configure Git to use ag-cli as a credential helper
- [ag-cli auth status](#ag-cli-auth-status) — View local authentication status or verify identity online
- [ag-cli auth switch](#ag-cli-auth-switch) — Switch the active account and synchronize Git identity
- [ag-cli auth token](#ag-cli-auth-token) — Print the authentication token
- [ag-cli branch](#ag-cli-branch) — Manage remote branches
- [ag-cli branch create](#ag-cli-branch-create) — Create a remote branch
- [ag-cli branch delete](#ag-cli-branch-delete) — Delete a remote branch
- [ag-cli branch list](#ag-cli-branch-list) — List remote branches
- [ag-cli branch protection](#ag-cli-branch-protection) — Manage protected branch rules
- [ag-cli branch protection delete](#ag-cli-branch-protection-delete) — Delete a protected branch rule
- [ag-cli branch protection list](#ag-cli-branch-protection-list) — List protected branch rules
- [ag-cli branch protection set](#ag-cli-branch-protection-set) — Create or update a protected branch rule
- [ag-cli branch protection view](#ag-cli-branch-protection-view) — View a protected branch rule
- [ag-cli branch view](#ag-cli-branch-view) — View a remote branch
- [ag-cli browse](#ag-cli-browse) — Open repositories, issues, pull requests, and more in the browser
- [ag-cli check-update](#ag-cli-check-update) — Check for a newer AtomGit CLI release
- [ag-cli commit](#ag-cli-commit) — Manage commits
- [ag-cli commit comment](#ag-cli-commit-comment) — Manage commit comments
- [ag-cli commit comment create](#ag-cli-commit-comment-create) — Create a comment on a commit
- [ag-cli commit comment delete](#ag-cli-commit-comment-delete) — Delete a commit comment
- [ag-cli commit comment edit](#ag-cli-commit-comment-edit) — Edit a commit comment
- [ag-cli commit comment list](#ag-cli-commit-comment-list) — List comments on a commit
- [ag-cli commit comment view](#ag-cli-commit-comment-view) — View a commit comment
- [ag-cli commit compare](#ag-cli-commit-compare) — Compare two commits, branches, or tags
- [ag-cli commit diff](#ag-cli-commit-diff) — Show a commit's diff
- [ag-cli commit list](#ag-cli-commit-list) — List commits
- [ag-cli commit patch](#ag-cli-commit-patch) — Show a commit's patch
- [ag-cli commit view](#ag-cli-commit-view) — View a commit
- [ag-cli discussion](#ag-cli-discussion) — View repository discussions
- [ag-cli discussion list](#ag-cli-discussion-list) — List repository discussions
- [ag-cli discussion view](#ag-cli-discussion-view) — View a repository discussion
- [ag-cli doctor](#ag-cli-doctor) — CLI health check: config, auth, and connectivity
- [ag-cli issue](#ag-cli-issue) — Manage issues
- [ag-cli issue activity](#ag-cli-issue-activity) — List operation logs for an issue
- [ag-cli issue branches](#ag-cli-issue-branches) — List or update related branches for an issue
- [ag-cli issue close](#ag-cli-issue-close) — Close an issue
- [ag-cli issue comment](#ag-cli-issue-comment) — Manage issue comments
- [ag-cli issue comment create](#ag-cli-issue-comment-create) — Create a comment on an issue
- [ag-cli issue comment delete](#ag-cli-issue-comment-delete) — Delete a comment on an issue
- [ag-cli issue comment edit](#ag-cli-issue-comment-edit) — Edit a comment on an issue
- [ag-cli issue comment view](#ag-cli-issue-comment-view) — View all comments on an issue
- [ag-cli issue create](#ag-cli-issue-create) — Create an issue
- [ag-cli issue edit](#ag-cli-issue-edit) — Edit an issue
- [ag-cli issue history](#ag-cli-issue-history) — List modification history for an issue
- [ag-cli issue label](#ag-cli-issue-label) — Add or remove labels on an issue
- [ag-cli issue list](#ag-cli-issue-list) — List issues
- [ag-cli issue prs](#ag-cli-issue-prs) — List pull requests linked to an issue
- [ag-cli issue reactions](#ag-cli-issue-reactions) — List reactions on an issue
- [ag-cli issue reopen](#ag-cli-issue-reopen) — Reopen an issue
- [ag-cli issue view](#ag-cli-issue-view) — View an issue
- [ag-cli kanban](#ag-cli-kanban) — View organization Kanban boards
- [ag-cli kanban items](#ag-cli-kanban-items) — List items on a Kanban board
- [ag-cli kanban list](#ag-cli-kanban-list) — List organization Kanban boards
- [ag-cli kanban view](#ag-cli-kanban-view) — View a Kanban board
- [ag-cli label](#ag-cli-label) — Manage repository labels
- [ag-cli label create](#ag-cli-label-create) — Create a repository label
- [ag-cli label delete](#ag-cli-label-delete) — Delete a repository label
- [ag-cli label edit](#ag-cli-label-edit) — Edit a repository label
- [ag-cli label list](#ag-cli-label-list) — List repository labels
- [ag-cli license](#ag-cli-license) — License compliance checking
- [ag-cli license check](#ag-cli-license-check) — Check license compliance
- [ag-cli milestone](#ag-cli-milestone) — Manage repository milestones
- [ag-cli milestone close](#ag-cli-milestone-close) — Close a repository milestone
- [ag-cli milestone create](#ag-cli-milestone-create) — Create a repository milestone
- [ag-cli milestone delete](#ag-cli-milestone-delete) — Delete a repository milestone
- [ag-cli milestone edit](#ag-cli-milestone-edit) — Edit a repository milestone
- [ag-cli milestone list](#ag-cli-milestone-list) — List repository milestones
- [ag-cli milestone reopen](#ag-cli-milestone-reopen) — Reopen a repository milestone
- [ag-cli milestone view](#ag-cli-milestone-view) — View a repository milestone
- [ag-cli notification](#ag-cli-notification) — Manage repository notifications
- [ag-cli notification list](#ag-cli-notification-list) — List repository notifications
- [ag-cli notification mark-read](#ag-cli-notification-mark-read) — Mark repository notifications as read
- [ag-cli org](#ag-cli-org) — Manage organizations
- [ag-cli org list](#ag-cli-org-list) — List organizations for the authenticated user
- [ag-cli org members](#ag-cli-org-members) — List organization members
- [ag-cli org repos](#ag-cli-org-repos) — List organization repositories
- [ag-cli org runner-group](#ag-cli-org-runner-group) — Inspect organization Actions runner groups
- [ag-cli org runner-group list](#ag-cli-org-runner-group-list) — List organization runner groups
- [ag-cli org runner-group namespaces](#ag-cli-org-runner-group-namespaces) — List repositories that can use an organization runner group
- [ag-cli org runner-group runner-sets](#ag-cli-org-runner-group-runner-sets) — List Kubernetes runner sets in an organization runner group
- [ag-cli org runner-group runners](#ag-cli-org-runner-group-runners) — List host runners in an organization runner group
- [ag-cli org runner-group view](#ag-cli-org-runner-group-view) — View an organization runner group
- [ag-cli org view](#ag-cli-org-view) — View an organization
- [ag-cli pr](#ag-cli-pr) — Manage pull requests
- [ag-cli pr activity](#ag-cli-pr-activity) — List the operation log of a pull request
- [ag-cli pr checkout](#ag-cli-pr-checkout) — Check out a pull request locally
- [ag-cli pr checks](#ag-cli-pr-checks) — Show CI checks for a pull request's current head commit
- [ag-cli pr close](#ag-cli-pr-close) — Close a pull request
- [ag-cli pr comment](#ag-cli-pr-comment) — Manage pull request comments
- [ag-cli pr comment create](#ag-cli-pr-comment-create) — Create a comment on a pull request
- [ag-cli pr comment delete](#ag-cli-pr-comment-delete) — Delete a comment on a pull request
- [ag-cli pr comment edit](#ag-cli-pr-comment-edit) — Edit a comment on a pull request
- [ag-cli pr comment reply](#ag-cli-pr-comment-reply) — Reply to a comment thread on a pull request
- [ag-cli pr comment view](#ag-cli-pr-comment-view) — View all comments on a pull request
- [ag-cli pr commits](#ag-cli-pr-commits) — List commits in a pull request
- [ag-cli pr create](#ag-cli-pr-create) — Create a pull request
- [ag-cli pr diff](#ag-cli-pr-diff) — Show diff of a pull request
- [ag-cli pr edit](#ag-cli-pr-edit) — Edit a pull request
- [ag-cli pr files](#ag-cli-pr-files) — List files changed in a pull request
- [ag-cli pr history](#ag-cli-pr-history) — List the modification history of a pull request
- [ag-cli pr issues](#ag-cli-pr-issues) — View linked issues of a pull request
- [ag-cli pr link-issues](#ag-cli-pr-link-issues) — Link issues to a pull request
- [ag-cli pr list](#ag-cli-pr-list) — List pull requests
- [ag-cli pr merge](#ag-cli-pr-merge) — Merge a pull request
- [ag-cli pr reactions](#ag-cli-pr-reactions) — List reactions on a pull request
- [ag-cli pr reopen](#ag-cli-pr-reopen) — Reopen a pull request
- [ag-cli pr review](#ag-cli-pr-review) — Approve a pull request review
- [ag-cli pr unlink-issues](#ag-cli-pr-unlink-issues) — Unlink issues from a pull request
- [ag-cli pr view](#ag-cli-pr-view) — View a pull request
- [ag-cli release](#ag-cli-release) — Manage repository releases
- [ag-cli release create](#ag-cli-release-create) — Create a release
- [ag-cli release download](#ag-cli-release-download) — Download an attachment from a release
- [ag-cli release edit](#ag-cli-release-edit) — Edit a release
- [ag-cli release list](#ag-cli-release-list) — List repository releases
- [ag-cli release upload](#ag-cli-release-upload) — Upload an attachment to a release
- [ag-cli release view](#ag-cli-release-view) — View a release by tag
- [ag-cli repo](#ag-cli-repo) — Manage repositories
- [ag-cli repo clone](#ag-cli-repo-clone) — Clone a repository
- [ag-cli repo collaborator](#ag-cli-repo-collaborator) — Manage repository collaborators
- [ag-cli repo collaborator add](#ag-cli-repo-collaborator-add) — Add a direct repository collaborator
- [ag-cli repo collaborator edit](#ag-cli-repo-collaborator-edit) — Update a direct repository collaborator's permission
- [ag-cli repo collaborator list](#ag-cli-repo-collaborator-list) — List repository collaborators
- [ag-cli repo collaborator remove](#ag-cli-repo-collaborator-remove) — Remove a direct repository collaborator
- [ag-cli repo collaborator view](#ag-cli-repo-collaborator-view) — View a repository collaborator's effective permission
- [ag-cli repo content](#ag-cli-repo-content) — Browse repository contents
- [ag-cli repo content list](#ag-cli-repo-content-list) — List a repository directory
- [ag-cli repo content view](#ag-cli-repo-content-view) — View a repository file
- [ag-cli repo create](#ag-cli-repo-create) — Create a new repository
- [ag-cli repo delete](#ag-cli-repo-delete) — Delete a repository
- [ag-cli repo edit](#ag-cli-repo-edit) — Edit repository settings
- [ag-cli repo fork](#ag-cli-repo-fork) — Fork a repository
- [ag-cli repo fork list](#ag-cli-repo-fork-list) — List forks of a repository
- [ag-cli repo insights](#ag-cli-repo-insights) — Inspect repository activity and statistics
- [ag-cli repo insights contributors](#ag-cli-repo-insights-contributors) — List repository contributor statistics
- [ag-cli repo insights downloads](#ag-cli-repo-insights-downloads) — Show repository download statistics
- [ag-cli repo insights events](#ag-cli-repo-insights-events) — List repository activity events
- [ag-cli repo insights languages](#ag-cli-repo-insights-languages) — Show repository language percentages
- [ag-cli repo insights stargazers](#ag-cli-repo-insights-stargazers) — List repository stargazers
- [ag-cli repo insights watchers](#ag-cli-repo-insights-watchers) — List repository watchers
- [ag-cli repo list](#ag-cli-repo-list) — List repositories
- [ag-cli repo mirror](#ag-cli-repo-mirror) — Inspect repository remote mirrors
- [ag-cli repo mirror list](#ag-cli-repo-mirror-list) — List configured push remote mirrors
- [ag-cli repo mirror view](#ag-cli-repo-mirror-view) — View repository remote mirror state
- [ag-cli repo policy](#ag-cli-repo-policy) — View and edit repository policy settings
- [ag-cli repo policy edit](#ag-cli-repo-policy-edit) — Edit one repository policy section
- [ag-cli repo policy view](#ag-cli-repo-policy-view) — View repository policy settings
- [ag-cli repo push-rule](#ag-cli-repo-push-rule) — Manage repository push rules
- [ag-cli repo push-rule edit](#ag-cli-repo-push-rule-edit) — Edit repository push rules
- [ag-cli repo push-rule view](#ag-cli-repo-push-rule-view) — View repository push rules
- [ag-cli repo read-dir](#ag-cli-repo-read-dir) — List contents of a repository directory
- [ag-cli repo read-file](#ag-cli-repo-read-file) — Read a file from a repository
- [ag-cli repo sync](#ag-cli-repo-sync) — Synchronize a fork with its upstream repository
- [ag-cli repo transfer](#ag-cli-repo-transfer) — Transfer a repository to an organization
- [ag-cli repo view](#ag-cli-repo-view) — View a repository
- [ag-cli repo webhook](#ag-cli-repo-webhook) — Manage repository webhooks
- [ag-cli repo webhook create](#ag-cli-repo-webhook-create) — Create a repository webhook
- [ag-cli repo webhook delete](#ag-cli-repo-webhook-delete) — Delete a repository webhook
- [ag-cli repo webhook edit](#ag-cli-repo-webhook-edit) — Edit a repository webhook
- [ag-cli repo webhook list](#ag-cli-repo-webhook-list) — List repository webhooks
- [ag-cli repo webhook test](#ag-cli-repo-webhook-test) — Send a test payload to a repository webhook
- [ag-cli repo webhook view](#ag-cli-repo-webhook-view) — View a repository webhook
- [ag-cli run](#ag-cli-run) — Inspect AtomGit Actions workflow runs and artifacts
- [ag-cli run artifact](#ag-cli-run-artifact) — Inspect and manage workflow artifacts
- [ag-cli run artifact delete](#ag-cli-run-artifact-delete) — Delete a workflow artifact
- [ag-cli run artifact view](#ag-cli-run-artifact-view) — View artifact metadata without downloading the archive
- [ag-cli run list](#ag-cli-run-list) — List workflow runs
- [ag-cli run step-log](#ag-cli-run-step-log) — Fetch step-level logs for a workflow job
- [ag-cli run view](#ag-cli-run-view) — View a workflow run, jobs, logs, and artifacts
- [ag-cli runner](#ag-cli-runner) — Inspect AtomGit Actions host runners
- [ag-cli runner list](#ag-cli-runner-list) — List host runners configured for a repository
- [ag-cli runner shared](#ag-cli-runner-shared) — List host runners shared with a repository
- [ag-cli schema](#ag-cli-schema) — Describe public commands as versioned JSON
- [ag-cli search](#ag-cli-search) — search atomgit
- [ag-cli search issues](#ag-cli-search-issues) — search issues
- [ag-cli search repositories](#ag-cli-search-repositories) — search repositories
- [ag-cli search users](#ag-cli-search-users) — search users
- [ag-cli ssh-key](#ag-cli-ssh-key) — Manage SSH keys
- [ag-cli ssh-key add](#ag-cli-ssh-key-add) — Add an SSH key to your AtomGit account
- [ag-cli ssh-key delete](#ag-cli-ssh-key-delete) — Delete an SSH key from your AtomGit account
- [ag-cli ssh-key list](#ag-cli-ssh-key-list) — List SSH keys registered with your AtomGit account
- [ag-cli tag](#ag-cli-tag) — Manage tags
- [ag-cli tag create](#ag-cli-tag-create) — Create a tag
- [ag-cli tag delete](#ag-cli-tag-delete) — Delete a tag
- [ag-cli tag list](#ag-cli-tag-list) — List tags
- [ag-cli tag protection](#ag-cli-tag-protection) — Manage protected tag rules
- [ag-cli tag protection delete](#ag-cli-tag-protection-delete) — Delete a protected tag rule
- [ag-cli tag protection list](#ag-cli-tag-protection-list) — List protected tag rules
- [ag-cli tag protection set](#ag-cli-tag-protection-set) — Create or update a protected tag rule
- [ag-cli tag protection view](#ag-cli-tag-protection-view) — View a protected tag rule
- [ag-cli update](#ag-cli-update) — Update AtomGit CLI to the latest stable release
- [ag-cli user](#ag-cli-user) — View AtomGit users, repositories, namespaces, and activity
- [ag-cli user edit](#ag-cli-user-edit) — Edit the authenticated user's profile
- [ag-cli user emails](#ag-cli-user-emails) — List email addresses for the authenticated user
- [ag-cli user events](#ag-cli-user-events) — List personal activity events for a user
- [ag-cli user namespaces](#ag-cli-user-namespaces) — List namespaces for the authenticated user
- [ag-cli user starred](#ag-cli-user-starred) — List starred repositories for a user
- [ag-cli user view](#ag-cli-user-view) — View the current user or a public user profile
- [ag-cli user watching](#ag-cli-user-watching) — List watched repositories for a user
- [ag-cli version](#ag-cli-version) — Show version information
- [ag-cli workflow](#ag-cli-workflow) — Manage AtomGit Actions workflows
- [ag-cli workflow list](#ag-cli-workflow-list) — List workflows in a repository
- [ag-cli workflow run](#ag-cli-workflow-run) — Run a workflow
- [ag-cli workflow validate](#ag-cli-workflow-validate) — Validate a local workflow YAML file

## ag-cli

Usage: `ag-cli <command> <subcommand> [flags]`

AtomGit CLI

Work seamlessly with AtomGit from the command line.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | local |
| `--version` | Show version information | `false` | local |
| `-h, --help` | Show help for command | `false` | local |


## ag-cli alias

Usage: `ag-cli alias`

Create command shortcuts

Create, list, and delete command shortcuts (aliases) for "ag-cli" commands.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli alias delete

Usage: `ag-cli alias delete <alias>`

Delete an alias

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli alias list

Usage: `ag-cli alias list`

List aliases

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli alias set

Usage: `ag-cli alias set <alias> <expansion>...`

Create a shortcut for an ag-cli command

Create a shortcut for an ag-cli command.

Aliases are expanded at invocation time: the first non-flag argument of an ag-cli
invocation is looked up and replaced with the expansion. Aliases never
override built-in commands, so names that conflict with a built-in command
are rejected, and the expansion must start with a known built-in command.

To include a literal space inside an expansion argument (for example a
Windows path), escape it with a backslash: C:\Program\ Files.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli alias set pl "pr list"
ag-cli alias set rv repo view
```


## ag-cli api

Usage: `ag-cli api <endpoint> [flags]`

Make an authenticated AtomGit API request

Make an authenticated request to a relative AtomGit API v5 endpoint.

GET is the default. Supported methods are GET, POST, PATCH, PUT, and DELETE.
Explicit non-GET requests may change remote resources; ag-cli does not infer or
confirm the endpoint's effects. Redirects only retain credentials on the exact
AtomGit API origin. Paginated output is one compact JSON page per line.
Response bytes use terminal-safe output unless --raw-output is specified.

Use --dry-run for a local, redacted JSON preview without reading credentials or
sending requests. Values and unrecognized names/path segments are omitted.
Explicit --input files or stdin may be read, but are never modified. A preview
does not verify remote permissions, resource existence, or server-side validation.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--dry-run` | Preview the redacted request as JSON without credentials or network access | `false` | local |
| `--input` | Read the raw request body from a file or - for stdin | `` | local |
| `--paginate` | Request all pages and emit compact JSON pages as NDJSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-H, --accept` | Set the Accept request header | `application/json` | local |
| `-X, --method` | HTTP method: GET, POST, PATCH, PUT, or DELETE | `GET` | local |
| `-f, --field` | Add a string field as key=value | `[]` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli api /user
ag-cli api /repos/owner/repo/issues --field state=open
ag-cli api /repos/owner/repo/issues --method POST --field title='New issue'
ag-cli api /repos/owner/repo/issues/42 --method PATCH --input update.json
ag-cli api /repos/owner/repo/issues --paginate
ag-cli api /repos/owner/repo/issues --method POST --field title=example --dry-run
```


## ag-cli auth

Usage: `ag-cli auth <command>`

Authenticate with AtomGit

Manage authentication state for AtomGit.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli auth git-credential

Usage: `ag-cli auth git-credential <operation>`

Implement the Git credential helper protocol

> Hidden command.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli auth list

Usage: `ag-cli auth list [flags]`

List saved AtomGit accounts

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output accounts as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli auth login

Usage: `ag-cli auth login [flags]`

Log in with AtomGit OAuth (opens browser, saves token.json)

Opens a browser to authorize ag-cli against atomgit.com, then writes
access_token and user to the XDG config path (see README). With --with-token,
skips the browser and reads an existing access token (PAT or OAuth token)
from standard input instead — useful in sandboxes, containers, and CI where
no browser is available:

    echo "$TOKEN" | ag-cli auth login --with-token
    ag-cli auth login --with-token < token.txt

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
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--with-token` | Read an access token from standard input instead of browser OAuth | `false` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli auth logout

Usage: `ag-cli auth logout [flags]`

Remove the active or a selected stored account

Remove the active account or one selected with --account.
An active account can only be removed when it is the last saved account;
otherwise switch to another account first. Use --all to remove every account.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--account` | Account username to remove | `` | local |
| `--all` | Remove all saved accounts | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli auth refresh

Usage: `ag-cli auth refresh`

Refresh the access token using the stored refresh_token

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli auth setup-git

Usage: `ag-cli auth setup-git`

Configure Git to use ag-cli as a credential helper

Configure Git to use AtomGit CLI as the HTTPS credential helper for
atomgit.com. Git requests credentials from the active account selected by
ag-cli auth switch; access tokens are not written to Git configuration.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli auth status

Usage: `ag-cli auth status [flags]`

View local authentication status or verify identity online

Inspect local credentials without modifying them. Local presence does not prove token validity. Use --verify to check identity with the read-only /user API (30 second timeout); this does not verify access to other resources. No token or token fragment is displayed.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output authentication status as JSON, including failures | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--verify` | Verify the active identity online without refreshing or changing credentials | `false` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli auth status
ag-cli auth status --json
ag-cli auth status --verify
ag-cli auth status --verify --json
```


## ag-cli auth switch

Usage: `ag-cli auth switch <account> [flags]`

Switch the active account and synchronize Git identity

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--git-email` | Override Git user.email for this switch | `` | local |
| `--git-name` | Override Git user.name for this switch | `` | local |
| `--global` | Update global Git identity instead of the current repository | `false` | local |
| `--no-git` | Do not update Git identity | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli auth token

Usage: `ag-cli auth token`

Print the authentication token

Display the authentication token used for AtomGit API requests.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli branch

Usage: `ag-cli branch`

Manage remote branches

List, view, create, delete, and protect AtomGit remote branches.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli branch list owner/repo
ag-cli branch view owner/repo main
ag-cli branch create owner/repo feature/foo --ref main
ag-cli branch delete owner/repo feature/foo
ag-cli branch protection list owner/repo
```


## ag-cli branch create

Usage: `ag-cli branch create [<owner>/<repo>] <branch> --ref <ref> [flags]`

Create a remote branch

Create a remote branch

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Source ref to create the branch from | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli branch create owner/repo feature/foo --ref main
```


## ag-cli branch delete

Usage: `ag-cli branch delete [<owner>/<repo>] <branch> [flags]`

Delete a remote branch

Delete a remote branch

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Confirm branch deletion without prompting | `false` | local |

### Example

```bash
ag-cli branch delete owner/repo feature/foo
ag-cli branch delete owner/repo feature/foo --yes
```


## ag-cli branch list

Usage: `ag-cli branch list [<owner>/<repo>] [flags]`

List remote branches

List remote branches

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output branches as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of branches to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli branch list owner/repo --limit 50
```


## ag-cli branch protection

Usage: `ag-cli branch protection`

Manage protected branch rules

List, view, create, update, and delete protected branch rules.

Rules may name an exact branch or contain an AtomGit wildcard pattern. Exact
rules take precedence over matching wildcard rules. AtomGit's API exposes only
push and merge allowlists; other web settings are not changed by this command.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli branch protection delete

Usage: `ag-cli branch protection delete [<owner>/<repo>] <branch-or-pattern> [flags]`

Delete a protected branch rule

Delete a protected branch rule

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip deletion confirmation | `false` | local |


## ag-cli branch protection list

Usage: `ag-cli branch protection list [<owner>/<repo>] [flags]`

List protected branch rules

List protected branch rules

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of protected branch rules to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli branch protection list owner/repo --limit 50
```


## ag-cli branch protection set

Usage: `ag-cli branch protection set [<owner>/<repo>] <branch-or-pattern> [flags]`

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
| `--merge` | Merge allowlist: develop, admin, maintainer, usernames, or empty | `` | local |
| `--push` | Push allowlist: develop, admin, maintainer, usernames, or empty | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation when updating an existing rule | `false` | local |

### Example

```bash
ag-cli branch protection set owner/repo main --push admin --merge admin
ag-cli branch protection set owner/repo main --push maintainer --merge maintainer
ag-cli branch protection set owner/repo "release/*" --push "develop;alice" --merge "develop;alice"
ag-cli branch protection set owner/repo main --push "" --yes
```


## ag-cli branch protection view

Usage: `ag-cli branch protection view [<owner>/<repo>] <branch-or-pattern>`

View a protected branch rule

View a protected branch rule

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli branch view

Usage: `ag-cli branch view [<owner>/<repo>] <branch>`

View a remote branch

View a remote branch

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli branch view owner/repo main
```


## ag-cli browse

Usage: `ag-cli browse [<number> | <path> | <commit-sha>] [flags]`

Open repositories, issues, pull requests, and more in the browser

Open repositories, issues, pull requests, and more in the browser

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-R, --repo` | Select another repository using the OWNER/REPO format | `` | local |
| `-a, --actions` | Open repository actions | `false` | local |
| `-b, --branch` | Select another branch by passing in the branch name | `` | local |
| `-c, --commit` | Select another commit by passing in the commit SHA, default is the last commit | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-n, --no-browser` | Print destination URL instead of opening the browser | `false` | local |
| `-r, --releases` | Open repository releases | `false` | local |
| `-s, --settings` | Open repository settings | `false` | local |
| `-w, --wiki` | Open repository wiki | `false` | local |


## ag-cli check-update

Usage: `ag-cli check-update`

Check for a newer AtomGit CLI release

> Deprecated: use "ag-cli update --check" instead

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit

Usage: `ag-cli commit`

Manage commits

List, view, compare, and inspect repository commits.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit comment

Usage: `ag-cli commit comment`

Manage commit comments

List, view, create, edit, and delete comments on repository commits.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit comment create

Usage: `ag-cli commit comment create [<owner>/<repo>] <sha> (--body <text> | --body-file <path-or->) [flags]`

Create a comment on a commit

Create a comment on a commit, identified by SHA (full or short) or branch name.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --body-file` | Read body text from file (use - for stdin) | `` | local |
| `-b, --body` | Comment body text | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit comment delete

Usage: `ag-cli commit comment delete [<owner>/<repo>] <comment-id> [flags]`

Delete a commit comment

Delete a commit comment you own. Asks for confirmation unless --yes is supplied.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |


## ag-cli commit comment edit

Usage: `ag-cli commit comment edit [<owner>/<repo>] <comment-id> (--body <text> | --body-file <path-or->) [flags]`

Edit a commit comment

Edit the body of a commit comment you own, replacing it with the new text.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --body-file` | Read new body text from file (use - for stdin) | `` | local |
| `-b, --body` | New comment body text | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit comment list

Usage: `ag-cli commit comment list [<owner>/<repo>] <ref> [flags]`

List comments on a commit

List comments on a commit, identified by SHA (full or short) or branch name.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output comments as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of comments to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit comment view

Usage: `ag-cli commit comment view [<owner>/<repo>] <comment-id> [flags]`

View a commit comment

View a single repository commit comment by ID.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output the comment as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit compare

Usage: `ag-cli commit compare [<owner>/<repo>] <base>...<head> [flags]`

Compare two commits, branches, or tags

Compare two commits, branches, or tags

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output comparison as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit diff

Usage: `ag-cli commit diff [<owner>/<repo>] <sha>`

Show a commit's diff

Show a commit's diff

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit list

Usage: `ag-cli commit list [<owner>/<repo>] [flags]`

List commits

List commits

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output commits as JSON | `false` | local |
| `--path` | Only list commits that touch the given file path | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Commit SHA or branch name to start from | `` | local |
| `--since` | Only list commits after this time (RFC 3339, e.g. 2024-11-08T16:25:44Z) | `` | local |
| `--until` | Only list commits before this time (RFC 3339, e.g. 2024-11-08T16:25:44Z) | `` | local |
| `-L, --limit` | Maximum number of commits to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit patch

Usage: `ag-cli commit patch [<owner>/<repo>] <sha>`

Show a commit's patch

Show a commit's patch

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli commit view

Usage: `ag-cli commit view [<owner>/<repo>] <sha> [flags]`

View a commit

View a commit

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output commit as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-w, --web` | Open a commit in the browser | `false` | local |


## ag-cli discussion

Usage: `ag-cli discussion`

View repository discussions

List AtomGit discussions for a repository

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli discussion list

Usage: `ag-cli discussion list [<owner>/<repo>] [flags]`

List repository discussions

List repository discussions

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output discussions as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of discussions to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli discussion view

Usage: `ag-cli discussion view [<owner>/<repo>] <number> [flags]`

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
| `--json` | Output the discussion as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli discussion view owner/repo 1
ag-cli discussion view owner/repo 1 --comments
ag-cli discussion view owner/repo 1 --comments --json
```


## ag-cli doctor

Usage: `ag-cli doctor [<owner>/<repo>] [flags]`

CLI health check: config, auth, and connectivity

Check local configuration without modifying it. Use --live for read-only API probes (30 second overall timeout). Missing optional capabilities are skipped. Failures return a nonzero exit status; warnings alone do not.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output a redacted health report as JSON | `false` | local |
| `--live` | Run read-only connectivity and authentication probes | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli doctor
ag-cli doctor --live
ag-cli doctor owner/repo --live --json
```


## ag-cli issue

Usage: `ag-cli issue`

Manage issues

Create, view, and manage issues.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli issue activity

Usage: `ag-cli issue activity [<owner>/<repo>] <number> [flags]`

List operation logs for an issue

List operation logs for an issue.

The API returns an unpaginated list; --limit caps the displayed entries in server order.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output entries as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of entries to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli issue activity owner/repo 42
ag-cli issue activity 42 --limit 100 --json
```


## ag-cli issue branches

Usage: `ag-cli issue branches [<owner>/<repo>] <number> [flags]`

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
| `--json` | Output branch names as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--remove` | Branch names to remove (repeatable) | `[]` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag-cli issue branches owner/repo 42
ag-cli issue branches owner/repo 42 --json
ag-cli issue branches owner/repo 42 --add feature/x
ag-cli issue branches owner/repo 42 --remove main --yes
```


## ag-cli issue close

Usage: `ag-cli issue close [<owner>/<repo>] <number>`

Close an issue

Close an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli issue comment

Usage: `ag-cli issue comment`

Manage issue comments

Create, view, edit, and delete issue comments.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli issue comment create

Usage: `ag-cli issue comment create [<owner>/<repo>] <number> [flags]`

Create a comment on an issue

Create a comment on an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --body-file` | Read body text from file | `` | local |
| `-b, --body` | Comment body text | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli issue comment delete

Usage: `ag-cli issue comment delete [<owner>/<repo>] <number> <comment-id> [flags]`

Delete a comment on an issue

Delete a comment on an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |


## ag-cli issue comment edit

Usage: `ag-cli issue comment edit [<owner>/<repo>] <number> <comment-id> [flags]`

Edit a comment on an issue

Edit a comment on an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --body` | New comment body text | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli issue comment view

Usage: `ag-cli issue comment view [<owner>/<repo>] <number>`

View all comments on an issue

View all comments on an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli issue create

Usage: `ag-cli issue create [<owner>/<repo>] [flags]`

Create an issue

Create an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--assignee` | Assign the issue to a user (login) | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --body-file` | Read issue body from file (use - for stdin) | `` | local |
| `-b, --body` | Issue body | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-t, --title` | Issue title | `` | local |

### Example

```bash
ag-cli issue create owner/repo --title "Bug report" --body "Description"
ag-cli issue create owner/repo --title "Bug report" --assignee alice
ag-cli issue create owner/repo --title "Bug report" --body-file description.md
ag-cli issue create owner/repo --title "Bug report" --body-file -
```


## ag-cli issue edit

Usage: `ag-cli issue edit [<owner>/<repo>] <number> [flags]`

Edit an issue

Edit an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--assignee` | Set the issue assignee (login) | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--remove-assignee` | Clear the issue assignee | `false` | local |
| `-F, --body-file` | Read the new issue body from a file | `` | local |
| `-b, --body` | New issue body | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-t, --title` | New issue title | `` | local |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag-cli issue edit owner/repo 42 --title "new title" --body "new body"
ag-cli issue edit owner/repo 42 --assignee alice
ag-cli issue edit owner/repo 42 --remove-assignee --yes
ag-cli issue edit owner/repo 42 --body-file description.md
```


## ag-cli issue history

Usage: `ag-cli issue history [<owner>/<repo>] <number> [flags]`

List modification history for an issue

List modification history for an issue.

The API returns an unpaginated list; --limit caps the displayed entries in server order. JSON includes both the creator and updater of each version.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output entries as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of entries to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli issue history owner/repo 42
ag-cli issue history 42 --limit 100 --json
```


## ag-cli issue label

Usage: `ag-cli issue label [<owner>/<repo>] <number> [<labels>] [flags]`

Add or remove labels on an issue

Add or remove labels on an issue.

Labels are comma-separated. Positional labels are treated as labels to add for
backward compatibility. Use --add or --remove to make the operation explicit.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--add` | Comma-separated labels to add | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--remove` | Comma-separated labels to remove | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli issue label owner/repo 42 "bug, help wanted"
ag-cli issue label owner/repo 42 --add "bug, help wanted"
ag-cli issue label owner/repo 42 --remove "priority/high"
```


## ag-cli issue list

Usage: `ag-cli issue list [<owner>/<repo>] [flags]`

List issues

List issues

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--assignee` | Filter by assignee: @me for issues assigned to you across all your repositories | `` | local |
| `--author` | Filter by author: @me for issues you created across all your repositories | `` | local |
| `--involved` | Filter by involvement: @me for issues you created or are assigned to across all your repositories | `` | local |
| `--json` | Output issues as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of issues to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-s, --state` | Filter by state: open, closed, all | `open` | local |


## ag-cli issue prs

Usage: `ag-cli issue prs [<owner>/<repo>] <number> [flags]`

List pull requests linked to an issue

List pull requests linked to an issue.

Concurrent updates to linked pull requests are not reflected until the next
request.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output linked pull requests as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli issue prs owner/repo 42
ag-cli issue prs owner/repo 42 --json
```


## ag-cli issue reactions

Usage: `ag-cli issue reactions [<owner>/<repo>] <number> [flags]`

List reactions on an issue

List reactions on an issue.

Fetch paginated reactions up to --limit. The API does not provide reaction timestamps.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output entries as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of entries to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli issue reactions owner/repo 42
ag-cli issue reactions 42 --limit 100 --json
```


## ag-cli issue reopen

Usage: `ag-cli issue reopen [<owner>/<repo>] <number>`

Reopen an issue

Reopen an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli issue view

Usage: `ag-cli issue view [<owner>/<repo>] <number> [flags]`

View an issue

View an issue

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output issue as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-w, --web` | Open an issue in the browser | `false` | local |


## ag-cli kanban

Usage: `ag-cli kanban`

View organization Kanban boards

List and inspect read-only organization Kanban boards and their Issue/Pull Request items.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli kanban list hust-open-atom-club
ag-cli kanban view hust-open-atom-club 1234567890
ag-cli kanban items hust-open-atom-club 1234567890 --json
```


## ag-cli kanban items

Usage: `ag-cli kanban items <owner> <kanban-id> [flags]`

List items on a Kanban board

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output Kanban items as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of Kanban items to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli kanban items hust-open-atom-club 1234567890
ag-cli kanban items hust-open-atom-club 1234567890 --limit 50 --json
```


## ag-cli kanban list

Usage: `ag-cli kanban list <owner> [flags]`

List organization Kanban boards

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output Kanban boards as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of Kanban boards to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli kanban list hust-open-atom-club
ag-cli kanban list hust-open-atom-club --limit 50 --json
```


## ag-cli kanban view

Usage: `ag-cli kanban view <owner> <kanban-id> [flags]`

View a Kanban board

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output the Kanban board as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli kanban view hust-open-atom-club 1234567890 --json
```


## ag-cli label

Usage: `ag-cli label`

Manage repository labels

List and manage repository labels.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli label create

Usage: `ag-cli label create [<owner>/<repo>] [flags]`

Create a repository label

Create a repository label. AtomGit API v5 accepts a name and color for label creation.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--color` | Label color in #RGB or #RRGGBB format | `` | local |
| `--name` | Label name | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli label create owner/repo --name bug --color "#ff0000"
```


## ag-cli label delete

Usage: `ag-cli label delete [<owner>/<repo>] <name> [flags]`

Delete a repository label

Delete a repository label from AtomGit.

By default, you will be prompted to confirm the deletion. Use --yes to skip
the confirmation prompt.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag-cli label delete owner/repo obsolete
ag-cli label delete owner/repo obsolete --yes
```


## ag-cli label edit

Usage: `ag-cli label edit [<owner>/<repo>] <name> [flags]`

Edit a repository label

Edit a repository label. AtomGit API v5 accepts a new name and color for label updates.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--color` | New label color in #RGB or #RRGGBB format | `` | local |
| `--name` | New label name | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli label edit owner/repo bug --name defect --color "#d73a4a"
```


## ag-cli label list

Usage: `ag-cli label list [<owner>/<repo>] [flags]`

List repository labels

List repository labels

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output labels as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of labels to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli label list owner/repo --limit 50
```


## ag-cli license

Usage: `ag-cli license`

License compliance checking

Check license compliance using openEuler compliance service.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli license check

Usage: `ag-cli license check <license>`

Check license compliance

Check if a license is compliant using openEuler compliance service.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli milestone

Usage: `ag-cli milestone`

Manage repository milestones

List, view, create, update, close, reopen, and delete repository milestones.

AtomGit requires title and due_on on every milestone PATCH. Edit, close, and
reopen read the current milestone first and preserve those required fields.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli milestone close

Usage: `ag-cli milestone close [<owner>/<repo>] <number>`

Close a repository milestone

Close a repository milestone

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli milestone create

Usage: `ag-cli milestone create [<owner>/<repo>] [flags]`

Create a repository milestone

Create a repository milestone

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--due-on` | Due date in YYYY-MM-DD format | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-d, --description` | Milestone description | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-t, --title` | Milestone title | `` | local |


## ag-cli milestone delete

Usage: `ag-cli milestone delete [<owner>/<repo>] <number> [flags]`

Delete a repository milestone

Permanently delete a milestone. This differs from closing it and requires confirmation unless --yes is used.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |


## ag-cli milestone edit

Usage: `ag-cli milestone edit [<owner>/<repo>] <number> [flags]`

Edit a repository milestone

Edit a repository milestone

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--due-on` | New due date in YYYY-MM-DD format | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-d, --description` | New milestone description; pass an empty value to clear | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-t, --title` | New milestone title | `` | local |


## ag-cli milestone list

Usage: `ag-cli milestone list [<owner>/<repo>] [flags]`

List repository milestones

List repository milestones

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--direction` | Sort direction: asc, desc | `asc` | local |
| `--json` | Output milestones as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--sort` | Sort milestones by created or due_on | `due_on` | local |
| `-L, --limit` | Maximum number of milestones to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-s, --state` | Filter by state: open, closed, all | `open` | local |

### Example

```bash
ag-cli milestone list owner/repo --state all --limit 50
```


## ag-cli milestone reopen

Usage: `ag-cli milestone reopen [<owner>/<repo>] <number>`

Reopen a repository milestone

Reopen a repository milestone

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli milestone view

Usage: `ag-cli milestone view [<owner>/<repo>] <number> [flags]`

View a repository milestone

View a repository milestone

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output milestone as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli notification

Usage: `ag-cli notification`

Manage repository notifications

List repository notifications and mark them as read.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli notification list

Usage: `ag-cli notification list [<owner>/<repo>] [flags]`

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
| `--json` | Output notifications as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--since` | Only list notifications updated at or after this RFC 3339 timestamp | `` | local |
| `--type` | Only list notifications of this type (for example merge_requests_open) | `` | local |
| `--unread` | Only list unread notifications | `false` | local |
| `-L, --limit` | Maximum number of notifications to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli notification list owner/repo --limit 20
ag-cli notification list --unread --json
ag-cli notification list owner/repo --type issue_open --since 2026-08-01T00:00:00Z
```


## ag-cli notification mark-read

Usage: `ag-cli notification mark-read [<owner>/<repo>] [<notification-id>...] [flags]`

Mark repository notifications as read

Mark notifications for a repository as read.

Pass one or more notification IDs (as shown by "ag-cli notification list") to
mark exactly those notifications, or pass --all to mark every unread
notification in the repository. --all asks for confirmation unless --yes is
supplied.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--all` | Mark every unread notification in the repository | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip the --all confirmation prompt | `false` | local |

### Example

```bash
ag-cli notification mark-read owner/repo 292ecbec857e4f27b426d66f2157938c
ag-cli notification mark-read --all --yes
```


## ag-cli org

Usage: `ag-cli org`

Manage organizations

List organizations associated with your AtomGit account and inspect organization details, members, repositories, and Actions runner groups.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli org list

Usage: `ag-cli org list [flags]`

List organizations for the authenticated user

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output organizations as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of organizations to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli org list
ag-cli org list --limit 100
ag-cli org list --json
```


## ag-cli org members

Usage: `ag-cli org members <org> [flags]`

List organization members

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output members as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of members to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli org members my-organization
ag-cli org members my-organization --limit 100
ag-cli org members my-organization --json
```


## ag-cli org repos

Usage: `ag-cli org repos <org> [flags]`

List organization repositories

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output repositories as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of repositories to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli org repos my-organization
ag-cli org repos my-organization --limit 100
ag-cli org repos my-organization --json
```


## ag-cli org runner-group

Usage: `ag-cli org runner-group`

Inspect organization Actions runner groups

Inspect organization-level AtomGit Actions runner groups and their read-only associations.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli org runner-group list my-organization
ag-cli org runner-group view my-organization group-id
ag-cli org runner-group runners my-organization group-id --json
```


## ag-cli org runner-group list

Usage: `ag-cli org runner-group list <org> [flags]`

List organization runner groups

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output runner groups as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of runner groups to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli org runner-group list my-organization
ag-cli org runner-group list my-organization --limit 100
ag-cli org runner-group list my-organization --json
```


## ag-cli org runner-group namespaces

Usage: `ag-cli org runner-group namespaces <org> <group-id> [flags]`

List repositories that can use an organization runner group

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output shared namespaces as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of shared namespaces to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli org runner-group runner-sets

Usage: `ag-cli org runner-group runner-sets <org> <group-id> [flags]`

List Kubernetes runner sets in an organization runner group

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output runner sets as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of runner sets to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli org runner-group runners

Usage: `ag-cli org runner-group runners <org> <group-id> [flags]`

List host runners in an organization runner group

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output runners as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of runners to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli org runner-group view

Usage: `ag-cli org runner-group view <org> <group-id> [flags]`

View an organization runner group

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output runner group as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli org runner-group view my-organization group-id
ag-cli org runner-group view my-organization group-id --json
```


## ag-cli org view

Usage: `ag-cli org view <org> [flags]`

View an organization

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output organization as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli org view my-organization
ag-cli org view my-organization --json
```


## ag-cli pr

Usage: `ag-cli pr`

Manage pull requests

Create, view, and checkout pull requests.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr activity

Usage: `ag-cli pr activity [<owner>/<repo>] <number> [flags]`

List the operation log of a pull request

List the operation log of a pull request.

The operate_logs endpoint supports pagination; --limit caps how many entries
are fetched across pages.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output activity as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of activity entries to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli pr activity owner/repo 42
ag-cli pr activity owner/repo 42 --limit 50 --json
```


## ag-cli pr checkout

Usage: `ag-cli pr checkout [<owner>/<repo>] <number> [flags]`

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
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--recurse-submodules` | Update submodules after checkout | `false` | local |
| `-b, --branch` | Local branch name (default: PR head branch name) | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
# Check out PR #42, inferring the repository from git remote
ag-cli pr checkout 42

# Check out PR #42 from a specific repository
ag-cli pr checkout owner/repo 42

# Check out to a custom branch name
ag-cli pr checkout 42 --branch review-fix

# Force checkout, discarding safety checks
ag-cli pr checkout 42 --force

# Check out in detached HEAD mode and update submodules
ag-cli pr checkout 42 --detach --recurse-submodules
```


## ag-cli pr checks

Usage: `ag-cli pr checks [<owner>/<repo>] <number> [flags]`

Show CI checks for a pull request's current head commit

Show AtomGit Actions runs for a pull request's current head commit. This command does not infer or report required-check semantics.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-i, --interval` | Polling interval when using --watch | `10s` | local |
| `-w, --watch` | Watch checks until they reach a terminal state | `false` | local |

### Example

```bash
ag-cli pr checks owner/repo 42
ag-cli pr checks 42 --watch
ag-cli pr checks owner/repo 42 --watch --interval 5s
```


## ag-cli pr close

Usage: `ag-cli pr close [<owner>/<repo>] <number>`

Close a pull request

Close a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr comment

Usage: `ag-cli pr comment`

Manage pull request comments

Create, view, edit, delete, and reply to pull request comments.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr comment create

Usage: `ag-cli pr comment create [<owner>/<repo>] <number> [flags]`

Create a comment on a pull request

Create a comment on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --body-file` | Read body text from file | `` | local |
| `-b, --body` | Comment body text | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr comment delete

Usage: `ag-cli pr comment delete [<owner>/<repo>] <number> <comment-id> [flags]`

Delete a comment on a pull request

Delete a comment on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |


## ag-cli pr comment edit

Usage: `ag-cli pr comment edit [<owner>/<repo>] <number> <comment-id> [flags]`

Edit a comment on a pull request

Edit a comment on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --body` | New comment body text | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr comment reply

Usage: `ag-cli pr comment reply [<owner>/<repo>] <number> <discussion-id> [flags]`

Reply to a comment thread on a pull request

Reply to a comment thread on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --body` | Reply body text | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr comment view

Usage: `ag-cli pr comment view [<owner>/<repo>] <number>`

View all comments on a pull request

View all comments on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr commits

Usage: `ag-cli pr commits [<owner>/<repo>] <number> [flags]`

List commits in a pull request

List commits in a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output commits as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of commits to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr create

Usage: `ag-cli pr create [<owner>/<repo>] [flags]`

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
| `--draft` | Mark pull request as a draft | `false` | local |
| `--head` | Head branch | `` | local |
| `--label` | Label name (repeat for multiple labels) | `[]` | local |
| `--milestone` | Milestone number or exact title | `` | local |
| `--prune-branch` | Delete the source branch after the PR is merged | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--reviewer` | Approval reviewer login (repeat for multiple users) | `[]` | local |
| `--tester` | Tester login (repeat for multiple users) | `[]` | local |
| `-F, --body-file` | Read PR body from file (use - for stdin) | `` | local |
| `-b, --body` | PR body | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-t, --title` | PR title | `` | local |

### Example

```bash
ag-cli pr create owner/repo --title "Fix bug" --body "Description" --base main --head feature
ag-cli pr create owner/repo --title "Fix bug" --body "Description" --base main --head feature --draft
ag-cli pr create owner/repo --title "Fix bug" --body-file description.md --base main --head feature
ag-cli pr create owner/repo --title "Fix bug" --body-file - --base main --head feature
ag-cli pr create owner/repo --title "Fix bug" --head feature --prune-branch
```


## ag-cli pr diff

Usage: `ag-cli pr diff [<owner>/<repo>] <number>`

Show diff of a pull request

Show diff of a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr edit

Usage: `ag-cli pr edit [<owner>/<repo>] <number> [flags]`

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
| `--milestone` | Milestone number, exact title, or 'none' to clear | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--remove-assignee` | Assignee login to remove (repeat for multiple users) | `[]` | local |
| `--remove-label` | Label name to remove (repeat for multiple labels) | `[]` | local |
| `--remove-reviewer` | Approval reviewer login to remove (repeat for multiple users) | `[]` | local |
| `--remove-tester` | Tester login to remove (repeat for multiple users) | `[]` | local |
| `-F, --body-file` | Read new PR body from file (use - for stdin) | `` | local |
| `-b, --body` | New PR body | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-t, --title` | New PR title | `` | local |

### Example

```bash
ag-cli pr edit owner/repo 123 --title "Updated title"
ag-cli pr edit owner/repo 123 --body "Updated description"
ag-cli pr edit owner/repo 123 --body-file description.md
ag-cli pr edit owner/repo 123 --body-file -
```


## ag-cli pr files

Usage: `ag-cli pr files [<owner>/<repo>] <number> [flags]`

List files changed in a pull request

List files changed in a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output files as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr history

Usage: `ag-cli pr history [<owner>/<repo>] <number> [flags]`

List the modification history of a pull request

List the modification history of a pull request.

The modify_history endpoint does not support pagination: the full response is
fetched and then truncated to --limit.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output history as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of history entries to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli pr history owner/repo 42
ag-cli pr history owner/repo 42 --limit 50 --json
```


## ag-cli pr issues

Usage: `ag-cli pr issues [<owner>/<repo>] <pr_number>`

View linked issues of a pull request

View all issues linked to a pull request.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr link-issues

Usage: `ag-cli pr link-issues [<owner>/<repo>] <pr_number> [flags]`

Link issues to a pull request

Link one or more issues to a pull request.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-i, --issue` | Issue number to link (can be specified multiple times) | `[]` | local |


## ag-cli pr list

Usage: `ag-cli pr list [<owner>/<repo>] [flags]`

List pull requests

List pull requests

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--assignee` | Filter by assignee: @me for PRs assigned to you across all your repositories | `` | local |
| `--author` | Filter by author: @me for PRs you created across all your repositories | `` | local |
| `--json` | Output pull requests as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--review-needed` | Filter by requested reviewer: @me for PRs that need your review across all your repositories | `` | local |
| `--review-requested` | Filter by requested approver: @me for PRs that need your approval across all your repositories | `` | local |
| `-L, --limit` | Maximum number of PRs to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-s, --state` | Filter by state: open, closed, locked, merged, all | `open` | local |


## ag-cli pr merge

Usage: `ag-cli pr merge [<owner>/<repo>] <number> [flags]`

Merge a pull request

Merge a pull request.

By default, ag-cli creates a merge commit. Use --rebase to rebase the commits onto the base branch.


When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--admin` | Use administrator privileges to merge a pull request that does not meet requirements | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --body` | Body text for the merge commit | `` | local |
| `-d, --delete-branch` | Delete the source branch after merge | `false` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-r, --rebase` | Rebase the commits onto the base branch | `false` | local |
| `-s, --squash` | Squash the commits into one commit | `false` | local |
| `-t, --subject` | Subject text for the merge commit | `` | local |


## ag-cli pr reactions

Usage: `ag-cli pr reactions [<owner>/<repo>] <number> [flags]`

List reactions on a pull request

List reactions on a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output reactions as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of reactions to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli pr reactions owner/repo 42
ag-cli pr reactions owner/repo 42 --limit 50 --json
```


## ag-cli pr reopen

Usage: `ag-cli pr reopen [<owner>/<repo>] <number>`

Reopen a pull request

Reopen a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli pr review

Usage: `ag-cli pr review <owner>/<repo> <number> [flags]`

Approve a pull request review

Approve a pull request using AtomGit's formal review API.

AtomGit currently exposes approval as the only review action. Use ag-cli pr comment
create to leave an ordinary comment; request-changes reviews are not supported
by the public API.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--approve` | Approve the pull request | `false` | local |
| `--force` | Force approval as a repository administrator | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli pr review owner/repo 42 --approve
ag-cli pr review owner/repo 42 --approve --force
```


## ag-cli pr unlink-issues

Usage: `ag-cli pr unlink-issues [<owner>/<repo>] <pr_number> [flags]`

Unlink issues from a pull request

Unlink one or more issues from a pull request.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-i, --issue` | Issue number to unlink (can be specified multiple times) | `[]` | local |


## ag-cli pr view

Usage: `ag-cli pr view [<owner>/<repo>] <number> [flags]`

View a pull request

View a pull request

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output pull request as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-w, --web` | Open a pull request in the browser | `false` | local |


## ag-cli release

Usage: `ag-cli release`

Manage repository releases

List, view, create, and edit repository releases on AtomGit, and upload or download release attachments.

The repository's automated release pipeline validates tags and artifacts, then
uses these primitives through "make publish VERSION=vX.Y.Z NOTES_FILE=notes.md".

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli release list owner/repo
ag-cli release view owner/repo v1.0.0
ag-cli release create owner/repo v1.0.0 --name "Version 1.0.0" --body "Release notes"
ag-cli release upload owner/repo v1.0.0 ./dist/app.tar.gz
```


## ag-cli release create

Usage: `ag-cli release create [<owner>/<repo>] <tag> [flags]`

Create a release

Create a release for a repository identified by its tag. A non-empty release body is required by the AtomGit API.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--prerelease` | Mark release as a prerelease (release_status=pre) | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--target` | Target commitish (branch or SHA) | `` | local |
| `-F, --body-file` | Path to file containing release body | `` | local |
| `-b, --body` | Release body text | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-n, --name` | Release name (defaults to tag) | `` | local |

### Example

```bash
ag-cli release create owner/repo v1.0.0 --name "First" --body "Initial release"
ag-cli release create owner/repo v1.0.0-rc --prerelease --body-file notes.md
```


## ag-cli release download

Usage: `ag-cli release download [<owner>/<repo>] <tag> <asset> [flags]`

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
| `--overwrite` | Replace an existing local file at --output | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--timeout` | Maximum attachment transfer time (0 disables the limit) | `30m0s` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-o, --output` | Local file path to write the attachment to (required) | `` | local |

### Example

```bash
ag-cli release download owner/repo v1.0.0 app.tar.gz -o ./dist/app.tar.gz
ag-cli release download owner/repo v1.0.0 app.tar.gz --output ./existing.tar.gz --overwrite
```


## ag-cli release edit

Usage: `ag-cli release edit [<owner>/<repo>] <tag> [flags]`

Edit a release

Edit an existing release identified by its tag.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--latest` | Set release status to latest (release_status=latest) | `false` | local |
| `--prerelease` | Set release status to prerelease (release_status=pre) | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --body-file` | Path to file containing new release body | `` | local |
| `-b, --body` | New release body text | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-n, --name` | New release name | `` | local |

### Example

```bash
ag-cli release edit owner/repo v1.0.0 --name "First Release"
ag-cli release edit owner/repo v1.0.0 --latest --body-file notes.md
ag-cli release edit owner/repo v1.0.0-rc --prerelease
```


## ag-cli release list

Usage: `ag-cli release list [<owner>/<repo>] [flags]`

List repository releases

List releases for a repository, ordered most recent first.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output releases as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of releases to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli release list owner/repo --limit 50
```


## ag-cli release upload

Usage: `ag-cli release upload [<owner>/<repo>] <tag> <file> [flags]`

Upload an attachment to a release

Upload a local file as an attachment to an existing release identified by its tag.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--overwrite` | Delete an existing attachment with the same name before uploading | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--skip-existing` | Do nothing and report success if an attachment with the same name already exists | `false` | local |
| `--timeout` | Maximum attachment transfer time (0 disables the limit) | `30m0s` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-n, --name` | Remote attachment name (defaults to the local file's base name) | `` | local |

### Example

```bash
ag-cli release upload owner/repo v1.0.0 ./dist/app.tar.gz
ag-cli release upload owner/repo v1.0.0 ./build/app.zip --name app-v1.zip
ag-cli release upload owner/repo v1.0.0 ./new.tar.gz --overwrite
ag-cli release upload owner/repo v1.0.0 ./existing.tar.gz --skip-existing
```


## ag-cli release view

Usage: `ag-cli release view [<owner>/<repo>] <tag> [flags]`

View a release by tag

Show details of a single release identified by its tag.
Use --json to output one JSON object with release metadata, body, and assets.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output release details as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli release view owner/repo v1.0.0
ag-cli release view owner/repo v1.0.0 --json
```


## ag-cli repo

Usage: `ag-cli repo`

Manage repositories

Create, clone, edit, fork, sync, transfer, view, browse contents, and manage repository collaborators and webhooks. Use `ag-cli repo fork list` to inspect existing forks; `ag-cli repo fork` creates a fork.

For repository-scoped commands, OWNER/REPO may be omitted and inferred from the current Git repository.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo clone

Usage: `ag-cli repo clone <repository> [<directory>] [flags]`

Clone a repository

Clone a repository from AtomGit.

The repository argument can be:
- Full URL: https://atomgit.com/owner/repo
- Owner/repo format: owner/repo
- Just repo name (uses current user as owner)

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --branch` | Clone specific branch | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
# Clone using full URL
ag-cli repo clone https://atomgit.com/shinwell_hu/my-project

# Clone using owner/repo format
ag-cli repo clone shinwell_hu/my-project

# Clone to specific directory
ag-cli repo clone shinwell_hu/my-project my-project-local

# Clone specific branch
ag-cli repo clone shinwell_hu/my-project --branch develop
```


## ag-cli repo collaborator

Usage: `ag-cli repo collaborator`

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
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo collaborator add

Usage: `ag-cli repo collaborator add [<owner>/<repo>] <username> [flags]`

Add a direct repository collaborator

Add a direct repository collaborator

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-p, --permission` | Permission: pull, push, or admin | `push` | local |

### Example

```bash
ag-cli repo collaborator add owner/repo octocat --permission push
```


## ag-cli repo collaborator edit

Usage: `ag-cli repo collaborator edit [<owner>/<repo>] <username> [flags]`

Update a direct repository collaborator's permission

Update a direct repository collaborator's permission

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-p, --permission` | Permission: pull, push, or admin | `` | local |
| `-y, --yes` | Skip confirmation for permission reductions | `false` | local |

### Example

```bash
ag-cli repo collaborator edit owner/repo octocat --permission pull
```


## ag-cli repo collaborator list

Usage: `ag-cli repo collaborator list [<owner>/<repo>] [flags]`

List repository collaborators

List accepted repository collaborators. Pending invitations are not exposed by AtomGit API v5 and cannot be included in this listing.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output collaborators as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of collaborators to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo collaborator list owner/repo --limit 50
```


## ag-cli repo collaborator remove

Usage: `ag-cli repo collaborator remove [<owner>/<repo>] <username> [flags]`

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
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag-cli repo collaborator remove owner/repo octocat --yes
```


## ag-cli repo collaborator view

Usage: `ag-cli repo collaborator view [<owner>/<repo>] <username> [flags]`

View a repository collaborator's effective permission

View a repository collaborator's effective permission

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output collaborator as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo collaborator view owner/repo octocat
```


## ag-cli repo content

Usage: `ag-cli repo content`

Browse repository contents

List directories and view files without modifying repository contents.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo content list

Usage: `ag-cli repo content list [<owner>/<repo>] [<path>] [flags]`

List a repository directory

List a repository directory without modifying it. With no arguments, the repository is inferred and its root is listed. A single argument is a path in the inferred repository; use OWNER/REPO . to list the root of an explicit repository.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output the complete API directory array as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Branch, tag, or commit identifier | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo content list
ag-cli repo content list docs
ag-cli repo content list owner/repo .
ag-cli repo content list owner/repo docs/guides --ref v1.0.0 --json
```


## ag-cli repo content view

Usage: `ag-cli repo content view [<owner>/<repo>] <path> [flags]`

View a repository file

View a repository file

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output the complete API file object as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Branch, tag, or commit identifier | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo content view README.md
ag-cli repo content view owner/repo src/main.go --ref dev
ag-cli repo content view owner/repo README.md --json
```


## ag-cli repo create

Usage: `ag-cli repo create <repo> | <owner>/<repo> [flags]`

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
| `--private` | Make the repository private | `false` | local |
| `--public` | Make the repository public | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-c, --clone` | Clone the repository after creation | `false` | local |
| `-d, --description` | Description of the repository | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
# Create a new private repository under your account
ag-cli repo create my-project

# Create a public repository and clone it
ag-cli repo create my-project --public --clone

# Create a repository in an organization
ag-cli repo create my-org/my-project --public
```


## ag-cli repo delete

Usage: `ag-cli repo delete [<repository>] [flags]`

Delete a repository

Delete a repository from AtomGit.

This command permanently deletes a repository. This action cannot be undone.

By default, you will be prompted to confirm the deletion. Use --yes to skip
the confirmation prompt.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
# Delete a repository (with confirmation)
ag-cli repo delete my-project

# Delete a repository without confirmation
ag-cli repo delete my-project --yes

# Delete a repository in an organization
ag-cli repo delete my-org/my-project --yes
```


## ag-cli repo edit

Usage: `ag-cli repo edit [<owner>/<repo>] [flags]`

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
| `--name` | New repository name (does not change the repository path) | `` | local |
| `--private` | Make the repository private | `false` | local |
| `--public` | Make the repository public | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--visibility` | New visibility: public or private | `` | local |
| `-d, --description` | New repository description | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation for name or visibility changes | `false` | local |

### Example

```bash
# Update the current Git repository
ag-cli repo edit --description "New description"

# Update an explicitly selected repository
ag-cli repo edit owner/repo --description "New description"

# Clear a description without changing other settings
ag-cli repo edit owner/repo --description ""

# Update several settings
ag-cli repo edit owner/repo --name "New name" --default-branch main --visibility private

# Skip confirmation for a visibility update
ag-cli repo edit owner/repo --public --yes
```


## ag-cli repo fork

Usage: `ag-cli repo fork [<owner>/<repo>] [flags]`

Fork a repository

Fork a repository on AtomGit.

Creates a fork of the specified repository under your account or an organization.

By default, the fork will have the same visibility as the original repository.
Use --private or --public to override.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--private` | Make the forked repository private | `false` | local |
| `--public` | Make the forked repository public | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-c, --clone` | Clone the forked repository | `false` | local |
| `-d, --description` | Description for the forked repository | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-n, --name` | Name for the forked repository | `` | local |


## ag-cli repo fork list

Usage: `ag-cli repo fork list [<owner>/<repo>] [flags]`

List forks of a repository

List existing forks of a repository.

This is read-only and is separate from `ag-cli repo fork`, which creates a new fork.
The repository can be supplied explicitly or inferred from the current Git repository.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output forks as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of forks to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo fork list owner/repo
ag-cli repo fork list owner/repo --limit 100 --json
ag-cli repo fork list
```


## ag-cli repo insights

Usage: `ag-cli repo insights`

Inspect repository activity and statistics

Inspect read-only repository languages, contributor statistics, events, watchers, stargazers, and download statistics.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo insights contributors

Usage: `ag-cli repo insights contributors [<owner>/<repo>] [flags]`

List repository contributor statistics

List repository contributor statistics

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output contributors as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of contributors to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo insights downloads

Usage: `ag-cli repo insights downloads [<owner>/<repo>] [flags]`

Show repository download statistics

Show repository download statistics

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output download statistics as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo insights events

Usage: `ag-cli repo insights events [<owner>/<repo>] [flags]`

List repository activity events

List repository activity events

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output events as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of events to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo insights languages

Usage: `ag-cli repo insights languages [<owner>/<repo>] [flags]`

Show repository language percentages

Show repository language percentages

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output languages as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo insights stargazers

Usage: `ag-cli repo insights stargazers [<owner>/<repo>] [flags]`

List repository stargazers

List repository stargazers

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output stargazers as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of stargazers to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo insights watchers

Usage: `ag-cli repo insights watchers [<owner>/<repo>] [flags]`

List repository watchers

List repository watchers

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output watchers as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of watchers to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo list

Usage: `ag-cli repo list [<owner>] [flags]`

List repositories

List repositories for the authenticated user, a specified user, or an organization. A specified owner is checked as a user first and retried as an organization only when the user is not found.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output repositories as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of repositories to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo list
ag-cli repo list alice
ag-cli repo list my-organization --limit 100
ag-cli repo list alice --json
```


## ag-cli repo mirror

Usage: `ag-cli repo mirror`

Inspect repository remote mirrors

Inspect configured push mirrors and repository mirror state. These commands are read-only and do not synchronize local Git remotes.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo mirror list

Usage: `ag-cli repo mirror list [<owner>/<repo>] [flags]`

List configured push remote mirrors

List configured push remote mirrors. Destinations and returned messages are sanitized before text or JSON output.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output push remote mirrors as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of push remote mirrors to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo mirror list owner/repo
ag-cli repo mirror list owner/repo --limit 100 --json
ag-cli repo mirror list
```


## ag-cli repo mirror view

Usage: `ag-cli repo mirror view [<owner>/<repo>] [flags]`

View repository remote mirror state

View repository remote mirror state. Destinations and returned errors are sanitized before text or JSON output.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output repository remote mirror state as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo mirror view owner/repo
ag-cli repo mirror view --json
```


## ag-cli repo policy

Usage: `ag-cli repo policy`

View and edit repository policy settings

Manage permission mode, code-review defaults, and pull-request settings.
Push rules and branch/tag protection are managed separately.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo policy edit

Usage: `ag-cli repo policy edit [<owner>/<repo>] [flags]`

Edit one repository policy section

Send only explicitly supplied settings. False, zero, and empty strings are preserved.
All changes require confirmation unless --yes is supplied.
Code-review assignees/testers are usernames; pull-request approver/tester IDs are IDs.
Permission mode must be 1 (inherited) or 2 (independent).
Approval-required-reviewers must be 0..5; other counts must be nonnegative.
Merge-method: merge, rebase_merge, ff. Merged-commit-author: merged_by, created_by.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--add-notes-after-merged` | Allow review and comments after merging (pull-request section) | `false` | local |
| `--approval-approver-ids` | Comma-separated approver user IDs (empty clears) (pull-request section) | `` | local |
| `--approval-required-approvers` | Minimum approver count (nonnegative) (pull-request section) | `0` | local |
| `--approval-required-reviewers` | Minimum reviewer count: 0 disables, otherwise 1..5 (pull-request section) | `0` | local |
| `--approval-required-reviewers-enable` | Enable the minimum reviewer gate (pull-request section) | `false` | local |
| `--approval-required-testers` | Minimum tester count (nonnegative) (pull-request section) | `0` | local |
| `--approval-tester-ids` | Comma-separated tester user IDs (empty clears) (pull-request section) | `` | local |
| `--assignees` | Comma-separated approver usernames (empty clears) | `` | local |
| `--assignees-number` | Minimum approver count (0 disables) | `0` | local |
| `--auto-squash-merge` | Enable squash by default for new pull requests (pull-request section) | `false` | local |
| `--can-force-merge` | Allow administrators to force merge (pull-request section) | `false` | local |
| `--can-reopen` | Allow reopening closed pull requests (pull-request section) | `false` | local |
| `--close-issue-when-mr-merged` | Select closing linked issues by default (pull-request section) | `false` | local |
| `--delete-source-branch-when-merged` | Delete the source branch by default after merging (pull-request section) | `false` | local |
| `--disable-merge-by-self` | Prevent authors from merging their own pull requests (pull-request section) | `false` | local |
| `--disable-squash-merge` | Disable squash merging (pull-request section) | `false` | local |
| `--forbidden-pr-related-issue-closed` | Disable the option to close linked issues after merging (pull-request section) | `false` | local |
| `--is-allow-lite-merge-request` | Enable lightweight pull requests (pull-request section) | `false` | local |
| `--is-check-cla` | Require CLA validation (pull-request section) | `false` | local |
| `--json` | Output the submitted fields as JSON | `false` | local |
| `--lite-merge-request-prefix-title` | Lightweight pull request title prefix (empty clears) (pull-request section) | `` | local |
| `--mark-auto-merged-mr-as-closed` | Mark automatically merged pull requests as closed (pull-request section) | `false` | local |
| `--merge-method` | Merge method: merge, rebase_merge, or ff (pull-request section) | `` | local |
| `--merged-commit-author` | Merge commit author: merged_by or created_by (pull-request section) | `` | local |
| `--mode` | Permission mode: 1 (inherited), 2 (independent) | `0` | local |
| `--only-allow-merge-if-all-discussions-are-resolved` | Require all review discussions to be resolved (pull-request section) | `false` | local |
| `--only-allow-merge-if-pipeline-succeeds` | Require a successful pipeline before merging (pull-request section) | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--section` | Section: permission, code-review, or pull-request (required) | `` | local |
| `--squash-merge-with-no-merge-commit` | Do not create a merge commit for squash merges (pull-request section) | `false` | local |
| `--testers` | Comma-separated tester usernames (empty clears) | `` | local |
| `--testers-number` | Minimum tester count (0 disables) | `0` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip update confirmation | `false` | local |

### Example

```bash
ag-cli repo policy edit owner/repo --section permission --mode 2 --yes
ag-cli repo policy edit --section code-review --assignees alice,bob --testers-number 0
ag-cli repo policy edit owner/repo --section pull-request --can-force-merge=false --yes
```


## ag-cli repo policy view

Usage: `ag-cli repo policy view [<owner>/<repo>] [flags]`

View repository policy settings

View all three sections, or select one with --section.
Code-review defaults are read from pull_request_settings, not GET /reviewer.
JSON uses repository and settings keys; unavailable optional fields are null.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output policy settings as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--section` | Section: permission, code-review, or pull-request (default: all) | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo policy view owner/repo
ag-cli repo policy view --section permission --json
```


## ag-cli repo push-rule

Usage: `ag-cli repo push-rule`

Manage repository push rules

View and edit repository-wide push rules.

These rules cover signed commits, commit message validation, maximum file
size, owner exemptions, and force pushes. Branch and tag protection rules are
managed separately.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo push-rule edit

Usage: `ag-cli repo push-rule edit [<owner>/<repo>] [flags]`

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
| `--json` | Output the updated fields as JSON | `false` | local |
| `--max-file-size` | Maximum committed file size in MB (0 disables the limit) | `0` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--reject-not-signed-by-gpg` | Require commits to have verified GPG signatures | `false` | local |
| `--skip-rule-for-owner` | Exempt repository administrators from applicable push rules | `false` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip update confirmation | `false` | local |

### Example

```bash
ag-cli repo push-rule edit owner/repo --deny-force-push --yes
ag-cli repo push-rule edit --commit-message-regex '^(feat|fix): '
ag-cli repo push-rule edit owner/repo --reject-not-signed-by-gpg=false --max-file-size 0
```


## ag-cli repo push-rule view

Usage: `ag-cli repo push-rule view [<owner>/<repo>] [flags]`

View repository push rules

View repository push rules

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output push rules as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo push-rule view owner/repo
ag-cli repo push-rule view --json
```


## ag-cli repo read-dir

Usage: `ag-cli repo read-dir [<owner>/<repo>] <path> [flags]`

List contents of a repository directory

> Deprecated: use 'ag-cli repo content list' instead

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output directory entries as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Branch, tag, or commit identifier | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo read-file

Usage: `ag-cli repo read-file [<owner>/<repo>] <path> [flags]`

Read a file from a repository

> Deprecated: use 'ag-cli repo content view' instead

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output file content as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Branch, tag, or commit identifier | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo sync

Usage: `ag-cli repo sync [<owner>/<repo>] [flags]`

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
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-b, --branch` | Branch to synchronize (defaults to the repository default branch) | `` | local |
| `-f, --force` | Overwrite divergent commits after confirmation | `false` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation when --force is used | `false` | local |

### Example

```bash
# Synchronize the current repository's default branch
ag-cli repo sync

# Synchronize an explicit branch of a fork
ag-cli repo sync owner/fork --branch develop

# Force synchronization after interactive confirmation
ag-cli repo sync owner/fork --branch develop --force

# Force synchronization non-interactively
ag-cli repo sync owner/fork --branch develop --force --yes
```


## ag-cli repo transfer

Usage: `ag-cli repo transfer [<owner>/<repo>] --to <organization> [flags]`

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
| `--password-stdin` | Read the organization-transfer password from standard input (requires --yes) | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--to` | Destination organization namespace (required) | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip the confirmation prompt | `false` | local |

### Example

```bash
ag-cli repo transfer owner/repo --to target-organization
ag-cli repo transfer owner/repo --to target-organization --yes
printf '%s\n' "$PASSWORD" | ag-cli repo transfer source-organization/repo --to target-organization --yes --password-stdin
```


## ag-cli repo view

Usage: `ag-cli repo view [<owner>/<repo>] [flags]`

View a repository

View a repository

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output repository as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-w, --web` | Open a repository in the browser | `false` | local |


## ag-cli repo webhook

Usage: `ag-cli repo webhook`

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
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli repo webhook create

Usage: `ag-cli repo webhook create [<owner>/<repo>] [flags]`

Create a repository webhook

Create a repository webhook

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--encryption` | Secret mode: password or signature | `` | local |
| `--events` | Comma-separated events: push, tag-push, issues, note, merge-requests | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--secret-env` | Read the webhook secret from this environment variable | `` | local |
| `--secret-file` | Read the webhook secret from a file | `` | local |
| `--secret-stdin` | Read the webhook secret from standard input | `false` | local |
| `--url` | Webhook target HTTP(S) URL | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo webhook create owner/repo --url https://example.com/hook --events push,issues --secret-env WEBHOOK_SECRET
```


## ag-cli repo webhook delete

Usage: `ag-cli repo webhook delete [<owner>/<repo>] <id> [flags]`

Delete a repository webhook

Delete a repository webhook

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag-cli repo webhook delete owner/repo 42 --yes
```


## ag-cli repo webhook edit

Usage: `ag-cli repo webhook edit [<owner>/<repo>] <id> [flags]`

Edit a repository webhook

Edit a repository webhook

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--encryption` | Secret mode: password or signature | `` | local |
| `--events` | Replace events; use none to disable all events | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--secret-env` | Read the webhook secret from this environment variable | `` | local |
| `--secret-file` | Read the webhook secret from a file | `` | local |
| `--secret-stdin` | Read the webhook secret from standard input | `false` | local |
| `--url` | New webhook target HTTP(S) URL | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo webhook edit owner/repo 42 --events push,merge-requests
```


## ag-cli repo webhook list

Usage: `ag-cli repo webhook list [<owner>/<repo>] [flags]`

List repository webhooks

List repository webhooks

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output webhooks as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of webhooks to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo webhook list owner/repo --limit 50
```


## ag-cli repo webhook test

Usage: `ag-cli repo webhook test [<owner>/<repo>] <id> [flags]`

Send a test payload to a repository webhook

Send a test payload to a repository webhook

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag-cli repo webhook test owner/repo 42 --yes
```


## ag-cli repo webhook view

Usage: `ag-cli repo webhook view [<owner>/<repo>] <id> [flags]`

View a repository webhook

View a repository webhook

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output webhook as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli repo webhook view owner/repo 42
```


## ag-cli run

Usage: `ag-cli run`

Inspect AtomGit Actions workflow runs and artifacts

List and inspect AtomGit Actions workflow runs, jobs, logs, and artifacts.

Artifacts can also be deleted after confirmation. Workflow run dispatch,
rerun, cancel, and deletion operations are not supported.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli run artifact

Usage: `ag-cli run artifact`

Inspect and manage workflow artifacts

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli run artifact delete

Usage: `ag-cli run artifact delete [<owner>/<repo>] <artifact-id> [flags]`

Delete a workflow artifact

Delete an AtomGit Actions artifact after reading its metadata.

By default, the command displays the artifact details and asks for
confirmation. Deletion cannot be undone. Use --yes to skip the prompt.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip deletion confirmation | `false` | local |

### Example

```bash
ag-cli run artifact delete owner/repo <artifact-id>
ag-cli run artifact delete <artifact-id> --yes
```


## ag-cli run artifact view

Usage: `ag-cli run artifact view [<owner>/<repo>] <artifact-id> [flags]`

View artifact metadata without downloading the archive

Display AtomGit Actions artifact metadata.

This command does not download the archive. Use ag-cli run view --artifact to
download a zip from a specific workflow run.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output artifact metadata as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli run artifact view owner/repo <artifact-id>
ag-cli run artifact view <artifact-id> --json
```


## ag-cli run list

Usage: `ag-cli run list [<owner>/<repo>] [flags]`

List workflow runs

List workflow runs

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--actor` | Filter by triggering username | `` | local |
| `--end-time` | Filter runs ending at or before this Unix timestamp in milliseconds | `0` | local |
| `--event` | Filter by event: mr, push, manual | `` | local |
| `--json` | Output workflow runs as JSON | `false` | local |
| `--pr` | Filter by pull request number | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--start-time` | Filter runs starting at or after this Unix timestamp in milliseconds | `0` | local |
| `--workflow` | Filter by workflow ID | `` | local |
| `--workflow-name` | Filter by workflow name | `` | local |
| `-L, --limit` | Maximum number of runs to list | `30` | local |
| `-b, --branch` | Filter by head branch | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-s, --status` | Filter by status: completed, running, failed, canceled, ignored, paused, suspend | `` | local |

### Example

```bash
ag-cli run list owner/repo
ag-cli run list
ag-cli run list owner/repo --branch main --status failed
ag-cli run list owner/repo --event push --workflow-name CI --limit 50
```


## ag-cli run step-log

Usage: `ag-cli run step-log [<owner>/<repo>] <run-id> <job-id> <step-id> [flags]`

Fetch step-level logs for a workflow job

Retrieve paginated AtomGit Actions step logs and write them as text.

Use ag-cli run view to discover step IDs. --output writes the complete log
atomically and refuses to replace an existing file unless --overwrite is set.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--output` | Write the complete log to a file instead of stdout | `` | local |
| `--overwrite` | Replace an existing --output destination | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli run step-log owner/repo <run-id> <job-id> <step-id>
ag-cli run step-log <run-id> <job-id> <step-id>
ag-cli run step-log owner/repo <run-id> <job-id> <step-id> --output step.log
ag-cli run step-log owner/repo <run-id> <job-id> <step-id> --output step.log --overwrite
```


## ag-cli run view

Usage: `ag-cli run view [<owner>/<repo>] <run-id> [flags]`

View a workflow run, jobs, logs, and artifacts

View a workflow run, jobs, logs, and artifacts

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--artifact` | Download a specific artifact as a zip archive | `` | local |
| `--artifact-file` | Artifact destination path (defaults to the artifact name) | `` | local |
| `--log` | Write the selected job log text to stdout | `false` | local |
| `--log-file` | Download the selected job log archive to a file | `` | local |
| `--overwrite` | Replace an existing download destination | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-j, --job` | View a specific job | `` | local |

### Example

```bash
ag-cli run view owner/repo 12345
ag-cli run view 12345
ag-cli run view owner/repo 12345 --job job-id
ag-cli run view owner/repo 12345 --job job-id --log
ag-cli run view owner/repo 12345 --job job-id --log-file job-logs.zip
ag-cli run view owner/repo 12345 --artifact artifact-id
ag-cli run view owner/repo 12345 --artifact artifact-id --artifact-file build.zip --overwrite
```


## ag-cli runner

Usage: `ag-cli runner`

Inspect AtomGit Actions host runners

List repository-specific and shared AtomGit Actions host runners. These commands are read-only.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli runner list owner/repo
ag-cli runner shared owner/repo --json
```


## ag-cli runner list

Usage: `ag-cli runner list [<owner>/<repo>] [flags]`

List host runners configured for a repository

List host runners configured for a repository. This command is read-only and does not change runner configuration.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output runners as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of runners to list (0 means all) | `0` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli runner list owner/repo
ag-cli runner list owner/repo --limit 25 --json
```


## ag-cli runner shared

Usage: `ag-cli runner shared [<owner>/<repo>] [flags]`

List host runners shared with a repository

List host runners shared with a repository. This command is read-only and does not change runner configuration.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output runners as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of runners to list (0 means all) | `0` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli runner shared owner/repo
ag-cli runner shared owner/repo --limit 25 --json
```


## ag-cli schema

Usage: `ag-cli schema [<command> ...]`

Describe public commands as versioned JSON

List public commands or describe an exact command path without executing it. Paths may include the ag-cli prefix; use 'ag-cli schema ag-cli' for root command details. Uses static command metadata only; no login or network is required. This is a versioned command description format, not JSON Schema. Undescribed behavior must not be inferred.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli schema
ag-cli schema ag-cli
ag-cli schema pr create
ag-cli schema ag-cli pr create
ag-cli schema api
ag-cli schema pr comment create
```


## ag-cli search

Usage: `ag-cli search`

search atomgit

Search AtomGit repositories, issues, and users. Pull request search is not supported.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli search issues

Usage: `ag-cli search issues <query> [flags]`

search issues

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output results as JSON | `false` | local |
| `--order` | Sort order: asc or desc | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--repo` | Filter by repository path | `` | local |
| `--sort` | Sort by created_at or last_push_at | `` | local |
| `--state` | Filter by state: open or closed | `` | local |
| `-L, --limit` | Maximum number of results | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli search repositories

Usage: `ag-cli search repositories <query> [flags]`

search repositories

Aliases: `repos`

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--fork` | Include forked repositories | `false` | local |
| `--json` | Output results as JSON | `false` | local |
| `--language` | Filter by repository language | `` | local |
| `--order` | Sort order: asc or desc | `` | local |
| `--owner` | Filter by repository owner path | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--sort` | Sort by last_push_at, stars_count, or forks_count | `` | local |
| `-L, --limit` | Maximum number of results | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli search users

Usage: `ag-cli search users <query> [flags]`

search users

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output results as JSON | `false` | local |
| `--order` | Sort order: asc or desc | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--sort` | Sort by joined_at | `` | local |
| `-L, --limit` | Maximum number of results | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli ssh-key

Usage: `ag-cli ssh-key <command>`

Manage SSH keys

Manage SSH keys registered with your AtomGit account.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli ssh-key add

Usage: `ag-cli ssh-key add [<key-file>] [flags]`

Add an SSH key to your AtomGit account

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-t, --title` | Title for the new key | `` | local |


## ag-cli ssh-key delete

Usage: `ag-cli ssh-key delete <id> [flags]`

Delete an SSH key from your AtomGit account

Delete an SSH key from your AtomGit account.

The target key is retrieved before deletion. By default, you will be prompted
to confirm the deletion. Use --yes to skip the confirmation prompt.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag-cli ssh-key delete 123
ag-cli ssh-key delete 123 --yes
```


## ag-cli ssh-key list

Usage: `ag-cli ssh-key list [flags]`

List SSH keys registered with your AtomGit account

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--limit` | Maximum number of SSH keys to list | `1000` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli ssh-key list
ag-cli ssh-key list --limit 200
```


## ag-cli tag

Usage: `ag-cli tag`

Manage tags

List, create, and delete tags, and manage protected tag rules.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli tag create

Usage: `ag-cli tag create [<owner>/<repo>] <tag_name> [flags]`

Create a tag

Create a tag

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--ref` | Branch, tag, or commit SHA to create the tag from (required) | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-m, --message` | Tag message | `` | local |


## ag-cli tag delete

Usage: `ag-cli tag delete [<owner>/<repo>] <tag_name> [flags]`

Delete a tag

Delete a tag from AtomGit.

By default, you will be prompted to confirm the deletion. Use --yes to skip
the confirmation prompt.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation prompt | `false` | local |

### Example

```bash
ag-cli tag delete owner/repo v1.0.0
ag-cli tag delete owner/repo v1.0.0 --yes
```


## ag-cli tag list

Usage: `ag-cli tag list [<owner>/<repo>] [flags]`

List tags

List tags

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output tags as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of tags to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli tag list owner/repo --limit 50
```


## ag-cli tag protection

Usage: `ag-cli tag protection`

Manage protected tag rules

List, view, create, update, and delete protected tag rules.

Rules may name an exact tag or contain an AtomGit wildcard pattern. AtomGit's
API exposes only create/push access levels; other web settings are not changed
by this command.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli tag protection delete

Usage: `ag-cli tag protection delete [<owner>/<repo>] <tag-or-pattern> [flags]`

Delete a protected tag rule

Delete a protected tag rule.

By default, the current repository and rule are shown and you will be prompted
to confirm. Use --yes to skip the confirmation prompt.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip deletion confirmation | `false` | local |

### Example

```bash
ag-cli tag protection delete owner/repo "v*"
ag-cli tag protection delete owner/repo "v*" --yes
```


## ag-cli tag protection list

Usage: `ag-cli tag protection list [<owner>/<repo>] [flags]`

List protected tag rules

List protected tag rules

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output protected tag rules as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of protected tag rules to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli tag protection list owner/repo --limit 50
```


## ag-cli tag protection set

Usage: `ag-cli tag protection set [<owner>/<repo>] <tag-or-pattern> [flags]`

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
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-y, --yes` | Skip confirmation when updating an existing rule | `false` | local |

### Example

```bash
ag-cli tag protection set owner/repo v1.0.0 --create-access maintainer
ag-cli tag protection set owner/repo "v*" --create-access developer
ag-cli tag protection set owner/repo v1.0.0
ag-cli tag protection set owner/repo v1.0.0 --create-access none --yes
```


## ag-cli tag protection view

Usage: `ag-cli tag protection view [<owner>/<repo>] <tag-or-pattern> [flags]`

View a protected tag rule

View a protected tag rule

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output the protected tag rule as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli update

Usage: `ag-cli update [flags]`

Update AtomGit CLI to the latest stable release

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-c, --check` | Check for an update without installing it | `false` | local |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli user

Usage: `ag-cli user`

View AtomGit users, repositories, namespaces, and activity

View AtomGit user profiles, email addresses, starred and watched repositories, authenticated namespaces, and personal activity events.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli user edit

Usage: `ag-cli user edit [flags]`

Edit the authenticated user's profile

Edit supported profile fields for the authenticated AtomGit user.

Only flags explicitly provided are sent to AtomGit. Pass an empty string to
clear a supported field. This command does not upload avatar files, verify
email addresses, or rename the account login.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--avatar` | Avatar URL | `` | local |
| `--company` | Company name | `` | local |
| `--description` | Profile description | `` | local |
| `--email` | Public email address | `` | local |
| `--github-account` | GitHub account name | `` | local |
| `--json` | Output the updated profile as JSON | `false` | local |
| `--location` | Profile location | `` | local |
| `--nickname` | Profile nickname | `` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--website` | Website URL | `` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli user edit --nickname "Alice" --company "Example Inc."
ag-cli user edit --description "" --location "Wuhan"
ag-cli user edit --website "https://example.com" --json
```


## ag-cli user emails

Usage: `ag-cli user emails [flags]`

List email addresses for the authenticated user

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output email addresses as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli user emails
ag-cli user emails --json
```


## ag-cli user events

Usage: `ag-cli user events [<username>] [flags]`

List personal activity events for a user

List personal activity events for a user. Without an explicit username, the authenticated user is used.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output events as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `--year` | Filter events to the specified year (0 disables the filter) | `0` | local |
| `-L, --limit` | Maximum number of events to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli user events
ag-cli user events alice
ag-cli user events alice --year 2026 --limit 50
ag-cli user events alice --json
```


## ag-cli user namespaces

Usage: `ag-cli user namespaces [flags]`

List namespaces for the authenticated user

List user and group namespaces visible to the authenticated user. The default mode is intrant.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output namespaces as JSON | `false` | local |
| `--mode` | Namespace source: intrant, project, or all | `intrant` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of namespaces to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli user namespaces
ag-cli user namespaces --mode project --limit 100
ag-cli user namespaces --mode all --json
```


## ag-cli user starred

Usage: `ag-cli user starred [<username>] [flags]`

List starred repositories for a user

List starred repositories for a user. Without a username, the authenticated-user endpoint is used; an explicit username selects the public-user endpoint. Authentication is required for both endpoints.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output repositories as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of repositories to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli user starred
ag-cli user starred alice --limit 100
ag-cli user starred alice --json
```


## ag-cli user view

Usage: `ag-cli user view [<login>] [flags]`

View the current user or a public user profile

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output the user profile as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |
| `-w, --web` | Open the user profile in the browser | `false` | local |

### Example

```bash
ag-cli user view
ag-cli user view alice
ag-cli user view alice --json
ag-cli user view alice --web
```


## ag-cli user watching

Usage: `ag-cli user watching [<username>] [flags]`

List watched repositories for a user

List watched repositories for a user. Without a username, the authenticated-user endpoint is used; an explicit username selects the public-user endpoint. Authentication is required for both endpoints.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output repositories as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-L, --limit` | Maximum number of repositories to list | `30` | local |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli user watching
ag-cli user watching alice --limit 100
ag-cli user watching alice --json
```


## ag-cli version

Usage: `ag-cli version [flags]`

Show version information

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output version information as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |


## ag-cli workflow

Usage: `ag-cli workflow`

Manage AtomGit Actions workflows

List, validate, and run AtomGit Actions workflows.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli workflow list owner/repo
ag-cli workflow validate --file .gitcode/workflows/ci.yml
ag-cli workflow run owner/repo 12345 --ref main
ag-cli workflow run owner/repo ci.yml -f env=production
```


## ag-cli workflow list

Usage: `ag-cli workflow list [<owner>/<repo>] [flags]`

List workflows in a repository

List workflows in a repository

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--json` | Output workflows as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli workflow list owner/repo
```


## ag-cli workflow run

Usage: `ag-cli workflow run [<owner>/<repo>] <workflow_id> [flags]`

Run a workflow

Manually trigger an AtomGit Actions workflow run (workflow_dispatch).

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

Aliases: `dispatch`

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-F, --field` | Add a string parameter in key=value format | `[]` | local |
| `-f, --raw-field` | Add a string parameter in key=value format | `[]` | local |
| `-h, --help` | Show help for command | `false` | inherited |
| `-r, --ref` | The git reference (branch or tag) to run the workflow on | `` | local |

### Example

```bash
ag-cli workflow run owner/repo 12345 --ref main
ag-cli workflow run owner/repo ci.yml --ref feature-branch -f env=prod -f debug=true
```


## ag-cli workflow validate

Usage: `ag-cli workflow validate [<owner>/<repo>] [flags]`

Validate a local workflow YAML file

Validate a local AtomGit Actions workflow file against the documented v8 endpoint.

The file is not modified. HTTP 200 with valid=false is treated as a command
error so CI can fail on invalid YAML.

When OWNER/REPO is omitted, the repository is inferred from the current Git repository. An explicit OWNER/REPO argument always takes precedence. Remote selection prefers remote.pushDefault, the current branch upstream, origin, then a unique AtomGit/GitCode remote.

### Flags

| Flag | Description | Default | Scope |
| --- | --- | --- | --- |
| `--file` | Path to a local workflow YAML file | `` | local |
| `--json` | Output the validation response as JSON | `false` | local |
| `--raw-output` | Disable terminal output sanitization for machine processing | `false` | inherited |
| `-h, --help` | Show help for command | `false` | inherited |

### Example

```bash
ag-cli workflow validate --file .gitcode/workflows/ci.yml
ag-cli workflow validate owner/repo --file workflow.yml
ag-cli workflow validate owner/repo --file workflow.yml --json
```
