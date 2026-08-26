package run

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type runTestConfig struct {
	token    string
	tokenErr error
	host     string
}

func (c runTestConfig) GetToken() (string, error) { return c.token, c.tokenErr }
func (c runTestConfig) GetUser() (string, error)  { return "alice", nil }
func (c runTestConfig) GetHost() string {
	if c.host != "" {
		return c.host
	}
	return "atomgit.com"
}

type runRoundTripFunc func(*http.Request) (*http.Response, error)

func (f runRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func runResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func runFactory(config runTestConfig, transport runRoundTripFunc) *cmdutil.Factory {
	factory := &cmdutil.Factory{Config: config}
	if transport != nil {
		factory.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return factory
}

func TestNewCmdRunRegistersCommandsAndFlags(t *testing.T) {
	cmd := NewCmdRun(&cmdutil.Factory{})
	for _, value := range []string{"artifact", "dispatch", "rerun", "cancel", "delete"} {
		if !strings.Contains(strings.ToLower(cmd.Long), value) {
			t.Errorf("run help does not mention %q: %s", value, cmd.Long)
		}
	}

	want := map[string][]string{
		"list":     {"actor", "branch", "end-time", "event", "limit", "pr", "start-time", "status", "workflow", "workflow-name"},
		"view":     {"artifact", "artifact-file", "job", "log", "log-file", "overwrite"},
		"step-log": {"output", "overwrite"},
		"artifact": {},
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

	artifact, _, err := cmd.Find([]string{"artifact"})
	if err != nil {
		t.Fatal(err)
	}
	deleteCmd, _, err := artifact.Find([]string{"delete"})
	if err != nil || deleteCmd.Name() != "delete" {
		t.Fatalf("artifact delete command: %v", err)
	}
	if deleteCmd.Flags().Lookup("yes") == nil {
		t.Error("artifact delete --yes flag was not registered")
	}
}

func TestRunListPaginatesFiltersAndHonorsLimit(t *testing.T) {
	allRuns := make([]actions.Run, 102)
	for i := range allRuns {
		allRuns[i] = actions.Run{
			WorkflowRunID: fmt.Sprintf("run-%d", i),
			WorkflowName:  "CI",
			Status:        "FAILED",
			HeadBranch:    "main",
			Event:         "Push",
			RunNumber:     i + 1,
		}
	}

	requests := 0
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/api/v8/repos/team/demo/actions/runs" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		query := req.URL.Query()
		if query.Get("page") != fmt.Sprint(requests) || query.Get("status") != "FAILED" || query.Get("event") != "Push" || query.Get("branch") != "main" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		if query.Get("per_page") != "100" || query.Get("startTime") != "1000" || query.Get("endTime") != "2000" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}

		page, err := strconv.Atoi(query.Get("page"))
		if err != nil {
			t.Fatal(err)
		}
		perPage, err := strconv.Atoi(query.Get("per_page"))
		if err != nil {
			t.Fatal(err)
		}
		start := (page - 1) * perPage
		end := min(start+perPage, len(allRuns))
		if start > len(allRuns) {
			start = len(allRuns)
		}

		data, err := json.Marshal(actions.RunListResponse{TotalCount: len(allRuns), WorkflowRuns: allRuns[start:end]})
		if err != nil {
			t.Fatal(err)
		}
		return runResponse(req, http.StatusOK, string(data)), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	cmd := newCmdRunList(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	for flag, value := range map[string]string{
		"limit": "101", "status": "failed", "event": "push", "branch": "main", "start-time": "1000", "end-time": "2000",
	} {
		if err := cmd.Flags().Set(flag, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmd.RunE(cmd, []string{"team/demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("request count = %d", requests)
	}
	output := out.String()
	if !strings.Contains(output, "STATUS") || !strings.Contains(output, "run-100") || strings.Contains(output, "run-101") {
		t.Fatalf("output = %s", output)
	}
	seen := make(map[string]struct{}, 101)
	for _, field := range strings.Fields(output) {
		if !strings.HasPrefix(field, "run-") {
			continue
		}
		if _, exists := seen[field]; exists {
			t.Fatalf("duplicate run ID %q in output", field)
		}
		seen[field] = struct{}{}
	}
	if len(seen) != 101 {
		t.Fatalf("unique run IDs = %d, want 101", len(seen))
	}
}

func TestRunListNormalizesMultilineTableCells(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return runResponse(req, http.StatusOK, `{"total_count":1,"workflow_runs":[{"workflow_run_id":"run-1","run_number":1,"title":"update:\tfile\n\nSigned-off-by: alice\u001b[31m","workflow_name":"CI","status":"COMPLETED","head_branch":"main","event":"Push"}]}`), nil
	})
	cmd := newCmdRunList(runFactory(runTestConfig{token: "secret"}, transport))
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"team/demo"}); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("table lines = %d, output = %q", len(lines), out.String())
	}
	if !strings.Contains(lines[1], "update: file Signed-off-by: alice [31m") {
		t.Fatalf("row = %q", lines[1])
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Fatalf("output contains terminal escape: %q", out.String())
	}
}

func TestRunListEmptyAndValidation(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return runResponse(req, http.StatusOK, `{"total_count":0,"workflow_runs":[]}`), nil
		})
		cmd := newCmdRunList(runFactory(runTestConfig{token: "secret"}, transport))
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := cmd.RunE(cmd, []string{"team/demo"}); err != nil {
			t.Fatal(err)
		}
		if out.String() != "No workflow runs found.\n" {
			t.Fatalf("output = %q", out.String())
		}
	})

	tests := []struct {
		name  string
		opts  listOptions
		repo  string
		match string
	}{
		{name: "limit", opts: listOptions{Limit: 0}, repo: "team/demo", match: "must be positive"},
		{name: "status", opts: listOptions{Limit: 1, Status: "success"}, repo: "team/demo", match: "invalid status"},
		{name: "event", opts: listOptions{Limit: 1, Event: "schedule"}, repo: "team/demo", match: "invalid event"},
		{name: "time range", opts: listOptions{Limit: 1, StartTime: 20, EndTime: 10}, repo: "team/demo", match: "start-time"},
		{name: "repository", opts: listOptions{Limit: 1}, repo: "demo", match: "expected owner/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdRunList(runFactory(runTestConfig{token: "secret"}, nil))
			err := runList(cmd, runFactory(runTestConfig{token: "secret"}, nil), tt.opts, tt.repo)
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRunListInfersRepositoryFromCurrentCheckout(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/repos/inferred/repo/actions/runs" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "1" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		return runResponse(req, http.StatusOK, `{"total_count":1,"workflow_runs":[{"workflow_run_id":"run-1","run_number":1,"status":"COMPLETED"}]}`), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "inferred", Name: "repo"}, nil
	}
	cmd := newCmdRunList(factory)
	_ = cmd.Flags().Set("limit", "1")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "run-1") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunListExplicitRepositoryTakesPrecedence(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/repos/explicit/repo/actions/runs" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return runResponse(req, http.StatusOK, `{"total_count":1,"workflow_runs":[{"workflow_run_id":"run-1","status":"COMPLETED"}]}`), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{}, errors.New("resolver must not be called")
	}
	cmd := newCmdRunList(factory)
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"explicit/repo"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRunCommandsResolveRepositoryBeforeAuthentication(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		new  func(*cmdutil.Factory) *cobra.Command
	}{
		{name: "list", args: nil, new: newCmdRunList},
		{name: "view", args: []string{"run-1"}, new: newCmdRunView},
	} {
		t.Run(test.name, func(t *testing.T) {
			factory := runFactory(runTestConfig{tokenErr: errors.New("missing token")}, nil)
			factory.RepositoryResolver = func() (cmdutil.Repository, error) {
				return cmdutil.Repository{}, errors.New("repository resolver failed")
			}
			cmd := test.new(factory)
			cmd.SetOut(io.Discard)
			cmd.SetArgs(test.args)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "repository resolver failed") {
				t.Fatalf("error = %v, want resolver error", err)
			}
			if strings.Contains(err.Error(), "missing token") {
				t.Fatalf("authentication happened before repository resolution: %v", err)
			}
		})
	}
}

