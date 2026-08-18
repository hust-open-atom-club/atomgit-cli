package notification

import (
	"fmt"
	"io"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdNotificationList(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		unread  bool
		nType   string
		since   string
		before  string
		limit   int
		jsonOut bool
	}

	cmd := &cobra.Command{
		Use:   "list [<owner>/<repo>]",
		Short: "List repository notifications",
		Long: `List notifications for a repository, most recent first.

By default all notifications are listed; --unread keeps only unread ones.
--since and --before accept RFC 3339 timestamps (for example
2026-08-14T00:00:00+08:00). --type keeps only notifications of one type,
such as merge_requests_open or issue_open.`,
		Example: `  ag notification list owner/repo --limit 20
  ag notification list --unread --json
  ag notification list owner/repo --type issue_open --since 2026-08-01T00:00:00Z`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}

			if opts.limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.limit)
			}
			if opts.since != "" {
				if _, err := time.Parse(time.RFC3339, opts.since); err != nil {
					return fmt.Errorf("invalid --since timestamp %q: must be RFC 3339 like 2026-08-14T00:00:00+08:00", opts.since)
				}
			}
			if opts.before != "" {
				if _, err := time.Parse(time.RFC3339, opts.before); err != nil {
					return fmt.Errorf("invalid --before timestamp %q: must be RFC 3339 like 2026-08-14T00:00:00+08:00", opts.before)
				}
			}

			token, err := f.Config.GetToken()
			if err != nil {
				// GetToken already returns the canonical login guidance for a
				// missing credential; wrapping it would duplicate the prefix.
				return err
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			notifications, err := api.ListNotifications(client, repository.Owner, repository.Name, api.NotificationListOptions{
				Limit:      opts.limit,
				UnreadOnly: opts.unread,
				Type:       opts.nType,
				Since:      opts.since,
				Before:     opts.before,
			})
			if err != nil {
				return fmt.Errorf("failed to list notifications: %w", err)
			}

			if opts.jsonOut {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), notificationsJSON(notifications))
			}
			printNotifications(cmd.OutOrStdout(), notifications)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.unread, "unread", false, "Only list unread notifications")
	cmd.Flags().StringVar(&opts.nType, "type", "", "Only list notifications of this type (for example merge_requests_open)")
	cmd.Flags().StringVar(&opts.since, "since", "", "Only list notifications updated at or after this RFC 3339 timestamp")
	cmd.Flags().StringVar(&opts.before, "before", "", "Only list notifications updated before this RFC 3339 timestamp")
	cmd.Flags().IntVarP(&opts.limit, "limit", "L", 30, "Maximum number of notifications to list")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output notifications as JSON")

	return cmd
}

// printNotifications writes the human-readable listing. Every row carries the
// unread state, type, update time, subject, and URL.
func printNotifications(out io.Writer, notifications []api.Notification) {
	if len(notifications) == 0 {
		fmt.Fprintln(out, "No notifications found")
		return
	}
	for _, notification := range notifications {
		unreadState := "read"
		if notification.Unread {
			unreadState = "unread"
		}
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", unreadState, notification.Type, notification.UpdateAt, notification.Content, notification.HTMLURL)
	}
}

type notificationJSON struct {
	ID        string `json:"id"`
	Unread    bool   `json:"unread"`
	Type      string `json:"type"`
	Subject   string `json:"subject"`
	UpdatedAt string `json:"updatedAt"`
	URL       string `json:"url"`
}

func notificationsJSON(notifications []api.Notification) []notificationJSON {
	result := make([]notificationJSON, len(notifications))
	for index, notification := range notifications {
		result[index] = notificationJSON{
			ID:        notification.ID,
			Unread:    notification.Unread,
			Type:      notification.Type,
			Subject:   notification.Content,
			UpdatedAt: notification.UpdateAt,
			URL:       notification.HTMLURL,
		}
	}
	return result
}
