package issue

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type inspectionOptions struct {
	Limit int
	JSON  bool
}

type issueActivityJSON struct {
	ID        int64  `json:"id"`
	Author    string `json:"author"`
	Action    string `json:"action"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	IssueID   string `json:"issueId"`
	Title     string `json:"title"`
}
type issueHistoryJSON struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	UpdatedBy string `json:"updatedBy"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Created   bool   `json:"created"`
	Deleted   bool   `json:"deleted"`
}
type issueReactionJSON struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	Emoji     string `json:"emoji"`
	EmojiName string `json:"emojiName"`
}

func newCmdIssueActivity(f *cmdutil.Factory) *cobra.Command {
	return newIssueInspection(f, "activity", "List operation logs for an issue", api.ListIssueActivity,
		func(item api.IssueActivity) any {
			return issueActivityJSON{item.ID, item.User.Login, item.ActionType, item.Content, item.CreatedAt, item.UpdatedAt, item.IssueID, item.Title}
		},
		"The API returns an unpaginated list; --limit caps the displayed entries in server order.",
		"ACTOR\tACTION\tISSUE\tCREATED AT\tCONTENT", func(out io.Writer, number string, item api.IssueActivity) {
			fmt.Fprintf(out, "%s\t%s\t#%s\t%s\t%s\n", issueActorName(item.User), inspectionCell(item.ActionType), number, inspectionCell(item.CreatedAt), inspectionCell(item.Content))
		})
}

func newCmdIssueHistory(f *cmdutil.Factory) *cobra.Command {
	return newIssueInspection(f, "history", "List modification history for an issue", api.ListIssueHistory,
		func(item api.IssueHistory) any {
			return issueHistoryJSON{item.ID, item.User.Login, item.UpdatedUser.Login, item.Content, item.CreatedAt, item.UpdatedAt, item.Created, item.Deleted}
		},
		"The API returns an unpaginated list; --limit caps the displayed entries in server order. JSON includes both the creator and updater of each version.",
		"ACTOR\tACTION\tISSUE\tTIME\tCONTENT", func(out io.Writer, number string, item api.IssueHistory) {
			action := "updated"
			if item.Created {
				action = "created"
			}
			if item.Deleted {
				action = "deleted"
			}
			actor, timestamp := item.UpdatedUser, item.UpdatedAt
			if actor.Login == "" && actor.Name == "" {
				actor = item.User
			}
			if timestamp == "" {
				timestamp = item.CreatedAt
			}
			fmt.Fprintf(out, "%s\t%s\t#%s\t%s\t%s\n", issueActorName(actor), action, number, inspectionCell(timestamp), inspectionCell(item.Content))
		})
}

func newCmdIssueReactions(f *cmdutil.Factory) *cobra.Command {
	return newIssueInspection(f, "reactions", "List reactions on an issue", api.ListIssueReactions,
		func(item api.IssueReaction) any {
			return issueReactionJSON{item.ID, item.User.Login, item.Emoji, item.EmojiName}
		},
		"Fetch paginated reactions up to --limit. The API does not provide reaction timestamps.",
		"ACTOR\tREACTION\tISSUE", func(out io.Writer, number string, item api.IssueReaction) {
			fmt.Fprintf(out, "%s\t%s\t#%s\n", issueActorName(item.User), inspectionCell(item.EmojiName+" "+item.Emoji), number)
		})
}

func newIssueInspection[T any](f *cmdutil.Factory, name, description string, list func(*api.Client, string, string, string, int) ([]T, error), toJSON func(T) any, detail, header string, row func(io.Writer, string, T)) *cobra.Command {
	opts := &inspectionOptions{Limit: 30}
	cmd := &cobra.Command{
		Use: name + " [<owner>/<repo>] <number>", Short: description,
		Long:    description + ".\n\n" + detail,
		Example: fmt.Sprintf("  ag issue %s owner/repo 42\n  ag issue %s 42 --limit 100 --json", name, name),
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number, err := parseIssueNumber(remaining[0])
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
			items, err := list(client, repository.Owner, repository.Name, number, opts.Limit)
			if err != nil {
				return fmt.Errorf("failed to list issue #%s %s: %w", number, name, err)
			}
			if opts.JSON {
				rows := make([]any, 0, len(items))
				for _, item := range items {
					rows = append(rows, toJSON(item))
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), rows)
			}
			if len(items) == 0 {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "No %s found for issue #%s.\n", name, number)
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, header)
			for _, item := range items {
				row(w, number, item)
			}
			return w.Flush()
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of entries to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output entries as JSON")
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func inspectionCell(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "-"
	}
	return value
}

func issueActorName(user api.IssueActor) string {
	if strings.TrimSpace(user.Login) != "" {
		return inspectionCell(user.Login)
	}
	return inspectionCell(user.Name)
}
