package search

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type searchTestConfig struct {
	token    string
	tokenErr error
}

func (c searchTestConfig) GetToken() (string, error) {
	if c.token == "" {
		return "token", c.tokenErr
	}
	return c.token, c.tokenErr
}
func (searchTestConfig) GetUser() (string, error) { return "alice", nil }
func (searchTestConfig) GetHost() string          { return "atomgit.com" }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSearchRejectsUnexpectedArguments(t *testing.T) {
	cmd := NewCmdSearch(&cmdutil.Factory{})

	if err := cmd.Args(cmd, []string{"unknown", "query"}); err == nil {
		t.Fatal("search accepted unexpected positional arguments")
	}
}

func TestSearchRepositoriesRegistersReposAlias(t *testing.T) {
	cmd := NewCmdSearch(&cmdutil.Factory{})
	found, args, err := cmd.Find([]string{"repos", "kernel"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := found.Name(), "repositories"; got != want {
		t.Fatalf("resolved command = %q, want %q", got, want)
	}
	if len(args) != 1 || args[0] != "kernel" {
		t.Fatalf("remaining args = %q, want [kernel]", args)
	}
}

func TestSearchRepositoriesUsesEffectivePageSize(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: searchTestConfig{token: "token"},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if got := req.URL.Query().Get("per_page"); got != "20" {
					t.Fatalf("per_page = %q, want 20", got)
				}
				page := req.URL.Query().Get("page")
				if page != "1" && page != "2" {
					t.Fatalf("unexpected page request: %s", page)
				}

				start := 0
				if page == "2" {
					start = 20
				}
				repositories := make([]map[string]any, 20)
				for i := range repositories {
					repositories[i] = map[string]any{"full_name": fmt.Sprintf("owner/repo-%d", start+i)}
				}
				var body strings.Builder
				if err := json.NewEncoder(&body).Encode(repositories); err != nil {
					t.Fatal(err)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body.String())), Header: make(http.Header)}, nil
			})}, nil
		},
	}

	cmd := newCmdSearchRepositories(factory)
	_ = cmd.Flags().Set("limit", "30")
	_ = cmd.Flags().Set("json", "true")
	var out strings.Builder
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"repo"}); err != nil {
		t.Fatal(err)
	}

	var repositories []map[string]any
	if err := json.Unmarshal([]byte(out.String()), &repositories); err != nil {
		t.Fatal(err)
	}
	if len(repositories) != 30 {
		t.Fatalf("len(repositories) = %d, want 30", len(repositories))
	}
	if got := repositories[20]["full_name"]; got != "owner/repo-20" {
		t.Fatalf("second page first repository = %v, want owner/repo-20", got)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestSearchRepositoriesNormalizesDescriptionWhitespace(t *testing.T) {
	const description = "AtomGit CLI\r\ncommand-line\ttool"
	newFactory := func() *cmdutil.Factory {
		return &cmdutil.Factory{
			Config: searchTestConfig{token: "token"},
			HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					body := `[{"full_name":"hust-open-atom-club/atomgit-cli","stargazers_count":1,"description":"AtomGit CLI\r\ncommand-line\ttool"}]`
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
				})}, nil
			},
		}
	}

	t.Run("human readable output", func(t *testing.T) {
		cmd := newCmdSearchRepositories(newFactory())
		_ = cmd.Flags().Set("limit", "1")
		var out strings.Builder
		cmd.SetOut(&out)
		if err := cmd.RunE(cmd, []string{"atomgit-cli"}); err != nil {
			t.Fatal(err)
		}
		want := "hust-open-atom-club/atomgit-cli\t★1\tAtomGit CLI command-line tool\n"
		if got := out.String(); got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})

	t.Run("JSON preserves original description", func(t *testing.T) {
		cmd := newCmdSearchRepositories(newFactory())
		_ = cmd.Flags().Set("limit", "1")
		_ = cmd.Flags().Set("json", "true")
		var out strings.Builder
		cmd.SetOut(&out)
		if err := cmd.RunE(cmd, []string{"atomgit-cli"}); err != nil {
			t.Fatal(err)
		}
		var repositories []struct {
			Description string `json:"description"`
		}
		if err := json.Unmarshal([]byte(out.String()), &repositories); err != nil {
			t.Fatal(err)
		}
		if len(repositories) != 1 || repositories[0].Description != description {
			t.Fatalf("repositories = %#v, want original description %q", repositories, description)
		}
	})
}

