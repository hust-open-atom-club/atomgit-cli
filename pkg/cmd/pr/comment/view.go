package comment

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shinwell/ag-cli/internal/api"
	"github.com/shinwell/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "view [<owner>/]<repo> <number>",
		Short: "View all comments on a pull request",
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

			client := api.NewClient(token)

			var comments []api.Comment
			path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number)
			if err := client.Get(path, &comments); err != nil {
				return err
			}

			if len(comments) == 0 {
				fmt.Println("No comments found.")
				return nil
			}

			fmt.Printf("PR #%d 的评论 (共 %d 条):\n\n", number, len(comments))

			// Sort comments by creation time
			sort.Slice(comments, func(i, j int) bool {
				t1, _ := time.Parse(time.RFC3339, comments[i].CreatedAt)
				t2, _ := time.Parse(time.RFC3339, comments[j].CreatedAt)
				return t1.Before(t2)
			})

			// Build comment tree
			commentMap := make(map[string]*api.Comment)
			children := make(map[string][]string)
			var roots []string

			for i := range comments {
				commentMap[comments[i].ID] = &comments[i]
				if comments[i].ParentID != nil {
					children[*comments[i].ParentID] = append(children[*comments[i].ParentID], comments[i].ID)
				} else {
					roots = append(roots, comments[i].ID)
				}
			}

			// Print comment tree
			currentUser, _ := f.Config.GetUser()
			for _, rootID := range roots {
				printCommentTree(commentMap, children, rootID, 0, currentUser)
			}

			return nil
		},
	}
}

func printCommentTree(commentMap map[string]*api.Comment, children map[string][]string, id string, depth int, currentUser string) {
	comment := commentMap[id]
	if comment == nil {
		return
	}

	indent := strings.Repeat("    ", depth)

	// Format timestamp
	t, _ := time.Parse(time.RFC3339, comment.CreatedAt)
	timeStr := t.Format("2006-01-02 15:04")

	// Mark current user's comments
	userMarker := ""
	if comment.User.Login == currentUser {
		userMarker = " (你)"
	}

	fmt.Printf("%s[%s] @%s %s%s\n", indent, comment.ID, comment.User.Login, timeStr, userMarker)

	// Print body with indentation
	bodyLines := strings.Split(comment.Body, "\n")
	for _, line := range bodyLines {
		fmt.Printf("%s    %s\n", indent, line)
	}
	fmt.Println()

	// Print children
	for _, childID := range children[id] {
		printCommentTree(commentMap, children, childID, depth+1, currentUser)
	}
}
