package tag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type tagTestConfig struct{}

func (tagTestConfig) GetToken() (string, error) { return "token", nil }
func (tagTestConfig) GetUser() (string, error)  { return "alice", nil }
func (tagTestConfig) GetHost() string           { return "atomgit.com" }

type recordingTagConfig struct{ getTokenCalls int }

func (c *recordingTagConfig) GetToken() (string, error) {
	c.getTokenCalls++
	return "token", nil
}
func (*recordingTagConfig) GetUser() (string, error) { return "alice", nil }
func (*recordingTagConfig) GetHost() string          { return "atomgit.com" }

type tagRoundTripFunc func(*http.Request) (*http.Response, error)

func (f tagRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestNewCmdTagRegistersSubcommands(t *testing.T) {
	cmd := NewCmdTag(&cmdutil.Factory{})
	want := map[string]bool{"create": false, "delete": false, "list": false}
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

	create, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	if create.Flags().Lookup("message") == nil || create.Flags().Lookup("ref") == nil {
		t.Fatal("create flags were not registered")
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
}

func TestTagCommandsValidateBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name      string
		command   func(*cmdutil.Factory) *cobra.Command
		args      []string
		wantError string
	}{
		{name: "list repository", command: newCmdTagList, args: []string{"demo"}, wantError: "invalid repository format"},
		{name: "create repository", command: newCmdTagCreate, args: []string{"demo", "v1"}, wantError: "invalid repository format"},
		{name: "create name", command: newCmdTagCreate, args: []string{"alice/demo", "  "}, wantError: "tag name is required"},
		{name: "delete repository", command: newCmdTagDelete, args: []string{"demo", "v1"}, wantError: "invalid repository format"},
		{name: "delete name", command: newCmdTagDelete, args: []string{"alice/demo", "  "}, wantError: "tag name is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &recordingTagConfig{}
			cmd := tt.command(&cmdutil.Factory{Config: cfg})
			err := cmd.RunE(cmd, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
			if cfg.getTokenCalls != 0 {
				t.Fatalf("GetToken was called %d times; validation must finish before authentication", cfg.getTokenCalls)
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
