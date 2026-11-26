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
		Long:  "Reply to a pull request discussion and report the new comment ID from noteId (legacy note_id is also accepted).\nThe discussion ID identifies the thread, not the reply. Requests are never automatically retried or redirected.\nIf creation cannot be confirmed, check AtomGit before retrying to avoid duplicate comments.",
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

			// discussion_id is the thread identifier (a hex string), shown by
			// `ag-cli pr comment view` on the [discussion_id] header line.
			discussionID := strings.TrimSpace(remaining[1])
			if discussionID == "" {
				return fmt.Errorf("discussion ID cannot be empty")
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
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

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			resp, err := api.CreatePullRequestReply(client.WithContext(cmd.Context()), owner, repo, strconv.Itoa(number), discussionID, body)
			if err != nil {
				return fmt.Errorf("failed to create reply: %w", err)
			}
			id, _ := resp.ReplyID() // Validated by CreatePullRequestReply.
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created reply #%d in discussion %s\n", id, resp.DiscussionID)
			return err
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Reply body text")

	return cmd
}
