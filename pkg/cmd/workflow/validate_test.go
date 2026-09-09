package workflow

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func writeWorkflowFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workflow.yml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWorkflowValidateRequiresFile(t *testing.T) {
	requests := 0
	f := newTestFactory(roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return nil, nil
	}), "token")
	cmd := NewCmdWorkflow(f)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"validate", "owner/repo"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--file is required") {
		t.Fatalf("error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestWorkflowValidateMissingFileDoesNotRequest(t *testing.T) {
	requests := 0
	f := newTestFactory(roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return nil, nil
	}), "token")
	cmd := NewCmdWorkflow(f)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"validate", "owner/repo", "--file", filepath.Join(t.TempDir(), "missing.yml")})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "read workflow file") {
		t.Fatalf("error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestWorkflowValidateMissingAuthentication(t *testing.T) {
	path := writeWorkflowFile(t, "name: ci\n")
	f := newTestFactory(nil, "")
	f.Config = workflowTestConfig{tokenErr: os.ErrPermission}
	cmd := NewCmdWorkflow(f)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"validate", "owner/repo", "--file", path})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error = %v", err)
	}
}

func TestWorkflowValidateRejectsInvalidRepositoryBeforeAuth(t *testing.T) {
	path := writeWorkflowFile(t, "name: ci\n")
	requests := 0
	f := newTestFactory(roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return nil, nil
	}), "")
	f.Config = workflowTestConfig{tokenErr: os.ErrPermission}
	cmd := NewCmdWorkflow(f)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"validate", "invalid", "--file", path})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid repository format") {
		t.Fatalf("error = %v, want repository validation to run before authentication", err)
	}
	if strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error = %v, authentication must not run before repository validation", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestWorkflowValidateValidFile(t *testing.T) {
	contents := "name: ci\n"
	path := writeWorkflowFile(t, contents)
	var gotBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v8/repos/owner/repo/actions/workflows/validate" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.URL.RawQuery != "" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		gotBody = string(body)
		return response(req, http.StatusOK, `{"valid":true,"diagnostics":[]}`), nil
	})
	f := newTestFactory(transport, "token")
	cmd := NewCmdWorkflow(f)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"validate", "owner/repo", "--file", path})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Workflow is valid.") {
		t.Fatalf("output = %q", out.String())
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(contents))
	if !strings.Contains(gotBody, `"base64_content":"`+encoded+`"`) {
		t.Fatalf("body = %q", gotBody)
	}
}

func TestWorkflowValidateInvalidFileJSON(t *testing.T) {
	path := writeWorkflowFile(t, "name: [\n")
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return response(req, http.StatusOK, `{"valid":false,"diagnostics":[{"range":{"start":{"line":2,"column":1},"end":{"line":2,"column":1}},"severity":"Error","message":"expected node content"}]}`), nil
	})
	f := newTestFactory(transport, "token")
	cmd := NewCmdWorkflow(f)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"validate", "owner/repo", "--file", path, "--json"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "workflow is invalid") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(out.String(), `"valid": false`) || !strings.Contains(out.String(), "expected node content") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestWorkflowValidateInvalidFileText(t *testing.T) {
	path := writeWorkflowFile(t, "name: [\n")
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return response(req, http.StatusOK, `{"valid":false,"diagnostics":[{"range":{"start":{"line":2,"column":1},"end":{"line":2,"column":1}},"severity":"Error","message":"expected node content"}]}`), nil
	})
	f := newTestFactory(transport, "token")
	cmd := NewCmdWorkflow(f)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"validate", "owner/repo", "--file", path})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "workflow is invalid") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(out.String(), "Workflow is invalid:") || !strings.Contains(out.String(), "Error: line 2, column 1: expected node content") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestWorkflowValidateInfersRepository(t *testing.T) {
	path := writeWorkflowFile(t, "name: ci\n")
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/repos/inferred/repo/actions/workflows/validate" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return response(req, http.StatusOK, `{"valid":true,"diagnostics":[]}`), nil
	})
	f := newTestFactory(transport, "token")
	f.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "inferred", Name: "repo"}, nil
	}
	cmd := NewCmdWorkflow(f)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"validate", "--file", path})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Workflow is valid.") {
		t.Fatalf("output = %q", out.String())
	}
}
