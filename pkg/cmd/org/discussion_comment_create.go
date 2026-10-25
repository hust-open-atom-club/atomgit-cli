package org

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdOrgDiscussionCommentCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "create <org> <number> (--body <text> | --body-file <path-or->)",
		Short: "Create a comment on a discussion",
		Long:  "Create a top-level Markdown comment on an organization discussion.",
		Example: `  ag-cli org discussion comment create my-org 1 --body "Looks good"
  ag-cli org discussion comment create my-org 1 --body-file comment.md
  ag-cli org discussion comment create my-org 1 --body-file -`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseDiscussionOrganization(args[0])
			if err != nil {
				return err
			}
			number, err := orgDiscussionNumber(args[1])
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
			created, err := api.CreateOrganizationDiscussionComment(client, organization, number, body)
			if err != nil {
				return fmt.Errorf("failed to create comment on discussion #%d for organization %s (not retried; inspect the discussion before retrying): %w", number, organization, err)
			}
			if !validDiscussionPathID(created.ID) {
				return fmt.Errorf("cannot confirm created comment: response has no valid comment ID; creation may have succeeded, inspect the discussion before retrying")
			}
			printOrgDiscussionCommentResult(cmd, organization, number, fmt.Sprintf("Created comment #%s", created.ID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Comment body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read body text from file (use - for stdin)")
	cmd.MarkFlagsMutuallyExclusive("body", "body-file")
	cmd.MarkFlagsOneRequired("body", "body-file")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 2, Description: "Explicit organization path and positive discussion number."},
		Output:      "text", Effects: "write",
		Notes: []string{"Creates a top-level discussion comment. A non-empty Markdown body is required from exactly one of --body or --body-file; use - for stdin. Body text is preserved without trimming.", "The write is not retried. A valid response comment ID is required and kept as a string."},
	})
	return cmd
}
