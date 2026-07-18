package branch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type branchCommandConfig struct {
	token    string
	tokenErr error
}

func (c branchCommandConfig) GetToken() (string, error) { return c.token, c.tokenErr }
func (c branchCommandConfig) GetUser() (string, error)  { return "alice", nil }
func (c branchCommandConfig) GetHost() string           { return "atomgit.com" }

type branchRoundTripFunc func(*http.Request) (*http.Response, error)

func (f branchRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func branchFactory(config branchCommandConfig, transport branchRoundTripFunc) *cmdutil.Factory {
	factory := &cmdutil.Factory{Config: config}
	if transport != nil {
		factory.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return factory
}

func branchResponse(status int, body string) *http.Response {
	if body == "" {
		body = "{}"
	}
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       ioNopCloser{strings.NewReader(body)},
		Header:     make(http.Header),
	}
}

type ioNopCloser struct {
	*strings.Reader
}

func (c ioNopCloser) Close() error { return nil }

func TestNewCmdBranchRegistersSubcommandsAndHelp(t *testing.T) {
	cmd := NewCmdBranch(&cmdutil.Factory{})
	want := map[string][]string{
		"list":   {"limit"},
		"view":   {},
		"create": {"ref"},
		"delete": {"yes"},
	}
	for name, flags := range want {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child.Name() != name {
			t.Fatalf("subcommand %q: %v", name, err)
		}
		for _, flag := range flags {
			if child.Flags().Lookup(flag) == nil {
				t.Errorf("%s --%s flag was not registered", name, flag)
			}
		}
	}

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := out.String()
	for _, text := range []string{"list", "view", "create", "delete", "ag branch create owner/repo feature/foo --ref main"} {
		if !strings.Contains(help, text) {
			t.Fatalf("help missing %q:\n%s", text, help)
		}
	}
}

func TestBranchListPaginatesHonorsLimitAndFormatsOutput(t *testing.T) {
	requests := 0
	transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/branches" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		query, err := url.ParseQuery(req.URL.RawQuery)
		if err != nil {
			t.Fatal(err)
		}
		if query.Get("per_page") != "100" || query.Get("page") != fmt.Sprint(requests) {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}

		count := 100
		start := 0
		if requests == 2 {
			count = 2
			start = 100
		}
		branches := make([]map[string]any, count)
		for i := range branches {
			branches[i] = map[string]any{
				"name":       fmt.Sprintf("branch-%d", start+i),
				"protected":  i%2 == 0,
				"created_at": "2026-07-18T00:00:00Z",
				"creator":    map[string]any{"login": "alice"},
				"commit":     map[string]any{"sha": fmt.Sprintf("%040d", start+i), "commit": map[string]any{"message": "commit message"}},
			}
		}
		data, err := json.Marshal(branches)
		if err != nil {
			t.Fatal(err)
		}
		return branchResponse(http.StatusOK, string(data)), nil
	})
	cmd := newCmdBranchList(branchFactory(branchCommandConfig{token: "token"}, transport))
	if err := cmd.Flags().Set("limit", "101"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("request count = %d", requests)
	}
	output := out.String()
	if !strings.Contains(output, "branch-0 000000000000 commit message protected:true created:2026-07-18T00:00:00Z creator:alice") ||
		!strings.Contains(output, "branch-100") ||
		strings.Contains(output, "branch-101") {
		t.Fatalf("output =\n%s", output)
	}
}

func TestBranchListRejectsInvalidLimitBeforeRequest(t *testing.T) {
	requests := 0
	cmd := newCmdBranchList(branchFactory(branchCommandConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		requests++
		return branchResponse(http.StatusOK, "[]"), nil
	}))
	if err := cmd.Flags().Set("limit", "0"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("request count = %d", requests)
	}
}

func TestBranchListReportsPermissionError(t *testing.T) {
	cmd := newCmdBranchList(branchFactory(branchCommandConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		return branchResponse(http.StatusForbidden, `{"message":"denied"}`), nil
	}))
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil || !strings.Contains(err.Error(), "failed to list branches") || !strings.Contains(err.Error(), "403") {
		t.Fatalf("error = %v", err)
	}
}

