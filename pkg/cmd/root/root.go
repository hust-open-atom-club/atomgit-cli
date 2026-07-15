package root

import (
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/auth"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/issue"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/license"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/pr"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/repo"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/ssh-key"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/tag"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/version"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdRoot(f *cmdutil.Factory) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:     "ag <command> <subcommand> [flags]",
		Short:   "AtomGit CLI",
		Long:    `Work seamlessly with AtomGit from the command line.`,
		Version: version.Text(),
	}
	cmd.SetVersionTemplate(`{{.Version}}`)
	cmd.Flags().Bool("version", false, "Show version information")

	cmd.PersistentFlags().Bool("help", false, "Show help for command")

	// Add commands
	cmd.AddCommand(repo.NewCmdRepo(f))
	cmd.AddCommand(pr.NewCmdPR(f))
	cmd.AddCommand(issue.NewCmdIssue(f))
	cmd.AddCommand(tag.NewCmdTag(f))
	cmd.AddCommand(auth.NewCmdAuth(f))
	cmd.AddCommand(key.NewCmdSSHKey(f))
	cmd.AddCommand(license.NewCmdLicense(f))
	cmd.AddCommand(version.NewCmdVersion())

	return cmd, nil
}
