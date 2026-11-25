package actions

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRunActionContract(t *testing.T) {
	for _, action := range []string{"stop", "rerun", "retry"} {
		for _, tc := range []struct {
			status int
			body   string
			ok     bool
		}{
			{200, `{"success":true}`, true},
			{200, `{"success":false}`, false}, {200, `{}`, false},
			{200, `null`, false}, {200, `[]`, false}, {200, ``, false},
			{200, `{"success":"true"}`, false}, {200, `{"success":`, false},
			{201, `{"success":true}`, false}, {202, `{"success":true}`, false},
			{204, ``, false}, {301, `{"success":true}`, false},
			{302, `{"success":true}`, false}, {303, `{"success":true}`, false},
			{307, `{"success":true}`, false}, {308, `{"success":true}`, false},
			{403, `{}`, false}, {404, `{}`, false}, {409, `{}`, false},
			{429, `{}`, false}, {503, `{}`, false},
		} {
			t.Run(fmt.Sprintf("%s/%d/%s", action, tc.status, tc.body), func(t *testing.T) {
				calls, redirects := 0, 0
				type key struct{}
				ctx := context.WithValue(t.Context(), key{}, "run-action")
				hc := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil },
					Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						if calls > 1 {
							t.Fatal("write replayed")
						}
						if r.Method != "POST" || r.URL.Path != "/api/v8/repos/team/demo/actions/runs/run-1/"+action || r.Context().Value(key{}) != "run-action" {
							t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
						}
						return &http.Response{StatusCode: tc.status, Header: http.Header{"Location": {"https://redirect.test/"}, "Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
					})}
				c := NewClientWithHTTPClient("fixture", hc).WithContext(ctx)
				var err error
				switch action {
				case "stop":
					err = c.StopRun("team", "demo", "run-1")
				case "rerun":
					err = c.RerunRun("team", "demo", "run-1")
				case "retry":
					err = c.RetryRunJobs("team", "demo", "run-1", []string{"job-1"})
				}
				if (err == nil) != tc.ok || calls != 1 || redirects != 0 {
					t.Fatalf("error=%v calls=%d redirects=%d", err, calls, redirects)
				}
				if err := hc.CheckRedirect(nil, nil); err != nil || redirects != 1 {
					t.Fatal("changed caller redirect policy")
				}
			})
		}
	}
}

func TestRetryRejectsEmptyJobIDsBeforeRequest(t *testing.T) {
	c := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("invalid request sent"); return nil, nil })})
	for _, ids := range [][]string{nil, {}, {""}, {"job-1", " "}} {
		if err := c.RetryRunJobs("team", "demo", "run-1", ids); err == nil {
			t.Fatalf("accepted %q", ids)
		}
	}
}
