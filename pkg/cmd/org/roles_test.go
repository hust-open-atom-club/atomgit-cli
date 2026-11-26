package org

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// orgRolesFixture keeps a tab and a newline in the Chinese name plus an unknown
// field so the tests can prove the table stays intact and JSON keeps unknown
// data.
const orgRolesFixture = `[{"id":"123456","name":"admin-custom","name_cn":"管理员\t自定义角色\n第二行","description":"组织自定义管理员角色","product_id":"gitcode","access_level":60,"role_type":"project-customized","member_count":3,"role_from":"custom","mgnt_mode":1,"future_field":{"x":1}}]`

// executeOrgRoles runs the command with separate stdout/stderr capture and
// without Cobra's own error and usage printing, so output assertions only see
// what the command writes to its writer.
func executeOrgRoles(t *testing.T, f *cmdutil.Factory, args ...string) (string, string, error) {
	t.Helper()
	cmd := newCmdOrgRoles(f)
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err := cmd.Execute()
	return out.String(), stderr.String(), err
}

func TestOrgRolesOutput(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		asJSON     bool
		contains   []string
		wantJSON   string
		lines      int
	}{
		{name: "text", body: orgRolesFixture, contains: []string{"ROLE ID", "ACCESS LEVEL", "MEMBERS", "NAME", "123456", "admin-custom", `管理员\t自定义角色\n第二行`, "60", "3"}, lines: 2},
		{name: "JSON", body: orgRolesFixture, asJSON: true, wantJSON: orgRolesFixture},
		{name: "empty text", body: `[]`, contains: []string{`No customized roles found for organization "my organization".`}, lines: 1},
		{name: "empty JSON", body: `[]`, asJSON: true, wantJSON: `[]`},
		{name: "absent fields", body: `[{"id":"onlyid"}]`, contains: []string{"onlyid", "-"}, lines: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			f := orgFactory(orgTestConfig{}, func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/api/v5/orgs/my%20organization/customized_roles" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
				}
				if req.URL.RawQuery != "" {
					t.Fatalf("unexpected query: %q", req.URL.RawQuery)
				}
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("Authorization = %q", got)
				}
				return orgResponse(http.StatusOK, tc.body), nil
			})
			args := []string{"my organization"}
			if tc.asJSON {
				args = append(args, "--json")
			}
			out, _, err := executeOrgRoles(t, f, args...)
			if err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want 1", requests)
			}
			if tc.asJSON {
				assertJSONEqual(t, out, tc.wantJSON)
				return
			}
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			if len(lines) != tc.lines {
				t.Fatalf("lines = %d, want %d: %q", len(lines), tc.lines, out)
			}
			for _, want := range tc.contains {
				if !strings.Contains(out, want) {
					t.Fatalf("output missing %q: %s", want, out)
				}
			}
		})
	}
}

func TestOrgRolesAbsentFieldsAreNotFilledIn(t *testing.T) {
	f := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, `[{"id":"onlyid"}]`), nil
	})
	out, _, err := executeOrgRoles(t, f, "my-org")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("output=%q", out)
	}
	// ACCESS LEVEL, MEMBERS and NAME: three absent fields.
	if got := strings.Count(lines[1], "-"); got != 3 {
		t.Fatalf("absent fields were filled in: %q", lines[1])
	}
}

func TestOrgRolesTableKeepsASCIIColumnsAligned(t *testing.T) {
	// NAME is the trailing column, so names containing CJK, combining marks or
	// emoji cannot shift the ASCII columns. Positions are compared between rows
	// and never computed from the implementation under test.
	body := `[{"id":"a1","name":"abcd","access_level":15,"member_count":7},` +
		`{"id":"b2","name":"代码审计角色","access_level":40,"member_count":22},` +
		`{"id":"c3","name":"cafe\u0301 \ud83d\ude80","access_level":55,"member_count":9}]`
	f := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, body), nil
	})
	out, _, err := executeOrgRoles(t, f, "my-org")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("lines = %d: %q", len(lines), out)
	}
	columnIndex := func(line, value string) int {
		index := strings.Index(line, value)
		if index < 0 {
			t.Fatalf("%q missing from %q", value, line)
		}
		return index
	}
	for _, column := range []struct {
		name   string
		values []string
	}{
		{"ACCESS LEVEL", []string{"15", "40", "55"}},
		{"MEMBERS", []string{"7", "22", "9"}},
	} {
		want := columnIndex(lines[1], column.values[0])
		for i, value := range column.values[1:] {
			if got := columnIndex(lines[i+2], value); got != want {
				t.Fatalf("%s starts at byte %d and %d\n%q\n%q", column.name, want, got, lines[1], lines[i+2])
			}
		}
	}
}

