package tag

import (
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestNewCmdTagRegistersSubcommands(t *testing.T) {
	cmd := NewCmdTag(&cmdutil.Factory{})
	want := map[string]bool{"create": false, "delete": false, "list": false}
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
	if create.Flags().Lookup("message") == nil || create.Flags().Lookup("ref") == nil {
		t.Fatal("create flags were not registered")
	}
	if err := create.Args(create, []string{"owner/repo"}); err == nil {
		t.Fatal("create accepted too few arguments")
	}
	if err := create.Args(create, []string{"owner/repo", "v1.0.0"}); err != nil {
		t.Fatalf("create rejected valid arguments: %v", err)
	}
}
