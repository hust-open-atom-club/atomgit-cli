package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func executeSettings(t *testing.T, f *cmdutil.Factory, input string, args ...string) (string, string, error) {
	t.Helper()
	cmd := newCmdRepoSettings(f)
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(args)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err := cmd.ExecuteContext(t.Context())
	return out.String(), stderr.String(), err
}

func TestRepoSettingsViewContextAndMissingValues(t *testing.T) {
	for _, explicit := range []bool{true, false} {
		for _, asJSON := range []bool{true, false} {
			f := policyFactory(t, func(r *http.Request) (int, string) {
				if r.Method != "GET" || r.URL.Path != "/api/v5/repos/team/demo/repo_settings" || r.Header.Get("Authorization") != "Bearer synthetic" {
					t.Fatalf("request=%s %s", r.Method, r.URL)
				}
				return 200, `{"action_pipeline_enabled":false,"branch_name_regex":"line1\nline2\t\u001b[31m","forbidden_developer_create_branch_users":[]}`
			})
			f.RepositoryResolver = func() (cmdutil.Repository, error) {
				if explicit {
					t.Fatal("explicit repo used Git inference")
				}
				return cmdutil.Repository{Owner: "team", Name: "demo"}, nil
			}
			args := []string{"view"}
			if explicit {
				args = append(args, "team/demo")
			}
			if asJSON {
				args = append(args, "--json")
			}
			out, stderr, err := executeSettings(t, f, "", args...)
			if err != nil || stderr != "" {
				t.Fatalf("error=%v stderr=%q", err, stderr)
			}
			if strings.ContainsAny(out, "\t\x1b") || strings.Contains(out, "line1\nline2") {
				t.Fatalf("unsafe text=%q", out)
			}
			if asJSON {
				var result map[string]json.RawMessage
				if err := json.Unmarshal([]byte(out), &result); err != nil {
					t.Fatal(err)
				}
				if len(result) != 2 || string(result["repository"]) != `"team/demo"` {
					t.Fatalf("output=%s", out)
				}
				var settings map[string]json.RawMessage
				if err := json.Unmarshal(result["settings"], &settings); err != nil {
					t.Fatal(err)
				}
				if len(settings) != 15 || string(settings["action_pipeline_enabled"]) != "false" || string(settings["disable_fork"]) != "null" || string(settings["forbidden_developer_create_branch_users"]) != "[]" {
					t.Fatalf("settings=%s", result["settings"])
				}
			} else if !strings.Contains(out, "Repository: team/demo") || !strings.Contains(out, `"disable_fork": null`) {
				t.Fatalf("output=%s", out)
			}
		}
	}
}

func TestRepoSettingsEditFlagsPreserveExplicitValues(t *testing.T) {
	for _, tc := range []struct {
		flag, key, value string
		boolean          bool
	}{
		{"disable-fork", "disable_fork", "false", true},
		{"forbidden-developer-create-branch", "forbidden_developer_create_branch", "false", true},
		{"forbidden-developer-create-tag", "forbidden_developer_create_tag", "false", true},
		{"forbidden-committer-create-branch", "forbidden_committer_create_branch", "false", true},
		{"generate-pre-merge-ref", "generate_pre_merge_ref", "false", true},
		{"rebase-disable-trigger-webhook", "rebase_disable_trigger_webhook", "false", true},
		{"open-gpg-verified", "open_gpg_verified", "false", true},
		{"include-lfs-objects", "include_lfs_objects", "false", true},
		{"action-pipeline-enabled", "action_pipeline_enabled", "false", true},
		{"block-cross-repo-pr-trigger", "block_cross_repo_pr_trigger", "false", true},
		{"show-branch-creator", "show_branch_creator", "false", true},
		{"forbidden-developer-create-branch-user-ids", "forbidden_developer_create_branch_user_ids", "", false},
		{"branch-name-regex", "branch_name_regex", "", false},
		{"tag-name-regex", "tag_name_regex", "", false},
	} {
		t.Run(tc.flag, func(t *testing.T) {
			calls := 0
			f := policyFactory(t, func(r *http.Request) (int, string) {
				calls++
				if r.Method != "PUT" || r.URL.Path != "/api/v5/repos/team/demo/repo_settings" {
					t.Fatalf("request=%s %s", r.Method, r.URL)
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				want := `""`
				if tc.boolean {
					want = "false"
				}
				if len(body) != 1 || string(body[tc.key]) != want {
					t.Fatalf("body=%s want %s=%s", body, tc.key, want)
				}
				return 200, `{"show_branch_creator":true}`
			})
			out, stderr, err := executeSettings(t, f, "", "edit", "team/demo", "--"+tc.flag+"="+tc.value, "--yes", "--json")
			if err != nil || stderr != "" || calls != 1 {
				t.Fatalf("error=%v stderr=%q calls=%d", err, stderr, calls)
			}
			var result map[string]json.RawMessage
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatal(err)
			}
			if len(result) != 4 || string(result["verification"]) != `"response_only"` {
				t.Fatalf("output=%s", out)
			}
			var submitted map[string]json.RawMessage
			if err := json.Unmarshal(result["submitted_fields"], &submitted); err != nil {
				t.Fatal(err)
			}
			if len(submitted) != 1 {
				t.Fatalf("output=%s", out)
			}
		})
	}
}