func TestBranchViewUsesEscapedPathAndFormatsOutput(t *testing.T) {
	transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.EscapedPath() != "/api/v5/repos/alice/demo/branches/feature%2Ffoo" {
			t.Fatalf("request = %s %s raw=%s", req.Method, req.URL.Path, req.URL.RawPath)
		}
		return branchResponse(http.StatusOK, `{"name":"feature/foo","protected":1,"default":0,"can_push":true,"commit":{"id":"1234567890abcdef","title":"Feature work"},"created_at":"2026-07-18T00:00:00Z","creator":{"login":"alice"}}`), nil
	})
	cmd := newCmdBranchView(branchFactory(branchCommandConfig{token: "token"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(cmd, []string{"alice/demo", "feature/foo"}); err != nil {
		t.Fatal(err)
	}
	output := out.String()
	for _, text := range []string{"Name: feature/foo", "Commit: 1234567890ab Feature work", "Protected: true", "Can Push: true", "Creator: alice"} {
		if !strings.Contains(output, text) {
			t.Fatalf("output missing %q:\n%s", text, output)
		}
	}
}

func TestBranchViewReportsNotFoundAndPermissionErrors(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cmd := newCmdBranchView(branchFactory(branchCommandConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				return branchResponse(status, `{"message":"error"}`), nil
			}))
			err := cmd.RunE(cmd, []string{"alice/demo", "missing"})
			if err == nil || !strings.Contains(err.Error(), "failed to view branch") || !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestBranchCreateSendsRequestBodyAndReportsSuccess(t *testing.T) {
	transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/branches" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["branch_name"] != "feature/foo" || body["refs"] != "main" {
			t.Fatalf("body = %#v", body)
		}
		return branchResponse(http.StatusOK, `{"name":"feature/foo"}`), nil
	})
	cmd := newCmdBranchCreate(branchFactory(branchCommandConfig{token: "token"}, transport))
	if err := cmd.Flags().Set("ref", "main"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(cmd, []string{"alice/demo", "feature/foo"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Created branch feature/foo") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestBranchCreateValidatesBeforeRequest(t *testing.T) {
	requests := 0
	cmd := newCmdBranchCreate(branchFactory(branchCommandConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		requests++
		return branchResponse(http.StatusOK, `{}`), nil
	}))
	err := cmd.RunE(cmd, []string{"alice/demo", "feature/foo"})
	if err == nil || !strings.Contains(err.Error(), "source ref is required") {
		t.Fatalf("error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("request count = %d", requests)
	}
}

func TestBranchCreateReportsInvalidDuplicateAndPermissionErrors(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusConflict, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cmd := newCmdBranchCreate(branchFactory(branchCommandConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				return branchResponse(status, `{"message":"error"}`), nil
			}))
			_ = cmd.Flags().Set("ref", "main")
			err := cmd.RunE(cmd, []string{"alice/demo", "feature/foo"})
			if err == nil || !strings.Contains(err.Error(), "failed to create branch") || !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestBranchDeleteConfirmedAndYes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		yes   bool
	}{
		{name: "confirmed", input: "yes\n"},
		{name: "yes flag", yes: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var methods []string
			transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method+" "+req.URL.EscapedPath())
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo":
					return branchResponse(http.StatusOK, `{"default_branch":"main"}`), nil
				case req.Method == http.MethodGet && req.URL.EscapedPath() == "/api/v5/repos/alice/demo/branches/feature%2Ffoo":
					return branchResponse(http.StatusOK, `{"name":"feature/foo","protected":false}`), nil
				case req.Method == http.MethodDelete && req.URL.EscapedPath() == "/api/v5/repos/alice/demo/branches/feature%2Ffoo":
					return branchResponse(http.StatusNoContent, ""), nil
				default:
					t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
				}
				return nil, nil
			})
			cmd := newCmdBranchDelete(branchFactory(branchCommandConfig{token: "token"}, transport))
			if tt.yes {
				_ = cmd.Flags().Set("yes", "true")
			}
			cmd.SetIn(strings.NewReader(tt.input))
			var out bytes.Buffer
			cmd.SetOut(&out)

			if err := cmd.RunE(cmd, []string{"alice/demo", "feature/foo"}); err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(methods, ","); !strings.Contains(got, "DELETE /api/v5/repos/alice/demo/branches/feature%2Ffoo") {
				t.Fatalf("methods = %v", methods)
			}
			if !strings.Contains(out.String(), "Deleted branch feature/foo") {
				t.Fatalf("output = %s", out.String())
			}
		})
	}
}

