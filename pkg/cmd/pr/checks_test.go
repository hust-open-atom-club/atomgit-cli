package pr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestPRChecksCommandRegistration(t *testing.T) {
	cmd := NewCmdPR(&cmdutil.Factory{})
	checks, _, err := cmd.Find([]string{"checks"})
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"watch", "interval"} {
		if checks.Flags().Lookup(flag) == nil {
			t.Fatalf("checks flag %q was not registered", flag)
		}
	}
	if !strings.Contains(checks.Example, "ag pr checks") || !strings.Contains(checks.Long, "required-check") {
		t.Fatalf("checks help is incomplete: %s\n%s", checks.Long, checks.Example)
	}
}

func TestPRChecksFiltersCurrentHeadAndClassifiesStatus(t *testing.T) {
	tests := []struct {
		name      string
		prState   string
		status    string
		wantError string
	}{
		{name: "open PR completed", prState: "open", status: "COMPLETED"},
		{name: "closed fork PR ignored", prState: "closed", status: "IGNORED"},
		{name: "failed", prState: "open", status: "FAILED", wantError: "failed or were canceled"},
		{name: "canceled", prState: "open", status: "CANCELED", wantError: "failed or were canceled"},
		{name: "pending", prState: "open", status: "RUNNING", wantError: "still pending"},
		{name: "unknown is pending", prState: "open", status: "QUEUED", wantError: "still pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := checksFactory(func(req *http.Request) (*http.Response, error) {
				requests++
				switch req.URL.Path {
				case "/api/v5/repos/alice/demo/pulls/7":
					body := fmt.Sprintf(`{"state":%q,"head":{"sha":"current-sha","ref":"feature","repo":{"full_name":"fork-owner/demo"}}}`, tt.prState)
					return prResponse(http.StatusOK, body), nil
				case "/api/v8/repos/alice/demo/actions/runs":
					assertChecksQuery(t, req, "7", "1")
					body := fmt.Sprintf(`{"total_count":2,"workflow_runs":[{"workflow_run_id":"old-run","workflow_name":"Old CI","status":"FAILED","head_sha":"old-sha"},{"workflow_run_id":"current-run","workflow_name":"Current CI","status":%q,"head_branch":"feature","head_sha":"current-sha"}]}`, tt.status)
					return prResponse(http.StatusOK, body), nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
					return nil, nil
				}
			})
			cmd := newCmdPRChecks(factory)
			var output bytes.Buffer
			cmd.SetOut(&output)
			err := cmd.RunE(cmd, []string{"alice/demo", "7"})
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantError)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if requests != 2 {
				t.Fatalf("requests = %d, want 2", requests)
			}
			for _, want := range []string{"CHECK", "STATUS", "BRANCH", "COMMIT", "URL", "Current CI", tt.status, "feature", "current-sha", "https://atomgit.com/alice/demo/actions/runs/current-run"} {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("output missing %q:\n%s", want, output.String())
				}
			}
			if strings.Contains(output.String(), "Old CI") || strings.Contains(output.String(), "old-run") {
				t.Fatalf("output included a stale-head run:\n%s", output.String())
			}
		})
	}
}

func TestPRChecksPaginatesUntilCurrentHead(t *testing.T) {
	actionRequests := 0
	factory := checksFactory(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v5/repos/alice/demo/pulls/7" {
			return prResponse(http.StatusOK, `{"head":{"sha":"current-sha"}}`), nil
		}
		if req.URL.Path != "/api/v8/repos/alice/demo/actions/runs" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		actionRequests++
		assertChecksQuery(t, req, "7", fmt.Sprint(actionRequests))
		if actionRequests == 2 {
			return prResponse(http.StatusOK, `{"total_count":101,"workflow_runs":[{"workflow_run_id":"current","workflow_name":"Current CI","status":"COMPLETED","head_sha":"current-sha"}]}`), nil
		}
		runs := make([]actions.Run, 100)
		for index := range runs {
			runs[index] = actions.Run{WorkflowRunID: fmt.Sprintf("old-%d", index), HeadSHA: "old-sha", Status: "FAILED"}
		}
		body, err := json.Marshal(actions.RunListResponse{TotalCount: 101, WorkflowRuns: runs})
		if err != nil {
			t.Fatal(err)
		}
		return prResponse(http.StatusOK, string(body)), nil
	})
	cmd := newCmdPRChecks(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
	if actionRequests != 2 || !strings.Contains(output.String(), "Current CI") {
		t.Fatalf("action requests = %d, output = %s", actionRequests, output.String())
	}
}

