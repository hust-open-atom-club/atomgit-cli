package pr

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type prTestConfig struct{}

func (prTestConfig) GetToken() (string, error) { return "token", nil }
func (prTestConfig) GetUser() (string, error)  { return "alice", nil }
func (prTestConfig) GetHost() string           { return "atomgit.com" }

type prRoundTripFunc func(*http.Request) (*http.Response, error)

func (f prRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestResolveBaseBranch(t *testing.T) {
	tests := []struct {
		name       string
		requested  string
		repository api.Repository
		want       string
		wantError  bool
	}{
		{name: "explicit branch", requested: " release ", repository: api.Repository{DefaultBranch: "main"}, want: "release"},
		{name: "repository default", repository: api.Repository{DefaultBranch: " main "}, want: "main"},
		{name: "missing branch", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveBaseBranch(tt.requested, tt.repository)
			if tt.wantError {
				if err == nil {
					t.Fatal("resolveBaseBranch() expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveBaseBranch() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveBaseBranch() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPRListHonorsLimit(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.URL.Path != "/api/v5/repos/alice/demo/pulls" || req.URL.Query().Get("state") != "closed" || req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("request URL = %s", req.URL.String())
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"number":"1","title":"first","state":"closed"},{"number":"2","title":"second","state":"closed"}]`)), Header: make(http.Header)}, nil
			})}, nil
		},
	}
	cmd := newCmdPRList(factory)
	_ = cmd.Flags().Set("state", "closed")
	_ = cmd.Flags().Set("limit", "1")
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestPRListRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []string{"0", "-1"} {
		t.Run(limit, func(t *testing.T) {
			cmd := newCmdPRList(&cmdutil.Factory{Config: prTestConfig{}})
			_ = cmd.Flags().Set("limit", limit)
			if err := cmd.RunE(cmd, []string{"alice/demo"}); err == nil || !strings.Contains(err.Error(), "must be positive") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
