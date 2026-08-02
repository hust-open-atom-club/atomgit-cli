package root

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/alias"
	apiCmd "atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/auth"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/branch"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/browse"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/issue"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/label"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/license"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/org"
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
	cmd.AddCommand(release.NewCmdRelease(f))
	cmd.AddCommand(runcmd.NewCmdRun(f))
	cmd.AddCommand(tag.NewCmdTag(f))
	cmd.AddCommand(auth.NewCmdAuth(f))
	cmd.AddCommand(browse.NewCmdBrowse(f))
	cmd.AddCommand(key.NewCmdSSHKey(f))
	cmd.AddCommand(license.NewCmdLicense(f))
	cmd.AddCommand(org.NewCmdOrg(f))
	cmd.AddCommand(search.NewCmdSearch(f))
	cmd.AddCommand(version.NewCmdVersion())
	cmd.AddCommand(alias.NewCmdAlias(f))

	return cmd, nil
}

// ExpandAlias replaces the first invocation argument with its configured
// alias expansion, if one exists. Built-in commands always take precedence
// over aliases with the same name, and root-level flags (e.g. --raw-output,
// --version) are skipped so that `ag --raw-output <alias>` still expands.
func ExpandAlias(cmd *cobra.Command, args []string) ([]string, error) {
	// Root-level flags precede the command word; skip them when locating
	// the token that may be an alias.
	idx := 0
	for idx < len(args) && strings.HasPrefix(args[idx], "-") {
		idx++
	}
	if idx >= len(args) {
		return args, nil
	}
	// A built-in command wins over an alias that happens to share its name.
	if c, _, err := cmd.Find(args); err == nil && c != nil && c != cmd {
		return args, nil
	}
	aliases, err := config.LoadAliases()
	if err != nil {
		// A corrupted alias file must not take the whole CLI down: warn and
		// fall back to no aliases so every other command keeps working. The
		// alias subcommands still surface the underlying error.
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to load aliases, continuing without them: %v\n", err)
		return args, nil
	}
	expansion, ok := aliases[args[idx]]
	if !ok {
		return args, nil
	}
	if strings.HasPrefix(expansion, "!") {
		return nil, errors.New("shell-style aliases (starting with '!') are not supported")
	}
	expanded := make([]string, 0, len(args)-1+idx)
	expanded = append(expanded, args[:idx]...)
	expanded = append(expanded, splitExpansion(expansion)...)
	expanded = append(expanded, args[idx+1:]...)
	return expanded, nil
}

// splitExpansion tokenizes an alias expansion on whitespace while honoring
// backslash-escaped spaces and tabs, so Windows paths such as
// "C:\Program\ Files" survive as a single token. Any other backslash
// sequence is kept verbatim.
func splitExpansion(s string) []string {
	var fields []string
	var cur strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' && i+1 < len(runes) && (runes[i+1] == ' ' || runes[i+1] == '\t') {
			cur.WriteRune(runes[i+1])
			i++
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' {
			if cur.Len() > 0 {
				fields = append(fields, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		fields = append(fields, cur.String())
	}
	return fields
}
