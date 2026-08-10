package root

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	internalversion "atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	versioncmd "atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/version"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type rootTestConfig struct{}

func (rootTestConfig) GetToken() (string, error) { return "secret", nil }
func (rootTestConfig) GetUser() (string, error)  { return "tester", nil }
func (rootTestConfig) GetHost() string           { return "atomgit.com" }

type rootRoundTripFunc func(*http.Request) (*http.Response, error)

func (f rootRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func setVersionMetadata(t *testing.T, source string) {
	t.Helper()
	oldV, oldC, oldB := internalversion.Version, internalversion.Commit, internalversion.BuildDate
	oldSource := internalversion.Source
	internalversion.Version = "v1.2.3"
	internalversion.Commit = "abc1234"
	internalversion.BuildDate = "2026-07-15T00:00:00Z"
	internalversion.Source = source
	t.Cleanup(func() {
		internalversion.Version = oldV
		internalversion.Commit = oldC
		internalversion.BuildDate = oldB
		internalversion.Source = oldSource
	})
}

func TestNewCmdRootRegistersCommands(t *testing.T) {
	cmd, err := NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"api": false, "auth": false, "branch": false, "issue": false, "label": false, "license": false, "milestone": false,
		"check-update": false,
		"org":          false, "pr": false, "release": false, "repo": false, "run": false, "ssh-key": false, "tag": false, "version": false,
	}
	for _, child := range cmd.Commands() {
		if _, ok := want[child.Name()]; ok {
			want[child.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("command %q was not registered", name)
		}
	}
	if cmd.Use != "ag <command> <subcommand> [flags]" {
		t.Fatalf("Use = %q", cmd.Use)
	}
	if cmd.PersistentFlags().Lookup("help") == nil {
		t.Fatal("persistent help flag was not registered")
	}
	versionFlag := cmd.Flags().Lookup("version")
	if versionFlag == nil {
		t.Fatal("version flag was not registered")
	}
	if versionFlag.Shorthand != "" {
		t.Fatalf("version shorthand = %q, want none", versionFlag.Shorthand)
	}
}

func TestRootReturnsErrorsWithoutPrintingThem(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd, err := newCmdRootWithWriters(&cmdutil.Factory{}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	cmd.AddCommand(&cobra.Command{
		Use: "fail",
		RunE: func(*cobra.Command, []string) error {
			return errors.New("boom")
		},
	})
	cmd.SetArgs([]string{"fail"})

	err = cmd.Execute()
	if err == nil || err.Error() != "boom" {
		t.Fatalf("Execute() error = %v, want boom", err)
	}
	if err := cmdutil.FlushWriter(cmd.ErrOrStderr()); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want no Cobra error output", stderr.String())
	}
}

func TestAPIHelpDocumentsSafetyContract(t *testing.T) {
	cmd, err := NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"api", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := out.String()
	for _, want := range []string{"api <endpoint>", "GET is the default", "POST", "PATCH", "PUT", "DELETE", "relative AtomGit API v5", "does not infer or", "--paginate", "--raw-output"} {
		if !strings.Contains(help, want) {
			t.Errorf("help does not contain %q:\n%s", want, help)
		}
	}
}

func TestNewCmdRootVersionMatchesVersionCommandForEveryProfile(t *testing.T) {
	tests := []struct{ name, source string }{
		{name: "source", source: "source"},
		{name: "release", source: "release"},
		{name: "development", source: "development"},
		{name: "npm", source: "npm"},
		{name: "homebrew", source: "homebrew"},
		{name: "winget", source: "winget"},
		{name: "scoop", source: "scoop"},
		{name: "nix", source: "nix"},
		{name: "extension source", source: "corp-repo"},
		{name: "invalid source", source: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setVersionMetadata(t, tt.source)

			rootCmd, err := NewCmdRoot(&cmdutil.Factory{})
			if err != nil {
				t.Fatal(err)
			}
			var rootOut bytes.Buffer
			rootCmd.SetOut(&rootOut)
			rootCmd.SetArgs([]string{"--version"})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("root Execute() error = %v", err)
			}

			versionCmd := versioncmd.NewCmdVersion()
			var versionOut bytes.Buffer
			versionCmd.SetOut(&versionOut)
			if err := versionCmd.Execute(); err != nil {
				t.Fatalf("version Execute() error = %v", err)
			}

			if got, want := rootOut.String(), versionOut.String(); got != want {
				t.Errorf("--version output = %q, version output = %q", got, want)
			}
		})
	}
}

func TestNewCmdRootVersionHelp(t *testing.T) {
	cmd, err := NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "--version") {
		t.Errorf("help output does not mention --version: %s", out.String())
	}
	if !strings.Contains(out.String(), "--raw-output") {
		t.Errorf("help output does not mention --raw-output: %s", out.String())
	}
}

