## ADDED Requirements

### Requirement: Branch command group

The CLI SHALL provide a top-level `branch` command group for managing AtomGit remote branches.

#### Scenario: Branch subcommands are available

- **WHEN** a user runs `ag branch --help`
- **THEN** the help output lists `list`, `view`, `create`, and `delete` subcommands

#### Scenario: Branch operations are remote-only

- **WHEN** a user runs any `ag branch` subcommand
- **THEN** the command only communicates with AtomGit APIs and MUST NOT create, checkout, rename, or delete local Git branches

### Requirement: List remote branches

The CLI SHALL list remote branches for a repository with limit-controlled pagination.

#### Scenario: List branches with default limit

- **WHEN** a user runs `ag branch list owner/repo`
- **THEN** the CLI requests AtomGit branch list pages for `owner/repo`
- **AND** displays each returned branch name, latest commit information, protected status, and API-available metadata

#### Scenario: List branches with explicit limit

- **WHEN** a user runs `ag branch list owner/repo --limit 150`
- **THEN** the CLI requests enough paginated API pages to return at most 150 branches
- **AND** stops requesting pages when the limit is reached or the API returns the last page

#### Scenario: Reject invalid list limit

- **WHEN** a user runs `ag branch list owner/repo --limit 0`
- **THEN** the CLI exits with a clear error that the limit must be positive
- **AND** no AtomGit branch list request is sent

### Requirement: View remote branch details

The CLI SHALL show details for a single remote branch.

#### Scenario: View branch details

- **WHEN** a user runs `ag branch view owner/repo main`
- **THEN** the CLI requests the AtomGit branch detail endpoint for `main`
- **AND** displays the branch name, latest commit information, protected status, and API-available metadata

#### Scenario: View branch with slash in name

- **WHEN** a user runs `ag branch view owner/repo feature/foo`
- **THEN** the CLI URL-encodes `feature/foo` as one branch path segment
- **AND** sends a request that does not split the branch name into multiple path components

#### Scenario: View nonexistent branch

- **WHEN** AtomGit reports that the requested branch does not exist
- **THEN** the CLI exits with a clear error identifying the branch lookup operation

### Requirement: Create remote branch

The CLI SHALL create a remote branch from an explicit source ref.

#### Scenario: Create branch from ref

- **WHEN** a user runs `ag branch create owner/repo feature/foo --ref main`
- **THEN** the CLI sends a branch creation request to AtomGit for repository `owner/repo`
- **AND** the request includes the new branch name and source ref
- **AND** the CLI reports the created branch on success

#### Scenario: Require source ref

- **WHEN** a user runs `ag branch create owner/repo feature/foo` without a source ref
- **THEN** the CLI exits with a clear error that a source ref is required
- **AND** no AtomGit branch creation request is sent

#### Scenario: Create duplicate or invalid branch

- **WHEN** AtomGit rejects branch creation because the branch name is invalid or already exists
- **THEN** the CLI exits with a clear error identifying the branch creation operation and preserving the API error context

### Requirement: Delete remote branch safely

The CLI SHALL delete ordinary remote branches only after safety checks and confirmation.

#### Scenario: Delete ordinary branch with confirmation

- **WHEN** a user runs `ag branch delete owner/repo feature/foo`
- **AND** the branch is neither the repository default branch nor protected
- **AND** the user confirms the prompt
- **THEN** the CLI URL-encodes `feature/foo` as one branch path segment
- **AND** sends one AtomGit DELETE request for that branch
- **AND** reports successful deletion

#### Scenario: Cancel branch deletion

- **WHEN** a user runs `ag branch delete owner/repo feature/foo`
- **AND** the user declines the confirmation prompt
- **THEN** the CLI reports that deletion was cancelled
- **AND** no AtomGit DELETE request is sent

#### Scenario: Skip confirmation

- **WHEN** a user runs `ag branch delete owner/repo feature/foo --yes`
- **AND** the branch is neither the repository default branch nor protected
- **THEN** the CLI sends the AtomGit DELETE request without prompting

#### Scenario: Refuse default branch deletion

- **WHEN** a user runs `ag branch delete owner/repo main`
- **AND** `main` is the repository default branch
- **THEN** the CLI exits with a clear error that the default branch cannot be deleted
- **AND** no AtomGit DELETE request is sent

#### Scenario: Refuse protected branch deletion

- **WHEN** a user runs `ag branch delete owner/repo release`
- **AND** `release` is protected according to AtomGit API data
- **THEN** the CLI exits with a clear error that protected branches cannot be deleted
- **AND** no AtomGit DELETE request is sent

### Requirement: Branch command authentication and API errors

The CLI SHALL use existing authentication and API client behavior for all branch commands.

#### Scenario: Missing authentication

- **WHEN** a user runs any branch command without a usable AtomGit token
- **THEN** the CLI exits with a clear not-authenticated error
- **AND** no AtomGit API request is sent

#### Scenario: Permission error

- **WHEN** AtomGit returns a permission error for a branch operation
- **THEN** the CLI exits with a clear error identifying the attempted operation and preserving the API error context

### Requirement: Branch command documentation

The CLI SHALL document branch commands in help output and README examples.

#### Scenario: Command help includes examples

- **WHEN** a user runs help for `ag branch` or any branch subcommand
- **THEN** the help output includes usage and examples for listing, viewing, creating, and deleting remote branches

#### Scenario: README includes branch usage

- **WHEN** a user reads the README command usage section
- **THEN** it includes examples for `ag branch list`, `ag branch view`, `ag branch create`, and `ag branch delete`
