package comment

import (
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
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
