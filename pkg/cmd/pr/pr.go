package pr

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
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
	cmd.AddCommand(newCmdPRReopen(f))
	cmd.AddCommand(newCmdPRReview(f))
	cmd.AddCommand(newCmdPRDiff(f))
	cmd.AddCommand(newCmdPRChecks(f))
	cmd.AddCommand(newCmdViewIssues(f))
	cmd.AddCommand(newCmdLinkIssues(f))
	cmd.AddCommand(newCmdUnlinkIssues(f))
	cmd.AddCommand(comment.NewCmdComment(f))
	cmd.AddCommand(newCmdPRMerge(f))
	cmdutil.AddRepositoryContextHelp(cmd)

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

func pullRequestResultURL(rawURL, host, owner, repo, number string) string {
	rawURL = strings.Replace(strings.TrimSpace(rawURL), "/pulls/", "/pull/", 1)
	return cmdutil.ResolveWebURL(rawURL, host, owner, repo, "pull", number)
}

func parsePRNumber(numberArg string) (string, error) {
	numberText := strings.TrimSpace(numberArg)
	number, err := strconv.Atoi(numberText)
	if err != nil || number <= 0 {
		return "", fmt.Errorf("invalid PR number: %s (expected positive integer)", numberArg)
	}

	return strconv.Itoa(number), nil
}

func newCmdPRList(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		State string
		Limit int
		JSON  bool
	}

	cmd := &cobra.Command{
		Use:   "list [<owner>/<repo>]",
		Short: "List pull requests",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}

			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}
			prs, err := api.GetPaginated[api.PullRequest](client, opts.Limit, func(page, perPage int) string {
				return fmt.Sprintf("/repos/%s/%s/pulls?state=%s&page=%d&per_page=%d", owner, repo, opts.State, page, perPage)
			})
			if err != nil {
				return err
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), pullRequestsJSON(prs))
			}

			out := cmd.OutOrStdout()
			for _, pr := range prs {
				fmt.Fprintf(out, "#%s %s [%s]\n", pr.GetNumber(), pr.Title, pr.State)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.State, "state", "s", "open", "Filter by state: open, closed, all")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of PRs to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output pull requests as JSON")

	return cmd
}

