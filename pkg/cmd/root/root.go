package root

import (
<<<<<<< HEAD
	"atomgit.com/openeuler/ag-cli/pkg/cmd/auth"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/issue"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/license"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/pr"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/repo"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/ssh-key"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/tag"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
=======
	"atomgit.com/openeuler/ag-cli/pkg/cmd/auth"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/issue"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/license"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/pr"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/repo"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/ssh-key"
	"atomgit.com/openeuler/ag-cli/pkg/cmd/tag"
	"atomgit.com/openeuler/ag-cli/pkg/cmdutil"
>>>>>>> 4ec08c7 (fix: update module path to atomgit.com/openeuler/ag-cli)
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
