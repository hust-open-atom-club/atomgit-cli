package tag

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type tagProtectionConfig struct {
	token    string
	tokenErr error
}

func (c tagProtectionConfig) GetToken() (string, error) { return c.token, c.tokenErr }
func (c tagProtectionConfig) GetUser() (string, error)  { return "alice", nil }
func (c tagProtectionConfig) GetHost() string           { return "atomgit.com" }

func tagProtectionFactory(config tagProtectionConfig, transport tagRoundTripFunc) *cmdutil.Factory {
	factory := &cmdutil.Factory{Config: config}
	if transport != nil {
		factory.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return factory
}

func tagProtectionResponse(status int, body string) *http.Response {
	if body == "" {
		body = "{}"
	}
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

const protectedTagsFixture = `[
  {
    "name":"v1.0.0",
    "create_access_level":40,
    "create_access_level_desc":"Maintainer, Admin"
  },
  {
    "name":"v*",
    "create_access_level":30,
    "create_access_level_desc":"Developer, Maintainer, Admin"
  }
]`

func TestProtectionListAndViewDistinguishExactAndWildcard(t *testing.T) {
	transport := tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/protected_tags":
			return tagProtectionResponse(http.StatusOK, protectedTagsFixture), nil
		case req.Method == http.MethodGet && req.URL.EscapedPath() == "/api/v5/repos/alice/demo/protected_tags/v%2A":
			return tagProtectionResponse(http.StatusOK, `{
				"name":"v*",
				"create_access_level":30,
				"create_access_level_desc":"Developer, Maintainer, Admin"
			}`), nil
		default:
			t.Fatalf("request = %s %s", req.Method, req.URL.EscapedPath())
			return nil, nil
		}
	})
	factory := tagProtectionFactory(tagProtectionConfig{token: "token"}, transport)

	list := newCmdTagProtectionList(factory)
	var listOut bytes.Buffer
	list.SetOut(&listOut)
	if err := list.RunE(list, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"v1.0.0 type:exact create-access:maintainer",
		"v* type:wildcard create-access:developer",
	} {
		if !strings.Contains(listOut.String(), text) {
			t.Fatalf("list output missing %q:\n%s", text, listOut.String())
		}
	}

	view := newCmdTagProtectionView(factory)
	var viewOut bytes.Buffer
	view.SetOut(&viewOut)
	if err := view.RunE(view, []string{"alice/demo", "v*"}); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Repository: alice/demo", "Rule: v*", "Type: wildcard", "Create access: developer"} {
		if !strings.Contains(viewOut.String(), text) {
			t.Fatalf("view output missing %q:\n%s", text, viewOut.String())
		}
	}
}

func TestProtectionListJSONAndEmpty(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "rules", body: protectedTagsFixture, want: 2},
		{name: "empty", body: `[]`, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := tagProtectionFactory(tagProtectionConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				return tagProtectionResponse(http.StatusOK, tt.body), nil
			})
			cmd := newCmdTagProtectionList(factory)
			_ = cmd.Flags().Set("json", "true")
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
				t.Fatal(err)
			}
			var values []protectedTagJSON
			if err := json.Unmarshal(out.Bytes(), &values); err != nil {
				t.Fatalf("invalid JSON %q: %v", out.String(), err)
			}
			if len(values) != tt.want {
				t.Fatalf("values = %#v", values)
			}
			if tt.want == 0 {
				return
			}
			if values[0].Name != "v*" || values[0].Type != "wildcard" || values[0].CreateAccess != "developer" || values[0].CreateAccessLevel != 30 {
				t.Fatalf("first = %#v", values[0])
			}
			if values[1].Name != "v1.0.0" || values[1].Type != "exact" || values[1].CreateAccess != "maintainer" {
				t.Fatalf("second = %#v", values[1])
			}
		})
	}
}

