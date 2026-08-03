package actions

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
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

func TestTimestampUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int64
	}{
		{name: "number", value: `1700000000123`, want: 1700000000123},
		{name: "quoted number", value: `"1700000000123"`, want: 1700000000123},
		{name: "RFC3339", value: `"2026-07-16T00:00:00Z"`, want: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC).UnixMilli()},
		{name: "null", value: `null`, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var timestamp Timestamp
			if err := timestamp.UnmarshalJSON([]byte(tt.value)); err != nil {
				t.Fatal(err)
			}
			if int64(timestamp) != tt.want {
				t.Fatalf("timestamp = %d, want %d", timestamp, tt.want)
			}
		})
	}

	var timestamp Timestamp
	if err := timestamp.UnmarshalJSON([]byte(`"not-a-time"`)); err == nil {
		t.Fatal("invalid timestamp was accepted")
	} else if !strings.Contains(err.Error(), "cannot parse") {
		t.Fatalf("invalid timestamp error = %v", err)
	}
}

func TestDefaultHTTPClientAllowsStreamingBodies(t *testing.T) {
	client := defaultHTTPClient()
	if client.Timeout != 0 {
		t.Fatalf("client timeout = %s", client.Timeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T", client.Transport)
	}
	if transport.ResponseHeaderTimeout != 30*time.Second {
		t.Fatalf("response header timeout = %s", transport.ResponseHeaderTimeout)
	}
}

func TestListRunsUsesV8PathFiltersAndBearerAuth(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v8/repos/team/demo/actions/runs" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q", got)
		}
		want := url.Values{
			"event":           {"Push"},
			"status":          {"FAILED"},
			"branch":          {"main"},
			"executor":        {"alice"},
			"pull_request_id": {"42"},
			"workflow_id":     {"workflow-1"},
			"workflow_name":   {"CI"},
			"page":            {"2"},
			"per_page":        {"50"},
			"startTime":       {"1000"},
			"endTime":         {"2000"},
		}
		if req.URL.Query().Encode() != want.Encode() {
			t.Fatalf("query = %q, want %q", req.URL.RawQuery, want.Encode())
		}
		return response(req, http.StatusOK, `{"total_count":1,"workflow_runs":[{"workflow_run_id":"run-1","status":"FAILED"}]}`), nil
	})
	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})

	result, err := client.ListRuns("team", "demo", ListRunsOptions{
		Event: "Push", Status: "FAILED", Branch: "main", Executor: "alice",
		PullRequestID: "42", WorkflowID: "workflow-1", WorkflowName: "CI",
		Page: 2, PerPage: 50, StartTime: 1000, EndTime: 2000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalCount != 1 || len(result.WorkflowRuns) != 1 || result.WorkflowRuns[0].WorkflowRunID != "run-1" {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunJobAndArtifactJSONPaths(t *testing.T) {
	expected := []string{
		"/api/v8/repos/team/demo/actions/runs/run-1",
		"/api/v8/repos/team/demo/actions/runs/run-1/jobs",
		"/api/v8/repos/team/demo/actions/runs/run-1/jobs/job-1",
		"/api/v8/repos/team/demo/actions/runs/run-1/artifacts",
		"/api/v8/repos/team/demo/actions/artifacts",
		"/api/v8/repos/team/demo/actions/artifacts/artifact-1",
	}
	request := 0
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if request >= len(expected) || req.URL.Path != expected[request] {
			t.Fatalf("request %d path = %q", request, req.URL.Path)
		}
		request++
		switch request {
		case 1:
			return response(req, http.StatusOK, `{"workflow_run_id":"run-1"}`), nil
		case 2:
			return response(req, http.StatusOK, `{"total_count":1,"jobs":[{"id":"job-1"}]}`), nil
		case 3:
			return response(req, http.StatusOK, `{"id":"job-1"}`), nil
		case 4, 5:
			if req.URL.Query().Get("page") != "2" || req.URL.Query().Get("per_page") != "25" || req.URL.Query().Get("sort") != "created" {
				t.Fatalf("artifact query = %q", req.URL.RawQuery)
			}
			return response(req, http.StatusOK, `{"total_count":1,"artifacts":[{"id":"artifact-1"}]}`), nil
		default:
			return response(req, http.StatusOK, `{"id":"artifact-1","created_at":"1700000000123"}`), nil
		}
	})
	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	artifactOptions := ListArtifactsOptions{Sort: "created", Page: 2, PerPage: 25}

	if _, err := client.GetRun("team", "demo", "run-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListJobs("team", "demo", "run-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetJob("team", "demo", "run-1", "job-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListRunArtifacts("team", "demo", "run-1", artifactOptions); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListArtifacts("team", "demo", artifactOptions); err != nil {
		t.Fatal(err)
	}
	artifact, err := client.GetArtifact("team", "demo", "artifact-1")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.ID != "artifact-1" || int64(artifact.CreatedAt) != 1700000000123 {
		t.Fatalf("artifact = %#v", artifact)
	}
	if request != len(expected) {
		t.Fatalf("request count = %d", request)
	}
}

func TestGetJobHandlesEmptySuccessResponse(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/repos/team/demo/actions/runs/run-1/jobs/missing-job" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return response(req, http.StatusOK, ""), nil
	})
	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})

	_, err := client.GetJob("team", "demo", "run-1", "missing-job")
	if err == nil || err.Error() != "get workflow run job: job missing-job not found (API returned an empty response)" {
		t.Fatalf("error = %v", err)
	}
}

