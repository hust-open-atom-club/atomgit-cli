package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type repositoryCollectionCountingConfig struct {
	tokenCalls int
	tokenErr   error
}

func (c *repositoryCollectionCountingConfig) GetToken() (string, error) {
	c.tokenCalls++
	return "token", c.tokenErr
}

func (c *repositoryCollectionCountingConfig) GetUser() (string, error) { return "alice", nil }
func (c *repositoryCollectionCountingConfig) GetHost() string          { return "atomgit.com" }

func runCollectionCommand(t *testing.T, factory *cmdutil.Factory, name string, args []string, flags map[string]string, out *bytes.Buffer) error {
	t.Helper()
	var cmd *cobra.Command
	if name == "starred" {
		cmd = newCmdUserStarred(factory)
	} else {
		cmd = newCmdUserWatching(factory)
	}
	cmd.SetArgs(args)
	for flag, value := range flags {
		if err := cmd.Flags().Set(flag, value); err != nil {
			t.Fatalf("set --%s: %v", flag, err)
		}
	}
	cmd.SetOut(out)
	return cmd.RunE(cmd, args)
}

func TestNewCmdUserRegistersRepositoryCollections(t *testing.T) {
	cmd := NewCmdUser(&cmdutil.Factory{})
	for _, name := range []string{"starred", "watching"} {
		collection, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatalf("find %s: %v", name, err)
		}
		for _, flag := range []string{"limit", "json"} {
			if collection.Flags().Lookup(flag) == nil {
				t.Fatalf("%s flag %q was not registered", name, flag)
			}
		}
		if got := collection.Flags().Lookup("limit").DefValue; got != "30" {
			t.Fatalf("%s default limit = %q, want 30", name, got)
		}
		if err := collection.Args(collection, []string{"one", "two"}); err == nil {
			t.Fatalf("user %s accepted more than one argument", name)
		}
	}
}

func TestUserRepositoryCollectionsValidateBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		flags map[string]string
		want  string
	}{
		{name: "zero limit", flags: map[string]string{"limit": "0"}, want: "must be positive"},
		{name: "negative limit", flags: map[string]string{"limit": "-1"}, want: "must be positive"},
		{name: "invalid username", args: []string{"alice/admin"}, want: "invalid login"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &repositoryCollectionCountingConfig{}
			var out bytes.Buffer
			err := runCollectionCommand(t, &cmdutil.Factory{Config: config}, "starred", tt.args, tt.flags, &out)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if config.tokenCalls != 0 {
				t.Fatalf("GetToken calls = %d, want 0", config.tokenCalls)
			}
		})
	}
}

func TestUserRepositoryCollectionsRequireAuthentication(t *testing.T) {
	for _, args := range [][]string{nil, {"bob"}} {
		config := &repositoryCollectionCountingConfig{tokenErr: errors.New("not authenticated")}
		var out bytes.Buffer
		err := runCollectionCommand(t, &cmdutil.Factory{Config: config}, "watching", args, nil, &out)
		if err == nil || !strings.Contains(err.Error(), "not authenticated") {
			t.Fatalf("args = %v, error = %v", args, err)
		}
		if config.tokenCalls != 1 {
			t.Fatalf("args = %v, GetToken calls = %d, want 1", args, config.tokenCalls)
		}
	}
}

func TestUserRepositoryCollectionEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		args     []string
		wantPath string
	}{
		{name: "current starred", command: "starred", wantPath: "/api/v5/user/starred"},
		{name: "public starred", command: "starred", args: []string{"bob"}, wantPath: "/api/v5/users/bob/starred"},
		{name: "current watching", command: "watching", wantPath: "/api/v5/user/subscriptions"},
		{name: "public watching", command: "watching", args: []string{"bob"}, wantPath: "/api/v5/users/bob/subscriptions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := namespaceTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != tt.wantPath {
					t.Errorf("request = %s %s, want GET %s", req.Method, req.URL.Path, tt.wantPath)
				}
				if req.URL.RawQuery != "page=1&per_page=100" {
					t.Errorf("query = %q, want page=1&per_page=100", req.URL.RawQuery)
				}
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Errorf("Authorization = %q", got)
				}
				fmt.Fprint(w, `[]`)
			})
			var out bytes.Buffer
			if err := runCollectionCommand(t, factory, tt.command, tt.args, nil, &out); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUserRepositoryCollectionEscapesUsername(t *testing.T) {
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		if got := req.URL.EscapedPath(); got != "/api/v5/users/alice%252Fadmin/starred" {
			t.Fatalf("escaped path = %q", got)
		}
		fmt.Fprint(w, `[]`)
	})
	var out bytes.Buffer
	if err := runCollectionCommand(t, factory, "starred", []string{"alice%2Fadmin"}, nil, &out); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepositoryCollectionsPaginateAndHonorLimit(t *testing.T) {
	requests := 0
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, req *http.Request) {
		requests++
		if got := req.URL.Query().Get("page"); got != strconv.Itoa(requests) {
			t.Errorf("page = %q, want %d", got, requests)
		}
		if got := req.URL.Query().Get("per_page"); got != "100" {
			t.Errorf("per_page = %q, want 100", got)
		}
		count, start := 100, 1
		if requests == 2 {
			count, start = 2, 101
		}
		repositories := make([]api.Repository, count)
		for index := range repositories {
			id := start + index
			repositories[index] = api.Repository{ID: int64(id), FullName: fmt.Sprintf("team/repo-%d", id), HTMLURL: fmt.Sprintf("https://atomgit.com/team/repo-%d", id)}
		}
		if err := json.NewEncoder(w).Encode(repositories); err != nil {
			t.Fatal(err)
		}
	})
	var out bytes.Buffer
	if err := runCollectionCommand(t, factory, "watching", nil, map[string]string{"limit": "101", "json": "true"}, &out); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	var got []userRepositoryJSON
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 101 || got[100].FullName != "team/repo-101" {
		t.Fatalf("repositories = %d, last = %#v", len(got), got[len(got)-1])
	}
}

