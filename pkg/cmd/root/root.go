package root

import (
	"fmt"
	"io"
	"os"

	apiCmd "atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/auth"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/branch"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/browse"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/issue"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/label"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/license"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/milestone"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/pr"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/release"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/repo"
	runcmd "atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/run"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/search"
	key "atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/ssh-key"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/tag"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/version"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdRoot(f *cmdutil.Factory) (*cobra.Command, error) {
	return newCmdRootWithWriters(f, os.Stdout, os.Stderr)
}

func newCmdRootWithWriters(f *cmdutil.Factory, stdout, stderr io.Writer) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:     "ag <command> <subcommand> [flags]",
		Short:   "AtomGit CLI",
		Long:    `Work seamlessly with AtomGit from the command line.`,
		Version: version.Text(),
	}
	cmd.SetVersionTemplate(`{{.Version}}`)
	cmd.Flags().Bool("version", false, "Show version information")

	cmd.PersistentFlags().Bool("help", false, "Show help for command")

	// Sanitize by default even when output is piped: a downstream program may
	// forward bytes to a terminal. Machine consumers can explicitly opt into
	// byte-for-byte output with --raw-output.
	safeOut := cmdutil.NewSanitizingWriter(stdout)
	safeErr := cmdutil.NewSanitizingWriter(stderr)
	cmd.SetOut(safeOut)
	cmd.SetErr(safeErr)
	var rawOutput bool
	cmd.PersistentFlags().BoolVar(&rawOutput, "raw-output", false, "Disable terminal output sanitization for machine processing")
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		if err := safeOut.SetRaw(rawOutput); err != nil {
			return fmt.Errorf("configure stdout: %w", err)
		}
		if err := safeErr.SetRaw(rawOutput); err != nil {
			return fmt.Errorf("configure stderr: %w", err)
		}
		return nil
	}

	// Add commands
	cmd.AddCommand(apiCmd.NewCmdAPI(f))
	cmd.AddCommand(repo.NewCmdRepo(f))
	cmd.AddCommand(branch.NewCmdBranch(f))
	cmd.AddCommand(pr.NewCmdPR(f))
	cmd.AddCommand(issue.NewCmdIssue(f))
	cmd.AddCommand(label.NewCmdLabel(f))
	cmd.AddCommand(milestone.NewCmdMilestone(f))
	cmd.AddCommand(release.NewCmdRelease(f))
	cmd.AddCommand(runcmd.NewCmdRun(f))
	cmd.AddCommand(tag.NewCmdTag(f))
	cmd.AddCommand(auth.NewCmdAuth(f))
	cmd.AddCommand(browse.NewCmdBrowse(f))
	cmd.AddCommand(key.NewCmdSSHKey(f))
	cmd.AddCommand(license.NewCmdLicense(f))
	cmd.AddCommand(search.NewCmdSearch(f))
	cmd.AddCommand(version.NewCmdVersion())

	return cmd, nil
}
