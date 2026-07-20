package comment

import (
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Yes bool
	}

	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <number> <comment-id>",
		Short: "Delete a comment on a pull request",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 2)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			number, err := strconv.Atoi(remaining[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", remaining[0])
			}

			commentID, err := strconv.Atoi(remaining[1])
			if err != nil {
				return fmt.Errorf("invalid comment ID: %s", remaining[1])
			}

			client := api.NewClient(token)
			currentUser, _ := f.Config.GetUser()

			// Verify PR exists (number is validated but not used directly)
			_ = number

			// Get the comment first to check ownership
			var comment api.Comment
			path := fmt.Sprintf("/repos/%s/%s/pulls/comments/%d", owner, repo, commentID)
			if err := client.Get(path, &comment); err != nil {
				return fmt.Errorf("failed to get comment: %w", err)
			}

			// Check if current user owns this comment
			if comment.User.Login != currentUser {
				return fmt.Errorf("只能删除自己的评论")
			}

			out := cmd.OutOrStdout()

			// Confirm deletion
			if !opts.Yes {
				fmt.Fprintf(out, "确定要删除评论 #%d 吗? [y/N]: ", commentID)
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Fprintln(out, "取消删除")
					return nil
				}
			}

			// Delete comment
			if err := client.Delete(path); err != nil {
				return fmt.Errorf("failed to delete comment: %w", err)
			}

			fmt.Fprintf(out, "Deleted comment #%d\n", commentID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
