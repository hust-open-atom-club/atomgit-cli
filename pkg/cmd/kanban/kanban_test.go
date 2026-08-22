package kanban

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type kanbanCommandConfig struct{ tokenErr error }

func (c kanbanCommandConfig) GetToken() (string, error) { return "token", c.tokenErr }
func (c kanbanCommandConfig) GetUser() (string, error)  { return "alice", nil }
func (c kanbanCommandConfig) GetHost() string           { return "atomgit.com" }

type kanbanCommandRoundTripper func(*http.Request) (*http.Response, error)

func (f kanbanCommandRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func commandResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func commandFactory(config kanbanCommandConfig, transport kanbanCommandRoundTripper) *cmdutil.Factory {
	return &cmdutil.Factory{Config: config, HttpClient: func() (*http.Client, error) {
		return &http.Client{Transport: transport}, nil
	}}
}

func TestNewCmdKanbanRegistersCommandsAndFlags(t *testing.T) {
	cmd := NewCmdKanban(&cmdutil.Factory{})
	for _, name := range []string{"list", "view", "items"} {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child.Name() != name {
			t.Fatalf("command %q: %v", name, err)
		}
	}
	for _, name := range []string{"list", "items"} {
		child, _, _ := cmd.Find([]string{name})
		for _, flag := range []string{"limit", "json"} {
			if child.Flags().Lookup(flag) == nil {
				t.Errorf("%s missing --%s", name, flag)
			}
		}
	}
	view, _, _ := cmd.Find([]string{"view"})
	if view.Flags().Lookup("json") == nil {
		t.Fatal("view missing --json")
	}
}

func TestKanbanListTextAndJSONOutput(t *testing.T) {
	requests := 0
	factory := commandFactory(kanbanCommandConfig{}, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != "/api/v5/org/team/kanban/list" || req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "30" {
			t.Fatalf("request = %s %s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
		}
		content := make([]api.Kanban, 0, 2)
		content = append(content, api.Kanban{ID: "123", IID: 7, Name: "Board", Status: 0, UpdatedAt: "today"})
		body, err := json.Marshal(map[string]any{"content": content, "all_count": 1})
		if err != nil {
			t.Fatal(err)
		}
		return commandResponse(http.StatusOK, string(body)), nil
	})
	cmd := newCmdKanbanList(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"team"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ID") || !strings.Contains(out.String(), "Board") || requests != 1 {
		t.Fatalf("output=%q requests=%d", out.String(), requests)
	}

	cmd = newCmdKanbanList(factory)
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"team"}); err != nil {
		t.Fatal(err)
	}
	var got []kanbanJSON
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || len(got) != 1 || got[0].ID != "123" {
		t.Fatalf("json=%q err=%v", out.String(), err)
	}
}

func TestKanbanItemsDistinguishIssueAndPullRequest(t *testing.T) {
	factory := commandFactory(kanbanCommandConfig{}, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/org/team/kanban/123/item_list" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return commandResponse(http.StatusOK, `[{"id":1,"number":42,"title":"Bug","source_type":"issue","status":"opened","html_url":"https://atomgit.com/team/repo/issues/42","values":[{"field_name":"状态","field_type":"status","value":"Doing"}]},{"id":2,"number":"9","title":"Fix","source_type":"pull_request","status":"closed","html_url":"https://atomgit.com/team/repo/pulls/9"}]`), nil
	})
	cmd := newCmdKanbanItems(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"team", "123"}); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Issue #42", "Pull Request #9", "column:Doing"} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("output missing %q: %s", text, out.String())
		}
	}
}

func TestKanbanValidationHappensBeforeAuthentication(t *testing.T) {
	requests := 0
	transport := func(*http.Request) (*http.Response, error) {
		requests++
		return commandResponse(http.StatusOK, `[]`), nil
	}
	factory := commandFactory(kanbanCommandConfig{tokenErr: config.ErrNotAuthenticated}, transport)
	cmd := newCmdKanbanList(factory)
	if err := cmd.Flags().Set("limit", "0"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"team"}); err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("error = %v", err)
	}
	cmd = newCmdKanbanView(factory)
	if err := cmd.RunE(cmd, []string{"team", "not-a-number"}); err == nil || !strings.Contains(err.Error(), "positive integer") {
		t.Fatalf("error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}
