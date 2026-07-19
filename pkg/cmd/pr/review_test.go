package pr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func TestPRReviewModeMapping(t *testing.T) {
	tests := []struct {
		name       string
		modeFlag   string
		body       string
		wantEvent  api.PullRequestReviewEvent
		wantOutput string
	}{
		{name: "approve", modeFlag: "approve", wantEvent: api.PullRequestReviewApprove, wantOutput: "Approved PR #42: https://atomgit.com/alice/demo/reviews/9\n"},
		{name: "request changes", modeFlag: "request-changes", body: "Please add tests.", wantEvent: api.PullRequestReviewRequestChanges, wantOutput: "Requested changes on PR #42: https://atomgit.com/alice/demo/reviews/9\n"},
		{name: "comment", modeFlag: "comment", body: "A few notes.", wantEvent: api.PullRequestReviewComment, wantOutput: "Submitted review comment on PR #42: https://atomgit.com/alice/demo/reviews/9\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := reviewTestFactory(t, func(req *http.Request) (*http.Response, error) {
				requests++
				switch requests {
				case 1:
					assertReviewRequest(t, req, http.MethodGet, "/api/v5/repos/alice/demo/pulls/42")
					return prResponse(http.StatusOK, `{"number":42,"state":"open","html_url":"https://atomgit.com/alice/demo/pulls/42","user":{"login":"bob"}}`), nil
				case 2:
					assertReviewRequest(t, req, http.MethodPost, "/api/v5/repos/alice/demo/pulls/42/review")
					var body api.PullRequestReviewRequest
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if body.Event != tt.wantEvent || body.Body != tt.body {
						t.Fatalf("review body = %#v, want event %q and body %q", body, tt.wantEvent, tt.body)
					}
					return prResponse(http.StatusOK, `{"id":"9","html_url":"https://atomgit.com/alice/demo/reviews/9"}`), nil
				default:
					t.Fatalf("unexpected request %d", requests)
					return nil, nil
				}
			})

			cmd := newCmdPRReview(factory)
			setFlag(t, cmd, tt.modeFlag, "true")
			if tt.body != "" {
				setFlag(t, cmd, "body", tt.body)
			}
			var output strings.Builder
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
				t.Fatal(err)
			}
			if requests != 2 {
				t.Fatalf("requests = %d, want 2", requests)
			}
			if output.String() != tt.wantOutput {
				t.Fatalf("output = %q, want %q", output.String(), tt.wantOutput)
			}
		})
	}
}

func TestPRReviewBodySources(t *testing.T) {
	tempDir := t.TempDir()
	bodyPath := filepath.Join(tempDir, "review.md")
	if err := os.WriteFile(bodyPath, []byte("body from file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		configure func(*cobra.Command)
		editor    reviewEditor
		wantBody  string
	}{
		{
			name: "file",
			configure: func(cmd *cobra.Command) {
				setFlag(t, cmd, "body-file", bodyPath)
			},
			wantBody: "body from file\n",
		},
		{
			name: "standard input",
			configure: func(cmd *cobra.Command) {
				setFlag(t, cmd, "body-file", "-")
				cmd.SetIn(strings.NewReader("body from stdin\n"))
			},
			wantBody: "body from stdin\n",
		},
		{
			name: "editor",
			configure: func(cmd *cobra.Command) {
				setFlag(t, cmd, "editor", "true")
			},
			editor: func(*cobra.Command) (string, error) {
				return "body from editor\n", nil
			},
			wantBody: "body from editor\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := reviewTestFactory(t, reviewSuccessTransport(t, api.PullRequestReviewComment, tt.wantBody))
			cmd := newCmdPRReviewWithEditor(factory, tt.editor)
			setFlag(t, cmd, "comment", "true")
			tt.configure(cmd)
			if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPRReviewRejectsFlagConflictsBeforeRequest(t *testing.T) {
	tests := []struct {
		name      string
		flags     map[string]string
		wantError string
	}{
		{name: "missing mode", flags: map[string]string{}, wantError: "exactly one"},
		{name: "conflicting modes", flags: map[string]string{"approve": "true", "comment": "true"}, wantError: "exactly one"},
		{name: "body and file", flags: map[string]string{"comment": "true", "body": "text", "body-file": "review.md"}, wantError: "only one of --body"},
		{name: "body and editor", flags: map[string]string{"comment": "true", "body": "text", "editor": "true"}, wantError: "only one of --body"},
		{name: "request changes without body", flags: map[string]string{"request-changes": "true"}, wantError: "body is required"},
		{name: "comment without body", flags: map[string]string{"comment": "true"}, wantError: "body is required"},
		{name: "empty request changes body", flags: map[string]string{"request-changes": "true", "body": "  \n"}, wantError: "body is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := reviewTestFactory(t, func(*http.Request) (*http.Response, error) {
				t.Fatal("validation error made an HTTP request")
				return nil, nil
			})
			cmd := newCmdPRReview(factory)
			for name, value := range tt.flags {
				setFlag(t, cmd, name, value)
			}
			err := cmd.RunE(cmd, []string{"alice/demo", "42"})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestPRReviewRejectsInvalidBodySourcesBeforeRequest(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*cobra.Command)
		editor    reviewEditor
		wantError string
	}{
		{
			name: "missing file",
			configure: func(cmd *cobra.Command) {
				setFlag(t, cmd, "body-file", filepath.Join(t.TempDir(), "missing.md"))
			},
			wantError: "failed to read review body file",
		},
		{
			name: "empty editor body",
			configure: func(cmd *cobra.Command) {
				setFlag(t, cmd, "editor", "true")
			},
			editor: func(*cobra.Command) (string, error) {
				return " \n", nil
			},
			wantError: "body is required",
		},
		{
			name: "editor failure",
			configure: func(cmd *cobra.Command) {
				setFlag(t, cmd, "editor", "true")
			},
			editor: func(*cobra.Command) (string, error) {
				return "", fmt.Errorf("editor failed")
			},
			wantError: "editor failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := reviewTestFactory(t, func(*http.Request) (*http.Response, error) {
				t.Fatal("invalid body source made an HTTP request")
				return nil, nil
			})
			cmd := newCmdPRReviewWithEditor(factory, tt.editor)
			setFlag(t, cmd, "comment", "true")
			tt.configure(cmd)
			err := cmd.RunE(cmd, []string{"alice/demo", "42"})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestPRReviewValidatesPullRequestBeforeSubmission(t *testing.T) {
	tests := []struct {
		name      string
		state     string
		author    string
		wantError string
	}{
		{name: "closed", state: "closed", author: "bob", wantError: "because it is closed"},
		{name: "merged", state: "merged", author: "bob", wantError: "because it is merged"},
		{name: "own pull request", state: "open", author: "ALICE", wantError: "your own pull request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := reviewTestFactory(t, func(req *http.Request) (*http.Response, error) {
				requests++
				if requests > 1 {
					t.Fatal("preflight failure submitted a review")
				}
				body := fmt.Sprintf(`{"number":42,"state":%q,"user":{"login":%q}}`, tt.state, tt.author)
				return prResponse(http.StatusOK, body), nil
			})
			cmd := newCmdPRReview(factory)
			setFlag(t, cmd, "approve", "true")
			err := cmd.RunE(cmd, []string{"alice/demo", "42"})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want 1", requests)
			}
		})
	}
}

