package comment

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdEdit(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "edit [<owner>/<repo>] <number> <comment-id> (--body <text> | --body-file <path-or->)",
		Short: "Edit a discussion comment",
		Long: `Replace the Markdown body of a discussion comment or reply.

The comment is confirmed to belong to the given discussion before the update
is sent. A comment that cannot be found is left unchanged.`,
		Example: `  ag discussion comment edit owner/repo 1 abc123 --body "Updated"
  ag discussion comment edit 1 reply.id --body-file note.md`,
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
			if _, err := api.FindDiscussionComment(client, repository.Owner, repository.Name, number, commentID); err != nil {
				return fmt.Errorf("failed to verify comment %s on discussion #%d for %s: %w", commentID, number, repository, err)
			}

			updated, err := api.UpdateDiscussionComment(client, repository.Owner, repository.Name, number, commentID, body)
			if err != nil {
				return fmt.Errorf("failed to update comment %s on discussion #%d for %s: %w", commentID, number, repository, err)
			}
			id := updated.ID
			if id == "" {
				id = commentID
			}
			printCommentResult(cmd, f, repository.Owner, repository.Name, number, fmt.Sprintf("Updated comment #%s", id), updated)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New comment body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read new body text from file (use - for stdin)")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 3, Description: "Discussion number and comment ID, optionally preceded by owner/repo. The comment may be a top-level comment or a reply."},
		Output:      "text", Effects: "write",
		Notes: []string{"Membership in the named discussion is confirmed before the update. An unconfirmed comment is not written.", "A non-empty Markdown body is required from exactly one of --body or --body-file. The write is not retried."},
	})
	return cmd
}
