package agcmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestIsExtensionCommand(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	extension := &cobra.Command{Use: "extension", GroupID: "extension"}
	regular := &cobra.Command{Use: "regular"}
	root.AddCommand(extension, regular)

	if !isExtensionCommand(root, []string{"extension"}) {
		t.Fatal("extension command was not recognized")
	}
	if isExtensionCommand(root, []string{"regular"}) {
		t.Fatal("regular command was recognized as an extension")
	}
	if isExtensionCommand(root, []string{"missing"}) {
		t.Fatal("missing command was recognized as an extension")
	}
}
