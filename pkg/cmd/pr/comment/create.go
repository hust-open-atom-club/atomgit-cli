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

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body     string
		BodyFile string
	}

	cmd := &cobra.Command{
		Use:   "create [<owner>/<repo>] <number>",
		Short: "Create a comment on a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			number, err := strconv.Atoi(remaining[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", remaining[0])
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
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
				fmt.Fprintln(cmd.OutOrStdout(), "Enter comment body (press Ctrl+D when done):")
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

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}
			req := api.CommentRequest{Body: body}

			var comment api.CreateCommentResponse
			path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number)
			if err := client.Post(path, req, &comment); err != nil {
				return err
			}

			commentID := comment.GetID()
			if commentID == "" {
				return fmt.Errorf("created comment response did not include a comment ID")
			}
			commentURL := cmdutil.ResolveWebURL(comment.GetURL(), f.Config.GetHost(), owner, repo, "pull", strconv.Itoa(number))
			summary := fmt.Sprintf("Created comment #%s on PR #%d", commentID, number)
			cmdutil.PrintResultWithOptionalURL(cmd.OutOrStdout(), summary, commentURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Comment body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read body text from file")

	return cmd
}
