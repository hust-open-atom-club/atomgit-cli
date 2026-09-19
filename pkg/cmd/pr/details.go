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
	ID        string `json:"id"`
	Author    string `json:"author"`
	Emoji     string `json:"emoji"`
	EmojiName string `json:"emojiName"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type prActivityJSON struct {
	ID         int64  `json:"id"`
	Action     string `json:"action"`
	ActionType string `json:"actionType"`
	Content    string `json:"content"`
	Author     string `json:"author"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
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
		Example: `  ag pr reactions owner/repo 42
  ag pr reactions owner/repo 42 --limit 50 --json`,
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

			reactions, err := api.ListPullRequestReactions(client, owner, repo, number, opts.limit)
			if err != nil {
				return fmt.Errorf("failed to list pull request #%s reactions: %w", number, err)
			}

			if opts.json {
				result := make([]prReactionJSON, 0, len(reactions))
				for _, r := range reactions {
					result = append(result, prReactionJSON{
						ID:        string(r.ID),
						Author:    r.User.Login,
						Emoji:     r.Emoji,
						EmojiName: r.EmojiName,
						Content:   r.GetContent(),
						CreatedAt: r.CreatedAt,
					})
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}

			out := cmd.OutOrStdout()
			if len(reactions) == 0 {
				_, err := fmt.Fprintf(out, "No reactions found for pull request #%s.\n", number)
				return err
			}
			for _, r := range reactions {
				createdDisplay := r.CreatedAt
				if t, err := parseTimestamp(r.CreatedAt); err == nil {
					createdDisplay = t
				}
				content := r.GetContent()
				if r.Emoji != "" && r.Emoji != content {
					content += " " + r.Emoji
				}
				line := fmt.Sprintf("%s by %s", content, r.User.Login)
				if createdDisplay != "" {
					line += " " + createdDisplay
				}
				fmt.Fprintln(out, line)
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
		Long: `List the operation log of a pull request.

The operate_logs endpoint supports pagination; --limit caps how many entries
are fetched across pages.`,
		Example: `  ag pr activity owner/repo 42
  ag pr activity owner/repo 42 --limit 50 --json`,
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

			logs, err := api.ListPullRequestOperateLogs(client, owner, repo, number, opts.limit)
			if err != nil {
				return fmt.Errorf("failed to list pull request #%s activity: %w", number, err)
			}

			if opts.json {
				result := make([]prActivityJSON, 0, len(logs))
				for _, entry := range logs {
					result = append(result, prActivityJSON{
						ID:         entry.ID,
						Action:     entry.Action,
						ActionType: entry.ActionType,
						Content:    entry.Content,
						Author:     entry.User.Login,
						CreatedAt:  entry.CreatedAt,
						UpdatedAt:  entry.UpdatedAt,
					})
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}

			out := cmd.OutOrStdout()
			if len(logs) == 0 {
				_, err := fmt.Fprintf(out, "No activity found for pull request #%s.\n", number)
				return err
			}
			for _, entry := range logs {
				action := entry.Action
				if strings.TrimSpace(action) == "" {
					action = entry.ActionType
				}
				content := singleLine(entry.Content)
				display := auditCell(displayTimestamp(entry.CreatedAt))
				if content != "" {
					fmt.Fprintf(out, "%s by %s %s: %s\n", auditCell(action), auditActor(entry.User), display, content)
					continue
				}
				fmt.Fprintf(out, "%s by %s %s\n", auditCell(action), auditActor(entry.User), display)
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
		Example: `  ag pr history owner/repo 42
  ag pr history owner/repo 42 --limit 50 --json`,
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
				return fmt.Errorf("failed to list pull request #%s history: %w", number, err)
			}

			if opts.json {
				result := make([]prHistoryJSON, 0, len(history))
				for _, entry := range history {
					result = append(result, prHistoryJSON{
						ID:        entry.ID,
						Content:   entry.Content,
						Created:   entry.Created,
						Deleted:   entry.Deleted,
						Author:    entry.User.Login,
						UpdatedBy: entry.UpdatedUser.Login,
						CreatedAt: entry.CreatedAt,
						UpdatedAt: entry.UpdatedAt,
					})
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}

			out := cmd.OutOrStdout()
			if len(history) == 0 {
				_, err := fmt.Fprintf(out, "No history found for pull request #%s.\n", number)
				return err
			}
			for _, entry := range history {
				kind := "updated"
				if entry.Created {
					kind = "created"
				}
				if entry.Deleted {
					kind = "deleted"
				}
				display := displayTimestamp(entry.CreatedAt)
				actor := entry.User
				if kind != "created" {
					display = displayTimestamp(entry.UpdatedAt)
					if display == "" {
						display = displayTimestamp(entry.CreatedAt)
					}
					// Updates and deletions are attributed to the user who
					// performed them, falling back to the original author when
					// the server omits updated_user.
					if userLogin(entry.UpdatedUser) != "" {
						actor = entry.UpdatedUser
					}
				}
				content := singleLine(entry.Content)
				if content != "" {
					fmt.Fprintf(out, "%s by %s %s: %s\n", kind, auditActor(actor), auditCell(display), content)
					continue
				}
				fmt.Fprintf(out, "%s by %s %s\n", kind, auditActor(actor), auditCell(display))
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

// auditActor renders the best available actor identifier, or "-" when absent.
func auditActor(user api.User) string {
	if actor := userLogin(user); actor != "" {
		return actor
	}
	return "-"
}

// auditCell collapses whitespace and renders an empty value as "-".
func auditCell(value string) string {
	value = singleLine(value)
	if value == "" {
		return "-"
	}
	return value
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
