package org

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdOrgDiscussionComment(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage organization discussion comments",
		Long: `Create, reply to, edit, and delete comments on organization discussions.

Comment identifiers stay opaque strings. Edit and delete confirm that the
comment belongs to the given discussion before writing. Reply targets must be
top-level comments; replies are only one level deep.`,
	}
	cmd.AddCommand(newCmdOrgDiscussionCommentCreate(f))
	cmd.AddCommand(newCmdOrgDiscussionCommentReply(f))
	cmd.AddCommand(newCmdOrgDiscussionCommentEdit(f))
	cmd.AddCommand(newCmdOrgDiscussionCommentDelete(f))
	commandschema.Annotate(cmd, commandschema.Metadata{
		Output: "text", Effects: "none",
		Notes: []string{"Groups organization discussion comment writes. Each subcommand authenticates and validates its own arguments."},
	})
	return cmd
}

func validateOrgDiscussionCommentID(value string) (string, error) {
	if !validDiscussionPathID(value) {
		return "", fmt.Errorf("invalid comment ID: %q", value)
	}
	return value, nil
}

func orgDiscussionCommentBody(cmd *cobra.Command, body, bodyFile string) (string, error) {
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

func printOrgDiscussionCommentResult(cmd *cobra.Command, organization string, number int, summary string) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s on discussion #%d for organization %s\n", summary, number, organization)
}
