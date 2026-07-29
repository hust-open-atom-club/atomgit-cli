package pr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type prTestConfig struct{ tokenErr error }

func (c prTestConfig) GetToken() (string, error) { return "token", c.tokenErr }
func (prTestConfig) GetUser() (string, error)    { return "alice", nil }
func (prTestConfig) GetHost() string             { return "atomgit.com" }

type prRoundTripFunc func(*http.Request) (*http.Response, error)

func (f prRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestPRIssueLinksValidateBeforeAuthentication(t *testing.T) {
	factory := &cmdutil.Factory{Config: prTestConfig{tokenErr: config.ErrNotAuthenticated}}
	tests := []struct {
		name    string
		command func(*cmdutil.Factory) *cobra.Command
		args    []string
		issue   string
		want    string
	}{
		{name: "link missing issue", command: newCmdLinkIssues, args: []string{"alice/demo", "1"}, want: "at least one issue number"},
		{name: "link invalid issue", command: newCmdLinkIssues, args: []string{"alice/demo", "1"}, issue: "bad", want: "invalid issue number"},
		{name: "link invalid PR", command: newCmdLinkIssues, args: []string{"alice/demo", "bad"}, issue: "1", want: "invalid PR number"},
		{name: "unlink missing issue", command: newCmdUnlinkIssues, args: []string{"alice/demo", "1"}, want: "at least one issue number"},
		{name: "unlink invalid issue", command: newCmdUnlinkIssues, args: []string{"alice/demo", "1"}, issue: "bad", want: "invalid issue number"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := test.command(factory)
			if test.issue != "" {
				_ = cmd.Flags().Set("issue", test.issue)
			}
			err := cmd.RunE(cmd, test.args)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestResolveBaseBranch(t *testing.T) {
	tests := []struct {
		name       string
		requested  string
		repository api.Repository
		want       string
		wantError  bool
	}{
		{name: "explicit branch", requested: " release ", repository: api.Repository{DefaultBranch: "main"}, want: "release"},
		{name: "main default", repository: api.Repository{DefaultBranch: " main "}, want: "main"},
		{name: "master default", repository: api.Repository{DefaultBranch: "master"}, want: "master"},
		{name: "custom default", repository: api.Repository{DefaultBranch: "stable/1.x"}, want: "stable/1.x"},
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

func TestPRHelpExplainsRepositoryInference(t *testing.T) {
	cmd := NewCmdPR(&cmdutil.Factory{})
	list, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(list.Long, cmdutil.RepositoryContextHelp) {
		t.Fatal("list help does not explain repository inference")
	}
}

func TestPRCreateUsesRequestedOrRepositoryDefaultBase(t *testing.T) {
	tests := []struct {
		name            string
		requestedBase   string
		defaultBranch   string
		wantBase        string
		wantRepoRequest bool
	}{
		{name: "main default", defaultBranch: "main", wantBase: "main", wantRepoRequest: true},
		{name: "master default", defaultBranch: "master", wantBase: "master", wantRepoRequest: true},
		{name: "custom default", defaultBranch: "stable/1.x", wantBase: "stable/1.x", wantRepoRequest: true},
		{name: "explicit base", requestedBase: "release", defaultBranch: "main", wantBase: "release"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: prTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if got := req.Header.Get("Authorization"); got != "Bearer token" {
							t.Fatalf("Authorization = %q", got)
						}

						if req.Method == http.MethodGet {
							if !tt.wantRepoRequest {
								t.Fatal("explicit --base unexpectedly requested repository metadata")
							}
							if req.URL.Path != "/api/v5/repos/alice/demo" {
								t.Fatalf("repository request path = %s", req.URL.Path)
							}
							body := fmt.Sprintf(`{"default_branch":%q}`, tt.defaultBranch)
							return prResponse(http.StatusOK, body), nil
						}

						if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/pulls" {
							t.Fatalf("pull request = %s %s", req.Method, req.URL.Path)
						}
						var body map[string]interface{}
						if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
							t.Fatal(err)
						}
						if got := body["base"]; got != tt.wantBase {
							t.Fatalf("base = %q, want %q", got, tt.wantBase)
						}
						return prResponse(http.StatusCreated, `{"number":"7","web_url":"https://atomgit.com/alice/demo/merge_requests/7"}`), nil
					})}, nil
				},
			}

			cmd := newCmdPRCreate(factory)
			_ = cmd.Flags().Set("title", "Test PR")
			_ = cmd.Flags().Set("head", "feature")
			if tt.requestedBase != "" {
				_ = cmd.Flags().Set("base", tt.requestedBase)
			}
			var output strings.Builder
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
				t.Fatal(err)
			}

			wantRequests := 1
			if tt.wantRepoRequest {
				wantRequests = 2
			}
			if requests != wantRequests {
				t.Fatalf("requests = %d, want %d", requests, wantRequests)
			}
			if got := output.String(); got != "Created PR #7: https://atomgit.com/alice/demo/merge_requests/7\n" {
				t.Fatalf("output = %q", got)
			}
		})
	}
}

