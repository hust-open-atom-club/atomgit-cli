package notification

import (
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmdNotification returns the notification command group.
func NewCmdNotification(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notification",
		Short: "Manage repository notifications",
		Long:  `List repository notifications and mark them as read.`,
	}

	cmd.AddCommand(newCmdNotificationList(f))
	cmd.AddCommand(newCmdNotificationMarkRead(f))
	cmdutil.AddRepositoryContextHelp(cmd)

	return cmd
}
