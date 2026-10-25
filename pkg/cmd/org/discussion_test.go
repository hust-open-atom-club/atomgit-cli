package org

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const orgDiscussionFixture = `{"id":"d7","number":7,"title":"Roadmap","md_content":"# Plans\n- Improve docs","author":{"login":"alice","name":"Alice"},"category":{"name":"Ideas","icon":"💡"},"is_closed":0,"is_answered":1,"is_lock":1,"is_pin":1,"comment_total":0,"created_at":"2026-10-01","updated_at":"2026-10-02"}`

func runOrgDiscussionCommand(t *testing.T, cmd *cobra.Command, args []string, flags map[string]string) (string, error) {
	t.Helper()
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Args(cmd, args); err != nil {
		return out.String(), err
	}
	err := cmd.RunE(cmd, args)
	return out.String(), err
}

func TestOrgDiscussionRegistersCommands(t *testing.T) {
	cmd := NewCmdOrg(&cmdutil.Factory{})
	discussionCount := 0
	for _, child := range cmd.Commands() {
		if child.Name() == "discussion" {
			discussionCount++
		}
	}
	if discussionCount != 1 {
		t.Fatalf("registered %d discussion commands, want one", discussionCount)
	}
	for _, tt := range []struct {
		name  string
		flags []string
		args  []string
	}{
		{name: "list", flags: []string{"limit", "json"}, args: []string{"team"}},
		{name: "view", flags: []string{"comments", "json"}, args: []string{"team", "7"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			child, _, err := cmd.Find([]string{"discussion", tt.name})
			if err != nil || child.Name() != tt.name {
				t.Fatalf("command = %v, error = %v", child, err)
			}
			for _, flag := range tt.flags {
				if child.Flags().Lookup(flag) == nil {
					t.Fatalf("flag --%s is missing", flag)
				}
			}
			if err := child.Args(child, tt.args); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{nil, tt.args[:len(tt.args)-1], append(append([]string{}, tt.args...), "extra")} {
				if err := child.Args(child, args); err == nil {
					t.Fatalf("accepted invalid argument count: %v", args)
				}
			}
			var help bytes.Buffer
			child.SetOut(&help)
			if err := child.Help(); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(help.String(), "ag-cli org discussion "+tt.name+" my-organization") {
				t.Fatalf("help = %q", help.String())
			}
		})
	}
}

func TestOrgDiscussionListOutput(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
		json bool
	}{
		{name: "text", body: "[" + orgDiscussionFixture + "]"},
		{name: "json", body: "[" + orgDiscussionFixture + "]", json: true},
		{name: "empty text", body: `[]`},
		{name: "empty json", body: `[]`, json: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/api/v5/orgs/team%3F%23/discuss" {
					t.Fatalf("request = %s %s", req.Method, req.URL)
				}
				if req.Header.Get("Authorization") != "Bearer token" {
					t.Fatal("stored token was not used")
				}
				body := tt.body
				if req.URL.Query().Get("page") != "1" {
					body = `[]`
				}
				return orgResponse(http.StatusOK, body), nil
			})
			factory.RepositoryResolver = func() (cmdutil.Repository, error) {
				t.Fatal("organization discussions must not infer a repository")
				return cmdutil.Repository{}, nil
			}
			flags := map[string]string{"json": fmt.Sprint(tt.json)}
			output, err := runOrgDiscussionCommand(t, newCmdOrgDiscussionList(factory), []string{" team?# "}, flags)
			if err != nil {
				t.Fatal(err)
			}
			if tt.json {
				var items []api.Discussion
				if err := json.Unmarshal([]byte(output), &items); err != nil {
					t.Fatal(err)
				}
				if tt.body == `[]` {
					if strings.TrimSpace(output) != `[]` {
						t.Fatalf("empty JSON = %q", output)
					}
				} else if len(items) != 1 || items[0].Number != 7 || items[0].Title != "Roadmap" || items[0].Category.Name != "Ideas" {
					t.Fatalf("items = %#v", items)
				}
				return
			}
			if tt.body == `[]` {
				if output != "No discussions found.\n" {
					t.Fatalf("output = %q", output)
				}
				return
			}
			for _, want := range []string{"NUMBER", "TITLE", "CATEGORY", "AUTHOR", "STATUS", "COMMENTS", "UPDATED", "#7", "Roadmap", "Ideas", "alice", "ANSWERED,LOCKED,PINNED", "2026-10-02"} {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q:\n%s", want, output)
				}
			}
		})
	}
}