func TestBranchDeleteCancellationDoesNotDelete(t *testing.T) {
	deletes := 0
	transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodDelete {
			deletes++
		}
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo":
			return branchResponse(http.StatusOK, `{"default_branch":"main"}`), nil
		case req.Method == http.MethodGet && req.URL.EscapedPath() == "/api/v5/repos/alice/demo/branches/feature%2Ffoo":
			return branchResponse(http.StatusOK, `{"name":"feature/foo","protected":false}`), nil
		default:
			t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
		}
		return nil, nil
	})
	cmd := newCmdBranchDelete(branchFactory(branchCommandConfig{token: "token"}, transport))
	cmd.SetIn(strings.NewReader("no\n"))
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(cmd, []string{"alice/demo", "feature/foo"}); err != nil {
		t.Fatal(err)
	}
	if deletes != 0 {
		t.Fatalf("delete count = %d", deletes)
	}
	if !strings.Contains(out.String(), "cancelled") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestBranchDeleteRefusesDefaultAndProtectedBeforeDelete(t *testing.T) {
	tests := []struct {
		name       string
		branch     string
		repoBody   string
		branchBody string
		want       string
	}{
		{name: "default from repo", branch: "main", repoBody: `{"default_branch":"main"}`, branchBody: `{"name":"main","protected":false}`, want: "default branch"},
		{name: "default from branch", branch: "main", repoBody: `{"default_branch":"trunk"}`, branchBody: `{"name":"main","default":true,"protected":false}`, want: "default branch"},
		{name: "protected", branch: "release", repoBody: `{"default_branch":"main"}`, branchBody: `{"name":"release","protected":1}`, want: "protected branch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deletes := 0
			transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodDelete {
					deletes++
				}
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo":
					return branchResponse(http.StatusOK, tt.repoBody), nil
				case req.Method == http.MethodGet && strings.HasPrefix(req.URL.EscapedPath(), "/api/v5/repos/alice/demo/branches/"):
					return branchResponse(http.StatusOK, tt.branchBody), nil
				default:
					t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
				}
				return nil, nil
			})
			cmd := newCmdBranchDelete(branchFactory(branchCommandConfig{token: "token"}, transport))
			_ = cmd.Flags().Set("yes", "true")
			err := cmd.RunE(cmd, []string{"alice/demo", tt.branch})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
			if deletes != 0 {
				t.Fatalf("delete count = %d", deletes)
			}
		})
	}
}

func TestBranchDeleteReportsNotFoundAndPermissionErrors(t *testing.T) {
	tests := []struct {
		name       string
		failMethod string
		status     int
		want       string
	}{
		{name: "branch not found", failMethod: http.MethodGet, status: http.StatusNotFound, want: "failed to view branch"},
		{name: "delete forbidden", failMethod: http.MethodDelete, status: http.StatusForbidden, want: "failed to delete branch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo":
					return branchResponse(http.StatusOK, `{"default_branch":"main"}`), nil
				case req.Method == http.MethodGet && req.URL.EscapedPath() == "/api/v5/repos/alice/demo/branches/feature%2Ffoo":
					if tt.failMethod == http.MethodGet {
						return branchResponse(tt.status, `{"message":"missing"}`), nil
					}
					return branchResponse(http.StatusOK, `{"name":"feature/foo","protected":false}`), nil
				case req.Method == http.MethodDelete:
					return branchResponse(tt.status, `{"message":"denied"}`), nil
				default:
					t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
				}
				return nil, nil
			})
			cmd := newCmdBranchDelete(branchFactory(branchCommandConfig{token: "token"}, transport))
			_ = cmd.Flags().Set("yes", "true")
			err := cmd.RunE(cmd, []string{"alice/demo", "feature/foo"})
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), fmt.Sprint(tt.status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestBranchCommandsReportAuthenticationErrorsWithoutRequest(t *testing.T) {
	requests := 0
	factory := branchFactory(branchCommandConfig{tokenErr: errors.New("missing token")}, func(*http.Request) (*http.Response, error) {
		requests++
		return branchResponse(http.StatusOK, `{}`), nil
	})
	tests := []struct {
		name string
		call func() error
	}{
		{name: "list", call: func() error { return newCmdBranchList(factory).RunE(newCmdBranchList(factory), []string{"alice/demo"}) }},
		{name: "view", call: func() error {
			return newCmdBranchView(factory).RunE(newCmdBranchView(factory), []string{"alice/demo", "main"})
		}},
		{name: "create", call: func() error {
			cmd := newCmdBranchCreate(factory)
			_ = cmd.Flags().Set("ref", "main")
			return cmd.RunE(cmd, []string{"alice/demo", "feature"})
		}},
		{name: "delete", call: func() error {
			cmd := newCmdBranchDelete(factory)
			_ = cmd.Flags().Set("yes", "true")
			return cmd.RunE(cmd, []string{"alice/demo", "feature"})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil || !strings.Contains(err.Error(), "missing token") {
				t.Fatalf("error = %v", err)
			}
		})
	}
	if requests != 0 {
		t.Fatalf("request count = %d", requests)
	}
}

func TestParseRepositoryArg(t *testing.T) {
	if _, err := parseRepositoryArg("demo"); err == nil {
		t.Fatal("accepted repository without owner")
	}
	if _, err := parseRepositoryArg("alice/demo/extra"); err == nil {
		t.Fatal("accepted repository with extra path segment")
	}
	repository, err := parseRepositoryArg("alice/demo")
	if err != nil {
		t.Fatal(err)
	}
	if repository.Owner != "alice" || repository.Repo != "demo" {
		t.Fatalf("repository = %#v", repository)
	}
}
