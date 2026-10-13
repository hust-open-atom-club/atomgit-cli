// Package comment provides the ag discussion comment subcommands for creating,
// replying to, editing, and deleting comments on repository discussions.
package comment

import (
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdComment(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage discussion comments",
		Long: `Create, reply to, edit, and delete comments on repository discussions.

Comment identifiers stay opaque strings. Edit and delete confirm that the
comment belongs to the given discussion before writing. Reply targets must be
top-level comments; replies are only one level deep.`,
	}
	commandschema.Annotate(cmd, commandschema.Metadata{
		Output: "text", Effects: "none",
		Notes: []string{"Groups discussion comment writes. Each subcommand authenticates and validates its own arguments."},
	})

	cmd.AddCommand(newCmdCreate(f))
	cmd.AddCommand(newCmdReply(f))
	cmd.AddCommand(newCmdEdit(f))
	cmd.AddCommand(newCmdDelete(f))
	return cmd
}

func discussionNumber(value string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 || strconv.Itoa(number) != value {
		return 0, fmt.Errorf("invalid discussion number %q: must be a positive integer", value)
	}
	return number, nil
}

// validateCommentID checks that a comment identifier is safe to embed in an
// API path. AtomGit documents discussion comment IDs as strings, so any
// non-empty run of unreserved path characters is accepted. "." and ".." are
// rejected because path escaping leaves them unchanged.
func validateCommentID(value string) (string, error) {
	if value == "" || value == "." || value == ".." {
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

func commentBody(cmd *cobra.Command, body, bodyFile string) (string, error) {
	bodyChanged := cmd.Flags().Changed("body")
	bodyFileChanged := cmd.Flags().Changed("body-file")
	if !bodyChanged && !bodyFileChanged {
		return "", fmt.Errorf("comment body is required; use --body or --body-file")
	}
	resolved, err := cmdutil.ReadBody(body, bodyFile, bodyChanged, bodyFileChanged, cmd.InOrStdin())
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(resolved) == "" {
		return "", fmt.Errorf("comment body cannot be empty")
	}
	return resolved, nil
}

func printCommentResult(cmd *cobra.Command, f *cmdutil.Factory, owner, repo string, number int, summary string, comment api.DiscussionComment) {
	commentURL := ""
	if strings.TrimSpace(comment.ID) != "" {
		commentURL = cmdutil.ResolveWebURL("", f.Config.GetHost(), owner, repo, "discussions", strconv.Itoa(number))
	}
	cmdutil.PrintResultWithOptionalURL(cmd.OutOrStdout(), summary, commentURL)
}