func TestRunViewExplicitRepositoryTakesPrecedence(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if !strings.HasPrefix(req.URL.Path, "/api/v8/repos/explicit/repo/actions/") {
			t.Fatalf("path = %q", req.URL.Path)
		}
		switch req.URL.Path {
		case "/api/v8/repos/explicit/repo/actions/runs/run-1":
			return runResponse(req, http.StatusOK, `{"workflow_run_id":"run-1","status":"RUNNING","stages":[]}`), nil
		case "/api/v8/repos/explicit/repo/actions/runs/run-1/jobs":
			return runResponse(req, http.StatusOK, `{"total_count":0,"jobs":[]}`), nil
		case "/api/v8/repos/explicit/repo/actions/runs/run-1/artifacts":
			return runResponse(req, http.StatusOK, `{"total_count":0,"artifacts":[]}`), nil
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
			return nil, nil
		}
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{}, errors.New("resolver must not be called")
	}
	cmd := newCmdRunView(factory)
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"explicit/repo", "run-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRunViewDisplaysRunJobsStepsURLAndArtifacts(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/api/v8/repos/team/demo/actions/runs/run-1":
			return runResponse(req, http.StatusOK, `{"workflow_run_id":"run-1","run_number":7,"workflow_name":"CI","title":"Build\nmain\u001b[31m","status":"FAILED","event":"Push","head_branch":"main","head_sha":"abc123","actor":{"login":"alice"},"start_time":1700000000000,"end_time":1700000060000}`), nil
		case "/api/v8/repos/team/demo/actions/runs/run-1/jobs":
			return runResponse(req, http.StatusOK, `{"total_count":1,"jobs":[{"id":"job-1","name":"build","status":"FAILED","steps":[{"id":"step-1","name":"go test","status":"FAILED"}]}]}`), nil
		case "/api/v8/repos/team/demo/actions/runs/run-1/artifacts":
			if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
				t.Fatalf("artifact query = %q", req.URL.RawQuery)
			}
			return runResponse(req, http.StatusOK, `{"total_count":1,"artifacts":[{"id":"artifact-1","name":"coverage","size_bytes":2048}]}`), nil
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
			return nil, nil
		}
	})
	factory := runFactory(runTestConfig{token: "secret", host: "atomgit.com"}, transport)
	cmd := newCmdRunView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runView(cmd, factory, viewOptions{}, "team/demo", "run-1"); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"Run: run-1", "Number: #7", "Title: Build main [31m", "Status: FAILED", "Actor: alice",
		"https://atomgit.com/team/demo/actions/runs/run-1", "[FAILED] build (job-1)", "[FAILED] go test (step-1)", "coverage (artifact-1, 2.0 KiB)",
	} {
		if !strings.Contains(out.String(), value) {
			t.Fatalf("output missing %q:\n%s", value, out.String())
		}
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Fatalf("output contains terminal escape: %q", out.String())
	}
}

