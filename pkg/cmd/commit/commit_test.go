package commit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type commitTestConfig struct {
	tokenCalls int
	tokenErr   error
}

func (c *commitTestConfig) GetToken() (string, error) {
	c.tokenCalls++
	if c.tokenErr != nil {
		return "", c.tokenErr
	}
	return "token", nil
}

func (*commitTestConfig) GetUser() (string, error) { return "alice", nil }
func (*commitTestConfig) GetHost() string          { return "atomgit.com" }

type commitAuthErrorConfig struct{}

func (commitAuthErrorConfig) GetToken() (string, error) {
	return "", errors.New("not authenticated: run `ag auth login`")
}
func (commitAuthErrorConfig) GetUser() (string, error) { return "alice", nil }
func (commitAuthErrorConfig) GetHost() string          { return "atomgit.com" }

type commitRoundTripFunc func(*http.Request) (*http.Response, error)

func (f commitRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func commitResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func factory(config *commitTestConfig, transport commitRoundTripFunc) *cmdutil.Factory {
	f := &cmdutil.Factory{Config: config}
	if transport != nil {
		f.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return f
}

func TestNewCmdCommitRegistersSubcommands(t *testing.T) {
	cmd := NewCmdCommit(&cmdutil.Factory{})
	want := map[string]bool{
		"list": false, "view": false, "compare": false, "diff": false, "patch": false,
	}
	for _, child := range cmd.Commands() {
		if _, ok := want[child.Name()]; ok {
			want[child.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q was not registered", name)
		}
	}

	list, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ref", "path", "since", "until", "limit", "json"} {
		if list.Flags().Lookup(name) == nil {
			t.Fatalf("list flag %q was not registered", name)
		}
	}
	if !strings.Contains(list.Long, cmdutil.RepositoryContextHelp) {
		t.Fatal("list help does not explain repository inference")
	}
	if err := list.Args(list, []string{"one", "two"}); err == nil {
		t.Fatal("list accepted too many arguments")
	}

	view, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"web", "json"} {
		if view.Flags().Lookup(name) == nil {
			t.Fatalf("view flag %q was not registered", name)
		}
	}

	for _, name := range []string{"compare", "diff", "patch"} {
		child, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(child.Long, cmdutil.RepositoryContextHelp) {
			t.Errorf("%s help does not explain repository inference", name)
		}
	}
}

func TestCommitListInfersRepositoryAndForwardsFilters(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/commits" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				query := req.URL.Query()
				if got := query.Get("sha"); got != "feature/foo" {
					t.Fatalf("sha filter = %q, want feature/foo", got)
				}
				if got := query.Get("path"); got != "src/main.go" {
					t.Fatalf("path filter = %q, want src/main.go", got)
				}
				if got := query.Get("since"); got != "2026-07-01T00:00:00Z" {
					t.Fatalf("since filter = %q", got)
				}
				if got := query.Get("until"); got != "2026-07-31T00:00:00Z" {
					t.Fatalf("until filter = %q", got)
				}
				if got := query.Get("page"); got != "1" {
					t.Fatalf("page = %q, want 1", got)
				}
				if got := query.Get("per_page"); got != "100" {
					t.Fatalf("per_page = %q, want 100", got)
				}
				return commitResponse(http.StatusOK, commitListBody(1)), nil
			})}, nil
		},
	}

	cmd := newCmdCommitList(factory)
	_ = cmd.Flags().Set("ref", "feature/foo")
	_ = cmd.Flags().Set("path", "src/main.go")
	_ = cmd.Flags().Set("since", "2026-07-01T00:00:00Z")
	_ = cmd.Flags().Set("until", "2026-07-31T00:00:00Z")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestCommitListOmitsUnusedFilters(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				query := req.URL.Query()
				for _, name := range []string{"sha", "path", "since", "until"} {
					if query.Get(name) != "" {
						t.Fatalf("unexpected filter %s = %q", name, query.Get(name))
					}
				}
				return commitResponse(http.StatusOK, `[]`), nil
			})}, nil
		},
	}

	cmd := newCmdCommitList(factory)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCommitListPagination(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				page := req.URL.Query().Get("page")
				if got := req.URL.Query().Get("per_page"); got != "100" {
					t.Fatalf("per_page = %q, want 100", got)
				}
				switch page {
				case "1":
					return commitResponse(http.StatusOK, commitListBody(100)), nil
				case "2":
					return commitResponse(http.StatusOK, commitListBody(30)), nil
				default:
					t.Fatalf("unexpected page %q", page)
					return nil, nil
				}
			})}, nil
		},
	}

	cmd := newCmdCommitList(factory)
	_ = cmd.Flags().Set("limit", "150")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if got := strings.Count(output.String(), "\n"); got != 130 {
		t.Fatalf("output lines = %d, want 130", got)
	}
}

func TestCommitListRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []string{"0", "-1"} {
		t.Run(limit, func(t *testing.T) {
			cmd := newCmdCommitList(&cmdutil.Factory{Config: &commitTestConfig{}})
			_ = cmd.Flags().Set("limit", limit)
			if err := cmd.RunE(cmd, []string{"alice/demo"}); err == nil || !strings.Contains(err.Error(), "must be positive") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestCommitListReturnsCanonicalAuthenticationError(t *testing.T) {
	cmd := newCmdCommitList(&cmdutil.Factory{Config: commitAuthErrorConfig{}})
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil || err.Error() != "not authenticated: run `ag auth login`" {
		t.Fatalf("error = %v", err)
	}
}

func TestCommitListEmptyOutput(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return commitResponse(http.StatusOK, `[]`), nil
			})}, nil
		},
	}

	cmd := newCmdCommitList(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "No commits found\n" {
		t.Fatalf("output = %q, want %q", got, "No commits found\n")
	}
}

func TestCommitListTextOutput(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return commitResponse(http.StatusOK, `[
					{"sha":"abcdef1234567890abcdef1234567890abcdef12","html_url":"https://atomgit.com/alice/demo/commit/abcdef1234567890abcdef1234567890abcdef12","commit":{"message":"Fix bug\nbody","author":{"name":"Alice","date":"2026-07-15T10:00:00Z"}},"author":{"login":"alice"}},
					{"sha":"1234567fedcba9876543210fedcba9876543210fe","html_url":"https://atomgit.com/alice/demo/commit/1234567fedcba9876543210fedcba9876543210fe","commit":{"message":"Second commit","author":{"name":"Bob","date":"2026-07-16T10:00:00Z"}},"author":{"login":"bob"}}
				]`), nil
			})}, nil
		},
	}

	cmd := newCmdCommitList(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	want := "abcdef1\tFix bug\talice\t2026-07-15T10:00:00Z\thttps://atomgit.com/alice/demo/commit/abcdef1234567890abcdef1234567890abcdef12\n" +
		"1234567\tSecond commit\tbob\t2026-07-16T10:00:00Z\thttps://atomgit.com/alice/demo/commit/1234567fedcba9876543210fedcba9876543210fe\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestCommitListJSON(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return commitResponse(http.StatusOK, `[{"sha":"abcdef1234567890abcdef1234567890abcdef12","html_url":"https://atomgit.com/alice/demo/commit/x","commit":{"message":"Fix bug\nbody","author":{"name":"Alice","date":"2026-07-15T10:00:00Z"}},"author":{"login":"alice"}}]`), nil
			})}, nil
		},
	}

	cmd := newCmdCommitList(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	var values []struct {
		SHA    string `json:"sha"`
		Title  string `json:"title"`
		Author string `json:"author"`
		Date   string `json:"date"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(output.Bytes(), &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 {
		t.Fatalf("len(values) = %d, want 1", len(values))
	}
	if values[0].SHA != "abcdef1234567890abcdef1234567890abcdef12" ||
		values[0].Title != "Fix bug" ||
		values[0].Author != "alice" ||
		values[0].Date != "2026-07-15T10:00:00Z" ||
		values[0].URL != "https://atomgit.com/alice/demo/commit/x" {
		t.Fatalf("commit = %#v", values[0])
	}
}

func TestCommitListErrorResponse(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return commitResponse(http.StatusInternalServerError, `{"error_message":"boom"}`), nil
			})}, nil
		},
	}

	cmd := newCmdCommitList(factory)
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "Internal Server Error") {
		t.Fatalf("error = %v, want API error", err)
	}
}

func TestCommitViewOutputsDetail(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/commits/abcdef1234567890abcdef1234567890abcdef12" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("Authorization = %q", got)
				}
				return commitResponse(http.StatusOK, `{
					"sha":"abcdef1234567890abcdef1234567890abcdef12",
					"html_url":"https://atomgit.com/alice/demo/commit/abcdef1234567890abcdef1234567890abcdef12",
					"commit":{"message":"Fix bug\nbody text","author":{"name":"Alice","email":"alice@example.com","date":"2026-07-15T10:00:00Z"}},
					"author":{"login":"alice"},
					"parents":[{"sha":"1234567fedcba9876543210fedcba9876543210fe"}]
				}`), nil
			})}, nil
		},
	}

	cmd := newCmdCommitView(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "abcdef1234567890abcdef1234567890abcdef12"}); err != nil {
		t.Fatal(err)
	}
	want := "SHA: abcdef1234567890abcdef1234567890abcdef12\n" +
		"Title: Fix bug\n" +
		"Author: alice\n" +
		"Date: 2026-07-15T10:00:00Z\n" +
		"URL: https://atomgit.com/alice/demo/commit/abcdef1234567890abcdef1234567890abcdef12\n" +
		"Parents: 1234567\n" +
		"\nFix bug\nbody text\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestCommitViewEscapesSHA(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if got := req.URL.EscapedPath(); got != "/api/v5/repos/alice/demo/commits/feature%2Ffoo" {
					t.Fatalf("escaped path = %q", got)
				}
				return commitResponse(http.StatusOK, `{"sha":"feature/foo","commit":{"message":"m","author":{"date":""}}}`), nil
			})}, nil
		},
	}

	cmd := newCmdCommitView(factory)
	if err := cmd.RunE(cmd, []string{"alice/demo", "feature/foo"}); err != nil {
		t.Fatal(err)
	}
}

func TestCommitViewJSON(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return commitResponse(http.StatusOK, `{
					"sha":"abcdef1234567890abcdef1234567890abcdef12",
					"html_url":"https://atomgit.com/alice/demo/commit/x",
					"commit":{"message":"Fix bug\nbody","author":{"name":"Alice","date":"2026-07-15T10:00:00Z"}},
					"author":{"login":"alice"}
				}`), nil
			})}, nil
		},
	}

	cmd := newCmdCommitView(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "abcdef1234567890abcdef1234567890abcdef12"}); err != nil {
		t.Fatal(err)
	}
	var value struct {
		SHA     string `json:"sha"`
		Title   string `json:"title"`
		Message string `json:"message"`
		Author  string `json:"author"`
	}
	if err := json.Unmarshal(output.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.SHA != "abcdef1234567890abcdef1234567890abcdef12" ||
		value.Title != "Fix bug" ||
		value.Message != "Fix bug\nbody" ||
		value.Author != "alice" {
		t.Fatalf("commit = %#v", value)
	}
}

func TestCommitViewWebFlag(t *testing.T) {
	var capturedURL string
	factory := &cmdutil.Factory{
		Config:        &commitTestConfig{},
		BrowserOpener: func(rawURL string) error { capturedURL = rawURL; return nil },
	}
	cmd := newCmdCommitView(factory)
	_ = cmd.Flags().Set("web", "true")
	if err := cmd.RunE(cmd, []string{"alice/demo", "abcdef1234567890abcdef1234567890abcdef12"}); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/commit/abcdef1234567890abcdef1234567890abcdef12" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func TestCommitViewRejectsBlankSHA(t *testing.T) {
	tests := []struct {
		name string
		sha  string
		web  bool
	}{
		{name: "empty sha", sha: ""},
		{name: "whitespace sha", sha: "   "},
		{name: "empty sha web", sha: "", web: true},
		{name: "whitespace sha web", sha: "   ", web: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			browserCalls := 0
			factory := &cmdutil.Factory{
				Config: &commitTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						return commitResponse(http.StatusOK, `{}`), nil
					})}, nil
				},
				BrowserOpener: func(string) error { browserCalls++; return nil },
			}
			cmd := newCmdCommitView(factory)
			if tt.web {
				_ = cmd.Flags().Set("web", "true")
			}
			err := cmd.RunE(cmd, []string{"alice/demo", tt.sha})
			if err == nil || !strings.Contains(err.Error(), "commit SHA is required") {
				t.Fatalf("error = %v, want error containing %q", err, "commit SHA is required")
			}
			if requests != 0 {
				t.Fatalf("HTTP requests = %d, want 0", requests)
			}
			if browserCalls != 0 {
				t.Fatalf("browser calls = %d, want 0", browserCalls)
			}
		})
	}
}

func TestCommitViewErrorResponse(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return commitResponse(http.StatusNotFound, `{"error_message":"not found"}`), nil
			})}, nil
		},
	}

	cmd := newCmdCommitView(factory)
	if err := cmd.RunE(cmd, []string{"alice/demo", "deadbeef"}); err == nil || !strings.Contains(err.Error(), "Not Found") {
		t.Fatalf("error = %v, want API error", err)
	}
}

func TestCommitViewReturnsCanonicalAuthenticationError(t *testing.T) {
	cmd := newCmdCommitView(&cmdutil.Factory{Config: commitAuthErrorConfig{}})
	err := cmd.RunE(cmd, []string{"alice/demo", "deadbeef"})
	if err == nil || err.Error() != "not authenticated: run `ag auth login`" {
		t.Fatalf("error = %v", err)
	}
}

// commitListBody builds a JSON array of count commit summaries for tests.
func commitListBody(count int) string {
	commits := make([]map[string]any, count)
	for i := range commits {
		sha := strings.Repeat("abcdef", 7) + fmt.Sprintf("%07d", i)
		commits[i] = map[string]any{
			"sha":      sha,
			"html_url": "https://atomgit.com/alice/demo/commit/" + sha,
			"commit": map[string]any{
				"message": "commit " + string(rune('a'+i%26)) + "\nbody",
				"author":  map[string]any{"name": "Alice", "date": "2026-07-15T10:00:00Z"},
			},
			"author": map[string]any{"login": "alice"},
		}
	}
	data, err := json.Marshal(commits)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func TestCommitListRejectsInvalidLimitBeforeAuth(t *testing.T) {
	cmd := newCmdCommitList(&cmdutil.Factory{Config: commitAuthErrorConfig{}})
	_ = cmd.Flags().Set("limit", "0")
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("error = %v, want limit validation to run before authentication", err)
	}
	if strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error = %v, authentication must not run before limit validation", err)
	}
}

func TestEscapeCell(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: "plain text", want: "plain text"},
		{name: "backslash", input: `a\b`, want: `a\\b`},
		{name: "tab", input: "a\tb", want: `a\tb`},
		{name: "newline", input: "a\nb", want: `a\nb`},
		{name: "carriage return", input: "a\rb", want: `a\rb`},
		{name: "mixed", input: "a\tb\nc\\d\re", want: `a\tb\nc\\d\re`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeCell(tt.input); got != tt.want {
				t.Fatalf("escapeCell(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCommitListEscapesTextColumns(t *testing.T) {
	// The title contains a tab and a CRLF. After escaping, the tab-separated
	// row must still have exactly the four structural separators, even when the
	// output goes through the root command's sanitizing writer.
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return commitResponse(http.StatusOK, `[
					{"sha":"abcdef1234567890abcdef1234567890abcdef12","html_url":"https://atomgit.com/alice/demo/commit/x","commit":{"message":"Fix\tbug\r\nline","author":{"name":"Alice","date":"2026-07-15T10:00:00Z"}},"author":{"login":"alice"}}
				]`), nil
			})}, nil
		},
	}

	cmd := newCmdCommitList(factory)
	var output bytes.Buffer
	cmd.SetOut(cmdutil.NewSanitizingWriter(&output))
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	want := "abcdef1\tFix\\tbug\talice\t2026-07-15T10:00:00Z\thttps://atomgit.com/alice/demo/commit/x\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if got := strings.Count(output.String(), "\t"); got != 4 {
		t.Fatalf("structural tab count = %d, want 4", got)
	}
}

func TestCommitAuthorFallback(t *testing.T) {
	tests := []struct {
		name   string
		commit api.Commit
		want   string
	}{
		{
			name:   "account login",
			commit: api.Commit{Author: api.CommitAccount{Login: "alice"}},
			want:   "alice",
		},
		{
			name:   "account name",
			commit: api.Commit{Author: api.CommitAccount{Name: "Alice"}},
			want:   "Alice",
		},
		{
			name:   "commit author name",
			commit: api.Commit{Commit: api.CommitDetail{Author: api.CommitPerson{Name: "Alice"}}},
			want:   "Alice",
		},
		{
			name:   "commit author email",
			commit: api.Commit{Commit: api.CommitDetail{Author: api.CommitPerson{Email: "alice@example.com"}}},
			want:   "alice@example.com",
		},
		{
			name:   "account email",
			commit: api.Commit{Author: api.CommitAccount{Email: "alice@example.com"}},
			want:   "alice@example.com",
		},
		{
			name:   "login preferred over account name",
			commit: api.Commit{Author: api.CommitAccount{Login: "alice", Name: "Alice"}},
			want:   "alice",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commitAuthor(tt.commit); got != tt.want {
				t.Fatalf("commitAuthor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCommitListAuthorEmailFallback(t *testing.T) {
	// Neither the account nor the nested commit author has a login or name, so
	// the author column and the JSON author field must fall back to the email.
	factory := &cmdutil.Factory{
		Config: &commitTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return commitResponse(http.StatusOK, `[
					{"sha":"abcdef1234567890abcdef1234567890abcdef12","html_url":"https://atomgit.com/alice/demo/commit/x","commit":{"message":"Fix bug","author":{"name":"","email":"bot@example.com","date":"2026-07-15T10:00:00Z"}},"author":{"login":"","name":""}}
				]`), nil
			})}, nil
		},
	}

	// Text output falls back to the commit author email.
	var text bytes.Buffer
	cmd := newCmdCommitList(factory)
	cmd.SetOut(&text)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	want := "abcdef1\tFix bug\tbot@example.com\t2026-07-15T10:00:00Z\thttps://atomgit.com/alice/demo/commit/x\n"
	if got := text.String(); got != want {
		t.Fatalf("text output = %q, want %q", got, want)
	}

	// JSON output uses the same author fallback.
	var jsonOut bytes.Buffer
	cmd = newCmdCommitList(factory)
	_ = cmd.Flags().Set("json", "true")
	cmd.SetOut(&jsonOut)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	var values []struct {
		Author string `json:"author"`
	}
	if err := json.Unmarshal(jsonOut.Bytes(), &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].Author != "bot@example.com" {
		t.Fatalf("json author = %#v, want bot@example.com", values)
	}
}

func TestParseComparison(t *testing.T) {
	tests := []struct {
		value     string
		wantBase  string
		wantHead  string
		wantError string
	}{
		{value: "main...feature/one", wantBase: "main", wantHead: "feature/one"},
		{value: "\u00a0topic...main", wantBase: "\u00a0topic", wantHead: "main"},
		{value: "main..feature", wantError: "form <base>...<head>"},
		{value: "main...feature...next", wantError: "form <base>...<head>"},
		{value: "...feature", wantError: "base ref cannot be empty"},
		{value: "main...", wantError: "head ref cannot be empty"},
		{value: "main...\nfeature", wantError: "invalid head ref"},
		{value: "main...feature/../secret", wantError: "invalid head ref"},
		{value: "main...feature//one", wantError: "invalid head ref"},
		{value: "main....feature", wantError: "invalid head ref"},
		{value: "feature..old...main", wantError: "invalid base ref"},
		{value: "name@{1}...main", wantError: "invalid base ref"},
		{value: "release.lock...main", wantError: "invalid base ref"},
		{value: "bad ref...main", wantError: "invalid base ref"},
		{value: "x?y...main", wantError: "invalid base ref"},
		{value: "foo/.bar...main", wantError: "invalid base ref"},
		{value: "@...main", wantError: "invalid base ref"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			base, head, err := parseComparison(tt.value)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if base != tt.wantBase || head != tt.wantHead {
				t.Fatalf("got %q...%q, want %q...%q", base, head, tt.wantBase, tt.wantHead)
			}
		})
	}
}

func TestCompareValidationBeforeAuthentication(t *testing.T) {
	config := &commitTestConfig{tokenErr: errors.New("missing token")}
	requests := 0
	f := factory(config, commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return response(http.StatusOK, `{}`), nil
	}))

	cmd := newCmdCompare(f)
	err := cmd.RunE(cmd, []string{"alice/demo", "main..feature"})
	if err == nil || !strings.Contains(err.Error(), "form <base>...<head>") {
		t.Fatalf("error = %v", err)
	}
	if config.tokenCalls != 0 || requests != 0 {
		t.Fatalf("token calls = %d, requests = %d", config.tokenCalls, requests)
	}
}

func TestMalformedFourDotComparisonFailsBeforeAuthentication(t *testing.T) {
	config := &commitTestConfig{tokenErr: errors.New("missing token")}
	requests := 0
	f := factory(config, commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return response(http.StatusOK, `{}`), nil
	}))

	cmd := newCmdCompare(f)
	err := cmd.RunE(cmd, []string{"alice/demo", "main....feature"})
	if err == nil || !strings.Contains(err.Error(), "invalid head ref") {
		t.Fatalf("error = %v", err)
	}
	if config.tokenCalls != 0 || requests != 0 {
		t.Fatalf("token calls = %d, requests = %d", config.tokenCalls, requests)
	}
}

func TestCommitTextValidationBeforeAuthentication(t *testing.T) {
	config := &commitTestConfig{tokenErr: errors.New("missing token")}
	cmd := newCmdCommitText(factory(config, nil), "diff")
	err := cmd.RunE(cmd, []string{"alice/demo", "\n"})
	if err == nil || !strings.Contains(err.Error(), "invalid commit SHA") {
		t.Fatalf("error = %v", err)
	}
	if config.tokenCalls != 0 {
		t.Fatalf("token calls = %d", config.tokenCalls)
	}
}

func TestCompareTextOutput(t *testing.T) {
	body := `{
  "base_commit":{"sha":"111111111111"},
  "merge_base_commit":{"sha":"222222222222"},
  "commits":[
    {"sha":"aaaaaaaaaaaa","commit":{"message":"feat: first\n\nbody","author":{"name":"Alice","email":"alice@example.com","date":"2026-08-13T00:00:00Z"}},"author":{"login":"alice"}},
    {"sha":"bbbbbbbbbbbb","commit":{"message":"fix: second","author":{"name":"Bob"}},"author":{}}
  ],
  "files":[
    {"sha":"f1","filename":"main.go","status":"modified","additions":3,"deletions":1,"changes":4,"patch":"@@ -1 +1 @@","truncated":false},
    {"sha":"f2","filename":"logo.png","status":"added","additions":0,"deletions":0,"changes":0,"patch":"Binary files /dev/null and b/logo.png differ\n","truncated":true}
  ],
  "truncated":false
}`
	f := factory(&commitTestConfig{}, commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.EscapedPath() != "/api/v5/repos/alice/demo/compare/main...feature%2Fone" {
			t.Fatalf("path = %q", req.URL.EscapedPath())
		}
		return response(http.StatusOK, body), nil
	}))

	cmd := newCmdCompare(f)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "main...feature/one"}); err != nil {
		t.Fatal(err)
	}
	want := "Comparison: main...feature/one\n" +
		"Base: 111111111111\n" +
		"Merge base: 222222222222\n" +
		"Commits: 2\n" +
		"Files: 2\n" +
		"Truncated: false\n\n" +
		"Commits:\n" +
		"aaaaaaa\tfeat: first\talice\n" +
		"bbbbbbb\tfix: second\tBob\n\n" +
		"Files:\n" +
		"modified\t+3\t-1\tmain.go\t-\n" +
		"added\t+0\t-0\tlogo.png\tbinary,truncated\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestCompareTextOutputEscapesFields(t *testing.T) {
	comparison := comparisonJSON{
		Base:    "main\tbase",
		Head:    "feature\nhead",
		Commits: []comparisonCommit{{SHA: "abcdef123", Message: "subject\tcolumn", Author: "alice\tadmin"}},
		Files:   []comparisonFile{{Status: "modified", Filename: "safe.go\nmodified\t+0\t-0\tspoofed.go"}},
	}
	var out bytes.Buffer
	if err := writeComparison(&out, comparison); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "safe.go\nmodified") {
		t.Fatalf("filename introduced an extra row: %q", out.String())
	}
	for _, want := range []string{`Comparison: main\tbase...feature\nhead`, `subject\tcolumn`, `alice\tadmin`, `safe.go\nmodified\t+0\t-0\tspoofed.go`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want %q", out.String(), want)
		}
	}
}

func TestAuthenticatedClientAvoidsDuplicatePrefix(t *testing.T) {
	config := &commitTestConfig{tokenErr: errors.New("not authenticated: run `ag auth login`")}
	_, err := authenticatedClient(factory(config, nil))
	if err == nil || err.Error() != "not authenticated: run `ag auth login`" {
		t.Fatalf("error = %v", err)
	}
}

func TestCompareJSONOutput(t *testing.T) {
	body := `{
  "base_commit":{"sha":"base"},
  "merge_base_commit":{"sha":"merge"},
  "commits":[{"sha":"abc1234567890abc1234567890abc1234567890a","commit":{"message":"message","author":{"name":"Alice","email":"a@example.com","date":"today"},"committer":{"name":"Alice","email":"a@example.com","date":"today"}},"author":{"name":"Alice","id":1,"login":"alice"},"committer":{"name":"Alice","id":1,"login":"alice"}}],
  "files":[{"sha":"file","filename":"image.png","status":"removed","additions":0,"deletions":0,"changes":0,"blob_url":"blob","raw_url":"raw","patch":"Binary files a/image.png and /dev/null differ\n","truncated":0}],
  "truncated":1
}`
	f := factory(&commitTestConfig{}, commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, body), nil
	}))
	cmd := newCmdCompare(f)
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "main...feature"}); err != nil {
		t.Fatal(err)
	}

	var result comparisonJSON
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Base != "main" || result.Head != "feature" || !result.Truncated {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Commits) != 1 || result.Commits[0].Author != "alice" || result.Commits[0].URL != "https://atomgit.com/alice/demo/commit/abc1234567890abc1234567890abc1234567890a" {
		t.Fatalf("commits = %#v", result.Commits)
	}
	if len(result.Files) != 1 || !result.Files[0].Binary || result.Files[0].Truncated {
		t.Fatalf("files = %#v", result.Files)
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Fatalf("JSON output does not end with newline: %q", out.String())
	}
}

func TestCompareEmptyOutput(t *testing.T) {
	f := factory(&commitTestConfig{}, commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, `{"base_commit":{"sha":"same"},"merge_base_commit":{"sha":"same"},"files":[],"truncated":false}`), nil
	}))

	t.Run("text", func(t *testing.T) {
		cmd := newCmdCompare(f)
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := cmd.RunE(cmd, []string{"alice/demo", "main...main"}); err != nil {
			t.Fatal(err)
		}
		want := "Comparison: main...main\nBase: same\nMerge base: same\nCommits: 0\nFiles: 0\nTruncated: false\n"
		if out.String() != want {
			t.Fatalf("output = %q, want %q", out.String(), want)
		}
	})

	t.Run("json", func(t *testing.T) {
		cmd := newCmdCompare(f)
		_ = cmd.Flags().Set("json", "true")
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := cmd.RunE(cmd, []string{"alice/demo", "main...main"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), `"commits": []`) || !strings.Contains(out.String(), `"files": []`) {
			t.Fatalf("output = %s", out.String())
		}
	})
}

func TestCommitTextPreservesPayloadAndErrors(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		status    int
		body      string
		wantOut   string
		wantError string
	}{
		{name: "diff", format: "diff", status: http.StatusOK, body: "diff --git a/a b/a\n-old\n+new\n", wantOut: "diff --git a/a b/a\n-old\n+new\n"},
		{name: "patch", format: "patch", status: http.StatusOK, body: "From abc Mon Sep 17 00:00:00 2001\n\nsubject\n", wantOut: "From abc Mon Sep 17 00:00:00 2001\n\nsubject\n"},
		{name: "not found", format: "diff", status: http.StatusNotFound, body: `{"message":"not found"}`, wantError: "API error: 404 Not Found"},
		{name: "server error", format: "patch", status: http.StatusInternalServerError, body: "failed", wantError: "API error: 500 Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := factory(&commitTestConfig{}, commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				wantPath := "/api/v5/repos/alice/demo/commit/feature%2Fone/" + tt.format
				if req.URL.EscapedPath() != wantPath {
					t.Fatalf("path = %q, want %q", req.URL.EscapedPath(), wantPath)
				}
				return response(tt.status, tt.body), nil
			}))
			cmd := newCmdCommitText(f, tt.format)
			var out bytes.Buffer
			cmd.SetOut(&out)
			err := cmd.RunE(cmd, []string{"alice/demo", "feature/one"})
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want %q", err, tt.wantError)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.wantOut {
				t.Fatalf("output = %q, want %q", out.String(), tt.wantOut)
			}
		})
	}
}

func TestCompareInfersRepository(t *testing.T) {
	f := factory(&commitTestConfig{}, commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.EscapedPath() != "/api/v5/repos/alice/demo/compare/main...feature" {
			t.Fatalf("path = %q", req.URL.EscapedPath())
		}
		return response(http.StatusOK, `{"commits":[],"files":[]}`), nil
	}))
	f.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
	}

	cmd := newCmdCompare(f)
	cmd.SetOut(io.Discard)
	if err := cmd.RunE(cmd, []string{"main...feature"}); err != nil {
		t.Fatal(err)
	}
}

func TestCommitTextPropagatesWriterError(t *testing.T) {
	want := errors.New("write failed")
	f := factory(&commitTestConfig{}, commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, "diff"), nil
	}))
	cmd := newCmdCommitText(f, "diff")
	cmd.SetOut(errorWriter{err: want})
	err := cmd.RunE(cmd, []string{"alice/demo", "abc"})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}

func TestCommitTextPropagatesContextCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	f := factory(&commitTestConfig{}, commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		close(requestStarted)
		<-req.Context().Done()
		return nil, req.Context().Err()
	}))
	cmd := newCmdCommitText(f, "diff")
	ctx, cancel := context.WithCancel(context.Background())
	cmd.SetContext(ctx)

	done := make(chan error, 1)
	go func() {
		done <- cmd.RunE(cmd, []string{"alice/demo", "abc"})
	}()
	<-requestStarted
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestComparePropagatesContextCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	f := factory(&commitTestConfig{}, commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		close(requestStarted)
		<-req.Context().Done()
		return nil, req.Context().Err()
	}))
	cmd := newCmdCompare(f)
	ctx, cancel := context.WithCancel(context.Background())
	cmd.SetContext(ctx)

	done := make(chan error, 1)
	go func() {
		done <- cmd.RunE(cmd, []string{"alice/demo", "main...feature"})
	}()
	<-requestStarted
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestCommitTextCopyErrorHasNeutralContext(t *testing.T) {
	want := errors.New("unexpected EOF")
	f := factory(&commitTestConfig{}, commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(errorReader{err: want}),
		}, nil
	}))
	cmd := newCmdCommitText(f, "diff")
	cmd.SetOut(io.Discard)
	err := cmd.RunE(cmd, []string{"alice/demo", "abc"})
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "stream commit diff output") || strings.Contains(err.Error(), "write commit diff") {
		t.Fatalf("error = %v", err)
	}
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }
