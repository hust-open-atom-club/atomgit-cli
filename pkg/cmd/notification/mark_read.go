package notification

import (
	"fmt"
	"io"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdNotificationMarkRead(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		all bool
		yes bool
	}

	cmd := &cobra.Command{
		Use:   "mark-read [<owner>/<repo>] [<notification-id>...]",
		Short: "Mark repository notifications as read",
		Long: `Mark notifications for a repository as read.

Pass one or more notification IDs (as shown by "ag notification list") to
mark exactly those notifications, or pass --all to mark every unread
notification in the repository. --all asks for confirmation unless --yes is
supplied.`,
		Example: `  ag notification mark-read owner/repo 292ecbec857e4f27b426d66f2157938c
  ag notification mark-read --all --yes`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Notification IDs are plain hex strings, so an argument
			// containing a slash can only be the optional repository.
			var repositoryArgs, ids []string
			if len(args) > 0 && strings.Contains(args[0], "/") {
				repositoryArgs = args[:1]
				ids = args[1:]
			} else {
				ids = args
			}

			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, repositoryArgs, 0)
			if err != nil {
				return err
			}

			if opts.all && len(ids) > 0 {
				return fmt.Errorf("cannot combine --all with explicit notification IDs")
			}
			if !opts.all && len(ids) == 0 {
				return fmt.Errorf("specify at least one notification ID or use --all")
			}
			for _, id := range ids {
				if strings.TrimSpace(id) == "" {
					return fmt.Errorf("notification IDs must not be empty")
				}
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if opts.all {
				notifications, err := api.ListNotifications(client, repository.Owner, repository.Name, api.NotificationListOptions{
					Limit:      maxMarkAllNotifications,
					UnreadOnly: true,
				})
				if err != nil {
					return fmt.Errorf("failed to list notifications: %w", err)
				}
				if len(notifications) == 0 {
					fmt.Fprintln(out, "No unread notifications")
					return nil
				}
				if !opts.yes {
					confirmed, err := confirmMarkAll(cmd.InOrStdin(), out, len(notifications), repository.String())
					if err != nil {
						return err
					}
					if !confirmed {
						fmt.Fprintln(out, "Cancelled")
						return nil
					}
				}
				ids = notificationIDs(notifications)
			}

			if err := api.MarkNotificationsRead(client, repository.Owner, repository.Name, ids); err != nil {
				return fmt.Errorf("failed to mark notifications as read: %w", err)
			}

			fmt.Fprintf(out, "Marked %d notification(s) as read\n", len(ids))
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.all, "all", false, "Mark every unread notification in the repository")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip the --all confirmation prompt")

	return cmd
}

// maxMarkAllNotifications bounds how many unread notifications --all collects
// before marking them read.
const maxMarkAllNotifications = 500

// confirmMarkAll asks the user to confirm marking count unread notifications
// as read. It accepts y, Y, yes, and YES; anything else declines.
func confirmMarkAll(in io.Reader, out io.Writer, count int, repository string) (bool, error) {
	fmt.Fprintf(out, "Mark %d unread notification(s) as read in %s? [y/N] ", count, repository)
	var response string
	if _, err := fmt.Fscan(in, &response); err != nil && err != io.EOF {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	switch strings.ToLower(response) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func notificationIDs(notifications []api.Notification) []string {
	ids := make([]string, len(notifications))
	for index, notification := range notifications {
		ids[index] = notification.ID
	}
	return ids
}
