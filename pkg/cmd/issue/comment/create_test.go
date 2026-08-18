package comment

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type issueCommentTestConfig struct{ tokenErr error }

func (c issueCommentTestConfig) GetToken() (string, error) { return "token", c.tokenErr }
func (issueCommentTestConfig) GetUser() (string, error)    { return "alice", nil }
func (issueCommentTestConfig) GetHost() string             { return "atomgit.com" }

type issueCommentRoundTripFunc func(*http.Request) (*http.Response, error)

func (f issueCommentRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestIssueCommentCreateFallsBackToIssueURL(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: issueCommentTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueCommentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/issues/8/comments" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				return &http.Response{
					StatusCode: http.StatusCreated,
					Status:     fmt.Sprintf("%d %s", http.StatusCreated, http.StatusText(http.StatusCreated)),
					Body:       io.NopCloser(strings.NewReader(`{"id":180041703,"body":"Looks good"}`)),
					Header:     make(http.Header),
				}, nil
			})}, nil
		},
	}

	cmd := newCmdCreate(factory)
	_ = cmd.Flags().Set("body", "Looks good")
	var output strings.Builder
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "8"}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "Created comment #180041703: https://atomgit.com/alice/demo/issues/8\n" {
		t.Fatalf("output = %q", got)
	}
}
