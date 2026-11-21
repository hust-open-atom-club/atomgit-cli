package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// UpdateRepositorySettingsRequest contains only documented writable fields.
// Pointers preserve explicit false and empty strings in partial updates.
type UpdateRepositorySettingsRequest struct {
	DisableFork                           *bool   `json:"disable_fork,omitempty"`
	ForbiddenDeveloperCreateBranch        *bool   `json:"forbidden_developer_create_branch,omitempty"`
	ForbiddenDeveloperCreateTag           *bool   `json:"forbidden_developer_create_tag,omitempty"`
	ForbiddenCommitterCreateBranch        *bool   `json:"forbidden_committer_create_branch,omitempty"`
	ForbiddenDeveloperCreateBranchUserIDs *string `json:"forbidden_developer_create_branch_user_ids,omitempty"`
	BranchNameRegex                       *string `json:"branch_name_regex,omitempty"`
	TagNameRegex                          *string `json:"tag_name_regex,omitempty"`
	GeneratePreMergeRef                   *bool   `json:"generate_pre_merge_ref,omitempty"`
	RebaseDisableTriggerWebhook           *bool   `json:"rebase_disable_trigger_webhook,omitempty"`
	OpenGPGVerified                       *bool   `json:"open_gpg_verified,omitempty"`
	IncludeLFSObjects                     *bool   `json:"include_lfs_objects,omitempty"`
	ActionPipelineEnabled                 *bool   `json:"action_pipeline_enabled,omitempty"`
	BlockCrossRepoPRTrigger               *bool   `json:"block_cross_repo_pr_trigger,omitempty"`
	ShowBranchCreator                     *bool   `json:"show_branch_creator,omitempty"`
}

func (r UpdateRepositorySettingsRequest) Validate() error {
	if r == (UpdateRepositorySettingsRequest{}) {
		return fmt.Errorf("at least one repository setting must be provided")
	}
	return nil
}

// RepositorySettingsBranchUser retains nullable whitelist fields without
// inventing IDs or names. Numeric IDs are decoded directly as integers.
type RepositorySettingsBranchUser struct {
	ID         *int64  `json:"id"`
	PushRuleID *int64  `json:"push_rule_id"`
	UserID     *int64  `json:"user_id"`
	UserName   *string `json:"user_name"`
}

// RepositorySettings covers the GET and PUT response fields. Missing or null
// settings remain null in JSON; in particular they must not become false.
type RepositorySettings struct {
	ForbiddenDeveloperCreateBranch      *bool                           `json:"forbidden_developer_create_branch"`
	ForbiddenDeveloperCreateTag         *bool                           `json:"forbidden_developer_create_tag"`
	ForbiddenCommitterCreateBranch      *bool                           `json:"forbidden_committer_create_branch"`
	GeneratePreMergeRef                 *bool                           `json:"generate_pre_merge_ref"`
	ForbiddenGitLabAccess               *bool                           `json:"forbidden_gitlab_access"`
	RebaseDisableTriggerWebhook         *bool                           `json:"rebase_disable_trigger_webhook"`
	ActionPipelineEnabled               *bool                           `json:"action_pipeline_enabled"`
	BlockCrossRepoPRTrigger             *bool                           `json:"block_cross_repo_pr_trigger"`
	ShowBranchCreator                   *bool                           `json:"show_branch_creator"`
	DisableFork                         *bool                           `json:"disable_fork"`
	IncludeLFSObjects                   *bool                           `json:"include_lfs_objects"`
	BranchNameRegex                     *string                         `json:"branch_name_regex"`
	TagNameRegex                        *string                         `json:"tag_name_regex"`
	OpenGPGVerified                     *bool                           `json:"open_gpg_verified"`
	ForbiddenDeveloperCreateBranchUsers []*RepositorySettingsBranchUser `json:"forbidden_developer_create_branch_users"`
}

