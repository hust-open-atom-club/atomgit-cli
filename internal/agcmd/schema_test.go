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
		err = loadCommandConfig(cmd, f, args, func() (config.Config, error) { t.Fatal("credential loader called"); return nil, nil })
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
	if err := loadCommandConfig(cmd, f, []string{"pr", "view", "1"}, func() (config.Config, error) { return nil, want }); err != want {
		t.Fatal("ordinary startup no longer loads config")
	}
}

type forbiddenSchemaTransport struct{}

func (forbiddenSchemaTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic("schema performed HTTP request")
}

func TestSchemaProcess(t *testing.T) {
	if os.Getenv("AG_SCHEMA_TEST_PROCESS") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("AG_SCHEMA_TEST_ARGS")), &args); err != nil {
		t.Fatal(err)
	}
	http.DefaultTransport = forbiddenSchemaTransport{}
	os.Args = append([]string{"ag"}, args...)
	os.Exit(Main())
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
			if state == "corrupt" || state == "legacy" || state == "local-alias" {
				if err := os.WriteFile(token, content, 0o644); err != nil {
					t.Fatal(err)
				}
			} else if state == "directory" {
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
				data, err := json.Marshal(args)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				process := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSchemaProcess$")
				process.Env = append(os.Environ(), "AG_SCHEMA_TEST_PROCESS=1", "AG_SCHEMA_TEST_ARGS="+string(data))
				var out, stderr bytes.Buffer
				process.Stdout = &out
				process.Stderr = &stderr
				err = process.Run()
				cancel()
				unknown := args[len(args)-1] == "missing" || args[len(args)-1] == "private-alias-secret"
				if unknown {
					if err == nil || out.Len() != 0 || !strings.Contains(stderr.String(), "unknown or hidden command path") {
						t.Fatalf("unknown path: %v %s %s", err, out.String(), stderr.String())
					}
				} else if err != nil || !json.Valid(out.Bytes()) || stderr.Len() != 0 {
					t.Fatalf("%v: %v stdout=%s stderr=%s", args, err, out.String(), stderr.String())
				}
				if strings.Contains(out.String()+stderr.String(), "secret") {
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
