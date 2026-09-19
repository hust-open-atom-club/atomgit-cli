package comment_test

import (
	"testing"

	commitcmd "atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/commit"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestCommitCommandRegistersCommentSubcommand(t *testing.T) {
	cmd := commitcmd.NewCmdCommit(&cmdutil.Factory{})
	comment, _, err := cmd.Find([]string{"comment"})
	if err != nil || comment == nil || comment.Name() != "comment" {
		t.Fatalf("comment subcommand not registered under commit: %v", err)
	}

	wantSubcommands := map[string]bool{
		"list":   false,
		"view":   false,
		"create": false,
		"edit":   false,
		"delete": false,
	}
	for _, sub := range comment.Commands() {
		if _, ok := wantSubcommands[sub.Name()]; ok {
			wantSubcommands[sub.Name()] = true
		}
	}
	for name, found := range wantSubcommands {
		if !found {
			t.Fatalf("subcommand %q missing under commit comment", name)
		}
	}
}
