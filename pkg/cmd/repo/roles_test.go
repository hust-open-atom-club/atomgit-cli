package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// repoRolesFixture keeps a tab, a newline and an unknown field so the tests can
// prove the table stays intact and JSON keeps unknown data.
const repoRolesFixture = `[{"role_id":"e6cd76d6c82f46c78c71fd7f67eaf3bf","access_level":15,"role_name":"审计\t角色\n第二行","role_chinese_name":"审计角色","role_description":"仓库自定义角色","role_type":"project-customized","member_count":2,"role_from":"custom","future_field":{"x":1}}]`

func executeRoles(t *testing.T, f *cmdutil.Factory, args ...string) (string, error) {
	t.Helper()
	cmd := newCmdRepoRoles(f)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(args)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err := cmd.Execute()
	return out.String(), err
}

func TestRepoRolesContextAndOutput(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("explicit=%t/json=%t", explicit, asJSON), func(t *testing.T) {
				f := policyFactory(t, func(r *http.Request) (int, string) {
					if r.Method != "GET" || r.URL.Path != "/api/v5/repos/team/demo/customized_roles" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "Bearer synthetic" {
						t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
					}
					return 200, repoRolesFixture
				})
				f.RepositoryResolver = func() (cmdutil.Repository, error) {
					if explicit {
						t.Fatal("explicit repository must bypass resolver")
					}
					return cmdutil.Repository{Owner: "team", Name: "demo"}, nil
				}
				var args []string
				if explicit {
					args = append(args, "team/demo")
				}
				if asJSON {
					args = append(args, "--json")
				}
				out, err := executeRoles(t, f, args...)
				if err != nil {
					t.Fatal(err)
				}

				if asJSON {
					var result []map[string]json.RawMessage
					if err := json.Unmarshal([]byte(out), &result); err != nil {
						t.Fatal(err)
					}
					if len(result) != 1 {
						t.Fatalf("result=%s", out)
					}
					if id := string(result[0]["role_id"]); id != `"e6cd76d6c82f46c78c71fd7f67eaf3bf"` {
						t.Fatalf("role_id changed: %s", id)
					}
					if _, ok := result[0]["future_field"]; !ok {
						t.Fatalf("unknown field dropped: %s", out)
					}
					return
				}

				lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
				if len(lines) != 2 {
					t.Fatalf("embedded newline broke the table: %q", out)
				}
				for _, column := range []string{"ROLE ID", "ACCESS LEVEL", "MEMBERS", "NAME"} {
					if !strings.Contains(lines[0], column) {
						t.Fatalf("missing column %q in %q", column, lines[0])
					}
				}
				if !strings.Contains(lines[1], "e6cd76d6c82f46c78c71fd7f67eaf3bf") {
					t.Fatalf("role ID missing: %q", lines[1])
				}
				if !strings.Contains(out, `审计\t角色\n第二行`) {
					t.Fatalf("tab and newline not escaped: %q", out)
				}
				for _, want := range []string{"15", "2"} {
					if !strings.Contains(lines[1], want) {
						t.Fatalf("missing numeric field %q in %q", want, lines[1])
					}
				}
			})
		}
	}
}

