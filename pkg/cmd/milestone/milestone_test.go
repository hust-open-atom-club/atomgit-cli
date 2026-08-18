package milestone

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type milestoneTestConfig struct{ tokenErr error }

func (c milestoneTestConfig) GetToken() (string, error) { return "token", c.tokenErr }
func (milestoneTestConfig) GetUser() (string, error)    { return "alice", nil }
func (milestoneTestConfig) GetHost() string             { return "atomgit.com" }

type milestoneRoundTripFunc func(*http.Request) (*http.Response, error)

func (f milestoneRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestMilestoneRegistersSubcommandsAndFlags(t *testing.T) {
	cmd := NewCmdMilestone(&cmdutil.Factory{})
	want := map[string]bool{"list": false, "view": false, "create": false, "edit": false, "close": false, "reopen": false, "delete": false}
	for _, child := range cmd.Commands() {
		if _, ok := want[child.Name()]; ok {
			want[child.Name()] = true
		}
		if !strings.Contains(child.Long, cmdutil.RepositoryContextHelp) {
			t.Errorf("%s help does not explain repository inference", child.Name())
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
	for _, flag := range []string{"title", "description", "due-on"} {
		if create.Flags().Lookup(flag) == nil {
			t.Errorf("create flag --%s was not registered", flag)
		}
	}
}

func TestMilestoneReturnsCanonicalAuthenticationError(t *testing.T) {
	_, err := authenticatedClient(&cmdutil.Factory{Config: milestoneTestConfig{tokenErr: config.ErrNotAuthenticated}})
	if err != config.ErrNotAuthenticated {
		t.Fatalf("error = %v, want canonical authentication error", err)
	}
}

func TestMilestoneDeleteAuthenticatesBeforeConfirmation(t *testing.T) {
	cmd := newCmdMilestoneDelete(&cmdutil.Factory{Config: milestoneTestConfig{tokenErr: config.ErrNotAuthenticated}})
	cmd.SetIn(strings.NewReader("yes\n"))
	var output bytes.Buffer
	cmd.SetOut(&output)

	err := cmd.RunE(cmd, []string{"alice/demo", "7"})
	if err != config.ErrNotAuthenticated {
		t.Fatalf("error = %v, want canonical authentication error", err)
	}
	if output.Len() != 0 {
		t.Fatalf("prompted before authentication: %q", output.String())
	}
}

func TestMilestoneListPaginatesAndFilters(t *testing.T) {
	requests := 0
	factory := milestoneFactory(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/milestones" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		query := req.URL.Query()
		if query.Get("state") != "all" || query.Get("sort") != "created" || query.Get("direction") != "desc" || query.Get("per_page") != "100" {
			t.Fatalf("query = %v", query)
		}
		page := query.Get("page")
		count := 100
		if page == "2" {
			count = 1
		} else if page != "1" {
			t.Fatalf("page = %s", page)
		}
		items := make([]map[string]interface{}, count)
		for index := range items {
			items[index] = map[string]interface{}{"number": index + 1, "title": "Iteration", "state": "active", "due_on": "2026-08-31"}
		}
		body, err := json.Marshal(items)
		if err != nil {
			t.Fatal(err)
		}
		return milestoneResponse(http.StatusOK, string(body)), nil
	})
	cmd := newCmdMilestoneList(factory)
	setFlags(t, cmd, map[string]string{"state": "all", "sort": "created", "direction": "desc", "limit": "101"})
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if lines := strings.Count(output.String(), "\n"); lines != 101 {
		t.Fatalf("output lines = %d, want 101", lines)
	}
}

func TestMilestoneListJSONAndEmptyText(t *testing.T) {
	for _, test := range []struct {
		name       string
		jsonOutput bool
		body       string
		want       string
	}{
		{name: "json", jsonOutput: true, body: `[{"number":7,"title":"v1","description":"scope","state":"active","due_on":"2026-08-31","open_issues":2,"closed_issues":3,"url":"https://atomgit.com/alice/demo/milestones/7"}]`, want: `"number": "7"`},
		{name: "empty text", body: `[]`, want: "No milestones found.\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			factory := milestoneFactory(func(*http.Request) (*http.Response, error) {
				return milestoneResponse(http.StatusOK, test.body), nil
			})
			cmd := newCmdMilestoneList(factory)
			if test.jsonOutput {
				_ = cmd.Flags().Set("json", "true")
			}
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), test.want) {
				t.Fatalf("output = %q, want containing %q", output.String(), test.want)
			}
			if test.jsonOutput {
				var values []map[string]interface{}
				if err := json.Unmarshal(output.Bytes(), &values); err != nil || len(values) != 1 || values[0]["dueOn"] != "2026-08-31" {
					t.Fatalf("JSON = %#v, error = %v", values, err)
				}
			}
		})
	}
}

