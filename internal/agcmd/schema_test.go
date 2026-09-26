package agcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/root"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestSchemaDoesNotInitializeConfig(t *testing.T) {
	for _, args := range [][]string{{"schema"}, {"schema", "ag"}, {"schema", "ag", "pr", "create"}, {"schema", "pr", "create"}, {"schema", "unknown"}, {"--raw-output", "schema", "api"}, {"doctor"}} {
		f := &cmdutil.Factory{}
		cmd, err := root.NewCmdRoot(f)
		if err != nil {
			t.Fatal(err)
		}
		selected, _, err := cmd.Find(args)
		if err != nil {
			t.Fatalf("Find(%v): %v", args, err)
		}
		err = loadCommandConfig(selected, f, func() (config.Config, error) { t.Fatal("credential loader called"); return nil, nil })
		if err != nil || f.Config != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	f := &cmdutil.Factory{}
	cmd, err := root.NewCmdRoot(f)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Errorf("loader called")
	selected, _, err := cmd.Find([]string{"pr", "view", "1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := loadCommandConfig(selected, f, func() (config.Config, error) { return nil, want }); err != want {
		t.Fatal("ordinary startup no longer loads config")
	}
}

func TestLoadCommandConfigSkipsHelpAndVersionCommands(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"help", "pr"}, {"version"}, {"version", "--json"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			factory := &cmdutil.Factory{}
			cmd, err := root.NewCmdRoot(factory)
			if err != nil {
				t.Fatal(err)
			}
			cmd.InitDefaultHelpCmd()
			selected, _, err := cmd.Find(args)
			if err != nil {
				t.Fatalf("Find(%v): %v", args, err)
			}
			if err := loadCommandConfig(selected, factory, func() (config.Config, error) {
				t.Fatal("credential loader called")
				return nil, nil
			}); err != nil {
				t.Fatal(err)
			}
			if factory.Config != nil {
				t.Fatal("credential-free command populated the config")
			}
		})
	}
}

type forbiddenCommandTransport struct{}

func (forbiddenCommandTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic("credential-independent command performed HTTP request")
}

func TestAgCommandProcess(t *testing.T) {
	if os.Getenv("AG_COMMAND_TEST_PROCESS") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("AG_COMMAND_TEST_ARGS")), &args); err != nil {
		t.Fatal(err)
	}
	http.DefaultTransport = forbiddenCommandTransport{}
	os.Args = append([]string{"ag"}, args...)
	os.Exit(Main())
}

func runAgCommandProcess(t *testing.T, args []string) (string, string, error) {
	t.Helper()
	data, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	process := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAgCommandProcess$")
	process.Env = append(os.Environ(), "AG_COMMAND_TEST_PROCESS=1", "AG_COMMAND_TEST_ARGS="+string(data))
	var stdout, stderr bytes.Buffer
	process.Stdout = &stdout
	process.Stderr = &stderr
	err = process.Run()
	if ctx.Err() != nil {
		t.Fatalf("command process timed out: %v", args)
	}
	return stdout.String(), stderr.String(), err
}

