// Package comment provides the ag-cli commit comment subcommands for listing,
// viewing, creating, editing, and deleting comments on repository commits.
package comment

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdComment(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage commit comments",
		Long:  `List, view, create, edit, and delete comments on repository commits.`,
	}

	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdView(f))
	cmd.AddCommand(newCmdCreate(f))
	cmd.AddCommand(newCmdEdit(f))
	cmd.AddCommand(newCmdDelete(f))

	return cmd
}

// validateCommentID checks that a comment identifier is safe to embed in an
// API path. AtomGit documents comment IDs as opaque strings (the create
// endpoint returns forms like "12312sadsa"), so any non-empty run of
// unreserved path characters is accepted and passed through verbatim.
// The complete values "." and ".." are rejected because url.PathEscape
// leaves them unchanged and HTTP clients or reverse proxies may normalize
// them to the current or parent path.
func validateCommentID(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("invalid comment ID: cannot be empty")
	}
	if value == "." || value == ".." {
		return "", fmt.Errorf("invalid comment ID: %q", value)
	}
	for _, r := range value {
		if !isUnreservedPathRune(r) {
			return "", fmt.Errorf("invalid comment ID: %q", value)
		}
	}
	return value, nil
}

func isUnreservedPathRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '-' || r == '.' || r == '_' || r == '~':
		return true
	default:
		return false
	}
}
