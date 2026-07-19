package issue

import (
	"fmt"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
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
	cmd.AddCommand(newCmdIssueEdit(f))
	cmd.AddCommand(newCmdIssueClose(f))
	cmd.AddCommand(newCmdIssueLabel(f))
	cmd.AddCommand(newCmdIssueReopen(f))
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

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			issuePath := fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, number)
			var current api.Issue
			if err := client.Get(issuePath, &current); err != nil {
				return fmt.Errorf("failed to get issue before closing: %w", err)
			}

			updatePath := fmt.Sprintf("/repos/%s/issues/%s", owner, number)
			fields := map[string]string{
				"repo":  repo,
				"title": current.Title,
				"state": "close",
			}
			if err := client.PatchForm(updatePath, fields, nil); err != nil {
				return fmt.Errorf("failed to close issue: %w", err)
			}

			var verified api.Issue
			if err := client.Get(issuePath, &verified); err != nil {
				return fmt.Errorf("failed to verify issue state: %w", err)
			}
			if !strings.EqualFold(strings.TrimSpace(verified.State), "closed") {
				return fmt.Errorf("issue #%s is still not closed after update (state: %q)", number, verified.State)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Closed issue #%s\n", number)

			return nil
		},
	}

	return cmd
}

func newCmdIssueReopen(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reopen <owner>/<repo> <number>",
		Short: "Reopen an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo := parts[0], parts[1]
			number := args[1]

			issuePath := fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, number)
			var current api.Issue
			if err := client.Get(issuePath, &current); err != nil {
				return fmt.Errorf("failed to get issue: %w", err)
			}

			updatePath := fmt.Sprintf("/repos/%s/issues/%s", owner, number)
			fields := map[string]string{
				"repo":  repo,
				"title": current.Title,
				"state": "reopen",
			}
			if err := client.PatchForm(updatePath, fields, nil); err != nil {
				return fmt.Errorf("failed to reopen issue: %w", err)
			}

			var verified api.Issue
			if err := client.Get(issuePath, &verified); err != nil {
				return fmt.Errorf("failed to verify issue state: %w", err)
			}
			if !strings.EqualFold(strings.TrimSpace(verified.State), "open") {
				return fmt.Errorf("issue #%s is still not open after update (state: %q)", number, verified.State)
			}

			cmd.Printf("Reopened issue #%s\n", number)

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

			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}

			var owner, repo string
			if len(args) == 0 {
				return fmt.Errorf("repository required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}
			issues, err := api.GetPaginated[api.Issue](client, opts.Limit, func(page, perPage int) string {
				return fmt.Sprintf("/repos/%s/%s/issues?state=%s&page=%d&per_page=%d", owner, repo, opts.State, page, perPage)
			})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, issue := range issues {
				fmt.Fprintf(out, "#%s %s [%s]\n", issue.GetNumber(), issue.Title, issue.State)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.State, "state", "s", "open", "Filter by state: open, closed, all")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of issues to list")

	return cmd
}

func newCmdIssueView(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		web bool
	}

	cmd := &cobra.Command{
		Use:   "view [<owner>/]<repo> <number>",
		Short: "View an issue",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if opts.web {
				num, err := strconv.Atoi(number)
				if err != nil {
					return fmt.Errorf("invalid issue number: %s", number)
				}
				u := browser.BuildIssueURL(owner, repo, num)
				fmt.Fprintf(cmd.OutOrStdout(), "Opening %s in your browser.\n", u)
				if f.BrowserOpener != nil {
					if err := f.BrowserOpener(u); err != nil {
						return fmt.Errorf("failed to open browser: %w", err)
					}
				}
				return nil
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			var issue api.Issue
			path := fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, number)
			if err := client.Get(path, &issue); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Title: %s\n", issue.Title)
			fmt.Fprintf(out, "State: %s\n", issue.State)
			if labels := formatIssueLabels(issue.Labels); labels != "" {
				fmt.Fprintf(out, "Labels: %s\n", labels)
			}
			fmt.Fprintf(out, "Author: %s\n", issue.User.Login)
			fmt.Fprintf(out, "URL: %s\n", issue.HTMLURL)
			fmt.Fprintf(out, "Created: %s\n", issue.CreatedAt)
			if issue.Body != "" {
				fmt.Fprintf(out, "\n%s\n", issue.Body)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open an issue in the browser")

	return cmd
}

func formatIssueLabels(labels []api.Label) string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if name := strings.TrimSpace(label.Name); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
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

			fmt.Fprintf(cmd.OutOrStdout(), "Created issue #%s: %s\n", issue.GetNumber(), issue.HTMLURL)

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Issue title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Issue body")

	return cmd
}