func TestOrgRolesNameRendering(t *testing.T) {
	body := `[{"id":"a1","name":"auditor","name_cn":"审计员"},` +
		`{"id":"b2","name":"same","name_cn":"same"},` +
		`{"id":"c3"},` +
		`{"id":"d4","name_cn":"仅中文名"}]`
	f := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, body), nil
	})
	out, _, err := executeOrgRoles(t, f, "my-org")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"auditor（审计员）", "same\n", "仅中文名"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q: %q", want, out)
		}
	}
	if strings.Contains(out, "same（same）") {
		t.Fatalf("identical names were duplicated: %q", out)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.HasSuffix(lines[3], "-") {
		t.Fatalf("absent name not rendered as -: %q", lines[3])
	}
}

func TestOrgRolesValidationRunsBeforeAuth(t *testing.T) {
	tokenCalls := 0
	for _, args := range [][]string{{}, {""}, {"a/b"}, {"my-org", "extra"}} {
		f := orgFactory(orgTestConfig{tokenErr: errors.New("credential store unavailable"), tokenCalls: &tokenCalls}, func(*http.Request) (*http.Response, error) {
			t.Fatalf("invalid args made request")
			return nil, nil
		})
		out, _, err := executeOrgRoles(t, f, args...)
		if err == nil {
			t.Fatalf("args=%q: expected a validation error", args)
		}
		if out != "" {
			t.Fatalf("args=%q: wrote output before validation: %q", args, out)
		}
		if strings.Contains(err.Error(), "credential store") {
			t.Fatalf("args=%q: authentication ran before argument validation", args)
		}
	}
	if tokenCalls != 0 {
		t.Fatalf("token was read %d times for invalid arguments", tokenCalls)
	}
}

func TestOrgRolesErrorsProduceNoOutput(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{"forbidden", 403, `{"error_message":"forbidden"}`},
		{"not found", 404, `{"error_message":"missing"}`},
		{"invalid", 200, `[{"id":1}]`},
	} {
		for _, asJSON := range []bool{false, true} {
			t.Run(tt.name, func(t *testing.T) {
				f := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
					return orgResponse(tt.status, tt.body), nil
				})
				args := []string{"my-org"}
				if asJSON {
					args = append(args, "--json")
				}
				out, _, err := executeOrgRoles(t, f, args...)
				if err == nil || out != "" {
					t.Fatalf("output=%q error=%v", out, err)
				}
			})
		}
	}
}

func TestOrgRolesAuthenticationError(t *testing.T) {
	for _, authErr := range []error{config.ErrNotAuthenticated, errors.New("credential store unavailable")} {
		f := orgFactory(orgTestConfig{tokenErr: authErr}, func(*http.Request) (*http.Response, error) {
			t.Fatalf("authentication failure made request")
			return nil, nil
		})
		out, _, err := executeOrgRoles(t, f, "my-org")
		if !errors.Is(err, authErr) || out != "" {
			t.Fatalf("output=%q error=%v", out, err)
		}
	}
}

func TestOrgRolesJSONPreservesUnknownFields(t *testing.T) {
	f := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, orgRolesFixture), nil
	})
	out, _, err := executeOrgRoles(t, f, "my-org", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("result=%s", out)
	}
	if _, ok := result[0]["future_field"]; !ok {
		t.Fatalf("unknown field dropped: %s", out)
	}
	if id := string(result[0]["id"]); id != `"123456"` {
		t.Fatalf("id changed: %s", id)
	}
}
