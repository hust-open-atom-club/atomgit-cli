package agcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPRReplyProcessDiagnostics(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "ag-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "token.json"), []byte(`{"access_token":"xyz789","user":"alice"}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"invalid", "identity", "created", "reply created identity", "accepted", "no content", "redirect", "server error", "execution", "read plain", "HTTP"} {
		t.Run(scenario, func(t *testing.T) {
			for _, raw := range []bool{false, true} {
				args := []string{"pr", "comment", "reply", "team/demo", "8", "thread", "--body", "fixture"}
				if raw {
					args = append([]string{"--raw-output"}, args...)
				}
				encoded, err := json.Marshal(args)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAPIErrorProcess$")
				cmd.Env = append(os.Environ(), "AG_API_ERROR_TEST_PROCESS=1", "AG_API_ERROR_TEST_ARGS="+string(encoded), "AG_API_ERROR_TEST_SCENARIO="+scenario)
				var out, stderr bytes.Buffer
				cmd.Stdout = &out
				cmd.Stderr = &stderr
				err = cmd.Run()
				cancel()
				exit, ok := errors.AsType[*exec.ExitError](err)
				if !ok || exit.ExitCode() != 1 || out.Len() != 0 {
					t.Fatalf("exit=%v stdout=%q stderr=%q", err, out.String(), stderr.String())
				}
				text := stderr.String()
				if scenario == "HTTP" {
					if text != "Error: API request failed.\n\nThe server returned HTTP 403.\n" {
						t.Fatalf("diagnostic=%q", text)
					}
				} else if !strings.Contains(text, "Pull request reply could not be confirmed.") || !strings.Contains(text, "The reply may have been created. Check AtomGit before retrying to avoid duplicate comments.") {
					t.Fatalf("diagnostic=%q", text)
				}
				for _, secret := range []string{"xyz789", "private-body", "private-network", "private-read"} {
					if strings.Contains(text, secret) {
						t.Fatalf("leaked private data: %q", text)
					}
				}
			}
		})
	}
}

// Run through Main with isolated credentials so a successful 201 cannot be
// hidden by a command-only mock or the top-level error renderer.
func TestPRReplyProcessSuccess(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "ag-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "token.json"), []byte(`{"access_token":"xyz789","user":"alice"}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"reply ok official", "reply ok legacy", "reply created official", "reply created legacy"} {
		t.Run(scenario, func(t *testing.T) {
			for _, raw := range []bool{false, true} {
				args := []string{"pr", "comment", "reply", "team/demo", "8", "thread", "--body", "fixture"}
				if raw {
					args = append([]string{"--raw-output"}, args...)
				}
				encoded, err := json.Marshal(args)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAPIErrorProcess$")
				cmd.Env = append(os.Environ(), "AG_API_ERROR_TEST_PROCESS=1", "AG_API_ERROR_TEST_ARGS="+string(encoded), "AG_API_ERROR_TEST_SCENARIO="+scenario)
				var out, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &out, &stderr
				err = cmd.Run()
				cancel()
				if err != nil || out.String() != "Created reply #42 in discussion thread\n" || stderr.Len() != 0 {
					t.Fatalf("raw=%t exit=%v stdout=%q stderr=%q", raw, err, out.String(), stderr.String())
				}
			}
		})
	}
}
