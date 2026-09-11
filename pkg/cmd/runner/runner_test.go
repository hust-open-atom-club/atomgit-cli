package runner

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type runnerTestConfig struct {
	token    string
	tokenErr error
}

func (c runnerTestConfig) GetToken() (string, error) { return c.token, c.tokenErr }
func (c runnerTestConfig) GetUser() (string, error)  { return "alice", nil }
func (c runnerTestConfig) GetHost() string           { return "atomgit.com" }

type runnerRoundTripFunc func(*http.Request) (*http.Response, error)

func (f runnerRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func runnerResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func newRunnerFactory(transport http.RoundTripper, token string) *cmdutil.Factory {
	f := &cmdutil.Factory{Config: runnerTestConfig{token: token}}
	if transport != nil {
		f.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return f
}

func TestRunnerListTextAndJSON(t *testing.T) {
	transport := runnerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/repos/owner/repo/actions/runners" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return runnerResponse(req, http.StatusOK, `{"total_count":1,"runners":[{"id":42,"name":"build-1","status":"online","busy":false,"labels":[{"name":"linux"}],"platform":"linux"}]}`), nil
	})

	t.Run("text", func(t *testing.T) {
		cmd := NewCmdRunner(newRunnerFactory(transport, "token"))
		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetArgs([]string{"list", "owner/repo"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"SOURCE", "repository", "42", "build-1", "false", "linux"} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("output %q missing %q", out.String(), want)
			}
		}
	})

	t.Run("json", func(t *testing.T) {
		cmd := NewCmdRunner(newRunnerFactory(transport, "token"))
		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetArgs([]string{"list", "owner/repo", "--json"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`"total_count": 1`, `"id": "42"`, `"busy": false`, `"platform": "linux"`} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("JSON output %q missing %q", out.String(), want)
			}
		}
	})
}

func TestRunnerListTextPreservesReturnedScope(t *testing.T) {
	cmd := NewCmdRunner(&cmdutil.Factory{})
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	if err := writeRunnerTable(cmd, "repository", []actions.Runner{
		{ID: "scoped", Name: "scoped-runner", Scope: "repository"},
		{ID: "unscoped", Name: "unscoped-runner"},
	}); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("output lines = %d, want 3: %q", len(lines), out.String())
	}
	header := strings.Fields(lines[0])
	if len(header) < 2 || header[0] != "SOURCE" || header[1] != "SCOPE" {
		t.Fatalf("header = %v, want SOURCE SCOPE first", header)
	}
	returned := strings.Fields(lines[1])
	if len(returned) < 2 || returned[1] != "repository" {
		t.Fatalf("returned-scope row = %v", returned)
	}
	absent := strings.Fields(lines[2])
	if len(absent) < 2 || absent[1] != "-" {
		t.Fatalf("absent-scope row = %v, want '-' scope", absent)
	}
}

