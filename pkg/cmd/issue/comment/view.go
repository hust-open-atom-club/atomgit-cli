package comment

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

<<<<<<< HEAD
	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
=======
	"atomgit.com/openeuler/ag-cli/internal/api"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
>>>>>>> 4ec08c7 (fix: update module path to atomgit.com/openeuler/ag-cli)
	"github.com/spf13/cobra"
)

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "view [<owner>/]<repo> <number>",
		Short: "View all comments on an issue",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			var owner, repo string
			var number int

			if len(args) == 1 {
				return fmt.Errorf("repository and issue number required")
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

			client := api.NewClient(token)

			var comments []api.Comment
			path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)
			if err := client.Get(path, &comments); err != nil {
				return err
			}

			if len(comments) == 0 {
				fmt.Println("No comments found.")
				return nil
			}

			fmt.Printf("Issue #%d 的评论 (共 %d 条):\n\n", number, len(comments))

			// Sort comments by creation time
			sort.Slice(comments, func(i, j int) bool {
				t1, _ := time.Parse(time.RFC3339, comments[i].CreatedAt)
				t2, _ := time.Parse(time.RFC3339, comments[j].CreatedAt)
				return t1.Before(t2)
			})

			// Print comments (flat structure for issues)
			currentUser, _ := f.Config.GetUser()
			for _, comment := range comments {
				// Format timestamp
				t, _ := time.Parse(time.RFC3339, comment.CreatedAt)
				timeStr := t.Format("2006-01-02 15:04")

				// Mark current user's comments
				userMarker := ""
				if comment.User.Login == currentUser {
					userMarker = " (你)"
				}

				fmt.Printf("[%s] @%s %s%s\n", comment.ID, comment.User.Login, timeStr, userMarker)

				// Print body
				bodyLines := strings.Split(comment.Body, "\n")
				for _, line := range bodyLines {
					fmt.Printf("    %s\n", line)
				}
				fmt.Println()
			}

			return nil
		},
	}
}
