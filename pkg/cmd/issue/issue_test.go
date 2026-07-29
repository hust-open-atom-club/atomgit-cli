package issue

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type issueTestConfig struct{}

func (issueTestConfig) GetToken() (string, error) { return "token", nil }
func (issueTestConfig) GetUser() (string, error)  { return "alice", nil }
func (issueTestConfig) GetHost() string           { return "atomgit.com" }

type issueUnauthenticatedConfig struct{}

func (issueUnauthenticatedConfig) GetToken() (string, error) {
	return "", config.ErrNotAuthenticated
}
func (issueUnauthenticatedConfig) GetUser() (string, error) {
	return "", config.ErrNotAuthenticated
}
func (issueUnauthenticatedConfig) GetHost() string { return "atomgit.com" }

type issueRoundTripFunc func(*http.Request) (*http.Response, error)

func (f issueRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestNewCmdIssueRegistersSubcommands(t *testing.T) {
	cmd := NewCmdIssue(&cmdutil.Factory{})
	want := map[string]bool{"close": false, "comment": false, "create": false, "edit": false, "label": false, "list": false, "view": false, "reopen": false}
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
	if !strings.Contains(list.Long, cmdutil.RepositoryContextHelp) {
		t.Fatal("list help does not explain repository inference")
	}
	if err := list.Args(list, []string{"one", "two"}); err == nil {
		t.Fatal("list accepted too many arguments")
	}

	edit, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"title", "body", "body-file"} {
		if edit.Flags().Lookup(name) == nil {
			t.Fatalf("edit flag %q was not registered", name)
		}
	}
	if !strings.Contains(edit.Example, "ag issue edit") {
		t.Fatalf("edit command example = %q", edit.Example)
	}
}

func TestIssueCreateBodyInput(t *testing.T) {
	tests := []struct {
		name      string
		body      *string
		bodyFile  *string
		stdin     string
		wantBody  string
		wantError string
	}{
		{name: "inline body", body: issueStringPointer("inline\nbody"), wantBody: "inline\nbody"},
		{name: "UTF-8 file with trailing newlines", bodyFile: issueStringPointer("file"), wantBody: "标题\n\n正文\n"},
		{name: "empty file", bodyFile: issueStringPointer("empty"), wantBody: ""},
		{name: "stdin", bodyFile: issueStringPointer("-"), stdin: "stdin body\n\n", wantBody: "stdin body\n\n"},
		{name: "conflicting flags", body: issueStringPointer("inline"), bodyFile: issueStringPointer("-"), wantError: "mutually exclusive"},
		{name: "missing file", bodyFile: issueStringPointer("missing"), wantError: "failed to read body file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: issueTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/issues" {
							t.Fatalf("request = %s %s", req.Method, req.URL.Path)
						}
						var body map[string]interface{}
						if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
							t.Fatal(err)
						}
						if got := body["body"]; got != tt.wantBody {
							t.Fatalf("body = %q, want %q", got, tt.wantBody)
						}
						return issueResponse(http.StatusCreated, `{"number":"7","html_url":"https://atomgit.com/alice/demo/issues/7"}`), nil
					})}, nil
				},
			}

			cmd := newCmdIssueCreate(factory)
			_ = cmd.Flags().Set("title", "Test issue")
			if tt.body != nil {
				_ = cmd.Flags().Set("body", *tt.body)
			}
			if tt.bodyFile != nil {
				path := *tt.bodyFile
				switch path {
				case "file":
					path = filepath.Join(t.TempDir(), "body.md")
					if err := os.WriteFile(path, []byte(tt.wantBody), 0o600); err != nil {
						t.Fatal(err)
					}
				case "empty":
					path = filepath.Join(t.TempDir(), "empty.md")
					if err := os.WriteFile(path, nil, 0o600); err != nil {
						t.Fatal(err)
					}
				case "missing":
					path = filepath.Join(t.TempDir(), "missing.md")
				}
				_ = cmd.Flags().Set("body-file", path)
			}
			cmd.SetIn(strings.NewReader(tt.stdin))

			err := cmd.RunE(cmd, []string{"alice/demo"})
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantError)
				}
				if requests != 0 {
					t.Fatalf("requests = %d, want 0", requests)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want 1", requests)
			}
		})
	}
}

func issueStringPointer(value string) *string {
	return &value
}

