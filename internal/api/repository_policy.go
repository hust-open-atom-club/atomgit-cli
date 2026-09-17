package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// PermissionPolicy uses the read API's camelCase key; edits use "mode".
type PermissionPolicy struct {
	MemberManagementMode *int `json:"memberMgntMode"`
}
type UpdatePermissionPolicyRequest struct {
	Mode *int `json:"mode,omitempty"`
}
type UpdateCodeReviewPolicyRequest struct {
	Assignees       *string `json:"assignees,omitempty"`
	Testers         *string `json:"testers,omitempty"`
	AssigneesNumber *int    `json:"assignees_number,omitempty"`
	TestersNumber   *int    `json:"testers_number,omitempty"`
}
type CodeReviewPolicyUpdateResponse struct {
	ID        *int64 `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
type PermissionPolicyUpdateResponse struct {
	Code *int   `json:"code"`
	Msg  string `json:"msg"`
}

// CodeReviewPolicy is a read-only projection of pull_request_settings.
// There is no documented GET /reviewer. Reviewer identities are retained as
// JSON objects because the documented example specifies only empty arrays.
type CodeReviewPolicy struct {
	ApprovalApprovers         []json.RawMessage `json:"approval_approvers"`
	ApprovalTesters           []json.RawMessage `json:"approval_testers"`
	ApprovalRequiredApprovers *int              `json:"approval_required_approvers"`
	ApprovalRequiredTesters   *int              `json:"approval_required_testers"`
}
type UpdatePullRequestPolicyRequest struct {
	ApprovalRequiredReviewersEnable           *bool   `json:"approval_required_reviewers_enable,omitempty"`
	ApprovalRequiredReviewers                 *int    `json:"approval_required_reviewers,omitempty"`
	OnlyAllowMergeIfAllDiscussionsAreResolved *bool   `json:"only_allow_merge_if_all_discussions_are_resolved,omitempty"`
	OnlyAllowMergeIfPipelineSucceeds          *bool   `json:"only_allow_merge_if_pipeline_succeeds,omitempty"`
	DisableMergeBySelf                        *bool   `json:"disable_merge_by_self,omitempty"`
	CanForceMerge                             *bool   `json:"can_force_merge,omitempty"`
	AddNotesAfterMerged                       *bool   `json:"add_notes_after_merged,omitempty"`
	MarkAutoMergedMRAsClosed                  *bool   `json:"mark_auto_merged_mr_as_closed,omitempty"`
	CanReopen                                 *bool   `json:"can_reopen,omitempty"`
	DeleteSourceBranchWhenMerged              *bool   `json:"delete_source_branch_when_merged,omitempty"`
	DisableSquashMerge                        *bool   `json:"disable_squash_merge,omitempty"`
	AutoSquashMerge                           *bool   `json:"auto_squash_merge,omitempty"`
	MergeMethod                               *string `json:"merge_method,omitempty"`
	SquashMergeWithNoMergeCommit              *bool   `json:"squash_merge_with_no_merge_commit,omitempty"`
	MergedCommitAuthor                        *string `json:"merged_commit_author,omitempty"`
	ApprovalRequiredApprovers                 *int    `json:"approval_required_approvers,omitempty"`
	ApprovalApproverIDs                       *string `json:"approval_approver_ids,omitempty"`
	ApprovalTesterIDs                         *string `json:"approval_tester_ids,omitempty"`
	ApprovalRequiredTesters                   *int    `json:"approval_required_testers,omitempty"`
	IsCheckCLA                                *bool   `json:"is_check_cla,omitempty"`
	IsAllowLiteMergeRequest                   *bool   `json:"is_allow_lite_merge_request,omitempty"`
	LiteMergeRequestPrefixTitle               *string `json:"lite_merge_request_prefix_title,omitempty"`
	CloseIssueWhenMRMerged                    *bool   `json:"close_issue_when_mr_merged,omitempty"`
}
type MergeRequestPolicySettings struct {
	ApprovalRequiredReviewersEnable *bool             `json:"approval_required_reviewers_enable"`
	ApprovalRequiredReviewers       *int              `json:"approval_required_reviewers"`
	DisableMergeBySelf              *bool             `json:"disable_merge_by_self"`
	CanForceMerge                   *bool             `json:"can_force_merge"`
	AddNotesAfterMerged             *bool             `json:"add_notes_after_merged"`
	MarkAutoMergedMRAsClosed        *bool             `json:"mark_auto_merged_mr_as_closed"`
	CanReopen                       *bool             `json:"can_reopen"`
	DeleteSourceBranchWhenMerged    *bool             `json:"delete_source_branch_when_merged"`
	DisableSquashMerge              *bool             `json:"disable_squash_merge"`
	AutoSquashMerge                 *bool             `json:"auto_squash_merge"`
	SquashMergeWithNoMergeCommit    *bool             `json:"squash_merge_with_no_merge_commit"`
	MergedCommitAuthor              *string           `json:"merged_commit_author"`
	ApprovalRequiredApprovers       *int              `json:"approval_required_approvers"`
	ApprovalRequiredTesters         *int              `json:"approval_required_testers"`
	IsCheckCLA                      *bool             `json:"is_check_cla"`
	IsAllowLiteMergeRequest         *bool             `json:"is_allow_lite_merge_request"`
	LiteMergeRequestPrefixTitle     *string           `json:"lite_merge_request_prefix_title"`
	CloseIssueWhenMRMerged          *bool             `json:"close_issue_when_mr_merged"`
	ApprovalApprovers               []json.RawMessage `json:"approval_approvers"`
	ApprovalTesters                 []json.RawMessage `json:"approval_testers"`
}
type PullRequestPolicy struct {
	MergeRequestSetting                       *MergeRequestPolicySettings `json:"merge_request_setting"`
	OnlyAllowMergeIfAllDiscussionsAreResolved *bool                       `json:"only_allow_merge_if_all_discussions_are_resolved"`
	OnlyAllowMergeIfPipelineSucceeds          *bool                       `json:"only_allow_merge_if_pipeline_succeeds"`
	MergeMethod                               *string                     `json:"merge_method"`
}

func policyPath(owner, repo, endpoint string) string {
	return fmt.Sprintf("/repos/%s/%s/%s", url.PathEscape(owner), url.PathEscape(repo), endpoint)
}

// All policy endpoints document 200 JSON responses. Do not retry writes:
// even idempotent settings can race with another administrator's changes.
func policyRequest(client *Client, method, path string, request, result any) error {
	var body *bytes.Reader
	payload := []byte(nil)
	if request != nil {
		var err error
		payload, err = json.Marshal(request)
		if err != nil {
			return fmt.Errorf("encode policy request: %w", err)
		}
	}
	body = bytes.NewReader(payload)
	return client.doJSONRequest(method, path, body, "application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: method == http.MethodGet}, result)
}
func GetPermissionPolicy(client *Client, owner, repo string) (PermissionPolicy, error) {
	var result PermissionPolicy
	err := policyRequest(client, http.MethodGet, policyPath(owner, repo, "transition"), nil, &result)
	if err == nil && result.MemberManagementMode == nil {
		err = fmt.Errorf("invalid permission policy response: missing memberMgntMode")
	}
	return result, err
}
func GetPullRequestPolicy(client *Client, owner, repo string) (PullRequestPolicy, error) {
	var result PullRequestPolicy
	err := policyRequest(client, http.MethodGet, policyPath(owner, repo, "pull_request_settings"), nil, &result)
	if err == nil {
		err = validatePullRequestPolicy(result)
	}
	return result, err
}
func validatePullRequestPolicy(result PullRequestPolicy) error {
	if result.MergeRequestSetting == nil || result.MergeMethod == nil ||
		result.OnlyAllowMergeIfAllDiscussionsAreResolved == nil || result.OnlyAllowMergeIfPipelineSucceeds == nil {
		return fmt.Errorf("invalid pull-request policy response: missing settings")
	}
	return validatePolicyReviewers(result.MergeRequestSetting)
}

func validatePolicyReviewers(settings *MergeRequestPolicySettings) error {
	for _, field := range []struct {
		name    string
		entries []json.RawMessage
	}{
		{"approval_approvers", settings.ApprovalApprovers},
		{"approval_testers", settings.ApprovalTesters},
	} {
		for i, entry := range field.entries {
			var object map[string]json.RawMessage
			if err := json.Unmarshal(entry, &object); err != nil || object == nil {
				return fmt.Errorf("invalid policy response: %s[%d] must be an object", field.name, i)
			}
		}
	}
	return nil
}
func CodeReviewPolicyFrom(settings PullRequestPolicy) (CodeReviewPolicy, error) {
	s := settings.MergeRequestSetting
	if s == nil || s.ApprovalRequiredApprovers == nil || s.ApprovalRequiredTesters == nil ||
		s.ApprovalApprovers == nil || s.ApprovalTesters == nil {
		return CodeReviewPolicy{}, fmt.Errorf("invalid code-review policy response: missing approval settings")
	}
	if err := validatePolicyReviewers(s); err != nil {
		return CodeReviewPolicy{}, err
	}
	return CodeReviewPolicy{s.ApprovalApprovers, s.ApprovalTesters, s.ApprovalRequiredApprovers, s.ApprovalRequiredTesters}, nil
}
func UpdatePermissionPolicy(client *Client, owner, repo string, request UpdatePermissionPolicyRequest) error {
	var result PermissionPolicyUpdateResponse
	if err := policyRequest(client, http.MethodPut, policyPath(owner, repo, "transition"), request, &result); err != nil {
		return err
	}
	if result.Code == nil || *result.Code != 1 {
		return fmt.Errorf("permission policy update was not acknowledged as successful")
	}
	return nil
}
func UpdateCodeReviewPolicy(client *Client, owner, repo string, request UpdateCodeReviewPolicyRequest) error {
	var result CodeReviewPolicyUpdateResponse
	if err := policyRequest(client, http.MethodPut, policyPath(owner, repo, "reviewer"), request, &result); err != nil {
		return err
	}
	if result.ID == nil || *result.ID <= 0 {
		return fmt.Errorf("invalid code-review update response: missing repository id")
	}
	return nil
}
func UpdatePullRequestPolicy(client *Client, owner, repo string, request UpdatePullRequestPolicyRequest) error {
	var result PullRequestPolicy
	if err := policyRequest(client, http.MethodPut, policyPath(owner, repo, "pull_request_settings"), request, &result); err != nil {
		return err
	}
	if err := validatePullRequestPolicy(result); err != nil {
		return err
	}
	return validatePolicyUpdate(request, result)
}

// Compare submitted scalars with fields actually returned by the server. Some
// settings are omitted by the API, and reviewer IDs have no documented mapping
// to the returned identity objects; neither can be verified from this response.
func validatePolicyUpdate(request UpdatePullRequestPolicyRequest, result PullRequestPolicy) error {
	wantedJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode requested policy: %w", err)
	}
	actualJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode returned policy: %w", err)
	}
	var wanted, actual, nested map[string]json.RawMessage
	if err := json.Unmarshal(wantedJSON, &wanted); err != nil {
		return fmt.Errorf("decode requested policy: %w", err)
	}
	if err := json.Unmarshal(actualJSON, &actual); err != nil {
		return fmt.Errorf("decode returned policy: %w", err)
	}
	if err := json.Unmarshal(actual["merge_request_setting"], &nested); err != nil {
		return fmt.Errorf("decode returned policy settings: %w", err)
	}
	for field, value := range wanted {
		returned, exists := actual[field]
		if !exists {
			returned = nested[field]
		}
		if len(returned) == 0 || bytes.Equal(returned, []byte("null")) {
			continue
		}
		if !bytes.Equal(value, returned) {
			return fmt.Errorf("policy update response differs from requested %s; changes may have been partially applied, inspect the policy before retrying", field)
		}
	}
	return nil
}