func TestRepoRolesAbsentFieldsAreNotFilledIn(t *testing.T) {
	f := policyFactory(t, func(*http.Request) (int, string) {
		return 200, `[{"role_id":"onlyid"}]`
	})
	out, err := executeRoles(t, f, "team/demo")
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

func TestRepoRolesEmptyList(t *testing.T) {
	f := policyFactory(t, func(*http.Request) (int, string) { return 200, `[]` })
	out, err := executeRoles(t, f, "team/demo")
	if err != nil {
		t.Fatal(err)
	}
	if out != "No customized roles found.\n" {
		t.Fatalf("output=%q", out)
	}
	out, err = executeRoles(t, f, "team/demo", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if out != "[]\n" {
		t.Fatalf("output=%q", out)
	}
}

func TestRepoRolesTableKeepsASCIIColumnsAligned(t *testing.T) {
	// ROLE ID, ACCESS LEVEL and MEMBERS are ASCII under the API contract, so
	// their byte positions must not move when NAME holds wide characters. The
	// fixture deliberately uses IDs and numbers of different lengths: an
	// implementation that joins cells with a single space, or that lets a wide
	// cell precede an aligned column, fails here.
	body := `[{"role_id":"a","role_name":"one","access_level":5,"member_count":7},` +
		`{"role_id":"role-identifier-longer","role_name":"代码审计角色","access_level":400,"member_count":22},` +
		`{"role_id":"cc","role_name":"cafe\u0301 \ud83d\ude80","access_level":55,"member_count":9}]`
	f := policyFactory(t, func(*http.Request) (int, string) { return 200, body })
	out, err := executeRoles(t, f, "team/demo")
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

func TestRepoRolesTableEscapesTableControlCharacters(t *testing.T) {
	// EscapeTSVField handles tab, newline and carriage return, but tabwriter
	// also treats a vertical tab as a cell terminator and a form feed as a
	// line-block terminator, and the command writer expands other control
	// characters after the table has been laid out. All of them must be escaped
	// before column widths are computed.
	body := `[{"role_id":"id\u000bwith-vt","role_name":"name\u000cwith-ff","access_level":5,"member_count":7},` +
		`{"role_id":"id\u001bwith-esc","role_name":"esc-name","access_level":6,"member_count":8}]`
	f := policyFactory(t, func(*http.Request) (int, string) { return 200, body })
	out, err := executeRoles(t, f, "team/demo")
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

func TestRepoRolesNameRendering(t *testing.T) {
	body := `[{"role_id":"a1","role_name":"auditor","role_chinese_name":"审计员"},` +
		`{"role_id":"b2","role_name":"same","role_chinese_name":"same"},` +
		`{"role_id":"c3"},` +
		`{"role_id":"d4","role_chinese_name":"仅中文名"},` +
		`{"role_id":"e5","role_name":"only-english"},` +
		`{"role_id":"f6","role_name":"  spaced  ","role_chinese_name":"spaced"}]`
	f := policyFactory(t, func(*http.Request) (int, string) { return 200, body })
	out, err := executeRoles(t, f, "team/demo")
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

func TestRepoRolesErrorsProduceNoOutput(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{"forbidden", 403, `{"error_message":"forbidden"}`},
		{"not found", 404, `{"error_message":"missing"}`},
		{"invalid", 200, `[{"role_id":1}]`},
	} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%t", tt.name, asJSON), func(t *testing.T) {
				f := policyFactory(t, func(*http.Request) (int, string) { return tt.status, tt.body })
				args := []string{"team/demo"}
				if asJSON {
					args = append(args, "--json")
				}
				out, err := executeRoles(t, f, args...)
				if err == nil || out != "" {
					t.Fatalf("output=%q error=%v", out, err)
				}
			})
		}
	}
}

func TestRepoRolesValidationAndAuthentication(t *testing.T) {
	cause := errors.New("credential store unavailable")
	for _, args := range [][]string{{"bad"}, {"team/.."}, {"team/a?b"}, {""}, {"team/demo", "extra"}} {
		f := repoFactory(repoCommandConfig{tokenErr: cause}, forkRoundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("invalid args made request"); return nil, nil }))
		out, err := executeRoles(t, f, args...)
		if err == nil || errors.Is(err, cause) || out != "" {
			t.Fatalf("args=%q output=%q error=%v", args, out, err)
		}
	}
	for _, authErr := range []error{config.ErrNotAuthenticated, cause} {
		f := repoFactory(repoCommandConfig{tokenErr: authErr}, forkRoundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("auth failure made request"); return nil, nil }))
		out, err := executeRoles(t, f, "team/demo")
		if !errors.Is(err, authErr) || out != "" {
			t.Fatalf("output=%q error=%v", out, err)
		}
	}
	f := policyFactory(t, func(*http.Request) (int, string) { t.Fatal("resolver failure made request"); return 500, "" })
	f.RepositoryResolver = func() (cmdutil.Repository, error) { return cmdutil.Repository{}, cause }
	if out, err := executeRoles(t, f); !errors.Is(err, cause) || out != "" {
		t.Fatalf("output=%q error=%v", out, err)
	}
}

func TestRepoRolesBidiUsesSafeWriter(t *testing.T) {
	for _, asJSON := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%t", asJSON), func(t *testing.T) {
			const name = "role\u202eoverride\u2066isolate"
			body := `[{"role_id":"id","role_name":"` + name + `"}]`
			f := policyFactory(t, func(*http.Request) (int, string) { return 200, body })
			cmd := newCmdRepoRoles(f)
			var out bytes.Buffer
			writer := cmdutil.NewSanitizingWriter(&out)
			cmd.SetOut(writer)
			args := []string{"team/demo"}
			if asJSON {
				args = append(args, "--json")
			}
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if err := cmdutil.FlushWriter(writer); err != nil {
				t.Fatal(err)
			}
			output := out.String()
			if strings.ContainsAny(output, "\u202e\u2066") {
				t.Fatalf("unsafe bidi output: %q", output)
			}
			if !strings.Contains(output, `\u202e`) || !strings.Contains(output, `\u2066`) {
				t.Fatalf("missing escape sequence: %q", output)
			}
		})
	}
}
