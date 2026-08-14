// Package commit provides the ag commit command for listing and viewing
// repository commits.
package commit

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdCommit(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Manage commits",
		Long:  `List and view repository commits.`,
	}

	cmd.AddCommand(newCmdCommitList(f))
	cmd.AddCommand(newCmdCommitView(f))
	cmdutil.AddRepositoryContextHelp(cmd)

	return cmd
}

func newCmdCommitList(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Ref   string
		Path  string
		Since string
		Until string
		Limit int
		JSON  bool
	}

	cmd := &cobra.Command{
		Use:   "list [<owner>/<repo>]",
		Short: "List commits",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return err
			}

			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}

			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			commits, err := api.GetPaginated[api.Commit](client, opts.Limit, func(page, perPage int) string {
				query := url.Values{}
				// Forward only the filters the user supplied.
				if opts.Ref != "" {
					query.Set("sha", opts.Ref)
				}
				if opts.Path != "" {
					query.Set("path", opts.Path)
				}
				if opts.Since != "" {
					query.Set("since", opts.Since)
				}
				if opts.Until != "" {
					query.Set("until", opts.Until)
				}
				query.Set("page", strconv.Itoa(page))
				query.Set("per_page", strconv.Itoa(perPage))
				return fmt.Sprintf("/repos/%s/%s/commits?%s", owner, repo, query.Encode())
			})
			if err != nil {
				return err
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), commitsJSON(commits))
			}

			out := cmd.OutOrStdout()
			if len(commits) == 0 {
				fmt.Fprintln(out, "No commits found")
				return nil
			}

			for _, commit := range commits {
				fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n",
					shortSHA(commit.SHA),
					commitTitle(commit),
					commitAuthor(commit),
					commit.Commit.Author.Date,
					commitWebURL(owner, repo, commit),
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Ref, "ref", "", "Commit SHA or branch name to start from")
	cmd.Flags().StringVar(&opts.Path, "path", "", "Only list commits that touch the given file path")
	cmd.Flags().StringVar(&opts.Since, "since", "", "Only list commits after this time (RFC 3339, e.g. 2024-11-08T16:25:44Z)")
	cmd.Flags().StringVar(&opts.Until, "until", "", "Only list commits before this time (RFC 3339, e.g. 2024-11-08T16:25:44Z)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of commits to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output commits as JSON")

	return cmd
}

func newCmdCommitView(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		web  bool
		json bool
	}

	cmd := &cobra.Command{
		Use:   "view [<owner>/<repo>] <sha>",
		Short: "View a commit",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			sha := remaining[0]

			if opts.web {
				u := browser.BuildCommitURL(owner, repo, sha)
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
				return err
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			var commit api.Commit
			path := fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, url.PathEscape(sha))
			if err := client.Get(path, &commit); err != nil {
				return err
			}
			if opts.json {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newCommitJSON(commit))
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "SHA: %s\n", commit.SHA)
			fmt.Fprintf(out, "Title: %s\n", commitTitle(commit))
			fmt.Fprintf(out, "Author: %s\n", commitAuthor(commit))
			fmt.Fprintf(out, "Date: %s\n", commit.Commit.Author.Date)
			fmt.Fprintf(out, "URL: %s\n", commitWebURL(owner, repo, commit))
			if len(commit.Parents) > 0 {
				parents := make([]string, 0, len(commit.Parents))
				for _, parent := range commit.Parents {
					parents = append(parents, shortSHA(parent.SHA))
				}
				fmt.Fprintf(out, "Parents: %s\n", strings.Join(parents, ", "))
			}
			if commit.Commit.Message != "" {
				fmt.Fprintf(out, "\n%s\n", commit.Commit.Message)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open a commit in the browser")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output commit as JSON")
	cmd.MarkFlagsMutuallyExclusive("web", "json")

	return cmd
}

type commitJSON struct {
	SHA     string `json:"sha"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	URL     string `json:"url"`
}

func commitsJSON(commits []api.Commit) []commitJSON {
	result := make([]commitJSON, len(commits))
	for index, commit := range commits {
		result[index] = newCommitJSON(commit)
	}
	return result
}

func newCommitJSON(commit api.Commit) commitJSON {
	return commitJSON{
		SHA:     commit.SHA,
		Title:   commitTitle(commit),
		Message: commit.Commit.Message,
		Author:  commitAuthor(commit),
		Date:    commit.Commit.Author.Date,
		URL:     commit.HTMLURL,
	}
}

// shortSHA returns the first 7 characters of a commit SHA.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// commitTitle returns the first line of the commit message.
func commitTitle(commit api.Commit) string {
	message := commit.Commit.Message
	if idx := strings.IndexByte(message, '\n'); idx >= 0 {
		message = message[:idx]
	}
	return strings.TrimSpace(message)
}

// commitAuthor returns the account login, falling back to the commit author name.
func commitAuthor(commit api.Commit) string {
	if commit.Author.Login != "" {
		return commit.Author.Login
	}
	return commit.Commit.Author.Name
}

// commitWebURL returns the commit's web URL, falling back to a constructed URL.
func commitWebURL(owner, repo string, commit api.Commit) string {
	if commit.HTMLURL != "" {
		return commit.HTMLURL
	}
	return browser.BuildCommitURL(owner, repo, commit.SHA)
}
