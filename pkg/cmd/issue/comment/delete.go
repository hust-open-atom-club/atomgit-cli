package comment

import (
	"fmt"
	"strconv"
	"strings"

<<<<<<< HEAD
	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
=======
	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
>>>>>>> 4ec08c7 (fix: update module path to atomgit.com/openeuler/ag-cli)
	"github.com/spf13/cobra"
)

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Yes bool
	}

	cmd := &cobra.Command{
		Use:   "delete [<owner>/]<repo> <number> <comment-id>",
		Short: "Delete a comment on an issue",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			var owner, repo string
			var number, commentID int

			if len(args) < 3 {
				return fmt.Errorf("repository, issue number, and comment ID required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			number, err = strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[1])
			}

			commentID, err = strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid comment ID: %s", args[2])
			}

			client := api.NewClient(token)
			currentUser, _ := f.Config.GetUser()

			// Verify issue exists (number is validated but not used directly)
			_ = number

			// Get the comment first to check ownership
			var comment api.Comment
			path := fmt.Sprintf("/repos/%s/%s/issues/comments/%d", owner, repo, commentID)
			if err := client.Get(path, &comment); err != nil {
				return fmt.Errorf("failed to get comment: %w", err)
			}

			// Check if current user owns this comment
			if comment.User.Login != currentUser {
				return fmt.Errorf("只能删除自己的评论")
			}

			// Confirm deletion
			if !opts.Yes {
				fmt.Printf("确定要删除评论 #%d 吗? [y/N]: ", commentID)
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Println("取消删除")
					return nil
				}
			}

			// Delete comment
			if err := client.Delete(path); err != nil {
				return fmt.Errorf("failed to delete comment: %w", err)
			}

			fmt.Printf("Deleted comment #%d\n", commentID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