func TestPRReviewReturnsAPIErrors(t *testing.T) {
	tests := []struct {
		name       string
		failCall   int
		statusCode int
		body       string
		wantError  string
	}{
		{name: "permission denied during preflight", failCall: 1, statusCode: http.StatusForbidden, body: `{"message":"forbidden"}`, wantError: "403 Forbidden"},
		{name: "permission denied during review", failCall: 2, statusCode: http.StatusForbidden, body: `{"message":"forbidden"}`, wantError: "403 Forbidden"},
		{name: "duplicate approval", failCall: 2, statusCode: http.StatusConflict, body: `{"message":"already approved"}`, wantError: "already approved"},
		{name: "review API failure", failCall: 2, statusCode: http.StatusInternalServerError, body: `{"message":"review failed"}`, wantError: "500 Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			factory := reviewTestFactory(t, func(*http.Request) (*http.Response, error) {
				calls++
				if calls == tt.failCall {
					return prResponse(tt.statusCode, tt.body), nil
				}
				return prResponse(http.StatusOK, `{"number":42,"state":"open","user":{"login":"bob"}}`), nil
			})
			cmd := newCmdPRReview(factory)
			setFlag(t, cmd, "approve", "true")
			err := cmd.RunE(cmd, []string{"alice/demo", "42"})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestPRReviewCommandHelp(t *testing.T) {
	cmd := NewCmdPR(&cmdutil.Factory{})
	review, _, err := cmd.Find([]string{"review"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"approve", "request-changes", "comment", "body", "body-file", "editor"} {
		if review.Flags().Lookup(name) == nil {
			t.Fatalf("review flag %q was not registered", name)
		}
	}
	if !strings.Contains(review.Example, "ag pr review") {
		t.Fatalf("review examples = %q", review.Example)
	}
}

func TestParseReviewArgs(t *testing.T) {
	for _, tt := range []struct {
		name      string
		args      []string
		wantError string
	}{
		{name: "missing owner", args: []string{"demo", "42"}, wantError: "expected owner/repo"},
		{name: "extra repository segment", args: []string{"team/sub/demo", "42"}, wantError: "expected owner/repo"},
		{name: "non numeric number", args: []string{"alice/demo", "abc"}, wantError: "must be positive"},
		{name: "zero number", args: []string{"alice/demo", "0"}, wantError: "must be positive"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := parseReviewArgs(tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func reviewTestFactory(t *testing.T, transport prRoundTripFunc) *cmdutil.Factory {
	t.Helper()
	return &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		},
	}
}

func reviewSuccessTransport(t *testing.T, wantEvent api.PullRequestReviewEvent, wantBody string) prRoundTripFunc {
	t.Helper()
	requests := 0
	return func(req *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return prResponse(http.StatusOK, `{"number":42,"state":"open","user":{"login":"bob"}}`), nil
		}
		if requests != 2 {
			t.Fatalf("unexpected request %d", requests)
		}
		var body api.PullRequestReviewRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Event != wantEvent || body.Body != wantBody {
			t.Fatalf("review body = %#v, want event %q and body %q", body, wantEvent, wantBody)
		}
		return prResponse(http.StatusNoContent, ""), nil
	}
}

func assertReviewRequest(t *testing.T, req *http.Request, method, path string) {
	t.Helper()
	if req.Method != method || req.URL.Path != path {
		t.Fatalf("request = %s %s, want %s %s", req.Method, req.URL.Path, method, path)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer token" {
		t.Fatalf("Authorization = %q", got)
	}
}

func setFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	if err := cmd.Flags().Set(name, value); err != nil {
		t.Fatal(err)
	}
}