func TestRunViewHandlesRunningRunWithNoJobsOrArtifacts(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(req.URL.Path, "/jobs"):
			return runResponse(req, http.StatusOK, `{"total_count":0,"jobs":[]}`), nil
		case strings.HasSuffix(req.URL.Path, "/artifacts"):
			return runResponse(req, http.StatusOK, `{"total_count":0,"artifacts":[]}`), nil
		default:
			return runResponse(req, http.StatusOK, `{"workflow_run_id":"run-1","status":"RUNNING","stages":[]}`), nil
		}
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	cmd := newCmdRunView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runView(cmd, factory, viewOptions{}, "team/demo", "run-1"); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"Status: RUNNING", "Jobs: none", "Artifacts: none"} {
		if !strings.Contains(out.String(), value) {
			t.Fatalf("output missing %q: %s", value, out.String())
		}
	}
}

func TestRunViewSpecificJobUsesJobDetailEndpoint(t *testing.T) {
	requests := 0
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		switch req.URL.Path {
		case "/api/v8/repos/team/demo/actions/runs/run-1":
			return runResponse(req, http.StatusOK, `{"workflow_run_id":"run-1","status":"COMPLETED"}`), nil
		case "/api/v8/repos/team/demo/actions/runs/run-1/jobs/job-1":
			return runResponse(req, http.StatusOK, `{"id":"job-1","name":"test","status":"COMPLETED","steps":[{"name":"go test","status":"COMPLETED"}]}`), nil
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
			return nil, nil
		}
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	cmd := newCmdRunView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runView(cmd, factory, viewOptions{JobID: "job-1"}, "team/demo", "run-1"); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || !strings.Contains(out.String(), "[COMPLETED] test (job-1)") || strings.Contains(out.String(), "Artifacts:") {
		t.Fatalf("requests = %d, output = %s", requests, out.String())
	}
}