func TestPRCreateBodyInput(t *testing.T) {
	tests := []struct {
		name      string
		body      *string
		bodyFile  *string
		stdin     string
		wantBody  string
		wantError string
	}{
		{name: "inline body", body: prStringPointer("inline\nbody"), wantBody: "inline\nbody"},
		{name: "UTF-8 file with trailing newlines", bodyFile: prStringPointer("file"), wantBody: "标题\n\n正文\n"},
		{name: "empty file", bodyFile: prStringPointer("empty"), wantBody: ""},
		{name: "stdin", bodyFile: prStringPointer("-"), stdin: "stdin body\n\n", wantBody: "stdin body\n\n"},
		{name: "conflicting flags", body: prStringPointer("inline"), bodyFile: prStringPointer("-"), wantError: "mutually exclusive"},
		{name: "missing file", bodyFile: prStringPointer("missing"), wantError: "failed to read body file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: prTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/pulls" {
							t.Fatalf("request = %s %s", req.Method, req.URL.Path)
						}
						var body map[string]interface{}
						if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
							t.Fatal(err)
						}
						if got := body["body"]; got != tt.wantBody {
							t.Fatalf("body = %q, want %q", got, tt.wantBody)
						}
						return prResponse(http.StatusCreated, `{"number":"7","web_url":"https://atomgit.com/alice/demo/merge_requests/7"}`), nil
					})}, nil
				},
			}

			cmd := newCmdPRCreate(factory)
			_ = cmd.Flags().Set("title", "Test PR")
			_ = cmd.Flags().Set("base", "main")
			_ = cmd.Flags().Set("head", "feature")
			if tt.body != nil {
				_ = cmd.Flags().Set("body", *tt.body)
			}
			if tt.bodyFile != nil {
				path := *tt.bodyFile
				switch path {
				case "file":
					path = filepath.Join(t.TempDir(), "body.md")
					if err := os.WriteFile(path, []byte(tt.wantBody), 0o600); err != nil {
						t.Fatal(err)
					}
				case "empty":
					path = filepath.Join(t.TempDir(), "empty.md")
					if err := os.WriteFile(path, nil, 0o600); err != nil {
						t.Fatal(err)
					}
				case "missing":
					path = filepath.Join(t.TempDir(), "missing.md")
				}
				_ = cmd.Flags().Set("body-file", path)
			}
			cmd.SetIn(strings.NewReader(tt.stdin))

			err := cmd.RunE(cmd, []string{"alice/demo"})
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantError)
				}
				if requests != 0 {
					t.Fatalf("requests = %d, want 0", requests)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want 1", requests)
			}
		})
	}
}

func prStringPointer(value string) *string {
	return &value
}

