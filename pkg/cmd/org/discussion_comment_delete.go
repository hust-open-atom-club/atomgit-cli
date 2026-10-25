package org

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdOrgDiscussionCommentDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <org> <number> <comment-id>",
		Short: "Delete a discussion comment",
		Long: `Delete a discussion comment or reply.

The comment is confirmed to belong to the given discussion before deletion.
Asks for confirmation unless --yes is supplied. Cancelling does not send the
delete request.`,
		Example: `  ag-cli org discussion comment delete my-org 1 abc123
  ag-cli org discussion comment delete my-org 1 abc123 --yes`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseDiscussionOrganization(args[0])
			if err != nil {
				return err
			}
			number, err := orgDiscussionNumber(args[1])
			if err != nil {
				return err
			}
			commentID, err := validateOrgDiscussionCommentID(args[2])
			if err != nil {
				return err
			}

			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			if _, err := api.FindOrganizationDiscussionComment(client, organization, number, commentID); err != nil {
				return fmt.Errorf("failed to verify comment %s on discussion #%d for organization %s: %w", commentID, number, organization, err)
			}

			if !yes {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("Delete comment %s on discussion #%d for organization %s? [y/N] ", commentID, number, organization))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Discussion comment deletion cancelled")
					return nil
				}
			}

			if err := api.DeleteOrganizationDiscussionComment(client, organization, number, commentID); err != nil {
				return fmt.Errorf("failed to delete comment %s on discussion #%d for organization %s (not retried; inspect the discussion before retrying): %w", commentID, number, organization, err)
			}
			printOrgDiscussionCommentResult(cmd, organization, number, fmt.Sprintf("Deleted comment #%s", commentID))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip deletion confirmation")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 3, MaxCount: 3, Description: "Explicit organization path, positive discussion number, and comment ID. The comment may be a top-level comment or a reply."},
		Output:      "text", Effects: "write",
		Notes: []string{"Membership in the named discussion is confirmed before prompting. Cancellation and an unconfirmed comment send no delete request.", "A successful empty delete response is not an error. The delete is not retried."},
	})
	return cmd
}
