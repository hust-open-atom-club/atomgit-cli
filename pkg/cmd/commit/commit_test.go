package commit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type commitTestConfig struct{}

func (commitTestConfig) GetToken() (string, error) { return "token", nil }
func (commitTestConfig) GetUser() (string, error)  { return "alice", nil }
func (commitTestConfig) GetHost() string           { return "atomgit.com" }

type commitAuthErrorConfig struct{ commitTestConfig }

func (commitAuthErrorConfig) GetToken() (string, error) {
	return "", errors.New("not authenticated: run `ag auth login`")
}

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

func TestNewCmdCommitRegistersSubcommands(t *testing.T) {
	cmd := NewCmdCommit(&cmdutil.Factory{})
	want := map[string]bool{"list": false, "view": false}
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
}

func TestCommitListInfersRepositoryAndForwardsFilters(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: commitTestConfig{},
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
		Config: commitTestConfig{},
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
		Config: commitTestConfig{},
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
			cmd := newCmdCommitList(&cmdutil.Factory{Config: commitTestConfig{}})
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
		Config: commitTestConfig{},
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
		Config: commitTestConfig{},
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
		Config: commitTestConfig{},
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
		Config: commitTestConfig{},
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
		Config: commitTestConfig{},
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
		Config: commitTestConfig{},
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
		Config: commitTestConfig{},
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
		Config:        commitTestConfig{},
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

func TestCommitViewErrorResponse(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: commitTestConfig{},
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
		Config: commitTestConfig{},
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