func TestSchemaStartupWithIsolatedCredentials(t *testing.T) {
	for _, state := range []string{"missing", "corrupt", "legacy", "directory", "local-alias"} {
		t.Run(state, func(t *testing.T) {
			home, xdg := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_CONFIG_HOME", xdg)
			t.Setenv("ATOMGIT_TOKEN", "environment-secret-must-not-appear")
			dir := filepath.Join(xdg, "ag-cli")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			token := filepath.Join(dir, "token.json")
			content := []byte(`{"access_token":"file-secret-must-not-appear",broken`)
			if state == "legacy" {
				token = filepath.Join(home, ".atomgit_personal_token.json")
				content = []byte(`{"access_token":"file-secret-must-not-appear","user":"user-secret-must-not-appear"}`)
			}
			switch state {
			case "corrupt", "legacy", "local-alias":
				if err := os.WriteFile(token, content, 0o644); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(token, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			// A broken aliases file must not even produce a warning for direct
			// schema calls, including an invalid target path.
			aliases := filepath.Join(dir, "config.json")
			aliasContent := []byte(`{"private-alias-secret": broken`)
			calls := [][]string{{"schema"}, {"schema", "ag"}, {"schema", "ag", "pr", "create"}, {"schema", "pr", "create"}, {"--raw-output", "schema", "api"}, {"schema", "pr", "comment", "create"}, {"schema", "missing"}, {"schema", "ag", "missing"}}
			if state == "local-alias" {
				aliasContent = []byte(`{"aliases":{"inspect-schema":"schema","private-alias-secret":"api /user"}}`)
				calls = append(calls, []string{"schema", "private-alias-secret"}, []string{"inspect-schema", "api"})
			}
			if err := os.WriteFile(aliases, aliasContent, 0o600); err != nil {
				t.Fatal(err)
			}
			for _, args := range calls {
				stdout, stderr, err := runAgCommandProcess(t, args)
				unknown := args[len(args)-1] == "missing" || args[len(args)-1] == "private-alias-secret"
				if unknown {
					if err == nil || stdout != "" || !strings.Contains(stderr, "unknown or hidden command path") {
						t.Fatalf("unknown path: %v %s %s", err, stdout, stderr)
					}
				} else if err != nil || !json.Valid([]byte(stdout)) || stderr != "" {
					t.Fatalf("%v: %v stdout=%s stderr=%s", args, err, stdout, stderr)
				}
				if strings.Contains(stdout+stderr, "secret") {
					t.Fatal("private data leaked")
				}
			}
			if state == "corrupt" || state == "legacy" || state == "local-alias" {
				got, err := os.ReadFile(token)
				if err != nil || !bytes.Equal(got, content) {
					t.Fatal("credential file changed")
				}
				info, err := os.Stat(token)
				if err != nil {
					t.Fatal(err)
				}
				if runtime.GOOS != "windows" && info.Mode().Perm() != 0o644 {
					t.Fatal("credential permissions repaired")
				}
			}
			if state == "missing" || state == "legacy" {
				if _, err := os.Stat(filepath.Join(dir, "token.json")); !os.IsNotExist(err) {
					t.Fatal("created or migrated credentials")
				}
			}
			got, err := os.ReadFile(aliases)
			if err != nil || !bytes.Equal(got, aliasContent) {
				t.Fatal("alias config changed")
			}
		})
	}
}

func TestHelpAndVersionCommandsIgnoreCredentialFailures(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		contains string
		help     bool
		json     bool
	}{
		{name: "no arguments", contains: "ag [command]", help: true},
		{name: "root help flag", args: []string{"--help"}, contains: "ag [command]", help: true},
		{name: "root help shorthand", args: []string{"-h"}, contains: "ag [command]", help: true},
		{name: "help command", args: []string{"help"}, contains: "ag [command]", help: true},
		{name: "help command target", args: []string{"help", "pr"}, contains: "ag pr [command]", help: true},
		{name: "subcommand help flag", args: []string{"pr", "--help"}, contains: "ag pr [command]", help: true},
		{name: "subcommand help shorthand", args: []string{"pr", "-h"}, contains: "ag pr [command]", help: true},
		{name: "version command", args: []string{"version"}, contains: "ag version"},
		{name: "version JSON", args: []string{"version", "--json"}, json: true},
		{name: "root version flag", args: []string{"--version"}, contains: "ag version"},
		{name: "help alias", args: []string{"prhelp"}, contains: "ag pr [command]", help: true},
		{name: "version alias", args: []string{"v"}, contains: "ag version"},
	}

	credentialStates := []string{"missing", "corrupt", "read-failure", "legacy"}
	for _, state := range credentialStates {
		t.Run(state, func(t *testing.T) {
			home, xdg := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_CONFIG_HOME", xdg)
			t.Setenv("ATOMGIT_TOKEN", "environment-secret-must-not-appear")

			configDir := filepath.Join(xdg, "ag-cli")
			if err := os.MkdirAll(configDir, 0o700); err != nil {
				t.Fatal(err)
			}
			primaryToken := filepath.Join(configDir, "token.json")
			legacyToken := filepath.Join(home, ".atomgit_personal_token.json")
			var credentialPath string
			var credentialContent []byte
			switch state {
			case "corrupt":
				credentialPath = primaryToken
				credentialContent = []byte(`{"access_token":"credential-secret",broken`)
				if err := os.WriteFile(credentialPath, credentialContent, 0o644); err != nil {
					t.Fatal(err)
				}
			case "read-failure":
				credentialPath = primaryToken
				if err := os.Mkdir(credentialPath, 0o700); err != nil {
					t.Fatal(err)
				}
			case "legacy":
				credentialPath = legacyToken
				credentialContent = []byte(`{"access_token":"legacy-secret","user":"alice"}`)
				if err := os.WriteFile(credentialPath, credentialContent, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			aliasPath, err := config.AliasFilePath()
			if err != nil {
				t.Fatal(err)
			}
			aliasContent := []byte(`{"aliases":{"prhelp":"pr --help","v":"version"}}`)
			if err := os.WriteFile(aliasPath, aliasContent, 0o600); err != nil {
				t.Fatal(err)
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					stdout, stderr, err := runAgCommandProcess(t, tt.args)
					if err != nil || stderr != "" {
						t.Fatalf("command failed: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
					}
					if tt.json {
						var info map[string]json.RawMessage
						if err := json.Unmarshal([]byte(stdout), &info); err != nil {
							t.Fatalf("version output is not JSON: %v: %s", err, stdout)
						}
						if len(info) != 3 || info["version"] == nil || info["commit"] == nil || info["buildDate"] == nil {
							t.Fatalf("version JSON fields = %v, want version, commit and buildDate", info)
						}
					} else {
						if tt.help && !strings.Contains(stdout, "Usage:") {
							t.Fatalf("help output has no Usage section: %s", stdout)
						}
						if !strings.Contains(stdout, tt.contains) {
							t.Fatalf("stdout does not contain %q: %s", tt.contains, stdout)
						}
					}
					if strings.Contains(stdout+stderr, "secret") {
						t.Fatal("credential data leaked into command output")
					}
				})
			}

			if state == "corrupt" || state == "legacy" {
				got, err := os.ReadFile(credentialPath)
				if err != nil || !bytes.Equal(got, credentialContent) {
					t.Fatal("credential file changed")
				}
				info, err := os.Stat(credentialPath)
				if err != nil {
					t.Fatal(err)
				}
				if runtime.GOOS != "windows" && info.Mode().Perm() != 0o644 {
					t.Fatalf("credential permissions = %o, want 644", info.Mode().Perm())
				}
			}
			if state == "missing" || state == "legacy" {
				if _, err := os.Stat(primaryToken); !os.IsNotExist(err) {
					t.Fatalf("primary token path unexpectedly exists: %v", err)
				}
			}
			if state == "read-failure" {
				entries, err := os.ReadDir(primaryToken)
				if err != nil || len(entries) != 0 {
					t.Fatalf("read-failure fixture changed: entries=%v error=%v", entries, err)
				}
			}
			gotAliases, err := os.ReadFile(aliasPath)
			if err != nil || !bytes.Equal(gotAliases, aliasContent) {
				t.Fatal("alias config changed")
			}
		})
	}
}

func TestAuthenticatedCommandStillReportsCorruptCredentials(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	configDir := filepath.Join(xdg, "ag-cli")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "token.json"), []byte(`{broken`), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runAgCommandProcess(t, []string{"pr", "view", "1"})
	if err == nil || stdout != "" || !strings.Contains(stderr, "failed to load config:") {
		t.Fatalf("pr view: error=%v stdout=%s stderr=%s", err, stdout, stderr)
	}
}
