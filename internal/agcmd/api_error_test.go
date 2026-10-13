package agcmd

import (
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAPIErrorDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		cause  error
		detail string
	}{
		{
			"v5 status",
			&api.HTTPError{StatusCode: 401, Status: "private-status", Body: "private-body", RetryAfter: "private-retry"},
			"The server returned HTTP 401.",
		},
		{
			"v5 secondary deadline",
			&api.HTTPError{StatusCode: 403, ReadError: context.DeadlineExceeded},
			"The server returned HTTP 403.\nThe error response body could not be read.",
		},
		{
			"Actions secondary cancel",
			&actions.HTTPError{
				Operation:  "private-operation",
				StatusCode: 503,
				Message:    "private-body",
				ReadError:  context.Canceled,
			},
			"The server returned HTTP 503.\nThe error response body could not be read.",
		},
		{
			"Actions status",
			&actions.HTTPError{StatusCode: 429, Status: "private-status", RetryAfter: "private-retry"},
			"The server returned HTTP 429.",
		},
		{
			"preparation cause",
			fmt.Errorf("%w: %w", api.ErrRequestPreparation, context.DeadlineExceeded),
			"The HTTP request could not be prepared.",
		},
		{
			"invalid response cause",
			fmt.Errorf("%w: %w", api.ErrInvalidResponse, context.Canceled),
			"The server returned an invalid response.",
		},
		{
			"execution",
			&api.RequestExecutionError{Err: errors.New("private-network")},
			"The HTTP request could not be completed.",
		},
		{"execution canceled", &api.RequestExecutionError{Err: context.Canceled}, "The HTTP request was canceled."},
		{"execution deadline", &api.RequestExecutionError{Err: context.DeadlineExceeded}, "The HTTP request timed out."},
		{
			"execution network timeout",
			&api.RequestExecutionError{Err: &net.DNSError{Err: "private-network", IsTimeout: true}},
			"The HTTP request timed out.",
		},
		{"read", &api.ResponseReadError{Err: errors.New("private-read")}, "The server response could not be read."},
		{"read canceled", &api.ResponseReadError{Err: context.Canceled}, "Reading the server response was canceled."},
		{
			"read timeout",
			&api.ResponseReadError{Err: &net.DNSError{Err: "private-read", IsTimeout: true}},
			"Reading the server response timed out.",
		},
		{
			"read credential cause",
			&api.ResponseReadError{Err: config.ErrNotAuthenticated},
			"The server response could not be read.",
		},
		{
			"HTTP credential cause",
			&api.HTTPError{StatusCode: 404, ReadError: config.ErrNotAuthenticated},
			"The server returned HTTP 404.\nThe error response body could not be read.",
		},
		{
			"read role only",
			fmt.Errorf(
				"read: %w; decode: %w",
				&api.ResponseReadError{Err: errors.New("private-read")},
				context.DeadlineExceeded,
			),
			"The server response could not be read.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("private-prefix: %w", tt.cause)
			before := err.Error()
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
				want := "Error: API request failed.\n\n" + tt.detail + "\n"
				if out.String() != want {
					t.Fatalf("raw=%t: output = %q, want %q", raw, out.String(), want)
				}
			}
			if before != err.Error() || !errors.Is(err, tt.cause) {
				t.Fatal("presentation changed the error chain")
			}
		})
	}
}

type apiProcessReadFailure struct{ cause error }

func (r apiProcessReadFailure) Read([]byte) (int, error) {
	return 0, fmt.Errorf("password=reader-secret xyz789: %w", r.cause)
}