func TestIssueListInfersRepositoryAndHonorsLimit(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
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
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestIssueListJSON(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
		want int
	}{
		{name: "issues", body: `[{"id":1,"number":7,"title":"bug","state":"open","labels":[{"name":"bug"}],"user":{"login":"alice"}}]`, want: 1},
		{name: "empty", body: `[]`, want: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: issueTestConfig{},
				RepositoryResolver: func() (cmdutil.Repository, error) {
					return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
				},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(*http.Request) (*http.Response, error) {
						return issueResponse(http.StatusOK, tt.body), nil
					})}, nil
				},
			}
			cmd := newCmdIssueList(factory)
			_ = cmd.Flags().Set("json", "true")
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, nil); err != nil {
				t.Fatal(err)
			}
			var values []map[string]any
			if err := json.Unmarshal(output.Bytes(), &values); err != nil {
				t.Fatalf("invalid JSON %q: %v", output.String(), err)
			}
			if len(values) != tt.want {
				t.Fatalf("items = %d, want %d", len(values), tt.want)
			}
			if tt.want > 0 && (values[0]["number"] != "7" || values[0]["author"] != "alice") {
				t.Fatalf("issue = %#v", values[0])
			}
		})
	}
}

// TestIssueReopenUsesOwnerPathWithFormData verifies that reopen:
//  1. GETs the issue via /repos/{owner}/{repo}/issues/{number} to retrieve the title
//  2. PATCHes via /repos/{owner}/issues/{number} (owner-only path) with
//     multipart form fields repo, title, and state=reopen
func TestIssueReopenUsesOwnerPathWithFormData(t *testing.T) {
	var patchPath string
	var formFields map[string]string
	call := 0

	issueJSON := `{"number":"7","state":"open","title":"my issue","html_url":"https://atomgit.com/alice/demo/issues/7","user":{"login":"alice"},"created_at":""}`

	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				call++
				switch call {
				case 1:
					// GET /repos/alice/demo/issues/7
					if req.Method != http.MethodGet {
						t.Fatalf("call 1: method = %s, want GET", req.Method)
					}
					if req.URL.Path != "/api/v5/repos/alice/demo/issues/7" {
						t.Fatalf("call 1: path = %s", req.URL.Path)
					}
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueJSON)), Header: make(http.Header)}, nil
				case 2:
					// PATCH /repos/alice/issues/7 with multipart form
					if req.Method != http.MethodPatch {
						t.Fatalf("call 2: method = %s, want PATCH", req.Method)
					}
					patchPath = req.URL.Path
					ct := req.Header.Get("Content-Type")
					mediaType, params, _ := mime.ParseMediaType(ct)
					if mediaType != "multipart/form-data" {
						t.Fatalf("call 2: content-type = %q, want multipart/form-data", ct)
					}
					mr := multipart.NewReader(req.Body, params["boundary"])
					formFields = make(map[string]string)
					for {
						p, err := mr.NextPart()
						if err != nil {
							break
						}
						b, _ := io.ReadAll(p)
						formFields[p.FormName()] = string(b)
					}
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
				case 3:
					// GET /repos/alice/demo/issues/7 — verify state is open
					if req.Method != http.MethodGet {
						t.Fatalf("call 3: method = %s, want GET", req.Method)
					}
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueJSON)), Header: make(http.Header)}, nil
				default:
					t.Fatalf("unexpected call %d", call)
					return nil, nil
				}
			})}, nil
		},
	}

	cmd := newCmdIssueReopen(factory)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}

	if patchPath != "/api/v5/repos/alice/issues/7" {
		t.Errorf("PATCH path = %s, want /api/v5/repos/alice/issues/7", patchPath)
	}
	if formFields["state"] != "reopen" {
		t.Errorf("form field state = %q, want reopen", formFields["state"])
	}
	if formFields["repo"] != "demo" {
		t.Errorf("form field repo = %q, want demo", formFields["repo"])
	}
	if formFields["title"] != "my issue" {
		t.Errorf("form field title = %q, want 'my issue'", formFields["title"])
	}
}

func TestIssueReopenFailsWhenStateRemainsClosedAfterUpdate(t *testing.T) {
	call := 0
	issueJSON := `{"number":"7","state":"open","title":"my issue","html_url":"https://atomgit.com/alice/demo/issues/7","user":{"login":"alice"},"created_at":""}`
	closedJSON := `{"number":"7","state":"closed","title":"my issue","html_url":"https://atomgit.com/alice/demo/issues/7","user":{"login":"alice"},"created_at":""}`

	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				call++
				switch call {
				case 1:
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueJSON)), Header: make(http.Header)}, nil
				case 2:
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
				case 3:
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(closedJSON)), Header: make(http.Header)}, nil
				default:
					t.Fatalf("unexpected call %d", call)
					return nil, nil
				}
			})}, nil
		},
	}

	cmd := newCmdIssueReopen(factory)
	err := cmd.RunE(cmd, []string{"alice/demo", "7"})
	if err == nil || !strings.Contains(err.Error(), "still not open") {
		t.Fatalf("error = %v, want 'still not open'", err)
	}
}