func TestOrgDiscussionListPaginatesAndHonorsLimit(t *testing.T) {
	requests := 0
	factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Query().Get("page") != fmt.Sprint(requests) {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		items := make([]api.Discussion, 100)
		for i := range items {
			items[i].Number = (requests-1)*100 + i + 1
		}
		body, err := json.Marshal(items)
		if err != nil {
			t.Fatal(err)
		}
		return orgResponse(http.StatusOK, string(body)), nil
	})
	output, err := runOrgDiscussionCommand(t, newCmdOrgDiscussionList(factory), []string{"team"}, map[string]string{"limit": "101", "json": "true"})
	if err != nil {
		t.Fatal(err)
	}
	var items []api.Discussion
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || len(items) != 101 || items[100].Number != 101 {
		t.Fatalf("requests = %d, items = %d", requests, len(items))
	}
}

func TestOrgDiscussionViewWithoutComments(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%t", jsonOutput), func(t *testing.T) {
			requests := 0
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/orgs/team/discuss/7" {
					t.Fatalf("request = %s %s", req.Method, req.URL)
				}
				return orgResponse(http.StatusOK, orgDiscussionFixture), nil
			})
			factory.RepositoryResolver = func() (cmdutil.Repository, error) {
				t.Fatal("organization discussions must not infer a repository")
				return cmdutil.Repository{}, nil
			}
			output, err := runOrgDiscussionCommand(t, newCmdOrgDiscussionView(factory), []string{"team", "7"}, map[string]string{"json": fmt.Sprint(jsonOutput)})
			if err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want only detail", requests)
			}
			if jsonOutput {
				assertJSONEqual(t, output, `{"number":7,"title":"Roadmap","author":"alice","category":"💡 Ideas","status":"ANSWERED,LOCKED,PINNED","createdAt":"2026-10-01","updatedAt":"2026-10-02","commentTotal":0,"body":"# Plans\n- Improve docs"}`)
				return
			}
			for _, want := range []string{"#7", "Roadmap", "alice", "💡 Ideas", "ANSWERED,LOCKED,PINNED", "# Plans\n- Improve docs"} {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q:\n%s", want, output)
				}
			}
		})
	}
}

func TestOrgDiscussionViewCommentsAndReplies(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%t", jsonOutput), func(t *testing.T) {
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					t.Fatalf("method = %s", req.Method)
				}
				body := `[]`
				switch req.URL.Path {
				case "/api/v5/orgs/team/discuss/7":
					body = orgDiscussionFixture
				case "/api/v5/orgs/team/discuss/7/comment":
					if req.URL.Query().Get("page") == "1" {
						body = `[{"id":"c1","author":{"login":"bob"},"md_content":"first **comment**","reply_total":0},{"id":"c2","author":{"name":"Carol"},"content":"plain fallback","is_hide":1},{"id":"c3","md_content":"private deleted text","is_deleted":1},{"id":"c4"}]`
					}
				case "/api/v5/orgs/team/discuss/7/comment/c1/reply":
					if req.URL.Query().Get("page") == "1" {
						body = `[{"id":"r1","author":{"login":"dave"},"content":"reply fallback"},{"id":"r2","is_hide":1,"md_content":"private hidden reply"}]`
					}
				case "/api/v5/orgs/team/discuss/7/comment/c2/reply", "/api/v5/orgs/team/discuss/7/comment/c3/reply", "/api/v5/orgs/team/discuss/7/comment/c4/reply":
				default:
					t.Fatalf("unexpected URL = %s", req.URL)
				}
				return orgResponse(http.StatusOK, body), nil
			})
			output, err := runOrgDiscussionCommand(t, newCmdOrgDiscussionView(factory), []string{"team", "7"}, map[string]string{"comments": "true", "json": fmt.Sprint(jsonOutput)})
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"first **comment**", "reply fallback", "[hidden]", "[deleted]", "[no content]"} {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q:\n%s", want, output)
				}
			}
			if strings.Contains(output, "private hidden reply") || strings.Contains(output, "private deleted text") || strings.Contains(output, "plain fallback") {
				t.Fatalf("hidden or deleted content exposed:\n%s", output)
			}
			if jsonOutput {
				var result struct {
					Comments []struct {
						ID      string `json:"id"`
						Replies []struct {
							ID string `json:"id"`
						} `json:"replies"`
					} `json:"comments"`
				}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatal(err)
				}
				if len(result.Comments) != 4 || result.Comments[0].ID != "c1" || len(result.Comments[0].Replies) != 2 || result.Comments[0].Replies[1].ID != "r2" {
					t.Fatalf("comment/reply order = %#v", result.Comments)
				}
			}
		})
	}
}

