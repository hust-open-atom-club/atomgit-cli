package comment

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdEdit(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body string
	}

	cmd := &cobra.Command{
		Use:   "edit [<owner>/<repo>] <number> <comment-id>",
		Short: "Edit a comment on a pull request",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 2)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			number, err := strconv.Atoi(remaining[0])
			if err != nil || number <= 0 {
				return fmt.Errorf("invalid PR number: %s", remaining[0])
			}

			commentID, err := strconv.Atoi(remaining[1])
			if err != nil || commentID <= 0 {
				return fmt.Errorf("invalid comment ID: %s", remaining[1])
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
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

			out := cmd.OutOrStdout()

			// Get new body
			body := opts.Body
			if body == "" {
				// Interactive mode with existing content
				fmt.Fprintf(out, "Editing comment #%s (press Ctrl+D when done):\n", remaining[1])
				fmt.Fprintln(out, "Current content:")
				fmt.Fprintln(out, comment.Body)
				fmt.Fprintln(out, "\n--- Enter new content below ---")

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

			summary := fmt.Sprintf("Updated comment #%d", comment.ID)
			cmdutil.PrintResultWithOptionalURL(out, summary, comment.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New comment body text")

	return cmd
}
