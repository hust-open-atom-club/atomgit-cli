package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func executeRunnerJobHistoryCommand(t *testing.T, factory *cmdutil.Factory, args ...string) (string, error) {
	t.Helper()
	cmd := NewCmdOrg(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(append([]string{"runner-job-history"}, args...))
	err := cmd.Execute()
	return output.String(), err
}

func testJobHistoryItem(id string) actions.JobHistory {
	return actions.JobHistory{
		JobRunID:        "run-" + id,
		JobID:           "job-" + id,
		JobName:         "job " + id,
		State:           "SUCCESS",
		RunsOn:          "ubuntu-latest",
		PipelineName:    "CI",
		PipelineRunID:   "pipeline-run-" + id,
		OrgNamespaceID:  "org-1",
		RepoNamespaceID: "repo-1",
		StartTime:       "2026-10-10 10:00:00",
		CompleteTime:    "2026-10-10 10:01:00",
		Duration:        60,
	}
}

func TestRunnerJobHistoryCommandRegistered(t *testing.T) {
	cmd := NewCmdOrg(&cmdutil.Factory{})
	subcommand, _, err := cmd.Find([]string{"runner-job-history"})
	if err != nil || subcommand == cmd {
		t.Fatalf("runner-job-history command missing: %v", err)
	}
	for _, flag := range []string{"start-time-begin", "start-time-end", "limit", "json"} {
		if subcommand.Flags().Lookup(flag) == nil {
			t.Fatalf("flag %q missing", flag)
		}
	}
	if err := subcommand.Args(subcommand, nil); err == nil {
		t.Fatal("command accepted no organization")
	}
	if err := subcommand.Args(subcommand, []string{"one", "two"}); err == nil {
		t.Fatal("command accepted two organizations")
	}
}

func TestRunnerJobHistoryTextAndJSON(t *testing.T) {
	body := `{"page_num":1,"page_size":30,"total":2,"page_count":1,"content":[
		{"state":"SUCCESS","duration":125,"org_namespace_id":"org-1","repo_namespace_id":"repo-1","job_run_id":"job-run-1","job_id":"job-1","job_external_id":"external-1","job_name":"build","pipeline_external_id":"pipeline-external-1","pipeline_origin_id":"pipeline-origin-1","pipeline_run_id":"pipeline-run-1","pipeline_name":"CI","runs_on":"ubuntu-latest","start_time":"2026-10-10 10:00:00","complete_time":"2026-10-10 10:02:05"},
		{"state":"RUNNING","duration":0,"org_namespace_id":"org-1","repo_namespace_id":"repo-1","job_run_id":"job-run-2","job_id":"job-2","job_external_id":"external-2","job_name":"pending","pipeline_external_id":"pipeline-external-2","pipeline_origin_id":"pipeline-origin-2","pipeline_run_id":"pipeline-run-2","pipeline_name":"CI","runs_on":"ubuntu-latest","start_time":"","complete_time":""}
	]}`
	transport := orgRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v8/orgs/team/actions/runner-resources/job-history" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.URL.Query().Encode() != "page=1&per_page=30" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		return orgResponse(http.StatusOK, body), nil
	})

	output, err := executeRunnerJobHistoryCommand(t, orgFactory(orgTestConfig{}, transport), "team")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"JOB RUN", "JOB ID", "JOB NAME", "STATE", "RUNS ON", "PIPELINE", "ORG", "REPO", "START", "COMPLETE", "DURATION", "job-run-1", "job-1", "build", "SUCCESS", "ubuntu-latest", "CI", "pipeline-run-1", "org-1", "repo-1", "2026-10-10 10:00:00", "2026-10-10 10:02:05", "125", "job-run-2", "pending", "RUNNING"} {
		if !strings.Contains(output, want) {
			t.Fatalf("text output missing %q:\n%s", want, output)
		}
	}
	var pendingLine string
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, "job-run-2") {
			pendingLine = line
			break
		}
	}
	if pendingLine == "" {
		t.Fatalf("pending job line missing:\n%s", output)
	}
	fields := strings.Fields(pendingLine)
	if len(fields) < 3 || fields[len(fields)-3] != "-" || fields[len(fields)-2] != "-" || fields[len(fields)-1] != "0" {
		t.Fatalf("missing time fields were not rendered as '-': %q", pendingLine)
	}

	output, err = executeRunnerJobHistoryCommand(t, orgFactory(orgTestConfig{}, transport), "team", "--json")
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, output, `{"total_count":2,"job_history":[
		{"state":"SUCCESS","duration":125,"org_namespace_id":"org-1","repo_namespace_id":"repo-1","job_run_id":"job-run-1","job_id":"job-1","job_external_id":"external-1","job_name":"build","pipeline_external_id":"pipeline-external-1","pipeline_origin_id":"pipeline-origin-1","pipeline_run_id":"pipeline-run-1","pipeline_name":"CI","runs_on":"ubuntu-latest","start_time":"2026-10-10 10:00:00","complete_time":"2026-10-10 10:02:05"},
		{"state":"RUNNING","duration":0,"org_namespace_id":"org-1","repo_namespace_id":"repo-1","job_run_id":"job-run-2","job_id":"job-2","job_external_id":"external-2","job_name":"pending","pipeline_external_id":"pipeline-external-2","pipeline_origin_id":"pipeline-origin-2","pipeline_run_id":"pipeline-run-2","pipeline_name":"CI","runs_on":"ubuntu-latest","start_time":"","complete_time":""}
	]}`)
}

