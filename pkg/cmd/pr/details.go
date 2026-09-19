package pr

import (
	"fmt"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type prCommitJSON struct {
	SHA        string         `json:"sha"`
	Message    string         `json:"message"`
	Author     prCommitAuthor `json:"author"`
	AuthoredAt string         `json:"authoredAt"`
	URL        string         `json:"url"`
}

type prCommitAuthor struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type prFileJSON struct {
	OldPath    string `json:"oldPath"`
	NewPath    string `json:"newPath"`
	ChangeType string `json:"changeType"`
	Additions  int    `json:"additions"`
	Deletions  int    `json:"deletions"`
	TooLarge   bool   `json:"tooLarge"`
	BlobURL    string `json:"blobURL"`
	RawURL     string `json:"rawURL"`
}

type prReactionJSON struct {
	ID        int64  `json:"id"`
	Author    string `json:"author"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type prActivityJSON struct {
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	Content   string `json:"content"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type prHistoryJSON struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Created   bool   `json:"created"`
	Deleted   bool   `json:"deleted"`
	Author    string `json:"author"`
	UpdatedBy string `json:"updatedBy"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func newCmdPRCommits(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		limit int
		json  bool
	}

	cmd := &cobra.Command{
		Use:   "commits [<owner>/<repo>] <number>",
		Short: "List commits in a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.limit)
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			number := remaining[0]
			_, err = parsePRNumber(number)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			commits, err := api.ListPullRequestCommits(client, owner, repo, number, opts.limit)
			if err != nil {
				return err
			}

			if opts.json {
				result := make([]prCommitJSON, 0, len(commits))
				for _, c := range commits {
					result = append(result, prCommitJSON{
						SHA:     c.SHA,
						Message: c.Commit.Message,
						Author: prCommitAuthor{
							Login: c.Commit.Author.Login,
							Name:  c.Commit.Author.Name,
							Email: c.Commit.Author.Email,
						},
						AuthoredAt: c.Commit.Author.Date,
						URL:        c.HTMLURL,
					})
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}

			out := cmd.OutOrStdout()
			for _, c := range commits {
				shortSHA := c.SHA
				if len(shortSHA) > 7 {
					shortSHA = shortSHA[:7]
				}
				msgLine := c.Commit.Message
				if idx := strings.Index(msgLine, "\n"); idx >= 0 {
					msgLine = msgLine[:idx]
				}
				author := c.Commit.Author.Login
				if author == "" {
					author = c.Commit.Author.Name
				}
				authoredAt := c.Commit.Author.Date
				authoredDisplay := authoredAt
				if t, err := parseTimestamp(authoredAt); err == nil {
					authoredDisplay = t
				}
				fmt.Fprintf(out, "%s %s by %s %s\n", shortSHA, msgLine, author, authoredDisplay)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "L", 30, "Maximum number of commits to list")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output commits as JSON")

	return cmd
}

func newCmdPRFiles(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		json bool
	}

	cmd := &cobra.Command{
		Use:   "files [<owner>/<repo>] <number>",
		Short: "List files changed in a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			number := remaining[0]
			_, err = parsePRNumber(number)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			files, err := api.ListPullRequestFiles(client, owner, repo, number)
			if err != nil {
				return err
			}

			if opts.json {
				result := make([]prFileJSON, 0, len(files))
				for _, pf := range files {
					oldPath := pf.Patch.OldPath
					if oldPath == "" {
						oldPath = pf.Filename
					}
					newPath := pf.Patch.NewPath
					if newPath == "" {
						newPath = pf.Filename
					}
					additions := pf.Additions
					if additions == 0 {
						additions = pf.Patch.AddedLines
					}
					deletions := pf.Deletions
					if deletions == 0 {
						deletions = pf.Patch.RemovedLines
					}
					tooLarge := pf.TooLarge || pf.Patch.TooLarge
					result = append(result, prFileJSON{
						OldPath:    oldPath,
						NewPath:    newPath,
						ChangeType: pf.GetChangeType(),
						Additions:  additions,
						Deletions:  deletions,
						TooLarge:   tooLarge,
						BlobURL:    pf.BlobURL,
						RawURL:     pf.RawURL,
					})
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}

			out := cmd.OutOrStdout()
			for _, pf := range files {
				oldPath := pf.Patch.OldPath
				if oldPath == "" {
					oldPath = pf.Filename
				}
				newPath := pf.Patch.NewPath
				if newPath == "" {
					newPath = pf.Filename
				}
				additions := pf.Additions
				if additions == 0 {
					additions = pf.Patch.AddedLines
				}
				deletions := pf.Deletions
				if deletions == 0 {
					deletions = pf.Patch.RemovedLines
				}
				line := fmt.Sprintf("%s +%d -%d", pf.GetChangeType(), additions, deletions)
				if newPath != oldPath {
					line += fmt.Sprintf(" %s -> %s", oldPath, newPath)
				} else {
					line += " " + oldPath
				}
				fmt.Fprintln(out, line)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.json, "json", false, "Output files as JSON")

	return cmd
}

func newCmdPRReactions(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		limit int
		json  bool
	}

	cmd := &cobra.Command{
		Use:   "reactions [<owner>/<repo>] <number>",
		Short: "List reactions on a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.limit)
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

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			reactions, err := api.ListPullRequestReactions(client, owner, repo, number, opts.limit)
			if err != nil {
				return err
			}

			if opts.json {
				result := make([]prReactionJSON, 0, len(reactions))
				for _, r := range reactions {
					result = append(result, prReactionJSON{
						ID:        r.ID,
						Author:    r.User.Login,
						Content:   r.Content,
						CreatedAt: r.CreatedAt,
					})
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}

			out := cmd.OutOrStdout()
			for _, r := range reactions {
				createdDisplay := r.CreatedAt
				if t, err := parseTimestamp(r.CreatedAt); err == nil {
					createdDisplay = t
				}
				fmt.Fprintf(out, "%s by %s %s\n", r.Content, r.User.Login, createdDisplay)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "L", 30, "Maximum number of reactions to list")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output reactions as JSON")

	return cmd
}

func newCmdPRActivity(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		limit int
		json  bool
	}

	cmd := &cobra.Command{
		Use:   "activity [<owner>/<repo>] <number>",
		Short: "List the operation log of a pull request",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.limit)
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

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			logs, err := api.ListPullRequestOperateLogs(client, owner, repo, number, opts.limit)
			if err != nil {
				return err
			}

			if opts.json {
				result := make([]prActivityJSON, 0, len(logs))
				for _, entry := range logs {
					result = append(result, prActivityJSON{
						ID:        entry.ID,
						Action:    entry.Action,
						Content:   entry.Content,
						Author:    userLogin(entry.User),
						CreatedAt: entry.CreatedAt,
						UpdatedAt: entry.UpdatedAt,
					})
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}

			out := cmd.OutOrStdout()
			for _, entry := range logs {
				createdDisplay := displayTimestamp(entry.CreatedAt)
				content := singleLine(entry.Content)
				if content != "" {
					fmt.Fprintf(out, "%s by %s %s: %s\n", entry.Action, userLogin(entry.User), createdDisplay, content)
					continue
				}
				fmt.Fprintf(out, "%s by %s %s\n", entry.Action, userLogin(entry.User), createdDisplay)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "L", 30, "Maximum number of activity entries to list")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output activity as JSON")

	return cmd
}

func newCmdPRHistory(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		limit int
		json  bool
	}

	cmd := &cobra.Command{
		Use:   "history [<owner>/<repo>] <number>",
		Short: "List the modification history of a pull request",
		Long: `List the modification history of a pull request.

The modify_history endpoint does not support pagination: the full response is
fetched and then truncated to --limit.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.limit)
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

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			history, err := api.ListPullRequestModifyHistory(client, owner, repo, number, opts.limit)
			if err != nil {
				return err
			}

			if opts.json {
				result := make([]prHistoryJSON, 0, len(history))
				for _, entry := range history {
					result = append(result, prHistoryJSON{
						ID:        entry.ID,
						Content:   entry.Content,
						Created:   entry.Created,
						Deleted:   entry.Deleted,
						Author:    userLogin(entry.User),
						UpdatedBy: userLogin(entry.UpdatedUser),
						CreatedAt: entry.CreatedAt,
						UpdatedAt: entry.UpdatedAt,
					})
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}

			out := cmd.OutOrStdout()
			for _, entry := range history {
				kind := "updated"
				if entry.Created {
					kind = "created"
				}
				if entry.Deleted {
					kind = "deleted"
				}
				display := displayTimestamp(entry.CreatedAt)
				actor := userLogin(entry.User)
				if kind != "created" {
					display = displayTimestamp(entry.UpdatedAt)
					// Updates and deletions are attributed to the user who
					// performed them, falling back to the original author when
					// the server omits updated_user.
					if updatedBy := userLogin(entry.UpdatedUser); updatedBy != "" {
						actor = updatedBy
					}
				}
				content := singleLine(entry.Content)
				if content != "" {
					fmt.Fprintf(out, "%s by %s %s: %s\n", kind, actor, display, content)
					continue
				}
				fmt.Fprintf(out, "%s by %s %s\n", kind, actor, display)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "L", 30, "Maximum number of history entries to list")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output history as JSON")

	return cmd
}

// userLogin returns the most specific available user identifier.
func userLogin(user api.User) string {
	if login := strings.TrimSpace(user.Login); login != "" {
		return login
	}
	return strings.TrimSpace(user.Name)
}

// singleLine collapses a server-provided message to a single output line so a
// multi-line body never breaks the one-entry-per-line text format.
func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// displayTimestamp formats an RFC 3339 timestamp for display, falling back to
// the raw value when it cannot be parsed.
func displayTimestamp(value string) string {
	if formatted, err := parseTimestamp(value); err == nil {
		return formatted
	}
	return value
}

func parseTimestamp(ts string) (string, error) {
	if ts == "" {
		return "", fmt.Errorf("empty timestamp")
	}
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return "", fmt.Errorf("invalid timestamp %q: %w", ts, err)
	}
	return parsed.Format("2006-01-02 15:04:05Z07:00"), nil
}