func newCmdPRView(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		web  bool
		json bool
	}

	cmd := &cobra.Command{
		Use:   "view [<owner>/<repo>] <number>",
		Short: "View a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			number := remaining[0]

			if opts.web {
				num, err := strconv.Atoi(number)
				if err != nil {
					return fmt.Errorf("invalid PR number: %s", number)
				}
				u := browser.BuildPRURL(owner, repo, num)
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
			if opts.json {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newPullRequestJSON(pr, labels))
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Title: %s\n", pr.Title)
			fmt.Fprintf(out, "State: %s\n", pr.State)
			fmt.Fprintf(out, "Author: %s\n", pr.User.Login)
			fmt.Fprintf(out, "URL: %s\n", pr.HTMLURL)
			fmt.Fprintf(out, "Branch: %s -> %s\n", pr.Head.Ref, pr.Base.Ref)
			if len(labels) > 0 {
				labelNames := make([]string, len(labels))
				for i, label := range labels {
					labelNames[i] = label.Name
				}
				fmt.Fprintf(out, "Labels: %s\n", strings.Join(labelNames, ", "))
			}
			fmt.Fprintf(out, "Created: %s\n", pr.CreatedAt)
			if pr.Body != "" {
				fmt.Fprintf(out, "\n%s\n", pr.Body)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open a pull request in the browser")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output pull request as JSON")
	cmd.MarkFlagsMutuallyExclusive("web", "json")

	return cmd
}

type pullRequestJSON struct {
	ID        int64    `json:"id"`
	Number    string   `json:"number"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	State     string   `json:"state"`
	URL       string   `json:"url"`
	Author    string   `json:"author"`
	Head      string   `json:"head"`
	Base      string   `json:"base"`
	Labels    []string `json:"labels"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
	Merged    bool     `json:"merged"`
	Mergeable bool     `json:"mergeable"`
}

func pullRequestsJSON(pullRequests []api.PullRequest) []pullRequestJSON {
	result := make([]pullRequestJSON, len(pullRequests))
	for index, pullRequest := range pullRequests {
		result[index] = newPullRequestJSON(pullRequest, pullRequest.Labels)
	}
	return result
}

func newPullRequestJSON(pullRequest api.PullRequest, labels []api.Label) pullRequestJSON {
	labelNames := make([]string, 0, len(labels))
	for _, label := range labels {
		if name := strings.TrimSpace(label.Name); name != "" {
			labelNames = append(labelNames, name)
		}
	}
	return pullRequestJSON{ID: pullRequest.ID, Number: pullRequest.GetNumber(), Title: pullRequest.Title, Body: pullRequest.Body, State: pullRequest.State, URL: pullRequest.HTMLURL, Author: pullRequest.User.Login, Head: pullRequest.Head.Ref, Base: pullRequest.Base.Ref, Labels: labelNames, CreatedAt: pullRequest.CreatedAt, UpdatedAt: pullRequest.UpdatedAt, Merged: pullRequest.Merged, Mergeable: pullRequest.Mergeable}
}

func newCmdPRCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Title string
		Body  string
		Base  string
		Head  string
	}

	cmd := &cobra.Command{
		Use:   "create [<owner>/<repo>]",
		Short: "Create a pull request",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

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

			var pr api.PullRequestWriteResponse
			path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
			if err := client.Post(path, body, &pr); err != nil {
				return err
			}

			number := pr.GetNumber()
			if number == "" {
				return fmt.Errorf("created PR response did not include a PR number")
			}
			htmlURL := pullRequestResultURL(pr.GetURL(), f.Config.GetHost(), owner, repo, number)
			fmt.Fprintf(cmd.OutOrStdout(), "Created PR #%s: %s\n", number, htmlURL)

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
		Use:   "edit [<owner>/<repo>] <number>",
		Short: "Edit a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			number := remaining[0]

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

			var pr api.PullRequestWriteResponse
			path := fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number)
			if err := client.Patch(path, body, &pr); err != nil {
				return err
			}

			resultNumber := pr.GetNumber()
			if resultNumber == "" {
				resultNumber = number
			}
			htmlURL := pullRequestResultURL(pr.GetURL(), f.Config.GetHost(), owner, repo, resultNumber)
			fmt.Fprintf(cmd.OutOrStdout(), "Updated PR #%s: %s\n", resultNumber, htmlURL)

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "New PR title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "New PR body")

	return cmd
}

func newCmdPRClose(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close [<owner>/<repo>] <number>",
		Short: "Close a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			number := remaining[0]

			body := map[string]string{
				"state": "closed",
			}

			var pr api.PullRequestWriteResponse
			path := fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number)
			if err := client.Patch(path, body, &pr); err != nil {
				return err
			}

			resultNumber := pr.GetNumber()
			if resultNumber == "" {
				resultNumber = number
			}
			htmlURL := pullRequestResultURL(pr.GetURL(), f.Config.GetHost(), owner, repo, resultNumber)
			fmt.Fprintf(cmd.OutOrStdout(), "Closed PR #%s: %s\n", resultNumber, htmlURL)

			return nil
		},
	}

	return cmd
}

func newCmdPRReopen(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reopen [<owner>/<repo>] <number>",
		Short: "Reopen a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			number := remaining[0]

			body := map[string]string{
				"state": "open",
			}

			path := fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number)
			if err := client.Patch(path, body, nil); err != nil {
				return fmt.Errorf("failed to reopen PR: %w", err)
			}

			htmlURL := pullRequestResultURL("", f.Config.GetHost(), owner, repo, number)
			cmd.Printf("Reopened PR #%s: %s\n", number, htmlURL)

			return nil
		},
	}

	return cmd
}

func newCmdPRDiff(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [<owner>/<repo>] <number>",
		Short: "Show diff of a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			number := remaining[0]

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repos/%s/%s/pulls/%s/diff", owner, repo, number)
			resp, err := client.DoRequestRaw(http.MethodGet, path)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
			}

			_, err = io.Copy(cmd.OutOrStdout(), resp.Body)
			return err
		},
	}

	return cmd
}