func TestRunViewMergesMissingJobsFromRunStages(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/api/v8/repos/team/demo/actions/runs/run-1":
			return runResponse(req, http.StatusOK, `{"workflow_run_id":"run-1","status":"FAILED","stages":[{"id":"stage-1","jobs":[{"id":"job-1","name":"stage build","status":"FAILED"},{"id":"job-2","name":"deploy","status":"FAILED"}]}]}`), nil
		case "/api/v8/repos/team/demo/actions/runs/run-1/jobs":
			return runResponse(req, http.StatusOK, `{"total_count":2,"jobs":[{"id":"job-1","name":"API build","status":"FAILED","steps":[{"name":"go test","status":"FAILED"}]}]}`), nil
		case "/api/v8/repos/team/demo/actions/runs/run-1/artifacts":
			return runResponse(req, http.StatusOK, `{"total_count":0,"artifacts":[]}`), nil
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
			return nil, nil
		}
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	cmd := newCmdRunView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := runView(cmd, factory, viewOptions{}, "team/demo", "run-1"); err != nil {
		t.Fatal(err)
	}
	output := out.String()
	if strings.Count(output, "(job-1)") != 1 || !strings.Contains(output, "[FAILED] API build (job-1)") {
		t.Fatalf("primary job was not preserved exactly once: %s", output)
	}
	if strings.Count(output, "(job-2)") != 1 || !strings.Contains(output, "[FAILED] deploy (job-2)") {
		t.Fatalf("stage job was not merged exactly once: %s", output)
	}
}

func TestListRunArtifactsPaginates(t *testing.T) {
	requests := 0
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		count := 100
		start := 0
		if requests == 2 {
			count = 1
			start = 100
		}
		if req.URL.Query().Get("page") != fmt.Sprint(requests) {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		artifacts := make([]actions.Artifact, count)
		for i := range artifacts {
			artifacts[i].ID = fmt.Sprintf("artifact-%d", start+i)
		}
		data, err := json.Marshal(actions.ArtifactListResponse{TotalCount: 101, Artifacts: artifacts})
		if err != nil {
			t.Fatal(err)
		}
		return runResponse(req, http.StatusOK, string(data)), nil
	})
	client := actions.NewClientWithHTTPClient("secret", &http.Client{Transport: transport})
	artifacts, err := listRunArtifacts(client, "team", "demo", "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || len(artifacts) != 101 || artifacts[100].ID != "artifact-100" {
		t.Fatalf("requests = %d, artifacts = %d", requests, len(artifacts))
	}
}

