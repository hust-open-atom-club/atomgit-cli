package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// RepositoryCustomRole preserves one customized repository role as returned by
// GET /repos/{owner}/{repo}/customized_roles. Unknown fields, string role IDs
// and exact JSON scalar forms are kept verbatim; documented fields are
// type-checked by validateRepositoryCustomRoles before the value is returned.
type RepositoryCustomRole map[string]json.RawMessage

// StringField reads a documented string field. The boolean reports whether the
// field is present and non-null, so callers can show an absent value instead of
// inventing one.
func (role RepositoryCustomRole) StringField(field string) (string, bool) {
	raw, ok := role[field]
	if !ok || isJSONNull(raw) {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

// IntegerField reads a documented integer field. The boolean reports whether
// the field is present and non-null.
func (role RepositoryCustomRole) IntegerField(field string) (int64, bool) {
	raw, ok := role[field]
	if !ok || isJSONNull(raw) {
		return 0, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return 0, false
	}
	number, ok := decoded.(json.Number)
	if !ok {
		return 0, false
	}
	// Access levels and member counts are JSON integers; a fractional or
	// exponent form is not accepted as one.
	if strings.ContainsAny(number.String(), ".eE") {
		return 0, false
	}
	value, err := number.Int64()
	if err != nil {
		return 0, false
	}
	return value, true
}

// GetRepositoryCustomRoles lists the customized roles defined on a repository.
// The endpoint is documented for authenticated user tokens only and without
// pagination parameters, so the request carries no page/per_page query and the
// full array is returned as received. This is a read-only query; no role is
// created, edited or assigned.
func GetRepositoryCustomRoles(client *Client, owner, repo string) ([]RepositoryCustomRole, error) {
	path := fmt.Sprintf("/repos/%s/%s/customized_roles", url.PathEscape(owner), url.PathEscape(repo))
	var result []RepositoryCustomRole
	if err := client.doJSONRequest(http.MethodGet, path, nil, "", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}, &result); err != nil {
		return nil, fmt.Errorf("get repository customized roles: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: repository customized roles: response must be an array", ErrInvalidResponse)
	}
	if err := validateRepositoryCustomRoles(result); err != nil {
		return nil, fmt.Errorf("%w: repository customized roles: %s", ErrInvalidResponse, err)
	}
	return result, nil
}

// validateRepositoryCustomRoles checks the fields documented for this endpoint.
// role_id identifies a role and must therefore be a non-empty JSON string; the
// remaining documented fields are optional here because a response that omits
// one must be reported as absent instead of being filled with a substitute.
// Unknown fields and unknown role_type values are preserved as received.
func validateRepositoryCustomRoles(roles []RepositoryCustomRole) error {
	stringFields := []string{"role_name", "role_chinese_name", "role_description", "role_type", "role_from", "created_at", "updated_at"}
	integerFields := []string{"access_level", "member_count"}
	for index, role := range roles {
		if role == nil {
			return fmt.Errorf("roles[%d] must be an object", index)
		}
		rawRoleID, hasRoleID := role["role_id"]
		if !hasRoleID || isJSONNull(rawRoleID) {
			return fmt.Errorf("roles[%d].role_id must be a non-empty string", index)
		}
		roleID, ok := role.StringField("role_id")
		if !ok {
			return fmt.Errorf("roles[%d].role_id must be a string", index)
		}
		if strings.TrimSpace(roleID) == "" {
			return fmt.Errorf("roles[%d].role_id must be a non-empty string", index)
		}
		for _, field := range stringFields {
			if raw, ok := role[field]; ok && !isJSONNull(raw) {
				if _, ok := role.StringField(field); !ok {
					return fmt.Errorf("roles[%d].%s must be a string", index, field)
				}
			}
		}
		for _, field := range integerFields {
			if raw, ok := role[field]; ok && !isJSONNull(raw) {
				if _, ok := role.IntegerField(field); !ok {
					return fmt.Errorf("roles[%d].%s must be an integer", index, field)
				}
			}
		}
	}
	return nil
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
