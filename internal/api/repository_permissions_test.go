package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

const permissionsFixture = `{"role_info":{"role_uuid":"custom-1","name":"Auditor","cn_name":"审计员","roles_type":2,"access_level":17,"future_role":{"id":9007199254740993123}},"resource_trees":[{"resource_id":9007199254740993123,"name":"code","cn_name":"代码","scope":"code","path":"repo/code","is_own":0,"order_num":1,"future_resource":false,"actions":[{"permission_id":9007199254740993124,"action":"push","name":"push","cn_name":"推送","selected":false,"future_action":{"allowed":false}},{"permission_id":15,"action":"download","name":"download","selected":true}]}],"future_matrix":{"version":"next"}}`

func TestGetRepositoryPermissions(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/repos/team%23one/demo%2Frepo/collaborators/self-permission" || r.URL.RawQuery != "" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Accept") != "application/json" {
			t.Fatal("missing auth or accept header")
		}
		_, _ = w.Write([]byte(permissionsFixture))
	})
	result, err := GetRepositoryPermissions(client, "team#one", "demo/repo")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var want, got map[string]json.RawMessage
	_ = json.Unmarshal([]byte(permissionsFixture), &want)
	_ = json.Unmarshal(data, &got)
	if len(got) != len(want) {
		t.Fatalf("fields lost: %s", data)
	}
	for key, value := range want {
		if string(got[key]) != string(value) {
			t.Fatalf("%s changed: %s", key, got[key])
		}
	}
}

func TestRepositoryPermissionsResponseValidation(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
		fail       bool
	}{
		{"empty trees", `{"role_info":{},"resource_trees":[]}`, 200, false},
		{"empty actions", `{"role_info":{},"resource_trees":[{"actions":[]}]}`, 200, false},
		{"null", `null`, 200, true},
		{"array", `[]`, 200, true},
		{"missing fields", `{}`, 200, true},
		{"null role", `{"role_info":null,"resource_trees":[]}`, 200, true},
		{"invalid role", `{"role_info":[],"resource_trees":[]}`, 200, true},
		{"null trees", `{"role_info":{},"resource_trees":null}`, 200, true},
		{"null resource", `{"role_info":{},"resource_trees":[null]}`, 200, true},
		{"missing actions", `{"role_info":{},"resource_trees":[{}]}`, 200, true},
		{"null actions", `{"role_info":{},"resource_trees":[{"actions":null}]}`, 200, true},
		{"null action", `{"role_info":{},"resource_trees":[{"actions":[null]}]}`, 200, true},
		{"missing selected", `{"role_info":{},"resource_trees":[{"actions":[{}]}]}`, 200, true},
		{"null selected", `{"role_info":{},"resource_trees":[{"actions":[{"selected":null}]}]}`, 200, true},
		{"string selected", `{"role_info":{},"resource_trees":[{"actions":[{"selected":"false"}]}]}`, 200, true},
		{"malformed", `{`, 200, true},
		{"created", permissionsFixture, 201, true},
		{"not modified", "", 304, true},
		{"unauthorized", `{}`, 401, true},
		{"forbidden", `{}`, 403, true},
		{"not found", `{}`, 404, true},
		{"rate limited", `{}`, 429, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "999999")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			result, err := GetRepositoryPermissions(client, "team", "demo")
			if (err != nil) != tt.fail {
				t.Fatalf("result=%#v error=%v", result, err)
			}
			if tt.fail && result != nil {
				t.Fatalf("partial result=%#v", result)
			}
			if tt.fail && tt.status == 200 && !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("missing invalid-response cause: %v", err)
			}
		})
	}
}

func TestRepositoryPermissionsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := NewClient("test-token")
	client.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !errors.Is(r.Context().Err(), context.Canceled) {
			t.Fatal("request did not inherit the cancelled context")
		}
		return nil, r.Context().Err()
	})}
	cancel()
	_, err := GetRepositoryPermissions(client.WithContext(ctx), "team", "demo")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if strings.Contains(err.Error(), "test-token") {
		t.Fatal("token exposed")
	}
}