func (s RepositorySettings) validate() error {
	if s.ForbiddenDeveloperCreateBranch == nil && s.ForbiddenDeveloperCreateTag == nil &&
		s.ForbiddenCommitterCreateBranch == nil && s.GeneratePreMergeRef == nil &&
		s.ForbiddenGitLabAccess == nil && s.RebaseDisableTriggerWebhook == nil &&
		s.ActionPipelineEnabled == nil && s.BlockCrossRepoPRTrigger == nil &&
		s.ShowBranchCreator == nil && s.DisableFork == nil && s.IncludeLFSObjects == nil &&
		s.BranchNameRegex == nil && s.TagNameRegex == nil && s.OpenGPGVerified == nil &&
		s.ForbiddenDeveloperCreateBranchUsers == nil {
		return fmt.Errorf("%w: repository settings contain no known values", ErrInvalidResponse)
	}
	for _, user := range s.ForbiddenDeveloperCreateBranchUsers {
		if user == nil {
			return fmt.Errorf("%w: null repository settings whitelist user", ErrInvalidResponse)
		}
	}
	return nil
}

// RepositorySettingsUpdateUncertainError retains failures after a PUT that
// may have changed settings. Callers should check AtomGit before retrying.
type RepositorySettingsUpdateUncertainError struct{ Err error }

func (e *RepositorySettingsUpdateUncertainError) Error() string {
	return fmt.Sprintf("update repository settings: %v; settings may have changed, check AtomGit before retrying", e.Err)
}
func (e *RepositorySettingsUpdateUncertainError) Unwrap() error { return e.Err }

func GetRepositorySettings(client *Client, owner, repo string) (RepositorySettings, error) {
	settings, err := repositorySettingsRequest(client, owner, repo, http.MethodGet, nil)
	if err != nil {
		return RepositorySettings{}, fmt.Errorf("get repository settings: %w", err)
	}
	return settings, nil
}

func UpdateRepositorySettings(client *Client, owner, repo string, request UpdateRepositorySettingsRequest) (RepositorySettings, error) {
	if err := request.Validate(); err != nil {
		return RepositorySettings{}, err
	}
	settings, err := repositorySettingsRequest(client, owner, repo, http.MethodPut, &request)
	if err != nil {
		status, isHTTP := errors.AsType[*HTTPError](err)
		uncertainHTTP := isHTTP && ((status.StatusCode >= 200 && status.StatusCode < 400) || status.StatusCode >= 500)
		if !errors.Is(err, ErrRequestPreparation) && (!isHTTP || uncertainHTTP) {
			return RepositorySettings{}, &RepositorySettingsUpdateUncertainError{Err: err}
		}
		return RepositorySettings{}, fmt.Errorf("update repository settings: %w", err)
	}
	return settings, nil
}

func repositorySettingsRequest(client *Client, owner, repo, method string, request *UpdateRepositorySettingsRequest) (RepositorySettings, error) {
	var payload []byte
	contentType := ""
	if request != nil {
		var err error
		payload, err = json.Marshal(request)
		if err != nil {
			return RepositorySettings{}, fmt.Errorf("encode repository settings: %w: %w", ErrRequestPreparation, err)
		}
		contentType = "application/json"
	}
	httpClient := *metadataHTTPClient(client.httpClient)
	if method == http.MethodPut {
		httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	var settings *RepositorySettings
	policy := RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: method == http.MethodGet}
	if err := client.doJSONRequestContext(client.requestContext(), &httpClient, method,
		policyPath(owner, repo, "repo_settings"), bytes.NewReader(payload), contentType, "application/json", policy, &settings); err != nil {
		return RepositorySettings{}, err
	}
	if settings == nil {
		return RepositorySettings{}, fmt.Errorf("%w: missing repository settings object", ErrInvalidResponse)
	}
	if err := settings.validate(); err != nil {
		return RepositorySettings{}, err
	}
	return *settings, nil
}