func TestMilestoneViewTextAndJSON(t *testing.T) {
	response := `{"number":"12","title":"Release","description":"Ship it","state":"active","due_on":"2026-09-01","open_issues":4,"closed_issues":5,"url":"https://api.atomgit.com/api/v5/repos/alice/demo/milestones/12","html_url":"https://atomgit.com/alice/demo/milestones/12"}`
	for _, jsonOutput := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%t", jsonOutput), func(t *testing.T) {
			factory := milestoneFactory(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/milestones/12" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				return milestoneResponse(http.StatusOK, response), nil
			})
			cmd := newCmdMilestoneView(factory)
			if jsonOutput {
				_ = cmd.Flags().Set("json", "true")
			}
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "12"}); err != nil {
				t.Fatal(err)
			}
			if jsonOutput {
				if !strings.Contains(output.String(), `"openIssues": 4`) || !strings.Contains(output.String(), `"url": "https://atomgit.com/alice/demo/milestones/12"`) {
					t.Fatalf("output = %q", output.String())
				}
			} else {
				for _, want := range []string{"Milestone: #12 Release", "State: active", "Due: 2026-09-01", "Issues: 4 open, 5 closed", "https://atomgit.com/alice/demo/milestones/12", "Ship it"} {
					if !strings.Contains(output.String(), want) {
						t.Errorf("output does not contain %q: %s", want, output.String())
					}
				}
			}
		})
	}
}

func TestMilestoneCreateMapsFields(t *testing.T) {
	factory := milestoneFactory(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v5/repos/alice/demo/milestones" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		var body map[string]string
		decodeMilestoneBody(t, req, &body)
		want := map[string]string{"title": "Version 1.0", "description": "Release scope", "due_on": "2026-08-31"}
		if fmt.Sprint(body) != fmt.Sprint(want) {
			t.Fatalf("body = %#v, want %#v", body, want)
		}
		return milestoneResponse(http.StatusOK, `{"number":9}`), nil
	})
	cmd := newCmdMilestoneCreate(factory)
	setFlags(t, cmd, map[string]string{"title": " Version 1.0 ", "description": "Release scope", "due-on": "2026-08-31"})
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "Created milestone #9: https://atomgit.com/alice/demo/milestones/9\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestMilestoneEditSendsOnlyExplicitFields(t *testing.T) {
	requests := 0
	factory := milestoneFactory(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != "/api/v5/repos/alice/demo/milestones/9" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.Method == http.MethodGet {
			return milestoneResponse(http.StatusOK, `{"number":9,"title":"Version 1.0","due_on":"2026-08-31","description":"old"}`), nil
		}
		if req.Method != http.MethodPatch {
			t.Fatalf("request method = %s", req.Method)
		}
		var body map[string]string
		decodeMilestoneBody(t, req, &body)
		if len(body) != 3 || body["title"] != "Version 1.0" || body["due_on"] != "2026-08-31" || body["description"] != "" {
			t.Fatalf("body = %#v, want required fields plus empty description", body)
		}
		return milestoneResponse(http.StatusNoContent, ""), nil
	})
	cmd := newCmdMilestoneEdit(factory)
	_ = cmd.Flags().Set("description", "")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "9"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want GET then PATCH", requests)
	}
	if got := output.String(); got != "Updated milestone #9: https://atomgit.com/alice/demo/milestones/9\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestMilestoneCloseAndReopenMapStates(t *testing.T) {
	for _, test := range []struct {
		commandName string
		state       string
		wantOutput  string
	}{
		{commandName: "close", state: "closed", wantOutput: "Closed milestone #3:"},
		{commandName: "reopen", state: "open", wantOutput: "Reopened milestone #3:"},
	} {
		t.Run(test.commandName, func(t *testing.T) {
			requests := 0
			factory := milestoneFactory(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.URL.Path != "/api/v5/repos/alice/demo/milestones/3" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if req.Method == http.MethodGet {
					return milestoneResponse(http.StatusOK, `{"number":3,"title":"Release","due_on":"2026-08-31"}`), nil
				}
				if req.Method != http.MethodPatch {
					t.Fatalf("request method = %s", req.Method)
				}
				var body map[string]string
				decodeMilestoneBody(t, req, &body)
				if len(body) != 3 || body["state"] != test.state || body["title"] != "Release" || body["due_on"] != "2026-08-31" {
					t.Fatalf("body = %#v", body)
				}
				return milestoneResponse(http.StatusOK, `{"number":3,"url":"https://gitcode.com/alice/demo/milestones/3"}`), nil
			})
			cmd := newCmdMilestoneState(factory, test.commandName, test.state)
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "3"}); err != nil {
				t.Fatal(err)
			}
			if requests != 2 {
				t.Fatalf("requests = %d, want GET then PATCH", requests)
			}
			if !strings.HasPrefix(output.String(), test.wantOutput) {
				t.Fatalf("output = %q", output.String())
			}
		})
	}
}

