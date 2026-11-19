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
)

func TestOrgFollowersOutput(t *testing.T) {
	const body = `[{"id":496,"login":"alice","name":"Alice","avatar_url":"https://example.test/avatar.png","watch_at":"2024-11-13T16:15:53.287+08:00"}]`
	for _, tc := range []struct {
		name, body, wantJSON string
		contains             []string
	}{
		{name: "text", body: body, contains: []string{"LOGIN", "NAME", "WATCH AT", "alice", "Alice", "2024-11-13T16:15:53.287+08:00"}},
		{name: "JSON", body: body, wantJSON: body},
		{name: "empty text", body: `[]`, contains: []string{`No followers found for organization "my organization".`}},
		{name: "empty JSON", body: `[]`, wantJSON: `[]`},
		{name: "missing fields JSON", body: `[{}]`, wantJSON: `[{"id":0,"login":"","name":"","avatar_url":"","watch_at":""}]`},
		{name: "blank text", body: `[{"login":"alice","name":"  \n ","watch_at":""}]`, contains: []string{"alice", "-"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			f := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/api/v5/orgs/my%20organization/followers" || req.Header.Get("Authorization") != "Bearer token" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
				}
				if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("query = %q", req.URL.RawQuery)
				}
				return orgResponse(http.StatusOK, tc.body), nil
			})
			cmd := newCmdOrgFollowers(f)
			args := []string{"my organization"}
			if tc.wantJSON != "" {
				args = append(args, "--json")
			}
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want 1", requests)
			}
			if tc.wantJSON != "" {
				assertJSONEqual(t, out.String(), tc.wantJSON)
			}
			for _, want := range tc.contains {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("output missing %q: %s", want, &out)
				}
			}
		})
	}
}

func TestOrgFollowersPaginationAndLimit(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		limit, want, requests int
	}{
		{"default", 0, 30, 1}, {"one", 1, 1, 1}, {"page boundary", 100, 100, 1},
		{"truncate second page", 101, 101, 2}, {"short final page", 250, 203, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			f := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				requests++
				if req.URL.Path != "/api/v5/orgs/team/followers" || req.URL.Query().Get("page") != fmt.Sprint(requests) || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("request URL = %s", req.URL)
				}
				count := 100
				if requests == 3 {
					count = 3
				}
				followers := make([]api.OrganizationFollower, count)
				for i := range followers {
					followers[i].ID = int64((requests-1)*100 + i + 1)
				}
				data, err := json.Marshal(followers)
				if err != nil {
					t.Fatal(err)
				}
				return orgResponse(http.StatusOK, string(data)), nil
			})
			cmd := newCmdOrgFollowers(f)
			args := []string{"team", "--json"}
			if tc.limit > 0 {
				args = append(args, "-L", fmt.Sprint(tc.limit))
			}
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var followers []api.OrganizationFollower
			if err := json.Unmarshal(out.Bytes(), &followers); err != nil {
				t.Fatal(err)
			}
			if len(followers) != tc.want || requests != tc.requests {
				t.Fatalf("items=%d requests=%d, want %d/%d", len(followers), requests, tc.want, tc.requests)
			}
			for i, follower := range followers {
				if follower.ID != int64(i+1) {
					t.Fatalf("item %d has ID %d", i, follower.ID)
				}
			}
		})
	}
}

func TestOrgFollowersRejectsInvalidArgumentsBeforeAuthentication(t *testing.T) {
	for _, args := range [][]string{nil, {"team", "extra"}, {"  "}, {"owner/team"}, {"team", "-L", "0"}, {"team", "--limit", "-1"}} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			tokenCalls := 0
			f := orgFactory(orgTestConfig{tokenCalls: &tokenCalls}, nil)
			cmd := newCmdOrgFollowers(f)
			cmd.SilenceUsage, cmd.SilenceErrors = true, true
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("accepted invalid arguments")
			}
			if tokenCalls != 0 {
				t.Fatalf("token calls = %d, want 0", tokenCalls)
			}
		})
	}
}

func TestOrgFollowersAuthenticationFailure(t *testing.T) {
	f := orgFactory(orgTestConfig{tokenErr: config.ErrNotAuthenticated}, nil)
	cmd := newCmdOrgFollowers(f)
	err := cmd.RunE(cmd, []string{"team"})
	if !errors.Is(err, config.ErrNotAuthenticated) {
		t.Fatalf("error = %v", err)
	}
}

func TestOrgFollowersResponseFailuresDoNotPrintPartialResults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		page   int
	}{
		{"unauthorized", 401, `{}`, 1}, {"forbidden", 403, `{}`, 1}, {"not found", 404, `{}`, 1},
		{"rate limited", 429, `{}`, 1}, {"unexpected success status", 201, `[]`, 1},
		{"malformed JSON", 200, `[`, 1}, {"invalid ID type", 200, `[{"id":"invalid"}]`, 1},
		{"second page fails", 403, `{}`, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			f := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
				requests++
				if requests < tc.page {
					data, err := json.Marshal(make([]api.OrganizationFollower, 100))
					if err != nil {
						t.Fatal(err)
					}
					return orgResponse(200, string(data)), nil
				}
				return orgResponse(tc.status, tc.body), nil
			})
			var out bytes.Buffer
			err := runOrgFollowers(&out, f, "team", &collectionOptions{Limit: 101, JSON: true})
			if err == nil || !strings.Contains(err.Error(), `failed to list followers for organization "team"`) {
				t.Fatalf("error = %v", err)
			}
			if out.Len() != 0 {
				t.Fatalf("printed partial results: %s", &out)
			}
			if tc.status != 200 {
				var httpErr *api.HTTPError
				if !errors.As(err, &httpErr) || httpErr.StatusCode != tc.status {
					t.Fatalf("lost HTTP error: %v", err)
				}
			} else if !errors.Is(err, api.ErrInvalidResponse) {
				t.Fatalf("lost response error: %v", err)
			}
		})
	}
}
