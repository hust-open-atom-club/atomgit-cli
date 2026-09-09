package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestRepoSyncFastForwardsDefaultBranch(t *testing.T) {
	var requests []string
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.EscapedPath())
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo":
			return forkResponse(http.StatusOK, `{"fork":true,"parentfull_name":"upstream/demo","default_branch":"main"}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/branches/main":
			return forkResponse(http.StatusOK, `{"commit":{"sha":"old"}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/upstream/demo/branches/main":
			return forkResponse(http.StatusOK, `{"commit":{"sha":"new"}}`), nil
		case req.Method == http.MethodPut && req.URL.Path == "/api/v5/repos/alice/demo/sync_repo":
			var body map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["branch"] != "main" {
				t.Fatalf("body = %#v", body)
			}
			if _, present := body["force"]; present {
				t.Fatalf("non-forced request included force: %#v", body)
			}
			return forkResponse(http.StatusOK, `{"repo_sync_result":true}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.EscapedPath())
		}
		return nil, nil
	})
	cmd := newCmdRepoSync(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 4 {
		t.Fatalf("requests = %v", requests)
	}
	for _, text := range []string{"Repository: alice/demo", "Upstream: upstream/demo", "Branch: main", "Synchronized alice/demo:main from upstream/demo"} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("output missing %q:\n%s", text, out.String())
		}
	}
}

func TestRepoSyncAlreadyUpToDateDoesNotWrite(t *testing.T) {
	puts := 0
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPut {
			puts++
		}
		switch req.URL.Path {
		case "/api/v5/repos/alice/demo":
			return forkResponse(http.StatusOK, `{"fork":true,"parentfull_name":"upstream/demo","default_branch":"main"}`), nil
		case "/api/v5/repos/alice/demo/branches/main", "/api/v5/repos/upstream/demo/branches/main":
			return forkResponse(http.StatusOK, `{"commit":{"sha":"same"}}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.EscapedPath())
		}
		return nil, nil
	})
	cmd := newCmdRepoSync(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if puts != 0 || !strings.Contains(out.String(), "Already up to date.") {
		t.Fatalf("puts = %d, output = %q", puts, out.String())
	}
}

func TestRepoSyncForceRequiresConfirmation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		yes      bool
		wantPut  bool
		wantText string
	}{
		{name: "confirmed", input: "yes\n", wantPut: true, wantText: "(forced)"},
		{name: "cancelled", input: "no\n", wantText: "cancelled"},
		{name: "yes flag", yes: true, wantPut: true, wantText: "(forced)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			puts := 0
			transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo":
					return forkResponse(http.StatusOK, `{"fork":true,"parentfull_name":"upstream/demo","default_branch":"main"}`), nil
				case req.Method == http.MethodGet && strings.Contains(req.URL.Path, "/branches/"):
					if !strings.HasSuffix(req.URL.EscapedPath(), "/branches/feature%2Ffoo") {
						t.Fatalf("branch path was not escaped: %s", req.URL.EscapedPath())
					}
					sha := "fork"
					if strings.Contains(req.URL.Path, "/upstream/") {
						sha = "upstream"
					}
					return forkResponse(http.StatusOK, `{"commit":{"sha":"`+sha+`"}}`), nil
				case req.Method == http.MethodPut:
					puts++
					var body apiSyncRequest
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if body.Branch != "feature/foo" || !body.Force {
						t.Fatalf("body = %#v", body)
					}
					return forkResponse(http.StatusOK, `{"repo_sync_result":true}`), nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.EscapedPath())
				}
				return nil, nil
			})
			cmd := newCmdRepoSync(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
			_ = cmd.Flags().Set("branch", "feature/foo")
			_ = cmd.Flags().Set("force", "true")
			if tt.yes {
				_ = cmd.Flags().Set("yes", "true")
			}
			cmd.SetIn(strings.NewReader(tt.input))
			var out bytes.Buffer
			cmd.SetOut(&out)

			if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
				t.Fatal(err)
			}
			if (puts == 1) != tt.wantPut || !strings.Contains(out.String(), tt.wantText) {
				t.Fatalf("puts = %d, output = %q", puts, out.String())
			}
		})
	}
}

