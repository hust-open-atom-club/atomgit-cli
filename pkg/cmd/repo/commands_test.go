package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type repoCommandConfig struct {
	token    string
	user     string
	tokenErr error
	userErr  error
}

func (c repoCommandConfig) GetToken() (string, error) { return c.token, c.tokenErr }
func (c repoCommandConfig) GetUser() (string, error)  { return c.user, c.userErr }
func (c repoCommandConfig) GetHost() string           { return "atomgit.com" }

func repoFactory(config repoCommandConfig, transport forkRoundTripFunc) *cmdutil.Factory {
	factory := &cmdutil.Factory{Config: config}
	if transport != nil {
		factory.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return factory
}

func TestNewCmdRepoRegistersSubcommandsAndFlags(t *testing.T) {
	cmd := NewCmdRepo(&cmdutil.Factory{})
	want := map[string][]string{
		"clone":  {"branch"},
		"create": {"clone", "description", "private", "public"},
		"delete": {"yes"},
		"edit":   {"default-branch", "description", "name", "private", "public", "visibility", "yes"},
		"fork":   {"clone", "description", "name", "private", "public"},
		"list":   {"limit"},
		"view":   {"web"},
	}
	for name, flags := range want {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child.Name() != name {
			t.Fatalf("subcommand %q: %v", name, err)
		}
		for _, flag := range flags {
			if child.Flags().Lookup(flag) == nil {
				t.Errorf("%s --%s flag was not registered", name, flag)
			}
		}
	}

	clone, _, _ := cmd.Find([]string{"clone"})
	if err := clone.Args(clone, nil); err == nil {
		t.Fatal("clone accepted no repository")
	}
	if err := clone.Args(clone, []string{"owner/repo", "target", "extra"}); err == nil {
		t.Fatal("clone accepted too many arguments")
	}

	edit, _, _ := cmd.Find([]string{"edit"})
	if err := edit.Args(edit, []string{"owner/repo", "extra"}); err == nil {
		t.Fatal("edit accepted too many repositories")
	}
	for _, name := range []string{"view", "edit", "fork", "delete"} {
		child, _, _ := cmd.Find([]string{name})
		if !strings.Contains(child.Long, cmdutil.RepositoryContextHelp) {
			t.Errorf("%s help does not explain repository inference", name)
		}
		if err := child.Args(child, nil); err != nil {
			t.Errorf("%s rejected repository inference: %v", name, err)
		}
	}
}

func TestNewAPIClient(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		client, err := newAPIClient(&cmdutil.Factory{}, "token")
		if err != nil || client == nil {
			t.Fatalf("newAPIClient() = %v, %v", client, err)
		}
	})

	t.Run("factory error", func(t *testing.T) {
		factory := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
			return nil, errors.New("factory failed")
		}}
		client, err := newAPIClient(factory, "token")
		if client != nil || err == nil || !strings.Contains(err.Error(), "factory failed") {
			t.Fatalf("newAPIClient() = %v, %v", client, err)
		}
	})
}

func TestListReposPaginatesAndHonorsLimit(t *testing.T) {
	requests := 0
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		query, err := url.ParseQuery(req.URL.RawQuery)
		if err != nil {
			t.Fatal(err)
		}
		if query.Get("per_page") != "100" || query.Get("page") != fmt.Sprint(requests) {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}

		count := 100
		start := 0
		if requests == 2 {
			count = 2
			start = 100
		}
		repositories := make([]api.Repository, count)
		for i := range repositories {
			repositories[i].Name = fmt.Sprintf("repo-%d", start+i)
		}
		data, err := json.Marshal(repositories)
		if err != nil {
			t.Fatal(err)
		}
		return forkResponse(http.StatusOK, string(data)), nil
	})
	client := api.NewClientWithHTTPClient("token", &http.Client{Transport: transport})

	repositories, err := listRepos(client, 101)
	if err != nil {
		t.Fatal(err)
	}
	if len(repositories) != 101 || repositories[100].Name != "repo-100" {
		t.Fatalf("repositories = %d, last = %#v", len(repositories), repositories[len(repositories)-1])
	}
	if requests != 2 {
		t.Fatalf("request count = %d", requests)
	}
}

