package comment

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "create [<owner>/<repo>] <number> (--body <text> | --body-file <path-or->)",
		Short: "Create a comment on a discussion",
		Long:  "Create a top-level Markdown comment on a repository discussion.",
		Example: `  ag discussion comment create owner/repo 1 --body "Looks good"
  ag discussion comment create owner/repo 1 --body-file comment.md
  ag discussion comment create 1 --body-file -`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number, err := discussionNumber(remaining[0])
			if err != nil {
				return err
			}
			body, err := commentBody(cmd, opts.Body, opts.BodyFile)
			if err != nil {
				return err
			}

			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			created, err := api.CreateDiscussionComment(client, repository.Owner, repository.Name, number, body)
			if err != nil {
				return fmt.Errorf("failed to create comment on discussion #%d for %s: %w", number, repository, err)
			}
			if created.ID == "" {
				return fmt.Errorf("created comment response did not include a comment ID")
			}
			printCommentResult(cmd, f, repository.Owner, repository.Name, number, fmt.Sprintf("Created comment #%s", created.ID), created)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Comment body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read body text from file (use - for stdin)")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 2, Description: "Discussion number, optionally preceded by owner/repo. The number must be a positive integer; omitted repository uses Git context."},
		Output:      "text", Effects: "write",
		Notes: []string{"Creates a top-level discussion comment. A non-empty Markdown body is required from exactly one of --body or --body-file.", "The write is not retried. The response comment ID is kept as a string."},
	})
	return cmd
}
