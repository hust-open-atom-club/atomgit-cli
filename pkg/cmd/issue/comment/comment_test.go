package comment

import (
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func TestNewCmdCommentRegistersSubcommands(t *testing.T) {
	cmd := NewCmdComment(&cmdutil.Factory{})
	want := map[string]bool{"create": false, "delete": false, "edit": false, "view": false}
	for _, child := range cmd.Commands() {
		if _, ok := want[child.Name()]; ok {
			want[child.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q was not registered", name)
		}
	}

	create, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	if create.Flags().Lookup("body") == nil || create.Flags().Lookup("body-file") == nil {
		t.Fatal("create body flags were not registered")
	}
}

func TestIssueCommentCommandsValidateIdentifiersBeforeAuthentication(t *testing.T) {
	factory := &cmdutil.Factory{Config: issueCommentTestConfig{tokenErr: config.ErrNotAuthenticated}}
	tests := []struct {
		name    string
		command func(*cmdutil.Factory) *cobra.Command
		args    []string
		prepare func(*cobra.Command)
		want    string
	}{
		{name: "view issue", command: newCmdView, args: []string{"alice/demo", "bad"}, want: "invalid issue number"},
		{name: "create issue", command: newCmdCreate, args: []string{"alice/demo", "bad"}, prepare: func(cmd *cobra.Command) { _ = cmd.Flags().Set("body", "test") }, want: "invalid issue number"},
		{name: "edit comment", command: newCmdEdit, args: []string{"alice/demo", "1", "bad"}, prepare: func(cmd *cobra.Command) { _ = cmd.Flags().Set("body", "test") }, want: "invalid comment ID"},
		{name: "delete comment", command: newCmdDelete, args: []string{"alice/demo", "1", "bad"}, want: "invalid comment ID"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := test.command(factory)
			if test.prepare != nil {
				test.prepare(cmd)
			}
			err := cmd.RunE(cmd, test.args)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}