type apiSyncRequest struct {
	Branch string `json:"branch"`
	Force  bool   `json:"force"`
}

func TestRepoSyncValidatesRepositoryAndBranches(t *testing.T) {
	tests := []struct {
		name     string
		repoBody string
		failPath string
		status   int
		want     string
	}{
		{name: "not a fork", repoBody: `{"fork":false}`, want: "is not a fork"},
		{name: "missing upstream", repoBody: `{"fork":true,"default_branch":"main"}`, want: "no usable upstream"},
		{name: "missing default branch", repoBody: `{"fork":true,"parentfull_name":"upstream/demo"}`, want: "has no default branch"},
		{name: "fork branch missing", repoBody: `{"fork":true,"parentfull_name":"upstream/demo","default_branch":"main"}`, failPath: "/api/v5/repos/alice/demo/branches/main", status: http.StatusNotFound, want: "in fork alice/demo"},
		{name: "upstream branch missing", repoBody: `{"fork":true,"parentfull_name":"upstream/demo","default_branch":"main"}`, failPath: "/api/v5/repos/upstream/demo/branches/main", status: http.StatusNotFound, want: "in upstream upstream/demo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			puts := 0
			transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodPut {
					puts++
				}
				if req.URL.Path == "/api/v5/repos/alice/demo" {
					return forkResponse(http.StatusOK, tt.repoBody), nil
				}
				if req.URL.Path == tt.failPath {
					return forkResponse(tt.status, `{"message":"missing"}`), nil
				}
				if strings.Contains(req.URL.Path, "/branches/") {
					return forkResponse(http.StatusOK, `{"commit":{"sha":"sha"}}`), nil
				}
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.EscapedPath())
				return nil, nil
			})
			cmd := newCmdRepoSync(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
			err := cmd.RunE(cmd, []string{"alice/demo"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
			if puts != 0 {
				t.Fatalf("put count = %d", puts)
			}
		})
	}
}

func TestRepoSyncReportsConflictPermissionAndRejectedResult(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		response string
		want     string
	}{
		{name: "conflict", status: http.StatusConflict, response: `{"message":"conflict"}`, want: "409"},
		{name: "permission", status: http.StatusForbidden, response: `{"message":"denied"}`, want: "403"},
		{name: "rejected result", status: http.StatusOK, response: `{"repo_sync_result":false}`, want: "did not synchronize"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo":
					return forkResponse(http.StatusOK, `{"fork":true,"parentfull_name":"upstream/demo","default_branch":"main"}`), nil
				case req.Method == http.MethodGet && strings.Contains(req.URL.Path, "/branches/"):
					sha := "fork"
					if strings.Contains(req.URL.Path, "/upstream/") {
						sha = "upstream"
					}
					return forkResponse(http.StatusOK, `{"commit":{"sha":"`+sha+`"}}`), nil
				case req.Method == http.MethodPut:
					return forkResponse(tt.status, tt.response), nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.EscapedPath())
				}
				return nil, nil
			})
			cmd := newCmdRepoSync(repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport))
			err := cmd.RunE(cmd, []string{"alice/demo"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRepoSyncAuthenticationErrorDoesNotRequest(t *testing.T) {
	requests := 0
	transport := forkRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return forkResponse(http.StatusOK, `{}`), nil
	})
	tokenErr := errors.New("missing token")
	cmd := newCmdRepoSync(repoFactory(repoCommandConfig{tokenErr: tokenErr, user: "alice"}, transport))
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil || err.Error() != "not authenticated: missing token" || !errors.Is(err, tokenErr) || requests != 0 {
		t.Fatalf("error = %v, requests = %d", err, requests)
	}
}
