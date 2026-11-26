package search

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestUnifiedSearchCommands(t *testing.T) {
	const keyword = "C++ & 中文/#?"
	tests := []struct {
		name     string
		args     []string
		kind     string
		repoType string
		result   string
	}{
		{
			name: "pull requests", args: []string{"prs", keyword}, kind: "pullrequest",
			result: `{"id":"42","iid":17,"title":"Fix parser","source_branch":"fix/parser","author":{"id":123,"username":"alice","name":null,"avatar_url":null,"web_url":null,"nick_name":null},"state":"opened"}`,
		},
		{
			name: "organizations", args: []string{"orgs", keyword}, kind: "group",
			result: `{"id":"ab42","full_path":"example","full_name":"Example Org","project_num":0,"watch_num":null}`,
		},
		{
			name: "users", args: []string{"by-type", "USER", keyword}, kind: "user",
			result: `{"id":"ab123","username":"alice","nickname":"Alice","location":null,"updated_at":"2026-11-08T12:34:56+08:00"}`,
		},
		{
			name: "issues", args: []string{"by-type", "issue", keyword}, kind: "issue",
			result: `{"iid":7,"state":"opened","path_with_namespace":"org/project","labels":["bug"],"author":null}`,
		},
		{
			name: "repositories without a filter", args: []string{"by-type", "repo", keyword}, kind: "repo",
			result: `{"id":"4","repo_type":"repo","web_url":"https://atomgit.com/org/tools","star_count":12,"archived":false}`,
		},
		{
			name: "code repositories", args: []string{"by-type", "repo", keyword, "--repo-type", "code"}, kind: "repo", repoType: "0",
			result: `{"path_with_namespace":"org/tools","star_count":8,"repo_type":"code"}`,
		},
		{
			name: "model repositories", args: []string{"repos", keyword, "--repo-type", "model"}, kind: "repo", repoType: "1",
			result: `{"path_with_namespace":"org/model","model_param_size":"7B","task":[{"code":"chat","name":"Chat"}]}`,
		},
		{
			name: "dataset repositories", args: []string{"repositories", keyword, "--repo-type", "dataset"}, kind: "repo", repoType: "2",
			result: `{"path_with_namespace":"org/data","repo_type":"dataset","license":{"key":"mit","name":"MIT","nickname":null,"html_url":null,"source_url":null}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			result := fmt.Sprintf(`{"page_num":1,"page_size":5,"total":null,"page_count":1,"content":[%s]}`, tt.result)
			args := append(tt.args, "--limit", "5", "--json", "--sort", "updated_at", "--order", "desc")
			output, err := executeUnifiedSearch(t, searchTestConfig{tokenErr: config.ErrNotAuthenticated}, func(req *http.Request) string {
				requests++
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/search" || req.Header.Get("Authorization") != "" {
					t.Fatalf("expected anonymous GET /api/v5/search, got %s %s with headers %v", req.Method, req.URL.Path, req.Header)
				}
				want := url.Values{"type": {tt.kind}, "q": {keyword}, "p": {"1"}, "pp": {"5"}, "s": {"updated_at"}, "o": {"desc"}}
				if tt.repoType != "" {
					want.Set("repo_type", tt.repoType)
				}
				if got := req.URL.Query(); !reflect.DeepEqual(got, want) {
					t.Fatalf("query params = %v, want %v", got, want)
				}
				return result
			}, args...)
			if err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want 1", requests)
			}

			var got []map[string]any
			if err := json.Unmarshal([]byte(output), &got); err != nil {
				t.Fatal(err)
			}
			var want map[string]any
			if err := json.Unmarshal([]byte(tt.result), &want); err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 {
				t.Fatalf("got %d results, want 1", len(got))
			}
			for key, value := range want {
				if actual, present := got[0][key]; !present || !reflect.DeepEqual(actual, value) {
					t.Errorf("result field %s = %v (present: %t), want %v", key, actual, present, value)
				}
			}
		})
	}
}

func TestModelSearchForwardsFiltersOnEachPage(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		filters url.Values
	}{
		{
			name: "all filters",
			args: []string{
				"repos", "llama", "--repo-type", "model",
				"--task", "text-generation,chat+search", "--library", "transformers,torch&test",
				"--param-min", "1B", "--param-max", "7B",
			},
			filters: url.Values{
				"task": {"text-generation,chat+search"}, "library": {"transformers,torch&test"},
				"param.min": {"1B"}, "param.max": {"7B"},
			},
		},
		{
			name:    "minimum only",
			args:    []string{"by-type", "repo", "llama", "--repo-type", "model", "--param-min", "1.5B"},
			filters: url.Values{"param.min": {"1.5B"}},
		},
		{
			name:    "maximum only",
			args:    []string{"by-type", "repo", "llama", "--repo-type", "model", "--param-max", "7B"},
			filters: url.Values{"param.max": {"7B"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			args := append(tt.args, "--json", "--limit", "2")
			output, err := executeUnifiedSearch(t, searchTestConfig{tokenErr: config.ErrNotAuthenticated}, func(req *http.Request) string {
				requests++
				if req.URL.Path != "/api/v5/search" {
					t.Fatalf("path = %q, want /api/v5/search", req.URL.Path)
				}
				want := url.Values{
					"type": {"repo"}, "q": {"llama"}, "repo_type": {"1"},
					"p": {fmt.Sprint(requests)}, "pp": {"2"},
				}
				maps.Copy(want, tt.filters)
				if got := req.URL.Query(); !reflect.DeepEqual(got, want) {
					t.Fatalf("query params = %v, want %v", got, want)
				}
				return fmt.Sprintf(`{"page_count":2,"content":[{"id":"%d","model_param_size":"3B","repo_type":"model"}]}`, requests)
			}, args...)
			if err != nil {
				t.Fatal(err)
			}
			if requests != 2 {
				t.Fatalf("requests = %d, want 2", requests)
			}
			var results []struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(output), &results); err != nil {
				t.Fatal(err)
			}
			if len(results) != 2 || results[0].ID != "1" || results[1].ID != "2" {
				t.Fatalf("results = %#v, want one result from each page", results)
			}
		})
	}
}

func TestUnifiedSearchPagination(t *testing.T) {
	tests := []struct {
		name      string
		limit     string
		pageSize  string
		pageCount int
		pages     [][]int
		want      []int
	}{
		{
			name: "metadata ends pagination even when total is null", limit: "20", pageSize: "10", pageCount: 3,
			pages: [][]int{{9}, {}, {3, 2}}, want: []int{9, 3, 2},
		},
		{
			name: "limit caps results across pages", limit: "3", pageSize: "3", pageCount: 4,
			pages: [][]int{{9, 8}, {2, 1}}, want: []int{9, 8, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			output, err := executeUnifiedSearch(t, searchTestConfig{token: "mock-token"}, func(req *http.Request) string {
				requests++
				if requests > len(tt.pages) {
					t.Fatalf("unexpected request %d", requests)
				}
				if req.Header.Get("Authorization") != "Bearer mock-token" {
					t.Fatal("expected configured token")
				}
				want := url.Values{"type": {"pullrequest"}, "q": {"cli"}, "p": {fmt.Sprint(requests)}, "pp": {tt.pageSize}}
				if got := req.URL.Query(); !reflect.DeepEqual(got, want) {
					t.Fatalf("query params = %v, want %v", got, want)
				}

				content := make([]map[string]int, 0, len(tt.pages[requests-1]))
				for _, iid := range tt.pages[requests-1] {
					content = append(content, map[string]int{"iid": iid})
				}
				data, err := json.Marshal(content)
				if err != nil {
					t.Fatal(err)
				}
				return fmt.Sprintf(`{"page_count":%d,"total":null,"content":%s}`, tt.pageCount, data)
			}, "prs", "cli", "--limit", tt.limit, "--json")
			if err != nil {
				t.Fatal(err)
			}
			if requests != len(tt.pages) {
				t.Fatalf("requests = %d, want %d", requests, len(tt.pages))
			}
			var items []struct {
				IID int `json:"iid"`
			}
			if err := json.Unmarshal([]byte(output), &items); err != nil {
				t.Fatal(err)
			}
			var got []int
			for _, item := range items {
				got = append(got, item.IID)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got IIDs %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnifiedSearchEmptyResults(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		body string
		want string
	}{
		{
			name: "text", args: []string{"orgs", "missing"},
			body: `{"page_num":0,"page_size":0,"page_count":0,"content":[]}`, want: "No organizations found.\n",
		},
		{
			name: "JSON", args: []string{"prs", "missing", "--json"},
			body: `{"total":null,"page_count":0,"content":null}`, want: "[]\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeUnifiedSearch(t, searchTestConfig{}, func(*http.Request) string {
				return tt.body
			}, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if output != tt.want {
				t.Fatalf("output = %q, want %q", output, tt.want)
			}
		})
	}
}

func TestUnifiedSearchDiscardsResultsWhenLaterPageIsInvalid(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "missing page count", body: `{"content":[]}`},
		{name: "missing content", body: `{"page_count":2}`},
		{name: "negative page count", body: `{"page_count":-1,"content":[]}`},
		{name: "wrong content type", body: `{"page_count":2,"content":{}}`},
		{name: "null content item", body: `{"page_count":2,"content":[null]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeUnifiedSearch(t, searchTestConfig{}, func(req *http.Request) string {
				if req.URL.Query().Get("p") == "1" {
					return `{"page_count":2,"content":[{"iid":17,"title":"first page"}]}`
				}
				return tt.body
			}, "prs", "cli", "--json")
			if err == nil || !strings.Contains(err.Error(), "search pullrequest page 2") || output != "" {
				t.Fatalf("output = %q, error = %v, want no output and a second-page error", output, err)
			}
		})
	}
}