func newCmdPRMerge(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Rebase       bool
		Squash       bool
		Admin        bool
		Subject      string
		Body         string
		DeleteBranch bool
	}

	cmd := &cobra.Command{
		Use:   "merge [<owner>/<repo>] <number>",
		Short: "Merge a pull request",
		Long: `Merge a pull request.

By default, ag creates a merge commit. Use --rebase to rebase the commits onto the base branch.
`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			number, err := parsePRNumber(remaining[0])
			if err != nil {
				return err
			}

			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			var pr api.PullRequest
			path := fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number)
			if err := client.Get(path, &pr); err != nil {
				return fmt.Errorf("failed to get PR %s/%s #%s: %w", owner, repo, number, err)
			}

			if pr.Merged {
				return fmt.Errorf("PR #%s is already merged", pr.GetNumber())
			}
			if pr.State != "open" {
				return fmt.Errorf("PR #%s is closed, cannot merge", pr.GetNumber())
			}

			// Note: Work as intended.
			// AtomGit supports squash under rebase, see PR #32.
			mergeMethod := "merge"
			if opts.Rebase {
				mergeMethod = "rebase"
			}

			reqBody := api.MergePRRequest{
				MergeMethod: mergeMethod,
				Title:       opts.Subject,
				ForceMerge:  opts.Admin,
				Squash:      opts.Squash,
			}
			if opts.Squash {
				reqBody.SquashCommitMessage = opts.Body
			} else {
				reqBody.Description = opts.Body
			}

			mergePath := fmt.Sprintf("/repos/%s/%s/pulls/%s/merge", owner, repo, number)
			var mergeResp api.MergePRResponse
			if err := client.Put(mergePath, reqBody, &mergeResp); err != nil {
				return fmt.Errorf("failed to merge PR #%s: %w", number, err)
			}

			if !mergeResp.Merged {
				msg := mergeResp.Message
				return fmt.Errorf("failed to merge PR #%s: %s", number, msg)
			}

			switch {
			case mergeMethod == "merge" && !opts.Squash:
				fmt.Fprintf(cmd.OutOrStdout(), "Merged PR #%s: %s\n", pr.GetNumber(), pr.HTMLURL)
			case mergeMethod == "merge" && opts.Squash:
				fmt.Fprintf(cmd.OutOrStdout(), "Squashed and merged PR #%s: %s\n", pr.GetNumber(), pr.HTMLURL)
			case mergeMethod == "rebase" && !opts.Squash:
				fmt.Fprintf(cmd.OutOrStdout(), "Rebased and merged PR #%s: %s\n", pr.GetNumber(), pr.HTMLURL)
			case mergeMethod == "rebase" && opts.Squash:
				fmt.Fprintf(cmd.OutOrStdout(), "Rebased and merged PR with squash #%s: %s\n", pr.GetNumber(), pr.HTMLURL)
			}

			if opts.DeleteBranch {
				sourceRepo := strings.TrimSpace(pr.Head.Repo.FullName)
				sourceBranch := strings.TrimSpace(pr.Head.Ref)
				if sourceRepo == "" || sourceBranch == "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: cannot determine source repository or branch, skipping branch deletion\n")
				} else {
					branchName := url.PathEscape(sourceBranch)
					delPath := fmt.Sprintf("/repos/%s/branches/%s", sourceRepo, branchName)
					if err := client.Delete(delPath); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to delete branch %s: %v\n", sourceBranch, err)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "Deleted remote branch %s\n", sourceBranch)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.Rebase, "rebase", "r", false, "Rebase the commits onto the base branch")
	cmd.Flags().BoolVarP(&opts.Squash, "squash", "s", false, "Squash the commits into one commit")
	cmd.Flags().BoolVar(&opts.Admin, "admin", false, "Use administrator privileges to merge a pull request that does not meet requirements")
	cmd.Flags().StringVarP(&opts.Subject, "subject", "t", "", "Subject text for the merge commit")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Body text for the merge commit")
	cmd.Flags().BoolVarP(&opts.DeleteBranch, "delete-branch", "d", false, "Delete the source branch after merge")

	return cmd
}
