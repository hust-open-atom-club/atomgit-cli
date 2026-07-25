package comment

import (
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func TestPRCommentCommandsValidateIdentifiersBeforeAuthentication(t *testing.T) {
	factory := &cmdutil.Factory{Config: prCommentTestConfig{tokenErr: config.ErrNotAuthenticated}}
	tests := []struct {
		name    string
		command func(*cmdutil.Factory) *cobra.Command
		args    []string
		prepare func(*cobra.Command)
		want    string
	}{
		{name: "view PR", command: newCmdView, args: []string{"alice/demo", "bad"}, want: "invalid PR number"},
		{name: "create PR", command: newCmdCreate, args: []string{"alice/demo", "bad"}, prepare: func(cmd *cobra.Command) { _ = cmd.Flags().Set("body", "test") }, want: "invalid PR number"},
		{name: "edit comment", command: newCmdEdit, args: []string{"alice/demo", "1", "bad"}, prepare: func(cmd *cobra.Command) { _ = cmd.Flags().Set("body", "test") }, want: "invalid comment ID"},
		{name: "delete comment", command: newCmdDelete, args: []string{"alice/demo", "1", "bad"}, want: "invalid comment ID"},
		{name: "reply PR", command: newCmdReply, args: []string{"alice/demo", "bad", "discussion"}, prepare: func(cmd *cobra.Command) { _ = cmd.Flags().Set("body", "test") }, want: "invalid PR number"},
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
