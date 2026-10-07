package mangen_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/mangen"
	rootcmd "atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/root"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func pageMap(t *testing.T, root *cobra.Command) map[string]string {
	t.Helper()
	pages, err := mangen.Generate(root, mangen.Header{Version: "v1.2.3", Date: "2026-10-07"})
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]string)
	var names []string
	for _, page := range pages {
		out[page.Name] = string(page.Data)
		names = append(names, page.Name)
	}
	if !sort.StringsAreSorted(names) {
		t.Fatal("manual filenames are not sorted")
	}
	return out
}

func TestMetadataPolicyAndNavigation(t *testing.T) {
	fail := func(*cobra.Command, []string) error { t.Fatal("executed command or hook"); return nil }
	root := &cobra.Command{Use: "ag-cli <command>", Short: "Root", RunE: fail, PersistentPreRunE: fail}
	root.PersistentFlags().String("host", "atomgit.com", "API host")
	root.PersistentFlags().Bool("hidden", false, "must not appear")
	if err := root.PersistentFlags().MarkHidden("hidden"); err != nil {
		t.Fatal(err)
	}
	child := &cobra.Command{Use: "zebra [name]", Short: "Example", Long: ".SH injected\n'break\npath \\server - option", Aliases: []string{"z", "a"}, Example: "  ag-cli zebra --file x\n    .literal\n\n  'literal", RunE: fail}
	child.Flags().StringP("file", "f", "C:\\tmp", "File path")
	child.Flags().Bool("legacy", false, "Visible deprecated flag")
	child.Flags().Lookup("legacy").Deprecated = "use --file instead"
	child.Flags().StringP("retired-short", "r", "", "Flag with retired shorthand")
	child.Flags().Lookup("retired-short").ShorthandDeprecated = "use long form"
	child.Flags().Int("optional", 42, "Optional argument")
	child.Flags().Lookup("optional").NoOptDefVal = "7"
	root.AddCommand(child, &cobra.Command{Use: "alpha"}, &cobra.Command{Use: "old", Deprecated: "use zebra"})
	hidden := &cobra.Command{Use: "secret", Hidden: true}
	hidden.AddCommand(&cobra.Command{Use: "visible"})
	root.AddCommand(hidden)
	pages := pageMap(t, root)
	if len(pages) != 3 {
		t.Fatalf("pages = %v", pages)
	}
	for _, want := range []string{`.SH SYNOPSIS`, `ag\-cli zebra [name] [flags]`, `.SH INHERITED OPTIONS`, `atomgit.com`, `.SH ALIASES`, `a, z`, `.SH EXAMPLES`, `.SH SEE ALSO`, `Default: \(dqC:\etmp\(dq.`, `Deprecated: use \-\-file instead`, `Value when specified without an argument: \(dq7\(dq.`, `Deprecated shorthand \-r: use long form`, `\&.SH injected`, `\&'break`, `path \eserver \- option`, `\&    .literal`} {
		if !strings.Contains(pages["ag-cli-zebra.1"], want) {
			t.Errorf("child manual missing %q", want)
		}
	}
	if strings.Contains(pages["ag-cli-zebra.1"], "must not appear") {
		t.Fatal("hidden flag was included")
	}
	if strings.Contains(pages["ag-cli-alpha.1"], ".SH OPTIONS") {
		t.Fatal("no-local-options command has an options section")
	}
	for _, data := range pages {
		refs := regexp.MustCompile(`\\fB([^\n]+)\\fR\(1\)`).FindAllStringSubmatch(data, -1)
		for _, ref := range refs {
			name := strings.ReplaceAll(ref[1], `\-`, "-")
			if _, ok := pages[name+".1"]; !ok {
				t.Errorf("dangling SEE ALSO reference %s", name)
			}
		}
	}
}

type forbiddenConfig struct{}

func (forbiddenConfig) GetToken() (string, error) { panic("read credentials") }
func (forbiddenConfig) GetUser() (string, error)  { panic("read account") }
func (forbiddenConfig) GetHost() string           { panic("read host") }

