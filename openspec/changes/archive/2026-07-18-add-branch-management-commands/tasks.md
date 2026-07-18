## 1. API Contract and Types

- [x] 1.1 Confirm AtomGit branch create request field names and branch response fields from actual API behavior or a fuller OpenAPI source.
- [x] 1.2 Extend `internal/api` branch response types to include branch name, latest commit summary, protected status, and optional API metadata with explicit JSON tags.
- [x] 1.3 Add branch create request type(s) using the confirmed JSON field names.
- [x] 1.4 Add small helpers for URL-encoding branch names as single path segments.

## 2. Command Structure

- [x] 2.1 Create `pkg/cmd/branch` with `NewCmdBranch` and `list`, `view`, `create`, and `delete` subcommands.
- [x] 2.2 Register `branch.NewCmdBranch(f)` in the root command.
- [x] 2.3 Use `cmdutil.Factory` and injected HTTP clients consistently for branch commands.
- [x] 2.4 Decide and implement repository argument parsing for branch commands, matching the proposal/design scope and tests.

## 3. Branch List and View

- [x] 3.1 Implement `ag branch list <owner>/<repo> --limit <n>` using `api.GetPaginated`.
- [x] 3.2 Validate list limit before sending any API request.
- [x] 3.3 Format branch list output with name, latest commit information, protected status, and available API metadata.
- [x] 3.4 Implement `ag branch view <owner>/<repo> <branch>` using URL-encoded branch path segments.
- [x] 3.5 Format branch detail output with the same core fields as list plus available detail metadata.

## 4. Branch Create

- [x] 4.1 Implement `ag branch create <owner>/<repo> <branch> --ref <ref>`.
- [x] 4.2 Require an explicit source ref before sending any API request.
- [x] 4.3 Wrap create API errors with branch creation context while preserving underlying API details.
- [x] 4.4 Print a concise success message identifying the created branch.

## 5. Branch Delete Safety

- [x] 5.1 Implement `ag branch delete <owner>/<repo> <branch>` with `--yes` confirmation bypass.
- [x] 5.2 Fetch repository default branch data before deletion safety decisions.
- [x] 5.3 Fetch branch detail or protected branch data before deletion safety decisions.
- [x] 5.4 Refuse default branch deletion before sending DELETE.
- [x] 5.5 Refuse protected branch deletion before sending DELETE.
- [x] 5.6 Read confirmation from `cmd.InOrStdin()` and write prompts/results through command output streams.
- [x] 5.7 Ensure cancellation exits successfully or clearly without sending DELETE.
- [x] 5.8 URL-encode the branch name as a single path segment for DELETE.

## 6. Tests

- [x] 6.1 Add command registration/help tests for `ag branch` and subcommands.
- [x] 6.2 Add list tests for pagination, `--limit`, invalid limit, output fields, and permission errors.
- [x] 6.3 Add view tests for successful output, nonexistent branch errors, permission errors, and branch names containing `/`.
- [x] 6.4 Add create tests for successful request body, missing ref, invalid name, duplicate branch, and permission errors.
- [x] 6.5 Add delete tests for confirmed delete, `--yes`, cancellation without DELETE, default branch refusal, protected branch refusal, URL encoding, nonexistent branch, and permission errors.
- [x] 6.6 Ensure tests do not require real credentials, real AtomGit service, user home directories, browsers, fixed ports, or external network.

## 7. Documentation and Verification

- [x] 7.1 Update README with examples for branch list, view, create, and delete.
- [x] 7.2 Add command help examples for the branch command group and subcommands.
- [x] 7.3 Run `gofmt` on modified Go files.
- [x] 7.4 Run `go test ./...`.
- [x] 7.5 Run `go build ./cmd/ag`.
- [x] 7.6 Run help smoke checks for `go run ./cmd/ag branch --help` and each branch subcommand help.
