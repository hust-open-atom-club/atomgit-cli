package comment

import (
<<<<<<< HEAD
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
=======
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
>>>>>>> 4ec08c7 (fix: update module path to atomgit.com/openeuler/ag-cli)
	"github.com/spf13/cobra"
)

func NewCmdComment(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage pull request comments",
		Long:  `Create, view, edit, delete, and reply to pull request comments.`,
	}

	cmd.AddCommand(newCmdCreate(f))
	cmd.AddCommand(newCmdView(f))
	cmd.AddCommand(newCmdEdit(f))
	cmd.AddCommand(newCmdDelete(f))
	cmd.AddCommand(newCmdReply(f))

	return cmd
}
