package comment

import (
	"fmt"
	"strconv"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		JSON bool
	}

	cmd := &cobra.Command{
		Use:   "view [<owner>/<repo>] <comment-id>",
		Short: "View a commit comment",
		Long:  "View a single repository commit comment by ID.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			commentID, err := strconv.ParseInt(remaining[0], 10, 64)
			if err != nil || commentID <= 0 {
				return fmt.Errorf("invalid comment ID: %s", remaining[0])
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			comment, err := api.GetCommitComment(client, owner, repo, commentID)
			if err != nil {
				return err
			}

			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newCommentJSON(comment))
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Comment #%d\n", comment.ID)
			fmt.Fprintf(out, "Author: @%s\n", comment.User.Login)
			fmt.Fprintf(out, "Created: %s\n", formatCommentTime(comment.CreatedAt))
			if comment.UpdatedAt != "" && comment.UpdatedAt != comment.CreatedAt {
				fmt.Fprintf(out, "Updated: %s\n", formatCommentTime(comment.UpdatedAt))
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, comment.Body)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output the comment as JSON")

	return cmd
}
