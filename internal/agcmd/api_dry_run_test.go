package agcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/root"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestAPIDryRunSkipsConfigOnlyWhenEnabled(t *testing.T) {
	for _, flag := range []string{"--dry-run", "--dry-run=true", "--dry-run=false", ""} {
		t.Run(flag, func(t *testing.T) {
			factory := &cmdutil.Factory{}
			cmd, err := root.NewCmdRoot(factory)
			if err != nil {
				t.Fatal(err)
			}
			selected, _, err := cmd.Find([]string{"api"})
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"/user"}
			if flag != "" {
				args = append(args, flag)
			}
			if err := selected.ParseFlags(args); err != nil {
				t.Fatal(err)
			}
			want := errors.New("loader called")
			calls := 0
			err = loadCommandConfig(selected, factory, func() (config.Config, error) {
				calls++
				return nil, want
			})
			if flag == "--dry-run" || flag == "--dry-run=true" {
				if err != nil || calls != 0 || factory.Config != nil {
					t.Fatalf("dry run loaded config: %v, calls=%d", err, calls)
				}
			} else if err != want || calls != 1 {
				t.Fatalf("real mode bypassed config: %v, calls=%d", err, calls)
			}
		})
	}
}

func TestAPIDryRunStartupDoesNotReadOrMutateCredentials(t *testing.T) {
	for _, state := range []string{"missing", "corrupt", "legacy", "primary-legacy", "directory", "unreadable", "alias"} {
		t.Run(state, func(t *testing.T) {
			home, xdg := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_CONFIG_HOME", xdg)
			t.Setenv("ATOMGIT_TOKEN", "private-environment-token")
			t.Setenv("AG_OAUTH_CLIENT_SECRET", "private-oauth-secret")
			dir := filepath.Join(xdg, "ag-cli")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			primary := filepath.Join(dir, "token.json")
			tokenPath := primary
			content := []byte(`{"access_token":"x!","refresh_token":"private-refresh","user":"private-user"}`)
			if state == "legacy" {
				tokenPath = filepath.Join(home, ".atomgit_personal_token.json")
			} else if state == "corrupt" {
				content = []byte(`{"private-token":broken`)
			}
			filePresent := state != "missing" && state != "directory"
			mode := os.FileMode(0o644)
			if state == "unreadable" {
				mode = 0
			}
			if filePresent {
				if err := os.WriteFile(tokenPath, content, mode); err != nil {
					t.Fatal(err)
				}
			} else if state == "directory" {
				if err := os.Mkdir(tokenPath, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			var before os.FileInfo
			if state != "missing" {
				var err error
				before, err = os.Stat(tokenPath)
				if err != nil {
					t.Fatal(err)
				}
			}
			aliases := filepath.Join(dir, "config.json")
			aliasData := []byte(`{"private-alias":broken`)
			if state == "alias" {
				aliasData = []byte(`{"aliases":{"preview-api":"api --dry-run"}}`)
			}
			if err := os.WriteFile(aliases, aliasData, 0o600); err != nil {
				t.Fatal(err)
			}
			input := filepath.Join(home, "private-input.json")
			inputData := []byte(`{"token":"private-body-token","nested":{"password":"x!"}}`)
			if err := os.WriteFile(input, inputData, 0o600); err != nil {
				t.Fatal(err)
			}
			calls := [][]string{
				{"api", "/repos/private-owner/private-repo/issues?code=x!", "--dry-run"},
				{"--raw-output", "api", "--dry-run", "/user", "--method", "PATCH", "--input", input},
				{"api", "/user?token=private-token%zz", "--dry-run"},
				{"api", "/user?page=123456789&per_page=654321", "--paginate", "--dry-run", "--raw-output"},
			}
			if state == "alias" {
				calls = append(calls, []string{"preview-api", "/user"})
			}
			for i, args := range calls {
				stdout, stderr, err := runAgCommandProcess(t, args)
				if i == 2 {
					if err == nil || stdout != "" || !strings.Contains(stderr, "invalid endpoint") {
						t.Fatalf("bad endpoint result: %v %q %q", err, stdout, stderr)
					}
				} else {
					var preview struct {
						DryRun   bool `json:"dryRun"`
						Executed bool `json:"executed"`
					}
					if err != nil || stderr != "" || json.Unmarshal([]byte(stdout), &preview) != nil || !preview.DryRun || preview.Executed {
						t.Fatalf("dry run: %v %q %q", err, stdout, stderr)
					}
				}
				for _, secret := range []string{"private-", "x!", "123456789", "654321"} {
					if strings.Contains(stdout+stderr, secret) {
						t.Fatalf("secret leaked: %q %q", stdout, stderr)
					}
				}
			}
			if before != nil {
				after, err := os.Stat(tokenPath)
				if err != nil || !before.ModTime().Equal(after.ModTime()) || (runtime.GOOS != "windows" && before.Mode() != after.Mode()) {
					t.Fatal("credential metadata changed")
				}
			}
			if filePresent {
				// Only the test restores permission after checking that the CLI
				// left the unreadable fixture untouched.
				if state == "unreadable" {
					if err := os.Chmod(tokenPath, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				if got, err := os.ReadFile(tokenPath); err != nil || !bytes.Equal(got, content) {
					t.Fatal("credentials changed")
				}
			}
			if state == "missing" || state == "legacy" {
				if _, err := os.Stat(primary); !os.IsNotExist(err) {
					t.Fatal("created or migrated credentials")
				}
			}
			for path, expected := range map[string][]byte{aliases: aliasData, input: inputData} {
				if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, expected) {
					t.Fatalf("file changed: %s", path)
				}
			}
		})
	}
}

func TestAPIDryRunStartupFlagErrors(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	for _, badFlag := range []string{"--paginate=private-secret", "--dry-run=private-secret", "--private-secret=value", "-private-secret", "--raw-output=private-secret", "--help=private-secret"} {
		for _, before := range []bool{true, false} {
			t.Run(badFlag+"/"+map[bool]string{true: "before", false: "after"}[before], func(t *testing.T) {
				args := []string{"--raw-output", "api", "/user"}
				if before {
					args = append(args, "--dry-run", badFlag)
				} else {
					args = append(args, badFlag, "--dry-run")
				}
				stdout, stderr, err := runAgCommandProcess(t, args)
				if err == nil || stdout != "" || !strings.Contains(stderr, "invalid API flags") || strings.Contains(stderr, "private-") {
					t.Fatalf("unsafe flag error: %v %q %q", err, stdout, stderr)
				}
			})
		}
	}
	for _, dir := range []string{home, xdg} {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("flag errors modified configuration: %v %v", entries, err)
		}
	}
}
