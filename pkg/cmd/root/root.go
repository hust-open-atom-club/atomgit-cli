package root

import (
	"os"

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
		Use:   "ag <command> <subcommand> [flags]",
		Short: "AtomGit CLI",
		Long:  `Work seamlessly with AtomGit from the command line.`,
	}

	cmd.PersistentFlags().Bool("help", false, "Show help for command")

	// Sanitize terminal output at the boundary: when stdout/stderr is a TTY,
	// wrap with SanitizingWriter so every subcommand is protected against
	// terminal escape sequence injection (CWE-150) without explicit calls.
	if cmdutil.IsTerminal(os.Stdout) {
		cmd.SetOut(cmdutil.NewSanitizingWriter(os.Stdout))
	}
	if cmdutil.IsTerminal(os.Stderr) {
		cmd.SetErr(cmdutil.NewSanitizingWriter(os.Stderr))
	}

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
