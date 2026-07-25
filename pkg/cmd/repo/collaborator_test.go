package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type collaboratorTestConfig struct{}

func (collaboratorTestConfig) GetToken() (string, error) { return "token", nil }
func (collaboratorTestConfig) GetUser() (string, error)  { return "alice", nil }
func (collaboratorTestConfig) GetHost() string           { return "atomgit.com" }

type collaboratorRoundTripFunc func(*http.Request) (*http.Response, error)

func (f collaboratorRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRepoCollaboratorRegistersCommandsAndFlags(t *testing.T) {
	repoCmd := NewCmdRepo(&cmdutil.Factory{})
	registered, _, err := repoCmd.Find([]string{"collaborator"})
	if err != nil || registered.Name() != "collaborator" {
		t.Fatalf("repo collaborator was not registered: command = %v, error = %v", registered, err)
	}

	cmd := newCmdRepoCollaborator(&cmdutil.Factory{})
	want := map[string][]string{
		"list":   {"limit", "json"},
		"view":   {"json"},
		"add":    {"permission"},
		"edit":   {"permission", "yes"},
		"remove": {"yes"},
	}
	for name, flags := range want {
		child, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(child.Long, cmdutil.RepositoryContextHelp) {
			t.Errorf("%s help does not explain repository inference", name)
		}
		for _, flag := range flags {
			if child.Flags().Lookup(flag) == nil {
				t.Errorf("%s --%s flag was not registered", name, flag)
			}
		}
	}
}

func TestRepoCollaboratorListPaginatesAndShowsPermissionSources(t *testing.T) {
	requests := 0
	factory := collaboratorFactory(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/collaborators" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.URL.Query().Get("per_page") != "100" {
			t.Fatalf("query = %v", req.URL.Query())
		}
		if req.URL.Query().Get("page") == "1" {
			items := make([]map[string]interface{}, 100)
			for index := range items {
				items[index] = map[string]interface{}{
					"username": fmt.Sprintf("user-%d", index), "permission": "push",
					"type": "ProjectMember", "join_way": "normal", "source_name": "demo",
				}
			}
			body, err := json.Marshal(items)
			if err != nil {
				t.Fatal(err)
			}
			return collaboratorResponse(http.StatusOK, string(body)), nil
		}
		if req.URL.Query().Get("page") != "2" {
			t.Fatalf("page = %s", req.URL.Query().Get("page"))
		}
		return collaboratorResponse(http.StatusOK, `[{"username":"inherited","permission":"pull","type":"OrganizationMember","join_way":"inherit","source_name":"club","role_name":"Developer"}]`), nil
	})
	cmd := newCmdRepoCollaboratorList(factory)
	_ = cmd.Flags().Set("limit", "101")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	for _, want := range []string{"user-0 [push] direct", "inherited [pull] inherited from club (Developer)"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output does not contain %q", want)
		}
	}
}

func TestRepoCollaboratorListJSON(t *testing.T) {
	factory := collaboratorFactory(func(*http.Request) (*http.Response, error) {
		return collaboratorResponse(http.StatusOK, `[{"username":"bob","name":"Bob","permission":"admin","type":"ProjectMember","join_way":"normal","source_name":"demo","role_name":"Maintainer","web_url":"https://atomgit.com/bob"}]`), nil
	})
	cmd := newCmdRepoCollaboratorList(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	var values []map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0]["username"] != "bob" || values[0]["permission"] != "admin" || values[0]["direct"] != true {
		t.Fatalf("JSON = %#v", values)
	}
}

func TestRepoCollaboratorView(t *testing.T) {
	factory := collaboratorFactory(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/collaborators/bob/permission" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return collaboratorResponse(http.StatusOK, `{"login":"bob","permission":"push","type":"ProjectMember","join_way":"normal","source_name":"demo","role_name":"Developer","web_url":"https://atomgit.com/bob"}`), nil
	})
	cmd := newCmdRepoCollaboratorView(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "bob"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Username: bob", "Permission: push", "Access: direct", "Role: Developer", "Source: demo", "URL: https://atomgit.com/bob"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output does not contain %q: %s", want, output.String())
		}
	}
}

