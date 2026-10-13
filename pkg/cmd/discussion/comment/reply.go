package comment

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdReply(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "reply [<owner>/<repo>] <number> <comment-id> (--body <text> | --body-file <path-or->)",
		Short: "Reply to a discussion comment",
		Long: `Reply to a top-level comment on a repository discussion.

The parent comment is confirmed to belong to the discussion before the reply
is posted. Replying to a reply is rejected.`,
		Example: `  ag discussion comment reply owner/repo 1 abc123 --body "Agreed"
  ag discussion comment reply 1 abc123 --body-file reply.md`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 2)
			if err != nil {
				return err
			}
			number, err := discussionNumber(remaining[0])
			if err != nil {
				return err
			}
			commentID, err := validateCommentID(remaining[1])
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
			if _, err := api.RequireTopLevelDiscussionComment(client, repository.Owner, repository.Name, number, commentID); err != nil {
				return fmt.Errorf("failed to verify reply parent on discussion #%d for %s: %w", number, repository, err)
			}

			created, err := api.ReplyDiscussionComment(client, repository.Owner, repository.Name, number, commentID, body)
			if err != nil {
				return fmt.Errorf("failed to reply to comment %s on discussion #%d for %s: %w", commentID, number, repository, err)
			}
			if created.ID == "" {
				return fmt.Errorf("reply response did not include a comment ID")
			}
			printCommentResult(cmd, f, repository.Owner, repository.Name, number, fmt.Sprintf("Created reply #%s", created.ID), created)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Reply body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read body text from file (use - for stdin)")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 3, Description: "Discussion number and parent comment ID, optionally preceded by owner/repo."},
		Output:      "text", Effects: "write",
		Notes: []string{"The parent must be a top-level comment of the named discussion. Membership is checked before the write.", "A non-empty Markdown body is required from exactly one of --body or --body-file. The write is not retried."},
	})
	return cmd
}
