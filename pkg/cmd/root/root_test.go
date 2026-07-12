package root

import (
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestNewCmdRootRegistersCommands(t *testing.T) {
	cmd, err := NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"auth": false, "issue": false, "license": false, "pr": false,
		"repo": false, "ssh-key": false, "tag": false,
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
}
