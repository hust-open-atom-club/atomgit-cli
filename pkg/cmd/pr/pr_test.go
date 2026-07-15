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