func TestSearchUsersHonorsQueryAndLimit(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: searchTestConfig{token: "token"},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("Authorization = %q, want %q", got, "Bearer token")
				}
				if req.URL.Path != "/api/v5/search/users" {
					t.Fatalf("path = %s", req.URL.Path)
				}
				if got := req.URL.Query().Get("q"); got != "c++ dev" {
					t.Fatalf("q = %q, want %q", got, "c++ dev")
				}
				if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "50" {
					t.Fatalf("pagination = %s", req.URL.RawQuery)
				}
				body := `[{"login":"a","name":"A","html_url":"https://atomgit.com/a"},{"login":"b","name":"B","html_url":"https://atomgit.com/b"}]`
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}, nil
		},
	}
	cmd := newCmdSearchUsers(factory)
	_ = cmd.Flags().Set("limit", "1")
	var out strings.Builder
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"c++ dev"}); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "a\tA\thttps://atomgit.com/a\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestSearchUsersEncodesSupportedSorting(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: searchTestConfig{token: "token"},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				query := req.URL.Query()
				if got := query.Get("q"); got != "c++ dev" {
					t.Fatalf("q = %q, want c++ dev", got)
				}
				if got := query.Get("sort"); got != "joined_at" {
					t.Fatalf("sort = %q, want joined_at", got)
				}
				if got := query.Get("order"); got != "asc" {
					t.Fatalf("order = %q, want asc", got)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`)), Header: make(http.Header)}, nil
			})}, nil
		},
	}

	cmd := newCmdSearchUsers(factory)
	_ = cmd.Flags().Set("sort", "joined_at")
	_ = cmd.Flags().Set("order", "asc")
	if err := cmd.RunE(cmd, []string{"c++ dev"}); err != nil {
		t.Fatal(err)
	}
}

func TestSearchRepositoriesEncodesSupportedFilters(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: searchTestConfig{token: "token"},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				query := req.URL.Query()
				want := map[string]string{
					"q":        "cli tools",
					"sort":     "stars_count",
					"order":    "desc",
					"owner":    "hust open/atom",
					"fork":     "true",
					"language": "C++",
				}
				for name, value := range want {
					if got := query.Get(name); got != value {
						t.Fatalf("%s = %q, want %q (raw query: %s)", name, got, value, req.URL.RawQuery)
					}
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`)), Header: make(http.Header)}, nil
			})}, nil
		},
	}

	cmd := newCmdSearchRepositories(factory)
	_ = cmd.Flags().Set("sort", "stars_count")
	_ = cmd.Flags().Set("order", "desc")
	_ = cmd.Flags().Set("owner", "hust open/atom")
	_ = cmd.Flags().Set("fork", "true")
	_ = cmd.Flags().Set("language", "C++")
	if err := cmd.RunE(cmd, []string{"cli tools"}); err != nil {
		t.Fatal(err)
	}
}

func TestSearchCommandsRequireAuthentication(t *testing.T) {
	tests := []struct {
		name string
		new  func(*cmdutil.Factory) *cobra.Command
	}{
		{name: "users", new: newCmdSearchUsers},
		{name: "repositories", new: newCmdSearchRepositories},
		{name: "issues", new: newCmdSearchIssues},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: searchTestConfig{tokenErr: errors.New("token unavailable")},
				HttpClient: func() (*http.Client, error) {
					t.Fatal("HTTP client must not be created when authentication fails")
					return nil, nil
				},
			}

			cmd := tt.new(factory)
			err := cmd.RunE(cmd, []string{"query"})
			if err == nil || !strings.Contains(err.Error(), "not authenticated") {
				t.Fatalf("error = %v, want authentication error", err)
			}
		})
	}
}

func TestSearchCommandsValidateLimitBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name string
		new  func(*cmdutil.Factory) *cobra.Command
	}{
		{name: "users", new: newCmdSearchUsers},
		{name: "repositories", new: newCmdSearchRepositories},
		{name: "issues", new: newCmdSearchIssues},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factory := &cmdutil.Factory{Config: searchTestConfig{tokenErr: errors.New("not authenticated")}}
			cmd := test.new(factory)
			if err := cmd.Flags().Set("limit", "0"); err != nil {
				t.Fatal(err)
			}
			err := cmd.RunE(cmd, []string{"query"})
			if err == nil || !strings.Contains(err.Error(), "invalid limit: 0") {
				t.Fatalf("error = %v, want invalid limit before authentication", err)
			}
		})
	}
}