func TestRepoCollaboratorAddChecksThenSendsPermission(t *testing.T) {
	requests := 0
	factory := collaboratorFactory(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/api/v5/repos/alice/demo/collaborators/bob/permission" && req.Method == http.MethodGet {
			return collaboratorResponse(http.StatusNotFound, `{"message":"not found"}`), nil
		}
		if req.URL.Path != "/api/v5/repos/alice/demo/collaborators/bob" || req.Method != http.MethodPut {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		var body map[string]string
		decodeCollaboratorBody(t, req, &body)
		if len(body) != 1 || body["permission"] != "pull" {
			t.Fatalf("body = %#v", body)
		}
		return collaboratorResponse(http.StatusOK, `{"login":"bob","permission":"pull"}`), nil
	})
	cmd := newCmdRepoCollaboratorAdd(factory)
	_ = cmd.Flags().Set("permission", "pull")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "bob"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || output.String() != "Added collaborator bob with pull permission\n" {
		t.Fatalf("requests = %d, output = %q", requests, output.String())
	}
}

func TestRepoCollaboratorAddRejectsExistingAccess(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "direct", body: `{"username":"bob","permission":"push","type":"ProjectMember","join_way":"normal","source_name":"demo"}`, want: "already a direct collaborator"},
		{name: "inherited", body: `{"username":"bob","permission":"pull","type":"OrganizationMember","join_way":"inherit","source_name":"club"}`, want: "inherited from club"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			factory := collaboratorFactory(func(*http.Request) (*http.Response, error) {
				requests++
				return collaboratorResponse(http.StatusOK, test.body), nil
			})
			cmd := newCmdRepoCollaboratorAdd(factory)
			err := cmd.RunE(cmd, []string{"alice/demo", "bob"})
			if err == nil || !strings.Contains(err.Error(), test.want) || requests != 1 {
				t.Fatalf("error = %v, requests = %d", err, requests)
			}
		})
	}
}

func TestRepoCollaboratorEditConfirmsPermissionReduction(t *testing.T) {
	for _, test := range []struct {
		name       string
		input      string
		yes        bool
		wantWrites int
		wantOutput string
	}{
		{name: "cancel", input: "n\n", wantOutput: "Permission update cancelled.\n"},
		{name: "confirm", input: "yes\n", wantWrites: 1, wantOutput: "Updated collaborator bob to pull permission\n"},
		{name: "yes flag", yes: true, wantWrites: 1, wantOutput: "Updated collaborator bob to pull permission\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			writes := 0
			factory := collaboratorFactory(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return collaboratorResponse(http.StatusOK, `{"username":"bob","permission":"admin","type":"ProjectMember","join_way":"normal","source_name":"demo","role_name":"Maintainer"}`), nil
				}
				writes++
				var body map[string]string
				decodeCollaboratorBody(t, req, &body)
				if req.Method != http.MethodPut || body["permission"] != "pull" {
					t.Fatalf("request = %s, body = %#v", req.Method, body)
				}
				return collaboratorResponse(http.StatusOK, `{"username":"bob","permission":"pull"}`), nil
			})
			cmd := newCmdRepoCollaboratorEdit(factory)
			_ = cmd.Flags().Set("permission", "pull")
			if test.yes {
				_ = cmd.Flags().Set("yes", "true")
			}
			cmd.SetIn(strings.NewReader(test.input))
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "bob"}); err != nil {
				t.Fatal(err)
			}
			if writes != test.wantWrites || !strings.HasSuffix(output.String(), test.wantOutput) {
				t.Fatalf("writes = %d, output = %q", writes, output.String())
			}
		})
	}
}

func TestRepoCollaboratorEditUpgradeDoesNotPrompt(t *testing.T) {
	writes := 0
	factory := collaboratorFactory(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return collaboratorResponse(http.StatusOK, `{"username":"bob","permission":"pull","type":"ProjectMember","join_way":"normal","source_name":"demo","role_name":"Developer"}`), nil
		}
		writes++
		return collaboratorResponse(http.StatusOK, `{"username":"bob","permission":"push"}`), nil
	})
	cmd := newCmdRepoCollaboratorEdit(factory)
	_ = cmd.Flags().Set("permission", "push")
	cmd.SetIn(failingReader{})
	if err := cmd.RunE(cmd, []string{"alice/demo", "bob"}); err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Fatalf("writes = %d", writes)
	}
}

