package run

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRunActions(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		method    string
		path      string
		wantIDs   []string
		wantReads int
	}{
		{"stop", []string{"team/demo", "run-1"}, http.MethodPost, "/api/v8/repos/team/demo/actions/runs/run-1/stop", nil, 0},
		{"rerun", []string{"team/demo", "run-1"}, http.MethodPost, "/api/v8/repos/team/demo/actions/runs/run-1/rerun", nil, 0},
		{"retry selected", []string{"team/demo", "run-1", "--job", "exec-2", "--job", "exec-2"}, http.MethodPost, "/api/v8/repos/team/demo/actions/runs/run-1/retry", []string{"exec-2"}, 0},
		{"retry failed", []string{"team/demo", "run-1"}, http.MethodPost, "/api/v8/repos/team/demo/actions/runs/run-1/retry", []string{"exec-1"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reads := 0
			posts := 0
			transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Query().Has("access_token") || req.Header.Get("Authorization") == "" {
					t.Fatalf("request credentials: %s %q", req.URL, req.Header.Get("Authorization"))
				}
				if req.Method == http.MethodGet {
					reads++
					if !strings.HasSuffix(req.URL.Path, "/jobs") {
						t.Fatalf("GET path: %s", req.URL.Path)
					}
					return runResponse(req, http.StatusOK, `{"total_count":2,"jobs":[{"id":"job-1","exec_id":"exec-1","status":"FAILED"},{"id":"job-2","exec_id":"exec-2","status":"COMPLETED"}]}`), nil
				}
				posts++
				if req.Method != tt.method || req.URL.Path != tt.path {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				var body []byte
				if req.Body != nil {
					var err error
					body, err = io.ReadAll(req.Body)
					if err != nil {
						t.Fatal(err)
					}
				}
				if tt.wantIDs != nil {
					var payload struct {
						JobRunIDs []string `json:"job_run_ids"`
					}
					if err := json.Unmarshal(body, &payload); err != nil {
						t.Fatal(err)
					}
					if len(payload.JobRunIDs) != len(tt.wantIDs) || payload.JobRunIDs[0] != tt.wantIDs[0] {
						t.Fatalf("payload = %s", body)
					}
				} else if len(body) != 0 {
					t.Fatalf("unexpected body: %s", body)
				}
				return runResponse(req, http.StatusOK, `{"success":true}`), nil
			})
			factory := runFactory(runTestConfig{token: "secret"}, transport)
			cmd := NewCmdRun(factory)
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(append([]string{strings.Fields(tt.name)[0]}, tt.args...))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if posts != 1 || reads != tt.wantReads || !strings.Contains(out.String(), "run-1") {
				t.Fatalf("posts=%d reads=%d output=%q", posts, reads, out.String())
			}
		})
	}
}

func TestRunActionRejectsFalseSuccess(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return runResponse(req, http.StatusOK, `{"success":false}`), nil
	})
	cmd := newCmdRunStop(runFactory(runTestConfig{token: "secret"}, transport))
	if err := cmd.RunE(cmd, []string{"team/demo", "run-1"}); err == nil || !strings.Contains(err.Error(), "did not report success") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunRetryWithoutFailedJobsDoesNotPost(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("unexpected %s request", req.Method)
		}
		return runResponse(req, http.StatusOK, `{"total_count":0,"jobs":[]}`), nil
	})
	cmd := newCmdRunRetry(runFactory(runTestConfig{token: "secret"}, transport))
	if err := cmd.RunE(cmd, []string{"team/demo", "run-1"}); err == nil || !strings.Contains(err.Error(), "no failed jobs") {
		t.Fatalf("error = %v", err)
	}
}
