package tag

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type tagTestConfig struct{}

func (tagTestConfig) GetToken() (string, error) { return "token", nil }
func (tagTestConfig) GetUser() (string, error)  { return "alice", nil }
func (tagTestConfig) GetHost() string           { return "atomgit.com" }

type tagRoundTripFunc func(*http.Request) (*http.Response, error)

func (f tagRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newTagCountingFactory(config *recordingConfig, clientCreations, requests *int, handler tagRoundTripFunc) *cmdutil.Factory {
	return &cmdutil.Factory{
		Config: config,
		HttpClient: func() (*http.Client, error) {
			(*clientCreations)++
			return &http.Client{Transport: tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				(*requests)++
				if handler == nil {
					return tagNoContentResponse(), nil
				}
				return handler(req)
			})}, nil
		},
	}
}

func tagNoContentResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Status:     "204 No Content",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func TestNewCmdTagRegistersSubcommands(t *testing.T) {
	cmd := NewCmdTag(&cmdutil.Factory{})
	want := map[string]bool{"create": false, "delete": false, "list": false, "protection": false}
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
	for _, flag := range []string{"limit", "json"} {
		if list.Flags().Lookup(flag) == nil {
			t.Fatalf("list --%s flag was not registered", flag)
		}
	}
	if !strings.Contains(list.Example, "ag tag list") {
		t.Fatalf("list example = %q", list.Example)
	}

	create, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	if create.Flags().Lookup("message") == nil || create.Flags().Lookup("ref") == nil {
		t.Fatal("create flags were not registered")
	}
	if !strings.Contains(create.Flags().Lookup("ref").Usage, "required") {
		t.Fatal("create ref flag does not explain that it is required")
	}
	if !strings.Contains(create.Long, cmdutil.RepositoryContextHelp) {
		t.Fatal("create help does not explain repository inference")
	}
	if err := create.Args(create, nil); err == nil {
		t.Fatal("create accepted no tag name")
	}
	if err := create.Args(create, []string{"v1.0.0"}); err != nil {
		t.Fatalf("create rejected an inferred-repository invocation: %v", err)
	}
	if err := create.Args(create, []string{"owner/repo", "v1.0.0"}); err != nil {
		t.Fatalf("create rejected valid arguments: %v", err)
	}

	delete, _, err := cmd.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	if delete.Flags().Lookup("yes") == nil || delete.Flags().ShorthandLookup("y") == nil {
		t.Fatal("delete yes flag was not registered")
	}
	if !strings.Contains(delete.Long, "By default") || !strings.Contains(delete.Long, "--yes") {
		t.Fatalf("delete help does not explain confirmation: %q", delete.Long)
	}
	if !strings.Contains(delete.Example, "--yes") {
		t.Fatalf("delete example does not explain --yes: %q", delete.Example)
	}

	protection, _, err := cmd.Find([]string{"protection"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"list", "view", "set", "delete"} {
		child, _, findErr := protection.Find([]string{name})
		if findErr != nil || child.Name() != name {
			t.Fatalf("protection subcommand %q: %v", name, findErr)
		}
	}
	listProtection, _, err := protection.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"limit", "json"} {
		if listProtection.Flags().Lookup(flag) == nil {
			t.Errorf("protection list --%s flag was not registered", flag)
		}
	}
	viewProtection, _, err := protection.Find([]string{"view"})
	if err != nil {
		t.Fatal(err)
	}
	if viewProtection.Flags().Lookup("json") == nil {
		t.Fatal("protection view --json flag was not registered")
	}
	setProtection, _, err := protection.Find([]string{"set"})
	if err != nil {
		t.Fatal(err)
	}
	if setProtection.Flags().Lookup("create-access") == nil || setProtection.Flags().Lookup("yes") == nil {
		t.Fatal("protection set flags were not registered")
	}
	deleteProtection, _, err := protection.Find([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	if deleteProtection.Flags().Lookup("yes") == nil {
		t.Fatal("protection delete --yes flag was not registered")
	}
}

func TestTagCreateRejectsMissingRefBeforeAuthAndHTTP(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		setFlag bool
	}{
		{name: "missing", setFlag: false},
		{name: "empty", ref: "", setFlag: true},
		{name: "whitespace", ref: " \t ", setFlag: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &recordingConfig{}
			httpClientCalls := 0
			requests := 0
			factory := &cmdutil.Factory{
				Config: cfg,
				HttpClient: func() (*http.Client, error) {
					httpClientCalls++
					return &http.Client{Transport: tagRoundTripFunc(func(*http.Request) (*http.Response, error) {
						requests++
						return nil, fmt.Errorf("unexpected HTTP request")
					})}, nil
				},
			}
			cmd := newCmdTagCreate(factory)
			if tt.setFlag {
				if err := cmd.Flags().Set("ref", tt.ref); err != nil {
					t.Fatal(err)
				}
			}

			err := cmd.RunE(cmd, []string{"alice/demo", "v1.0.0"})
			if err == nil || !strings.Contains(err.Error(), "source ref is required") || !strings.Contains(err.Error(), "branch, tag, or commit SHA") {
				t.Fatalf("error = %v, want source ref guidance", err)
			}
			if cfg.getTokenCalls != 0 {
				t.Fatalf("GetToken was called %d times; invalid ref must be rejected before authentication", cfg.getTokenCalls)
			}
			if httpClientCalls != 0 {
				t.Fatalf("HttpClient was called %d times; invalid ref must be rejected before creating an HTTP client", httpClientCalls)
			}
			if requests != 0 {
				t.Fatalf("HTTP request count = %d, want 0", requests)
			}
		})
	}
}

