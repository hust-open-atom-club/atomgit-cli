package issue

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type issueTestConfig struct{}

func (issueTestConfig) GetToken() (string, error) { return "token", nil }
func (issueTestConfig) GetUser() (string, error)  { return "alice", nil }
func (issueTestConfig) GetHost() string           { return "atomgit.com" }

type issueRoundTripFunc func(*http.Request) (*http.Response, error)

func (f issueRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestNewCmdIssueRegistersSubcommands(t *testing.T) {
	cmd := NewCmdIssue(&cmdutil.Factory{})
	want := map[string]bool{"close": false, "comment": false, "create": false, "list": false, "view": false}
	for _, child := range cmd.Commands() {
		if _, ok := want[child.Name()]; ok {
			want[child.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q was not registered", name)
		}
	}

	list, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if list.Flags().Lookup("state") == nil || list.Flags().Lookup("limit") == nil {
		t.Fatal("list flags were not registered")
	}
	if err := list.Args(list, []string{"one", "two"}); err == nil {
		t.Fatal("list accepted too many arguments")
	}
}

func TestIssueListHonorsLimit(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.URL.Path != "/api/v5/repos/alice/demo/issues" || req.URL.Query().Get("state") != "closed" || req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("request URL = %s", req.URL.String())
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"number":"1","title":"first","state":"closed"},{"number":"2","title":"second","state":"closed"}]`)), Header: make(http.Header)}, nil
			})}, nil
		},
	}
	cmd := newCmdIssueList(factory)
	_ = cmd.Flags().Set("state", "closed")
	_ = cmd.Flags().Set("limit", "1")
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestIssueListRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []string{"0", "-1"} {
		t.Run(limit, func(t *testing.T) {
			cmd := newCmdIssueList(&cmdutil.Factory{Config: issueTestConfig{}})
			_ = cmd.Flags().Set("limit", limit)
			if err := cmd.RunE(cmd, []string{"alice/demo"}); err == nil || !strings.Contains(err.Error(), "must be positive") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestIssueCloseUpdatesAndVerifiesState(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				switch requests {
				case 1:
					assertIssueCloseRequest(t, req, http.MethodGet, "/api/v5/repos/alice/demo/issues/1")
					return issueResponse(http.StatusOK, `{"number":"1","title":"Issue title","state":"open"}`), nil
				case 2:
					assertIssueCloseRequest(t, req, http.MethodPatch, "/api/v5/repos/alice/issues/1")
					if err := req.ParseMultipartForm(1 << 20); err != nil {
						t.Fatal(err)
					}
					for key, want := range map[string]string{"repo": "demo", "title": "Issue title", "state": "close"} {
						if got := req.FormValue(key); got != want {
							t.Errorf("form field %s = %q, want %q", key, got, want)
						}
					}
					return issueResponse(http.StatusOK, `{"number":"1","title":"Issue title","state":"closed"}`), nil
				case 3:
					assertIssueCloseRequest(t, req, http.MethodGet, "/api/v5/repos/alice/demo/issues/1")
					return issueResponse(http.StatusOK, `{"number":"1","title":"Issue title","state":"closed"}`), nil
				default:
					t.Fatalf("unexpected request #%d: %s %s", requests, req.Method, req.URL.String())
					return nil, nil
				}
			})}, nil
		},
	}

	cmd := newCmdIssueClose(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "1"}); err != nil {
		t.Fatal(err)
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
	if got := output.String(); got != "Closed issue #1\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestIssueCloseFailsWhenStateRemainsOpen(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				switch requests {
				case 1:
					return issueResponse(http.StatusOK, `{"number":"1","title":"Issue title","state":"open"}`), nil
				case 2:
					return issueResponse(http.StatusOK, `{"number":"1","title":"Issue title","state":"open"}`), nil
				case 3:
					return issueResponse(http.StatusOK, `{"number":"1","title":"Issue title","state":"open"}`), nil
				default:
					t.Fatalf("unexpected request #%d", requests)
					return nil, nil
				}
			})}, nil
		},
	}

	cmd := newCmdIssueClose(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	err := cmd.RunE(cmd, []string{"alice/demo", "1"})
	if err == nil || !strings.Contains(err.Error(), "still not closed") {
		t.Fatalf("error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("unexpected success output: %q", output.String())
	}
}

func assertIssueCloseRequest(t *testing.T, req *http.Request, method, path string) {
	t.Helper()
	if req.Method != method || req.URL.Path != path {
		t.Fatalf("request = %s %s, want %s %s", req.Method, req.URL.Path, method, path)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer token" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestIssueViewWebFlag(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config: issueTestConfig{},
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := newCmdIssueView(f)
	cmd.SetArgs([]string{"--web", "alice/demo", "42"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo/issues/42" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func issueResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
