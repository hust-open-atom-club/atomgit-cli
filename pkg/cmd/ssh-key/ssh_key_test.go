package key

import (
	"bytes"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestNewCmdSSHKey(t *testing.T) {
	cmd := NewCmdSSHKey(&cmdutil.Factory{})
	want := map[string][]string{
		"add":    {"title"},
		"list":   {"limit"},
		"delete": {"yes"},
	}
	for name, flags := range want {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child.Name() != name {
			t.Fatalf("subcommand %q: %v", name, err)
		}
		for _, flag := range flags {
			if child.Flags().Lookup(flag) == nil {
				t.Fatalf("%s --%s flag was not registered", name, flag)
			}
		}
		if name != "add" && !strings.Contains(child.Example, "ag ssh-key "+name) {
			t.Fatalf("%s example = %q", name, child.Example)
		}
	}

	add, _, _ := cmd.Find([]string{"add"})
	if err := add.Args(add, nil); err != nil {
		t.Fatalf("add rejected stdin mode: %v", err)
	}
	if err := add.Args(add, []string{"one", "two"}); err == nil {
		t.Fatal("add accepted too many key files")
	}
	list, _, _ := cmd.Find([]string{"list"})
	if err := list.Args(list, []string{"unexpected"}); err == nil {
		t.Fatal("list accepted an argument")
	}
	deleteCmd, _, _ := cmd.Find([]string{"delete"})
	if err := deleteCmd.Args(deleteCmd, nil); err == nil {
		t.Fatal("delete accepted no ID")
	}
	if err := deleteCmd.Args(deleteCmd, []string{"1", "2"}); err == nil {
		t.Fatal("delete accepted multiple IDs")
	}

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"add", "list", "delete"} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("help missing %q:\n%s", text, out.String())
		}
	}
}
