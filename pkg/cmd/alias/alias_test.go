package alias

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// newAliasTestRoot returns a minimal command tree with the commands used in
// alias set-tests registered alongside alias, so expansion-target validation
// resolves them.
func newAliasTestRoot() *cobra.Command {
	rootCmd := &cobra.Command{Use: "ag"}
	prCmd := &cobra.Command{Use: "pr"}
	prCmd.AddCommand(&cobra.Command{Use: "list"})
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(&cobra.Command{Use: "repo"})
	rootCmd.AddCommand(NewCmdAlias(&cmdutil.Factory{}))
	return rootCmd
}

func runAliasSet(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := &bytes.Buffer{}
	rootCmd := newAliasTestRoot()
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestValidateAliasName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "pl", want: true},
		{name: "rv-2", want: true},
		{name: "", want: false},
		{name: "-pl", want: false},
		{name: "pl list", want: false},
		{name: "pl\tlist", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAliasName(tt.name)
			if tt.want && err != nil {
				t.Fatalf("validateAliasName(%q) error = %v, want nil", tt.name, err)
			}
			if !tt.want && err == nil {
				t.Fatalf("validateAliasName(%q) = nil, want error", tt.name)
			}
		})
	}
}

func TestValidateAliasExpansion(t *testing.T) {
	tests := []struct {
		expansion string
		want      bool
	}{
		{expansion: "pr list", want: true},
		{expansion: "repo view --limit 5", want: true},
		{expansion: "", want: false},
		{expansion: "!", want: false},
		{expansion: "!echo hi", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.expansion, func(t *testing.T) {
			err := validateAliasExpansion(tt.expansion)
			if tt.want && err != nil {
				t.Fatalf("validateAliasExpansion(%q) error = %v, want nil", tt.expansion, err)
			}
			if !tt.want && err == nil {
				t.Fatalf("validateAliasExpansion(%q) = nil, want error", tt.expansion)
			}
		})
	}
}

func TestAliasSetCommand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	out, err := runAliasSet(t, "alias", "set", "pl", "pr list")
	if err != nil {
		t.Fatalf("alias set error = %v", err)
	}

	if !strings.Contains(out, "Added alias pl: pr list") {
		t.Errorf("output = %q, want to contain %q", out, "Added alias pl: pr list")
	}

	aliases, err := config.LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if aliases["pl"] != "pr list" {
		t.Errorf("aliases[pl] = %q, want %q", aliases["pl"], "pr list")
	}
}

func TestAliasSetCommandJoinsExpansion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if _, err := runAliasSet(t, "alias", "set", "open-prs", "pr", "list"); err != nil {
		t.Fatalf("alias set error = %v", err)
	}

	aliases, err := config.LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if aliases["open-prs"] != "pr list" {
		t.Errorf("aliases[open-prs] = %q, want %q", aliases["open-prs"], "pr list")
	}
}

func TestAliasSetCommandQuotedExpansionWithFlags(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Flags inside an expansion must be quoted (same as GitHub CLI), so the
	// whole expansion arrives as a single argument.
	if _, err := runAliasSet(t, "alias", "set", "open-prs", "pr list --state open"); err != nil {
		t.Fatalf("alias set error = %v", err)
	}

	aliases, err := config.LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if aliases["open-prs"] != "pr list --state open" {
		t.Errorf("aliases[open-prs] = %q, want %q", aliases["open-prs"], "pr list --state open")
	}
}

func TestAliasSetCommandRejectsShellAlias(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if _, err := runAliasSet(t, "alias", "set", "hi", "!echo hi"); err == nil {
		t.Fatal("alias set with shell-style expansion succeeded, want error")
	}
}

func TestAliasSetCommandRejectsBuiltinName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if _, err := runAliasSet(t, "alias", "set", "repo", "repo list"); err == nil {
		t.Fatal("alias set with a built-in command name succeeded, want error")
	}

	aliases, err := config.LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if len(aliases) != 0 {
		t.Errorf("len(aliases) = %d, want 0 (rejected alias must not be saved)", len(aliases))
	}
}

