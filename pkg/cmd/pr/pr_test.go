package pr

import (
	"encoding/json"
	"fmt"
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
						return prResponse(http.StatusCreated, `{"number":"7","html_url":"https://atomgit.com/alice/demo/pulls/7"}`), nil
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
			if got := output.String(); got != "Created PR #7: https://atomgit.com/alice/demo/pull/7\n" {
				t.Fatalf("output = %q", got)
			}
		})
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
		{name: "default merge", args: []string{"alice/demo", "42"}, flags: map[string]string{}, wantOutput: "Merged PR #42: https://atomgit.com/alice/demo/pulls/42\n"},
		{name: "rebase merge", args: []string{"alice/demo", "42"}, flags: map[string]string{"rebase": "true"}, wantOutput: "Rebased and merged PR #42: https://atomgit.com/alice/demo/pulls/42\n"},
		{name: "squash merge", args: []string{"alice/demo", "42"}, flags: map[string]string{"squash": "true"}, wantOutput: "Squashed and merged PR #42: https://atomgit.com/alice/demo/pulls/42\n"},
		{name: "rebase with squash", args: []string{"alice/demo", "42"}, flags: map[string]string{"rebase": "true", "squash": "true"}, wantOutput: "Rebased and merged PR with squash #42: https://atomgit.com/alice/demo/pulls/42\n"},
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

			var body map[string]interface{}
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
		wantBody map[string]interface{}
	}{
		{
			name:  "rebase",
			flags: map[string]string{"rebase": "true"},
			wantBody: map[string]interface{}{
				"merge_method": "rebase",
			},
		},
		{
			name:  "squash",
			flags: map[string]string{"squash": "true"},
			wantBody: map[string]interface{}{
				"merge_method": "merge",
				"squash":       true,
			},
		},
		{
			name:  "admin",
			flags: map[string]string{"admin": "true"},
			wantBody: map[string]interface{}{
				"merge_method": "merge",
				"force_merge":  true,
			},
		},
		{
			name:  "subject and body without squash",
			flags: map[string]string{"subject": "my title", "body": "my desc"},
			wantBody: map[string]interface{}{
				"merge_method": "merge",
				"title":        "my title",
				"description":  "my desc",
			},
		},
		{
			name:  "body with squash",
			flags: map[string]string{"squash": "true", "body": "squash msg"},
			wantBody: map[string]interface{}{
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

			var got map[string]interface{}
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
	cmd := newCmdPRMerge(&cmdutil.Factory{Config: prTestConfig{}})
	err := cmd.RunE(cmd, []string{"invalid", "42"})
	if err == nil || !strings.Contains(err.Error(), "invalid repository format") {
		t.Fatalf("error = %v, want containing 'invalid repository format'", err)
	}
}