func TestPRCreateFallsBackToBrowserURL(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusCreated, `{"number":7}`), nil
			})}, nil
		},
	}
	cmd := newCmdPRCreate(factory)
	_ = cmd.Flags().Set("title", "Test PR")
	_ = cmd.Flags().Set("head", "feature")
	_ = cmd.Flags().Set("base", "main")
	var output strings.Builder
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "Created PR #7: https://atomgit.com/alice/demo/pull/7\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestPRWriteCommandsUseRequestNumberForEmptyResponses(t *testing.T) {
	tests := []struct {
		name string
		cmd  func(*cmdutil.Factory) *cobra.Command
		args []string
		want string
	}{
		{name: "edit", cmd: newCmdPREdit, args: []string{"alice/demo", "42"}, want: "Updated PR #42: https://atomgit.com/alice/demo/pull/42\n"},
		{name: "close", cmd: newCmdPRClose, args: []string{"alice/demo", "42"}, want: "Closed PR #42: https://atomgit.com/alice/demo/pull/42\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: prTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						return prResponse(http.StatusNoContent, ""), nil
					})}, nil
				},
			}
			cmd := tt.cmd(factory)
			if tt.name == "edit" {
				_ = cmd.Flags().Set("title", "Updated")
			}
			var output strings.Builder
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, tt.args); err != nil {
				t.Fatal(err)
			}
			if got := output.String(); got != tt.want {
				t.Fatalf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPRViewWebFlag(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config: prTestConfig{},
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := newCmdPRView(f)
	cmd.SetArgs([]string{"--web", "alice/demo", "7"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/pull/7" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func prResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestPRListInfersRepositoryAndHonorsLimit(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
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
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestPRListJSON(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[{"id":1,"number":9,"title":"change","state":"open","head":{"ref":"feature"},"base":{"ref":"main"},"labels":[{"name":"ready"}]}]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRList(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	var values []map[string]any
	if err := json.Unmarshal(output.Bytes(), &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0]["number"] != "9" || values[0]["head"] != "feature" || values[0]["base"] != "main" {
		t.Fatalf("pull requests = %#v", values)
	}
}

func TestPRViewJSON(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if strings.HasSuffix(req.URL.Path, "/labels") {
					return prResponse(http.StatusOK, `[{"name":"reviewed"}]`), nil
				}
				return prResponse(http.StatusOK, `{"id":2,"number":"10","title":"change","state":"open","html_url":"https://atomgit.com/alice/demo/pull/10","user":{"login":"alice"},"head":{"ref":"feature"},"base":{"ref":"main"},"merged":false,"mergeable":true}`), nil
			})}, nil
		},
	}
	cmd := newCmdPRView(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "10"}); err != nil {
		t.Fatal(err)
	}
	var value struct {
		Number    string   `json:"number"`
		Labels    []string `json:"labels"`
		Mergeable bool     `json:"mergeable"`
	}
	if err := json.Unmarshal(output.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.Number != "10" || !value.Mergeable || len(value.Labels) != 1 || value.Labels[0] != "reviewed" {
		t.Fatalf("pull request = %#v", value)
	}
}

func TestPRListRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []string{"0", "-1"} {
		t.Run(limit, func(t *testing.T) {
			cmd := newCmdPRList(&cmdutil.Factory{Config: prTestConfig{tokenErr: fmt.Errorf("not authenticated")}})
			_ = cmd.Flags().Set("limit", limit)
			if err := cmd.RunE(cmd, []string{"alice/demo"}); err == nil || !strings.Contains(err.Error(), "must be positive") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestPRDiff(t *testing.T) {
	const patch = "diff --git a/old.txt b/new.txt\n-old\n+new\n"
	tests := []struct {
		name       string
		statusCode int
		status     string
		body       string
		wantOutput string
		wantError  string
	}{
		{
			name:       "outputs raw patch",
			statusCode: http.StatusOK,
			status:     "200 OK",
			body:       patch,
			wantOutput: patch,
		},
		{
			name:       "reports API error",
			statusCode: http.StatusNotFound,
			status:     "404 Not Found",
			body:       `{"message":"pull request not found"}`,
			wantError:  `API error: 404 Not Found - {"message":"pull request not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: prTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						if req.Method != http.MethodGet {
							t.Errorf("request method = %s, want GET", req.Method)
						}
						if req.URL.Path != "/api/v5/repos/alice/demo/pulls/42/diff" {
							t.Errorf("request URL = %s", req.URL.String())
						}
						if got := req.Header.Get("Authorization"); got != "Bearer token" {
							t.Errorf("Authorization = %q", got)
						}
						return &http.Response{
							StatusCode: tt.statusCode,
							Status:     tt.status,
							Body:       io.NopCloser(strings.NewReader(tt.body)),
							Header:     make(http.Header),
						}, nil
					})}, nil
				},
			}

			cmd := newCmdPRDiff(factory)
			var output strings.Builder
			cmd.SetOut(&output)
			err := cmd.RunE(cmd, []string{"alice/demo", "42"})

			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantError)
				}
			} else if err != nil {
				t.Fatalf("RunE() error = %v", err)
			}
			if output.String() != tt.wantOutput {
				t.Fatalf("output = %q, want %q", output.String(), tt.wantOutput)
			}
		})
	}
}

func TestPRMergeSuccess(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		flags      map[string]string
		wantOutput string
	}{
		{
			name: "default merge",
			args: []string{
				"42",
			},
			flags:      map[string]string{},
			wantOutput: "Merged PR #42: https://atomgit.com/alice/demo/pulls/42\n",
		},
		{
			name: "rebase merge",
			args: []string{
				"alice/demo",
				"42",
			},
			flags: map[string]string{
				"rebase": "true",
			},
			wantOutput: "Rebased and merged PR #42: https://atomgit.com/alice/demo/pulls/42\n",
		},
		{
			name: "squash merge",
			args: []string{
				"alice/demo",
				"42",
			},
			flags: map[string]string{
				"squash": "true",
			},
			wantOutput: "Squashed and merged PR #42: https://atomgit.com/alice/demo/pulls/42\n",
		},
		{
			name: "rebase with squash",
			args: []string{
				"alice/demo",
				"42",
			},
			flags: map[string]string{
				"rebase": "true",
				"squash": "true",
			},
			wantOutput: "Rebased and merged PR with squash #42: https://atomgit.com/alice/demo/pulls/42\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mergeBody json.RawMessage
			factory := &cmdutil.Factory{
				Config: prTestConfig{},
				RepositoryResolver: func() (cmdutil.Repository, error) {
					return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
				},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						if req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42" {
							return prResponse(http.StatusOK, `{"id":1,"number":"42","title":"test","state":"open","merged":false,"html_url":"https://atomgit.com/alice/demo/pulls/42","head":{"ref":"feature-x","sha":"abc","repo":{}},"base":{"ref":"main","sha":"def","repo":{}}}`), nil
						}
						if req.Method == http.MethodPut && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/merge" {
							body, _ := io.ReadAll(req.Body)
							mergeBody = json.RawMessage(body)
							return prResponse(http.StatusOK, `{"sha":"c20ac962","merged":true,"message":"Pull Request已成功合并"}`), nil
						}
						t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
						return nil, nil
					})}, nil
				},
			}

			cmd := newCmdPRMerge(factory)
			for k, v := range tt.flags {
				_ = cmd.Flags().Set(k, v)
			}
			var output strings.Builder
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, tt.args); err != nil {
				t.Fatal(err)
			}

			if output.String() != tt.wantOutput {
				t.Fatalf("output = %q, want %q", output.String(), tt.wantOutput)
			}

			var body map[string]any
			if err := json.Unmarshal(mergeBody, &body); err != nil {
				t.Fatal(err)
			}
			if _, ok := body["merge_method"]; !ok {
				t.Fatal("merge_method not found in request body")
			}
		})
	}
}

func TestPRMergeFlagsInBody(t *testing.T) {
	tests := []struct {
		name     string
		flags    map[string]string
		wantBody map[string]any
	}{
		{
			name:  "rebase",
			flags: map[string]string{"rebase": "true"},
			wantBody: map[string]any{
				"merge_method": "rebase",
			},
		},
		{
			name:  "squash",
			flags: map[string]string{"squash": "true"},
			wantBody: map[string]any{
				"merge_method": "merge",
				"squash":       true,
			},
		},
		{
			name:  "admin",
			flags: map[string]string{"admin": "true"},
			wantBody: map[string]any{
				"merge_method": "merge",
				"force_merge":  true,
			},
		},
		{
			name:  "subject and body without squash",
			flags: map[string]string{"subject": "my title", "body": "my desc"},
			wantBody: map[string]any{
				"merge_method": "merge",
				"title":        "my title",
				"description":  "my desc",
			},
		},
		{
			name:  "body with squash",
			flags: map[string]string{"squash": "true", "body": "squash msg"},
			wantBody: map[string]any{
				"merge_method":          "merge",
				"squash":                true,
				"squash_commit_message": "squash msg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mergeBody json.RawMessage
			factory := &cmdutil.Factory{
				Config: prTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						if req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42" {
							return prResponse(http.StatusOK, `{"id":1,"number":"42","title":"test","state":"open","merged":false,"html_url":"https://atomgit.com/alice/demo/pulls/42","head":{"ref":"feature-x","sha":"abc","repo":{}},"base":{"ref":"main","sha":"def","repo":{}}}`), nil
						}
						if req.Method == http.MethodPut && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/merge" {
							body, _ := io.ReadAll(req.Body)
							mergeBody = json.RawMessage(body)
							return prResponse(http.StatusOK, `{"sha":"sha","merged":true,"message":"ok"}`), nil
						}
						t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
						return nil, nil
					})}, nil
				},
			}

			cmd := newCmdPRMerge(factory)
			for k, v := range tt.flags {
				_ = cmd.Flags().Set(k, v)
			}
			var output strings.Builder
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
				t.Fatal(err)
			}

			var got map[string]any
			if err := json.Unmarshal(mergeBody, &got); err != nil {
				t.Fatal(err)
			}
			for field, want := range tt.wantBody {
				if got[field] != want {
					t.Fatalf("body[%q] = %v, want %v", field, got[field], want)
				}
			}
		})
	}
}

func TestPRMergeDeleteBranch(t *testing.T) {
	branchDeleted := false
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42" {
					return prResponse(http.StatusOK, `{"id":1,"number":"42","title":"test","state":"open","merged":false,"html_url":"https://atomgit.com/alice/demo/pulls/42","head":{"ref":"feature-x","sha":"abc","repo":{"full_name":"alice/demo"}},"base":{"ref":"main","sha":"def","repo":{}}}`), nil
				}
				if req.Method == http.MethodPut && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/merge" {
					return prResponse(http.StatusOK, `{"sha":"sha","merged":true,"message":"ok"}`), nil
				}
				if req.Method == http.MethodDelete && req.URL.Path == "/api/v5/repos/alice/demo/branches/feature-x" {
					branchDeleted = true
					return prResponse(http.StatusNoContent, ""), nil
				}
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				return nil, nil
			})}, nil
		},
	}

	cmd := newCmdPRMerge(factory)
	_ = cmd.Flags().Set("delete-branch", "true")
	var output strings.Builder
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}

	if !branchDeleted {
		t.Fatal("branch was not deleted")
	}

	wantOutput := "Merged PR #42: https://atomgit.com/alice/demo/pulls/42\nDeleted remote branch feature-x\n"
	if output.String() != wantOutput {
		t.Fatalf("output = %q, want %q", output.String(), wantOutput)
	}
}

func TestPRMergeDeleteBranchFails(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42" {
					return prResponse(http.StatusOK, `{"id":1,"number":"42","title":"test","state":"open","merged":false,"html_url":"https://atomgit.com/alice/demo/pulls/42","head":{"ref":"feature-x","sha":"abc","repo":{"full_name":"alice/demo"}},"base":{"ref":"main","sha":"def","repo":{}}}`), nil
				}
				if req.Method == http.MethodPut && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/merge" {
					return prResponse(http.StatusOK, `{"sha":"sha","merged":true,"message":"ok"}`), nil
				}
				if req.Method == http.MethodDelete && req.URL.Path == "/api/v5/repos/alice/demo/branches/feature-x" {
					return prResponse(http.StatusNotFound, `{"message":"branch not found"}`), nil
				}
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				return nil, nil
			})}, nil
		},
	}

	cmd := newCmdPRMerge(factory)
	_ = cmd.Flags().Set("delete-branch", "true")
	var stderr strings.Builder
	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}

	wantStdout := "Merged PR #42: https://atomgit.com/alice/demo/pulls/42\n"
	if stdout.String() != wantStdout {
		t.Fatalf("stdout = %q, want %q", stdout.String(), wantStdout)
	}
	if !strings.Contains(stderr.String(), "Warning: failed to delete branch") {
		t.Fatalf("stderr = %q, want containing 'Warning: failed to delete branch'", stderr.String())
	}
}

func TestPRMergeDeleteBranchFork(t *testing.T) {
	branchDeleted := false
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42" {
					return prResponse(http.StatusOK, `{"id":1,"number":"42","title":"test","state":"open","merged":false,"html_url":"https://atomgit.com/alice/demo/pulls/42","head":{"ref":"feature-x","sha":"abc","repo":{"full_name":"alice/fork-demo"}},"base":{"ref":"main","sha":"def","repo":{}}}`), nil
				}
				if req.Method == http.MethodPut && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/merge" {
					return prResponse(http.StatusOK, `{"sha":"sha","merged":true,"message":"ok"}`), nil
				}
				if req.Method == http.MethodDelete && req.URL.Path == "/api/v5/repos/alice/fork-demo/branches/feature-x" {
					branchDeleted = true
					return prResponse(http.StatusNoContent, ""), nil
				}
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				return nil, nil
			})}, nil
		},
	}

	cmd := newCmdPRMerge(factory)
	_ = cmd.Flags().Set("delete-branch", "true")
	var output strings.Builder
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}

	if !branchDeleted {
		t.Fatal("branch was not deleted from fork repo")
	}

	wantOutput := "Merged PR #42: https://atomgit.com/alice/demo/pulls/42\nDeleted remote branch feature-x\n"
	if output.String() != wantOutput {
		t.Fatalf("output = %q, want %q", output.String(), wantOutput)
	}
}

func TestPRMergeDeleteBranchWithSlash(t *testing.T) {
	branchDeleted := false
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42" {
					return prResponse(http.StatusOK, `{"id":1,"number":"42","title":"test","state":"open","merged":false,"html_url":"https://atomgit.com/alice/demo/pulls/42","head":{"ref":"feature/x","sha":"abc","repo":{"full_name":"alice/demo"}},"base":{"ref":"main","sha":"def","repo":{}}}`), nil
				}
				if req.Method == http.MethodPut && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/merge" {
					return prResponse(http.StatusOK, `{"sha":"sha","merged":true,"message":"ok"}`), nil
				}
				if req.Method == http.MethodDelete && req.URL.EscapedPath() == "/api/v5/repos/alice/demo/branches/feature%2Fx" {
					branchDeleted = true
					return prResponse(http.StatusNoContent, ""), nil
				}
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				return nil, nil
			})}, nil
		},
	}

	cmd := newCmdPRMerge(factory)
	_ = cmd.Flags().Set("delete-branch", "true")
	var output strings.Builder
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}

	if !branchDeleted {
		t.Fatal("branch with slash was not deleted")
	}

	wantOutput := "Merged PR #42: https://atomgit.com/alice/demo/pulls/42\nDeleted remote branch feature/x\n"
	if output.String() != wantOutput {
		t.Fatalf("output = %q, want %q", output.String(), wantOutput)
	}
}

func TestPRMergeDeleteBranchSkipsEmptySourceBranch(t *testing.T) {
	branchDeleted := false
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42" {
					return prResponse(http.StatusOK, `{"id":1,"number":"42","title":"test","state":"open","merged":false,"html_url":"https://atomgit.com/alice/demo/pulls/42","head":{"ref":"","sha":"abc","repo":{"full_name":"alice/demo"}},"base":{"ref":"main","sha":"def","repo":{}}}`), nil
				}
				if req.Method == http.MethodPut && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42/merge" {
					return prResponse(http.StatusOK, `{"sha":"sha","merged":true,"message":"ok"}`), nil
				}
				if req.Method == http.MethodDelete {
					branchDeleted = true
					t.Fatalf("unexpected delete request: %s", req.URL.Path)
				}
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				return nil, nil
			})}, nil
		},
	}

	cmd := newCmdPRMerge(factory)
	_ = cmd.Flags().Set("delete-branch", "true")
	var output strings.Builder
	var stderr strings.Builder
	cmd.SetOut(&output)
	cmd.SetErr(&stderr)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}

	if branchDeleted {
		t.Fatal("branch deletion was attempted for an empty source branch")
	}
	if !strings.Contains(stderr.String(), "cannot determine source repository or branch") {
		t.Fatalf("stderr = %q, want warning about missing source repository or branch", stderr.String())
	}
	if got := output.String(); got != "Merged PR #42: https://atomgit.com/alice/demo/pulls/42\n" {
		t.Fatalf("output = %q, want merged message only", got)
	}
}

func TestPRMergeErrors(t *testing.T) {
	tests := []struct {
		name      string
		getStatus int
		getBody   string
		wantError string
	}{
		{
			name:      "PR not found",
			getStatus: http.StatusNotFound,
			getBody:   `{"message":"not found"}`,
			wantError: "failed to get PR alice/demo #42",
		},
		{
			name:      "PR already merged",
			getStatus: http.StatusOK,
			getBody:   `{"id":1,"number":"42","state":"open","merged":true}`,
			wantError: "PR #42 is already merged",
		},
		{
			name:      "PR is closed",
			getStatus: http.StatusOK,
			getBody:   `{"id":1,"number":"42","state":"closed","merged":false}`,
			wantError: "PR #42 is closed, cannot merge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: prTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						return prResponse(tt.getStatus, tt.getBody), nil
					})}, nil
				},
			}

			cmd := newCmdPRMerge(factory)
			err := cmd.RunE(cmd, []string{"alice/demo", "42"})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestPRMergeAPIError(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/pulls/42" {
					return prResponse(http.StatusOK, `{"id":1,"number":"42","title":"test","state":"open","merged":false,"html_url":"https://atomgit.com/alice/demo/pulls/42","head":{"ref":"feature-x","sha":"abc","repo":{}},"base":{"ref":"main","sha":"def","repo":{}}}`), nil
				}
				if req.Method == http.MethodPut {
					return prResponse(http.StatusUnprocessableEntity, `{"message":"merge conflict"}`), nil
				}
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				return nil, nil
			})}, nil
		},
	}

	cmd := newCmdPRMerge(factory)
	err := cmd.RunE(cmd, []string{"alice/demo", "42"})
	if err == nil || !strings.Contains(err.Error(), "failed to merge PR #42") {
		t.Fatalf("error = %v, want containing 'failed to merge PR #42'", err)
	}
}

func TestPRMergeValidation(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name: "invalid repository format",
			args: []string{
				"invalid",
				"42",
			},
			wantError: "invalid repository format",
		},
		{
			name: "empty owner",
			args: []string{
				"/repo",
				"42",
			},
			wantError: "invalid repository format",
		},
		{
			name: "empty repo",
			args: []string{
				"owner/",
				"42",
			},
			wantError: "invalid repository format",
		},
		{
			name: "zero PR number",
			args: []string{
				"owner/repo",
				"0",
			},
			wantError: "invalid PR number",
		},
		{
			name: "negative PR number",
			args: []string{
				"owner/repo",
				"-1",
			},
			wantError: "invalid PR number",
		},
		{
			name: "non-numeric PR number",
			args: []string{
				"owner/repo",
				"abc",
			},
			wantError: "invalid PR number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdPRMerge(&cmdutil.Factory{
				Config: prTestConfig{},
				HttpClient: func() (*http.Client, error) {
					t.Fatal("HTTP client should not be created for invalid merge input")
					return nil, nil
				},
			})
			err := cmd.RunE(cmd, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestPRReopenSendsOpenState(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				gotMethod = req.Method
				gotPath = req.URL.Path
				b, _ := io.ReadAll(req.Body)
				gotBody = string(b)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"state":"opened"}`)), Header: make(http.Header)}, nil
			})}, nil
		},
	}
	cmd := newCmdPRReopen(factory)
	var output strings.Builder
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "5"}); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %s, want PATCH", gotMethod)
	}
	if gotPath != "/api/v5/repos/alice/demo/pulls/5" {
		t.Errorf("path = %s, want /api/v5/repos/alice/demo/pulls/5", gotPath)
	}
	if !strings.Contains(gotBody, `"state":"open"`) {
		t.Errorf("body = %s, want state:open", gotBody)
	}
	if got := output.String(); got != "Reopened PR #5: https://atomgit.com/alice/demo/pull/5\n" {
		t.Errorf("output = %q", got)
	}
}

