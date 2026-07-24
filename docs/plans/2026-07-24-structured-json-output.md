# Structured JSON Output Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add stable JSON output to the resource list/view commands named in Issue #30 without changing their default text output.

**Architecture:** Add one shared JSON writer in `pkg/cmdutil`, then define explicit resource output DTOs and conversion functions inside the existing repo, issue, PR, and tag command packages. Commands fetch data exactly as before and select either JSON or current text rendering at the final output boundary.

**Tech Stack:** Go 1.24.2, Cobra, `encoding/json`, injected `http.Client` test doubles.

---

### Task 1: Shared JSON writer

**Files:**
- Create: `pkg/cmdutil/json.go`
- Create: `pkg/cmdutil/json_test.go`

1. Add a failing test asserting valid indented JSON and writer-error propagation.
2. Run `go test ./pkg/cmdutil -run TestWriteJSON` and confirm failure.
3. Implement `WriteJSON(io.Writer, any) error` with `json.Encoder`.
4. Re-run the focused test and commit.

### Task 2: Repository JSON output

**Files:**
- Modify: `pkg/cmd/repo/repo.go`
- Modify: `pkg/cmd/repo/commands_test.go`

1. Add tests for list arrays, empty arrays, view objects, optional values, and unchanged text output.
2. Run the focused repo tests and confirm `--json` is missing.
3. Add explicit repository DTO conversion and `--json` flags to list/view.
4. Make `repo view --json` mutually exclusive with `--web`.
5. Run `go test ./pkg/cmd/repo` and commit.

### Task 3: Issue JSON output

**Files:**
- Modify: `pkg/cmd/issue/issue.go`
- Modify: `pkg/cmd/issue/issue_test.go`

1. Add list/view tests covering labels, empty arrays, identifier normalization, and API errors.
2. Add the issue DTO and JSON rendering after existing data retrieval.
3. Make `issue view --json` mutually exclusive with `--web`.
4. Run `go test ./pkg/cmd/issue` and commit.

### Task 4: Pull request JSON output

**Files:**
- Modify: `pkg/cmd/pr/pr.go`
- Modify: `pkg/cmd/pr/pr_test.go`

1. Add list/view tests covering labels, branches, booleans, empty arrays, and errors.
2. Add the pull-request DTO and JSON rendering.
3. Make `pr view --json` mutually exclusive with `--web`.
4. Run `go test ./pkg/cmd/pr` and commit.

### Task 5: Tag JSON output

**Files:**
- Modify: `pkg/cmd/tag/tag.go`
- Modify: `pkg/cmd/tag/tag_test.go`

1. Add populated and empty-list JSON tests.
2. Add the tag DTO and `--json` rendering while retaining `No tags found` in text mode.
3. Run `go test ./pkg/cmd/tag` and commit.

### Task 6: Documentation and verification

**Files:**
- Modify: `README.md`

1. Document supported commands, complete-object semantics, lower-camel-case fields, and examples.
2. Run `gofmt` on changed Go files and `git diff --check`.
3. With isolated `HOME`, `USERPROFILE`, and `XDG_CONFIG_HOME`, run `go test ./...`, `go vet ./...`, and `go build ./cmd/ag`.
4. Run `go run ./cmd/ag repo list --help`, `issue view --help`, `pr view --help`, and `tag list --help` and confirm `--json` is documented.
5. Commit, push `feat/structured-json-output`, create a PR linked to Issue #30, and verify the remote head and CI state.