func TestAliasSetCommandRejectsUnknownExpansionTarget(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if _, err := runAliasSet(t, "alias", "set", "bad", "bogus list"); err == nil {
		t.Fatal("alias set with an unknown command word in the expansion succeeded, want error")
	}
	aliases, err := config.LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if len(aliases) != 0 {
		t.Errorf("len(aliases) = %d, want 0 (rejected alias must not be saved)", len(aliases))
	}
}

func TestAliasSetCommandRejectsRootNameExpansion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if _, err := runAliasSet(t, "alias", "set", "x", "ag pr list"); err == nil {
		t.Fatal("alias set with an expansion starting with the root command succeeded, want error")
	}
}

func TestAliasSetCommandRejectsAliasChain(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// An expansion that points at another alias can never resolve (aliases
	// expand only once), so it must be rejected at set time.
	if _, err := runAliasSet(t, "alias", "set", "x", "pl"); err == nil {
		t.Fatal("alias set with an alias-only expansion succeeded, want error")
	}
}

func TestAliasSetCommandAcceptsLeafExpansion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// A leaf command accepts arbitrary trailing arguments, including paths
	// with escaped spaces.
	if _, err := runAliasSet(t, "alias", "set", "go", "repo view C:\\temp\\x"); err != nil {
		t.Fatalf("alias set error = %v", err)
	}
}

func TestSplitExpansion(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: nil},
		{name: "single", in: "pr", want: []string{"pr"}},
		{name: "multi", in: "pr list", want: []string{"pr", "list"}},
		{name: "escaped space", in: "browse C:\\Program\\ Files\\x", want: []string{"browse", `C:\Program Files\x`}},
		{name: "escaped tab", in: "go\\\tnow", want: []string{"go\tnow"}},
		{name: "backslash letter kept verbatim", in: `C:\temp\x`, want: []string{`C:\temp\x`}},
		{name: "unquoted windows path split", in: `C:\Program Files`, want: []string{`C:\Program`, `Files`}},
		{name: "collapsed whitespace", in: "a   b", want: []string{"a", "b"}},
		{name: "leading whitespace", in: "  a b", want: []string{"a", "b"}},
		{name: "trailing whitespace", in: "a b  ", want: []string{"a", "b"}},
		{name: "tab separator", in: "a\tb", want: []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitExpansion(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitExpansion(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestAliasListCommand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.SaveAlias("rv", "repo view"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}
	if err := config.SaveAlias("pl", "pr list"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}

	buf := &bytes.Buffer{}
	cmd := NewCmdAlias(&cmdutil.Factory{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("alias list error = %v", err)
	}

	got := buf.String()
	want := "pl  pr list\nrv  repo view\n"
	if got != want {
		t.Errorf("alias list output = %q, want %q", got, want)
	}
}

func TestAliasListCommandEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	buf := &bytes.Buffer{}
	cmd := NewCmdAlias(&cmdutil.Factory{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("alias list error = %v", err)
	}

	if got := buf.String(); got != "No aliases configured.\n" {
		t.Errorf("alias list output = %q, want %q", got, "No aliases configured.\n")
	}
}

func TestAliasDeleteCommand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.SaveAlias("pl", "pr list"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}

	buf := &bytes.Buffer{}
	cmd := NewCmdAlias(&cmdutil.Factory{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"delete", "pl"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("alias delete error = %v", err)
	}

	if got := buf.String(); got != "Deleted alias pl\n" {
		t.Errorf("alias delete output = %q, want %q", got, "Deleted alias pl\n")
	}
	aliases, err := config.LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if len(aliases) != 0 {
		t.Errorf("len(aliases) = %d, want 0", len(aliases))
	}
}

func TestAliasDeleteCommandNotFound(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	buf := &bytes.Buffer{}
	cmd := NewCmdAlias(&cmdutil.Factory{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"delete", "missing"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("alias delete for missing alias succeeded, want error")
	}
}