func TestRepoListCommandRejectsInvalidLimit(t *testing.T) {
	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, nil)
	cmd := newCmdRepoList(factory)
	if err := cmd.Flags().Set("limit", "0"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoListCommandUsesInjectedClient(t *testing.T) {
	requests := 0
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/user/repos" || req.URL.Query().Get("page") != "1" {
			t.Fatalf("request = %s %s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
		}
		return forkResponse(http.StatusOK, `[{"full_name":"alice/demo"}]`), nil
	})
	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	cmd := newCmdRepoList(factory)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("request count = %d", requests)
	}
}

func TestRepoViewCommandReadsRepository(t *testing.T) {
	requests := 0
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return forkResponse(http.StatusOK, `{"full_name":"alice/demo","description":"demo repository","web_url":"https://atomgit.com/alice/demo","default_branch":"main","private":false}`), nil
	})
	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
	}
	cmd := newCmdRepoView(factory)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("request count = %d", requests)
	}
}

func TestRepoViewCommandValidation(t *testing.T) {
	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, nil)
	cmd := newCmdRepoView(factory)
	for name, args := range map[string][]string{
		"missing repository": nil,
		"invalid format":     {"demo"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := cmd.RunE(cmd, args); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestRunCreateSelectsNamespaceAndBody(t *testing.T) {
	tests := []struct {
		name        string
		repository  string
		public      bool
		wantPath    string
		wantPrivate bool
	}{
		{name: "current user", repository: "demo", public: true, wantPath: "/api/v5/user/repos"},
		{name: "organization", repository: "team/demo", wantPath: "/api/v5/orgs/team/repos", wantPrivate: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.Path != tt.wantPath {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["name"] != "demo" || body["description"] != "description" || body["private"] != tt.wantPrivate {
					t.Fatalf("body = %#v", body)
				}
				return forkResponse(http.StatusCreated, `{"name":"demo","web_url":"https://atomgit.com/alice/demo"}`), nil
			})
			factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
			err := runCreate(strings.NewReader(""), io.Discard, io.Discard, factory, &CreateOptions{Name: tt.repository, Description: "description", Public: tt.public})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRunCreateClonesPublicAndPrivateRepositories(t *testing.T) {
	tests := []struct {
		name        string
		repository  string
		public      bool
		wantPath    string
		wantPrivate bool
		wantClone   string
	}{
		{
			name:       "public user repository",
			repository: "demo",
			public:     true,
			wantPath:   "/api/v5/user/repos",
			wantClone:  "https://atomgit.com/alice/demo.git",
		},
		{
			name:        "private organization repository",
			repository:  "team/demo",
			wantPath:    "/api/v5/orgs/team/repos",
			wantPrivate: true,
			wantClone:   "https://atomgit.com/team/demo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.Path != tt.wantPath {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["private"] != tt.wantPrivate {
					t.Fatalf("private = %#v, want %v", body["private"], tt.wantPrivate)
				}
				// Deliberately return a malformed URL. Creation and cloning must use
				// the owner and repository name already known from the request.
				return forkResponse(http.StatusCreated, `{"name":"demo","web_url":"https://atomgit.com//demo"}`), nil
			})
			factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)

			cloneCalls := 0
			clone := func(_ io.Reader, _, _ io.Writer, cloneURL string, opts *CloneOptions) error {
				cloneCalls++
				if cloneURL != tt.wantClone {
					t.Fatalf("clone URL = %q, want %q", cloneURL, tt.wantClone)
				}
				if opts.Directory != "demo" {
					t.Fatalf("clone directory = %q, want demo", opts.Directory)
				}
				return nil
			}

			var out bytes.Buffer
			err := runCreateWithClone(strings.NewReader(""), &out, io.Discard, factory, &CreateOptions{
				Name:   tt.repository,
				Public: tt.public,
				Clone:  true,
			}, clone)
			if err != nil {
				t.Fatal(err)
			}
			if cloneCalls != 1 {
				t.Fatalf("clone calls = %d, want 1", cloneCalls)
			}
			wantURL := strings.TrimSuffix(tt.wantClone, ".git")
			if !strings.Contains(out.String(), "URL: "+wantURL) {
				t.Fatalf("output = %q, want canonical URL %q", out.String(), wantURL)
			}
		})
	}
}

