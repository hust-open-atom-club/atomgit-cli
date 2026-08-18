package comment

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type prCommentTestConfig struct{ tokenErr error }

func (c prCommentTestConfig) GetToken() (string, error) { return "token", c.tokenErr }
func (prCommentTestConfig) GetUser() (string, error)    { return "alice", nil }
func (prCommentTestConfig) GetHost() string             { return "atomgit.com" }

type prCommentRoundTripFunc func(*http.Request) (*http.Response, error)

func (f prCommentRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestPRCommentCreateFallsBackToPullRequestURL(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prCommentTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/pulls/8/comments" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				return &http.Response{
					StatusCode: http.StatusCreated,
					Status:     fmt.Sprintf("%d %s", http.StatusCreated, http.StatusText(http.StatusCreated)),
					Body:       io.NopCloser(strings.NewReader(`{"id":"180041704","body":"Please review"}`)),
					Header:     make(http.Header),
				}, nil
			})}, nil
		},
	}

	cmd := newCmdCreate(factory)
	_ = cmd.Flags().Set("body", "Please review")
	var output strings.Builder
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "8"}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "Created comment #180041704 on PR #8: https://atomgit.com/alice/demo/pull/8\n" {
		t.Fatalf("output = %q", got)
	}
}
