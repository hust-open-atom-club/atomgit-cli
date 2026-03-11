package comment

import (
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdComment(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage issue comments",
		Long:  `Create, view, edit, and delete issue comments.`,
	}

	cmd.AddCommand(newCmdCreate(f))
	cmd.AddCommand(newCmdView(f))
	cmd.AddCommand(newCmdEdit(f))
	cmd.AddCommand(newCmdDelete(f))

	return cmd
}
