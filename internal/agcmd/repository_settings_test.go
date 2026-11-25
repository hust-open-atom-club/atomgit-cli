package agcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestRepositorySettingsProcessPresentation(t *testing.T) {
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
	for _, tc := range []struct {
		name, scenario, detail string
		definite               bool
	}{
		{name: "unexpected success", scenario: "created", detail: "The server returned HTTP 201."},
		{name: "redirect", scenario: "redirect", detail: "The server returned HTTP 303."},
		{name: "server error", scenario: "server error", detail: "The server returned HTTP 502."},
		{name: "invalid response", scenario: "invalid", detail: "The server returned an invalid response."},
		{name: "execution failure", scenario: "execution", detail: "The HTTP request could not be completed."},
		{name: "read failure", scenario: "read plain", detail: "The server response could not be read."},
		{name: "permission denied", scenario: "HTTP", detail: "The server returned HTTP 403.", definite: true},
		{name: "validation rejected", scenario: "unprocessable", detail: "The server returned HTTP 422.", definite: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, raw := range []bool{false, true} {
				args := []string{"repo", "settings", "edit", "team/demo", "--action-pipeline-enabled=false", "--yes"}
				if raw {
					args = append([]string{"--raw-output"}, args...)
				}
				encoded, err := json.Marshal(args)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAPIErrorProcess$")
				cmd.Env = append(os.Environ(), "AG_API_ERROR_TEST_PROCESS=1", "AG_API_ERROR_TEST_ARGS="+string(encoded), "AG_API_ERROR_TEST_SCENARIO="+tc.scenario)
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				err = cmd.Run()
				cancel()
				exit, ok := errors.AsType[*exec.ExitError](err)
				if !ok || exit.ExitCode() != 1 {
					t.Fatalf("raw=%t exit=%v stderr=%q", raw, err, stderr.String())
				}
				summary := "Repository settings update could not be confirmed."
				hint := "\nRepository settings may have changed. Check AtomGit before retrying.\n"
				if tc.definite {
					summary, hint = "API request failed.", ""
				}
				want := "Error: " + summary + "\n\n" + tc.detail + "\n" + hint
				if stdout.Len() != 0 || stderr.String() != want {
					t.Fatalf("raw=%t stdout=%q stderr=%q want=%q", raw, stdout.String(), stderr.String(), want)
				}
			}
		})
	}
}

func TestRepositorySettingsFailureDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       int
		body, detail string
		cause        error
		uncertain    bool
	}{
		{name: "permission", status: 403, body: `{"secret":"private-token"}`, detail: "The server returned HTTP 403."},
		{name: "unexpected success", status: 201, detail: "The server returned HTTP 201.", uncertain: true},
		{name: "server error", status: 500, body: `{"secret":"private-token"}`, detail: "The server returned HTTP 500.", uncertain: true},
		{name: "malformed", status: 200, body: `{`, detail: "The server returned an invalid response.", uncertain: true},
		{name: "transport", cause: errors.New("private-token"), detail: "The HTTP request could not be completed.", uncertain: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := api.NewClientWithHTTPClient("private-token", &http.Client{Transport: statusTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.cause != nil {
					return nil, tc.cause
				}
				return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
			})})
			value := false
			_, err := api.UpdateRepositorySettings(client, "team", "demo", api.UpdateRepositorySettingsRequest{ActionPipelineEnabled: &value})
			if err == nil || calls != 1 {
				t.Fatalf("error=%v calls=%d", err, calls)
			}
			for _, raw := range []bool{false, true} {
				var out bytes.Buffer
				writer := cmdutil.NewSanitizingWriter(&out)
				if err := writer.SetRaw(raw); err != nil {
					t.Fatal(err)
				}
				renderCommandError(writer, err)
				if err := cmdutil.FlushWriter(writer); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out.String(), tc.detail) || strings.Contains(out.String(), "private-token") || strings.Contains(out.String(), "may have changed") != tc.uncertain {
					t.Fatalf("raw=%t diagnostic=%q", raw, out.String())
				}
			}
		})
	}
}
