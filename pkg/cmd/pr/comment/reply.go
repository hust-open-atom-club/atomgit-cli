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

func newCmdReply(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Body string
	}

	cmd := &cobra.Command{
		Use:   "reply [<owner>/<repo>] <number> <discussion-id>",
		Short: "Reply to a comment thread on a pull request",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
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

			// discussion_id is the thread identifier (a hex string), shown by
			// `ag pr comment view` on the [discussion_id] header line.
			discussionID := strings.TrimSpace(remaining[1])
			if discussionID == "" {
				return fmt.Errorf("discussion ID cannot be empty")
			}

			// Get body
			body := opts.Body
			if body == "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Enter reply to discussion %s (press Ctrl+D when done):\n", discussionID)
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

			// Reply using discussions API. The response carries the discussion id
			// as `id` and the new reply's comment id as `note_id`.
			var resp api.ReplyResponse
			req := api.CommentRequest{Body: body}
			path := fmt.Sprintf("/repos/%s/%s/pulls/%d/discussions/%s/comments", owner, repo, number, discussionID)
			if err := client.Post(path, req, &resp); err != nil {
				return fmt.Errorf("failed to create reply: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created reply #%d in discussion %s\n", resp.NoteID, resp.DiscussionID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Reply body text")

	return cmd
}
