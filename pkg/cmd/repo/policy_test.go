package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

const policySettingsFixture = `{"merge_request_setting":{"approval_approvers":[{"username":"alice"}],"approval_testers":[],"approval_required_approvers":1,"approval_required_testers":0,"can_force_merge":0,"forbidden_pr_related_issue_closed":"0"},"only_allow_merge_if_all_discussions_are_resolved":0,"only_allow_merge_if_pipeline_succeeds":1,"merge_method":"merge"}`

func policyFactory(t *testing.T, handle func(*http.Request) (int, string)) *cmdutil.Factory {
	t.Helper()
	return repoFactory(repoCommandConfig{token: "synthetic"}, forkRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		status, body := handle(r)
		return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
}
func executePolicy(t *testing.T, f *cmdutil.Factory, input string, args ...string) (string, string, error) {
	t.Helper()
	cmd := newCmdRepoPolicy(f)
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	return out.String(), stderr.String(), err
}
func TestPolicyViewSections(t *testing.T) {
	for _, section := range []string{"", "permission", "code-review", "pull-request"} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%t", section, asJSON), func(t *testing.T) {
				calls := 0
				f := policyFactory(t, func(r *http.Request) (int, string) {
					calls++
					if r.Method != "GET" {
						t.Fatalf("method %s", r.Method)
					}
					switch r.URL.Path {
					case "/api/v5/repos/team/demo/transition":
						return 200, `{"memberMgntMode":2}`
					case "/api/v5/repos/team/demo/pull_request_settings":
						return 200, policySettingsFixture
					default:
						t.Fatalf("unexpected path %s", r.URL.Path)
						return 500, ""
					}
				})
				args := []string{"view", "team/demo"}
				if section != "" {
					args = append(args, "--section", section)
				}
				if asJSON {
					args = append(args, "--json")
				}
				out, _, err := executePolicy(t, f, "", args...)
				if err != nil {
					t.Fatal(err)
				}
				wantCalls := 1
				if section == "" {
					wantCalls = 2
				}
				if calls != wantCalls {
					t.Fatalf("calls %d", calls)
				}
				if asJSON {
					var result policyViewJSON
					if err := json.Unmarshal([]byte(out), &result); err != nil {
						t.Fatal(err)
					}
					wantLen := 1
					if section == "" {
						wantLen = 3
					}
					if result.Repository != "team/demo" || len(result.Settings) != wantLen {
						t.Fatalf("output %s", out)
					}
					if section == "code-review" && !strings.Contains(out, `"approval_required_testers": 0`) {
						t.Fatalf("zero lost: %s", out)
					}
				} else if !strings.Contains(out, "Repository: team/demo") {
					t.Fatalf("output %s", out)
				}
			})
		}
	}
}
func TestPolicyEditPartialUpdates(t *testing.T) {
	cases := []struct {
		section, path, response string
		flags                   []string
		want                    map[string]any
	}{
		{"permission", "transition", `{"code":1,"msg":"success"}`, []string{"--mode", "2"}, map[string]any{"mode": float64(2)}},
		{"code-review", "reviewer", `{"id":1}`, []string{"--assignees=", "--testers-number", "0"}, map[string]any{"assignees": "", "testers_number": float64(0)}},
		{"pull-request", "pull_request_settings", policySettingsFixture, []string{"--can-force-merge=false", "--approval-required-reviewers", "0", "--lite-merge-request-prefix-title="}, map[string]any{"can_force_merge": false, "approval_required_reviewers": float64(0), "lite_merge_request_prefix_title": ""}},
	}
	for _, tc := range cases {
		t.Run(tc.section, func(t *testing.T) {
			for _, confirm := range []string{"flag", "yes", "no", "eof"} {
				t.Run(confirm, func(t *testing.T) {
					calls := 0
					f := policyFactory(t, func(r *http.Request) (int, string) {
						calls++
						if r.Method != "PUT" || r.URL.Path != "/api/v5/repos/team/demo/"+tc.path {
							t.Fatalf("request %s %s", r.Method, r.URL.Path)
						}
						var body map[string]any
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(body, tc.want) {
							t.Fatalf("body %#v, want %#v", body, tc.want)
						}
						return 200, tc.response
					})
					args := append([]string{"edit", "team/demo", "--section", tc.section, "--json"}, tc.flags...)
					input := ""
					if confirm == "flag" {
						args = append(args, "--yes")
					}
					if confirm == "yes" {
						input = "yes\n"
					}
					if confirm == "no" {
						input = "n\n"
					}
					out, stderr, err := executePolicy(t, f, input, args...)
					if confirm == "no" || confirm == "eof" {
						if calls != 0 || out != "" {
							t.Fatalf("cancel wrote: %d %s", calls, out)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if calls != 1 {
						t.Fatalf("calls %d", calls)
					}
					if confirm == "flag" && stderr != "" {
						t.Fatalf("unexpected prompt %s", stderr)
					}
					var result struct {
						Repository, Section string
						Changed             map[string]any `json:"changed_fields"`
					}
					if err := json.Unmarshal([]byte(out), &result); err != nil {
						t.Fatal(err)
					}
					if result.Repository != "team/demo" || result.Section != tc.section || !reflect.DeepEqual(result.Changed, tc.want) {
						t.Fatalf("output %s", out)
					}
				})
			}
		})
	}
}
func TestPolicyInvalidInputDoesNotRequest(t *testing.T) {
	for _, args := range [][]string{
		{"edit", "team/demo", "--mode", "1"}, {"edit", "team/demo", "--section", "permission"},
		{"edit", "team/demo", "--section", "other", "--mode", "1"},
		{"edit", "team/demo", "--section", "permission", "--mode", "0"},
		{"edit", "team/demo", "--section", "permission", "--mode", "3"},
		{"edit", "team/demo", "--section", "permission", "--mode", "1", "--assignees", "alice"},
		{"edit", "team/demo", "--section", "code-review", "--assignees-number", "-1"},
		{"edit", "team/demo", "--section", "code-review", "--forbidden-pr-related-issue-closed=false"},
		{"edit", "team/demo", "--section", "code-review", "--assignees", "alice,,bob"},
		{"edit", "team/demo", "--section", "pull-request", "--approval-required-reviewers", "6"},
		{"edit", "team/demo", "--section", "pull-request", "--merge-method="},
		{"edit", "team/demo", "--section", "pull-request", "--merged-commit-author", "owner"},
		{"view", "team/demo", "--section", "invalid"}, {"view", "team/demo", "--section="},
		{"view", "team/demo", "extra"}, {"edit", "team/demo", "extra", "--section", "permission", "--mode", "1"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			f := policyFactory(t, func(r *http.Request) (int, string) { t.Fatal("unexpected request"); return 500, "" })
			if _, _, err := executePolicy(t, f, "", args...); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
func TestPolicyErrors(t *testing.T) {
	for _, section := range []string{"permission", "code-review", "pull-request"} {
		for _, action := range []string{"view", "edit"} {
			for _, tc := range []struct {
				name   string
				status int
				body   string
			}{
				{"unauthorized", 401, `{"message":"denied"}`}, {"forbidden", 403, `{"message":"denied"}`}, {"missing", 404, `{}`},
				{"conflict", 409, `{}`}, {"wrong success", 201, `{}`}, {"empty success", 204, ""},
				{"empty", 200, ""}, {"null", 200, "null"}, {"array", 200, "[]"}, {"missing fields", 200, `{}`}, {"malformed", 200, `{"`},
			} {
				t.Run(section+"/"+action+"/"+tc.name, func(t *testing.T) {
					f := policyFactory(t, func(r *http.Request) (int, string) { return tc.status, tc.body })
					args := []string{action, "team/demo", "--section", section, "--json"}
					if action == "edit" {
						args = append(args, "--yes")
						switch section {
						case "permission":
							args = append(args, "--mode", "1")
						case "code-review":
							args = append(args, "--assignees-number", "0")
						default:
							args = append(args, "--can-force-merge=false")
						}
					}
					out, _, err := executePolicy(t, f, "", args...)
					if err == nil || out != "" {
						t.Fatalf("out %q err %v", out, err)
					}
				})
			}
		}
	}
}
func TestPolicyAuthenticationError(t *testing.T) {
	f := repoFactory(repoCommandConfig{tokenErr: errors.New("no credentials")}, nil)
	if _, _, err := executePolicy(t, f, "", "view", "team/demo"); err == nil {
		t.Fatal("expected auth error")
	}
}

func TestPolicyRepositoryContext(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprint(explicit), func(t *testing.T) {
			resolverCalls := 0
			owner := "inferred"
			if explicit {
				owner = "explicit"
			}
			f := policyFactory(t, func(r *http.Request) (int, string) {
				if r.URL.Path != "/api/v5/repos/"+owner+"/demo/transition" {
					t.Fatalf("path %s", r.URL.Path)
				}
				return 200, `{"memberMgntMode":1}`
			})
			f.RepositoryResolver = func() (cmdutil.Repository, error) {
				resolverCalls++
				return cmdutil.Repository{Owner: "inferred", Name: "demo"}, nil
			}
			args := []string{"view", "--section", "permission"}
			if explicit {
				args = append(args, "explicit/demo")
			}
			if _, _, err := executePolicy(t, f, "", args...); err != nil {
				t.Fatal(err)
			}
			if explicit && resolverCalls != 0 {
				t.Fatal("explicit repository did not take precedence")
			}
			if !explicit && resolverCalls != 1 {
				t.Fatalf("resolver calls %d", resolverCalls)
			}
		})
	}
}

func TestPolicyViewAllDoesNotPrintPartialSuccess(t *testing.T) {
	f := policyFactory(t, func(r *http.Request) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/transition") {
			return 200, `{"memberMgntMode":1}`
		}
		return 403, `{"message":"denied"}`
	})
	out, _, err := executePolicy(t, f, "", "view", "team/demo")
	if err == nil || out != "" {
		t.Fatalf("out %q error %v", out, err)
	}
}

func TestPolicyRegistrationAndStableText(t *testing.T) {
	root := NewCmdRepo(&cmdutil.Factory{})
	cmd, _, err := root.Find([]string{"policy", "edit"})
	if err != nil || cmd.Name() != "edit" || cmd.Flags().Lookup("section") == nil {
		t.Fatalf("missing policy registration: %v", err)
	}
	f := policyFactory(t, func(r *http.Request) (int, string) { return 200, `{"memberMgntMode":2}` })
	out, _, err := executePolicy(t, f, "", "view", "team/demo", "--section", "permission")
	if err != nil {
		t.Fatal(err)
	}
	if out != "Repository: team/demo\npermission:\n  memberMgntMode: 2\n" {
		t.Fatalf("text %q", out)
	}
}
