// Package comment provides the ag commit comment subcommands for listing,
// viewing, creating, editing, and deleting comments on repository commits.
package comment

import (
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdComment(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage commit comments",
		Long:  `List, view, create, edit, and delete comments on repository commits.`,
	}

	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdView(f))
	cmd.AddCommand(newCmdCreate(f))
	cmd.AddCommand(newCmdEdit(f))
	cmd.AddCommand(newCmdDelete(f))

	return cmd
}
