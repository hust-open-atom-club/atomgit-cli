package agcmd

import (
	"context"
	"fmt"
	"os"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/root"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func Main() int {
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %s\n", err)
		return 1
	}

	factory := &cmdutil.Factory{
		Config:             cfg,
		BrowserOpener:      browser.NewSyncOpener(),
		RepositoryResolver: cmdutil.NewGitRepositoryResolver(""),
	}

	rootCmd, err := root.NewCmdRoot(factory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create root command: %s\n", err)
		return 1
	}

	executeErr := rootCmd.ExecuteContext(context.Background())
	stdoutFlushErr := cmdutil.FlushWriter(rootCmd.OutOrStdout())
	stderrFlushErr := cmdutil.FlushWriter(rootCmd.ErrOrStderr())
	if executeErr != nil {
		// Error may contain raw API response body; write through the
		// sanitizing stderr writer that root.NewCmdRoot configured.
		fmt.Fprintf(rootCmd.ErrOrStderr(), "%s\n", executeErr)
		_ = cmdutil.FlushWriter(rootCmd.ErrOrStderr())
		return 1
	}
	if stdoutFlushErr != nil {
		fmt.Fprintf(rootCmd.ErrOrStderr(), "failed to flush stdout: %s\n", stdoutFlushErr)
		_ = cmdutil.FlushWriter(rootCmd.ErrOrStderr())
		return 1
	}
	if stderrFlushErr != nil {
		fmt.Fprintf(os.Stderr, "failed to flush stderr: %s\n", stderrFlushErr)
		return 1
	}

	return 0
}

func isExtensionCommand(rootCmd *cobra.Command, args []string) bool {
	c, _, err := rootCmd.Find(args)
	return err == nil && c != nil && c.GroupID == "extension"
}