func TestDownloadJobLogReturnsRawBody(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/repos/team/demo/actions/runs/run-1/jobs/job-1/download_log" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if got := req.Header.Get("Accept"); got != "*/*" {
			t.Fatalf("Accept = %q", got)
		}
		return response(req, http.StatusOK, "raw\x00log\n"), nil
	})
	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	resp, err := client.DownloadJobLog("team", "demo", "run-1", "job-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "raw\x00log\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestArtifactDownloadFollowsRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/api/v8/repos/team/demo/actions/artifacts/artifact-1/zip":
			http.Redirect(w, req, "/archive.zip", http.StatusFound)
		case "/archive.zip":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = io.WriteString(w, "zip-content")
		default:
			http.NotFound(w, req)
		}
	}))
	defer server.Close()

	client := newClientWithBaseURL("secret", server.URL+"/api/v8", server.Client())
	resp, err := client.DownloadArtifact("team", "demo", "artifact-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "zip-content" || resp.Request.URL.Path != "/archive.zip" {
		t.Fatalf("body = %q, final URL = %s", body, resp.Request.URL)
	}
}

func TestActionsErrorsAreDeterministic(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		retryAfter string
		contains   []string
	}{
		{name: "forbidden", status: http.StatusForbidden, body: `{"error_message":"missing permission"}`, contains: []string{"permission denied", "missing permission", "403"}},
		{name: "not found", status: http.StatusNotFound, body: `{"message":"missing run"}`, contains: []string{"not found", "missing run", "404"}},
		{name: "rate limited", status: http.StatusTooManyRequests, body: `{"error":"slow down"}`, retryAfter: "60", contains: []string{"rate limited", "slow down", "retry after 60", "429"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				resp := response(req, tt.status, tt.body)
				if tt.retryAfter != "" {
					resp.Header.Set("Retry-After", tt.retryAfter)
				}
				return resp, nil
			})
			client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
			_, err := client.GetRun("team", "demo", "run-1")
			if err == nil {
				t.Fatal("expected error")
			}
			var httpErr *HTTPError
			if !errors.As(err, &httpErr) || httpErr.StatusCode != tt.status {
				t.Fatalf("error type = %T (%v)", err, err)
			}
			for _, value := range tt.contains {
				if !strings.Contains(err.Error(), value) {
					t.Fatalf("error = %q, missing %q", err, value)
				}
			}
		})
	}
}

func TestResponseErrorPreservesBodyReadFailure(t *testing.T) {
	readErr := errors.New("connection reset")
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Status:     "500 Internal Server Error",
		Header:     make(http.Header),
		Body: io.NopCloser(io.MultiReader(
			strings.NewReader(`{"message":"server failed"}`),
			errorReader{err: readErr},
		)),
	}

	err := responseError("get workflow run", resp)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T (%v)", err, err)
	}
	if httpErr.Message != "server failed" {
		t.Fatalf("message = %q", httpErr.Message)
	}
	if !errors.Is(err, readErr) {
		t.Fatalf("error does not wrap read failure: %v", err)
	}
	for _, value := range []string{"server failed", "failed to read error response", "connection reset"} {
		if !strings.Contains(err.Error(), value) {
			t.Fatalf("error = %q, missing %q", err, value)
		}
	}
}

func TestListWorkflows(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v8/repos/team/demo/actions/workflows" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		body := `{"total_count":1,"workflows":[{"id":"wf-123","name":"CI","path":".atomgit/workflows/ci.yml","state":"active"}]}`
		return response(req, http.StatusOK, body), nil
	})

	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	res, err := client.ListWorkflows("team", "demo")
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}
	if res.TotalCount != 1 || len(res.Workflows) != 1 {
		t.Fatalf("unexpected result: %#v", res)
	}
	if res.Workflows[0].ID != "wf-123" || res.Workflows[0].Name != "CI" {
		t.Fatalf("unexpected workflow: %#v", res.Workflows[0])
	}
}

func TestCreateWorkflowDispatch(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v8/repos/team/demo/actions/workflows/wf-123/dispatches" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if contentType := req.Header.Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("Content-Type = %q", contentType)
		}
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(bodyBytes), `"ref":"main"`) || !strings.Contains(string(bodyBytes), `"env":"prod"`) {
			t.Fatalf("unexpected body: %s", string(bodyBytes))
		}
		return response(req, http.StatusNoContent, ""), nil
	})

	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	err := client.CreateWorkflowDispatch("team", "demo", "wf-123", WorkflowDispatchPayload{
		Ref:    "main",
		Inputs: map[string]string{"env": "prod"},
	})
	if err != nil {
		t.Fatalf("CreateWorkflowDispatch failed: %v", err)
	}
}
