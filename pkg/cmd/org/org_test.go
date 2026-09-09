package org

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

type orgTestConfig struct {
	tokenErr error
}

func (c orgTestConfig) GetToken() (string, error) { return "token", c.tokenErr }
func (c orgTestConfig) GetUser() (string, error)  { return "alice", nil }
func (c orgTestConfig) GetHost() string           { return "atomgit.com" }

type orgRoundTripFunc func(*http.Request) (*http.Response, error)

func (f orgRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func orgResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func orgFactory(config orgTestConfig, transport orgRoundTripFunc) *cmdutil.Factory {
	factory := &cmdutil.Factory{Config: config}
	if transport != nil {
		factory.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		}
	}
	return factory
}

func TestNewCmdOrgRegistersList(t *testing.T) {
	cmd := NewCmdOrg(&cmdutil.Factory{})
	list, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"limit", "json"} {
		if list.Flags().Lookup(flag) == nil {
			t.Fatalf("list flag %q was not registered", flag)
		}
	}
	if !strings.Contains(list.Example, "ag org list --json") {
		t.Fatalf("list examples = %q", list.Example)
	}
	if err := list.Args(list, []string{"extra"}); err == nil {
		t.Fatal("org list accepted an argument")
	}
}

func TestOrgListTextOutput(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		contains []string
	}{
		{name: "empty", body: `[]`, contains: []string{"No organizations found."}},
		{
			name: "organizations",
			body: `[
				{"id":1,"path":"open-source","name":"Open Source","html_url":"https://atomgit.com/open-source"},
				{"id":2,"login":"fallback-path","name":"  Multi\n Line  "}
			]`,
			contains: []string{"PATH", "NAME", "URL", "open-source", "Open Source", "https://atomgit.com/open-source", "fallback-path", "Multi Line", "https://atomgit.com/fallback-path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/users/orgs" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("query = %q", req.URL.RawQuery)
				}
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("Authorization = %q", got)
				}
				return orgResponse(http.StatusOK, tt.body), nil
			})
			cmd := newCmdOrgList(factory)
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, nil); err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d", requests)
			}
			for _, want := range tt.contains {
				if !strings.Contains(output.String(), want) {
					t.Fatalf("output missing %q:\n%s", want, output.String())
				}
			}
		})
	}
}

func TestOrgListJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "empty", body: `[]`, want: `[]`},
		{name: "organization", body: `[{"id":7,"path":"team","name":"Team","description":"Description","html_url":"https://example.test/team"}]`, want: `[{"id":7,"path":"team","name":"Team","url":"https://example.test/team","description":"Description"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
				return orgResponse(http.StatusOK, tt.body), nil
			})
			cmd := newCmdOrgList(factory)
			_ = cmd.Flags().Set("json", "true")
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, nil); err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(output.Bytes(), &got); err != nil {
				t.Fatalf("invalid JSON %q: %v", output.String(), err)
			}
			if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
				t.Fatal(err)
			}
			if fmt.Sprintf("%#v", got) != fmt.Sprintf("%#v", want) {
				t.Fatalf("JSON = %s, want %s", output.String(), tt.want)
			}
		})
	}
}

func TestOrgListPaginatesAndHonorsLimit(t *testing.T) {
	requests := 0
	factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Query().Get("page") != fmt.Sprint(requests) || req.URL.Query().Get("per_page") != "100" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		count := 100
		start := 1
		if requests == 2 {
			count = 2
			start = 101
		}
		organizations := make([]api.Organization, count)
		for index := range organizations {
			organizations[index].Path = fmt.Sprintf("org-%d", start+index)
		}
		body, err := json.Marshal(organizations)
		if err != nil {
			t.Fatal(err)
		}
		return orgResponse(http.StatusOK, string(body)), nil
	})
	cmd := newCmdOrgList(factory)
	_ = cmd.Flags().Set("limit", "101")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
	if !strings.Contains(output.String(), "org-101") || strings.Contains(output.String(), "org-102") {
		t.Fatalf("output did not honor limit:\n%s", output.String())
	}
}

func TestOrgListValidationBeforeRequest(t *testing.T) {
	requests := 0
	transport := func(*http.Request) (*http.Response, error) {
		requests++
		return orgResponse(http.StatusOK, `[]`), nil
	}

	cmd := newCmdOrgList(orgFactory(orgTestConfig{}, transport))
	_ = cmd.Flags().Set("limit", "0")
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("error = %v", err)
	}

	cmd = newCmdOrgList(orgFactory(orgTestConfig{tokenErr: config.ErrNotAuthenticated}, transport))
	if err := cmd.RunE(cmd, nil); err != config.ErrNotAuthenticated {
		t.Fatalf("error = %v, want canonical authentication error", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestOrgListReportsResponseErrors(t *testing.T) {
	tests := []struct {
		name       string
		firstBody  string
		firstCode  int
		secondCode int
		want       string
	}{
		{name: "forbidden", firstCode: http.StatusForbidden, firstBody: `{}`, want: "403"},
		{name: "rate limited", firstCode: http.StatusTooManyRequests, firstBody: `{}`, want: "429"},
		{name: "server error", firstCode: http.StatusInternalServerError, firstBody: `{}`, want: "500"},
		{name: "malformed response", firstCode: http.StatusOK, firstBody: `{}`, want: "cannot unmarshal"},
		{name: "later page failure", firstCode: http.StatusOK, secondCode: http.StatusInternalServerError, want: "500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
				requests++
				if requests == 2 {
					return orgResponse(tt.secondCode, `{}`), nil
				}
				body := tt.firstBody
				if tt.secondCode != 0 {
					organizations := make([]api.Organization, 100)
					data, err := json.Marshal(organizations)
					if err != nil {
						t.Fatal(err)
					}
					body = string(data)
				}
				return orgResponse(tt.firstCode, body), nil
			})
			cmd := newCmdOrgList(factory)
			if tt.secondCode != 0 {
				_ = cmd.Flags().Set("limit", "101")
			}
			err := cmd.RunE(cmd, nil)
			if err == nil || !strings.Contains(err.Error(), "failed to list organizations") || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
