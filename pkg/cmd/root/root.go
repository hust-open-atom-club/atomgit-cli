package root

import (
	"github.com/shinwell/ag-cli/pkg/cmd/auth"
	"github.com/shinwell/ag-cli/pkg/cmd/issue"
	"github.com/shinwell/ag-cli/pkg/cmd/license"
	"github.com/shinwell/ag-cli/pkg/cmd/pr"
	"github.com/shinwell/ag-cli/pkg/cmd/repo"
	"github.com/shinwell/ag-cli/pkg/cmd/ssh-key"
	"github.com/shinwell/ag-cli/pkg/cmd/tag"
	"github.com/shinwell/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdRoot(f *cmdutil.Factory) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "ag <command> <subcommand> [flags]",
		Short: "AtomGit CLI",
		Long:  `Work seamlessly with AtomGit from the command line.`,
	}

	cmd.PersistentFlags().Bool("help", false, "Show help for command")

	// Add commands
	cmd.AddCommand(repo.NewCmdRepo(f))
	cmd.AddCommand(pr.NewCmdPR(f))
	cmd.AddCommand(issue.NewCmdIssue(f))
	cmd.AddCommand(tag.NewCmdTag(f))
	cmd.AddCommand(auth.NewCmdAuth(f))
	cmd.AddCommand(key.NewCmdSSHKey(f))
	cmd.AddCommand(license.NewCmdLicense(f))

	return cmd, nil
}
