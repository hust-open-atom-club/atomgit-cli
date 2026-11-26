package run

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type forbiddenRunConfig struct{ t *testing.T }

func (c forbiddenRunConfig) GetToken() (string, error) { c.t.Fatal("read credentials"); return "", nil }
func (c forbiddenRunConfig) GetUser() (string, error)  { c.t.Fatal("read identity"); return "", nil }
func (c forbiddenRunConfig) GetHost() string           { return "atomgit.com" }

func TestRunActionCancellation(t *testing.T) {
	for _, action := range []string{"cancel", "stop", "rerun", "retry"} {
		for _, input := range []string{"n\n", "\n", ""} {
			t.Run(action+"/"+input, func(t *testing.T) {
				f := &cmdutil.Factory{Config: forbiddenRunConfig{t}, HTTPClient: func() (*http.Client, error) { t.Fatal("created client"); return nil, nil }}
				cmd := NewCmdRun(f)
				var out, stderr bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&stderr)
				cmd.SetIn(strings.NewReader(input))
				cmd.SetArgs([]string{action, "team/demo", "run-1"})
				if err := cmd.Execute(); err != nil || out.Len() != 0 || !strings.Contains(stderr.String(), "Run action canceled.") {
					t.Fatalf("err=%v out=%q stderr=%q", err, out.String(), stderr.String())
				}
			})
		}
	}
}

func TestRunActionConfirmation(t *testing.T) {
	for _, action := range []string{"cancel", "stop", "rerun", "retry"} {
		calls := 0
		f := runFactory(runTestConfig{token: "fixture"}, runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return runResponse(req, 200, `{"success":true}`), nil
		}))
		cmd := NewCmdRun(f)
		var out, stderr bytes.Buffer
		cmd.SetIn(strings.NewReader("yes\n"))
		cmd.SetOut(&out)
		cmd.SetErr(&stderr)
		args := []string{action, "team/demo", "run-1"}
		if action == "retry" {
			args = append(args, "--job", "job-1")
		}
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil || calls != 1 || !strings.Contains(stderr.String(), "Repository: team/demo\nRun: run-1") {
			t.Fatalf("action=%s err=%v calls=%d stderr=%q", action, err, calls, stderr.String())
		}
	}
}

func TestRunRetryValidationBeforeCredentials(t *testing.T) {
	for _, id := range []string{"", " ", "build/linux"} {
		f := &cmdutil.Factory{Config: forbiddenRunConfig{t}}
		cmd := NewCmdRun(f)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"retry", "team/demo", "run-1", "--job=" + id, "--yes"})
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "invalid job ID") {
			t.Fatalf("id=%q error=%v", id, err)
		}
	}
}

func TestRunActions(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		method    string
		path      string
		wantIDs   []string
		wantReads int
	}{
		{"cancel", []string{"team/demo", "run-1"}, http.MethodPost, "/api/v8/repos/team/demo/actions/runs/run-1/stop", nil, 0},
		{"stop", []string{"team/demo", "run-1"}, http.MethodPost, "/api/v8/repos/team/demo/actions/runs/run-1/stop", nil, 0},
		{"rerun", []string{"team/demo", "run-1"}, http.MethodPost, "/api/v8/repos/team/demo/actions/runs/run-1/rerun", nil, 0},
		{"retry selected", []string{"team/demo", "run-1", "--job", "job-2", "--job", "job-2"}, http.MethodPost, "/api/v8/repos/team/demo/actions/runs/run-1/retry", []string{"job-2"}, 0},
		{"retry failed", []string{"team/demo", "run-1"}, http.MethodPost, "/api/v8/repos/team/demo/actions/runs/run-1/retry", []string{"job-1", "job-3"}, 1},
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
					return runResponse(req, http.StatusOK, `{"total_count":3,"jobs":[
						{"id":"job-1","exec_id":"exec-1","status":"FAILED"},
						{"id":"job-2","exec_id":"exec-2","status":"COMPLETED"},
						{"id":"job-3","status":"FAILED"}
					]}`), nil
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
					if !slices.Equal(payload.JobRunIDs, tt.wantIDs) {
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
			cmd.SetArgs(append(append([]string{strings.Fields(tt.name)[0]}, tt.args...), "--yes"))
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
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"team/demo", "run-1"}); err == nil || !strings.Contains(err.Error(), "did not report success") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunRetryWithoutFailedJobIDsDoesNotPost(t *testing.T) {
	tests := []struct {
		name    string
		jobs    string
		wantErr string
	}{
		{"no failed jobs", `{"total_count":0,"jobs":[]}`, "no failed jobs"},
		{"failed job missing ID", `{"total_count":2,"jobs":[
			{"id":"job-1","exec_id":"exec-1","status":"FAILED"},
			{"exec_id":"exec-2","status":"FAILED"}
		]}`, "failed job has no ID"},
		{"missing ID with name", `{"total_count":1,"jobs":[
			{"name":"Build\nLinux","identifier":"build-linux","status":"FAILED"}
		]}`, `failed job "Build Linux" has no ID`},
		{"missing ID with identifier", `{"total_count":1,"jobs":[
			{"name":"\u0000","identifier":"deploy","status":"FAILED"}
		]}`, `failed job "deploy" has no ID`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					t.Fatalf("unexpected %s request", req.Method)
				}
				return runResponse(req, http.StatusOK, tt.jobs), nil
			})
			cmd := newCmdRunRetry(runFactory(runTestConfig{token: "secret"}, transport))
			if err := cmd.Flags().Set("yes", "true"); err != nil {
				t.Fatal(err)
			}
			if err := cmd.RunE(cmd, []string{"team/demo", "run-1"}); err == nil || !strings.HasSuffix(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestRunRetryRejectsInvalidExplicitJobID(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected %s request", req.Method)
		return nil, nil
	})
	cmd := newCmdRunRetry(runFactory(runTestConfig{token: "secret"}, transport))
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("job", "build/linux"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"team/demo", "run-1"}); err == nil || err.Error() != `invalid job ID: "build/linux"` {
		t.Fatalf("error = %v", err)
	}
}
