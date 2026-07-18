## Context

AtomGit CLI is a Go/Cobra command-line tool. The current root command registers repo, PR, issue, label, tag, auth, SSH key, license, and version commands, but no branch command. The existing `tag` command is close in endpoint shape, but it lacks pagination, default owner handling, safe deletion confirmation, and URL encoding for path segments.

The project already has reusable pieces that fit this change:

- `cmdutil.Factory` injects configuration and test HTTP clients.
- `internal/api.Client` centralizes authenticated HTTP calls.
- `internal/api.GetPaginated` handles page/per_page pagination and limit truncation.
- Existing command tests use mock HTTP transports to verify request methods, paths, query strings, bodies, and error handling.
- `internal/api.Repository` already includes `DefaultBranch`.

The AtomGit API documentation confirms branch endpoints for list, create, view, delete, and protected-branch rules. The docs do not fully expose request/response schemas, so implementation must keep branch JSON structs tolerant and confirm create request field names before final coding.

## Goals / Non-Goals

**Goals:**

- Add `ag branch list`, `ag branch view`, `ag branch create`, and `ag branch delete`.
- Keep operations scoped to AtomGit remote branches only.
- Reuse existing API client, pagination, command structure, and test injection patterns.
- Provide safe delete behavior that refuses default/protected branches before sending DELETE.
- Support branch names containing `/` by URL-encoding branch names as path segments.
- Add comprehensive mock HTTP tests and README/help examples.

**Non-Goals:**

- No local Git branch creation, checkout, rename, or deletion.
- No branch protection rule management beyond reading protected status needed for display and safe deletion.
- No interactive branch picker or local repository remote autodetection.
- No dependency upgrades or new external dependencies unless required by verified API behavior.

## Decisions

### Add a new `pkg/cmd/branch` package

Create a dedicated command package instead of extending `pkg/cmd/tag`. Branch behavior has additional safety checks, pagination, richer display fields, and URL encoding concerns. A separate package keeps the code easier to test and avoids forcing tag command behavior changes into this issue.

Alternative considered: copy and adapt `pkg/cmd/tag/tag.go`. This is fast but would preserve several gaps: no pagination, direct stdout use, direct `api.NewClient` instead of injected `newAPIClient`, and unescaped path segments.

### Use existing pagination helper for branch list

`ag branch list` should call `api.GetPaginated[api.Branch]` with:

```text
/repos/{owner}/{repo}/branches?page={page}&per_page={perPage}
```

The command should validate `--limit` before calling the API and use the same default limit convention as issue/label/PR list commands.

Alternative considered: make a single GET request and rely on server defaults. This would fail the issue acceptance requirement for pagination and limit.

### Parse repository arguments consistently and explicitly

Branch commands should accept an explicit repository argument. If implementation chooses to support short repository names, it should reuse the existing `parseRepositoryName` behavior from repo commands by moving shared parsing to a suitable shared location or by introducing branch-local tests that match the intended behavior.

For the first implementation, prefer `owner/repo` in command examples and validation because tag, issue, and label commands already use that format.

Alternative considered: infer repository from the current local Git remote. This is outside the issue scope and could accidentally imply local Git integration.

### URL-encode branch names as one path segment

Branch names must be escaped with path-segment semantics before insertion into:

```text
/repos/{owner}/{repo}/branches/{branch}
```

This is required for names like `feature/foo`. Tests must assert that such names become a single encoded path segment and do not create extra URL path components.

Alternative considered: use raw branch names in paths. This breaks common branch naming patterns and violates the URL encoding acceptance criterion.

### Delete uses preflight checks before confirmation and DELETE

`ag branch delete` should:

1. Resolve repository and branch name.
2. Fetch repository detail or equivalent data to determine `default_branch`.
3. Fetch branch detail to determine protected status.
4. Refuse deletion if the branch is the default branch.
5. Refuse deletion if the branch is protected.
6. Prompt for confirmation unless `--yes` is set.
7. Send DELETE only after all checks and confirmation pass.

This ordering ensures protected/default/confirmation tests can assert no DELETE request is sent.

Alternative considered: send DELETE and rely on server errors. That produces less clear local errors and cannot guarantee the "cancelled deletion does not send DELETE" and "refuse default/protected branch" behaviors.

### Use command input/output streams for testability

New branch commands should write through `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` and read confirmation from `cmd.InOrStdin()`. This keeps command tests deterministic and avoids global `fmt.Scanln` state.

Alternative considered: use `fmt.Printf` / `fmt.Scanln`, as `repo delete` currently does. This makes interactive behavior harder to test and should not be copied.

### Keep API types tolerant

Extend or replace the current `api.Branch` type to include branch name, commit summary, protected status, and optional API-returned metadata. The type should use explicit JSON tags and tolerate missing fields.

Create request type(s) for branch creation only after confirming the AtomGit API field names. Based on nearby tag API conventions, likely fields include branch name and source ref, but this must be verified against actual API behavior or a fuller OpenAPI source before implementation is finalized.

## Risks / Trade-offs

- API schema uncertainty → Confirm create request fields and response JSON with actual API behavior or a fuller OpenAPI source before finalizing tests.
- Branch protected status field may differ across endpoints → Model fields defensively and prefer branch detail for delete preflight.
- Fetching repo detail and branch detail before delete adds requests → Acceptable for safety; tests can verify no DELETE is sent on refusal.
- Existing repository parsing is not shared across packages → Either duplicate minimal parsing with tests or extract a small shared helper carefully to avoid unrelated behavior changes.
- Display formatting can drift from other commands → Keep output simple, stable, and documented in README/help examples.

## Migration Plan

No data migration is required. The change adds new commands and does not alter existing command behavior. Rollback is removing the branch command registration and package.

## Open Questions

- What are the exact JSON request fields for `POST /repos/:owner/:repo/branches`?
- Does branch detail include protected status reliably, or is `GET /protect_branches` needed to cross-check protected branch patterns?
- Should short repository names be supported from the initial branch command release, or should branch commands require `owner/repo` for consistency with tag/issue/label?
