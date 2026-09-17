package repo

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func TestPolicyEveryPullRequestFlagIsPartial(t *testing.T) {
	for _, tc := range []struct{ flag, value, body string }{
		{"approval-required-reviewers-enable", "false", `{"approval_required_reviewers_enable":false}`},
		{"approval-required-reviewers", "0", `{"approval_required_reviewers":0}`},
		{"only-allow-merge-if-all-discussions-are-resolved", "false", `{"only_allow_merge_if_all_discussions_are_resolved":false}`},
		{"only-allow-merge-if-pipeline-succeeds", "false", `{"only_allow_merge_if_pipeline_succeeds":false}`},
		{"disable-merge-by-self", "false", `{"disable_merge_by_self":false}`},
		{"can-force-merge", "false", `{"can_force_merge":false}`},
		{"add-notes-after-merged", "false", `{"add_notes_after_merged":false}`},
		{"mark-auto-merged-mr-as-closed", "false", `{"mark_auto_merged_mr_as_closed":false}`},
		{"can-reopen", "false", `{"can_reopen":false}`},
		{"delete-source-branch-when-merged", "false", `{"delete_source_branch_when_merged":false}`},
		{"disable-squash-merge", "false", `{"disable_squash_merge":false}`},
		{"auto-squash-merge", "false", `{"auto_squash_merge":false}`},
		{"merge-method", "ff", `{"merge_method":"ff"}`},
		{"squash-merge-with-no-merge-commit", "false", `{"squash_merge_with_no_merge_commit":false}`},
		{"merged-commit-author", "created_by", `{"merged_commit_author":"created_by"}`},
		{"approval-required-approvers", "0", `{"approval_required_approvers":0}`},
		{"approval-approver-ids", "", `{"approval_approver_ids":""}`},
		{"approval-tester-ids", "", `{"approval_tester_ids":""}`},
		{"approval-required-testers", "0", `{"approval_required_testers":0}`},
		{"is-check-cla", "false", `{"is_check_cla":false}`},
		{"is-allow-lite-merge-request", "false", `{"is_allow_lite_merge_request":false}`},
		{"lite-merge-request-prefix-title", "", `{"lite_merge_request_prefix_title":""}`},
		{"close-issue-when-mr-merged", "false", `{"close_issue_when_mr_merged":false}`},
	} {
		t.Run(tc.flag, func(t *testing.T) {
			f := policyFactory(t, func(r *http.Request) (int, string) {
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				var got, want map[string]any
				if err := json.Unmarshal(raw, &got); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("body %s, want %s", raw, tc.body)
				}
				var response map[string]any
				if err := json.Unmarshal([]byte(policySettingsFixture), &response); err != nil {
					t.Fatal(err)
				}
				settings := response["merge_request_setting"].(map[string]any)
				for field, value := range want {
					switch field {
					case "merge_method", "only_allow_merge_if_pipeline_succeeds", "only_allow_merge_if_all_discussions_are_resolved":
						response[field] = value
					case "approval_approver_ids":
						settings["approval_approvers"] = []any{}
					case "approval_tester_ids":
						settings["approval_testers"] = []any{}
					default:
						settings[field] = value
					}
				}
				body, err := json.Marshal(response)
				if err != nil {
					t.Fatal(err)
				}
				return 200, string(body)
			})
			if _, _, err := executePolicy(t, f, "", "edit", "team/demo", "--section", "pull-request", "--"+tc.flag+"="+tc.value, "--yes"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