func TestRunViewJobLogStdout(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "zip archive", body: jobLogArchive(t, "line one", "line two\n"), want: "line one\nline two\n"},
		{name: "plain text compatibility", body: "line one\nline two\n", want: "line one\nline two\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if !strings.HasSuffix(req.URL.Path, "/jobs/job-1/download_log") {
					t.Fatalf("path = %s", req.URL.Path)
				}
				return runResponse(req, http.StatusOK, tt.body), nil
			})
			factory := runFactory(runTestConfig{token: "secret"}, transport)
			cmd := newCmdRunView(factory)
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := runView(cmd, factory, viewOptions{JobID: "job-1", Log: true}, "team/demo", "run-1"); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
			if strings.HasPrefix(out.String(), "PK") {
				t.Fatalf("ZIP bytes were written to stdout: %q", out.String())
			}
		})
	}
}

func TestRunViewDownloadsLogAndHonorsOverwrite(t *testing.T) {
	requests := 0
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return runResponse(req, http.StatusOK, fmt.Sprintf("log-%d", requests)), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	destination := filepath.Join(t.TempDir(), "job-logs.zip")

	cmd := newCmdRunView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runView(cmd, factory, viewOptions{JobID: "job-1", LogFile: destination}, "team/demo", "run-1"); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, destination, "log-1")

	err := runView(cmd, factory, viewOptions{JobID: "job-1", LogFile: destination}, "team/demo", "run-1")
	if err == nil || !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("error = %v", err)
	}
	assertFileContent(t, destination, "log-1")

	if err := runView(cmd, factory, viewOptions{JobID: "job-1", LogFile: destination, Overwrite: true}, "team/demo", "run-1"); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, destination, "log-2")
	if requests != 2 {
		t.Fatalf("request count = %d", requests)
	}
	if matches, err := filepath.Glob(filepath.Join(filepath.Dir(destination), ".job-logs.zip.tmp-*")); err != nil || len(matches) != 0 {
		t.Fatalf("temporary files = %v, %v", matches, err)
	}
}

func TestRunViewDownloadsArtifactAndChecksRun(t *testing.T) {
	t.Run("download", func(t *testing.T) {
		requests := 0
		transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			switch {
			case strings.HasSuffix(req.URL.Path, "/artifacts/artifact-1"):
				return runResponse(req, http.StatusOK, `{"id":"artifact-1","name":"build/output","workflow_run_id":"run-1"}`), nil
			case strings.HasSuffix(req.URL.Path, "/artifacts/artifact-1/zip"):
				return runResponse(req, http.StatusOK, "zip-content"), nil
			default:
				t.Fatalf("path = %s", req.URL.Path)
				return nil, nil
			}
		})
		factory := runFactory(runTestConfig{token: "secret"}, transport)
		destination := filepath.Join(t.TempDir(), "build.zip")
		cmd := newCmdRunView(factory)
		if err := runView(cmd, factory, viewOptions{ArtifactID: "artifact-1", ArtifactFile: destination}, "team/demo", "run-1"); err != nil {
			t.Fatal(err)
		}
		assertFileContent(t, destination, "zip-content")
		if requests != 2 {
			t.Fatalf("request count = %d", requests)
		}
	})

	t.Run("wrong run", func(t *testing.T) {
		requests := 0
		transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			return runResponse(req, http.StatusOK, `{"id":"artifact-1","workflow_run_id":"run-2"}`), nil
		})
		factory := runFactory(runTestConfig{token: "secret"}, transport)
		cmd := newCmdRunView(factory)
		err := runView(cmd, factory, viewOptions{ArtifactID: "artifact-1", ArtifactFile: filepath.Join(t.TempDir(), "build.zip")}, "team/demo", "run-1")
		if err == nil || !strings.Contains(err.Error(), "belongs to run run-2") {
			t.Fatalf("error = %v", err)
		}
		if requests != 1 {
			t.Fatalf("request count = %d", requests)
		}
	})

	t.Run("existing destination skips archive request", func(t *testing.T) {
		requests := 0
		transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			if strings.HasSuffix(req.URL.Path, "/artifacts/artifact-1/zip") {
				t.Fatal("artifact archive was requested for an existing destination")
			}
			return runResponse(req, http.StatusOK, `{"id":"artifact-1","name":"build","workflow_run_id":"run-1"}`), nil
		})
		factory := runFactory(runTestConfig{token: "secret"}, transport)
		destination := filepath.Join(t.TempDir(), "build.zip")
		if err := os.WriteFile(destination, []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := newCmdRunView(factory)
		err := runView(cmd, factory, viewOptions{ArtifactID: "artifact-1", ArtifactFile: destination}, "team/demo", "run-1")
		if err == nil || !strings.Contains(err.Error(), "--overwrite") {
			t.Fatalf("error = %v", err)
		}
		if requests != 1 {
			t.Fatalf("request count = %d, want metadata request only", requests)
		}
		assertFileContent(t, destination, "existing")
	})
}

