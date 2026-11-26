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

const repositoryCustomRolesFixture = `[{"role_id":"e6cd76d6c82f46c78c71fd7f67eaf3bf","access_level":15,"role_name":"审计角色","role_chinese_name":"审计角色","role_description":"仓库自定义角色","role_type":"project-customized","member_count":2,"created_at":"2024-04-17 08:00","updated_at":"2024-04-17 08:00","role_from":"custom","future_field":{"nested":true},"big_id":9007199254740993123}]`

func TestGetRepositoryCustomRoles(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/repos/team%23one/demo%2Frepo/customized_roles" || r.URL.RawQuery != "" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Accept") != "application/json" {
			t.Fatal("missing auth or accept header")
		}
		_, _ = w.Write([]byte(repositoryCustomRolesFixture))
	})
	roles, err := GetRepositoryCustomRoles(client, "team#one", "demo/repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 {
		t.Fatalf("roles=%d", len(roles))
	}
	role := roles[0]

	// The documented string identifier must stay a string; it is what callers
	// pass back when a customized permission is required.
	if id, ok := role.StringField("role_id"); !ok || id != "e6cd76d6c82f46c78c71fd7f67eaf3bf" {
		t.Fatalf("role_id=%q present=%t", id, ok)
	}
	if level, ok := role.IntegerField("access_level"); !ok || level != 15 {
		t.Fatalf("access_level=%d present=%t", level, ok)
	}
	if count, ok := role.IntegerField("member_count"); !ok || count != 2 {
		t.Fatalf("member_count=%d present=%t", count, ok)
	}
	if from, ok := role.StringField("role_from"); !ok || from != "custom" {
		t.Fatalf("role_from=%q present=%t", from, ok)
	}

	// Unknown fields and large integers keep their exact JSON form.
	var want []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(repositoryCustomRolesFixture), &want); err != nil {
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

func TestRepositoryCustomRolesResponseValidation(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
		fail       bool
	}{
		{"empty array", `[]`, 200, false},
		{"unknown fields", `[{"role_id":"a","future":{"x":1}}]`, 200, false},
		{"unknown role type", `[{"role_id":"a","role_type":"future-customized"}]`, 200, false},
		{"null optional", `[{"role_id":"a","role_from":null,"access_level":null}]`, 200, false},
		{"absent optional", `[{"role_id":"a"}]`, 200, false},
		{"null", `null`, 200, true},
		{"object", `{}`, 200, true},
		{"null role", `[null]`, 200, true},
		{"missing role id", `[{}]`, 200, true},
		{"blank role id", `[{"role_id":"   "}]`, 200, true},
		{"null role id", `[{"role_id":null}]`, 200, true},
		{"numeric role id", `[{"role_id":123}]`, 200, true},
		{"string name", `[{"role_id":"a","role_name":1}]`, 200, true},
		{"string description", `[{"role_id":"a","role_description":[]}]`, 200, true},
		{"string access level", `[{"role_id":"a","access_level":"15"}]`, 200, true},
		{"fractional access level", `[{"role_id":"a","access_level":15.5}]`, 200, true},
		{"exponent member count", `[{"role_id":"a","member_count":1e3}]`, 200, true},
		{"malformed", `[`, 200, true},
		{"created", repositoryCustomRolesFixture, 201, true},
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
			roles, err := GetRepositoryCustomRoles(client, "team", "demo")
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

func TestRepositoryCustomRolesDiagnostics(t *testing.T) {
	for _, tt := range []struct {
		name, body, want string
	}{
		{"null role", `[null]`, "roles[0] must be an object"},
		{"missing id", `[{}]`, "roles[0].role_id must be a non-empty string"},
		{"numeric id", `[{"role_id":1}]`, "roles[0].role_id must be a string"},
		{"second role", `[{"role_id":"a"},{"role_id":"b","member_count":"1"}]`, "roles[1].member_count must be an integer"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			})
			_, err := GetRepositoryCustomRoles(client, "team", "demo")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestRepositoryCustomRolesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := NewClient("test-token")
	client.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !errors.Is(r.Context().Err(), context.Canceled) {
			t.Fatal("request did not inherit the cancelled context")
		}
		return nil, r.Context().Err()
	})}
	cancel()
	_, err := GetRepositoryCustomRoles(client.WithContext(ctx), "team", "demo")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if strings.Contains(err.Error(), "test-token") {
		t.Fatal("token exposed")
	}
}
