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
	// ROLE ID, ACCESS LEVEL and MEMBERS are ASCII under the API contract, so
	// their byte positions must not move when NAME holds wide characters. The
	// fixture deliberately uses IDs and numbers of different lengths: an
	// implementation that joins cells with a single space, or that lets a wide
	// cell precede an aligned column, fails here.
	body := `[{"id":"a","name":"one","access_level":5,"member_count":7},` +
		`{"id":"role-identifier-longer","name":"代码审计角色","access_level":400,"member_count":22},` +
		`{"id":"cc","name":"cafe\u0301 \ud83d\ude80","access_level":55,"member_count":9}]`
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
	// Column order is the invariant the layout relies on: NAME must be last, so
	// no aligned column follows a cell whose rune count can differ from its
	// terminal width.
	if got := strings.Join(strings.Fields(lines[0]), " "); got != "ROLE ID ACCESS LEVEL MEMBERS NAME" {
		t.Fatalf("header = %q", lines[0])
	}
	headerIndex := func(label string) int {
		index := strings.Index(lines[0], label)
		if index < 0 {
			t.Fatalf("header missing %q: %q", label, lines[0])
		}
		return index
	}
	columnIndex := func(line, value string) int {
		index := strings.Index(line, value)
		if index < 0 {
			t.Fatalf("%q missing from %q", value, line)
		}
		return index
	}
	// Every data cell must start in the column its header names. Binding each
	// value to the header column catches both a swapped data column and an
	// unaligned header.
	for row, cells := range []struct {
		id, access, members, name string
	}{
		{"a", "5", "7", "one"},
		{"role-identifier-longer", "400", "22", "代码审计角色"},
		{"cc", "55", "9", "cafe\u0301 \U0001F680"},
	} {
		line := lines[row+1]
		for _, column := range []struct {
			label, value string
		}{
			{"ROLE ID", cells.id},
			{"ACCESS LEVEL", cells.access},
			{"MEMBERS", cells.members},
		} {
			if got, want := columnIndex(line, column.value), headerIndex(column.label); got != want {
				t.Fatalf("row %d %s starts at byte %d, header column at %d\n%q\n%q", row+1, column.label, got, want, lines[0], line)
			}
		}
		if !strings.HasSuffix(line, cells.name) {
			t.Fatalf("row %d NAME = %q, want suffix %q", row+1, line, cells.name)
		}
		if got, want := len(line)-len(cells.name), headerIndex("NAME"); got != want {
			t.Fatalf("row %d NAME starts at byte %d, header column at %d", row+1, got, want)
		}
	}
}

func TestOrgRolesTableEscapesTableControlCharacters(t *testing.T) {
	// EscapeTSVField handles tab, newline and carriage return, but tabwriter
	// also treats a vertical tab as a cell terminator and a form feed as a
	// line-block terminator, and the command writer expands other control
	// characters after the table has been laid out. All of them must be escaped
	// before column widths are computed.
	body := `[{"id":"id\u000bwith-vt","name":"name\u000cwith-ff","access_level":5,"member_count":7},` +
		`{"id":"id\u001bwith-esc","name":"esc-name","access_level":6,"member_count":8}]`
	f := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, body), nil
	})
	out, _, err := executeOrgRoles(t, f, "my-org")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("control characters changed the layout: %q", out)
	}
	for _, want := range []string{`id\x0bwith-vt`, `name\x0cwith-ff`, `id\x1bwith-esc`} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q: %q", want, out)
		}
	}
	// The escaped ID is longer than the raw one; both rows must still start
	// their numeric columns at the header position.
	headerAccess := strings.Index(lines[0], "ACCESS LEVEL")
	for i, value := range []string{"5", "6"} {
		if got := strings.Index(lines[i+1], value); got != headerAccess {
			t.Fatalf("ACCESS LEVEL starts at %d, header column at %d: %q", got, headerAccess, lines[i+1])
		}
	}
}

func TestOrgRolesNameRendering(t *testing.T) {
	body := `[{"id":"a1","name":"auditor","name_cn":"审计员"},` +
		`{"id":"b2","name":"same","name_cn":"same"},` +
		`{"id":"c3"},` +
		`{"id":"d4","name_cn":"仅中文名"},` +
		`{"id":"e5","name":"only-english"},` +
		`{"id":"f6","name":"  spaced  ","name_cn":"spaced"}]`
	f := orgFactory(orgTestConfig{}, func(*http.Request) (*http.Response, error) {
		return orgResponse(http.StatusOK, body), nil
	})
	out, _, err := executeOrgRoles(t, f, "my-org")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"auditor（审计员）", "same\n", "only-english"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q: %q", want, out)
		}
	}
	if strings.Contains(out, "same（same）") {
		t.Fatalf("identical names were duplicated: %q", out)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 7 {
		t.Fatalf("lines = %d: %q", len(lines), out)
	}
	if !strings.HasSuffix(lines[3], "-") {
		t.Fatalf("absent name not rendered as -: %q", lines[3])
	}
	if !strings.HasSuffix(lines[4], "仅中文名") {
		t.Fatalf("Chinese-only name not rendered alone: %q", lines[4])
	}
	if !strings.HasSuffix(lines[5], "only-english") {
		t.Fatalf("English-only name not rendered: %q", lines[5])
	}
	// Surrounding whitespace is trimmed for both comparison and display, so the
	// two names count as equal and appear once without parentheses.
	if !strings.HasSuffix(lines[6], "spaced") || strings.Contains(out, "（spaced）") || strings.Count(out, "spaced") != 1 {
		t.Fatalf("whitespace-trimmed names were duplicated: %q", lines[6])
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
