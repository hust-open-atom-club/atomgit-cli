package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func executeRunnerGroupCommand(t *testing.T, factory *cmdutil.Factory, args ...string) (string, error) {
	t.Helper()
	cmd := NewCmdOrg(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(append([]string{"runner-group"}, args...))
	err := cmd.Execute()
	return output.String(), err
}

func TestRunnerGroupCommandsRegistered(t *testing.T) {
	cmd := newCmdOrgRunnerGroup(&cmdutil.Factory{})
	for _, name := range []string{"list", "view", "runners", "runner-sets", "namespaces"} {
		subcommand, _, err := cmd.Find([]string{name})
		if err != nil || subcommand == cmd {
			t.Fatalf("subcommand %q missing: %v", name, err)
		}
		if name != "view" {
			for _, flag := range []string{"limit", "json"} {
				if subcommand.Flags().Lookup(flag) == nil {
					t.Fatalf("%s flag %q missing", name, flag)
				}
			}
		}
	}
}

func TestRunnerGroupListTextAndJSON(t *testing.T) {
	body := `{"total_count":1,"runner_groups":[{"id":"group-1","name":"legacy","runner_group_name":"Builders","namespace_id":"org-1","creator":"alice","create_time":1700000000000,"runner_count":2,"namespace_type":"organization","share_all":true}]}`
	transport := orgRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/v8/orgs/team/actions/runner-groups" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.URL.Query().Encode() != "page=1&per_page=100" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		return orgResponse(http.StatusOK, body), nil
	})

	output, err := executeRunnerGroupCommand(t, orgFactory(orgTestConfig{}, transport), "list", "team")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ORGANIZATION", "GROUP ID", "team", "group-1", "Builders", "2", "true", "organization", "2023-11-14T22:13:20Z"} {
		if !strings.Contains(output, want) {
			t.Fatalf("text output missing %q:\n%s", want, output)
		}
	}

	output, err = executeRunnerGroupCommand(t, orgFactory(orgTestConfig{}, transport), "list", "team", "--json")
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, output, body)
}

func TestRunnerGroupViewTextJSONAndIncompleteFields(t *testing.T) {
	body := `{"runner_group_id":"group-1","runner_group_name":"Builders","share_all":false,"share_all_public_repos":true,"explicit_shared_repo_count":3,"created_at":1700000000000,"updated_at":1700000100000}`
	transport := orgRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v8/orgs/team/actions/runner-groups/group-1" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return orgResponse(http.StatusOK, body), nil
	})
	output, err := executeRunnerGroupCommand(t, orgFactory(orgTestConfig{}, transport), "view", "team", "group-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Organization: team", "Group ID: group-1", "Name: Builders", "Share all repositories: false", "Share all public repositories: true", "Explicit shared repositories: 3"} {
		if !strings.Contains(output, want) {
			t.Fatalf("view output missing %q:\n%s", want, output)
		}
	}
	output, err = executeRunnerGroupCommand(t, orgFactory(orgTestConfig{}, transport), "view", "team", "group-1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, output, body)

	emptyTransport := orgRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, `{}`), nil
	})
	output, err = executeRunnerGroupCommand(t, orgFactory(orgTestConfig{}, emptyTransport), "view", "team", "group-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Group ID: -", "Name: -", "Created: -", "Updated: -"} {
		if !strings.Contains(output, want) {
			t.Fatalf("incomplete output missing %q:\n%s", want, output)
		}
	}
}