func TestRunCreateReportsCloneFailure(t *testing.T) {
	transport := forkRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return forkResponse(http.StatusCreated, `{}`), nil
	})
	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	clone := func(io.Reader, io.Writer, io.Writer, string, *CloneOptions) error {
		return errors.New("git failed")
	}

	err := runCreateWithClone(strings.NewReader(""), io.Discard, io.Discard, factory, &CreateOptions{Name: "demo", Clone: true}, clone)
	if err == nil || !strings.Contains(err.Error(), "failed to clone newly created repository: git failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunCreateReportsAPIError(t *testing.T) {
	transport := forkRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return forkResponse(http.StatusForbidden, `{"message":"denied"}`), nil
	})
	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	err := runCreate(strings.NewReader(""), io.Discard, io.Discard, factory, &CreateOptions{Name: "demo"})
	if err == nil || !strings.Contains(err.Error(), "failed to create repository") || !strings.Contains(err.Error(), "403") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoDeleteCommand(t *testing.T) {
	requests := 0
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodDelete || req.URL.Path != "/api/v5/repos/alice/demo" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return forkResponse(http.StatusNoContent, ""), nil
	})
	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	cmd := newCmdRepoDelete(factory)
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("request count = %d", requests)
	}
}

func TestRepoViewWebFlag(t *testing.T) {
	var capturedURL string
	f := &cmdutil.Factory{
		Config: repoCommandConfig{token: "token", user: "alice"},
		BrowserOpener: func(rawURL string) error {
			capturedURL = rawURL
			return nil
		},
	}
	cmd := newCmdRepoView(f)
	cmd.SetArgs([]string{"--web", "alice/demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if capturedURL != "https://atomgit.com/alice/demo" {
		t.Fatalf("URL = %q", capturedURL)
	}
}

func TestRepoDeleteCommandInfersRepository(t *testing.T) {
	transport := forkRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete || req.URL.Path != "/api/v5/repos/team/inferred" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return forkResponse(http.StatusNoContent, ""), nil
	})
	factory := repoFactory(repoCommandConfig{token: "token", user: "alice"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "team", Name: "inferred"}, nil
	}
	cmd := newCmdRepoDelete(factory)
	_ = cmd.Flags().Set("yes", "true")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRepoCommandsReportAuthenticationErrors(t *testing.T) {
	cfg := repoCommandConfig{tokenErr: errors.New("missing token"), user: "alice"}
	factory := repoFactory(cfg, nil)
	tests := []struct {
		name string
		call func() error
	}{
		{name: "list", call: func() error { cmd := newCmdRepoList(factory); return cmd.RunE(cmd, nil) }},
		{name: "view", call: func() error { cmd := newCmdRepoView(factory); return cmd.RunE(cmd, []string{"alice/demo"}) }},
		{name: "create", call: func() error {
			return runCreate(strings.NewReader(""), io.Discard, io.Discard, factory, &CreateOptions{Name: "demo"})
		}},
		{name: "fork", call: func() error { return runFork(io.Discard, factory, &ForkOptions{}, "alice/demo") }},
		{name: "edit", call: func() error {
			cmd := newCmdRepoEdit(factory)
			_ = cmd.Flags().Set("description", "updated")
			return cmd.RunE(cmd, []string{"alice/demo"})
		}},
		{name: "delete", call: func() error {
			cmd := newCmdRepoDelete(factory)
			_ = cmd.Flags().Set("yes", "true")
			return cmd.RunE(cmd, []string{"demo"})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil || !strings.Contains(err.Error(), "missing token") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRepoListAndViewPreserveCanonicalAuthenticationError(t *testing.T) {
	factory := repoFactory(repoCommandConfig{tokenErr: config.ErrNotAuthenticated}, nil)
	tests := []struct {
		name string
		call func() error
	}{
		{name: "list", call: func() error {
			cmd := newCmdRepoList(factory)
			return cmd.RunE(cmd, nil)
		}},
		{name: "view", call: func() error {
			cmd := newCmdRepoView(factory)
			return cmd.RunE(cmd, []string{"alice/demo"})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, config.ErrNotAuthenticated) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRepoListValidatesLimitBeforeAuthentication(t *testing.T) {
	factory := repoFactory(repoCommandConfig{tokenErr: config.ErrNotAuthenticated}, nil)
	cmd := newCmdRepoList(factory)
	if err := cmd.Flags().Set("limit", "0"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "invalid limit: 0") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoViewResolvesRepositoryBeforeAuthentication(t *testing.T) {
	wantErr := errors.New("repository context unavailable")
	factory := repoFactory(repoCommandConfig{tokenErr: config.ErrNotAuthenticated}, nil)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{}, wantErr
	}
	cmd := newCmdRepoView(factory)
	if err := cmd.RunE(cmd, nil); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want repository resolution error", err)
	}
}
