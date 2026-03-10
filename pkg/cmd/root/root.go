package root

import (
	"gitcode.com/openeuler/ag-cli/pkg/cmd/auth"
	"gitcode.com/openeuler/ag-cli/pkg/cmd/issue"
	"gitcode.com/openeuler/ag-cli/pkg/cmd/license"
	"gitcode.com/openeuler/ag-cli/pkg/cmd/pr"
	"gitcode.com/openeuler/ag-cli/pkg/cmd/repo"
	"gitcode.com/openeuler/ag-cli/pkg/cmd/ssh-key"
	"gitcode.com/openeuler/ag-cli/pkg/cmd/tag"
	"gitcode.com/openeuler/ag-cli/pkg/cmdutil"
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
