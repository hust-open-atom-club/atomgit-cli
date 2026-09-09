// Package commit provides the ag commit command for listing, viewing,
// comparing, and inspecting repository commits.
package commit

import (
	"context"
	"errors"
	"fmt"
	"io"
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
		Long:  "List, view, compare, and inspect repository commits.",
	}

	cmd.AddCommand(newCmdCommitList(f))
	cmd.AddCommand(newCmdCommitView(f))
	cmd.AddCommand(newCmdCompare(f))
	cmd.AddCommand(newCmdCommitText(f, "diff"))
	cmd.AddCommand(newCmdCommitText(f, "patch"))
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
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}

			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

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
				fmt.Fprintf(
					out, "%s\t%s\t%s\t%s\t%s\n",
					escapeCell(shortSHA(commit.SHA)),
					escapeCell(commitTitle(commit)),
					escapeCell(commitAuthor(commit)),
					escapeCell(commit.Commit.Author.Date),
					escapeCell(commitWebURL(owner, repo, commit)),
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
			sha := strings.TrimSpace(remaining[0])
			if sha == "" {
				return fmt.Errorf("commit SHA is required")
			}

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
				return cmdutil.AuthenticationError(err)
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
			fmt.Fprintf(out, "SHA: %s\n", escapeCell(commit.SHA))
			fmt.Fprintf(out, "Title: %s\n", escapeCell(commitTitle(commit)))
			fmt.Fprintf(out, "Author: %s\n", escapeCell(commitAuthor(commit)))
			fmt.Fprintf(out, "Date: %s\n", escapeCell(commit.Commit.Author.Date))
			fmt.Fprintf(out, "URL: %s\n", escapeCell(commitWebURL(owner, repo, commit)))
			if len(commit.Parents) > 0 {
				parents := make([]string, 0, len(commit.Parents))
				for _, parent := range commit.Parents {
					parents = append(parents, shortSHA(parent.SHA))
				}
				fmt.Fprintf(out, "Parents: %s\n", escapeCell(strings.Join(parents, ", ")))
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

// commitAuthor returns the best available author display value, preferring the
// account login and falling back through the account name, the commit author
// name, and finally email addresses.
func commitAuthor(commit api.Commit) string {
	if commit.Author.Login != "" {
		return commit.Author.Login
	}
	if commit.Author.Name != "" {
		return commit.Author.Name
	}
	if commit.Commit.Author.Name != "" {
		return commit.Commit.Author.Name
	}
	if commit.Commit.Author.Email != "" {
		return commit.Commit.Author.Email
	}
	return commit.Author.Email
}

// commitWebURL returns the commit's web URL, falling back to a constructed URL.
func commitWebURL(owner, repo string, commit api.Commit) string {
	if commit.HTMLURL != "" {
		return commit.HTMLURL
	}
	return browser.BuildCommitURL(owner, repo, commit.SHA)
}

// escapeCell escapes characters that would otherwise break tab-separated or
// single-line text output: backslashes, tabs, newlines, and carriage returns.
func escapeCell(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

type comparisonJSON struct {
	Base      string             `json:"base"`
	Head      string             `json:"head"`
	BaseSHA   string             `json:"baseSHA"`
	MergeBase string             `json:"mergeBaseSHA"`
	Commits   []comparisonCommit `json:"commits"`
	Files     []comparisonFile   `json:"files"`
	Truncated bool               `json:"truncated"`
}

type comparisonCommit struct {
	SHA         string `json:"sha"`
	Message     string `json:"message"`
	Author      string `json:"author"`
	AuthorName  string `json:"authorName"`
	AuthorEmail string `json:"authorEmail"`
	AuthoredAt  string `json:"authoredAt"`
	URL         string `json:"url"`
}

type comparisonFile struct {
	SHA       string `json:"sha"`
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
	BlobURL   string `json:"blobURL"`
	RawURL    string `json:"rawURL"`
	Patch     string `json:"patch"`
	Binary    bool   `json:"binary"`
	Truncated bool   `json:"truncated"`
}

func newCmdCompare(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "compare [<owner>/<repo>] <base>...<head>",
		Short: "Compare two commits, branches, or tags",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.RangeArgs(1, 2)(cmd, args); err != nil {
				return err
			}
			_, err := comparisonArg(args)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			base, head, err := parseComparisonArg(args)
			if err != nil {
				return err
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}

			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			comparison, err := api.CompareCommits(commandContext(cmd), client, repository.Owner, repository.Name, base, head)
			if err != nil {
				return err
			}

			result := newComparisonJSON(base, head, repository, f.Config.GetHost(), comparison)
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}
			return writeComparison(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output comparison as JSON")
	return cmd
}

func newCmdCommitText(f *cmdutil.Factory, format string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   format + " [<owner>/<repo>] <sha>",
		Short: "Show a commit's " + format,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.RangeArgs(1, 2)(cmd, args); err != nil {
				return err
			}
			_, err := commitTextArg(args)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			sha, err := commitTextArg(args)
			if err != nil {
				return err
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}

			client, err := authenticatedClient(f)
			if err != nil {
				return err
			}
			body, err := api.GetCommitText(commandContext(cmd), client, repository.Owner, repository.Name, sha, format)
			if err != nil {
				return err
			}
			defer body.Close()
			if _, err := io.Copy(cmd.OutOrStdout(), body); err != nil {
				return fmt.Errorf("stream commit %s output: %w", format, err)
			}
			return nil
		},
	}
	return cmd
}

func comparisonArg(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("comparison is required")
	}
	value := args[len(args)-1]
	_, _, err := parseComparison(value)
	return value, err
}

