package pr

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// --author/--assignee/--review-requested @me lists PRs across all repositories
// through the authenticated-user endpoint, mapping the flags onto the
// server-side scope values; other values and combinations are rejected before
// auth.
func TestPRListUserScopedScopeRequests(t *testing.T) {
	for _, tt := range []struct {
		name      string
		flags     map[string]string
		args      []string
		wantErr   string
		wantPath  string
		wantScope string
		wantState string
	}{
		{
			name:      "author @me",
			flags:     map[string]string{"author": "@me"},
			wantPath:  "/api/v5/user/pulls",
			wantScope: "created_by_me",
			wantState: "open",
		},
		{
			name:      "assignee @me",
			flags:     map[string]string{"assignee": "@me", "state": "closed"},
			wantPath:  "/api/v5/user/pulls",
			wantScope: "assigned_to_me",
			wantState: "closed",
		},
		{
			name:      "review-requested @me",
			flags:     map[string]string{"review-requested": "@me", "state": "all"},
			wantPath:  "/api/v5/user/pulls",
			wantScope: "need_my_approve",
			wantState: "all",
		},
		{
			name:      "review-needed @me",
			flags:     map[string]string{"review-needed": "@me"},
			wantPath:  "/api/v5/user/pulls",
			wantScope: "need_my_review",
			wantState: "open",
		},
		{
			name:    "author rejects other users",
			flags:   map[string]string{"author": "alice"},
			wantErr: `--author only supports @me, got "alice"`,
		},
		{
			name:    "review-requested rejects other users",
			flags:   map[string]string{"review-requested": "bob"},
			wantErr: `--review-requested only supports @me, got "bob"`,
		},
		{
			name:    "review-needed rejects other users",
			flags:   map[string]string{"review-needed": "dave"},
			wantErr: `--review-needed only supports @me, got "dave"`,
		},
		{
			name:    "author and assignee are mutually exclusive",
			flags:   map[string]string{"author": "@me", "assignee": "@me"},
			wantErr: "cannot be used together",
		},
		{
			name:    "all four flags are mutually exclusive",
			flags:   map[string]string{"author": "@me", "assignee": "@me", "review-requested": "@me", "review-needed": "@me"},
			wantErr: "cannot be used together",
		},
		{
			name:    "author @me rejects explicit repository",
			flags:   map[string]string{"author": "@me"},
			args:    []string{"alice/demo"},
			wantErr: "cannot be combined with an explicit <owner>/<repo>",
		},
		{
			name:    "review-needed @me rejects explicit repository",
			flags:   map[string]string{"review-needed": "@me"},
			args:    []string{"alice/demo"},
			wantErr: "cannot be combined with an explicit <owner>/<repo>",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: prTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if tt.wantErr != "" {
							t.Fatalf("unexpected HTTP request: %s", req.URL.String())
						}
						if req.URL.Path != tt.wantPath {
							t.Fatalf("request path = %s, want %s", req.URL.Path, tt.wantPath)
						}
						query := req.URL.Query()
						if got := query.Get("scope"); got != tt.wantScope {
							t.Fatalf("scope = %q, want %q", got, tt.wantScope)
						}
						if got := query.Get("state"); got != tt.wantState {
							t.Fatalf("state = %q, want %q", got, tt.wantState)
						}
						if got := query.Get("page"); got != "1" {
							t.Fatalf("page = %q, want 1", got)
						}
						return prResponse(http.StatusOK, `[{"number":"9","title":"change","state":"open"}]`), nil
					})}, nil
				},
			}
			cmd := newCmdPRList(factory)
			for flag, value := range tt.flags {
				if err := cmd.Flags().Set(flag, value); err != nil {
					t.Fatalf("set %s: %v", flag, err)
				}
			}
			var out bytes.Buffer
			cmd.SetOut(&out)
			err := cmd.RunE(cmd, tt.args)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
				}
				if requests != 0 {
					t.Fatalf("requests = %d, want 0 (must fail before any HTTP call)", requests)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "#9 change [open]") {
				t.Fatalf("output = %q", out.String())
			}
		})
	}
}

// Cross-repository @me listings must disambiguate entries: numbers are only
// unique within their own repository, so each line carries the owner/repo
// prefix derived from the PR URL (review feedback on PR #264).
func TestPRListUserScopedOutputIncludesRepoIdentity(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(*http.Request) (*http.Response, error) {
				body := `[
					{"number":"1","title":"Update dependencies","state":"open","html_url":"https://atomgit.com/alice/one/pull/1"},
					{"number":"1","title":"Update dependencies","state":"open","html_url":"https://atomgit.com/bob/two/pulls/1"},
					{"number":"2","title":"No URL fallback","state":"closed"}
				]`
				return prResponse(http.StatusOK, body), nil
			})}, nil
		},
	}
	cmd := newCmdPRList(factory)
	if err := cmd.Flags().Set("author", "@me"); err != nil {
		t.Fatalf("set author: %v", err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	want := "alice/one #1 Update dependencies [open]\nbob/two #1 Update dependencies [open]\n#2 No URL fallback [closed]\n"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

// Without --author/--assignee/--review-requested the repository endpoint must
// be used unchanged.
func TestPRListWithoutUserScopeFlagsKeepsRepoEndpoint(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.URL.Path != "/api/v5/repos/alice/demo/pulls" {
					t.Fatalf("request path = %s", req.URL.Path)
				}
				if req.URL.Query().Get("scope") != "" {
					t.Fatalf("unexpected scope query: %s", req.URL.RawQuery)
				}
				return prResponse(http.StatusOK, `[]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRList(factory)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}
