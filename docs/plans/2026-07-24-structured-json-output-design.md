# Structured JSON Output Design

## Scope and compatibility

Issue #30 adds machine-readable output to `repo list/view`, `issue list/view`, `pr list/view`, and `tag list`. Each command receives a boolean `--json` flag, matching the existing `ag version --json` interface. Text output remains the default and retains its current wording. Raw byte-stream commands such as `pr diff` are outside this change. View commands treat `--json` and `--web` as mutually exclusive because one writes structured data while the other opens a browser and prints a navigation message.

The alternative `--json field1,field2` design was considered but rejected for this first iteration. It would require a field-expression parser, validation rules, nested-field behavior, and shell-completion semantics across every command. A boolean flag provides the requested reliable parser boundary with substantially less public surface area and follows the project's existing JSON convention.

## Output contract

Commands do not encode `internal/api` response structs directly. Each resource package defines explicit output DTOs with lower-camel-case JSON names, following `version --json`. This prevents newly added API fields from silently becoming public CLI fields. Identifier values are normalized to strings because AtomGit may return numbers as strings or JSON numbers. Labels are exposed as arrays of names, branches as branch-name strings, and repository visibility is normalized to `public`, `internal`, or `private`.

List commands always encode arrays. Empty results therefore produce `[]`, never a text sentinel or `null`. View commands encode one object. Optional server values remain present as their zero values so the documented schema does not change from one response to another. A shared `cmdutil.WriteJSON` helper performs indented encoding to the command's injected writer and returns encoding failures to Cobra.

## Testing and documentation

Package tests use injected HTTP clients and buffers; no test reads real credentials or contacts AtomGit. Table-driven coverage validates list, empty-list, view, optional/empty values, normalized identifiers, absence of text prefixes, and API error propagation. Existing text-output tests remain unchanged to protect backward compatibility. README examples document the supported commands, the full-object/array contract, field naming, and the incompatibility between `--json` and `--web`.
