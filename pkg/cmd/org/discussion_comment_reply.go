package org

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdOrgDiscussionCommentReply(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "reply <org> <number> <comment-id> (--body <text> | --body-file <path-or->)",
		Short: "Reply to a discussion comment",
		Long: `Reply to a top-level comment on an organization discussion.

The parent comment is confirmed to belong to the discussion before the reply
is posted. Replying to a reply is rejected.`,
		Example: `  ag-cli org discussion comment reply my-org 1 abc123 --body "Agreed"
  ag-cli org discussion comment reply my-org 1 abc123 --body-file reply.md`,
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
			if _, err := api.RequireTopLevelOrganizationDiscussionComment(client, organization, number, commentID); err != nil {
				return fmt.Errorf("failed to verify reply parent on discussion #%d for organization %s: %w", number, organization, err)
			}

			created, err := api.ReplyOrganizationDiscussionComment(client, organization, number, commentID, body)
			if err != nil {
				return fmt.Errorf("failed to reply to comment %s on discussion #%d for organization %s (not retried; inspect the discussion before retrying): %w", commentID, number, organization, err)
			}
			if !validDiscussionPathID(created.ID) {
				return fmt.Errorf("cannot confirm created reply: response has no valid comment ID; creation may have succeeded, inspect the discussion before retrying")
			}
			printOrgDiscussionCommentResult(cmd, organization, number, fmt.Sprintf("Created reply #%s", created.ID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Reply body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read body text from file (use - for stdin)")
	cmd.MarkFlagsMutuallyExclusive("body", "body-file")
	cmd.MarkFlagsOneRequired("body", "body-file")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 3, MaxCount: 3, Description: "Explicit organization path, positive discussion number, and parent comment ID."},
		Output:      "text", Effects: "write",
		Notes: []string{"The parent must be a top-level comment of the named discussion. Membership is checked before the write.", "A non-empty Markdown body is required from exactly one of --body or --body-file; use - for stdin. Body text is preserved without trimming. The write is not retried."},
	})
	return cmd
}