func parseComparisonArg(args []string) (string, string, error) {
	value, err := comparisonArg(args)
	if err != nil {
		return "", "", err
	}
	return parseComparison(value)
}

func commitTextArg(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("commit SHA is required")
	}
	return validateRef(args[len(args)-1], "commit SHA")
}

func authenticatedClient(f *cmdutil.Factory) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, cmdutil.AuthenticationError(err)
	}
	return f.NewAPIClient(token)
}

func commandContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

func parseComparison(value string) (string, string, error) {
	separator := strings.Index(value, "...")
	if separator < 0 || strings.Contains(value[separator+3:], "...") {
		return "", "", errors.New("comparison must use the form <base>...<head>")
	}
	base, err := validateRef(value[:separator], "base ref")
	if err != nil {
		return "", "", err
	}
	head, err := validateRef(value[separator+3:], "head ref")
	if err != nil {
		return "", "", err
	}
	return base, head, nil
}

func validateRef(value, name string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s cannot be empty", name)
	}
	if value == "@" || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "..") || strings.Contains(value, "@{") || strings.Contains(value, "//") {
		return "", fmt.Errorf("invalid %s %q", name, value)
	}
	for _, part := range strings.Split(value, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return "", fmt.Errorf("invalid %s %q", name, value)
		}
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*[\\", r) {
			return "", fmt.Errorf("invalid %s %q", name, value)
		}
	}
	return value, nil
}

func newComparisonJSON(base, head string, repository cmdutil.Repository, host string, comparison *api.CommitComparison) comparisonJSON {
	result := comparisonJSON{
		Base:      base,
		Head:      head,
		BaseSHA:   comparison.BaseCommit.SHA,
		MergeBase: comparison.MergeBaseCommit.SHA,
		Commits:   make([]comparisonCommit, 0, len(comparison.Commits)),
		Files:     make([]comparisonFile, 0, len(comparison.Files)),
		Truncated: comparison.Truncated.Bool(),
	}
	for _, commit := range comparison.Commits {
		result.Commits = append(result.Commits, comparisonCommit{
			SHA:         commit.SHA,
			Message:     commit.Commit.Message,
			Author:      commit.Author.Login,
			AuthorName:  firstNonEmpty(commit.Author.Name, commit.Commit.Author.Name),
			AuthorEmail: firstNonEmpty(commit.Author.Email, commit.Commit.Author.Email),
			AuthoredAt:  commit.Commit.Author.Date,
			URL:         cmdutil.ResolveWebURL("", host, repository.Owner, repository.Name, "commit", commit.SHA),
		})
	}
	for _, file := range comparison.Files {
		result.Files = append(result.Files, comparisonFile{
			SHA:       file.SHA,
			Filename:  file.Filename,
			Status:    file.Status,
			Additions: file.Additions,
			Deletions: file.Deletions,
			Changes:   file.Changes,
			BlobURL:   file.BlobURL,
			RawURL:    file.RawURL,
			Patch:     file.Patch,
			Binary:    isBinaryPatch(file.Patch),
			Truncated: file.Truncated.Bool(),
		})
	}
	return result
}

func writeComparison(out io.Writer, comparison comparisonJSON) error {
	if _, err := fmt.Fprintf(out, "Comparison: %s...%s\n", cmdutil.EscapeTSVField(comparison.Base), cmdutil.EscapeTSVField(comparison.Head)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Base: %s\nMerge base: %s\nCommits: %d\nFiles: %d\nTruncated: %t\n",
		comparison.BaseSHA, comparison.MergeBase, len(comparison.Commits), len(comparison.Files), comparison.Truncated); err != nil {
		return err
	}
	if len(comparison.Commits) > 0 {
		if _, err := fmt.Fprintln(out, "\nCommits:"); err != nil {
			return err
		}
		for _, commit := range comparison.Commits {
			author := firstNonEmpty(commit.Author, commit.AuthorName, commit.AuthorEmail, "-")
			if _, err := fmt.Fprintf(out, "%s\t%s\t%s\n", cmdutil.EscapeTSVField(shortSHA(commit.SHA)), cmdutil.EscapeTSVField(firstLine(commit.Message)), cmdutil.EscapeTSVField(author)); err != nil {
				return err
			}
		}
	}
	if len(comparison.Files) > 0 {
		if _, err := fmt.Fprintln(out, "\nFiles:"); err != nil {
			return err
		}
		for _, file := range comparison.Files {
			metadata := "-"
			if file.Binary {
				metadata = "binary"
			}
			if file.Truncated {
				if metadata == "-" {
					metadata = "truncated"
				} else {
					metadata += ",truncated"
				}
			}
			if _, err := fmt.Fprintf(out, "%s\t+%d\t-%d\t%s\t%s\n",
				cmdutil.EscapeTSVField(firstNonEmpty(file.Status, "modified")), file.Additions, file.Deletions, cmdutil.EscapeTSVField(file.Filename), metadata); err != nil {
				return err
			}
		}
	}
	return nil
}

func firstLine(value string) string {
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isBinaryPatch(patch string) bool {
	patch = strings.TrimSpace(patch)
	return strings.HasPrefix(patch, "Binary files ") || strings.HasPrefix(patch, "GIT binary patch")
}
