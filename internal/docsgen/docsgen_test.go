package docsgen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGenerateIsDeterministicAndIncludesCommandMetadata(t *testing.T) {
	newRoot := func() *cobra.Command {
		root := &cobra.Command{Use: "ag <command>", Short: "Root"}
		root.PersistentFlags().String("host", "atomgit.com", "API host")
		child := &cobra.Command{
			Use:     "repo [name]",
			Short:   "Manage repositories",
			Long:    "Long repository description",
			Aliases: []string{"r"},
			Example: "  ag repo demo",
		}
		child.Flags().BoolP("json", "j", false, "Output JSON")
		root.AddCommand(child)
		return root
	}

	var first, second bytes.Buffer
	if err := Generate(&first, newRoot()); err != nil {
		t.Fatal(err)
	}
	if err := Generate(&second, newRoot()); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Fatal("generation is not deterministic")
	}
	output := first.String()
	if strings.HasSuffix(output, "\n\n") {
		t.Fatal("generated output ends with a blank line")
	}
	for _, want := range []string{
		"# AtomGit CLI command reference",
		"- [ag repo](#ag-repo) — Manage repositories",
		"Usage: `ag repo [name] [flags]`",
		"Long repository description",
		"Aliases: `r`",
		"`-j, --json`",
		"`--host`",
		"Scope",
		"ag repo demo",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("generated output does not contain %q:\n%s", want, output)
		}
	}
}

func TestGenerateRejectsNilRoot(t *testing.T) {
	if err := Generate(&bytes.Buffer{}, nil); err == nil {
		t.Fatal("Generate(nil) succeeded")
	}
}

func TestGenerateEscapesIndexAndFlagCells(t *testing.T) {
	root := &cobra.Command{Use: "ag", Short: "Root"}
	root.PersistentFlags().String("query", "a|b", "Filter [name] | value\ncontinued")
	root.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List [items]",
	})

	var output bytes.Buffer
	if err := Generate(&output, root); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, `List \[items\]`) {
		t.Errorf("index description was not escaped: %s", text)
	}
	if !strings.Contains(text, "Filter [name] \\| value continued") {
		t.Errorf("flag cell was not escaped and normalized: %s", text)
	}
}

func TestGenerateOrdersCommandsAndFlagsAndOmitsEmptySections(t *testing.T) {
	root := &cobra.Command{Use: "ag", Short: "Root"}
	root.Flags().Bool("zulu", false, "Zulu")
	root.Flags().Bool("alpha", false, "Alpha")
	root.AddCommand(
		&cobra.Command{Use: "zulu", Short: "Zulu"},
		&cobra.Command{Use: "alpha", Short: "Alpha"},
	)

	var output bytes.Buffer
	if err := Generate(&output, root); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if strings.Index(text, "- [ag alpha]") > strings.Index(text, "- [ag zulu]") {
		t.Fatalf("commands are not ordered: %s", text)
	}
	if strings.Index(text, "`--alpha`") > strings.Index(text, "`--zulu`") {
		t.Fatalf("flags are not ordered: %s", text)
	}
	section := text[strings.Index(text, "## ag alpha"):]
	if strings.Contains(section, "### Flags") {
		t.Fatalf("command without local options unexpectedly has a flags section: %s", section)
	}
}