func TestAPIErrorProcess(t *testing.T) {
	if os.Getenv("AG_API_ERROR_TEST_PROCESS") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("AG_API_ERROR_TEST_ARGS")), &args); err != nil {
		t.Fatal(err)
	}
	requests := 0
	http.DefaultTransport = statusTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		if requests != 1 || r.Header.Get("Authorization") != "Bearer xyz789" {
			panic("unexpected API test request")
		}
		if r.URL.Host != "api.atomgit.com" && r.URL.Host != "actions-api.atomgit.com" {
			panic("unexpected API test host")
		}
		scenario := os.Getenv("AG_API_ERROR_TEST_SCENARIO")
		if scenario == "execution" {
			return nil, errors.New("private-network xyz789")
		}
		code, body := http.StatusOK, `[]`
		switch scenario {
		case "HTTP":
			code, body = http.StatusForbidden, `{"message":"private-body xyz789"}`
		case "HTTP read":
			code = http.StatusForbidden
		case "invalid":
			body = `{"private-body":"xyz789"`
		}
		var reader io.Reader = strings.NewReader(body)
		if scenario == "read" || scenario == "HTTP read" {
			reader = apiProcessReadFailure{cause: context.DeadlineExceeded}
		}
		if scenario == "read plain" {
			reader = apiProcessReadFailure{cause: errors.New("private-read")}
		}
		return &http.Response{
			StatusCode: code, Status: "private-status", Header: make(http.Header),
			Body: io.NopCloser(reader), Request: r,
		}, nil
	})
	os.Args = append([]string{"ag-cli"}, args...)
	os.Exit(Main())
}

func TestAPIErrorProcessPresentation(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "ag-cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"access_token":"xyz789","user":"alice"}`)
	if err := os.WriteFile(filepath.Join(dir, "token.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, scenario, detail string
		args                   []string
	}{
		{"api HTTP", "HTTP", "The server returned HTTP 403.", []string{"api", "/user", "-X", "POST"}},
		{"api raw read", "read plain", "The server response could not be read.", []string{"api", "/user"}},
		{"api raw read timeout", "read", "Reading the server response timed out.", []string{"api", "/user"}},
		{
			"api HTTP secondary read",
			"HTTP read",
			"The server returned HTTP 403.\nThe error response body could not be read.",
			[]string{"api", "/user", "-X", "POST"},
		},
		{
			"api invalid page",
			"invalid",
			"The server returned an invalid response.",
			[]string{"api", "/users", "--paginate"},
		},
		{
			"api interrupted page",
			"read",
			"Reading the server response timed out.",
			[]string{"api", "/users", "--paginate"},
		},
		{
			"api execution",
			"execution",
			"The HTTP request could not be completed.",
			[]string{"api", "/user", "-X", "POST"},
		},
		{"Actions HTTP", "HTTP", "The server returned HTTP 403.", []string{"run", "view", "owner/repo", "123"}},
		{
			"Actions HTTP secondary read",
			"HTTP read",
			"The server returned HTTP 403.\nThe error response body could not be read.",
			[]string{"run", "view", "owner/repo", "123"},
		},
		{"v5 HTTP", "HTTP", "The server returned HTTP 403.", []string{"user", "view"}},
		{"v5 interrupted response", "read", "Reading the server response timed out.", []string{"user", "view"}},
		{"v5 invalid JSON", "invalid", "The server returned an invalid response.", []string{"user", "view"}},
		{
			"Actions interrupted response",
			"read",
			"Reading the server response timed out.",
			[]string{"run", "view", "owner/repo", "123"},
		},
		{
			"Actions invalid JSON",
			"invalid",
			"The server returned an invalid response.",
			[]string{"run", "view", "owner/repo", "123"},
		},
		{"success", "success", "", []string{"api", "/user"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, raw := range []bool{false, true} {
				args := append([]string(nil), tt.args...)
				if raw {
					args = append([]string{"--raw-output"}, args...)
				}
				encoded, err := json.Marshal(args)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAPIErrorProcess$")
				cmd.Env = append(os.Environ(), "AG_API_ERROR_TEST_PROCESS=1",
					"AG_API_ERROR_TEST_ARGS="+string(encoded), "AG_API_ERROR_TEST_SCENARIO="+tt.scenario)
				var out, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &out, &stderr
				err = cmd.Run()
				cancel()
				if tt.scenario == "success" {
					if err != nil || out.String() != "[]" || stderr.Len() != 0 {
						t.Fatalf("raw=%t: success: err=%v stdout=%q stderr=%q", raw, err, out.String(), stderr.String())
					}
					continue
				}
				exit, ok := errors.AsType[*exec.ExitError](err)
				if !ok || exit.ExitCode() != 1 {
					t.Fatalf("process error = %v, want exit 1; stderr=%q", err, stderr.String())
				}
				want := "Error: API request failed.\n\n" + tt.detail + "\n"
				if out.Len() != 0 || stderr.String() != want {
					t.Fatalf("raw=%t: stdout=%q stderr=%q, want stderr=%q", raw, out.String(), stderr.String(), want)
				}
			}
		})
	}
}