func TestRootSanitizesPipedOutputByDefault(t *testing.T) {
	payload := "unsafe \x1b]52;c;attack\x07\n"
	var stdout, stderr bytes.Buffer
	cmd, err := newCmdRootWithWriters(&cmdutil.Factory{}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	cmd.AddCommand(&cobra.Command{
		Use: "emit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), payload)
			return err
		},
	})
	cmd.SetArgs([]string{"emit"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := cmdutil.FlushWriter(cmd.OutOrStdout()); err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), "unsafe \\x1b]52;c;attack\\x07\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRootRawOutputIsExplicitOptOut(t *testing.T) {
	payload := "raw \x1b[31mtext\x1b[0m\n"
	var stdout, stderr bytes.Buffer
	cmd, err := newCmdRootWithWriters(&cmdutil.Factory{}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	cmd.AddCommand(&cobra.Command{
		Use: "emit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), payload)
			return err
		},
	})
	cmd.SetArgs([]string{"--raw-output", "emit"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != payload {
		t.Fatalf("stdout = %q, want raw %q", got, payload)
	}
}

func TestAPIOutputHonorsRootSanitization(t *testing.T) {
	payload := "api \x1b[31moutput\x1b[0m\n"
	factory := &cmdutil.Factory{
		Config: rootTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: rootRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload)), Request: req}, nil
			})}, nil
		},
	}
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{name: "safe", args: []string{"api", "/user"}, want: "api \\x1b[31moutput\\x1b[0m\n"},
		{name: "raw", args: []string{"--raw-output", "api", "/user"}, want: payload},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd, err := newCmdRootWithWriters(factory, &stdout, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			cmd.SetArgs(tt.args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if err := cmdutil.FlushWriter(cmd.OutOrStdout()); err != nil {
				t.Fatal(err)
			}
			if stdout.String() != tt.want {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.want)
			}
		})
	}
}

func newTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	cmd, err := newCmdRootWithWriters(&cmdutil.Factory{}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("newCmdRootWithWriters() error = %v", err)
	}
	return cmd
}

func TestExpandAlias(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "no arguments", args: nil, want: nil},
		{name: "flag first", args: []string{"--version"}, want: []string{"--version"}},
		{name: "unknown command", args: []string{"unknown"}, want: []string{"unknown"}},
		{name: "no alias configured", args: []string{"nope"}, want: []string{"nope"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestRoot(t)
			got, err := ExpandAlias(cmd, tt.args)
			if err != nil {
				t.Fatalf("ExpandAlias() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExpandAlias(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestExpandAliasExpandsConfiguredAlias(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.SaveAlias("pl", "pr list"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}

	cmd := newTestRoot(t)
	got, err := ExpandAlias(cmd, []string{"pl", "--state", "open"})
	if err != nil {
		t.Fatalf("ExpandAlias() error = %v", err)
	}
	want := []string{"pr", "list", "--state", "open"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExpandAlias() = %v, want %v", got, want)
	}
}

func TestExpandAliasBuiltinTakesPrecedence(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// A user could try to shadow a built-in command; the built-in must win.
	if err := config.SaveAlias("repo", "pr list"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}

	cmd := newTestRoot(t)
	got, err := ExpandAlias(cmd, []string{"repo", "view"})
	if err != nil {
		t.Fatalf("ExpandAlias() error = %v", err)
	}
	want := []string{"repo", "view"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExpandAlias() = %v, want %v (built-in command must win)", got, want)
	}
}

func TestExpandAliasRejectsShellAlias(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.SaveAlias("hi", "!echo hi"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}

	cmd := newTestRoot(t)
	if _, err := ExpandAlias(cmd, []string{"hi"}); err == nil {
		t.Fatal("ExpandAlias() with shell-style alias succeeded, want error")
	}
}

func TestExpandAliasAfterRootFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.SaveAlias("pl", "pr list"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}

	cmd := newTestRoot(t)
	got, err := ExpandAlias(cmd, []string{"--raw-output", "pl"})
	if err != nil {
		t.Fatalf("ExpandAlias() error = %v", err)
	}
	want := []string{"--raw-output", "pr", "list"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExpandAlias() = %v, want %v", got, want)
	}
}

func TestExpandAliasCorruptConfigFallsBack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := config.AliasFilePath()
	if err != nil {
		t.Fatalf("AliasFilePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := newTestRoot(t)
	got, err := ExpandAlias(cmd, []string{"pl"})
	if err != nil {
		t.Fatalf("ExpandAlias() error = %v, want nil (fallback to no aliases)", err)
	}
	want := []string{"pl"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExpandAlias() = %v, want %v", got, want)
	}
}

func TestExpandAliasCorruptConfigWarnsOnStderr(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := config.AliasFilePath()
	if err != nil {
		t.Fatalf("AliasFilePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stderr bytes.Buffer
	cmd, err := newCmdRootWithWriters(&cmdutil.Factory{}, io.Discard, &stderr)
	if err != nil {
		t.Fatalf("newCmdRootWithWriters() error = %v", err)
	}
	if _, err := ExpandAlias(cmd, []string{"pl"}); err != nil {
		t.Fatalf("ExpandAlias() error = %v, want nil (fallback to no aliases)", err)
	}
	if !strings.Contains(stderr.String(), "warning: failed to load aliases") {
		t.Errorf("stderr = %q, want warning about failed alias load", stderr.String())
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
			if got := splitExpansion(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitExpansion(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandAliasWithEscapedSpaceInExpansion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// Windows path with a space survives as a single token via `\ `.
	if err := config.SaveAlias("go", "browse C:\\Program\\ Files\\x"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}

	cmd := newTestRoot(t)
	got, err := ExpandAlias(cmd, []string{"go"})
	if err != nil {
		t.Fatalf("ExpandAlias() error = %v", err)
	}
	want := []string{"browse", `C:\Program Files\x`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExpandAlias() = %v, want %v", got, want)
	}
}
