package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// OrganizationCustomRole preserves one customized organization role as returned
// by GET /orgs/{org}/customized_roles. The organization response documents
// different field names than the repository response (id/name/name_cn instead
// of role_id/role_name), so this type is deliberately separate: it must not be
// decoded with the repository role field names.
type OrganizationCustomRole map[string]json.RawMessage

// StringField reads a documented string field. The boolean reports whether the
// field is present and non-null, so callers can show an absent value instead of
// inventing one.
func (role OrganizationCustomRole) StringField(field string) (string, bool) {
	raw, ok := role[field]
	if !ok || organizationRoleJSONNull(raw) {
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
func (role OrganizationCustomRole) IntegerField(field string) (int64, bool) {
	raw, ok := role[field]
	if !ok || organizationRoleJSONNull(raw) {
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
	// Access levels, member counts and management modes are JSON integers; a
	// fractional or exponent form is not accepted as one.
	if strings.ContainsAny(number.String(), ".eE") {
		return 0, false
	}
	value, err := number.Int64()
	if err != nil {
		return 0, false
	}
	return value, true
}

// GetOrganizationCustomRoles lists the customized roles defined on an
// organization. The endpoint is documented for authenticated user tokens only
// and without pagination parameters, so the request carries no page/per_page
// query and the full array is returned as received. This is a read-only query;
// no role is created, edited or assigned.
func GetOrganizationCustomRoles(client *Client, organization string) ([]OrganizationCustomRole, error) {
	path := fmt.Sprintf("/orgs/%s/customized_roles", url.PathEscape(organization))
	var result []OrganizationCustomRole
	if err := client.doJSONRequest(http.MethodGet, path, nil, "", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}, &result); err != nil {
		return nil, fmt.Errorf("get organization customized roles: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: organization customized roles: response must be an array", ErrInvalidResponse)
	}
	if err := validateOrganizationCustomRoles(result); err != nil {
		return nil, fmt.Errorf("%w: organization customized roles: %w", ErrInvalidResponse, err)
	}
	return result, nil
}

// validateOrganizationCustomRoles checks the fields documented for this
// endpoint. id identifies a role and must therefore be a non-empty JSON string;
// the remaining documented fields are optional here because a response that
// omits one must be reported as absent instead of being filled with a
// substitute. Unknown fields and unknown role_type values are preserved as
// received.
func validateOrganizationCustomRoles(roles []OrganizationCustomRole) error {
	stringFields := []string{"name", "name_cn", "description", "product_id", "role_type", "role_from", "created_at", "updated_at"}
	integerFields := []string{"access_level", "member_count", "mgnt_mode"}
	for index, role := range roles {
		if role == nil {
			return fmt.Errorf("roles[%d] must be an object", index)
		}
		rawID, hasID := role["id"]
		if !hasID || organizationRoleJSONNull(rawID) {
			return fmt.Errorf("roles[%d].id must be a non-empty string", index)
		}
		roleID, ok := role.StringField("id")
		if !ok {
			return fmt.Errorf("roles[%d].id must be a string", index)
		}
		if strings.TrimSpace(roleID) == "" {
			return fmt.Errorf("roles[%d].id must be a non-empty string", index)
		}
		for _, field := range stringFields {
			if raw, ok := role[field]; ok && !organizationRoleJSONNull(raw) {
				if _, ok := role.StringField(field); !ok {
					return fmt.Errorf("roles[%d].%s must be a string", index, field)
				}
			}
		}
		for _, field := range integerFields {
			if raw, ok := role[field]; ok && !organizationRoleJSONNull(raw) {
				if _, ok := role.IntegerField(field); !ok {
					return fmt.Errorf("roles[%d].%s must be an integer", index, field)
				}
			}
		}
	}
	return nil
}

func organizationRoleJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
