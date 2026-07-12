package key

import (
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestNewCmdSSHKey(t *testing.T) {
	cmd := NewCmdSSHKey(&cmdutil.Factory{})
	add, _, err := cmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	if add.Flags().Lookup("title") == nil {
		t.Fatal("add --title flag was not registered")
	}
	if err := add.Args(add, nil); err != nil {
		t.Fatalf("add rejected stdin mode: %v", err)
	}
	if err := add.Args(add, []string{"one", "two"}); err == nil {
		t.Fatal("add accepted too many key files")
	}
}
