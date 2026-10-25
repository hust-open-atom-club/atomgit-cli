package org

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdOrgDiscussionCommentEdit(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "edit <org> <number> <comment-id> (--body <text> | --body-file <path-or->)",
		Short: "Edit a discussion comment",
		Long: `Replace the Markdown body of a discussion comment or reply.

The comment is confirmed to belong to the given discussion before the update
is sent. A comment that cannot be found is left unchanged.`,
		Example: `  ag-cli org discussion comment edit my-org 1 abc123 --body "Updated"
  ag-cli org discussion comment edit my-org 1 reply.id --body-file note.md`,
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
			body, err := orgDiscussionCommentBody(cmd, opts.Body, opts.BodyFile)
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

			updated, err := api.UpdateOrganizationDiscussionComment(client, organization, number, commentID, body)
			if err != nil {
				return fmt.Errorf("failed to update comment %s on discussion #%d for organization %s (not retried; inspect the discussion before retrying): %w", commentID, number, organization, err)
			}
			if updated.ID != commentID {
				return fmt.Errorf("cannot confirm update to comment %s: response has a missing or unexpected comment ID; update may have succeeded, inspect the discussion before retrying", commentID)
			}
			printOrgDiscussionCommentResult(cmd, organization, number, fmt.Sprintf("Updated comment #%s", updated.ID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New comment body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read new body text from file (use - for stdin)")
	cmd.MarkFlagsMutuallyExclusive("body", "body-file")
	cmd.MarkFlagsOneRequired("body", "body-file")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 3, MaxCount: 3, Description: "Explicit organization path, positive discussion number, and comment ID. The comment may be a top-level comment or a reply."},
		Output:      "text", Effects: "write",
		Notes: []string{"Membership in the named discussion is confirmed before the update. An unconfirmed comment is not written.", "A non-empty Markdown body is required from exactly one of --body or --body-file; use - for stdin. Body text is preserved without trimming. The write is not retried."},
	})
	return cmd
}
