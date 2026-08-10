package workflow

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type workflowTestConfig struct {
	token    string
	tokenErr error
}

func (c workflowTestConfig) GetToken() (string, error) { return c.token, c.tokenErr }
func (c workflowTestConfig) GetUser() (string, error)  { return "alice", nil }
func (c workflowTestConfig) GetHost() string           { return "atomgit.com" }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func response(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func newTestFactory(transport http.RoundTripper, token string) *cmdutil.Factory {
	f := &cmdutil.Factory{}
	if transport != nil {
		f.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	f.Config = workflowTestConfig{token: token}
	return f
}

func TestWorkflowList(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && req.URL.Path == "/api/v8/repos/owner/repo/actions/workflows" {
			body := `{"total_count":2,"workflows":[{"workflow_id":"wf-1","name":"CI","file_path":".atomgit/workflows/ci.yml","state":"active"},{"workflow_id":"wf-2","name":"Deploy","file_path":".atomgit/workflows/deploy.yml","state":"active"}]}`
			return response(req, http.StatusOK, body), nil
		}
		return response(req, http.StatusNotFound, `{"message":"not found"}`), nil
	})

	t.Run("table output", func(t *testing.T) {
		f := newTestFactory(transport, "test-token")
		cmd := NewCmdWorkflow(f)

		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetErr(out)
		cmd.SetArgs([]string{"list", "owner/repo"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := out.String()
		if !strings.Contains(output, "ID") || !strings.Contains(output, "wf-1") || !strings.Contains(output, "CI") {
			t.Fatalf("unexpected output: %q", output)
		}
	})

	t.Run("json output", func(t *testing.T) {
		f := newTestFactory(transport, "test-token")
		cmd := NewCmdWorkflow(f)

		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetErr(out)
		cmd.SetArgs([]string{"list", "owner/repo", "--json"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := out.String()
		if !strings.Contains(output, `"total_count": 2`) || !strings.Contains(output, `"workflow_id": "wf-1"`) {
			t.Fatalf("unexpected json output: %q", output)
		}
	})
}

func TestWorkflowListPaginatesAcrossPages(t *testing.T) {
	const total = 250
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v8/repos/owner/repo/actions/workflows" {
			return response(req, http.StatusNotFound, `{"message":"not found"}`), nil
		}
		page, _ := strconv.Atoi(req.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(req.URL.Query().Get("per_page"))
		if perPage != maxWorkflowsPerPage {
			t.Fatalf("per_page = %d, want %d", perPage, maxWorkflowsPerPage)
		}
		start := (page - 1) * perPage
		end := min(start+perPage, total)
		var workflows []string
		for i := start; i < end; i++ {
			workflows = append(workflows, fmt.Sprintf(`{"workflow_id":"wf-%d","name":"WF %d","file_path":".atomgit/workflows/wf-%d.yml","state":"active"}`, i, i, i))
		}
		body := fmt.Sprintf(`{"total_count":%d,"workflows":[%s]}`, total, strings.Join(workflows, ","))
		return response(req, http.StatusOK, body), nil
	})

	f := newTestFactory(transport, "test-token")
	cmd := NewCmdWorkflow(f)

	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"list", "owner/repo", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, `"total_count": 250`) ||
		!strings.Contains(output, `"workflow_id": "wf-0"`) ||
		!strings.Contains(output, `"workflow_id": "wf-249"`) {
		t.Fatalf("unexpected json output: %q", output)
	}
}

func TestWorkflowListEmpty(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return response(req, http.StatusOK, `{"total_count":0,"workflows":[]}`), nil
	})

	f := newTestFactory(transport, "test-token")
	cmd := NewCmdWorkflow(f)

	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"list", "owner/repo"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "No workflows found.") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestWorkflowRunSuccess(t *testing.T) {
	var postedBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && req.URL.Path == "/api/v8/repos/owner/repo/actions/workflows" {
			body := `{"total_count":1,"workflows":[{"workflow_id":"wf-123","name":"CI","file_path":".atomgit/workflows/ci.yml","state":"active"}]}`
			return response(req, http.StatusOK, body), nil
		}
		if req.Method == http.MethodPost && req.URL.Path == "/api/v8/repos/owner/repo/actions/workflows/wf-123/dispatches" {
			body, _ := io.ReadAll(req.Body)
			postedBody = string(body)
			return response(req, http.StatusNoContent, ""), nil
		}
		return response(req, http.StatusNotFound, `{"message":"not found"}`), nil
	})

	t.Run("dispatch by id with ref and inputs", func(t *testing.T) {
		postedBody = ""
		f := newTestFactory(transport, "test-token")
		cmd := NewCmdWorkflow(f)

		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetErr(out)
		cmd.SetArgs([]string{"run", "owner/repo", "wf-123", "--ref", "main", "-f", "env=production", "-F", "debug=true"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := out.String()
		if !strings.Contains(output, "Triggered workflow") || !strings.Contains(output, "wf-123") {
			t.Fatalf("unexpected output: %q", output)
		}

		if !strings.Contains(postedBody, `"ref":"main"`) || !strings.Contains(postedBody, `"env":"production"`) {
			t.Fatalf("unexpected posted body: %q", postedBody)
		}
	})

	t.Run("dispatch by name matching workflow", func(t *testing.T) {
		postedBody = ""
		f := newTestFactory(transport, "test-token")
		cmd := NewCmdWorkflow(f)

		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetErr(out)
		cmd.SetArgs([]string{"run", "owner/repo", "CI", "--ref", "dev"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := out.String()
		if !strings.Contains(output, "Triggered workflow \"CI\" (wf-123) on ref \"dev\"") {
			t.Fatalf("unexpected output: %q", output)
		}
	})
}

func TestWorkflowRunErrors(t *testing.T) {
	t.Run("unauthenticated error", func(t *testing.T) {
		f := newTestFactory(nil, "")
		f.Config = workflowTestConfig{tokenErr: fmt.Errorf("no token found")}
		cmd := NewCmdWorkflow(f)

		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetErr(out)
		cmd.SetArgs([]string{"run", "owner/repo", "wf-123"})

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not authenticated") {
			t.Fatalf("expected unauthenticated error, got: %v", err)
		}
	})

	t.Run("invalid field format", func(t *testing.T) {
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("unexpected network request for %s %s", req.Method, req.URL.Path)
		})
		f := newTestFactory(transport, "token")
		cmd := NewCmdWorkflow(f)

		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetErr(out)
		cmd.SetArgs([]string{"run", "owner/repo", "wf-123", "-f", "invalid_field_without_equals"})

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "invalid field format") {
			t.Fatalf("expected invalid field format error, got: %v", err)
		}
	})
}