func TestProtectionListEmptyText(t *testing.T) {
	factory := tagProtectionFactory(tagProtectionConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		return tagProtectionResponse(http.StatusOK, `[]`), nil
	})
	cmd := newCmdTagProtectionList(factory)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "No protected tag rules found\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestProtectionCommandsInferRepositoryContext(t *testing.T) {
	requests := 0
	factory := tagProtectionFactory(tagProtectionConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != "/api/v5/repos/alice/demo/protected_tags" && !strings.HasPrefix(req.URL.Path, "/api/v5/repos/alice/demo/protected_tags/") {
			t.Fatalf("path = %s", req.URL.Path)
		}
		if req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/protected_tags") {
			return tagProtectionResponse(http.StatusOK, `[]`), nil
		}
		if req.Method == http.MethodGet {
			return tagProtectionResponse(http.StatusNotFound, `{"message":"not found"}`), nil
		}
		return tagProtectionResponse(http.StatusOK, `{}`), nil
	})
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
	}

	list := newCmdTagProtectionList(factory)
	if err := list.RunE(list, nil); err != nil {
		t.Fatal(err)
	}
	view := newCmdTagProtectionView(factory)
	if err := view.RunE(view, []string{"v1.0.0"}); err == nil || !strings.Contains(err.Error(), "was not found") {
		t.Fatalf("view error = %v", err)
	}
	if requests == 0 {
		t.Fatal("expected inferred-repository requests")
	}
}

