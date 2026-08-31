package actions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
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

type contextBlockingBody struct {
	ctx     context.Context
	started chan struct{}
	once    sync.Once
}

func (b *contextBlockingBody) Read([]byte) (int, error) {
	b.once.Do(func() { close(b.started) })
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (*contextBlockingBody) Close() error { return nil }

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
	if _, err := client.ListJobs("team", "demo", "run-1", ListJobsOptions{}); err != nil {
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

func TestListJobsUsesPaginationQuery(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/repos/team/demo/actions/runs/run-1/jobs" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if got := req.URL.Query().Encode(); got != "page=2&per_page=25" {
			t.Fatalf("query = %q", got)
		}
		return response(req, http.StatusOK, `{"total_count":1,"jobs":[{"id":"job-1"}]}`), nil
	})
	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	result, err := client.ListJobs("team", "demo", "run-1", ListJobsOptions{Page: 2, PerPage: 25})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalCount != 1 || len(result.Jobs) != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestDeleteArtifactRequiresNoContent(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{name: "no content", status: http.StatusNoContent},
		{name: "forbidden", status: http.StatusForbidden, body: `{"message":"missing permission"}`, wantErr: true},
		{name: "not found", status: http.StatusNotFound, body: `{"message":"artifact not found"}`, wantErr: true},
		{name: "server error", status: http.StatusInternalServerError, body: `{"message":"backend failed"}`, wantErr: true},
		{name: "unexpected success", status: http.StatusOK, body: `{}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodDelete || req.URL.Path != "/api/v8/repos/team/demo/actions/artifacts/artifact-1" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if got := req.Header.Get("Authorization"); got != "Bearer secret" {
					t.Fatalf("Authorization = %q", got)
				}
				return response(req, tt.status, tt.body), nil
			})
			client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})

			err := client.DeleteArtifact("team", "demo", "artifact-1")
			if !tt.wantErr {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			var httpErr *HTTPError
			if !errors.As(err, &httpErr) || httpErr.StatusCode != tt.status {
				t.Fatalf("error type = %T (%v)", err, err)
			}
			if !strings.Contains(err.Error(), "delete artifact") {
				t.Fatalf("error = %q", err)
			}
		})
	}
}

func TestDeleteArtifactReportsTransportError(t *testing.T) {
	transportErr := errors.New("connection refused")
	client := NewClientWithHTTPClient("secret", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, transportErr
	})})

	err := client.DeleteArtifact("team", "demo", "artifact-1")
	if !errors.Is(err, transportErr) || !strings.Contains(err.Error(), "delete artifact") {
		t.Fatalf("error = %v", err)
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

func TestActionsContextCancelsStalledMetadataBody(t *testing.T) {
	readStarted := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	client := NewClientWithHTTPClient("secret", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := response(req, http.StatusOK, "")
		resp.Body = &contextBlockingBody{ctx: req.Context(), started: readStarted}
		return resp, nil
	})}).WithContext(ctx)
	result := make(chan error, 1)
	go func() {
		_, err := client.GetRun("team", "demo", "run-1")
		result <- err
	}()

	<-readStarted
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
		if !strings.Contains(err.Error(), "get workflow run: decode response") {
			t.Fatalf("error = %v, want operation and decode context", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Actions metadata decode did not stop promptly")
	}
}

func TestActionsMetadataTimeoutCoversStalledBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-req.Context().Done()
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = 40 * time.Millisecond
	client := newClientWithBaseURL("secret", server.URL+APIVersion, httpClient)
	started := time.Now()
	_, err := client.GetRun("team", "demo", "run-1")
	if err == nil {
		t.Fatal("expected metadata timeout")
	}
	if time.Since(started) > time.Second {
		t.Fatalf("metadata timeout took too long: %s", time.Since(started))
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
	if !strings.Contains(err.Error(), "get workflow run: decode response") {
		t.Fatalf("error = %v, want operation and decode context", err)
	}
}

func TestActionsStreamingDownloadIgnoresMetadataTimeout(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		reader, writer := io.Pipe()
		go func() {
			time.Sleep(40 * time.Millisecond)
			_, _ = io.WriteString(writer, "long-stream")
			_ = writer.Close()
		}()
		resp := response(req, http.StatusOK, "")
		resp.Body = reader
		return resp, nil
	})
	client := NewClientWithHTTPClient("secret", &http.Client{
		Timeout:   10 * time.Millisecond,
		Transport: transport,
	})

	resp, err := client.DownloadArtifact("team", "demo", "artifact-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read long stream: %v", err)
	}
	if string(body) != "long-stream" {
		t.Fatalf("body = %q", body)
	}
}

func TestActionsStreamingDownloadStopsOnContextCancellation(t *testing.T) {
	readStarted := make(chan struct{})
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := response(req, http.StatusOK, "")
		resp.Body = &contextBlockingBody{ctx: req.Context(), started: readStarted}
		return resp, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport}).WithContext(ctx)
	resp, err := client.DownloadJobLog("team", "demo", "run-1", "job-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	result := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(resp.Body)
		result <- err
	}()
	<-readStarted
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("stream read error = %v, want context canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not stop promptly after cancellation")
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
		if got := req.URL.Query().Encode(); got != "page=2&per_page=50" {
			t.Fatalf("query = %q", got)
		}
		body := `{"total_count":1,"workflows":[{"workflow_id":"wf-123","name":"CI","file_path":".atomgit/workflows/ci.yml","state":"active"}]}`
		return response(req, http.StatusOK, body), nil
	})

	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	res, err := client.ListWorkflows("team", "demo", ListWorkflowsOptions{Page: 2, PerPage: 50})
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

func TestListWorkflowsDefaultsToNoQuery(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.RawQuery != "" {
			t.Fatalf("query = %q, want empty", req.URL.RawQuery)
		}
		return response(req, http.StatusOK, `{"total_count":0,"workflows":[]}`), nil
	})

	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	if _, err := client.ListWorkflows("team", "demo", ListWorkflowsOptions{}); err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
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

func TestValidateWorkflowUsesBase64ContentAndBearerAuth(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v8/repos/team/demo/actions/workflows/validate" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q", got)
		}
		if req.URL.RawQuery != "" {
			t.Fatalf("query = %q, want empty", req.URL.RawQuery)
		}
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(bodyBytes), `"base64_content":"bmFtZTogY2kK"`) {
			t.Fatalf("unexpected body: %s", string(bodyBytes))
		}
		return response(req, http.StatusOK, `{"valid":true,"diagnostics":[]}`), nil
	})

	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	result, err := client.ValidateWorkflow("team", "demo", WorkflowValidationRequest{Base64Content: "bmFtZTogY2kK"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestValidateWorkflowDecodesInvalidDiagnostics(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return response(req, http.StatusOK, `{"valid":false,"diagnostics":[{"range":{"start":{"line":2,"column":1},"end":{"line":2,"column":1}},"severity":"Error","message":"expected node content"}]}`), nil
	})

	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	result, err := client.ValidateWorkflow("team", "demo", WorkflowValidationRequest{Base64Content: "bmFtZTogWwo="})
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid || len(result.Diagnostics) != 1 {
		t.Fatalf("result = %#v", result)
	}
	diag := result.Diagnostics[0]
	if diag.Severity != "Error" || diag.Message != "expected node content" || diag.Range.Start.Line != 2 || diag.Range.Start.Column != 1 {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestGetStepLogPostsPaginationFields(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v8/repos/team/demo/actions/runs/run-1/jobs/job-1/logs" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q", got)
		}
		if req.URL.RawQuery != "" {
			t.Fatalf("query = %q, want empty", req.URL.RawQuery)
		}
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(bodyBytes), `"step_id":"step-1"`) ||
			!strings.Contains(string(bodyBytes), `"offset":57`) ||
			!strings.Contains(string(bodyBytes), `"limit":1000`) ||
			!strings.Contains(string(bodyBytes), `"sort":"asc"`) {
			t.Fatalf("unexpected body: %s", string(bodyBytes))
		}
		return response(req, http.StatusOK, `{"has_more":true,"start_offset":57,"end_offset":178,"log":"next page\n"}`), nil
	})

	client := NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	result, err := client.GetStepLog("team", "demo", "run-1", "job-1", StepLogRequest{
		StepID: "step-1",
		Offset: 57,
		Limit:  1000,
		Sort:   "asc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasMore || result.StartOffset != 57 || result.EndOffset != 178 || result.Log != "next page\n" {
		t.Fatalf("result = %#v", result)
	}
}
