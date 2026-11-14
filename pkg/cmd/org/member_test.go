package org

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestOrgMemberWrites(t *testing.T) {
	for _, tt := range []struct {
		name, action, input, response, permission, roleID string
		flags                                             []string
	}{
		{name: "default", action: "invite", input: "y\n", response: `{}`, permission: "push"},
		{name: "pull", action: "invite", input: "YES\n", response: `{"id":"string-id"}`, permission: "pull", flags: []string{"--permission", "pull"}},
		{name: "push", action: "invite", permission: "push", flags: []string{"--permission", "push", "--yes"}},
		{name: "admin", action: "invite", permission: "admin", flags: []string{"--permission", "admin", "-y"}},
		{name: "custom", action: "invite", input: "yes\n", permission: "customized", roleID: "role-123", flags: []string{"--permission", "customized", "--role-id", "role-123"}},
		{name: "remove empty", action: "remove", input: "y\n"},
		{name: "remove object", action: "remove", response: `{}`, flags: []string{"--yes"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				calls++
				method := http.MethodPost
				if tt.action == "remove" {
					method = http.MethodDelete
				}
				if req.Method != method || req.URL.Path != "/api/v5/orgs/team/memberships/alice" || req.Header.Get("Authorization") != "Bearer token" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
				}
				if tt.action == "invite" {
					var body map[string]string
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					wantFields := 1
					if tt.roleID != "" {
						wantFields++
					}
					if body["permission"] != tt.permission || body["role_id"] != tt.roleID || len(body) != wantFields || req.Header.Get("Content-Type") != "application/json" {
						t.Fatalf("body=%v headers=%v", body, req.Header)
					}
				} else if req.Body != nil {
					t.Fatal("DELETE has a body")
				}
				return orgResponse(http.StatusOK, tt.response), nil
			})
			cmd := NewCmdOrg(factory)
			var out, prompt bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&prompt)
			cmd.SetIn(strings.NewReader(tt.input))
			cmd.SetArgs(append([]string{"member", tt.action, "team", "alice"}, tt.flags...))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("requests=%d", calls)
			}
			if tt.input != "" {
				for _, want := range []string{"team", "alice", tt.permission, tt.roleID} {
					if !strings.Contains(prompt.String(), want) {
						t.Fatalf("prompt=%q missing %q", prompt.String(), want)
					}
				}
			} else if prompt.Len() != 0 {
				t.Fatalf("unexpected prompt %q", prompt.String())
			}
			want := "Removed member alice from organization team"
			if tt.action == "invite" {
				want = "Acceptance by the user is not verified."
			}
			if !strings.Contains(out.String(), want) {
				t.Fatalf("output=%q", out.String())
			}
		})
	}
}

func TestOrgMemberCancellation(t *testing.T) {
	for _, action := range []string{"invite", "remove"} {
		for _, input := range []string{"", "\n", "n\n", "anything\n"} {
			t.Run(action+"/"+fmt.Sprintf("%q", input), func(t *testing.T) {
				tokens := 0
				cmd := NewCmdOrg(orgFactory(orgTestConfig{tokenCalls: &tokens}, func(*http.Request) (*http.Response, error) {
					t.Fatal("cancelled command made a request")
					return nil, nil
				}))
				var out, prompt bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&prompt)
				cmd.SetIn(strings.NewReader(input))
				cmd.SetArgs([]string{"member", action, "team", "alice"})
				if err := cmd.Execute(); err != nil {
					t.Fatal(err)
				}
				if tokens != 0 || !strings.Contains(out.String(), "cancelled") {
					t.Fatalf("tokens=%d output=%q", tokens, out.String())
				}
			})
		}
	}
}

func TestOrgMemberInvalidArgumentsBeforeAuthentication(t *testing.T) {
	cases := [][]string{
		{"invite"}, {"invite", "team"}, {"invite", "team", "alice", "extra"},
		{"remove"}, {"remove", "team"}, {"remove", "team", "alice", "extra"},
		{"invite", "team", "alice", "--permission", "owner"},
		{"invite", "team", "alice", "--permission="},
		{"invite", "team", "alice", "--permission", "customized"},
		{"invite", "team", "alice", "--permission", "customized", "--role-id", "  "},
		{"invite", "team", "alice", "--role-id", "role-123"},
		{"invite", "team", "alice", "--permission", "admin", "--role-id="},
	}
	for _, action := range []string{"invite", "remove"} {
		for _, invalid := range []string{"", ".", "..", "a/b", "a?b", "a\nb"} {
			cases = append(cases, []string{action, invalid, "alice"}, []string{action, "team", invalid})
		}
	}
	for _, args := range cases {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			tokens := 0
			cmd := NewCmdOrg(orgFactory(orgTestConfig{tokenCalls: &tokens}, func(*http.Request) (*http.Response, error) {
				t.Fatal("invalid input made a request")
				return nil, nil
			}))
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(append([]string{"member"}, args...))
			if err := cmd.Execute(); err == nil {
				t.Fatal("accepted invalid input")
			}
			if tokens != 0 || out.Len() != 0 {
				t.Fatalf("tokens=%d output=%q", tokens, out.String())
			}
		})
	}
}

func TestOrgMemberFailures(t *testing.T) {
	for _, action := range []string{"invite", "remove"} {
		for _, status := range []int{400, 403, 404} {
			t.Run(fmt.Sprintf("%s/%d", action, status), func(t *testing.T) {
				calls := 0
				cmd := NewCmdOrg(orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
					calls++
					return orgResponse(status, `{"message":"user does not exist or permission denied"}`), nil
				}))
				var out bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&out)
				cmd.SilenceErrors = true
				cmd.SilenceUsage = true
				cmd.SetArgs([]string{"member", action, "team", "alice", "--yes"})
				err := cmd.Execute()
				if err == nil || !strings.Contains(err.Error(), "team") || !strings.Contains(err.Error(), "alice") || calls != 1 || out.Len() != 0 {
					t.Fatalf("err=%v requests=%d output=%q", err, calls, out.String())
				}
			})
		}
		t.Run(action+"/authentication", func(t *testing.T) {
			cause := errors.New("missing credentials")
			cmd := NewCmdOrg(orgFactory(orgTestConfig{tokenErr: cause}, func(*http.Request) (*http.Response, error) {
				t.Fatal("unauthenticated request")
				return nil, nil
			}))
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"member", action, "team", "alice", "--yes"})
			if err := cmd.Execute(); !errors.Is(err, cause) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