func TestTagCreateExecuteRejectsMissingRefBeforeAuthAndHTTP(t *testing.T) {
	cfg := &recordingConfig{}
	httpClientCalls := 0
	requests := 0
	factory := &cmdutil.Factory{
		Config: cfg,
		HttpClient: func() (*http.Client, error) {
			httpClientCalls++
			return &http.Client{Transport: tagRoundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return nil, fmt.Errorf("unexpected HTTP request")
			})}, nil
		},
	}
	cmd := newCmdTagCreate(factory)
	cmd.SetArgs([]string{"alice/demo", "v1.0.0"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "source ref is required") || !strings.Contains(err.Error(), "branch, tag, or commit SHA") {
		t.Fatalf("error = %v, want source ref guidance", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; invalid ref must be rejected before authentication", cfg.getTokenCalls)
	}
	if httpClientCalls != 0 {
		t.Fatalf("HttpClient was called %d times; invalid ref must be rejected before creating an HTTP client", httpClientCalls)
	}
	if requests != 0 {
		t.Fatalf("HTTP request count = %d, want 0", requests)
	}
}

func TestTagCreatePreservesRefValues(t *testing.T) {
	for _, tt := range []struct {
		name string
		ref  string
	}{
		{name: "branch", ref: "feature/release"},
		{name: "tag", ref: "v1.2.3"},
		{name: "commit SHA", ref: "0123456789abcdef0123456789abcdef01234567"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: tagTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/tags" {
							t.Fatalf("request = %s %s", req.Method, req.URL.Path)
						}
						var body api.TagRequest
						if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
							t.Fatal(err)
						}
						if body.Refs != tt.ref {
							t.Fatalf("refs = %q, want %q", body.Refs, tt.ref)
						}
						return &http.Response{
							StatusCode: http.StatusOK,
							Status:     "200 OK",
							Header:     make(http.Header),
							Body:       io.NopCloser(strings.NewReader(`{"name":"v1.2.3"}`)),
						}, nil
					})}, nil
				},
			}
			cmd := newCmdTagCreate(factory)
			if err := cmd.Flags().Set("ref", tt.ref); err != nil {
				t.Fatal(err)
			}
			if err := cmd.RunE(cmd, []string{"alice/demo", "v1.2.3"}); err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("request count = %d, want 1", requests)
			}
		})
	}
}