func TestActualTreeIsOfflineCompleteAndDeterministic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "ag-cli"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "ag-cli", "token.json"), []byte("broken config"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := rootcmd.NewCmdRoot(&cmdutil.Factory{Config: forbiddenConfig{}})
	if err != nil {
		t.Fatal(err)
	}
	first := pageMap(t, root)
	second := pageMap(t, root)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("generation is not deterministic")
	}
	var visit func(*cobra.Command)
	visit = func(cmd *cobra.Command) {
		if cmd.Hidden || cmd.Deprecated != "" {
			return
		}
		name := strings.ReplaceAll(cmd.CommandPath(), " ", "-") + ".1"
		if _, ok := first[name]; !ok {
			t.Errorf("missing public command %s", cmd.CommandPath())
		}
		for _, child := range cmd.Commands() {
			visit(child)
		}
	}
	visit(root)
	for _, excluded := range []string{"ag-cli-auth-git-credential.1", "ag-cli-check-update.1", "ag-cli-completion.1"} {
		if _, ok := first[excluded]; ok {
			t.Errorf("included internal/deprecated/helper command %s", excluded)
		}
	}
	for _, representative := range []string{"ag-cli.1", "ag-cli-auth-login.1", "ag-cli-pr-create.1"} {
		if _, ok := first[representative]; !ok {
			t.Errorf("missing %s", representative)
		}
	}
}

func TestInvalidInputs(t *testing.T) {
	for _, root := range []*cobra.Command{nil, {Use: "../escape"}, {Use: "root", Hidden: true}, {Use: "root", Deprecated: "old"}} {
		if _, err := mangen.Generate(root, mangen.Header{}); err == nil {
			t.Fatal("accepted invalid root")
		}
	}
	root := &cobra.Command{Use: "ag-cli"}
	for _, header := range []mangen.Header{{Date: "today"}, {Version: "\n.SH injection"}} {
		if _, err := mangen.Generate(root, header); err == nil {
			t.Fatal("accepted invalid header")
		}
	}
	parent := &cobra.Command{Use: "a"}
	parent.AddCommand(&cobra.Command{Use: "b"})
	root.AddCommand(parent, &cobra.Command{Use: "a-b"})
	if _, err := mangen.Generate(root, mangen.Header{}); err == nil {
		t.Fatal("accepted colliding filenames")
	}
}

func TestWriteAndCheck(t *testing.T) {
	pages, err := mangen.Generate(&cobra.Command{Use: "ag-cli"}, mangen.Header{})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "man1")
	if err := mangen.Write(dir, pages, false); err != nil {
		t.Fatal(err)
	}
	if err := mangen.Write(dir, pages, true); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ag-cli.1"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mangen.Write(dir, pages, true); err == nil {
		t.Fatal("accepted changed manual")
	}
	if err := os.WriteFile(filepath.Join(dir, "old.1"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mangen.Write(dir, pages, false); err == nil {
		t.Fatal("silently kept or deleted stale manual")
	}
	if err := mangen.Write(t.TempDir(), pages, true); err == nil {
		t.Fatal("accepted missing manual")
	}
}

func TestManParser(t *testing.T) {
	mandoc, err := exec.LookPath("mandoc")
	if err != nil {
		t.Skip("mandoc is not installed; CI installs it explicitly")
	}
	root, err := rootcmd.NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range pageMap(t, root) {
		cmd := exec.Command(mandoc, "-Tlint", "-Werror")
		cmd.Stdin = strings.NewReader(data)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("mandoc rejected %s: %v\n%s", name, err, output)
		}
	}
	cmd := exec.Command(mandoc, "-Tascii")
	cmd.Stdin = strings.NewReader(pageMap(t, root)["ag-cli-pr-create.1"])
	out, err := cmd.Output()
	if err != nil || !bytes.Contains(out, []byte("ag-cli pr create")) {
		t.Fatalf("render manual: %v", err)
	}
}

func TestManReadsTemporaryPrefix(t *testing.T) {
	man, err := exec.LookPath("man")
	if err != nil {
		t.Skip("man is not installed")
	}
	if _, err := exec.LookPath("mandoc"); err != nil {
		if _, err := exec.LookPath("groff"); err != nil {
			t.Skip("no manual formatter is installed")
		}
	}
	root, err := rootcmd.NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		t.Fatal(err)
	}
	pages, err := mangen.Generate(root, mangen.Header{})
	if err != nil {
		t.Fatal(err)
	}
	prefix := t.TempDir()
	manRoot := filepath.Join(prefix, "share", "man")
	if err := mangen.Write(filepath.Join(manRoot, "man1"), pages, false); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MANPATH", manRoot)
	t.Setenv("MANPAGER", "cat")
	t.Setenv("PAGER", "cat")
	t.Setenv("MANOPT", "")
	for _, name := range []string{"ag-cli", "ag-cli-auth-login", "ag-cli-pr-create"} {
		cmd := exec.Command(man, "-P", "cat", "1", name)
		output, err := cmd.CombinedOutput()
		if err != nil || len(output) == 0 {
			t.Fatalf("read %s from temporary prefix: %v\n%s", name, err, output)
		}
	}
}