func TestRepoCollaboratorRemoveConfirmationAndInheritedProtection(t *testing.T) {
	for _, test := range []struct {
		name       string
		body       string
		input      string
		wantWrites int
		wantError  string
		wantOutput string
	}{
		{name: "cancel", body: directCollaboratorJSON("bob", "push"), input: "n\n", wantOutput: "Removal cancelled.\n"},
		{name: "confirm", body: directCollaboratorJSON("bob", "push"), input: "yes\n", wantWrites: 1, wantOutput: "Removed collaborator bob\n"},
		{name: "inherited", body: `{"username":"bob","permission":"pull","type":"OrganizationMember","join_way":"inherit","source_name":"club"}`, wantError: "inherited from club"},
		{name: "owner", body: `{"username":"alice","permission":"admin","type":"ProjectMember","join_way":"normal","source_name":"demo","role_name":"Owner"}`, wantError: "repository owner"},
	} {
		t.Run(test.name, func(t *testing.T) {
			writes := 0
			factory := collaboratorFactory(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return collaboratorResponse(http.StatusOK, test.body), nil
				}
				writes++
				if req.Method != http.MethodDelete {
					t.Fatalf("method = %s", req.Method)
				}
				return collaboratorResponse(http.StatusOK, `{}`), nil
			})
			cmd := newCmdRepoCollaboratorRemove(factory)
			cmd.SetIn(strings.NewReader(test.input))
			var output bytes.Buffer
			cmd.SetOut(&output)
			err := cmd.RunE(cmd, []string{"alice/demo", collaboratorTestUsername(test.body)})
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("error = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if writes != test.wantWrites || (test.wantOutput != "" && !strings.HasSuffix(output.String(), test.wantOutput)) {
				t.Fatalf("writes = %d, output = %q", writes, output.String())
			}
		})
	}
}

func TestRepoCollaboratorValidationAndAPIErrors(t *testing.T) {
	requests := 0
	factory := collaboratorFactory(func(*http.Request) (*http.Response, error) {
		requests++
		return collaboratorResponse(http.StatusForbidden, `{"message":"denied"}`), nil
	})
	add := newCmdRepoCollaboratorAdd(factory)
	_ = add.Flags().Set("permission", "maintain")
	if err := add.RunE(add, []string{"alice/demo", "bob"}); err == nil || !strings.Contains(err.Error(), "expected pull, push, or admin") {
		t.Fatalf("invalid permission error = %v", err)
	}
	view := newCmdRepoCollaboratorView(factory)
	if err := view.RunE(view, []string{"alice/demo", "bad/name"}); err == nil || !strings.Contains(err.Error(), "invalid collaborator username") {
		t.Fatalf("invalid username error = %v", err)
	}
	view = newCmdRepoCollaboratorView(factory)
	if err := view.RunE(view, []string{"alice/demo", "bob"}); err == nil || !strings.Contains(err.Error(), "failed to view collaborator") || !strings.Contains(err.Error(), "Forbidden") {
		t.Fatalf("API error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, fmt.Errorf("unexpected prompt") }

func collaboratorFactory(roundTrip collaboratorRoundTripFunc) *cmdutil.Factory {
	return &cmdutil.Factory{
		Config: collaboratorTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTrip}, nil
		},
	}
}

func collaboratorResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func decodeCollaboratorBody(t *testing.T, req *http.Request, target interface{}) {
	t.Helper()
	defer req.Body.Close()
	if err := json.NewDecoder(req.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func directCollaboratorJSON(username, permission string) string {
	return fmt.Sprintf(`{"username":%q,"permission":%q,"type":"ProjectMember","join_way":"normal","source_name":"demo","role_name":"Developer"}`, username, permission)
}

func collaboratorTestUsername(body string) string {
	if strings.Contains(body, `"username":"alice"`) {
		return "alice"
	}
	return "bob"
}