func TestUserRepositoryCollectionsTextAndJSON(t *testing.T) {
	response := `[
		{"id":1,"full_name":"team/demo","web_url":"https://atomgit.com/team/demo"},
		{"id":2,"name":"tools","namespace":{"path":"group"},"html_url":"https://atomgit.com/group/tools"}
	]`
	t.Run("text", func(t *testing.T) {
		factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, response) })
		var out bytes.Buffer
		if err := runCollectionCommand(t, factory, "starred", nil, nil, &out); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"REPOSITORY", "URL", "team/demo", "https://atomgit.com/team/demo", "group/tools", "https://atomgit.com/group/tools"} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("output missing %q:\n%s", want, out.String())
			}
		}
	})
	t.Run("stable json", func(t *testing.T) {
		factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, response) })
		var out bytes.Buffer
		if err := runCollectionCommand(t, factory, "watching", nil, map[string]string{"json": "true"}, &out); err != nil {
			t.Fatal(err)
		}
		var got []map[string]any
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		want := []map[string]any{
			{"id": float64(1), "fullName": "team/demo", "url": "https://atomgit.com/team/demo"},
			{"id": float64(2), "fullName": "group/tools", "url": "https://atomgit.com/group/tools"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("JSON = %#v, want %#v", got, want)
		}
	})
}

func TestUserRepositoryCollectionsEmptyResults(t *testing.T) {
	emptyText := map[string]string{
		"starred":  "No starred repositories found.\n",
		"watching": "No watched repositories found.\n",
	}
	for _, command := range []string{"starred", "watching"} {
		for _, jsonOutput := range []bool{false, true} {
			name := command
			if jsonOutput {
				name += " json"
			}
			t.Run(name, func(t *testing.T) {
				factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `[]`) })
				flags := map[string]string{}
				if jsonOutput {
					flags["json"] = "true"
				}
				var out bytes.Buffer
				if err := runCollectionCommand(t, factory, command, nil, flags, &out); err != nil {
					t.Fatal(err)
				}
				if jsonOutput {
					if got := strings.TrimSpace(out.String()); got != "[]" {
						t.Fatalf("output = %q, want []", got)
					}
				} else if got := out.String(); got != emptyText[command] {
					t.Fatalf("output = %q, want %q", got, emptyText[command])
				}
			})
		}
	}
}

func TestUserRepositoryCollectionsReportErrors(t *testing.T) {
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	})
	var out bytes.Buffer
	err := runCollectionCommand(t, factory, "starred", []string{"alice"}, nil, &out)
	if err == nil || !strings.Contains(err.Error(), "list starred repositories for user \"alice\"") || !strings.Contains(err.Error(), "403") {
		t.Fatalf("error = %v", err)
	}
}

func TestUserRepositoryCollectionsRejectAmbiguousRepository(t *testing.T) {
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"id":1,"name":"demo"}]`)
	})
	var out bytes.Buffer
	err := runCollectionCommand(t, factory, "watching", nil, nil, &out)
	if err == nil || !strings.Contains(err.Error(), "unambiguous full name") {
		t.Fatalf("error = %v", err)
	}
}

func TestUserRepositoryCollectionsSanitizeTextOutput(t *testing.T) {
	factory := namespaceTestFactory(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"id":1,"full_name":"team/demo\tname","web_url":"https://atomgit.com/team/demo\u001b[31m\nnext"}]`)
	})
	var raw bytes.Buffer
	sanitized := cmdutil.NewSanitizingWriter(&raw)
	cmd := newCmdUserStarred(factory)
	cmd.SetOut(sanitized)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := sanitized.Flush(); err != nil {
		t.Fatal(err)
	}
	got := raw.String()
	if strings.ContainsRune(got, '\x1b') || strings.Contains(got, "demo\tname") || strings.Contains(got, "[31m\nnext") {
		t.Fatalf("control bytes were not sanitized: %q", got)
	}
	for _, want := range []string{`demo\tname`, `\x1b[31m\nnext`} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, missing %q", got, want)
		}
	}
}

func TestUserRepositoriesJSONBuildsMissingURL(t *testing.T) {
	repository := api.Repository{ID: 1, Name: "demo"}
	repository.Owner.Login = "alice"
	got, err := userRepositoriesJSON([]api.Repository{repository}, "git.example.test")
	if err != nil {
		t.Fatal(err)
	}
	want := []userRepositoryJSON{{ID: 1, FullName: "alice/demo", URL: "https://git.example.test/alice/demo"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("repositories = %#v, want %#v", got, want)
	}
}