func TestOrgDiscussionViewEmptyCommentsJSON(t *testing.T) {
	factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
		body := `[]`
		if req.URL.Path == "/api/v5/orgs/team/discuss/7" {
			body = orgDiscussionFixture
		}
		return orgResponse(http.StatusOK, body), nil
	})
	output, err := runOrgDiscussionCommand(t, newCmdOrgDiscussionView(factory), []string{"team", "7"}, map[string]string{"comments": "true", "json": "true"})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if string(result["comments"]) != `[]` {
		t.Fatalf("comments = %s", result["comments"])
	}
}

func TestOrgDiscussionValidationBeforeAuthentication(t *testing.T) {
	for _, tt := range []struct {
		name  string
		build func(*cmdutil.Factory) *cobra.Command
		args  []string
		flags map[string]string
		want  string
	}{
		{name: "missing organization", build: newCmdOrgDiscussionList, want: "accepts 1 arg"},
		{name: "blank organization", build: newCmdOrgDiscussionList, args: []string{" "}, want: "invalid organization"},
		{name: "repository instead of organization", build: newCmdOrgDiscussionList, args: []string{"owner/repo"}, want: "invalid organization"},
		{name: "current directory organization", build: newCmdOrgDiscussionList, args: []string{"."}, want: "invalid organization"},
		{name: "parent directory organization", build: newCmdOrgDiscussionList, args: []string{".."}, want: "invalid organization"},
		{name: "organization with whitespace", build: newCmdOrgDiscussionList, args: []string{"my team"}, want: "invalid organization"},
		{name: "organization with control character", build: newCmdOrgDiscussionList, args: []string{"team\x1b"}, want: "invalid organization"},
		{name: "zero limit", build: newCmdOrgDiscussionList, args: []string{"team"}, flags: map[string]string{"limit": "0"}, want: "must be positive"},
		{name: "negative limit", build: newCmdOrgDiscussionList, args: []string{"team"}, flags: map[string]string{"limit": "-1"}, want: "must be positive"},
		{name: "missing view organization", build: newCmdOrgDiscussionView, args: []string{"7"}, want: "accepts 2 arg"},
		{name: "blank view organization", build: newCmdOrgDiscussionView, args: []string{" ", "7"}, want: "invalid organization"},
		{name: "parent directory view organization", build: newCmdOrgDiscussionView, args: []string{"..", "7"}, want: "invalid organization"},
		{name: "zero number", build: newCmdOrgDiscussionView, args: []string{"team", "0"}, want: "positive integer"},
		{name: "negative number", build: newCmdOrgDiscussionView, args: []string{"team", "-1"}, want: "positive integer"},
		{name: "invalid number", build: newCmdOrgDiscussionView, args: []string{"team", "1.5"}, want: "positive integer"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tokenCalls := 0
			factory := orgFactory(orgTestConfig{tokenCalls: &tokenCalls}, func(*http.Request) (*http.Response, error) {
				t.Fatal("request attempted before validation")
				return nil, nil
			})
			factory.RepositoryResolver = func() (cmdutil.Repository, error) {
				t.Fatal("repository inference attempted")
				return cmdutil.Repository{}, nil
			}
			output, err := runOrgDiscussionCommand(t, tt.build(factory), tt.args, tt.flags)
			if err == nil || !strings.Contains(err.Error(), tt.want) || tokenCalls != 0 || output != "" {
				t.Fatalf("error = %v, token calls = %d, output = %q", err, tokenCalls, output)
			}
		})
	}
}