func TestRunnerGroupAssociationCommands(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		pathSuffix string
		body       string
		contains   []string
		jsonField  string
	}{
		{
			name: "host runners", command: "runners", pathSuffix: "/runners",
			body:     `{"total_count":1,"runners":[{"id":"runner-1","runner_group_id":"group-1","runner_name":"host-a","work_dir":"/work","labels":[{"label_name":"os","label_value":"linux","label_color":"blue"}],"status":"IDLE","memory":12.5,"disk":40}]}`,
			contains: []string{"RUNNER ID", "runner-1", "host-a", "IDLE", "/work", "os=linux"}, jsonField: "runners",
		},
		{
			name: "runner sets", command: "runner-sets", pathSuffix: "/runner-sets",
			body:     `{"total_count":1,"runner_sets":[{"id":"set-1","runner_group_id":"group-1","name":"elastic","status":"using","required_labels":[{"label_name":"arch","label_value":"arm64"}],"min_runner_size":1,"max_runner_size":4,"limit_cpu":2,"limit_memory":8,"image_name":"ubuntu","user_k8s_cluster_name":"cluster-a","user_k8s_cluster_namespace":"ci"}]}`,
			contains: []string{"SET ID", "set-1", "elastic", "using", "1-4", "cluster-a", "arch=arm64"}, jsonField: "runner_sets",
		},
		{
			name: "namespaces", command: "namespaces", pathSuffix: "/shared-namespaces",
			body:     `{"total_count":1,"shared_namespaces":[{"id":"share-1","runner_group_id":"group-1","type":"repository","namespace_id":"repo-1","name":"demo","path":"demo","visibility":"private","path_with_namespace":"team/demo"}]}`,
			contains: []string{"SHARE ID", "share-1", "team/demo", "private", "repository"}, jsonField: "shared_namespaces",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := orgRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				wantPath := "/api/v8/orgs/team/actions/runner-groups/group-1" + tt.pathSuffix
				if req.URL.Path != wantPath {
					t.Fatalf("path = %q, want %q", req.URL.Path, wantPath)
				}
				return orgResponse(http.StatusOK, tt.body), nil
			})
			output, err := executeRunnerGroupCommand(t, orgFactory(orgTestConfig{}, transport), tt.command, "team", "group-1")
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range append([]string{"ORGANIZATION", "GROUP ID", "team", "group-1"}, tt.contains...) {
				if !strings.Contains(output, want) {
					t.Fatalf("output missing %q:\n%s", want, output)
				}
			}
			output, err = executeRunnerGroupCommand(t, orgFactory(orgTestConfig{}, transport), tt.command, "team", "group-1", "--json")
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]json.RawMessage
			if err := json.Unmarshal([]byte(output), &decoded); err != nil {
				t.Fatal(err)
			}
			if _, ok := decoded[tt.jsonField]; !ok {
				t.Fatalf("JSON output missing %q: %s", tt.jsonField, output)
			}
		})
	}
}

func TestRunnerGroupEmptyAssociations(t *testing.T) {
	tests := []struct {
		command string
		body    string
		want    string
	}{
		{"list", `{"total_count":0,"runner_groups":[]}`, "No runner groups found"},
		{"runners", `{"total_count":0,"runners":[]}`, "No runners found"},
		{"runner-sets", `{"total_count":0,"runner_sets":[]}`, "No runner sets found"},
		{"namespaces", `{"total_count":0,"shared_namespaces":[]}`, "No shared namespaces found"},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			transport := orgRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return orgResponse(http.StatusOK, tt.body), nil
			})
			args := []string{tt.command, "team"}
			if tt.command != "list" {
				args = append(args, "group-1")
			}
			output, err := executeRunnerGroupCommand(t, orgFactory(orgTestConfig{}, transport), args...)
			if err != nil || !strings.Contains(output, tt.want) {
				t.Fatalf("output = %q, error = %v", output, err)
			}
		})
	}
}

func TestRunnerGroupPaginationAndLimit(t *testing.T) {
	requests := 0
	transport := orgRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		page, _ := strconv.Atoi(req.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(req.URL.Query().Get("per_page"))
		if perPage != maxRunnerGroupsPerPage {
			t.Fatalf("per_page = %d", perPage)
		}
		start := (page - 1) * perPage
		end := min(start+perPage, 150)
		groups := make([]actions.RunnerGroup, 0, end-start)
		for i := start; i < end; i++ {
			groups = append(groups, actions.RunnerGroup{ID: fmt.Sprintf("group-%03d", i)})
		}
		data, _ := json.Marshal(actions.RunnerGroupListResponse{TotalCount: 150, RunnerGroups: groups})
		return orgResponse(http.StatusOK, string(data)), nil
	})
	output, err := executeRunnerGroupCommand(t, orgFactory(orgTestConfig{}, transport), "list", "team", "--limit", "120", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || !strings.Contains(output, `"total_count": 120`) || !strings.Contains(output, `"id": "group-119"`) || strings.Contains(output, `"id": "group-120"`) {
		t.Fatalf("requests = %d, output = %s", requests, output)
	}
}

