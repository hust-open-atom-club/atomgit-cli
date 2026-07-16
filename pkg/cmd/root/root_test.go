package root

import (
	"bytes"
	"strings"
	"testing"

	internalversion "atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	versioncmd "atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/version"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
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
		"repo": false, "run": false, "ssh-key": false, "tag": false, "version": false,
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
}
