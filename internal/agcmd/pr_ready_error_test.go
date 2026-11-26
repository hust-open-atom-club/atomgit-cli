package agcmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/pr"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestPullRequestDraftUncertainDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cause  error
		detail string
	}{
		{"server error", &api.HTTPError{StatusCode: 503, Body: "private-body", Status: "private-status"}, "The server returned HTTP 503."},
		{"invalid response", fmt.Errorf("private-body: %w", api.ErrInvalidResponse), "The server returned an invalid response."},
		{"transport", &api.RequestExecutionError{Err: errors.New("private-network")}, "The HTTP request could not be completed."},
		{"read", &api.ResponseReadError{Err: errors.New("private-read")}, "The server response could not be read."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, raw := range []bool{false, true} {
				var out bytes.Buffer
				writer := cmdutil.NewSanitizingWriter(&out)
				if err := writer.SetRaw(raw); err != nil {
					t.Fatal(err)
				}
				renderCommandError(writer, fmt.Errorf("update PR #42: %w", &api.PullRequestDraftUpdateUncertainError{Err: tc.cause}))
				if err := cmdutil.FlushWriter(writer); err != nil {
					t.Fatal(err)
				}
				want := "Error: Pull request draft status could not be confirmed.\n\n" + tc.detail +
					"\n\nThe pull request may have changed. Check AtomGit before retrying.\n"
				if out.String() != want {
					t.Fatalf("raw=%t got=%q want=%q", raw, out.String(), want)
				}
			}
		})
	}
}

type prReadyConfig struct{}

func (prReadyConfig) GetToken() (string, error) { return "token", nil }
func (prReadyConfig) GetUser() (string, error)  { return "alice", nil }
func (prReadyConfig) GetHost() string           { return "atomgit.com" }

func prReadyResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// TestPRReadyReportsUncertainWritesAtRendering verifies the real CLI renderer
// keeps the write uncertainty instead of unwrapping to the HTTP or decode cause.
func TestPRReadyReportsUncertainWritesAtRendering(t *testing.T) {
	for _, tt := range []struct {
		name string
		// handle receives the request method plus the number of earlier GET and
		// PATCH requests. It can assert the expected request sequence.
		handle func(t *testing.T, method string, reads, writes int) (*http.Response, error)
		want   string
	}{
		{
			name: "verification server error",
			handle: func(t *testing.T, method string, reads, _ int) (*http.Response, error) {
				if method == http.MethodPatch {
					return prReadyResponse(http.StatusOK, `{}`), nil
				}
				if reads == 1 {
					return prReadyResponse(http.StatusOK, `{"draft":true,"state":"open"}`), nil
				}
				return prReadyResponse(http.StatusServiceUnavailable, `{"message":"private-body"}`), nil
			},
			want: "Error: Pull request draft status could not be confirmed.\n\n" +
				"The server returned HTTP 503.\n\n" +
				"The pull request may have changed. Check AtomGit before retrying.\n",
		},
		{
			name: "verification malformed readback",
			handle: func(t *testing.T, method string, reads, _ int) (*http.Response, error) {
				if method == http.MethodPatch {
					return prReadyResponse(http.StatusOK, `{}`), nil
				}
				if reads == 1 {
					return prReadyResponse(http.StatusOK, `{"draft":true,"state":"open"}`), nil
				}
				return prReadyResponse(http.StatusOK, `{"draft":`), nil
			},
			want: "Error: Pull request draft status could not be confirmed.\n\n" +
				"The server returned an invalid response.\n\n" +
				"The pull request may have changed. Check AtomGit before retrying.\n",
		},
		{
			name: "patch server error",
			handle: func(t *testing.T, method string, _ int, writes int) (*http.Response, error) {
				if method == http.MethodGet {
					return prReadyResponse(http.StatusOK, `{"draft":true,"state":"open"}`), nil
				}
				if writes != 1 {
					t.Fatalf("PATCH replayed %d times", writes)
				}
				return prReadyResponse(http.StatusInternalServerError, `{"message":"private-body"}`), nil
			},
			want: "Error: Pull request draft status could not be confirmed.\n\n" +
				"The server returned HTTP 500.\n\n" +
				"The pull request may have changed. Check AtomGit before retrying.\n",
		},
		{
			name: "patch rejected",
			handle: func(t *testing.T, method string, _ int, _ int) (*http.Response, error) {
				if method == http.MethodGet {
					return prReadyResponse(http.StatusOK, `{"draft":true,"state":"open"}`), nil
				}
				return prReadyResponse(http.StatusUnprocessableEntity, `{"message":"private-body"}`), nil
			},
			want: "Error: API request failed.\n\nThe server returned HTTP 422.\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reads, writes := 0, 0
			factory := &cmdutil.Factory{
				Config: prReadyConfig{},
				HTTPClient: func() (*http.Client, error) {
					return &http.Client{Transport: statusTransport(func(req *http.Request) (*http.Response, error) {
						switch req.Method {
						case http.MethodGet:
							reads++
						case http.MethodPatch:
							writes++
						}
						return tt.handle(t, req.Method, reads, writes)
					})}, nil
				},
			}
			cmd := pr.NewCmdPR(factory)
			cmd.SilenceErrors, cmd.SilenceUsage = true, true
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stdout)
			cmd.SetArgs([]string{"ready", "alice/demo", "42"})
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected a draft update error")
			}

			var rendered bytes.Buffer
			writer := cmdutil.NewSanitizingWriter(&rendered)
			renderCommandError(writer, err)
			if flushErr := cmdutil.FlushWriter(writer); flushErr != nil {
				t.Fatal(flushErr)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if rendered.String() != tt.want {
				t.Fatalf("rendered = %q, want %q", rendered.String(), tt.want)
			}
		})
	}
}