func TestViewOptionValidation(t *testing.T) {
	tests := []struct {
		name  string
		opts  viewOptions
		match string
	}{
		{name: "two log modes", opts: viewOptions{JobID: "job", Log: true, LogFile: "job.log"}, match: "cannot be used together"},
		{name: "log without job", opts: viewOptions{Log: true}, match: "--job is required"},
		{name: "artifact file without artifact", opts: viewOptions{ArtifactFile: "build.zip"}, match: "--artifact is required"},
		{name: "artifact with job", opts: viewOptions{ArtifactID: "artifact", JobID: "job"}, match: "cannot be combined"},
		{name: "unused overwrite", opts: viewOptions{Overwrite: true}, match: "--overwrite requires"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateViewOptions(tt.opts)
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRunCommandsReportAuthenticationAndFactoryErrors(t *testing.T) {
	cmd := newCmdRunList(runFactory(runTestConfig{tokenErr: errors.New("missing token")}, nil))
	err := cmd.RunE(cmd, []string{"team/demo"})
	if err == nil || !strings.Contains(err.Error(), "missing token") {
		t.Fatalf("error = %v", err)
	}

	factory := &cmdutil.Factory{
		Config: runTestConfig{token: "secret"},
		HttpClient: func() (*http.Client, error) {
			return nil, errors.New("factory failed")
		},
	}
	view := newCmdRunView(factory)
	err = view.RunE(view, []string{"team/demo", "run-1"})
	if err == nil || !strings.Contains(err.Error(), "factory failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestArtifactFilenameIsSafe(t *testing.T) {
	tests := []struct {
		artifact actions.Artifact
		want     string
	}{
		{artifact: actions.Artifact{ID: "1", Name: "build"}, want: "build.zip"},
		{artifact: actions.Artifact{ID: "1", Name: "../release.zip"}, want: "release.zip"},
		{artifact: actions.Artifact{ID: "1", Name: `bad:name?.zip`}, want: "bad_name_.zip"},
		{artifact: actions.Artifact{ID: "1"}, want: "artifact-1.zip"},
	}
	for _, tt := range tests {
		if got := artifactFilename(tt.artifact); got != tt.want {
			t.Errorf("artifactFilename(%#v) = %q, want %q", tt.artifact, got, tt.want)
		}
	}
}

func jobLogArchive(t *testing.T, entries ...string) string {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for index, content := range entries {
		entry, err := archive.Create(fmt.Sprintf("%d.log", index))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.String()
}

func assertFileContent(t *testing.T, filename, want string) {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", filename, data, want)
	}
}

func decodeStepLogRequest(t *testing.T, req *http.Request) actions.StepLogRequest {
	t.Helper()
	var payload actions.StepLogRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestRunStepLogWritesSinglePageToStdout(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v8/repos/team/demo/actions/runs/run-1/jobs/job-1/logs" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.URL.RawQuery != "" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		payload := decodeStepLogRequest(t, req)
		if payload.StepID != "step-1" || payload.Offset != 0 || payload.Limit != 1000 || payload.Sort != "asc" {
			t.Fatalf("payload = %#v", payload)
		}
		return runResponse(req, http.StatusOK, `{"has_more":false,"start_offset":0,"end_offset":5,"log":"hello"}`), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	cmd := newCmdRunStepLog(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"team/demo", "run-1", "job-1", "step-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "hello" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunStepLogConcatenatesPages(t *testing.T) {
	var offsets []int64
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		payload := decodeStepLogRequest(t, req)
		offsets = append(offsets, payload.Offset)
		switch payload.Offset {
		case 0:
			return runResponse(req, http.StatusOK, `{"has_more":true,"start_offset":0,"end_offset":5,"log":"hello"}`), nil
		case 5:
			return runResponse(req, http.StatusOK, `{"has_more":false,"start_offset":5,"end_offset":11,"log":" world"}`), nil
		default:
			t.Fatalf("unexpected offset %d", payload.Offset)
			return nil, nil
		}
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	cmd := newCmdRunStepLog(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"team/demo", "run-1", "job-1", "step-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "hello world" {
		t.Fatalf("output = %q", out.String())
	}
	if len(offsets) != 2 || offsets[0] != 0 || offsets[1] != 5 {
		t.Fatalf("offsets = %#v", offsets)
	}
}

func TestRunStepLogAbortsStalledCursor(t *testing.T) {
	requests := 0
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return runResponse(req, http.StatusOK, `{"has_more":true,"start_offset":0,"end_offset":0,"log":"stuck"}`), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	cmd := newCmdRunStepLog(factory)
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"team/demo", "run-1", "job-1", "step-1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "pagination stalled") {
		t.Fatalf("error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestRunStepLogWritesFileAndRefusesOverwrite(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return runResponse(req, http.StatusOK, `{"has_more":false,"start_offset":0,"end_offset":5,"log":"saved"}`), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	destination := filepath.Join(t.TempDir(), "step.log")

	cmd := newCmdRunStepLog(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"team/demo", "run-1", "job-1", "step-1", "--output", destination})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, destination, "saved")
	if !strings.Contains(out.String(), destination) {
		t.Fatalf("output = %q", out.String())
	}

	requests := 0
	blocked := runFactory(runTestConfig{token: "secret"}, func(req *http.Request) (*http.Response, error) {
		requests++
		return runResponse(req, http.StatusOK, `{"has_more":false,"start_offset":0,"end_offset":7,"log":"changed"}`), nil
	})
	cmd = newCmdRunStepLog(blocked)
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"team/demo", "run-1", "job-1", "step-1", "--output", destination})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
	assertFileContent(t, destination, "saved")
}

func TestRunStepLogOutputOmitsFileWhenLaterPageFails(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		payload := decodeStepLogRequest(t, req)
		switch payload.Offset {
		case 0:
			return runResponse(req, http.StatusOK, `{"has_more":true,"start_offset":0,"end_offset":5,"log":"hello"}`), nil
		case 5:
			return runResponse(req, http.StatusInternalServerError, `{"message":"backend failed"}`), nil
		default:
			t.Fatalf("unexpected offset %d", payload.Offset)
			return nil, nil
		}
	})
	dir := t.TempDir()
	destination := filepath.Join(dir, "step.log")
	cmd := newCmdRunStepLog(runFactory(runTestConfig{token: "secret"}, transport))
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"team/demo", "run-1", "job-1", "step-1", "--output", destination})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "failed to get step log") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatalf("destination still exists: %v", statErr)
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("leftover files = %v", names)
	}
}