func TestTagListInfersRepository(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: tagTestConfig{},
		RepositoryResolver: func() (cmdutil.Repository, error) {
			return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
		},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/tags" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("request URL = %s", req.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`[{"name":"v1.0.0"}]`)),
				}, nil
			})}, nil
		},
	}

	cmd := newCmdTagList(factory)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestTagListPaginatesAndHonorsLimit(t *testing.T) {
	for _, tt := range []struct {
		name string
		json bool
	}{
		{name: "text"},
		{name: "json", json: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: tagTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/tags" {
							t.Fatalf("request = %s %s", req.Method, req.URL.Path)
						}
						if req.URL.Query().Get("page") != fmt.Sprint(requests) || req.URL.Query().Get("per_page") != "100" {
							t.Fatalf("request URL = %s", req.URL.String())
						}

						var body string
						switch requests {
						case 1:
							var tags strings.Builder
							tags.WriteByte('[')
							for index := 0; index < 100; index++ {
								if index > 0 {
									tags.WriteByte(',')
								}
								fmt.Fprintf(&tags, `{"name":"v%d"}`, index)
							}
							tags.WriteByte(']')
							body = tags.String()
						case 2:
							body = `[{"name":"v100"},{"name":"v101"}]`
						default:
							t.Fatalf("unexpected request %d", requests)
						}
						return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
					})}, nil
				},
			}

			cmd := newCmdTagList(factory)
			if err := cmd.Flags().Set("limit", "101"); err != nil {
				t.Fatal(err)
			}
			if tt.json {
				if err := cmd.Flags().Set("json", "true"); err != nil {
					t.Fatal(err)
				}
			}
			var output bytes.Buffer
			var errOutput bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&errOutput)
			if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
				t.Fatal(err)
			}
			if requests != 2 {
				t.Fatalf("requests = %d, want 2", requests)
			}

			if tt.json {
				var values []tagJSON
				if err := json.Unmarshal(output.Bytes(), &values); err != nil {
					t.Fatalf("invalid JSON %q: %v", output.String(), err)
				}
				if len(values) != 101 || values[0].Name != "v0" || values[100].Name != "v100" {
					t.Fatalf("values = %d, first = %q, last = %q", len(values), values[0].Name, values[len(values)-1].Name)
				}
				return
			}

			lines := strings.Split(strings.TrimSpace(output.String()), "\n")
			if len(lines) != 101 || lines[0] != "v0" || lines[100] != "v100" {
				t.Fatalf("lines = %d, first = %q, last = %q", len(lines), lines[0], lines[len(lines)-1])
			}
		})
	}
}

func TestTagListJSON(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
		want int
	}{
		{name: "tags", body: `[{"name":"v1.0.0","message":"release","commit":{"sha":"abc","url":"commit-url"},"tagger":{"name":"alice","date":"today"}}]`, want: 1},
		{name: "empty", body: `[]`, want: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: tagTestConfig{},
				RepositoryResolver: func() (cmdutil.Repository, error) {
					return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
				},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: tagRoundTripFunc(func(*http.Request) (*http.Response, error) {
						return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tt.body))}, nil
					})}, nil
				},
			}
			cmd := newCmdTagList(factory)
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
				t.Fatalf("tags = %#v", values)
			}
			if tt.want > 0 && (values[0]["commitSha"] != "abc" || values[0]["tagger"] != "alice") {
				t.Fatalf("tag = %#v", values[0])
			}
		})
	}
}

