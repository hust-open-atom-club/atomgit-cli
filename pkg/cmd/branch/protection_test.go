package branch

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

const protectionRulesJSON = `[
  {
    "name":"main",
    "updated_at":"2026-07-24T00:00:00Z",
    "master_can_push":1,
    "maintainer_can_push":1,
    "owner_can_push":1,
    "maintainer_can_merge":true,
    "owner_can_merge":true
  },
  {
    "name":"release/*",
    "developers_can_push":true,
    "push_users":[{"username":"alice"}],
    "master_can_merge":true,
    "merge_users":[{"login":"bob"}]
  }
]`

func TestProtectionListAndViewDistinguishExactAndWildcard(t *testing.T) {
	transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/protect_branches" {
			t.Fatalf("request = %s %s", req.Method, req.URL.EscapedPath())
		}
		return branchResponse(http.StatusOK, protectionRulesJSON), nil
	})
	factory := branchFactory(branchCommandConfig{token: "token"}, transport)

	list := newCmdProtectionList(factory)
	var listOut bytes.Buffer
	list.SetOut(&listOut)
	if err := list.RunE(list, []string{"alice/demo"}); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"main type:exact push:admin merge:maintainer",
		"release/* type:wildcard push:develop;alice merge:admin;bob",
	} {
		if !strings.Contains(listOut.String(), text) {
			t.Fatalf("list output missing %q:\n%s", text, listOut.String())
		}
	}

	view := newCmdProtectionView(factory)
	var viewOut bytes.Buffer
	view.SetOut(&viewOut)
	if err := view.RunE(view, []string{"alice/demo", "release/*"}); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Rule: release/*", "Type: wildcard", "Push: develop;alice", "Merge: admin;bob"} {
		if !strings.Contains(viewOut.String(), text) {
			t.Fatalf("view output missing %q:\n%s", text, viewOut.String())
		}
	}
}

func TestProtectionSetCreatesRule(t *testing.T) {
	requests := 0
	transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/api/v5/repos/alice/demo/protect_branches":
			return branchResponse(http.StatusOK, `[]`), nil
		case req.Method == http.MethodPut && req.URL.Path == "/api/v5/repos/alice/demo/branches/setting/new":
			var body map[string]string
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["wildcard"] != "release/*" || body["pusher"] != "maintainer" || body["merger"] != "admin" {
				t.Fatalf("body = %#v", body)
			}
			return branchResponse(http.StatusOK, `{}`), nil
		default:
			t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
		}
		return nil, nil
	})
	cmd := newCmdProtectionSet(branchFactory(branchCommandConfig{token: "token"}, transport))
	_ = cmd.Flags().Set("push", "maintainer")
	_ = cmd.Flags().Set("merge", "admin")
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(cmd, []string{"alice/demo", "release/*"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || !strings.Contains(out.String(), "Created protected branch rule release/*") {
		t.Fatalf("requests = %d, output = %q", requests, out.String())
	}
}

func TestProtectionSetPreservesOmittedPermissionAndConfirms(t *testing.T) {
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
			transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet:
					return branchResponse(http.StatusOK, `[{"name":"release/*","committer_can_push":true,"maintainer_can_merge":true}]`), nil
				case req.Method == http.MethodPut:
					writes++
					if req.URL.EscapedPath() != "/api/v5/repos/alice/demo/branches/release%2F%2A/setting" {
						t.Fatalf("path = %s", req.URL.EscapedPath())
					}
					var body map[string]string
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if _, present := body["wildcard"]; present || body["pusher"] != "" || body["merger"] != "maintainer" {
						t.Fatalf("body = %#v", body)
					}
					return branchResponse(http.StatusOK, `{}`), nil
				default:
					t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
				}
				return nil, nil
			})
			cmd := newCmdProtectionSet(branchFactory(branchCommandConfig{token: "token"}, transport))
			_ = cmd.Flags().Set("push", "")
			if tt.yes {
				_ = cmd.Flags().Set("yes", "true")
			}
			cmd.SetIn(strings.NewReader(tt.input))
			var out bytes.Buffer
			cmd.SetOut(&out)

			if err := cmd.RunE(cmd, []string{"alice/demo", "release/*"}); err != nil {
				t.Fatal(err)
			}
			if (writes == 1) != tt.wantWrite {
				t.Fatalf("writes = %d, output = %q", writes, out.String())
			}
			if !tt.wantWrite && !strings.Contains(out.String(), "cancelled") {
				t.Fatalf("output = %q", out.String())
			}
			if !tt.yes && (!strings.Contains(out.String(), "New Push: none") || !strings.Contains(out.String(), "New Merge: maintainer")) {
				t.Fatalf("proposed permissions were not shown: %q", out.String())
			}
		})
	}
}

func TestProtectionSetRefusesUnrepresentableOmittedPermission(t *testing.T) {
	tests := []struct {
		name string
		rule string
		want string
	}{
		{name: "owner", rule: `[{"name":"main","owner_can_merge":true}]`, want: "owner-only"},
		{name: "committer", rule: `[{"name":"main","committer_can_merge":true}]`, want: "committer-only"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writes := 0
			transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodPut {
					writes++
				}
				return branchResponse(http.StatusOK, tt.rule), nil
			})
			cmd := newCmdProtectionSet(branchFactory(branchCommandConfig{token: "token"}, transport))
			_ = cmd.Flags().Set("push", "admin")
			_ = cmd.Flags().Set("yes", "true")
			err := cmd.RunE(cmd, []string{"alice/demo", "main"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
			if writes != 0 {
				t.Fatalf("writes = %d", writes)
			}
		})
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
			transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return branchResponse(http.StatusOK, `[{"name":"release/*","no_one_can_push":true,"no_one_can_merge":true}]`), nil
				}
				if req.Method == http.MethodDelete {
					deletes++
					if req.URL.EscapedPath() != "/api/v5/repos/alice/demo/branches/release%2F%2A/setting" {
						t.Fatalf("path = %s", req.URL.EscapedPath())
					}
					return branchResponse(http.StatusNoContent, ""), nil
				}
				t.Fatalf("unexpected request = %s %s", req.Method, req.URL.EscapedPath())
				return nil, nil
			})
			cmd := newCmdProtectionDelete(branchFactory(branchCommandConfig{token: "token"}, transport))
			if tt.yes {
				_ = cmd.Flags().Set("yes", "true")
			}
			cmd.SetIn(strings.NewReader(tt.input))
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.RunE(cmd, []string{"alice/demo", "release/*"}); err != nil {
				t.Fatal(err)
			}
			if (deletes == 1) != tt.wantDelete {
				t.Fatalf("deletes = %d, output = %q", deletes, out.String())
			}
		})
	}
}

func TestProtectionValidationStopsBeforeRequests(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		push    string
		merge   string
		want    string
	}{
		{name: "empty pattern", pattern: " ", push: "admin", merge: "admin", want: "pattern"},
		{name: "backslash pattern", pattern: `release\\*`, push: "admin", merge: "admin", want: "pattern"},
		{name: "missing permissions", pattern: "main", want: "at least one"},
		{name: "empty segment", pattern: "main", push: "admin;;alice", want: "invalid --push"},
		{name: "spaces", pattern: "main", push: "admin; alice", want: "invalid --push"},
		{name: "duplicate", pattern: "main", push: "admin;admin", want: "duplicate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			cmd := newCmdProtectionSet(branchFactory(branchCommandConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				requests++
				return branchResponse(http.StatusOK, `[]`), nil
			}))
			if tt.push != "" || tt.name == "empty segment" || tt.name == "spaces" || tt.name == "duplicate" {
				_ = cmd.Flags().Set("push", tt.push)
			}
			if tt.merge != "" {
				_ = cmd.Flags().Set("merge", tt.merge)
			}
			err := cmd.RunE(cmd, []string{"alice/demo", tt.pattern})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
			if requests != 0 {
				t.Fatalf("requests = %d", requests)
			}
		})
	}
}

func TestProtectionSetRequiresBothPermissionsForNewRule(t *testing.T) {
	transport := branchRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return branchResponse(http.StatusOK, `[]`), nil
		}
		t.Fatalf("unexpected write request = %s %s", req.Method, req.URL.EscapedPath())
		return nil, nil
	})
	cmd := newCmdProtectionSet(branchFactory(branchCommandConfig{token: "token"}, transport))
	_ = cmd.Flags().Set("push", "admin")
	err := cmd.RunE(cmd, []string{"alice/demo", "main"})
	if err == nil || !strings.Contains(err.Error(), "require both") {
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
			factory := branchFactory(branchCommandConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				return branchResponse(tt.status, `{"message":"failed"}`), nil
			})
			cmd := newCmdProtectionList(factory)
			err := cmd.RunE(cmd, []string{"alice/demo"})
			if err == nil || !strings.Contains(err.Error(), http.StatusText(tt.status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestProtectionAuthenticationErrorDoesNotRequest(t *testing.T) {
	requests := 0
	factory := branchFactory(branchCommandConfig{tokenErr: errors.New("missing token")}, func(*http.Request) (*http.Response, error) {
		requests++
		return branchResponse(http.StatusOK, `[]`), nil
	})
	cmd := newCmdProtectionList(factory)
	err := cmd.RunE(cmd, []string{"alice/demo"})
	if err == nil || !strings.Contains(err.Error(), "missing token") || requests != 0 {
		t.Fatalf("error = %v, requests = %d", err, requests)
	}
}
