package notification

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// roundTripFunc adapts a handler into an http.RoundTripper so command-level
// tests can serve canned API responses without touching the network.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func cannedJSON(status int, body string) *http.Response {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// newFlowFactory builds a Factory whose API client delegates to the given
// transport handler.
func newFlowFactory(transport roundTripFunc) *cmdutil.Factory {
	return &cmdutil.Factory{
		Config: &recordingConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		},
	}
}

const flowNotificationsJSON = `{"total":2,"list":[
	{"id":"ida","content":"first","type":"merge_requests_open","unread":true,"update_at":"2026-08-14T23:30:20+08:00","html_url":"https://example.test/a"},
	{"id":"idb","content":"second","type":"issue_open","unread":true,"update_at":"2026-08-14T22:30:20+08:00","html_url":"https://example.test/b"}
]}`

func TestNotificationMarkReadAllConfirmedSendsExactlyFetchedIDs(t *testing.T) {
	var putBody string
	factory := newFlowFactory(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return cannedJSON(http.StatusOK, flowNotificationsJSON), nil
		}
		raw, _ := io.ReadAll(req.Body)
		putBody = string(raw)
		return cannedJSON(http.StatusOK, ""), nil
	})

	cmd := newCmdNotificationMarkRead(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetIn(strings.NewReader("y\n"))

	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("set all: %v", err)
	}
	if err := cmd.RunE(cmd, []string{"owner/repo"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "Mark 2 unread notification(s) as read in owner/repo?") {
		t.Fatalf("prompt = %q", out.String())
	}
	if !strings.Contains(out.String(), "Marked 2 notification(s) as read") {
		t.Fatalf("output = %q", out.String())
	}
	if got := strings.Count(putBody, "ids="); got != 2 {
		t.Fatalf("PUT body has %d ids fields, want 2 (body %q)", got, putBody)
	}
	if !strings.Contains(putBody, "ids=ida") || !strings.Contains(putBody, "ids=idb") {
		t.Fatalf("PUT body = %q, want exactly the fetched IDs", putBody)
	}
}

func TestNotificationMarkReadAllDeclinedSendsNoPUT(t *testing.T) {
	var sawPUT bool
	factory := newFlowFactory(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPut {
			sawPUT = true
			return cannedJSON(http.StatusOK, ""), nil
		}
		return cannedJSON(http.StatusOK, flowNotificationsJSON), nil
	})

	cmd := newCmdNotificationMarkRead(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetIn(strings.NewReader("n\n"))

	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("set all: %v", err)
	}
	if err := cmd.RunE(cmd, []string{"owner/repo"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if sawPUT {
		t.Fatal("mark-read PUT ran after the user declined")
	}
	if !strings.Contains(out.String(), "Cancelled") {
		t.Fatalf("output = %q, want Cancelled", out.String())
	}
}

func TestNotificationMarkReadAllWithYesSkipsPrompt(t *testing.T) {
	var putBody string
	factory := newFlowFactory(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return cannedJSON(http.StatusOK, flowNotificationsJSON), nil
		}
		raw, _ := io.ReadAll(req.Body)
		putBody = string(raw)
		return cannedJSON(http.StatusOK, ""), nil
	})

	cmd := newCmdNotificationMarkRead(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetIn(strings.NewReader(""))

	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("set all: %v", err)
	}
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("set yes: %v", err)
	}
	if err := cmd.RunE(cmd, []string{"owner/repo"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if strings.Contains(out.String(), "[y/N]") {
		t.Fatalf("prompt shown despite --yes: %q", out.String())
	}
	if !strings.Contains(out.String(), "Marked 2 notification(s) as read") {
		t.Fatalf("output = %q", out.String())
	}
	if !strings.Contains(putBody, "ids=ida") || !strings.Contains(putBody, "ids=idb") {
		t.Fatalf("PUT body = %q", putBody)
	}
}

func TestNotificationMarkReadExplicitIDsSkipsListingAndPrompt(t *testing.T) {
	var method, path, body string
	factory := newFlowFactory(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut {
			t.Fatalf("unexpected %s %s; explicit IDs must not list first", req.Method, req.URL.Path)
		}
		method, path = req.Method, req.URL.Path
		raw, _ := io.ReadAll(req.Body)
		body = string(raw)
		return cannedJSON(http.StatusOK, ""), nil
	})

	cmd := newCmdNotificationMarkRead(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetIn(strings.NewReader(""))

	if err := cmd.RunE(cmd, []string{"owner/repo", "requested-id"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if method != http.MethodPut || path != "/api/v5/repos/owner/repo/notifications" {
		t.Fatalf("request = %s %s, want PUT /api/v5/repos/owner/repo/notifications", method, path)
	}
	if body != "ids=requested-id" {
		t.Fatalf("body = %q, want only the requested ID", body)
	}
	if strings.Contains(out.String(), "[y/N]") {
		t.Fatalf("unexpected confirmation prompt: %q", out.String())
	}
	if !strings.Contains(out.String(), "Marked 1 notification(s) as read") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestNotificationMarkReadAllWithoutUnread(t *testing.T) {
	factory := newFlowFactory(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("unexpected %s; nothing to mark read", req.Method)
		}
		return cannedJSON(http.StatusOK, `{"total":0,"list":[]}`), nil
	})

	cmd := newCmdNotificationMarkRead(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetIn(strings.NewReader("y\n"))

	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("set all: %v", err)
	}
	if err := cmd.RunE(cmd, []string{"owner/repo"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "No unread notifications") {
		t.Fatalf("output = %q, want no-unread state", out.String())
	}
}

func TestNotificationListJSONFlow(t *testing.T) {
	factory := newFlowFactory(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/owner/repo/notifications" {
			t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
		}
		return cannedJSON(http.StatusOK, flowNotificationsJSON), nil
	})

	cmd := newCmdNotificationList(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set json: %v", err)
	}
	if err := cmd.RunE(cmd, []string{"owner/repo"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	want := `[{"id":"ida","unread":true,"type":"merge_requests_open","subject":"first","updatedAt":"2026-08-14T23:30:20+08:00","url":"https://example.test/a"},` +
		`{"id":"idb","unread":true,"type":"issue_open","subject":"second","updatedAt":"2026-08-14T22:30:20+08:00","url":"https://example.test/b"}]`
	var compact bytes.Buffer
	if err := json.Compact(&compact, out.Bytes()); err != nil {
		t.Fatalf("compact output: %v", err)
	}
	if compact.String() != want {
		t.Fatalf("json output = %s, want %s", compact.String(), want)
	}
}

func TestNotificationListHumanFlow(t *testing.T) {
	factory := newFlowFactory(func(req *http.Request) (*http.Response, error) {
		return cannedJSON(http.StatusOK, flowNotificationsJSON), nil
	})

	cmd := newCmdNotificationList(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := cmd.RunE(cmd, []string{"owner/repo"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2 (got %q)", len(lines), out.String())
	}
	if !strings.HasPrefix(lines[0], "unread\tmerge_requests_open\t2026-08-14T23:30:20+08:00\tfirst\t") {
		t.Fatalf("first row = %q", lines[0])
	}
}
