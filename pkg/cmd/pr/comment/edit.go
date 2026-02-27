package comment

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/shinwell/ag-cli/internal/api"
	"github.com/shinwell/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdEdit(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body string
	}

	cmd := &cobra.Command{
		Use:   "edit [<owner>/]<repo> <number> <comment-id>",
		Short: "Edit a comment on a pull request",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			var owner, repo string
			var number, commentID int

			if len(args) < 3 {
				return fmt.Errorf("repository, PR number, and comment ID required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			number, err = strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[1])
			}

			commentID, err = strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid comment ID: %s", args[2])
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
				return fmt.Errorf("只能编辑自己的评论")
			}

			// Get new body
			body := opts.Body
			if body == "" {
				// Interactive mode with existing content
				fmt.Printf("Editing comment #%s (press Ctrl+D when done):\n", args[2])
				fmt.Println("Current content:")
				fmt.Println(comment.Body)
				fmt.Println("\n--- Enter new content below ---")

				reader := bufio.NewReader(os.Stdin)
				var lines []string
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						break
					}
					lines = append(lines, line)
				}
				body = strings.Join(lines, "")
			}

			if strings.TrimSpace(body) == "" {
				return fmt.Errorf("comment body cannot be empty")
			}

			// Update comment
			req := api.CommentRequest{Body: body}
			if err := client.Patch(path, req, &comment); err != nil {
				return fmt.Errorf("failed to update comment: %w", err)
			}

			fmt.Printf("Updated comment #%s: %s\n", comment.ID, comment.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New comment body text")

	return cmd
}