func TestUnifiedSearchEscapesTextFieldsAndKeepsJSONValid(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		body string
		want string
	}{
		{
			name: "pull request", args: []string{"prs", "search"},
			body: `{"page_count":1,"content":[{"path_with_namespace":"org/repo","iid":7,"state":"opened","title":"line\nnext\tcol\rend\u001b[31m\\","url":"https://example.com/pr/7"}]}`,
			want: "org/repo\t!7 [opened]\t" + `line\nnext\tcol\rend\x1b[31m\\` + "\thttps://example.com/pr/7\n",
		},
		{
			name: "organization", args: []string{"orgs", "search"},
			body: `{"page_count":1,"content":[{"full_path":"org","full_name":"a\tb","description":"first\nsecond"}]}`,
			want: "org\ta\\tb\tfirst\\nsecond\t\n",
		},
		{
			name: "repository", args: []string{"repos", "search", "--repo-type", "model"},
			body: `{"page_count":1,"content":[{"path_with_namespace":"org/model","description":"line\tbreak","star_count":0}]}`,
			want: "org/model\t★0\tline\\tbreak\n",
		},
		{
			name: "JSON", args: []string{"prs", "search", "--json"},
			body: `{"page_count":1,"content":[{"title":"line\nnext\t\u007f\u001b"}]}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeUnifiedSearch(t, searchTestConfig{}, func(*http.Request) string {
				return tt.body
			}, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if tt.name == "JSON" {
				var items []struct{ Title string }
				if err := json.Unmarshal([]byte(output), &items); err != nil {
					t.Fatal(err)
				}
				if len(items) != 1 || items[0].Title != "line\nnext\t\x7f\x1b" {
					t.Fatalf("JSON result = %#v, want original title", items)
				}
			} else if output != tt.want {
				t.Fatalf("output = %q, want %q", output, tt.want)
			}
		})
	}
}

func TestUnifiedSearchRejectsInvalidOptions(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{name: "search type", args: []string{"by-type", "wiki", "cli"}, want: "invalid search type"},
		{name: "repository type", args: []string{"repos", "cli", "--repo-type", "commit"}, want: "invalid repo-type"},
		{name: "empty repository type", args: []string{"repos", "cli", "--repo-type="}, want: "invalid repo-type"},
		{name: "multiple repository types", args: []string{"repos", "cli", "--repo-type", "model", "--repo-type", "dataset"}, want: "accepts only one value"},
		{name: "repository type on group", args: []string{"by-type", "group", "cli", "--repo-type", "model"}, want: "only supported for repo searches"},
		{name: "model task without repository type", args: []string{"repos", "llama", "--task", "text-generation"}, want: "require --repo-type model"},
		{name: "model library with legacy filter and no repository type", args: []string{"repos", "llama", "--library", "transformers", "--owner", "org"}, want: "require --repo-type model"},
		{name: "model library on code repository", args: []string{"repos", "llama", "--repo-type", "code", "--library", "transformers"}, want: "require --repo-type model"},
		{name: "model parameter limit on dataset", args: []string{"by-type", "repo", "llama", "--repo-type", "dataset", "--param-max", "7B"}, want: "require --repo-type model"},
		{name: "model filter on group", args: []string{"by-type", "group", "llama", "--param-min", "1B"}, want: "only supported for repo searches"},
		{name: "empty task before legacy routing", args: []string{"repos", "llama", "--task="}, want: "--task must not be empty"},
		{name: "blank model library", args: []string{"repos", "llama", "--repo-type", "model", "--library", "  "}, want: "--library must not be empty"},
		{name: "missing model search query", args: []string{"repos", "--repo-type", "model", "--param-min", "1B"}, want: "received 0"},
		{name: "blank model search query", args: []string{"by-type", "repo", "  ", "--repo-type", "model", "--param-min", "1B"}, want: "search query must not be empty"},
		{name: "owner filter", args: []string{"repos", "cli", "--repo-type", "code", "--owner", "org"}, want: "cannot be combined"},
		{name: "fork filter", args: []string{"repos", "cli", "--repo-type", "code", "--fork"}, want: "cannot be combined"},
		{name: "language filter", args: []string{"repos", "cli", "--repo-type", "code", "--language", "Go"}, want: "cannot be combined"},
		{name: "legacy repository sort", args: []string{"repos", "cli", "--repo-type", "code", "--sort", "last_push_at"}, want: "invalid sort"},
		{name: "order", args: []string{"prs", "cli", "--order", "newer"}, want: "invalid order"},
		{name: "limit", args: []string{"orgs", "cli", "--limit", "0"}, want: "invalid limit"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmdSearch(&cmdutil.Factory{})
			cmd.SetArgs(tt.args)
			cmd.SetErr(io.Discard)
			cmd.SetOut(io.Discard)
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q before accessing configuration", err, tt.want)
			}
		})
	}
}

func executeUnifiedSearch(t *testing.T, cfg searchTestConfig, reply func(*http.Request) string, args ...string) (string, error) {
	t.Helper()
	factory := &cmdutil.Factory{
		Config: cfg,
		HTTPClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(reply(req))),
				}, nil
			})}, nil
		},
	}
	cmd := NewCmdSearch(factory)
	var output strings.Builder
	cmd.SetOut(cmdutil.NewSanitizingWriter(&output))
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return output.String(), err
}
