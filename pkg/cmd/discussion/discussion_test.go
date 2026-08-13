package discussion

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type discussionTestConfig struct {
	token    string
	tokenErr error
}

func (c discussionTestConfig) GetToken() (string, error) { return c.token, c.tokenErr }
func (discussionTestConfig) GetUser() (string, error)    { return "tester", nil }
func (discussionTestConfig) GetHost() string             { return "atomgit.com" }

type discussionRoundTripFunc func(*http.Request) (*http.Response, error)

func (f discussionRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func discussionResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func discussionFactory(config discussionTestConfig, transport discussionRoundTripFunc) *cmdutil.Factory {
	factory := &cmdutil.Factory{Config: config}
	if transport != nil {
		factory.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return factory
}

func runDiscussionCommand(t *testing.T, factory *cmdutil.Factory, args []string, flags map[string]string) (string, error) {
	t.Helper()
	cmd := newCmdDiscussionList(factory)
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.Args(cmd, args); err != nil {
		return output.String(), err
	}
	err := cmd.RunE(cmd, args)
	return output.String(), err
}

const completeDiscussionFixture = `[
  {
    "id": "d1",
    "number": 7,
    "title": "Roadmap",
    "author": {
      "id": "u1",
      "login": "alice",
      "name": "Alice",
      "avatar_url": "https://example.test/alice.png"
    },
    "category": {
      "id": "c1",
      "name": "Ideas",
      "icon": "idea",
      "description": "Product ideas",
      "type": 1
    },
    "is_closed": 0,
    "is_answered": 1,
    "is_lock": 0,
    "is_pin": 1,
    "comment_total": 5,
    "created_at": "2026-08-01T10:00:00+08:00",
    "updated_at": "2026-08-02T11:00:00+08:00"
  }
]`

func TestNewCmdDiscussionRegistersListAndHelp(t *testing.T) {
	cmd := NewCmdDiscussion(&cmdutil.Factory{})
	if cmd.Name() != "discussion" {
		t.Fatalf("command name = %q, want discussion", cmd.Name())
	}

	list, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if list.Flags().Lookup("limit") == nil || list.Flags().Lookup("json") == nil {
		t.Fatal("discussion list must register --limit and --json")
	}
	if got := list.Flags().Lookup("limit").DefValue; got != "30" {
		t.Fatalf("default limit = %q, want 30", got)
	}
	if err := list.Args(list, nil); err != nil {
		t.Fatalf("list rejected repository inference form: %v", err)
	}
	if err := list.Args(list, []string{"owner/repo"}); err != nil {
		t.Fatalf("list rejected explicit repository: %v", err)
	}
	if err := list.Args(list, []string{"owner/repo", "extra"}); err == nil {
		t.Fatal("list accepted more than one repository argument")
	}

	var output bytes.Buffer
	list.SetOut(&output)
	if err := list.Help(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"list [<owner>/<repo>]", "--limit", "--json", "repository is inferred"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("help missing %q:\n%s", want, output.String())
		}
	}
}

func TestDiscussionListUsesExplicitRepositoryAndRealArrayResponse(t *testing.T) {
	resolverCalled := false
	requests := 0
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/owner/repo/discuss" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization = %q", got)
		}
		return discussionResponse(http.StatusOK, completeDiscussionFixture), nil
	})
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		resolverCalled = true
		return cmdutil.Repository{}, errors.New("resolver must not be called")
	}

	output, err := runDiscussionCommand(t, factory, []string{"owner/repo"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolverCalled {
		t.Fatal("repository resolver was called for an explicit repository")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	for _, want := range []string{"NUMBER", "TITLE", "CATEGORY", "AUTHOR", "STATUS", "COMMENTS", "UPDATED", "7", "Roadmap", "Ideas", "alice", "5", "2026-08-02T11:00:00+08:00"} {
		if !strings.Contains(output, want) {
			t.Errorf("text output missing %q:\n%s", want, output)
		}
	}
}

func TestDiscussionListInfersRepository(t *testing.T) {
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/inferred/repo/discuss" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return discussionResponse(http.StatusOK, `[]`), nil
	})
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "inferred", Name: "repo"}, nil
	}

	if _, err := runDiscussionCommand(t, factory, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestDiscussionListRejectsInvalidRepositoryBeforeRequest(t *testing.T) {
	invalid := []string{"owner", "owner/", "/repo", "owner/repo/extra", "owner/repo?x=1", "owner/.."}
	for _, repository := range invalid {
		t.Run(repository, func(t *testing.T) {
			requests := 0
			factory := discussionFactory(discussionTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				requests++
				return discussionResponse(http.StatusOK, `[]`), nil
			})
			_, err := runDiscussionCommand(t, factory, []string{repository}, nil)
			if err == nil {
				t.Fatalf("repository %q was accepted", repository)
			}
			if requests != 0 {
				t.Fatalf("requests = %d, want 0", requests)
			}
		})
	}
}

