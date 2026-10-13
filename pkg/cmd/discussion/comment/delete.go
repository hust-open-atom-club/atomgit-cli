package comment

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <number> <comment-id>",
		Short: "Delete a discussion comment",
		Long: `Delete a discussion comment or reply.

The comment is confirmed to belong to the given discussion before deletion.
Asks for confirmation unless --yes is supplied. Cancelling does not send the
delete request.`,
		Example: `  ag discussion comment delete owner/repo 1 abc123
  ag discussion comment delete owner/repo 1 abc123 --yes`,
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

			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			if _, err := api.FindDiscussionComment(client, repository.Owner, repository.Name, number, commentID); err != nil {
				return fmt.Errorf("failed to verify comment %s on discussion #%d for %s: %w", commentID, number, repository, err)
			}

			if !yes {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("Delete comment %s on discussion #%d? [y/N] ", commentID, number))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Discussion comment deletion cancelled")
					return nil
				}
			}

			if err := api.DeleteDiscussionComment(client, repository.Owner, repository.Name, number, commentID); err != nil {
				return fmt.Errorf("failed to delete comment %s on discussion #%d for %s: %w", commentID, number, repository, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted comment #%s\n", commentID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip deletion confirmation")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 3, Description: "Discussion number and comment ID, optionally preceded by owner/repo. The comment may be a top-level comment or a reply."},
		Output:      "text", Effects: "write",
		Notes: []string{"Membership in the named discussion is confirmed before prompting. Cancellation and an unconfirmed comment send no delete request.", "A successful empty delete response is not an error. The delete is not retried."},
	})
	return cmd
}
