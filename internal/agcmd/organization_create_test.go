package agcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestOrgCreateProcessPresentation(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "ag-cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "token.json"), []byte(`{"access_token":"xyz789","user":"alice"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, scenario, detail string
		definite, success      bool
	}{
		{name: "5xx", scenario: "server error", detail: "The server returned HTTP 502."},
		{name: "5xx with interrupted body", scenario: "server error read", detail: "The server returned HTTP 502.\nThe error response body could not be read."},
		{name: "execution failure", scenario: "execution", detail: "The HTTP request could not be completed."},
		{name: "read timeout", scenario: "read", detail: "Reading the server response timed out."},
		{name: "read failure", scenario: "read plain", detail: "The server response could not be read."},
		{name: "malformed response", scenario: "invalid", detail: "The server returned an invalid response."},
		{name: "mismatched identity", scenario: "identity", detail: "The server returned an invalid response."},
		{name: "definite rejection", scenario: "HTTP", detail: "The server returned HTTP 403.", definite: true},
		{name: "success", scenario: "org success", success: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, raw := range []bool{false, true} {
				args := []string{"org", "create", "team"}
				if raw {
					args = append([]string{"--raw-output"}, args...)
				}
				encoded, err := json.Marshal(args)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAPIErrorProcess$")
				cmd.Env = append(os.Environ(), "AG_API_ERROR_TEST_PROCESS=1",
					"AG_API_ERROR_TEST_ARGS="+string(encoded), "AG_API_ERROR_TEST_SCENARIO="+tt.scenario)
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				err = cmd.Run()
				cancel()
				if tt.success {
					if err != nil || stdout.String() != "Created organization team: https://example.test/team\n" || stderr.Len() != 0 {
						t.Fatalf("raw=%t: err=%v stdout=%q stderr=%q", raw, err, stdout.String(), stderr.String())
					}
					continue
				}
				exit, ok := errors.AsType[*exec.ExitError](err)
				if !ok || exit.ExitCode() != 1 {
					t.Fatalf("raw=%t: exit=%v stderr=%q", raw, err, stderr.String())
				}
				summary := "Organization creation could not be confirmed."
				hint := "\nOrganization creation may have succeeded. Check AtomGit before retrying.\n"
				if tt.definite {
					summary, hint = "API request failed.", ""
				}
				want := "Error: " + summary + "\n\n" + tt.detail + "\n" + hint
				// Exact stderr also excludes response bodies, credentials and
				// terminal controls, including with --raw-output enabled.
				if stdout.Len() != 0 || stderr.String() != want {
					t.Fatalf("raw=%t: stdout=%q stderr=%q, want stderr=%q", raw, stdout.String(), stderr.String(), want)
				}
			}
		})
	}
}
