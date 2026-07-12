package issue

import (
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestNewCmdIssueRegistersSubcommands(t *testing.T) {
	cmd := NewCmdIssue(&cmdutil.Factory{})
	want := map[string]bool{"close": false, "comment": false, "create": false, "list": false, "view": false}
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

	list, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if list.Flags().Lookup("state") == nil || list.Flags().Lookup("limit") == nil {
		t.Fatal("list flags were not registered")
	}
	if err := list.Args(list, []string{"one", "two"}); err == nil {
		t.Fatal("list accepted too many arguments")
	}
}