func TestRunnerJobHistoryTimeFilters(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want map[string]string
	}{
		{name: "none", want: map[string]string{"page": "1", "per_page": "30"}},
		{name: "begin only", args: []string{"--start-time-begin", " 2026-10-01 00:00:00 "}, want: map[string]string{"page": "1", "per_page": "30", "start_time_begin": "2026-10-01 00:00:00"}},
		{name: "end only", args: []string{"--start-time-end", "2026-10-31 23:59:59"}, want: map[string]string{"page": "1", "per_page": "30", "start_time_end": "2026-10-31 23:59:59"}},
		{name: "both", args: []string{"--start-time-begin", "2026-10-01 00:00:00", "--start-time-end", "2026-10-31 23:59:59"}, want: map[string]string{"page": "1", "per_page": "30", "start_time_begin": "2026-10-01 00:00:00", "start_time_end": "2026-10-31 23:59:59"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := orgRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				query := req.URL.Query()
				if len(query) != len(tt.want) {
					t.Fatalf("query = %q, want %#v", req.URL.RawQuery, tt.want)
				}
				for key, value := range tt.want {
					if got := query.Get(key); got != value {
						t.Fatalf("query %s = %q, want %q", key, got, value)
					}
				}
				return orgResponse(http.StatusOK, `{"page_num":1,"page_size":30,"total":0,"page_count":0,"content":[]}`), nil
			})
			output, err := executeRunnerJobHistoryCommand(t, orgFactory(orgTestConfig{}, transport), append([]string{"team"}, tt.args...)...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output, `No runner job history found for organization "team".`) {
				t.Fatalf("output = %q", output)
			}
		})
	}
}

func TestRunnerJobHistoryValidatesBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "organization", args: []string{"bad/org"}, want: "invalid organization"},
		{name: "blank organization", args: []string{" "}, want: "invalid organization"},
		{name: "control character", args: []string{"team\nother"}, want: "invalid organization"},
		{name: "limit zero", args: []string{"team", "--limit", "0"}, want: "must be positive"},
		{name: "limit negative", args: []string{"team", "--limit", "-1"}, want: "must be positive"},
		{name: "invalid begin", args: []string{"team", "--start-time-begin", "2026-10-01"}, want: "invalid --start-time-begin"},
		{name: "invalid end", args: []string{"team", "--start-time-end", "2026-10-01T00:00:00"}, want: "invalid --start-time-end"},
		{name: "reversed", args: []string{"team", "--start-time-begin", "2026-10-31 00:00:00", "--start-time-end", "2026-10-01 00:00:00"}, want: "must not be after"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			factory := orgFactory(orgTestConfig{tokenErr: config.ErrNotAuthenticated, tokenCalls: &calls}, nil)
			_, err := executeRunnerJobHistoryCommand(t, factory, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if calls != 0 {
				t.Fatalf("GetToken calls = %d, want 0", calls)
			}
		})
	}
}

func TestRunnerJobHistoryPaginationAndLimit(t *testing.T) {
	var perPageRequests []int
	transport := orgRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		page, _ := strconv.Atoi(req.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(req.URL.Query().Get("per_page"))
		perPageRequests = append(perPageRequests, perPage)
		start, count := 0, 0
		switch page {
		case 1:
			start, count = 0, 100
		case 2:
			start, count = 100, 20
		default:
			t.Fatalf("unexpected page %d", page)
		}
		items := make([]actions.JobHistory, count)
		for index := range items {
			items[index] = testJobHistoryItem(fmt.Sprintf("%03d", start+index))
		}
		data, err := json.Marshal(actions.JobHistoryListResponse{
			PageNum: page, PageSize: perPage, Total: 150, PageCount: 2, Content: items,
		})
		if err != nil {
			t.Fatal(err)
		}
		return orgResponse(http.StatusOK, string(data)), nil
	})

	output, err := executeRunnerJobHistoryCommand(t, orgFactory(orgTestConfig{}, transport), "team", "--limit", "120", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(perPageRequests) != "[100 20]" {
		t.Fatalf("per_page requests = %v, want [100 20]", perPageRequests)
	}
	if !strings.Contains(output, `"total_count": 120`) || !strings.Contains(output, `"run-119"`) || strings.Contains(output, `"run-120"`) {
		t.Fatalf("output = %s", output)
	}
}

