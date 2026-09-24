package agcmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/browser"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/root"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func Main() int {
	factory := &cmdutil.Factory{
		BrowserOpener:      browser.NewSyncOpener(),
		RepositoryResolver: cmdutil.NewGitRepositoryResolver(""),
	}

	rootCmd, err := root.NewCmdRoot(factory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create root command: %s\n", err)
		return 1
	}

	args := os.Args[1:]
	expanded, err := root.ExpandAlias(rootCmd, args)
	if err != nil {
		fmt.Fprintf(rootCmd.ErrOrStderr(), "%s\n", err)
		return 1
	}
	previousPersistentPreRunE := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if previousPersistentPreRunE != nil {
			if err := previousPersistentPreRunE(cmd, args); err != nil {
				return err
			}
		}
		if err := loadCommandConfig(cmd, factory, config.NewConfig); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	}
	rootCmd.SetArgs(expanded)

	ctx, stopSignals := newSignalContext(context.Background(), signal.NotifyContext)
	defer stopSignals()
	executeErr := rootCmd.ExecuteContext(ctx)
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

func loadCommandConfig(selected *cobra.Command, factory *cmdutil.Factory, load func() (config.Config, error)) error {
	// Cobra handles help flags and the root --version flag before running
	// PersistentPreRunE. These commands also do not need credentials when they
	// run normally. Doctor inspects credentials itself; schema uses static
	// metadata only.
	if selected != nil {
		switch selected.Name() {
		case "doctor", "help", "schema", "version":
			return nil
		}
	}
	cfg, err := load()
	if err != nil {
		return err
	}
	factory.Config = cfg
	return nil
}

func isExtensionCommand(rootCmd *cobra.Command, args []string) bool {
	c, _, err := rootCmd.Find(args)
	return err == nil && c != nil && c.GroupID == "extension"
}
