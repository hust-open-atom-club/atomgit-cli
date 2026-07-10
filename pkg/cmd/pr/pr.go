package pr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/pr/comment"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
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
	cmd.AddCommand(newCmdPREdit(f))
	cmd.AddCommand(newCmdPRClose(f))
	cmd.AddCommand(newCmdPRDiff(f))
	cmd.AddCommand(newCmdViewIssues(f))
	cmd.AddCommand(newCmdLinkIssues(f))
	cmd.AddCommand(newCmdUnlinkIssues(f))
	cmd.AddCommand(comment.NewCmdComment(f))

	return cmd
}

func resolveBaseBranch(requested string, repository api.Repository) (string, error) {
	if base := strings.TrimSpace(requested); base != "" {
		return base, nil
	}
	if base := strings.TrimSpace(repository.DefaultBranch); base != "" {
		return base, nil
	}
	return "", fmt.Errorf("repository default branch is empty; specify --base")
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
				fmt.Printf("#%s %s [%s]\n", pr.GetNumber(), pr.Title, pr.State)
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

			// Get PR labels from separate endpoint
			var labels []api.Label
			labelsPath := fmt.Sprintf("/repos/%s/%s/pulls/%s/labels", owner, repo, number)
			if err := client.Get(labelsPath, &labels); err != nil {
				// Labels endpoint might not exist or fail, continue without labels
				labels = nil
			}

			fmt.Printf("Title: %s\n", pr.Title)
			fmt.Printf("State: %s\n", pr.State)
			fmt.Printf("Author: %s\n", pr.User.Login)
			fmt.Printf("URL: %s\n", pr.HTMLURL)
			fmt.Printf("Branch: %s -> %s\n", pr.Head.Ref, pr.Base.Ref)
			if len(labels) > 0 {
				labelNames := make([]string, len(labels))
				for i, label := range labels {
					labelNames[i] = label.Name
				}
				fmt.Printf("Labels: %s\n", strings.Join(labelNames, ", "))
			}
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

			base := strings.TrimSpace(opts.Base)
			if base == "" {
				var repository api.Repository
				path := fmt.Sprintf("/repos/%s/%s", owner, repo)
				if err := client.Get(path, &repository); err != nil {
					return err
				}
				base, err = resolveBaseBranch("", repository)
				if err != nil {
					return err
				}
			}

			head := opts.Head
			// Convert owner:branch format to owner/repo:branch for AtomGit API
			if strings.Contains(head, ":") && !strings.Contains(head, "/") {
				headParts := strings.SplitN(head, ":", 2)
				head = fmt.Sprintf("%s/%s:%s", headParts[0], repo, headParts[1])
			}

			body := map[string]interface{}{
				"title": opts.Title,
				"body":  opts.Body,
				"base":  base,
				"head":  head,
			}

			var pr api.PullRequest
			path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
			if err := client.Post(path, body, &pr); err != nil {
				return err
			}

			htmlURL := strings.Replace(pr.HTMLURL, "/pulls/", "/pull/", 1)
			fmt.Printf("Created PR #%s: %s\n", pr.GetNumber(), htmlURL)

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "PR title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "PR body")
	cmd.Flags().StringVar(&opts.Base, "base", "", "Base branch (defaults to repository default)")
	cmd.Flags().StringVar(&opts.Head, "head", "", "Head branch")

	return cmd
}

func newCmdPREdit(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Title string
		Body  string
	}

	cmd := &cobra.Command{
		Use:   "edit [<owner>/]<repo> <number>",
		Short: "Edit a pull request",
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

			body := map[string]interface{}{}
			if opts.Title != "" {
				body["title"] = opts.Title
			}
			if opts.Body != "" {
				body["body"] = opts.Body
			}

			if len(body) == 0 {
				return fmt.Errorf("at least one of --title or --body must be provided")
			}

			var pr api.PullRequest
			path := fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number)
			if err := client.Patch(path, body, &pr); err != nil {
				return err
			}

			fmt.Printf("Updated PR #%s: %s\n", pr.GetNumber(), pr.HTMLURL)

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "New PR title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New PR body")

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

			fmt.Printf("Closed PR #%s: %s\n", pr.GetNumber(), pr.HTMLURL)

			return nil
		},
	}

	return cmd
}

func newCmdPRDiff(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		JSON bool
	}

	cmd := &cobra.Command{
		Use:   "diff [<owner>/]<repo> <number>",
		Short: "Show diff of a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

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

			client := &http.Client{}
			url := fmt.Sprintf("%s%s/repos/%s/%s/pulls/%s/files.json", api.BaseURL, api.APIVersion, owner, repo, number)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			req.Header.Add("Authorization", "Bearer "+token)
			req.Header.Add("Accept", "application/json")

			res, err := client.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}

			if opts.JSON {
				fmt.Println(string(body))
				return nil
			}

			// Parse and format as patch
			var diffData struct {
				Code  int `json:"code"`
				Diffs []struct {
					Statistic struct {
						Path    string `json:"path"`
						OldPath string `json:"old_path"`
						NewPath string `json:"new_path"`
					} `json:"statistic"`
					AddedLines  int `json:"added_lines"`
					RemoveLines int `json:"remove_lines"`
					Content     struct {
						Text []struct {
							LineContent string `json:"line_content"`
							Type        string `json:"type"`
						} `json:"text"`
					} `json:"content"`
				} `json:"diffs"`
			}

			if err := json.Unmarshal(body, &diffData); err != nil {
				return fmt.Errorf("failed to parse diff: %w", err)
			}

			for i, diff := range diffData.Diffs {
				if i > 0 {
					fmt.Println()
				}

				fmt.Printf("diff --git a/%s b/%s\n", diff.Statistic.OldPath, diff.Statistic.NewPath)
				fmt.Printf("--- a/%s\n", diff.Statistic.OldPath)
				fmt.Printf("+++ b/%s\n", diff.Statistic.NewPath)

				// Output diff content with proper prefixes
				for _, line := range diff.Content.Text {
					switch line.Type {
					case "match":
						fmt.Printf(" %s\n", line.LineContent)
					case "old":
						fmt.Printf("-%s\n", line.LineContent)
					case "new":
						fmt.Printf("+%s\n", line.LineContent)
					default:
						fmt.Println(line.LineContent)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output raw JSON")

	return cmd
}
