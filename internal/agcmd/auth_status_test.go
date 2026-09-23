package agcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

type statusTransport func(*http.Request) (*http.Response, error)

func (f statusTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAuthStatusProcess(t *testing.T) {
	if os.Getenv("AG_STATUS_TEST_PROCESS") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("AG_STATUS_TEST_ARGS")), &args); err != nil {
		t.Fatal(err)
	}
	http.DefaultTransport = statusTransport(func(r *http.Request) (*http.Response, error) {
		if os.Getenv("AG_STATUS_TEST_VERIFY") != "1" {
			panic("offline status accessed network")
		}
		if r.Method != "GET" || r.URL.String() != "https://api.atomgit.com/api/v5/user" || r.Header.Get("Authorization") != "Bearer xyz789" {
			panic("unexpected verification request")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"login":"alice"}`)), Request: r}, nil
	})
	os.Args = append([]string{"ag"}, args...)
	os.Exit(Main())
}

func TestAuthStatusDoesNotInitializeConfig(t *testing.T) {
	for _, args := range [][]string{{"auth", "status"}, {"auth", "status", "--json"}, {"auth", "status", "--verify", "--json"}, {"--raw-output", "auth", "status", "--json"}} {
		f := &cmdutil.Factory{}
		cmd, err := root.NewCmdRoot(f)
		if err != nil {
			t.Fatal(err)
		}
		err = loadCommandConfig(cmd, f, args, func() (config.Config, error) { t.Fatal("status initialized mutable config"); return nil, nil })
		if err != nil || f.Config != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
}

func TestAuthStatusStartupReadOnly(t *testing.T) {
	for _, state := range []string{"missing", "corrupt", "incomplete", "legacy", "primary-legacy", "multi-account", "directory", "alias"} {
		t.Run(state, func(t *testing.T) {
			home, xdg := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_CONFIG_HOME", xdg)
			t.Setenv("AG_OAUTH_CLIENT_SECRET", "private-oauth-secret")
			dir := filepath.Join(xdg, "ag-cli")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			primary := filepath.Join(dir, "token.json")
			path := primary
			data := []byte(`{"access_token":"xyz789","user":"alice","refresh_token":"private-refresh"}`)
			valid := true
			switch state {
			case "missing", "directory":
				valid = false
			case "corrupt":
				data = []byte(`{"xyz789":`)
				valid = false
			case "incomplete":
				data = []byte(`{"access_token":"xyz789"}`)
				valid = false
			case "legacy":
				path = filepath.Join(home, ".atomgit_personal_token.json")
			case "multi-account":
				data = []byte(`{"version":2,"active":"alice","accounts":[{"user":"bob","access_token":"private-bob"},{"user":"alice","access_token":"xyz789","refresh_token":"private-refresh"}]}`)
			}
			if state == "directory" {
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			} else if state != "missing" {
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			aliases := filepath.Join(dir, "config.json")
			aliasData := []byte(`{"private-alias-secret":broken`)
			if state == "alias" {
				aliasData = []byte(`{"aliases":{"auth-state":"auth status"}}`)
			}
			if err := os.WriteFile(aliases, aliasData, 0o600); err != nil {
				t.Fatal(err)
			}
			for _, verify := range []bool{false, true} {
				args := []string{"--raw-output", "auth", "status", "--json"}
				if state == "alias" {
					args = []string{"auth-state", "--json"}
				}
				if verify {
					args = append(args, "--verify")
				}
				encoded, _ := json.Marshal(args)
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				process := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAuthStatusProcess$")
				allowed := "0"
				if verify && valid {
					allowed = "1"
				}
				process.Env = append(os.Environ(), "AG_STATUS_TEST_PROCESS=1", "AG_STATUS_TEST_ARGS="+string(encoded), "AG_STATUS_TEST_VERIFY="+allowed)
				var out, stderr bytes.Buffer
				process.Stdout = &out
				process.Stderr = &stderr
				err := process.Run()
				cancel()
				var report struct {
					OK           bool `json:"ok"`
					Verification struct {
						Status string `json:"status"`
					} `json:"verification"`
				}
				if json.Unmarshal(out.Bytes(), &report) != nil || report.OK != valid || (err == nil) != valid {
					t.Fatalf("%v: %v stdout=%s stderr=%s", args, err, out.String(), stderr.String())
				}
				if valid && stderr.Len() != 0 {
					t.Fatal(stderr.String())
				}
				if valid && verify && report.Verification.Status != "verified" {
					t.Fatal(out.String())
				}
				if !valid && stderr.Len() == 0 {
					t.Fatal("failure has no stderr diagnostic")
				}
				for _, secret := range []string{"xyz789", "private-refresh", "private-bob", "private-oauth-secret", "private-alias-secret"} {
					if strings.Contains(out.String()+stderr.String(), secret) {
						t.Fatal("secret leaked")
					}
				}
			}
			if state != "missing" && state != "directory" {
				got, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(got, data) {
					t.Fatal("credentials changed")
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if runtime.GOOS != "windows" && info.Mode().Perm() != 0o644 {
					t.Fatal("credential permissions changed")
				}
			}
			if state == "missing" || state == "legacy" {
				if _, err := os.Stat(primary); !os.IsNotExist(err) {
					t.Fatal("created or migrated credentials")
				}
			}
			got, err := os.ReadFile(aliases)
			if err != nil || !bytes.Equal(got, aliasData) {
				t.Fatal("aliases changed")
			}
		})
	}
}
