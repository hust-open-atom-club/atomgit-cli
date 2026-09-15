package comment

import (
	"fmt"
	"strconv"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Yes bool
	}

	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <comment-id>",
		Short: "Delete a commit comment",
		Long:  "Delete a commit comment you own. Asks for confirmation unless --yes is supplied.",
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
			currentUser, err := f.Config.GetUser()
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}

			comment, err := api.GetCommitComment(client, owner, repo, commentID)
			if err != nil {
				return fmt.Errorf("failed to verify commit comment: %w", err)
			}
			if comment.User.Login != currentUser {
				return fmt.Errorf("只能删除自己的评论")
			}

			out := cmd.OutOrStdout()

			if !opts.Yes {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("确定要删除评论 #%d 吗? [y/N]: ", commentID))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(out, "取消删除")
					return nil
				}
			}

			if err := api.DeleteCommitComment(client, owner, repo, commentID); err != nil {
				return err
			}

			fmt.Fprintf(out, "Deleted comment #%d\n", commentID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