func TestRunStepLogOutputReturnsWhenDestinationIsUnwritable(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0o755)
	})
	probe := filepath.Join(dir, "probe")
	if file, err := os.Create(probe); err == nil {
		_ = file.Close()
		_ = os.Remove(probe)
		t.Skip("process can write to a mode 0555 directory")
	}

	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return runResponse(req, http.StatusOK, `{"has_more":false,"start_offset":0,"end_offset":5,"log":"hello"}`), nil
	})
	destination := filepath.Join(dir, "step.log")
	cmd := newCmdRunStepLog(runFactory(runTestConfig{token: "secret"}, transport))
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"team/demo", "run-1", "job-1", "step-1", "--output", destination})

	done := make(chan error, 1)
	go func() {
		done <- cmd.Execute()
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "write step log") {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("command hung after destination write failure")
	}
}

func TestRunStepLogInfersRepositoryAndRequiresAuth(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/repos/inferred/repo/actions/runs/run-1/jobs/job-1/logs" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return runResponse(req, http.StatusOK, `{"has_more":false,"start_offset":0,"end_offset":3,"log":"ok"}`), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "inferred", Name: "repo"}, nil
	}
	cmd := newCmdRunStepLog(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"run-1", "job-1", "step-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "ok" {
		t.Fatalf("output = %q", out.String())
	}

	unauth := newCmdRunStepLog(runFactory(runTestConfig{tokenErr: errors.New("missing token")}, nil))
	unauth.SetOut(io.Discard)
	unauth.SetArgs([]string{"team/demo", "run-1", "job-1", "step-1"})
	err := unauth.Execute()
	if err == nil || !strings.Contains(err.Error(), "missing token") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunStepLogRequiresIDs(t *testing.T) {
	cmd := newCmdRunStepLog(runFactory(runTestConfig{token: "secret"}, nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"team/demo", "run-1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected argument error")
	}
}

