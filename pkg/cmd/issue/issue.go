package issue

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/issue/comment"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdIssue(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage issues",
		Long:  `Create, view, and manage issues.`,
	}

	cmd.AddCommand(newCmdIssueList(f))
	cmd.AddCommand(newCmdIssueView(f))
	cmd.AddCommand(newCmdIssueCreate(f))
	cmd.AddCommand(newCmdIssueClose(f))
	cmd.AddCommand(comment.NewCmdComment(f))

	return cmd
}

func newCmdIssueClose(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close [<owner>/]<repo> <number>",
		Short: "Close an issue",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client := api.NewClient(token)

			var owner, repo string
			var number string

			if len(args) == 1 {
				return fmt.Errorf("repository and issue number required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			number = args[1]

			// Close issue by adding "/close" comment
			req := api.CommentRequest{Body: "/close"}
			path := fmt.Sprintf("/repos/%s/%s/issues/%s/comments", owner, repo, number)
			var comment api.Comment
			if err := client.Post(path, req, &comment); err != nil {
				return fmt.Errorf("failed to close issue: %w", err)
			}

			fmt.Printf("Closed issue #%s\n", number)

			return nil
		},
	}

	return cmd
}

func newCmdIssueList(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		State string
		Limit int
	}

	cmd := &cobra.Command{
		Use:   "list [<owner>/]<repo>",
		Short: "List issues",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client := api.NewClient(token)

			var owner, repo string
			if len(args) == 0 {
				return fmt.Errorf("repository required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			var issues []api.Issue
			path := fmt.Sprintf("/repos/%s/%s/issues?state=%s", owner, repo, opts.State)
			if err := client.Get(path, &issues); err != nil {
				return err
			}

			for _, issue := range issues {
				fmt.Printf("#%s %s [%s]\n", issue.GetNumber(), issue.Title, issue.State)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.State, "state", "s", "open", "Filter by state: open, closed, all")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of issues to list")

	return cmd
}

func newCmdIssueView(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view [<owner>/]<repo> <number>",
		Short: "View an issue",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client := api.NewClient(token)

			var owner, repo string
			var number string

			if len(args) == 1 {
				return fmt.Errorf("repository and issue number required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			number = args[1]

			var issue api.Issue
			path := fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, number)
			if err := client.Get(path, &issue); err != nil {
				return err
			}

			fmt.Printf("Title: %s\n", issue.Title)
			fmt.Printf("State: %s\n", issue.State)
			fmt.Printf("Author: %s\n", issue.User.Login)
			fmt.Printf("URL: %s\n", issue.HTMLURL)
			fmt.Printf("Created: %s\n", issue.CreatedAt)
			if issue.Body != "" {
				fmt.Printf("\n%s\n", issue.Body)
			}

			return nil
		},
	}

	return cmd
}

func newCmdIssueCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Title string
		Body  string
	}

	cmd := &cobra.Command{
		Use:   "create [<owner>/]<repo>",
		Short: "Create an issue",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client := api.NewClient(token)

			var owner, repo string
			if len(args) == 0 {
				return fmt.Errorf("repository required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			if opts.Title == "" {
				return fmt.Errorf("title is required")
			}

			body := map[string]interface{}{
				"title": opts.Title,
				"body":  opts.Body,
			}

			var issue api.Issue
			path := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
			if err := client.Post(path, body, &issue); err != nil {
				return err
			}

			fmt.Printf("Created issue #%s: %s\n", issue.GetNumber(), issue.HTMLURL)

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Issue title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Issue body")

	return cmd
}