func TestOrgDiscussionAuthenticationAndClientErrors(t *testing.T) {
	for _, tt := range []struct {
		name  string
		build func(*cmdutil.Factory) *cobra.Command
		args  []string
	}{
		{name: "list", build: newCmdOrgDiscussionList, args: []string{"team"}},
		{name: "view", build: newCmdOrgDiscussionView, args: []string{"team", "7"}},
	} {
		t.Run(tt.name+" anonymous", func(t *testing.T) {
			factory := orgFactory(orgTestConfig{tokenErr: config.ErrNotAuthenticated}, func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Authorization") != "" {
					t.Fatal("anonymous request included authorization")
				}
				body := `[]`
				if tt.name == "view" {
					body = orgDiscussionFixture
				}
				return orgResponse(http.StatusOK, body), nil
			})
			if _, err := runOrgDiscussionCommand(t, tt.build(factory), tt.args, nil); err != nil {
				t.Fatal(err)
			}
		})
		t.Run(tt.name+" credential store error", func(t *testing.T) {
			want := errors.New("credential store unavailable")
			factory := orgFactory(orgTestConfig{tokenErr: want}, func(*http.Request) (*http.Response, error) {
				t.Fatal("request attempted after credential error")
				return nil, nil
			})
			if _, err := runOrgDiscussionCommand(t, tt.build(factory), tt.args, nil); !errors.Is(err, want) {
				t.Fatalf("error = %v, want %v", err, want)
			}
		})
		t.Run(tt.name+" client error", func(t *testing.T) {
			want := errors.New("client unavailable")
			factory := orgFactory(orgTestConfig{}, nil)
			factory.HttpClient = func() (*http.Client, error) { return nil, want }
			if _, err := runOrgDiscussionCommand(t, tt.build(factory), tt.args, nil); !errors.Is(err, want) {
				t.Fatalf("error = %v, want %v", err, want)
			}
		})
	}
}

func TestOrgDiscussionResponseErrors(t *testing.T) {
	for _, tt := range []struct {
		name  string
		build func(*cmdutil.Factory) *cobra.Command
		args  []string
		flags map[string]string
		path  string
		want  string
	}{
		{name: "list", build: newCmdOrgDiscussionList, args: []string{"team"}, path: "/api/v5/orgs/team/discuss", want: `failed to list discussions for organization "team"`},
		{name: "view", build: newCmdOrgDiscussionView, args: []string{"team", "7"}, path: "/api/v5/orgs/team/discuss/7", want: `failed to view discussion #7 for organization "team"`},
		{name: "comments", build: newCmdOrgDiscussionView, args: []string{"team", "7"}, flags: map[string]string{"comments": "true"}, path: "/api/v5/orgs/team/discuss/7/comment", want: `failed to list comments of discussion #7 for organization "team"`},
		{name: "replies", build: newCmdOrgDiscussionView, args: []string{"team", "7"}, flags: map[string]string{"comments": "true"}, path: "/api/v5/orgs/team/discuss/7/comment/c1/reply", want: `failed to list comments of discussion #7 for organization "team"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == tt.path {
					return orgResponse(http.StatusForbidden, `{"message":"Forbidden"}`), nil
				}
				body := `[]`
				if req.URL.Path == "/api/v5/orgs/team/discuss/7" {
					body = orgDiscussionFixture
				} else if req.URL.Path == "/api/v5/orgs/team/discuss/7/comment" && req.URL.Query().Get("page") == "1" {
					body = `[{"id":"c1","reply_total":1}]`
				}
				return orgResponse(http.StatusOK, body), nil
			})
			output, err := runOrgDiscussionCommand(t, tt.build(factory), tt.args, tt.flags)
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), "403") || output != "" {
				t.Fatalf("error = %v, output = %q", err, output)
			}
		})
	}
}