func TestPRReopenRejectsWrongArgCount(t *testing.T) {
	cmd := newCmdPRReopen(&cmdutil.Factory{Config: prTestConfig{}})
	if err := cmd.Args(cmd, nil); err == nil {
		t.Fatal("reopen accepted no PR number")
	}
	if err := cmd.Args(cmd, []string{"5"}); err != nil {
		t.Fatalf("reopen rejected an inferred-repository invocation: %v", err)
	}
	if err := cmd.Args(cmd, []string{"alice/demo", "5", "extra"}); err == nil {
		t.Fatal("reopen accepted 3 args, want error")
	}
}

func TestPRReopenWrapsAPIError(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"message":"not found"}`)), Header: make(http.Header)}, nil
			})}, nil
		},
	}
	cmd := newCmdPRReopen(factory)
	err := cmd.RunE(cmd, []string{"alice/demo", "5"})
	if err == nil || !strings.Contains(err.Error(), "failed to reopen PR") {
		t.Fatalf("error = %v, want 'failed to reopen PR'", err)
	}
}

func TestParsePRNumberRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    string
	}{
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "non_numeric", input: "not-a-number", wantErr: true},
		{name: "decimal", input: "1.5", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "whitespace_only", input: "   ", wantErr: true},
		{name: "mixed_alphanumeric", input: "12abc", wantErr: true},
		{name: "positive", input: "42", wantErr: false, want: "42"},
		{name: "trimmed_positive", input: "  7  ", wantErr: false, want: "7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePRNumber(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parsePRNumber(%q) = %q, want error", tt.input, got)
				}
				if !strings.Contains(err.Error(), "invalid PR number") {
					t.Fatalf("parsePRNumber(%q) err = %v, want 'invalid PR number' substring", tt.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePRNumber(%q) unexpected err: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("parsePRNumber(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

type recordingConfig struct {
	getTokenCalls int
}

func (r *recordingConfig) GetToken() (string, error) {
	r.getTokenCalls++
	return "token", nil
}
func (*recordingConfig) GetUser() (string, error) { return "alice", nil }
func (*recordingConfig) GetHost() string          { return "atomgit.com" }

func TestCmdPRCheckoutRejectsInvalidPRNumberBeforeAPI(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "zero", input: "0"},
		{name: "negative", input: "-1"},
		{name: "non_numeric", input: "not-a-number"},
		{name: "decimal", input: "1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &recordingConfig{}
			f := &cmdutil.Factory{Config: cfg}
			cmd := newCmdPRCheckout(f)
			// "--" prevents cobra from interpreting negative numbers as flags,
			// so the value reaches parsePRNumber and exercises its validation.
			cmd.SetArgs([]string{"--", "alice/demo", tt.input})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("Execute() with number=%q returned nil error, want validation failure", tt.input)
			}
			if !strings.Contains(err.Error(), "invalid PR number") {
				t.Fatalf("Execute() err = %v, want 'invalid PR number' substring", err)
			}
			if cfg.getTokenCalls != 0 {
				t.Fatalf("GetToken was called %d times; parsePRNumber must reject invalid input before authentication", cfg.getTokenCalls)
			}
		})
	}
}
