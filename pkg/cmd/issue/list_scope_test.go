package issue

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// --author/--assignee @me lists issues across all repositories through the
// authenticated-user endpoint, mapping the flags onto the server-side filter
// values; other values and flag combinations are rejected before auth.
func TestIssueListUserScopedFilterRequests(t *testing.T) {
	for _, tt := range []struct {
		name        string
		flags       map[string]string
		args        []string
		wantErr     string
		wantPath    string
		wantFilter  string
		wantState   string
		wantNoQuery bool // request must NOT hit /user/issues (repo endpoint)
	}{
		{
			name:       "author @me",
			flags:      map[string]string{"author": "@me"},
			wantPath:   "/api/v5/user/issues",
			wantFilter: "created",
			wantState:  "open",
		},
		{
			name:       "assignee @me",
			flags:      map[string]string{"assignee": "@me", "state": "closed"},
			wantPath:   "/api/v5/user/issues",
			wantFilter: "assigned",
			wantState:  "closed",
		},
		{
			name:       "author @me with state all",
			flags:      map[string]string{"author": "@me", "state": "all"},
			wantPath:   "/api/v5/user/issues",
			wantFilter: "created",
			wantState:  "all",
		},
		{
			name:       "involved @me",
			flags:      map[string]string{"involved": "@me"},
			wantPath:   "/api/v5/user/issues",
			wantFilter: "all",
			wantState:  "open",
		},
		{
			name:    "author rejects other users",
			flags:   map[string]string{"author": "alice"},
			wantErr: `--author only supports @me, got "alice"`,
		},
		{
			name:    "assignee rejects other users",
			flags:   map[string]string{"assignee": "bob"},
			wantErr: `--assignee only supports @me, got "bob"`,
		},
		{
			name:    "involved rejects other users",
			flags:   map[string]string{"involved": "carol"},
			wantErr: `--involved only supports @me, got "carol"`,
		},
		{
			name:    "author and assignee are mutually exclusive",
			flags:   map[string]string{"author": "@me", "assignee": "@me"},
			wantErr: "--author, --assignee and --involved cannot be used together",
		},
		{
			name:    "all three flags are mutually exclusive",
			flags:   map[string]string{"author": "@me", "assignee": "@me", "involved": "@me"},
			wantErr: "--author, --assignee and --involved cannot be used together",
		},
		{
			name:    "author @me rejects explicit repository",
			flags:   map[string]string{"author": "@me"},
			args:    []string{"alice/demo"},
			wantErr: "cannot be combined with an explicit <owner>/<repo>",
		},
		{
			name:    "involved @me rejects explicit repository",
			flags:   map[string]string{"involved": "@me"},
			args:    []string{"alice/demo"},
			wantErr: "cannot be combined with an explicit <owner>/<repo>",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: issueTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if tt.wantErr != "" {
							t.Fatalf("unexpected HTTP request: %s", req.URL.String())
						}
						if req.URL.Path != tt.wantPath {
							t.Fatalf("request path = %s, want %s", req.URL.Path, tt.wantPath)
						}
						query := req.URL.Query()
						if got := query.Get("filter"); got != tt.wantFilter {
							t.Fatalf("filter = %q, want %q", got, tt.wantFilter)
						}
						if got := query.Get("state"); got != tt.wantState {
							t.Fatalf("state = %q, want %q", got, tt.wantState)
						}
						if got := query.Get("page"); got != "1" {
							t.Fatalf("page = %q, want 1", got)
						}
						return issueResponse(http.StatusOK, `[{"number":"1","title":"first","state":"open"}]`), nil
					})}, nil
				},
			}
			cmd := newCmdIssueList(factory)
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
			if !strings.Contains(out.String(), "#1 first [open]") {
				t.Fatalf("output = %q", out.String())
			}
		})
	}
}

// Cross-repository @me listings must disambiguate entries: numbers are only
// unique within their own repository, so each line carries the owner/repo
// prefix derived from the issue URL (review feedback on PR #264).
func TestIssueListUserScopedOutputIncludesRepoIdentity(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(*http.Request) (*http.Response, error) {
				body := `[
					{"number":"1","title":"Update dependencies","state":"open","html_url":"https://atomgit.com/alice/one/issues/1"},
					{"number":"1","title":"Update dependencies","state":"open","html_url":"https://atomgit.com/bob/two/issues/1"},
					{"number":"2","title":"No URL fallback","state":"closed"}
				]`
				return issueResponse(http.StatusOK, body), nil
			})}, nil
		},
	}
	cmd := newCmdIssueList(factory)
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

// Without --author/--assignee the repository endpoint must be used unchanged.
func TestIssueListWithoutUserScopeFlagsKeepsRepoEndpoint(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.URL.Path != "/api/v5/repos/alice/demo/issues" {
					t.Fatalf("request path = %s", req.URL.Path)
				}
				if req.URL.Query().Get("filter") != "" {
					t.Fatalf("unexpected filter query: %s", req.URL.RawQuery)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`)), Header: make(http.Header)}, nil
			})}, nil
		},
	}
	cmd := newCmdIssueList(factory)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}
