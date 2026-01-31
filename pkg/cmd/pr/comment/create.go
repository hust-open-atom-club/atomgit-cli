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

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "create [<owner>/]<repo> <number>",
		Short: "Create a comment on a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			var owner, repo string
			var number int

			if len(args) == 1 {
				return fmt.Errorf("repository and PR number required")
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

			// Get body from file if specified
			body := opts.Body
			if opts.BodyFile != "" {
				content, err := os.ReadFile(opts.BodyFile)
				if err != nil {
					return fmt.Errorf("failed to read body file: %w", err)
				}
				body = string(content)
			}

			// Interactive mode if no body provided
			if body == "" {
				fmt.Println("Enter comment body (press Ctrl+D when done):")
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

			client := api.NewClient(token)
			req := api.CommentRequest{Body: body}

			var comment api.Comment
			path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number)
			if err := client.Post(path, req, &comment); err != nil {
				return err
			}

			fmt.Printf("Created comment #%d: %s\n", comment.ID, comment.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Comment body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read body text from file")

	return cmd
}
