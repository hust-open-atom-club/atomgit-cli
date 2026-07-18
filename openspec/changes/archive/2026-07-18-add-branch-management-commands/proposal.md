## Why

AtomGit CLI currently supports remote tag management but does not provide remote branch management commands. Users must manage server-side branches through Git remotes or the web UI even though AtomGit API v5 exposes branch list, detail, create, delete, and protection-related endpoints.

This change adds first-class CLI support for the basic remote branch lifecycle while keeping all operations scoped to AtomGit remote state and avoiding implicit local Git branch changes.

## What Changes

- Add a new top-level `ag branch` command with `list`, `view`, `create`, and `delete` subcommands.
- List remote branches with pagination and a `--limit` option.
- Show branch name, latest commit information, protected status, and API-available metadata where returned by AtomGit.
- Create a remote branch from an explicit source ref.
- Delete a remote branch with confirmation by default and a `--yes` bypass for non-interactive use.
- Refuse to delete the repository default branch or a protected branch before sending a DELETE request.
- Ensure branch names are URL-encoded as path segments so branch names containing `/` work correctly.
- Add mock HTTP tests for CRUD behavior, pagination, URL encoding, cancellation, protected/default branch refusal, duplicate/nonexistent/invalid branch errors, and permission errors.
- Update README and command help examples for the new branch commands.

## Capabilities

### New Capabilities

- `branch-management`: Remote branch listing, viewing, creation, and safe deletion through AtomGit CLI.

### Modified Capabilities

- None.

## Impact

- New command package under `pkg/cmd/branch/`.
- Root command registration in `pkg/cmd/root/root.go`.
- Branch-related API request/response types in `internal/api/types.go`.
- Reuse of `internal/api.Client`, `internal/api.GetPaginated`, `cmdutil.Factory`, and existing repository parsing/client injection patterns.
- README updates for user-facing command examples.
- No new external Go dependencies expected.