func TestCollectJobHistoryPages(t *testing.T) {
	item1 := testJobHistoryItem("1")
	item2 := testJobHistoryItem("2")
	changed := item1
	changed.State = "FAILED"
	tests := []struct {
		name      string
		limit     int
		pages     []actions.JobHistoryListResponse
		wantItems int
		wantErr   string
	}{
		{
			name: "empty history", limit: 30,
			pages: []actions.JobHistoryListResponse{{PageNum: 1, PageSize: 30, Total: 0, PageCount: 0, Content: []actions.JobHistory{}}},
		},
		{
			name: "complete", limit: 30,
			pages: []actions.JobHistoryListResponse{
				{PageNum: 1, PageSize: 1, Total: 2, PageCount: 2, Content: []actions.JobHistory{item1}},
				{PageNum: 2, PageSize: 1, Total: 2, PageCount: 2, Content: []actions.JobHistory{item2}},
			},
			wantItems: 2,
		},
		{
			name: "limit truncates", limit: 1,
			pages:     []actions.JobHistoryListResponse{{PageNum: 1, PageSize: 2, Total: 3, PageCount: 2, Content: []actions.JobHistory{item1, item2}}},
			wantItems: 1,
		},
		{
			name: "incomplete empty page", limit: 30,
			pages: []actions.JobHistoryListResponse{
				{PageNum: 1, PageSize: 1, Total: 2, PageCount: 2, Content: []actions.JobHistory{item1}},
				{PageNum: 2, PageSize: 1, Total: 2, PageCount: 2, Content: []actions.JobHistory{}},
			},
			wantErr: "incomplete pagination",
		},
		{
			name: "repeated page", limit: 30,
			pages: []actions.JobHistoryListResponse{
				{PageNum: 1, PageSize: 1, Total: 2, PageCount: 2, Content: []actions.JobHistory{item1}},
				{PageNum: 2, PageSize: 1, Total: 2, PageCount: 2, Content: []actions.JobHistory{item1}},
			},
			wantErr: "pagination made no progress",
		},
		{
			name: "duplicate item keys", limit: 30,
			pages: []actions.JobHistoryListResponse{
				{PageNum: 1, PageSize: 1, Total: 2, PageCount: 2, Content: []actions.JobHistory{item1}},
				{PageNum: 2, PageSize: 1, Total: 2, PageCount: 2, Content: []actions.JobHistory{changed}},
			},
			wantErr: "pagination made no progress",
		},
		{
			name: "total changed", limit: 30,
			pages: []actions.JobHistoryListResponse{
				{PageNum: 1, PageSize: 1, Total: 2, PageCount: 2, Content: []actions.JobHistory{item1}},
				{PageNum: 2, PageSize: 1, Total: 3, PageCount: 3, Content: []actions.JobHistory{item2}},
			},
			wantErr: "API reported total 3 after 2",
		},
		{
			name: "page number mismatch", limit: 30,
			pages:   []actions.JobHistoryListResponse{{PageNum: 2, PageSize: 1, Total: 1, PageCount: 1, Content: []actions.JobHistory{item1}}},
			wantErr: "page_num 2",
		},
		{
			name: "page count mismatch", limit: 30,
			pages:   []actions.JobHistoryListResponse{{PageNum: 1, PageSize: 3, Total: 10, PageCount: 5, Content: []actions.JobHistory{item1, item2}}},
			wantErr: "page_count 5",
		},
		{
			name: "page size too small", limit: 30,
			pages:   []actions.JobHistoryListResponse{{PageNum: 1, PageSize: 1, Total: 2, PageCount: 2, Content: []actions.JobHistory{item1, item2}}},
			wantErr: "page_size 1",
		},
		{
			name: "total zero with content", limit: 30,
			pages:   []actions.JobHistoryListResponse{{PageNum: 1, PageSize: 2, Total: 0, PageCount: 0, Content: []actions.JobHistory{item1}}},
			wantErr: "total is 0",
		},
		{
			name: "negative metadata", limit: 30,
			pages:   []actions.JobHistoryListResponse{{PageNum: 1, PageSize: 1, Total: -1, PageCount: 1, Content: []actions.JobHistory{}}},
			wantErr: "negative metadata",
		},
		{
			name: "short page without page count", limit: 30,
			pages:   []actions.JobHistoryListResponse{{PageNum: 1, PageSize: 10, Total: 5, PageCount: 0, Content: []actions.JobHistory{item1, item2}}},
			wantErr: "incomplete pagination",
		},
		{
			name: "page count exhausted", limit: 30,
			pages: []actions.JobHistoryListResponse{
				{PageNum: 1, PageSize: 3, Total: 5, PageCount: 2, Content: []actions.JobHistory{item1, item2, testJobHistoryItem("3")}},
				{PageNum: 2, PageSize: 3, Total: 5, PageCount: 2, Content: []actions.JobHistory{testJobHistoryItem("4")}},
			},
			wantErr: "incomplete pagination",
		},
		{
			name: "invalid limit", limit: 0, wantErr: "must be positive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			items, err := collectJobHistoryPages(tt.limit, func(page, _ int) (actions.JobHistoryListResponse, error) {
				called = true
				if page < 1 || page > len(tt.pages) {
					return actions.JobHistoryListResponse{}, fmt.Errorf("unexpected page %d", page)
				}
				return tt.pages[page-1], nil
			})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != tt.wantItems {
				t.Fatalf("items = %d, want %d", len(items), tt.wantItems)
			}
			if tt.limit == 0 && called {
				t.Fatal("fetch was called for an invalid limit")
			}
		})
	}
}

func TestRunnerJobHistoryAPIErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   []string
	}{
		{name: "forbidden", status: http.StatusForbidden, body: `{"message":"denied"}`, want: []string{"failed to list runner job history", "permission denied", "403"}},
		{name: "not found", status: http.StatusNotFound, body: `{"message":"missing"}`, want: []string{"failed to list runner job history", "not found", "404"}},
		{name: "server error", status: http.StatusInternalServerError, body: `{"message":"boom"}`, want: []string{"failed to list runner job history", "500"}},
		{name: "malformed JSON", status: http.StatusOK, body: `{`, want: []string{"failed to list runner job history", "decode response"}},
		{name: "empty body", status: http.StatusOK, body: ``, want: []string{"failed to list runner job history", "decode response"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := orgRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return orgResponse(tt.status, tt.body), nil
			})
			_, err := executeRunnerJobHistoryCommand(t, orgFactory(orgTestConfig{}, transport), "team")
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %v, want substring %q", err, want)
				}
			}
		})
	}
}

func TestRunnerJobHistoryEmptyOutput(t *testing.T) {
	transport := orgRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, `{"page_num":1,"page_size":30,"total":0,"page_count":0,"content":[]}`), nil
	})
	output, err := executeRunnerJobHistoryCommand(t, orgFactory(orgTestConfig{}, transport), "team")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `No runner job history found for organization "team".`) {
		t.Fatalf("output = %q", output)
	}

	output, err = executeRunnerJobHistoryCommand(t, orgFactory(orgTestConfig{}, transport), "team", "--json")
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, output, `{"total_count":0,"job_history":[]}`)
}

func TestRunnerJobHistoryHonorsFactoryContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		return orgResponse(http.StatusOK, `{"page_num":1,"page_size":30,"total":0,"page_count":0,"content":[]}`), nil
	})
	factory.Context = func() context.Context { return ctx }

	_, err := executeRunnerJobHistoryCommand(t, factory, "team")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
}

func TestRunnerJobHistoryOutputUsesSanitizingWriter(t *testing.T) {
	transport := orgRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, `{"page_num":1,"page_size":30,"total":1,"page_count":1,"content":[{"job_run_id":"job-run-1","job_id":"job-1","job_name":"unsafe\u001b[31mname","state":"SUCCESS","start_time":"2026-10-10 10:00:00","complete_time":"2026-10-10 10:01:00","duration":60}]}`), nil
	})
	cmd := NewCmdOrg(orgFactory(orgTestConfig{}, transport))
	var output bytes.Buffer
	safe := cmdutil.NewSanitizingWriter(&output)
	cmd.SetOut(safe)
	cmd.SetErr(safe)
	cmd.SetArgs([]string{"runner-job-history", "team"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := safe.Flush(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "\x1b") || !strings.Contains(output.String(), `unsafe\x1b[31mname`) {
		t.Fatalf("output was not safely escaped: %q", output.String())
	}
}
