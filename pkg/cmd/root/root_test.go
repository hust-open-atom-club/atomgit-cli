package root

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	internalversion "atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	versioncmd "atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/version"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func setVersionMetadata(t *testing.T) {
	t.Helper()
	oldV, oldC, oldB := internalversion.Version, internalversion.Commit, internalversion.BuildDate
	internalversion.Version = "v1.2.3"
	internalversion.Commit = "abc1234"
	internalversion.BuildDate = "2026-07-15T00:00:00Z"
	t.Cleanup(func() {
		internalversion.Version = oldV
		internalversion.Commit = oldC
		internalversion.BuildDate = oldB
	})
}

func TestNewCmdRootRegistersCommands(t *testing.T) {
	cmd, err := NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"auth": false, "issue": false, "label": false, "license": false, "pr": false,
		"repo": false, "ssh-key": false, "tag": false, "version": false,
	}
	for _, child := range cmd.Commands() {
		if _, ok := want[child.Name()]; ok {
			want[child.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("command %q was not registered", name)
		}
	}
	if cmd.Use != "ag <command> <subcommand> [flags]" {
		t.Fatalf("Use = %q", cmd.Use)
	}
	if cmd.PersistentFlags().Lookup("help") == nil {
		t.Fatal("persistent help flag was not registered")
	}
	versionFlag := cmd.Flags().Lookup("version")
	if versionFlag == nil {
		t.Fatal("version flag was not registered")
	}
	if versionFlag.Shorthand != "" {
		t.Fatalf("version shorthand = %q, want none", versionFlag.Shorthand)
	}
}

func TestNewCmdRootVersionMatchesVersionCommand(t *testing.T) {
	setVersionMetadata(t)

	rootCmd, err := NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		t.Fatal(err)
	}
	var rootOut bytes.Buffer
	rootCmd.SetOut(&rootOut)
	rootCmd.SetArgs([]string{"--version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v", err)
	}

	versionCmd := versioncmd.NewCmdVersion()
	var versionOut bytes.Buffer
	versionCmd.SetOut(&versionOut)
	if err := versionCmd.Execute(); err != nil {
		t.Fatalf("version Execute() error = %v", err)
	}

	if got, want := rootOut.String(), versionOut.String(); got != want {
		t.Errorf("--version output = %q, version output = %q", got, want)
	}
}

func TestNewCmdRootVersionHelp(t *testing.T) {
	cmd, err := NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "--version") {
		t.Errorf("help output does not mention --version: %s", out.String())
	}
	if !strings.Contains(out.String(), "--raw-output") {
		t.Errorf("help output does not mention --raw-output: %s", out.String())
	}
}

func TestRootSanitizesPipedOutputByDefault(t *testing.T) {
	payload := "unsafe \x1b]52;c;attack\x07\n"
	var stdout, stderr bytes.Buffer
	cmd, err := newCmdRootWithWriters(&cmdutil.Factory{}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	cmd.AddCommand(&cobra.Command{
		Use: "emit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), payload)
			return err
		},
	})
	cmd.SetArgs([]string{"emit"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := cmdutil.FlushWriter(cmd.OutOrStdout()); err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), "unsafe \\x1b]52;c;attack\\x07\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRootRawOutputIsExplicitOptOut(t *testing.T) {
	payload := "raw \x1b[31mtext\x1b[0m\n"
	var stdout, stderr bytes.Buffer
	cmd, err := newCmdRootWithWriters(&cmdutil.Factory{}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	cmd.AddCommand(&cobra.Command{
		Use: "emit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), payload)
			return err
		},
	})
	cmd.SetArgs([]string{"--raw-output", "emit"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != payload {
		t.Fatalf("stdout = %q, want raw %q", got, payload)
	}
}
