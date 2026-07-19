package pr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func TestPRReviewUsesAtomGitRequestModel(t *testing.T) {
	tests := []struct {
		name       string
		force      bool
		wantOutput string
	}{
		{name: "approve", wantOutput: "Approved PR #42: https://atomgit.com/alice/demo/pulls/42\n"},
		{name: "force approve", force: true, wantOutput: "Force-approved PR #42: https://atomgit.com/alice/demo/pulls/42\n"},
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
					if body.Force != tt.force {
						t.Fatalf("force = %v, want %v", body.Force, tt.force)
					}
					return prResponse(http.StatusNoContent, ""), nil
				default:
					t.Fatalf("unexpected request %d", requests)
					return nil, nil
				}
			})

			cmd := newCmdPRReview(factory)
			setFlag(t, cmd, "approve", "true")
			if tt.force {
				setFlag(t, cmd, "force", "true")
			}
			var output strings.Builder
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
				t.Fatal(err)
			}
			if output.String() != tt.wantOutput {
				t.Fatalf("output = %q, want %q", output.String(), tt.wantOutput)
			}
		})
	}
}

func TestPRReviewRequiresSupportedModeBeforeRequest(t *testing.T) {
	factory := reviewTestFactory(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("missing review mode made an HTTP request")
		return nil, nil
	})
	cmd := newCmdPRReview(factory)
	err := cmd.RunE(cmd, []string{"alice/demo", "42"})
	if err == nil || !strings.Contains(err.Error(), "--approve is required") {
		t.Fatalf("error = %v", err)
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
			factory := reviewTestFactory(t, func(*http.Request) (*http.Response, error) {
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
	for _, name := range []string{"approve", "force"} {
		if review.Flags().Lookup(name) == nil {
			t.Fatalf("review flag %q was not registered", name)
		}
	}
	for _, name := range []string{"request-changes", "comment", "body", "body-file", "editor"} {
		if review.Flags().Lookup(name) != nil {
			t.Fatalf("unsupported review flag %q was registered", name)
		}
	}
	if !strings.Contains(review.Long, "not supported") || !strings.Contains(review.Example, "ag pr review") {
		t.Fatalf("review help is missing API limitations or examples")
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
