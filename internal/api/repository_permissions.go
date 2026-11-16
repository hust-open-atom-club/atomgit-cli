package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// RepositoryPermissions preserves the server's permission matrix, including
// unknown fields and numeric identifiers without floating-point conversion.
// The required role_info and resource_trees containers are validated below.
type RepositoryPermissions map[string]json.RawMessage

// GetRepositoryPermissions reads the caller's role and permission points. A
// successful request is not evidence of any permission beyond the response.
func GetRepositoryPermissions(client *Client, owner, repo string) (RepositoryPermissions, error) {
	path := fmt.Sprintf("/repos/%s/%s/collaborators/self-permission", url.PathEscape(owner), url.PathEscape(repo))
	var result RepositoryPermissions
	if err := client.doJSONRequest(http.MethodGet, path, nil, "", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}, &result); err != nil {
		return nil, fmt.Errorf("get repository permissions: %w", err)
	}
	if err := validateRepositoryPermissions(result); err != nil {
		return nil, fmt.Errorf("%w: repository permissions: %s", ErrInvalidResponse, err)
	}
	return result, nil
}

func validateRepositoryPermissions(result RepositoryPermissions) error {
	var role map[string]json.RawMessage
	if err := json.Unmarshal(result["role_info"], &role); err != nil || role == nil {
		return fmt.Errorf("role_info must be an object")
	}
	var resources []map[string]json.RawMessage
	if err := json.Unmarshal(result["resource_trees"], &resources); err != nil || resources == nil {
		return fmt.Errorf("resource_trees must be an array")
	}
	for i, resource := range resources {
		var actions []map[string]json.RawMessage
		if resource == nil {
			return fmt.Errorf("resource_trees[%d] must be an object", i)
		}
		if err := json.Unmarshal(resource["actions"], &actions); err != nil || actions == nil {
			return fmt.Errorf("resource_trees[%d].actions must be an array", i)
		}
		for j, action := range actions {
			var selected *bool
			if err := json.Unmarshal(action["selected"], &selected); err != nil || selected == nil {
				return fmt.Errorf("resource_trees[%d].actions[%d].selected must be a boolean", i, j)
			}
		}
	}
	return nil
}