func TestWorkflowRunResolvesDefaultBranch(t *testing.T) {
	var postedBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/owner/repo" {
			return response(req, http.StatusOK, `{"full_name":"owner/repo","default_branch":"trunk"}`), nil
		}
		if req.Method == http.MethodGet && req.URL.Path == "/api/v8/repos/owner/repo/actions/workflows" {
			body := `{"total_count":1,"workflows":[{"workflow_id":"wf-123","name":"CI","file_path":".atomgit/workflows/ci.yml","state":"active"}]}`
			return response(req, http.StatusOK, body), nil
		}
		if req.Method == http.MethodPost && req.URL.Path == "/api/v8/repos/owner/repo/actions/workflows/wf-123/dispatches" {
			body, _ := io.ReadAll(req.Body)
			postedBody = string(body)
			return response(req, http.StatusNoContent, ""), nil
		}
		return response(req, http.StatusNotFound, `{"message":"not found"}`), nil
	})

	f := newTestFactory(transport, "test-token")
	cmd := NewCmdWorkflow(f)

	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "owner/repo", "CI"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(postedBody, `"ref":"trunk"`) {
		t.Fatalf("expected dispatch on default branch trunk, got body: %q", postedBody)
	}
}

func TestWorkflowRunRejectsAmbiguousTarget(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && req.URL.Path == "/api/v8/repos/owner/repo/actions/workflows" {
			body := `{"total_count":2,"workflows":[{"workflow_id":"wf-1","name":"CI","file_path":".gitcode/workflows/ci.yml","state":"active"},{"workflow_id":"wf-2","name":"CI","file_path":".atomgit/workflows/ci.yml","state":"active"}]}`
			return response(req, http.StatusOK, body), nil
		}
		return response(req, http.StatusNotFound, `{"message":"not found"}`), nil
	})

	f := newTestFactory(transport, "test-token")
	cmd := NewCmdWorkflow(f)

	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "owner/repo", "CI", "--ref", "main"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got: %v", err)
	}
}

func TestWorkflowRunListFailureIsSurfaced(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && req.URL.Path == "/api/v8/repos/owner/repo/actions/workflows" {
			return response(req, http.StatusInternalServerError, `{"message":"workflow listing failed"}`), nil
		}
		return response(req, http.StatusNotFound, `{"message":"not found"}`), nil
	})

	f := newTestFactory(transport, "test-token")
	cmd := NewCmdWorkflow(f)

	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "owner/repo", "ci.yml", "--ref", "main"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "failed to list workflows") {
		t.Fatalf("expected workflow listing error, got: %v", err)
	}
}

func TestWorkflowRunOpaqueTargetDispatchesDirectly(t *testing.T) {
	var dispatchedPath string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && req.URL.Path == "/api/v8/repos/owner/repo/actions/workflows" {
			body := `{"total_count":1,"workflows":[{"workflow_id":"wf-1","name":"CI","file_path":".atomgit/workflows/ci.yml","state":"active"}]}`
			return response(req, http.StatusOK, body), nil
		}
		if req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/api/v8/repos/owner/repo/actions/workflows/") {
			dispatchedPath = req.URL.Path
			return response(req, http.StatusNoContent, ""), nil
		}
		return response(req, http.StatusNotFound, `{"message":"not found"}`), nil
	})

	f := newTestFactory(transport, "test-token")
	cmd := NewCmdWorkflow(f)

	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "owner/repo", "unknown-workflow", "--ref", "main"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/api/v8/repos/owner/repo/actions/workflows/unknown-workflow/dispatches"
	if dispatchedPath != want {
		t.Fatalf("dispatched path = %q, want %q", dispatchedPath, want)
	}
}