func TestProtectionSetCreatesRule(t *testing.T) {
	requests := 0
	transport := tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		switch {
		case req.Method == http.MethodGet && req.URL.EscapedPath() == "/api/v5/repos/alice/demo/protected_tags/v%2A":
			return tagProtectionResponse(http.StatusNotFound, `{"message":"not found"}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/api/v5/repos/alice/demo/protected_tags":
			var body api.ProtectedTagRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Name != "v*" || body.CreateAccessLevel != api.ProtectedTagCreateAccessDeveloper {
				t.Fatalf("body = %#v", body)
			}
			return tagProtectionResponse(http.StatusCreated, `{"name":"v*","create_access_level":30}`), nil
		default:
			t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
			return nil, nil
		}
	})
	cmd := newCmdTagProtectionSet(tagProtectionFactory(tagProtectionConfig{token: "token"}, transport))
	_ = cmd.Flags().Set("create-access", "developer")
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo", "v*"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || !strings.Contains(out.String(), "Created protected tag rule v*") {
		t.Fatalf("requests = %d, output = %q", requests, out.String())
	}
}

func TestProtectionSetPreservesOmittedAccessAndConfirms(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		yes       bool
		wantWrite bool
	}{
		{name: "confirmed", input: "yes\n", wantWrite: true},
		{name: "cancelled", input: "no\n"},
		{name: "yes flag", yes: true, wantWrite: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writes := 0
			transport := tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet:
					if req.URL.EscapedPath() != "/api/v5/repos/alice/demo/protected_tags/v1.0.0" {
						t.Fatalf("path = %s", req.URL.EscapedPath())
					}
					return tagProtectionResponse(http.StatusOK, `{"name":"v1.0.0","create_access_level":40}`), nil
				case req.Method == http.MethodPut:
					writes++
					if req.URL.Path != "/api/v5/repos/alice/demo/protected_tags" {
						t.Fatalf("path = %s", req.URL.Path)
					}
					var body api.ProtectedTagRequest
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if body.Name != "v1.0.0" || body.CreateAccessLevel != api.ProtectedTagCreateAccessNone {
						t.Fatalf("body = %#v", body)
					}
					return tagProtectionResponse(http.StatusOK, `{}`), nil
				default:
					t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
					return nil, nil
				}
			})
			cmd := newCmdTagProtectionSet(tagProtectionFactory(tagProtectionConfig{token: "token"}, transport))
			_ = cmd.Flags().Set("create-access", "none")
			if tt.yes {
				_ = cmd.Flags().Set("yes", "true")
			}
			cmd.SetIn(strings.NewReader(tt.input))
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.RunE(cmd, []string{"alice/demo", "v1.0.0"}); err != nil {
				t.Fatal(err)
			}
			if (writes == 1) != tt.wantWrite {
				t.Fatalf("writes = %d, output = %q", writes, out.String())
			}
			if !tt.wantWrite && !strings.Contains(out.String(), "cancelled") {
				t.Fatalf("output = %q", out.String())
			}
			if !tt.yes && (!strings.Contains(out.String(), "Create access: maintainer") || !strings.Contains(out.String(), "New create access: none")) {
				t.Fatalf("proposed access was not shown: %q", out.String())
			}
		})
	}
}

func TestProtectionSetPreservesCurrentAccessWhenFlagOmitted(t *testing.T) {
	writes := 0
	transport := tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet:
			return tagProtectionResponse(http.StatusOK, `{"name":"v1.0.0","create_access_level":30}`), nil
		case req.Method == http.MethodPut:
			writes++
			var body api.ProtectedTagRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Name != "v1.0.0" || body.CreateAccessLevel != api.ProtectedTagCreateAccessDeveloper {
				t.Fatalf("body = %#v", body)
			}
			return tagProtectionResponse(http.StatusOK, `{}`), nil
		default:
			t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
			return nil, nil
		}
	})
	cmd := newCmdTagProtectionSet(tagProtectionFactory(tagProtectionConfig{token: "token"}, transport))
	_ = cmd.Flags().Set("yes", "true")
	if err := cmd.RunE(cmd, []string{"alice/demo", "v1.0.0"}); err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Fatalf("writes = %d", writes)
	}
}

func TestProtectionSetRefusesUnsupportedExistingAccess(t *testing.T) {
	writes := 0
	transport := tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return tagProtectionResponse(http.StatusOK, `{"name":"v1.0.0","create_access_level":15}`), nil
		}
		writes++
		return tagProtectionResponse(http.StatusOK, `{}`), nil
	})
	cmd := newCmdTagProtectionSet(tagProtectionFactory(tagProtectionConfig{token: "token"}, transport))
	_ = cmd.Flags().Set("yes", "true")
	err := cmd.RunE(cmd, []string{"alice/demo", "v1.0.0"})
	if err == nil || !strings.Contains(err.Error(), "cannot preserve create access") || writes != 0 {
		t.Fatalf("error = %v, writes = %d", err, writes)
	}
}

func TestProtectionDeleteCancellationAndYes(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		yes        bool
		wantDelete bool
	}{
		{name: "cancelled", input: "no\n"},
		{name: "confirmed", input: "yes\n", wantDelete: true},
		{name: "yes flag", yes: true, wantDelete: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deletes := 0
			transport := tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					if req.URL.EscapedPath() != "/api/v5/repos/alice/demo/protected_tags/v%2A" {
						t.Fatalf("path = %s", req.URL.EscapedPath())
					}
					return tagProtectionResponse(http.StatusOK, `{"name":"v*","create_access_level":0}`), nil
				}
				if req.Method == http.MethodDelete {
					deletes++
					if req.URL.EscapedPath() != "/api/v5/repos/alice/demo/protected_tags/v%2A" {
						t.Fatalf("path = %s", req.URL.EscapedPath())
					}
					return tagProtectionResponse(http.StatusNoContent, ""), nil
				}
				t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
				return nil, nil
			})
			cmd := newCmdTagProtectionDelete(tagProtectionFactory(tagProtectionConfig{token: "token"}, transport))
			if tt.yes {
				_ = cmd.Flags().Set("yes", "true")
			}
			cmd.SetIn(strings.NewReader(tt.input))
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.RunE(cmd, []string{"alice/demo", "v*"}); err != nil {
				t.Fatal(err)
			}
			if (deletes == 1) != tt.wantDelete {
				t.Fatalf("deletes = %d, output = %q", deletes, out.String())
			}
			if !tt.wantDelete && !strings.Contains(out.String(), "cancelled") {
				t.Fatalf("output = %q", out.String())
			}
			if !tt.yes && !strings.Contains(out.String(), "Repository: alice/demo") {
				t.Fatalf("current rule was not shown: %q", out.String())
			}
		})
	}
}

func TestProtectionValidationStopsBeforeRequests(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		access string
		set    bool
		want   string
	}{
		{name: "empty pattern", args: []string{"alice/demo", " "}, access: "maintainer", set: true, want: "tag or wildcard pattern is required"},
		{name: "backslash pattern", args: []string{"alice/demo", `v\\*`}, access: "maintainer", set: true, want: "invalid tag or wildcard pattern"},
		{name: "invalid access", args: []string{"alice/demo", "v1.0.0"}, access: "admin", set: true, want: "invalid --create-access"},
		{name: "blank view", args: []string{"alice/demo", " \t "}, want: "tag or wildcard pattern is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := tagProtectionFactory(tagProtectionConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				requests++
				return tagProtectionResponse(http.StatusOK, `{}`), nil
			})
			if tt.set {
				setCmd := newCmdTagProtectionSet(factory)
				if tt.access != "" || tt.name == "invalid access" {
					_ = setCmd.Flags().Set("create-access", tt.access)
				}
				err := setCmd.RunE(setCmd, tt.args)
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("error = %v", err)
				}
			} else {
				viewCmd := newCmdTagProtectionView(factory)
				err := viewCmd.RunE(viewCmd, tt.args)
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("error = %v", err)
				}
			}
			if requests != 0 {
				t.Fatalf("requests = %d", requests)
			}
		})
	}
}

func TestProtectionSetRequiresAccessForNewRule(t *testing.T) {
	transport := tagRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return tagProtectionResponse(http.StatusNotFound, `{"message":"not found"}`), nil
		}
		t.Fatalf("unexpected write request = %s %s", req.Method, req.URL.EscapedPath())
		return nil, nil
	})
	cmd := newCmdTagProtectionSet(tagProtectionFactory(tagProtectionConfig{token: "token"}, transport))
	err := cmd.RunE(cmd, []string{"alice/demo", "v1.0.0"})
	if err == nil || !strings.Contains(err.Error(), "require --create-access") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtectionCommandsReportAPIErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{name: "permission", status: http.StatusForbidden},
		{name: "not found", status: http.StatusNotFound},
		{name: "conflict", status: http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := tagProtectionFactory(tagProtectionConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				return tagProtectionResponse(tt.status, `{"message":"failed"}`), nil
			})
			cmd := newCmdTagProtectionList(factory)
			err := cmd.RunE(cmd, []string{"alice/demo"})
			if err == nil || !strings.Contains(err.Error(), http.StatusText(tt.status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestProtectionViewReportsMissingRuleWithoutRawAPIError(t *testing.T) {
	factory := tagProtectionFactory(tagProtectionConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		return tagProtectionResponse(http.StatusNotFound, `{"message":"failed"}`), nil
	})
	cmd := newCmdTagProtectionView(factory)
	err := cmd.RunE(cmd, []string{"alice/demo", "v1.0.0"})
	if err == nil || !strings.Contains(err.Error(), `protected tag rule "v1.0.0" was not found`) {
		t.Fatalf("error = %v", err)
	}
}

func TestProtectionAuthenticationErrorDoesNotRequest(t *testing.T) {
	requests := 0
	factory := tagProtectionFactory(tagProtectionConfig{tokenErr: errors.New("missing token")}, func(*http.Request) (*http.Response, error) {
		requests++
		return tagProtectionResponse(http.StatusOK, `[]`), nil
	})
	cmd := newCmdTagProtectionList(factory)
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil || !strings.Contains(err.Error(), "missing token") || requests != 0 {
		t.Fatalf("error = %v, requests = %d", err, requests)
	}
}

func TestProtectionListPaginatesAndHonorsLimit(t *testing.T) {
	requests := 0
	factory := tagProtectionFactory(tagProtectionConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Query().Get("page") != strconv.Itoa(requests) || req.URL.Query().Get("per_page") != "100" {
			t.Fatalf("request URL = %s", req.URL.String())
		}
		if requests == 1 {
			return tagProtectionResponse(http.StatusOK, protectedTagsWithNames(100)), nil
		}
		return tagProtectionResponse(http.StatusOK, `[{"name":"z-last","create_access_level":40}]`), nil
	})
	cmd := newCmdTagProtectionList(factory)
	_ = cmd.Flags().Set("limit", "101")
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 101 {
		t.Fatalf("lines = %d\n%s", len(lines), out.String())
	}
}

func TestProtectionListRejectsNonPositiveLimitBeforeAuthentication(t *testing.T) {
	cfg := &recordingConfig{}
	cmd := newCmdTagProtectionList(&cmdutil.Factory{Config: cfg})
	_ = cmd.Flags().Set("limit", "0")
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil || !strings.Contains(err.Error(), "invalid limit") || cfg.getTokenCalls != 0 {
		t.Fatalf("error = %v, token calls = %d", err, cfg.getTokenCalls)
	}
}

func TestProtectionMutationsRejectInvalidInputBeforeAuth(t *testing.T) {
	commands := []struct {
		name string
		new  func(*cmdutil.Factory) *cobra.Command
	}{
		{name: "set", new: newCmdTagProtectionSet},
		{name: "delete", new: newCmdTagProtectionDelete},
		{name: "view", new: newCmdTagProtectionView},
	}
	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			cfg := &recordingConfig{}
			cmd := command.new(&cmdutil.Factory{Config: cfg})
			err := cmd.RunE(cmd, []string{"alice/demo", " "})
			if err == nil || !strings.Contains(err.Error(), "tag or wildcard pattern is required") {
				t.Fatalf("error = %v", err)
			}
			if cfg.getTokenCalls != 0 {
				t.Fatalf("GetToken was called %d times", cfg.getTokenCalls)
			}
		})
	}
}

func protectedTagsWithNames(count int) string {
	rules := make([]string, count)
	for i := range rules {
		rules[i] = fmt.Sprintf(`{"name":"rule-%03d","create_access_level":40}`, i)
	}
	return "[" + strings.Join(rules, ",") + "]"
}