func TestPRChecksWatchPollsUntilTerminal(t *testing.T) {
	actionRequests := 0
	factory := checksFactory(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v5/repos/alice/demo/pulls/7" {
			return prResponse(http.StatusOK, `{"head":{"sha":"current-sha"}}`), nil
		}
		actionRequests++
		status := "RUNNING"
		if actionRequests == 2 {
			status = "COMPLETED"
		}
		return prResponse(http.StatusOK, fmt.Sprintf(`{"total_count":1,"workflow_runs":[{"workflow_run_id":"run-1","workflow_name":"CI","status":%q,"head_sha":"current-sha"}]}`, status)), nil
	})
	cmd := newCmdPRChecks(factory)
	_ = cmd.Flags().Set("watch", "true")
	_ = cmd.Flags().Set("interval", "1ms")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
	if actionRequests != 2 || !strings.Contains(output.String(), "polling again") || !strings.Contains(output.String(), "COMPLETED") {
		t.Fatalf("action requests = %d, output = %s", actionRequests, output.String())
	}
}

func TestPRChecksWatchHonorsContextCancellation(t *testing.T) {
	factory := checksFactory(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v5/repos/alice/demo/pulls/7" {
			return prResponse(http.StatusOK, `{"head":{"sha":"current-sha"}}`), nil
		}
		return prResponse(http.StatusOK, `{"total_count":1,"workflow_runs":[{"workflow_run_id":"run-1","status":"RUNNING","head_sha":"current-sha"}]}`), nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := newCmdPRChecks(factory)
	cmd.SetContext(ctx)
	_ = cmd.Flags().Set("watch", "true")
	_ = cmd.Flags().Set("interval", time.Hour.String())
	err := cmd.RunE(cmd, []string{"alice/demo", "7"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
}

func TestPRChecksReportsMissingAndAPIErrors(t *testing.T) {
	tests := []struct {
		name       string
		prStatus   int
		prBody     string
		runsStatus int
		runsBody   string
		want       string
	}{
		{name: "PR not found", prStatus: http.StatusNotFound, prBody: `{}`, want: "failed to get PR #7"},
		{name: "missing head SHA", prStatus: http.StatusOK, prBody: `{}`, want: "did not include a head commit SHA"},
		{name: "no runs", prStatus: http.StatusOK, prBody: `{"head":{"sha":"current"}}`, runsStatus: http.StatusOK, runsBody: `{"total_count":0,"workflow_runs":[]}`, want: "no workflow runs found"},
		{name: "only stale runs", prStatus: http.StatusOK, prBody: `{"head":{"sha":"current"}}`, runsStatus: http.StatusOK, runsBody: `{"total_count":1,"workflow_runs":[{"workflow_run_id":"old","head_sha":"old"}]}`, want: "no workflow runs found"},
		{name: "actions forbidden", prStatus: http.StatusOK, prBody: `{"head":{"sha":"current"}}`, runsStatus: http.StatusForbidden, runsBody: `{"message":"denied"}`, want: "permission denied"},
		{name: "actions rate limited", prStatus: http.StatusOK, prBody: `{"head":{"sha":"current"}}`, runsStatus: http.StatusTooManyRequests, runsBody: `{"message":"slow down"}`, want: "rate limited"},
		{name: "malformed actions response", prStatus: http.StatusOK, prBody: `{"head":{"sha":"current"}}`, runsStatus: http.StatusOK, runsBody: `[`, want: "decode response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := checksFactory(func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.Path, "/pulls/") {
					return prResponse(tt.prStatus, tt.prBody), nil
				}
				return prResponse(tt.runsStatus, tt.runsBody), nil
			})
			cmd := newCmdPRChecks(factory)
			err := cmd.RunE(cmd, []string{"alice/demo", "7"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestPRChecksValidatesFlagsBeforeRequests(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		watch     bool
		interval  string
		wantError string
	}{
		{name: "zero interval", args: []string{"alice/demo", "7"}, watch: true, interval: "0s", wantError: "interval must be positive"},
		{name: "interval without watch", args: []string{"alice/demo", "7"}, interval: "1s", wantError: "requires --watch"},
		{name: "invalid PR number", args: []string{"alice/demo", "zero"}, wantError: "invalid PR number"},
		{name: "invalid repository", args: []string{"invalid", "7"}, wantError: "invalid repository format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdPRChecks(&cmdutil.Factory{
				Config: prTestConfig{},
				HttpClient: func() (*http.Client, error) {
					t.Fatal("HTTP client should not be created for invalid input")
					return nil, nil
				},
			})
			if tt.watch {
				_ = cmd.Flags().Set("watch", "true")
			}
			if tt.interval != "" {
				_ = cmd.Flags().Set("interval", tt.interval)
			}
			err := cmd.RunE(cmd, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func checksFactory(transport prRoundTripFunc) *cmdutil.Factory {
	return &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		},
	}
}

func assertChecksQuery(t *testing.T, req *http.Request, number, page string) {
	t.Helper()
	if req.Method != http.MethodGet || req.URL.Query().Get("pull_request_id") != number || req.URL.Query().Get("page") != page || req.URL.Query().Get("per_page") != "100" {
		t.Fatalf("request = %s %s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
	}
}