func TestMilestoneDeleteConfirmation(t *testing.T) {
	for _, test := range []struct {
		name         string
		input        string
		yes          bool
		wantRequests int
		wantOutput   string
	}{
		{name: "confirmed", input: "yes\n", wantRequests: 1, wantOutput: "Deleted milestone #4\n"},
		{name: "cancelled", input: "n\n", wantOutput: "Deletion cancelled.\n"},
		{name: "yes flag", yes: true, wantRequests: 1, wantOutput: "Deleted milestone #4\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			factory := milestoneFactory(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodDelete || req.URL.Path != "/api/v5/repos/alice/demo/milestones/4" || req.Body != nil {
					t.Fatalf("request = %s %s body=%v", req.Method, req.URL.Path, req.Body)
				}
				return milestoneResponse(http.StatusOK, `{}`), nil
			})
			cmd := newCmdMilestoneDelete(factory)
			cmd.SetIn(strings.NewReader(test.input))
			if test.yes {
				_ = cmd.Flags().Set("yes", "true")
			}
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "4"}); err != nil {
				t.Fatal(err)
			}
			if requests != test.wantRequests {
				t.Fatalf("requests = %d, want %d", requests, test.wantRequests)
			}
			if !strings.HasSuffix(output.String(), test.wantOutput) {
				t.Fatalf("output = %q, want suffix %q", output.String(), test.wantOutput)
			}
		})
	}
}

func TestMilestoneValidationHappensBeforeRequests(t *testing.T) {
	requests := 0
	factory := milestoneFactory(func(*http.Request) (*http.Response, error) {
		requests++
		return milestoneResponse(http.StatusOK, `{}`), nil
	})
	tests := []struct {
		name      string
		command   func() *cobra.Command
		args      []string
		wantError string
	}{
		{name: "invalid list state", command: func() *cobra.Command {
			cmd := newCmdMilestoneList(factory)
			_ = cmd.Flags().Set("state", "active")
			return cmd
		}, args: []string{"alice/demo"}, wantError: "invalid milestone state"},
		{name: "invalid list limit", command: func() *cobra.Command {
			cmd := newCmdMilestoneList(factory)
			_ = cmd.Flags().Set("limit", "0")
			return cmd
		}, args: []string{"alice/demo"}, wantError: "must be positive"},
		{name: "invalid view number", command: func() *cobra.Command { return newCmdMilestoneView(factory) }, args: []string{"alice/demo", "zero"}, wantError: "invalid milestone number"},
		{name: "create missing title", command: func() *cobra.Command {
			cmd := newCmdMilestoneCreate(factory)
			_ = cmd.Flags().Set("due-on", "2026-08-31")
			return cmd
		}, args: []string{"alice/demo"}, wantError: "title is required"},
		{name: "create invalid date", command: func() *cobra.Command {
			cmd := newCmdMilestoneCreate(factory)
			_ = cmd.Flags().Set("title", "v1")
			_ = cmd.Flags().Set("due-on", "2026-2-3")
			return cmd
		}, args: []string{"alice/demo"}, wantError: "expected YYYY-MM-DD"},
		{name: "edit no fields", command: func() *cobra.Command { return newCmdMilestoneEdit(factory) }, args: []string{"alice/demo", "1"}, wantError: "at least one"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := test.command()
			err := cmd.RunE(cmd, test.args)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want containing %q", err, test.wantError)
			}
		})
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestMilestoneAPIErrorsIncludeOperation(t *testing.T) {
	factory := milestoneFactory(func(*http.Request) (*http.Response, error) {
		return milestoneResponse(http.StatusForbidden, `{"message":"denied"}`), nil
	})
	tests := []struct {
		name    string
		command *cobra.Command
		args    []string
		want    string
	}{
		{name: "list", command: newCmdMilestoneList(factory), args: []string{"alice/demo"}, want: "failed to list milestones"},
		{name: "view", command: newCmdMilestoneView(factory), args: []string{"alice/demo", "1"}, want: "failed to get milestone #1"},
		{name: "delete", command: func() *cobra.Command {
			cmd := newCmdMilestoneDelete(factory)
			_ = cmd.Flags().Set("yes", "true")
			return cmd
		}(), args: []string{"alice/demo", "1"}, want: "failed to delete milestone #1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.command.RunE(test.command, test.args)
			if err == nil || !strings.Contains(err.Error(), test.want) || !strings.Contains(err.Error(), "Forbidden") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func milestoneFactory(roundTrip milestoneRoundTripFunc) *cmdutil.Factory {
	return &cmdutil.Factory{
		Config: milestoneTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTrip}, nil
		},
	}
}

func milestoneResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func decodeMilestoneBody(t *testing.T, req *http.Request, target interface{}) {
	t.Helper()
	defer req.Body.Close()
	if err := json.NewDecoder(req.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func setFlags(t *testing.T, cmd *cobra.Command, values map[string]string) {
	t.Helper()
	for name, value := range values {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
}