func TestSearchIssuesFormatsRepositoryNumberAndTitle(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: searchTestConfig{token: "token"},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				body := `[{"number":7,"title":"Fix memory leak","state":"open","repository":{"full_name":"alice/demo"}}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			})}, nil
		},
	}

	cmd := newCmdSearchIssues(factory)
	_ = cmd.Flags().Set("limit", "1")
	var out strings.Builder
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"memory leak"}); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "alice/demo\t#7 [open]\tFix memory leak\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestSearchIssuesContinuesAfterShortPage(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: searchTestConfig{token: "token"},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if got := req.URL.Query().Get("per_page"); got != "50" {
					t.Fatalf("per_page = %q, want 50", got)
				}
				if got := req.URL.Query().Get("page"); got != fmt.Sprint(requests) {
					t.Fatalf("page = %q, want %d", got, requests)
				}

				count := 48
				start := 0
				if requests == 2 {
					count = 44
					start = 48
				} else if requests > 2 {
					t.Fatalf("unexpected request %d", requests)
				}

				issues := make([]map[string]any, count)
				for i := range issues {
					number := start + i + 1
					issues[i] = map[string]any{
						"number": number,
						"title":  fmt.Sprintf("Issue %d", number),
						"state":  "open",
						"repository": map[string]any{
							"full_name": "alice/demo",
						},
					}
				}

				var body strings.Builder
				if err := json.NewEncoder(&body).Encode(issues); err != nil {
					t.Fatal(err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body.String())),
					Header:     make(http.Header),
				}, nil
			})}, nil
		},
	}

	cmd := newCmdSearchIssues(factory)
	_ = cmd.Flags().Set("limit", "51")
	_ = cmd.Flags().Set("json", "true")
	var out strings.Builder
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"fix"}); err != nil {
		t.Fatal(err)
	}

	var issues []SearchIssue
	if err := json.Unmarshal([]byte(out.String()), &issues); err != nil {
		t.Fatal(err)
	}
	if len(issues) != 51 {
		t.Fatalf("len(issues) = %d, want 51", len(issues))
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if got := issues[48].GetNumber(); got != "49" {
		t.Fatalf("issues[48].number = %q, want first result from second page", got)
	}
}

func TestSearchIssuesDeduplicatesOverlappingPages(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: searchTestConfig{token: "token"},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				var numbers []int
				switch requests {
				case 1:
					numbers = []int{1, 2, 3}
				case 2:
					numbers = []int{3, 4}
				case 3:
					numbers = []int{4, 5}
				default:
					t.Fatalf("unexpected request %d", requests)
				}

				issues := make([]map[string]any, len(numbers))
				for index, number := range numbers {
					issues[index] = map[string]any{
						"id":     number + 1000,
						"number": number,
						"title":  fmt.Sprintf("Issue %d", number),
						"repository": map[string]any{
							"full_name": "alice/demo",
						},
					}
				}
				var body strings.Builder
				if err := json.NewEncoder(&body).Encode(issues); err != nil {
					t.Fatal(err)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body.String())), Header: make(http.Header)}, nil
			})}, nil
		},
	}

	cmd := newCmdSearchIssues(factory)
	_ = cmd.Flags().Set("limit", "5")
	_ = cmd.Flags().Set("json", "true")
	var out strings.Builder
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"test"}); err != nil {
		t.Fatal(err)
	}

	var issues []SearchIssue
	if err := json.Unmarshal([]byte(out.String()), &issues); err != nil {
		t.Fatal(err)
	}
	if len(issues) != 5 {
		t.Fatalf("len(issues) = %d, want 5", len(issues))
	}
	for index, issue := range issues {
		if got, want := issue.GetNumber(), fmt.Sprint(index+1); got != want {
			t.Fatalf("issues[%d].number = %q, want %q", index, got, want)
		}
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
}

func TestSearchIssuesEncodesSupportedFilters(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: searchTestConfig{token: "token"},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				query := req.URL.Query()
				want := map[string]string{
					"q":     "memory leak",
					"sort":  "created_at",
					"order": "asc",
					"repo":  "alice/demo repo",
					"state": "closed",
				}
				for name, value := range want {
					if got := query.Get(name); got != value {
						t.Fatalf("%s = %q, want %q (raw query: %s)", name, got, value, req.URL.RawQuery)
					}
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`)), Header: make(http.Header)}, nil
			})}, nil
		},
	}

	cmd := newCmdSearchIssues(factory)
	_ = cmd.Flags().Set("sort", "created_at")
	_ = cmd.Flags().Set("order", "asc")
	_ = cmd.Flags().Set("repo", "alice/demo repo")
	_ = cmd.Flags().Set("state", "closed")
	if err := cmd.RunE(cmd, []string{"memory leak"}); err != nil {
		t.Fatal(err)
	}
}

func TestSearchRejectsUnsupportedFilterValues(t *testing.T) {
	tests := []struct {
		name  string
		new   func(*cmdutil.Factory) *cobra.Command
		flag  string
		value string
	}{
		{name: "users sort", new: newCmdSearchUsers, flag: "sort", value: "stars"},
		{name: "repositories sort", new: newCmdSearchRepositories, flag: "sort", value: "created_at"},
		{name: "issues state", new: newCmdSearchIssues, flag: "state", value: "all"},
		{name: "order", new: newCmdSearchIssues, flag: "order", value: "newest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: searchTestConfig{token: "token"},
				HttpClient: func() (*http.Client, error) {
					t.Fatal("HTTP client must not be created for an invalid filter")
					return nil, nil
				},
			}
			cmd := tt.new(factory)
			_ = cmd.Flags().Set(tt.flag, tt.value)
			if err := cmd.RunE(cmd, []string{"query"}); err == nil || !strings.Contains(err.Error(), "invalid "+tt.flag) {
				t.Fatalf("error = %v, want invalid %s error", err, tt.flag)
			}
		})
	}
}

func TestSearchIssuesRequiresQuery(t *testing.T) {
	cmd := newCmdSearchIssues(&cmdutil.Factory{})
	if err := cmd.Args(cmd, nil); err == nil {
		t.Fatal("search issues accepted an empty query")
	}
}
