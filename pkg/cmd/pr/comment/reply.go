package comment

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdReply(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body string
	}

	cmd := &cobra.Command{
		Use:   "reply [<owner>/]<repo> <number> <parent-comment-id>",
		Short: "Reply to a comment on a pull request",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			var owner, repo string
			var number, parentID int

			if len(args) < 3 {
				return fmt.Errorf("repository, PR number, and parent comment ID required")
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

			parentID, err = strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid parent comment ID: %s", args[2])
			}

			// Get body
			body := opts.Body
			if body == "" {
				fmt.Printf("Enter reply to comment #%s (press Ctrl+D when done):\n", args[2])
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
				return fmt.Errorf("reply body cannot be empty")
			}

			client := api.NewClient(token)

			// Reply using discussions API
			var comment api.Comment
			req := api.CommentRequest{Body: body}
			path := fmt.Sprintf("/repos/%s/%s/pulls/%d/discussions/%d/comments", owner, repo, number, parentID)
			if err := client.Post(path, req, &comment); err != nil {
				return fmt.Errorf("failed to create reply: %w", err)
			}

			fmt.Printf("Created reply #%d: %s\n", comment.ID, comment.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Reply body text")

	return cmd
}