func TestDiscussionListUnauthenticated(t *testing.T) {
	requests := 0
	factory := discussionFactory(discussionTestConfig{tokenErr: errors.New(notAuthenticatedError)}, func(req *http.Request) (*http.Response, error) {
		requests++
		if got := req.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		return discussionResponse(http.StatusOK, completeDiscussionFixture), nil
	})

	output, err := runDiscussionCommand(t, factory, []string{"owner/repo"}, map[string]string{"json": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		t.Fatalf("invalid JSON %q: %v", output, err)
	}
	if len(items) != 1 || items[0]["title"] != "Roadmap" {
		t.Fatalf("items = %#v", items)
	}
}

func TestDiscussionListPropagatesCredentialErrors(t *testing.T) {
	wantErr := errors.New("cannot read token file: permission denied")
	requests := 0
	factory := discussionFactory(discussionTestConfig{tokenErr: wantErr}, func(*http.Request) (*http.Response, error) {
		requests++
		return discussionResponse(http.StatusOK, `[]`), nil
	})

	_, err := runDiscussionCommand(t, factory, []string{"owner/repo"}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestDiscussionListRejectsInvalidLimitBeforeRequest(t *testing.T) {
	for _, limit := range []string{"0", "-1"} {
		t.Run(limit, func(t *testing.T) {
			requests := 0
			factory := discussionFactory(discussionTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				requests++
				return discussionResponse(http.StatusOK, `[]`), nil
			})
			_, err := runDiscussionCommand(t, factory, []string{"owner/repo"}, map[string]string{"limit": limit})
			if err == nil || !strings.Contains(err.Error(), "must be positive") {
				t.Fatalf("error = %v, want positive-limit error", err)
			}
			if requests != 0 {
				t.Fatalf("requests = %d, want 0", requests)
			}
		})
	}
}

func TestDiscussionListHonorsLimit(t *testing.T) {
	body := `[
		{"id":"d1","number":1,"title":"First"},
		{"id":"d2","number":2,"title":"Second"}
	]`
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		return discussionResponse(http.StatusOK, body), nil
	})

	output, err := runDiscussionCommand(t, factory, []string{"owner/repo"}, map[string]string{"limit": "1", "json": "true"})
	if err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		t.Fatalf("invalid JSON %q: %v", output, err)
	}
	if len(items) != 1 || items[0]["title"] != "First" {
		t.Fatalf("items = %#v, want only First", items)
	}
}

func TestDiscussionListJSONMapsCompleteResponse(t *testing.T) {
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		return discussionResponse(http.StatusOK, completeDiscussionFixture), nil
	})
	output, err := runDiscussionCommand(t, factory, []string{"owner/repo"}, map[string]string{"json": "true"})
	if err != nil {
		t.Fatal(err)
	}

	var items []struct {
		ID     string `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		Category struct {
			Name string `json:"name"`
		} `json:"category"`
		IsClosed     bool   `json:"is_closed"`
		IsAnswered   bool   `json:"is_answered"`
		IsLocked     bool   `json:"is_lock"`
		IsPinned     bool   `json:"is_pin"`
		CommentTotal int    `json:"comment_total"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		t.Fatalf("invalid JSON %q: %v", output, err)
	}
	if len(items) != 1 || items[0].ID != "d1" || items[0].Number != 7 || items[0].Title != "Roadmap" ||
		items[0].Author.Login != "alice" || items[0].Category.Name != "Ideas" || items[0].IsClosed ||
		!items[0].IsAnswered || items[0].IsLocked || !items[0].IsPinned || items[0].CommentTotal != 5 ||
		items[0].CreatedAt == "" || items[0].UpdatedAt == "" {
		t.Fatalf("unexpected JSON mapping: %#v", items)
	}
}

func TestDiscussionListEmptyResults(t *testing.T) {
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		return discussionResponse(http.StatusOK, `[]`), nil
	})

	t.Run("text", func(t *testing.T) {
		output, err := runDiscussionCommand(t, factory, []string{"owner/repo"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if output != "No discussions found.\n" {
			t.Fatalf("output = %q", output)
		}
	})

	t.Run("json", func(t *testing.T) {
		output, err := runDiscussionCommand(t, factory, []string{"owner/repo"}, map[string]string{"json": "true"})
		if err != nil {
			t.Fatal(err)
		}
		var items []any
		if err := json.Unmarshal([]byte(output), &items); err != nil {
			t.Fatalf("invalid JSON %q: %v", output, err)
		}
		if len(items) != 0 || strings.Contains(output, "No discussions") {
			t.Fatalf("JSON output = %q", output)
		}
	})
}

func TestDiscussionListHandlesMissingFields(t *testing.T) {
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		return discussionResponse(http.StatusOK, `[{"id":"d1","number":1,"title":"Minimal"}]`), nil
	})
	output, err := runDiscussionCommand(t, factory, []string{"owner/repo"}, map[string]string{"json": "true"})
	if err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		t.Fatalf("invalid JSON %q: %v", output, err)
	}
	if len(items) != 1 || items[0]["title"] != "Minimal" {
		t.Fatalf("items = %#v", items)
	}
}

func TestDiscussionListNon2xxErrors(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			factory := discussionFactory(discussionTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				return discussionResponse(status, `{"error_message":"request failed"}`), nil
			})
			output, err := runDiscussionCommand(t, factory, []string{"owner/repo"}, nil)
			if err == nil || !strings.Contains(err.Error(), "failed to list discussions for owner/repo") || !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("error = %v", err)
			}
			if output != "" {
				t.Fatalf("output = %q, want empty", output)
			}
		})
	}
}