func TestRunArtifactViewTextAndJSON(t *testing.T) {
	body := `{"id":"artifact-1","name":"coverage","size_bytes":2048,"workflow_id":"wf-1","workflow_run_id":"run-1","digest":"sha256:abc","created_at":1700000000000,"updated_at":1700000060000,"expires_at":1700086400000}`
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v8/repos/team/demo/actions/artifacts/artifact-1" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return runResponse(req, http.StatusOK, body), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	cmd := newCmdRunArtifactView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"team/demo", "artifact-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"ID: artifact-1", "Name: coverage", "Size: 2.0 KiB", "Workflow: wf-1", "Run: run-1", "Digest: sha256:abc",
	} {
		if !strings.Contains(out.String(), value) {
			t.Fatalf("output missing %q:\n%s", value, out.String())
		}
	}

	jsonCmd := newCmdRunArtifactView(factory)
	out.Reset()
	jsonCmd.SetOut(&out)
	jsonCmd.SetArgs([]string{"team/demo", "artifact-1", "--json"})
	if err := jsonCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"id": "artifact-1"`) || !strings.Contains(out.String(), `"digest": "sha256:abc"`) {
		t.Fatalf("json output = %q", out.String())
	}
}

func TestRunArtifactViewMissingArtifactAndAuth(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return runResponse(req, http.StatusNotFound, `{"message":"artifact not found"}`), nil
	})
	cmd := newCmdRunArtifactView(runFactory(runTestConfig{token: "secret"}, transport))
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"team/demo", "missing"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "failed to get artifact") {
		t.Fatalf("error = %v", err)
	}

	unauth := newCmdRunArtifactView(runFactory(runTestConfig{tokenErr: errors.New("missing token")}, nil))
	unauth.SetOut(io.Discard)
	unauth.SetArgs([]string{"team/demo", "artifact-1"})
	err = unauth.Execute()
	if err == nil || !strings.Contains(err.Error(), "missing token") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunArtifactViewInfersRepository(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/repos/inferred/repo/actions/artifacts/artifact-1" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return runResponse(req, http.StatusOK, `{"id":"artifact-1","name":"logs","size_bytes":10}`), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "inferred", Name: "repo"}, nil
	}
	cmd := newCmdRunArtifactView(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"artifact-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ID: artifact-1") || !strings.Contains(out.String(), "Name: logs") {
		t.Fatalf("output = %q", out.String())
	}
}
