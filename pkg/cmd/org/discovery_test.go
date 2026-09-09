package org

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func TestNewCmdOrgRegistersDiscoveryCommands(t *testing.T) {
	cmd := NewCmdOrg(&cmdutil.Factory{})
	tests := []struct {
		name  string
		flags []string
	}{
		{name: "view", flags: []string{"json"}},
		{name: "members", flags: []string{"limit", "json"}},
		{name: "repos", flags: []string{"limit", "json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subcommand, _, err := cmd.Find([]string{tt.name})
			if err != nil {
				t.Fatal(err)
			}
			for _, flag := range tt.flags {
				if subcommand.Flags().Lookup(flag) == nil {
					t.Fatalf("%s flag %q was not registered", tt.name, flag)
				}
			}
			if !strings.Contains(subcommand.Example, "ag org "+tt.name) {
				t.Fatalf("examples = %q", subcommand.Example)
			}
			if err := subcommand.Args(subcommand, nil); err == nil {
				t.Fatalf("org %s accepted no organization", tt.name)
			}
		})
	}
}

func TestOrgViewTextAndEscapedPath(t *testing.T) {
	factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.EscapedPath() != "/api/v5/orgs/my%20organization" {
			t.Fatalf("request = %s %s", req.Method, req.URL.EscapedPath())
		}
		return orgResponse(http.StatusOK, `{"id":7,"path":"my-organization","name":"My Organization","description":"Description","html_url":"https://example.test/my-organization","public":true}`), nil
	})
	cmd := newCmdOrgView(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"my organization"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Path: my-organization", "Name: My Organization", "Visibility: public", "Description: Description", "URL: https://example.test/my-organization"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, output.String())
		}
	}
}

func TestOrgViewJSON(t *testing.T) {
	factory := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, `{"id":7,"login":"fallback","name":"Team","description":"","public":false}`), nil
	})
	cmd := newCmdOrgView(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"fallback"}); err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, output.String(), `{"id":7,"path":"fallback","name":"Team","visibility":"private","description":"","url":"https://atomgit.com/fallback"}`)
}

func TestOrgMembersTextJSONAndEmpty(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		json     bool
		wantJSON string
		contains []string
	}{
		{name: "text", body: `[{"login":"alice","name":"Alice Example","member_role":"Owner"}]`, contains: []string{"LOGIN", "NAME", "ROLE", "alice", "Alice Example", "Owner"}},
		{name: "empty text", body: `[]`, contains: []string{`No members found for organization "team".`}},
		{name: "json", body: `[{"login":"alice","name":"Alice","member_role":"Maintainer"}]`, json: true, wantJSON: `[{"login":"alice","name":"Alice","role":"Maintainer"}]`},
		{name: "empty json", body: `[]`, json: true, wantJSON: `[]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/api/v5/orgs/team/members" || req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("request URL = %s", req.URL.String())
				}
				return orgResponse(http.StatusOK, tt.body), nil
			})
			cmd := newCmdOrgMembers(factory)
			if tt.json {
				_ = cmd.Flags().Set("json", "true")
			}
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"team"}); err != nil {
				t.Fatal(err)
			}
			if tt.wantJSON != "" {
				assertJSONEqual(t, output.String(), tt.wantJSON)
			}
			for _, want := range tt.contains {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("output missing %q:\n%s", want, output.String())
				}
			}
		})
	}
}

func TestOrgReposTextJSONAndEmpty(t *testing.T) {
	repositoryBody := `[{"id":9,"name":"demo","path":"demo-path","description":"Demo repository","html_url":"https://example.test/team/demo","private":false,"default_branch":"main","language":"Go","stargazers_count":12,"forks_count":3,"updated_at":"2026-09-08T10:00:00+08:00"}]`
	tests := []struct {
		name     string
		body     string
		json     bool
		wantJSON string
		contains []string
	}{
		{name: "text", body: repositoryBody, contains: []string{"PATH", "VISIBILITY", "DESCRIPTION", "DEFAULT BRANCH", "LANGUAGE", "STARS", "FORKS", "UPDATED", "URL", "demo-path", "public", "Demo repository", "main", "Go", "12", "3", "https://example.test/team/demo"}},
		{name: "empty text", body: `[]`, contains: []string{`No repositories found for organization "team".`}},
		{name: "json", body: repositoryBody, json: true, wantJSON: `[{"id":9,"name":"demo","path":"demo-path","visibility":"public","description":"Demo repository","defaultBranch":"main","language":"Go","stars":12,"forks":3,"updatedAt":"2026-09-08T10:00:00+08:00","url":"https://example.test/team/demo"}]`},
		{name: "empty json", body: `[]`, json: true, wantJSON: `[]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/api/v5/orgs/team/repos" {
					t.Fatalf("request URL = %s", req.URL.String())
				}
				return orgResponse(http.StatusOK, tt.body), nil
			})
			cmd := newCmdOrgRepos(factory)
			if tt.json {
				_ = cmd.Flags().Set("json", "true")
			}
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"team"}); err != nil {
				t.Fatal(err)
			}
			if tt.wantJSON != "" {
				assertJSONEqual(t, output.String(), tt.wantJSON)
			}
			for _, want := range tt.contains {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("output missing %q:\n%s", want, output.String())
				}
			}
		})
	}
}