func TestRunnerGroupPaginationRejectsIncompleteAndRepeatedPages(t *testing.T) {
	_, err := collectRunnerGroupPages(200, func(group actions.RunnerGroup) string { return group.ID }, func(page, _ int) (int, []actions.RunnerGroup, error) {
		return 2, []actions.RunnerGroup{{ID: "one"}}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "incomplete pagination") {
		t.Fatalf("error = %v", err)
	}

	page := make([]actions.RunnerGroup, maxRunnerGroupsPerPage)
	for i := range page {
		page[i].ID = fmt.Sprintf("group-%d", i)
	}
	_, err = collectRunnerGroupPages(200, func(group actions.RunnerGroup) string { return group.ID }, func(_ int, _ int) (int, []actions.RunnerGroup, error) {
		return 200, page, nil
	})
	if err == nil || !strings.Contains(err.Error(), "pagination made no progress") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunnerGroupValidationPrecedesAuthentication(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"organization", []string{"list", "bad/org"}, "invalid organization"},
		{"blank organization", []string{"list", " "}, "invalid organization"},
		{"organization control character", []string{"list", "team\nother"}, "invalid organization"},
		{"group ID", []string{"view", "team", "bad/group"}, "invalid runner group ID"},
		{"blank group ID", []string{"view", "team", " "}, "invalid runner group ID"},
		{"group ID control character", []string{"view", "team", "group\tid"}, "invalid runner group ID"},
		{"limit zero", []string{"list", "team", "--limit", "0"}, "must be positive"},
		{"limit negative", []string{"runners", "team", "group-1", "--limit", "-1"}, "must be positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			factory := orgFactory(orgTestConfig{tokenErr: config.ErrNotAuthenticated, tokenCalls: &calls}, nil)
			_, err := executeRunnerGroupCommand(t, factory, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if calls != 0 {
				t.Fatalf("GetToken calls = %d, want 0", calls)
			}
		})
	}
}

func TestRunnerGroupContextualAPIErrors(t *testing.T) {
	tests := []struct {
		name    string
		command string
		status  int
		want    string
	}{
		{"list forbidden", "list", http.StatusForbidden, "failed to list runner groups"},
		{"view missing", "view", http.StatusNotFound, "failed to view runner group"},
		{"runners forbidden", "runners", http.StatusForbidden, "failed to list runners"},
		{"sets missing", "runner-sets", http.StatusNotFound, "failed to list runner sets"},
		{"namespaces forbidden", "namespaces", http.StatusForbidden, "failed to list shared namespaces"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := orgRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return orgResponse(tt.status, `{"message":"denied"}`), nil
			})
			args := []string{tt.command, "team"}
			if tt.command != "list" {
				args = append(args, "group-1")
			}
			_, err := executeRunnerGroupCommand(t, orgFactory(orgTestConfig{}, transport), args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), strconv.Itoa(tt.status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRunnerGroupCommandsHonorFactoryContextCancellation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		body string
	}{
		{"list", []string{"list", "team"}, `{"total_count":0,"runner_groups":[]}`},
		{"view", []string{"view", "team", "group-1"}, `{}`},
		{"runners", []string{"runners", "team", "group-1"}, `{"total_count":0,"runners":[]}`},
		{"runner sets", []string{"runner-sets", "team", "group-1"}, `{"total_count":0,"runner_sets":[]}`},
		{"namespaces", []string{"namespaces", "team", "group-1"}, `{"total_count":0,"shared_namespaces":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			factory := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				if err := req.Context().Err(); err != nil {
					return nil, err
				}
				return orgResponse(http.StatusOK, tt.body), nil
			})
			factory.Context = func() context.Context { return ctx }

			_, err := executeRunnerGroupCommand(t, factory, tt.args...)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context canceled", err)
			}
		})
	}
}

func TestRunnerGroupDefaultClientHonorsFactoryContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	factory := &cmdutil.Factory{
		Config:  orgTestConfig{},
		Context: func() context.Context { return ctx },
	}
	client, err := organizationActionsClient(factory)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListOrganizationRunnerGroups("team", actions.ListRunnerGroupsOptions{Page: 1, PerPage: 1})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
}

func TestRunnerGroupOutputUsesSanitizingWriter(t *testing.T) {
	transport := orgRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, `{"total_count":1,"runner_groups":[{"id":"group-1","runner_group_name":"unsafe\u001b[31mname"}]}`), nil
	})
	cmd := NewCmdOrg(orgFactory(orgTestConfig{}, transport))
	var output bytes.Buffer
	safe := cmdutil.NewSanitizingWriter(&output)
	cmd.SetOut(safe)
	cmd.SetArgs([]string{"runner-group", "list", "team"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := safe.Flush(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "\x1b") || !strings.Contains(output.String(), `unsafe\x1b[31mname`) {
		t.Fatalf("output was not safely escaped: %q", output.String())
	}
}
