package repo

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestPolicyEditRejectsConflictingResponse(t *testing.T) {
	for _, tc := range []struct{ flag, field, body string }{
		{"--merge-method=ff", "merge_method", policySettingsFixture},
		{"--only-allow-merge-if-pipeline-succeeds=false", "only_allow_merge_if_pipeline_succeeds", policySettingsFixture},
		{"--can-force-merge=true", "can_force_merge", policySettingsFixture},
		{"--forbidden-pr-related-issue-closed=true", "forbidden_pr_related_issue_closed", policySettingsFixture},
		{"--approval-required-approvers=0", "approval_required_approvers", policySettingsFixture},
		{"--lite-merge-request-prefix-title=", "lite_merge_request_prefix_title", strings.Replace(policySettingsFixture, `"can_force_merge":0`, `"can_force_merge":0,"lite_merge_request_prefix_title":"prefix"`, 1)},
	} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%t", tc.field, asJSON), func(t *testing.T) {
				calls := 0
				f := policyFactory(t, func(r *http.Request) (int, string) {
					calls++
					if r.Method != http.MethodPut {
						t.Fatalf("method %s", r.Method)
					}
					return 200, tc.body
				})
				args := []string{"edit", "team/demo", "--section", "pull-request", tc.flag, "--yes"}
				if asJSON {
					args = append(args, "--json")
				}
				out, _, err := executePolicy(t, f, "", args...)
				if err == nil || !strings.Contains(err.Error(), tc.field) || !strings.Contains(err.Error(), "partially applied") {
					t.Fatalf("error %v", err)
				}
				if out != "" || calls != 1 {
					t.Fatalf("stdout %q, calls %d", out, calls)
				}
			})
		}
	}
}

func TestPolicyRejectsMalformedReviewerEntries(t *testing.T) {
	for _, field := range []string{"approval_approvers", "approval_testers"} {
		for _, entry := range []string{"false", "42", "null", `"alice"`, "[]"} {
			for _, operation := range []string{"view-all", "view-code-review", "view-pull-request", "edit"} {
				t.Run(field+"/"+entry+"/"+operation, func(t *testing.T) {
					body := `{"merge_request_setting":{"approval_approvers":[],"approval_testers":[],"approval_required_approvers":0,"approval_required_testers":0},"merge_method":"merge","only_allow_merge_if_pipeline_succeeds":true,"only_allow_merge_if_all_discussions_are_resolved":false}`
					body = strings.Replace(body, `"`+field+`":[]`, `"`+field+`":[{},`+entry+`]`, 1)
					f := policyFactory(t, func(r *http.Request) (int, string) {
						if strings.HasSuffix(r.URL.Path, "/transition") {
							return 200, `{"memberMgntMode":1}`
						}
						return 200, body
					})
					args := []string{"view", "team/demo", "--json"}
					switch operation {
					case "view-code-review":
						args = append(args, "--section", "code-review")
					case "view-pull-request":
						args = append(args, "--section", "pull-request")
					case "edit":
						args = []string{"edit", "team/demo", "--section", "pull-request", "--merge-method=merge", "--yes", "--json"}
					}
					out, _, err := executePolicy(t, f, "", args...)
					if err == nil || !strings.Contains(err.Error(), field+"[1]") || out != "" {
						t.Fatalf("stdout %q, error %v", out, err)
					}
				})
			}
		}
	}
}