func TestIssueReopenRejectsWrongArgCount(t *testing.T) {
	cmd := newCmdIssueReopen(&cmdutil.Factory{Config: issueTestConfig{}})
	if err := cmd.Args(cmd, nil); err == nil {
		t.Fatal("reopen accepted no issue number")
	}
	if err := cmd.Args(cmd, []string{"7"}); err != nil {
		t.Fatalf("reopen rejected an inferred-repository invocation: %v", err)
	}
	if err := cmd.Args(cmd, []string{"alice/demo", "7", "extra"}); err == nil {
		t.Fatal("reopen accepted 3 args, want error")
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

func TestIssueListPreservesCanonicalAuthenticationError(t *testing.T) {
	cmd := newCmdIssueList(&cmdutil.Factory{
		Config: issueUnauthenticatedConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
	})
	if err := cmd.RunE(cmd, nil); !errors.Is(err, config.ErrNotAuthenticated) {
		t.Fatalf("error = %v", err)
	}
}

func TestIssueListValidatesInputBeforeAuthentication(t *testing.T) {
	t.Run("invalid limit", func(t *testing.T) {
		cmd := newCmdIssueList(&cmdutil.Factory{Config: issueUnauthenticatedConfig{}})
		if err := cmd.Flags().Set("limit", "0"); err != nil {
			t.Fatal(err)
		}
		if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("repository resolution", func(t *testing.T) {
		wantErr := errors.New("repository context unavailable")
		cmd := newCmdIssueList(&cmdutil.Factory{
			Config: issueUnauthenticatedConfig{},
			RepositoryResolver: func() (cmdutil.Repository, error) {
				return cmdutil.Repository{}, wantErr
			},
		})
		if err := cmd.RunE(cmd, nil); !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want repository resolution error", err)
		}
	})
}

func TestIssueViewOutputsLabels(t *testing.T) {
	tests := []struct {
		name       string
		labelsJSON string
		wantLine   string
	}{
		{name: "multiple labels", labelsJSON: `[{"name":"bug"},{"name":"priority/high"}]`, wantLine: "Labels: bug, priority/high\n"},
		{name: "no labels", labelsJSON: `[]`},
		{name: "empty label names", labelsJSON: `[{"name":"  "}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: issueTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/issues/1" {
							t.Fatalf("request = %s %s", req.Method, req.URL.Path)
						}
						if got := req.Header.Get("Authorization"); got != "Bearer token" {
							t.Fatalf("Authorization = %q", got)
						}
						body := `{"title":"Issue title","state":"open","user":{"login":"alice"},` +
							`"html_url":"https://atomgit.com/alice/demo/issues/1","created_at":"2026-07-15","labels":` + tt.labelsJSON + `}`
						return issueResponse(http.StatusOK, body), nil
					})}, nil
				},
			}

			cmd := newCmdIssueView(factory)
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "1"}); err != nil {
				t.Fatal(err)
			}
			want := "Title: Issue title\n" +
				"State: open\n" +
				tt.wantLine +
				"Author: alice\n" +
				"URL: https://atomgit.com/alice/demo/issues/1\n" +
				"Created: 2026-07-15\n"
			if got := output.String(); got != want {
				t.Fatalf("output = %q, want %q", got, want)
			}
		})
	}
}

func TestIssueViewJSON(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return issueResponse(http.StatusOK, `{"id":1,"number":"8","title":"bug","body":"details","state":"open","html_url":"https://atomgit.com/alice/demo/issues/8","user":{"login":"alice"},"labels":[{"name":"bug"},{"name":" "}],"created_at":"today"}`), nil
			})}, nil
		},
	}
	cmd := newCmdIssueView(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "8"}); err != nil {
		t.Fatal(err)
	}
	var value struct {
		Number string   `json:"number"`
		Labels []string `json:"labels"`
		Body   string   `json:"body"`
	}
	if err := json.Unmarshal(output.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.Number != "8" || value.Body != "details" || len(value.Labels) != 1 || value.Labels[0] != "bug" {
		t.Fatalf("issue = %#v", value)
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
