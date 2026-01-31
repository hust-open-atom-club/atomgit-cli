package pr

import (
	"fmt"
	"strings"

	"github.com/shinwell/ag-cli/internal/api"
	"github.com/shinwell/ag-cli/pkg/cmd/pr/comment"
	"github.com/shinwell/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdPR(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Manage pull requests",
		Long:  `Create, view, and checkout pull requests.`,
	}

	cmd.AddCommand(newCmdPRList(f))
	cmd.AddCommand(newCmdPRView(f))
	cmd.AddCommand(newCmdPRCreate(f))
	cmd.AddCommand(newCmdPRClose(f))
	cmd.AddCommand(comment.NewCmdComment(f))

	return cmd
}

func newCmdPRList(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		State string
		Limit int
	}

	cmd := &cobra.Command{
		Use:   "list [<owner>/]<repo>",
		Short: "List pull requests",
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

			var prs []api.PullRequest
			path := fmt.Sprintf("/repos/%s/%s/pulls?state=%s", owner, repo, opts.State)
			if err := client.Get(path, &prs); err != nil {
				return err
			}

			for _, pr := range prs {
				fmt.Printf("#%s %s [%s]\n", pr.Number, pr.Title, pr.State)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.State, "state", "s", "open", "Filter by state: open, closed, all")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of PRs to list")

	return cmd
}

func newCmdPRView(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view [<owner>/]<repo> <number>",
		Short: "View a pull request",
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
				return fmt.Errorf("repository and PR number required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			number = args[1]

			var pr api.PullRequest
			path := fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number)
			if err := client.Get(path, &pr); err != nil {
				return err
			}

			fmt.Printf("Title: %s\n", pr.Title)
			fmt.Printf("State: %s\n", pr.State)
			fmt.Printf("Author: %s\n", pr.User.Login)
			fmt.Printf("URL: %s\n", pr.HTMLURL)
			fmt.Printf("Branch: %s -> %s\n", pr.Head.Ref, pr.Base.Ref)
			fmt.Printf("Created: %s\n", pr.CreatedAt)
			if pr.Body != "" {
				fmt.Printf("\n%s\n", pr.Body)
			}

			return nil
		},
	}

	return cmd
}

func newCmdPRCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Title string
		Body  string
		Base  string
		Head  string
	}

	cmd := &cobra.Command{
		Use:   "create [<owner>/]<repo>",
		Short: "Create a pull request",
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
				"base":  opts.Base,
				"head":  opts.Head,
			}

			var pr api.PullRequest
			path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
			if err := client.Post(path, body, &pr); err != nil {
				return err
			}

			fmt.Printf("Created PR #%s: %s\n", pr.Number, pr.HTMLURL)

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "PR title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "PR body")
	cmd.Flags().StringVar(&opts.Base, "base", "master", "Base branch")
	cmd.Flags().StringVar(&opts.Head, "head", "", "Head branch")

	return cmd
}

func newCmdPRClose(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close [<owner>/]<repo> <number>",
		Short: "Close a pull request",
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
				return fmt.Errorf("repository and PR number required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			number = args[1]

			body := map[string]string{
				"state": "closed",
			}

			var pr api.PullRequest
			path := fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number)
			if err := client.Patch(path, body, &pr); err != nil {
				return err
			}

			fmt.Printf("Closed PR #%s: %s\n", pr.Number, pr.HTMLURL)

			return nil
		},
	}

	return cmd
}
