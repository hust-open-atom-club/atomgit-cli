package tag

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type tagTestConfig struct{}

func (tagTestConfig) GetToken() (string, error) { return "token", nil }
func (tagTestConfig) GetUser() (string, error)  { return "alice", nil }
func (tagTestConfig) GetHost() string           { return "atomgit.com" }

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
