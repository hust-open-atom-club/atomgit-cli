package comment

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestPRCommentEditPropagatesCurrentUserError(t *testing.T) {
	factory := &cmdutil.Factory{Config: prCommentTestConfig{userErr: errors.New("user lookup failed")}}
	cmd := newCmdEdit(factory)
	_ = cmd.Flags().Set("body", "new")
	err := cmd.RunE(cmd, []string{"alice/demo", "8", "42"})
	if err == nil || !strings.Contains(err.Error(), "failed to get current user: user lookup failed") {
		t.Fatalf("error = %v, want current-user lookup error", err)
	}
}

func prJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestPRCommentEditVerifiesNestedParentBeforePatch(t *testing.T) {
	var methods []string
	factory := &cmdutil.Factory{
		Config: prCommentTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method+" "+req.URL.RequestURI())
				switch req.Method {
				case http.MethodGet:
					if req.URL.Path != "/api/v5/repos/alice/demo/pulls/8/comments" || req.URL.Query().Get("view") != "all" {
						t.Fatalf("parent lookup = %s?%s", req.URL.Path, req.URL.RawQuery)
					}
					return prJSONResponse(http.StatusOK, `[{"id":7,"reply":[{"id":42,"body":"old","user":{"login":"alice"}}]}]`), nil
				case http.MethodPatch:
					if req.URL.Path != "/api/v5/repos/alice/demo/pulls/comments/42" {
						t.Fatalf("patch path = %s", req.URL.Path)
					}
					return prJSONResponse(http.StatusOK, `{"id":42,"body":"new","user":{"login":"alice"}}`), nil
				default:
					t.Fatalf("unexpected method %s", req.Method)
					return nil, nil
				}
			})}, nil
		},
	}
	cmd := newCmdEdit(factory)
	_ = cmd.Flags().Set("body", "new")
	if err := cmd.RunE(cmd, []string{"alice/demo", "8", "42"}); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(methods, ","), "GET /api/v5/repos/alice/demo/pulls/8/comments?view=all,PATCH /api/v5/repos/alice/demo/pulls/comments/42"; got != want {
		t.Fatalf("request order = %q, want %q", got, want)
	}
}

func TestPRCommentDeleteRejectsCommentFromAnotherPullRequest(t *testing.T) {
	deleteCount := 0
	factory := &cmdutil.Factory{
		Config: prCommentTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodDelete {
					deleteCount++
				}
				if req.Method != http.MethodGet {
					t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
				}
				return prJSONResponse(http.StatusOK, `[{"id":99,"user":{"login":"alice"}}]`), nil
			})}, nil
		},
	}
	cmd := newCmdDelete(factory)
	_ = cmd.Flags().Set("yes", "true")
	err := cmd.RunE(cmd, []string{"alice/demo", "8", "42"})
	if err == nil || !strings.Contains(err.Error(), "does not belong to pull request #8") {
		t.Fatalf("error = %v, want parent mismatch", err)
	}
	if deleteCount != 0 {
		t.Fatalf("delete requests = %d, want 0", deleteCount)
	}
}
