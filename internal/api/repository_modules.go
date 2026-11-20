package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// RepositoryModuleUpdate preserves omitted fields and explicit false values.
type RepositoryModuleUpdate struct {
	Wiki         *bool `json:"has_wiki,omitempty"`
	Issue        *bool `json:"has_issue,omitempty"`
	Security     *bool `json:"has_security,omitempty"`
	MergeRequest *bool `json:"has_merge_request,omitempty"`
	Fork         *bool `json:"has_fork,omitempty"`
	Analysis     *bool `json:"has_analysis,omitempty"`
	Discussion   *bool `json:"has_discussion,omitempty"`
}

func (r RepositoryModuleUpdate) Validate() error {
	if r.Wiki == nil && r.Issue == nil && r.Security == nil && r.MergeRequest == nil && r.Fork == nil && r.Analysis == nil && r.Discussion == nil {
		return fmt.Errorf("at least one repository module switch must be provided")
	}
	return nil
}

// RepositoryModuleUpdateUncertainError retains a failure after a write which
// may have taken effect. Callers should inspect remote settings before retrying.
type RepositoryModuleUpdateUncertainError struct{ Err error }

func (e *RepositoryModuleUpdateUncertainError) Error() string {
	return fmt.Sprintf("repository module update could not be confirmed: %v; check AtomGit before retrying", e.Err)
}
func (e *RepositoryModuleUpdateUncertainError) Unwrap() error { return e.Err }

// UpdateRepositoryModules uses the documented code=1 and msg="success"
// acknowledgement. It never reads a fictitious GET module/setting endpoint.
func UpdateRepositoryModules(client *Client, owner, repo string, request RepositoryModuleUpdate) error {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return fmt.Errorf("repository owner and name must not be empty")
	}
	if err := request.Validate(); err != nil {
		return err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode module settings: %w: %w", ErrRequestPreparation, err)
	}
	hc := *metadataHTTPClient(client.httpClient)
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var response struct {
		Code    *int64  `json:"code"`
		Message *string `json:"msg"`
	}
	path := RepositoryPath(owner, repo, "module", "setting")
	err = client.doJSONRequestContext(client.requestContext(), &hc, http.MethodPut, path, bytes.NewReader(body), "application/json", "application/json", RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, &response)
	if err != nil {
		var status *HTTPError
		if errors.Is(err, ErrRequestPreparation) || errors.As(err, &status) && status.StatusCode >= 400 && status.StatusCode < 500 {
			return fmt.Errorf("update repository modules: %w", err)
		}
		return &RepositoryModuleUpdateUncertainError{Err: err}
	}
	if response.Code == nil {
		return &RepositoryModuleUpdateUncertainError{Err: fmt.Errorf("%w: missing module update business code", ErrInvalidResponse)}
	}
	if *response.Code != 1 {
		return fmt.Errorf("repository module update rejected with business code %d", *response.Code)
	}
	if response.Message == nil || *response.Message != "success" {
		return &RepositoryModuleUpdateUncertainError{Err: fmt.Errorf("%w: missing or unexpected module update acknowledgement", ErrInvalidResponse)}
	}
	return nil
}