func TestTagListReportsAPIErrorWithContext(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: tagTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: tagRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Status:     "403 Forbidden",
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"message":"denied"}`)),
				}, nil
			})}, nil
		},
	}

	cmd := newCmdTagList(factory)
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil || !strings.Contains(err.Error(), "failed to list tags for alice/demo") || !strings.Contains(err.Error(), "403") {
		t.Fatalf("error = %v", err)
	}
}

func TestTagDeleteConfirmation(t *testing.T) {
	for _, tt := range []struct {
		name        string
		input       string
		yes         bool
		wantDeletes int
		wantPrompt  bool
	}{
		{name: "y", input: " \tY \n", wantDeletes: 1, wantPrompt: true},
		{name: "yes", input: " YeS \n", wantDeletes: 1, wantPrompt: true},
		{name: "no", input: "n\n", wantPrompt: true},
		{name: "other", input: "confirm\n", wantPrompt: true},
		{name: "empty", input: "\n", wantPrompt: true},
		{name: "eof", wantPrompt: true},
		{name: "yes flag", yes: true, wantDeletes: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config := &recordingConfig{}
			clientCreations := 0
			requests := 0
			factory := newTagCountingFactory(config, &clientCreations, &requests, func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodDelete {
					t.Fatalf("request method = %s, want DELETE", req.Method)
				}
				const wantRequestURI = "/api/v5/repos/alice/demo/tags/v1.0%2Frc1"
				if req.URL.RequestURI() != wantRequestURI {
					t.Fatalf("request URI = %q, want %q (path = %q)", req.URL.RequestURI(), wantRequestURI, req.URL.Path)
				}
				return tagNoContentResponse(), nil
			})
			cmd := newCmdTagDelete(factory)
			cmd.SetIn(strings.NewReader(tt.input))
			if tt.yes {
				if err := cmd.Flags().Set("yes", "true"); err != nil {
					t.Fatal(err)
				}
			}
			var output bytes.Buffer
			var errOutput bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&errOutput)

			if err := cmd.RunE(cmd, []string{"alice/demo", "  v1.0/rc1  "}); err != nil {
				t.Fatal(err)
			}
			if config.getTokenCalls != tt.wantDeletes {
				t.Fatalf("GetToken calls = %d, want %d", config.getTokenCalls, tt.wantDeletes)
			}
			if clientCreations != tt.wantDeletes {
				t.Fatalf("HTTP client creations = %d, want %d", clientCreations, tt.wantDeletes)
			}
			if requests != tt.wantDeletes {
				t.Fatalf("HTTP requests = %d, want %d", requests, tt.wantDeletes)
			}
			if tt.wantPrompt {
				const wantPrompt = "Delete tag v1.0/rc1 from alice/demo? [y/N] "
				if !strings.Contains(errOutput.String(), wantPrompt) {
					t.Fatalf("stderr = %q, want prompt containing %q", errOutput.String(), wantPrompt)
				}
			} else if strings.Contains(errOutput.String(), "[y/N]") {
				t.Fatalf("--yes unexpectedly prompted: %q", errOutput.String())
			}
			if tt.wantDeletes == 0 {
				if !strings.Contains(output.String(), "Deletion cancelled.") {
					t.Fatalf("output = %q, want cancellation", output.String())
				}
			} else if !strings.Contains(output.String(), "Deleted tag v1.0/rc1") {
				t.Fatalf("output = %q, want success", output.String())
			}
		})
	}
}

func TestTagDeleteYesRejectsBlankTagBeforeAuth(t *testing.T) {
	config := &recordingConfig{}
	clientCreations := 0
	requests := 0
	factory := newTagCountingFactory(config, &clientCreations, &requests, nil)
	cmd := newCmdTagDelete(factory)
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}

	err := cmd.RunE(cmd, []string{"alice/demo", " \t "})
	if err == nil || !strings.Contains(err.Error(), "tag name is required") {
		t.Fatalf("error = %v, want tag name validation error", err)
	}
	if config.getTokenCalls != 0 {
		t.Fatalf("GetToken calls = %d, want 0", config.getTokenCalls)
	}
	if clientCreations != 0 {
		t.Fatalf("HTTP client creations = %d, want 0", clientCreations)
	}
	if requests != 0 {
		t.Fatalf("HTTP requests = %d, want 0", requests)
	}
}

type tagConfirmationErrorReader struct{}

func (tagConfirmationErrorReader) Read([]byte) (int, error) {
	return 0, errors.New("input failed")
}

func TestTagDeleteConfirmationReaderError(t *testing.T) {
	cmd := newCmdTagDelete(&cmdutil.Factory{Config: tagTestConfig{}})
	cmd.SetIn(tagConfirmationErrorReader{})
	var output bytes.Buffer
	cmd.SetOut(&output)

	err := cmd.RunE(cmd, []string{"alice/demo", "v1.0.0"})
	if err == nil || !strings.Contains(err.Error(), "read confirmation") || !strings.Contains(err.Error(), "input failed") {
		t.Fatalf("error = %v, want contextual confirmation read error", err)
	}
}
