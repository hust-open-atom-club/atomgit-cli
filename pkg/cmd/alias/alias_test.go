package alias

import (
	"bytes"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

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

	buf := &bytes.Buffer{}
	cmd := NewCmdAlias(&cmdutil.Factory{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "pl", "pr list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("alias set error = %v", err)
	}

	if got := buf.String(); !strings.Contains(got, "Added alias pl: pr list") {
		t.Errorf("output = %q, want to contain %q", got, "Added alias pl: pr list")
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

	buf := &bytes.Buffer{}
	cmd := NewCmdAlias(&cmdutil.Factory{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "open-prs", "pr", "list"})
	if err := cmd.Execute(); err != nil {
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

	buf := &bytes.Buffer{}
	cmd := NewCmdAlias(&cmdutil.Factory{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	// Flags inside an expansion must be quoted (same as GitHub CLI), so the
	// whole expansion arrives as a single argument.
	cmd.SetArgs([]string{"set", "open-prs", "pr list --state open"})
	if err := cmd.Execute(); err != nil {
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

	buf := &bytes.Buffer{}
	cmd := NewCmdAlias(&cmdutil.Factory{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "hi", "!echo hi"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("alias set with shell-style expansion succeeded, want error")
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
