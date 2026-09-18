package org

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
)

func TestRunnerGroupNamespacesChecksGroupBeforeListing(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"missing", http.StatusNotFound, `{"message":"Runner Group not found"}`},
		{"forbidden", http.StatusForbidden, `{"message":"denied"}`},
		{"missing ID", http.StatusOK, `{}`},
		{"null detail", http.StatusOK, `null`},
		{"different ID", http.StatusOK, `{"runner_group_id":"another-group"}`},
	} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%t", tc.name, asJSON), func(t *testing.T) {
				var paths []string
				factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
					paths = append(paths, req.URL.Path)
					if strings.HasSuffix(req.URL.Path, "/shared-namespaces") {
						// The real service returns success even when the group is missing.
						return orgResponse(http.StatusOK, `{"total_count":0,"shared_namespaces":[]}`), nil
					}
					return orgResponse(tc.status, tc.body), nil
				})
				cmd := newCmdRunnerGroupNamespaces(factory)
				var stdout, stderr bytes.Buffer
				cmd.SetOut(&stdout)
				cmd.SetErr(&stderr)
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
				args := []string{"team", "group-1"}
				if asJSON {
					args = append(args, "--json")
				}
				cmd.SetArgs(args)
				err := cmd.Execute()
				if err == nil || !strings.Contains(err.Error(), "group-1") || !strings.Contains(err.Error(), "team") {
					t.Fatalf("error = %v, want contextual group verification error", err)
				}
				if tc.status != http.StatusOK {
					var httpErr *actions.HTTPError
					if !errors.As(err, &httpErr) || httpErr.StatusCode != tc.status {
						t.Fatalf("error = %v, want wrapped HTTP %d", err, tc.status)
					}
				}
				if stdout.Len() != 0 {
					t.Fatalf("unexpected success output: %q", stdout.String())
				}
				if len(paths) != 1 || paths[0] != "/api/v8/orgs/team/actions/runner-groups/group-1" {
					t.Fatalf("requests = %v, want only group detail", paths)
				}
			})
		}
	}
}

func TestRunnerGroupNamespacesExistingGroupWithNoShares(t *testing.T) {
	for _, asJSON := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%t", asJSON), func(t *testing.T) {
			calls := 0
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				calls++
				wantPath := "/api/v8/orgs/team/actions/runner-groups/group-1"
				if calls == 2 {
					wantPath += "/shared-namespaces"
				}
				if req.Method != http.MethodGet || req.URL.Path != wantPath || calls > 2 {
					t.Fatalf("request %d = %s %s", calls, req.Method, req.URL.Path)
				}
				if calls == 1 {
					if req.URL.RawQuery != "" {
						t.Fatalf("detail query = %q", req.URL.RawQuery)
					}
					return orgResponse(http.StatusOK, `{"runner_group_id":"group-1"}`), nil
				}
				if req.URL.Query().Encode() != "page=1&per_page=100" {
					t.Fatalf("namespaces query = %q", req.URL.RawQuery)
				}
				return orgResponse(http.StatusOK, `{"total_count":0,"shared_namespaces":[]}`), nil
			})
			args := []string{"namespaces", "team", "group-1"}
			if asJSON {
				args = append(args, "--json")
			}
			out, err := executeRunnerGroupCommand(t, factory, args...)
			if err != nil || calls != 2 {
				t.Fatalf("calls = %d, error = %v", calls, err)
			}
			if asJSON {
				assertJSONEqual(t, out, `{"total_count":0,"shared_namespaces":[]}`)
			} else if out != "No shared namespaces found for runner group \"group-1\" in organization \"team\".\n" {
				t.Fatalf("output = %q", out)
			}
		})
	}
}