func TestRunnerListPaginatesAndHonorsLimit(t *testing.T) {
	const total = 250
	requests := 0
	transport := runnerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/repos/owner/repo/actions/runners/shared-runners" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		requests++
		page, _ := strconv.Atoi(req.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(req.URL.Query().Get("per_page"))
		if perPage != maxRunnersPerPage {
			t.Fatalf("per_page = %d", perPage)
		}
		start := (page - 1) * perPage
		end := min(start+perPage, total)
		items := make([]string, 0, end-start)
		for i := start; i < end; i++ {
			items = append(items, fmt.Sprintf(`{"id":"runner-%d","name":"runner-%d"}`, i, i))
		}
		return runnerResponse(req, http.StatusOK, fmt.Sprintf(`{"total_count":%d,"runners":[%s]}`, total, strings.Join(items, ","))), nil
	})

	cmd := NewCmdRunner(newRunnerFactory(transport, "token"))
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetArgs([]string{"shared", "owner/repo", "--limit", "150", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if !strings.Contains(out.String(), `"total_count": 150`) || !strings.Contains(out.String(), `"id": "runner-149"`) || strings.Contains(out.String(), `"id": "runner-150"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestRunnerListRejectsIncompleteOrRepeatedPages(t *testing.T) {
	t.Run("incomplete total", func(t *testing.T) {
		transport := runnerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return runnerResponse(req, http.StatusOK, `{"total_count":3,"runners":[{"id":"r1"}]}`), nil
		})
		cmd := NewCmdRunner(newRunnerFactory(transport, "token"))
		cmd.SetArgs([]string{"list", "owner/repo"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "incomplete pagination") {
			t.Fatalf("error = %v, want incomplete pagination", err)
		}
	})

	t.Run("repeated full page", func(t *testing.T) {
		items := make([]string, 0, maxRunnersPerPage)
		for i := 0; i < maxRunnersPerPage; i++ {
			items = append(items, fmt.Sprintf(`{"id":"r%d"}`, i))
		}
		body := fmt.Sprintf(`{"total_count":200,"runners":[%s]}`, strings.Join(items, ","))
		transport := runnerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return runnerResponse(req, http.StatusOK, body), nil
		})
		cmd := NewCmdRunner(newRunnerFactory(transport, "token"))
		cmd.SetArgs([]string{"list", "owner/repo"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "pagination made no progress") {
			t.Fatalf("error = %v, want pagination made no progress", err)
		}
	})
}

func TestRunnerListRejectsChangingTotalCount(t *testing.T) {
	requests := 0
	transport := runnerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		page, _ := strconv.Atoi(req.URL.Query().Get("page"))
		if page == 1 {
			items := make([]string, 0, maxRunnersPerPage)
			for i := 0; i < maxRunnersPerPage; i++ {
				items = append(items, fmt.Sprintf(`{"id":"r%d"}`, i))
			}
			return runnerResponse(req, http.StatusOK, fmt.Sprintf(`{"total_count":200,"runners":[%s]}`, strings.Join(items, ","))), nil
		}
		return runnerResponse(req, http.StatusOK, `{"total_count":150,"runners":[{"id":"r100"}]}`), nil
	})

	cmd := NewCmdRunner(newRunnerFactory(transport, "token"))
	cmd.SetArgs([]string{"list", "owner/repo"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "inconsistent pagination") {
		t.Fatalf("error = %v, want inconsistent pagination", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestRunnerListRejectsTotalCountOverrun(t *testing.T) {
	requests := 0
	transport := runnerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		page, _ := strconv.Atoi(req.URL.Query().Get("page"))
		items := make([]string, 0, maxRunnersPerPage)
		start := (page - 1) * maxRunnersPerPage
		for i := start; i < start+maxRunnersPerPage; i++ {
			items = append(items, fmt.Sprintf(`{"id":"r%d"}`, i))
		}
		return runnerResponse(req, http.StatusOK, fmt.Sprintf(`{"total_count":150,"runners":[%s]}`, strings.Join(items, ","))), nil
	})

	cmd := NewCmdRunner(newRunnerFactory(transport, "token"))
	cmd.SetArgs([]string{"list", "owner/repo"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "inconsistent pagination") {
		t.Fatalf("error = %v, want inconsistent pagination", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestRunnerListValidationPrecedesAuthentication(t *testing.T) {
	f := newRunnerFactory(nil, "")
	f.Config = runnerTestConfig{tokenErr: config.ErrNotAuthenticated}
	cmd := NewCmdRunner(f)
	cmd.SetArgs([]string{"list", "invalid/repo/extra"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid repository format") {
		t.Fatalf("error = %v, want invalid repository format", err)
	}
}

func TestRunnerListRejectsNegativeLimit(t *testing.T) {
	cmd := NewCmdRunner(newRunnerFactory(nil, "token"))
	cmd.SetArgs([]string{"list", "owner/repo", "--limit", "-1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "must be zero or a positive integer") {
		t.Fatalf("error = %v", err)
	}
}
