package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const organizationCustomRolesFixture = `[{"id":"123456","name":"admin-custom","name_cn":"管理员自定义角色","product_id":"gitcode","access_level":60,"created_at":"2024-01-01 00:00:00","role_type":"project-customized","updated_at":"2024-01-02 00:00:00","description":"组织自定义管理员角色","member_count":3,"role_from":"custom","mgnt_mode":1,"future_field":{"nested":true},"big_id":9007199254740993123}]`

func TestGetOrganizationCustomRoles(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/orgs/team%2Fname/customized_roles" || r.URL.RawQuery != "" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Accept") != "application/json" {
			t.Fatal("missing auth or accept header")
		}
		_, _ = w.Write([]byte(organizationCustomRolesFixture))
	})
	roles, err := GetOrganizationCustomRoles(client, "team/name")
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 {
		t.Fatalf("roles=%d", len(roles))
	}
	role := roles[0]

	// The organization response uses id/name/name_cn; the repository field
	// names must not be applied to it.
	if id, ok := role.StringField("id"); !ok || id != "123456" {
		t.Fatalf("id=%q present=%t", id, ok)
	}
	if name, ok := role.StringField("name_cn"); !ok || name != "管理员自定义角色" {
		t.Fatalf("name_cn=%q present=%t", name, ok)
	}
	if count, ok := role.IntegerField("member_count"); !ok || count != 3 {
		t.Fatalf("member_count=%d present=%t", count, ok)
	}
	if mode, ok := role.IntegerField("mgnt_mode"); !ok || mode != 1 {
		t.Fatalf("mgnt_mode=%d present=%t", mode, ok)
	}
	if _, ok := role.StringField("role_id"); ok {
		t.Fatal("repository field name role_id was read from the organization response")
	}

	// Unknown fields and large integers keep their exact JSON form.
	var want []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(organizationCustomRolesFixture), &want); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(role)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want[0]) {
		t.Fatalf("fields changed: %s", data)
	}
}

func TestOrganizationCustomRolesResponseValidation(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
		fail       bool
	}{
		{"empty array", `[]`, 200, false},
		{"unknown fields", `[{"id":"a","future":{"x":1}}]`, 200, false},
		{"unknown role type", `[{"id":"a","role_type":"future-customized"}]`, 200, false},
		{"null optional", `[{"id":"a","name":null,"member_count":null}]`, 200, false},
		{"absent optional", `[{"id":"a"}]`, 200, false},
		{"null", `null`, 200, true},
		{"object", `{}`, 200, true},
		{"null role", `[null]`, 200, true},
		{"missing id", `[{}]`, 200, true},
		{"blank id", `[{"id":"   "}]`, 200, true},
		{"null id", `[{"id":null}]`, 200, true},
		{"numeric id", `[{"id":123}]`, 200, true},
		{"string name", `[{"id":"a","name":1}]`, 200, true},
		{"string access level", `[{"id":"a","access_level":"60"}]`, 200, true},
		{"fractional member count", `[{"id":"a","member_count":3.5}]`, 200, true},
		{"exponent mgnt mode", `[{"id":"a","mgnt_mode":1e0}]`, 200, true},
		{"malformed", `[`, 200, true},
		{"created", organizationCustomRolesFixture, 201, true},
		{"not modified", "", 304, true},
		{"unauthorized", `{}`, 401, true},
		{"forbidden", `{}`, 403, true},
		{"not found", `{}`, 404, true},
		{"rate limited", `{}`, 429, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Retry-After", "999999")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			roles, err := GetOrganizationCustomRoles(client, "team")
			if (err != nil) != tt.fail {
				t.Fatalf("roles=%#v error=%v", roles, err)
			}
			if tt.fail && roles != nil {
				t.Fatalf("partial result=%#v", roles)
			}
			if tt.fail && tt.status == 200 && !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("missing invalid-response cause: %v", err)
			}
		})
	}
}

func TestOrganizationCustomRolesDiagnostics(t *testing.T) {
	for _, tt := range []struct {
		name, body, want string
	}{
		{"null role", `[null]`, "roles[0] must be an object"},
		{"missing id", `[{}]`, "roles[0].id must be a non-empty string"},
		{"numeric id", `[{"id":1}]`, "roles[0].id must be a string"},
		{"second role", `[{"id":"a"},{"id":"b","name_cn":[]}]`, "roles[1].name_cn must be a string"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			})
			_, err := GetOrganizationCustomRoles(client, "team")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestOrganizationCustomRolesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := NewClient("test-token")
	client.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !errors.Is(r.Context().Err(), context.Canceled) {
			t.Fatal("request did not inherit the cancelled context")
		}
		return nil, r.Context().Err()
	})}
	cancel()
	_, err := GetOrganizationCustomRoles(client.WithContext(ctx), "team")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if strings.Contains(err.Error(), "test-token") {
		t.Fatal("token exposed")
	}
}