func TestOrgDiscoveryCollectionsPaginateAndHonorLimit(t *testing.T) {
	tests := []struct {
		name string
		cmd  func(*cmdutil.Factory) *cobra.Command
		path string
		body func(int, int) string
		last string
	}{
		{name: "members", cmd: newCmdOrgMembers, path: "/api/v5/orgs/team/members", body: memberPageBody, last: "member-101"},
		{name: "repos", cmd: newCmdOrgRepos, path: "/api/v5/orgs/team/repos", body: repositoryPageBody, last: "repo-101"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				requests++
				if req.URL.Path != tt.path || req.URL.Query().Get("page") != fmt.Sprint(requests) || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("request URL = %s", req.URL.String())
				}
				count, start := 100, 1
				if requests == 2 {
					count, start = 2, 101
				}
				return orgResponse(http.StatusOK, tt.body(start, count)), nil
			})
			cmd := tt.cmd(factory)
			_ = cmd.Flags().Set("limit", "101")
			_ = cmd.Flags().Set("json", "true")
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"team"}); err != nil {
				t.Fatal(err)
			}
			if requests != 2 || !strings.Contains(output.String(), tt.last) || strings.Contains(output.String(), strings.Replace(tt.last, "101", "102", 1)) {
				t.Fatalf("requests = %d, output did not honor limit:\n%s", requests, output.String())
			}
		})
	}
}

func TestOrgDiscoveryValidatesBeforeAuthenticationAndRequest(t *testing.T) {
	requests := 0
	transport := func(*http.Request) (*http.Response, error) {
		requests++
		return orgResponse(http.StatusOK, `[]`), nil
	}
	tests := []struct {
		name string
		cmd  func(*cmdutil.Factory) *cobra.Command
		args []string
		flag string
	}{
		{name: "view blank organization", cmd: newCmdOrgView, args: []string{"  "}},
		{name: "members organization with slash", cmd: newCmdOrgMembers, args: []string{"owner/team"}},
		{name: "members invalid limit", cmd: newCmdOrgMembers, args: []string{"team"}, flag: "0"},
		{name: "repos invalid limit", cmd: newCmdOrgRepos, args: []string{"team"}, flag: "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmd(orgFactory(orgTestConfig{tokenErr: config.ErrNotAuthenticated}, transport))
			if tt.flag != "" {
				_ = cmd.Flags().Set("limit", tt.flag)
			}
			if err := cmd.RunE(cmd, tt.args); err == nil || err == config.ErrNotAuthenticated {
				t.Fatalf("error = %v, want validation error", err)
			}
		})
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestOrgDiscoveryReportsContextualResponseErrors(t *testing.T) {
	tests := []struct {
		name       string
		cmd        func(*cmdutil.Factory) *cobra.Command
		status     int
		wantPrefix string
	}{
		{name: "view unauthorized", cmd: newCmdOrgView, status: http.StatusUnauthorized, wantPrefix: "failed to view organization"},
		{name: "members forbidden", cmd: newCmdOrgMembers, status: http.StatusForbidden, wantPrefix: "failed to list members"},
		{name: "members rate limited", cmd: newCmdOrgMembers, status: http.StatusTooManyRequests, wantPrefix: "failed to list members"},
		{name: "repos server error", cmd: newCmdOrgRepos, status: http.StatusInternalServerError, wantPrefix: "failed to list repositories"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
				return orgResponse(tt.status, `{}`), nil
			})
			cmd := tt.cmd(factory)
			err := cmd.RunE(cmd, []string{"team"})
			if err == nil || !strings.Contains(err.Error(), tt.wantPrefix) || !strings.Contains(err.Error(), fmt.Sprint(tt.status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func assertJSONEqual(t *testing.T, got, want string) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal([]byte(got), &gotValue); err != nil {
		t.Fatalf("invalid JSON %q: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%#v", gotValue) != fmt.Sprintf("%#v", wantValue) {
		t.Fatalf("JSON = %s, want %s", got, want)
	}
}

func memberPageBody(start, count int) string {
	members := make([]api.OrganizationMember, count)
	for index := range members {
		members[index].Login = fmt.Sprintf("member-%d", start+index)
	}
	data, _ := json.Marshal(members)
	return string(data)
}

func repositoryPageBody(start, count int) string {
	repositories := make([]api.Repository, count)
	for index := range repositories {
		repositories[index].Name = fmt.Sprintf("repo-%d", start+index)
		repositories[index].Path = repositories[index].Name
	}
	data, _ := json.Marshal(repositories)
	return string(data)
}