func TestRepoSettingsEditConfirmationAndMultipleFields(t *testing.T) {
	for _, input := range []string{"yes\n", "n\n", ""} {
		calls := 0
		f := policyFactory(t, func(r *http.Request) (int, string) {
			calls++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"action_pipeline_enabled": false, "block_cross_repo_pr_trigger": true, "branch_name_regex": "(?=server-dialect)"}
			if !reflect.DeepEqual(body, want) {
				t.Fatalf("body=%v", body)
			}
			return 200, `{"action_pipeline_enabled":false}`
		})
		f.RepositoryResolver = func() (cmdutil.Repository, error) { return cmdutil.Repository{Owner: "team", Name: "demo"}, nil }
		if input != "yes\n" {
			f.Config = settingsForbiddenConfig{t}
		}
		out, stderr, err := executeSettings(t, f, input, "edit", "--action-pipeline-enabled=false", "--block-cross-repo-pr-trigger", "--branch-name-regex", "(?=server-dialect)")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stderr, "Repository: team/demo") || !strings.Contains(stderr, `"action_pipeline_enabled": false`) {
			t.Fatalf("confirmation=%q", stderr)
		}
		if input == "yes\n" {
			if calls != 1 || !strings.Contains(out, "acknowledged by API") || !strings.Contains(out, "not independently verified") {
				t.Fatalf("calls=%d output=%q", calls, out)
			}
		} else if calls != 0 || out != "" || !strings.Contains(stderr, "cancelled") {
			t.Fatalf("calls=%d output=%q stderr=%q", calls, out, stderr)
		}
	}
}

type settingsForbiddenConfig struct{ t *testing.T }

func (c settingsForbiddenConfig) GetToken() (string, error) {
	c.t.Fatal("read credentials before validation or on cancellation")
	return "", nil
}
func (c settingsForbiddenConfig) GetUser() (string, error) { c.t.Fatal("read user"); return "", nil }
func (c settingsForbiddenConfig) GetHost() string          { return "atomgit.com" }

func TestRepoSettingsInvalidParametersBeforeAuthentication(t *testing.T) {
	for _, args := range [][]string{
		{"unknown"},
		{"edit", "team/demo"}, {"edit", "team/demo", "--yes"}, {"edit", "a/b/c", "--disable-fork"}, {"view", ""}, {"view", "one", "two"},
		{"edit", "team/demo", "--action-pipeline-enabled=not-bool"}, {"edit", "team/demo", "--forbidden-gitlab-access"},
		{"edit", "team/demo", "--merge-method", "ff"}, {"edit", "team/demo", "--name", "New"},
	} {
		f := policyFactory(t, func(*http.Request) (int, string) { t.Fatal("invalid parameters sent request"); return 500, "" })
		f.Config = settingsForbiddenConfig{t}
		out, stderr, err := executeSettings(t, f, "yes\n", args...)
		if err == nil || out != "" || stderr != "" {
			t.Fatalf("args=%q error=%v output=%q stderr=%q", args, err, out, stderr)
		}
	}
}

func TestRepoSettingsAuthenticationAndHTTPFailures(t *testing.T) {
	for _, action := range []string{"view", "edit"} {
		args := []string{action, "team/demo", "--json"}
		if action == "edit" {
			args = append(args, "--yes", "--disable-fork")
		}
		for _, cause := range []error{config.ErrNotAuthenticated, errors.New("credential read failed")} {
			f := repoFactory(repoCommandConfig{tokenErr: cause}, forkRoundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("credential error sent request"); return nil, nil }))
			out, _, err := executeSettings(t, f, "", args...)
			if !errors.Is(err, cause) || out != "" {
				t.Fatalf("error=%v output=%q", err, out)
			}
		}
		for _, status := range []int{403, 404} {
			calls := 0
			f := policyFactory(t, func(*http.Request) (int, string) { calls++; return status, `{"message":"denied"}` })
			out, _, err := executeSettings(t, f, "", args...)
			if err == nil || calls != 1 || out != "" {
				t.Fatalf("error=%v calls=%d output=%q", err, calls, out)
			}
		}
	}
}
