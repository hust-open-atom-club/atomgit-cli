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
				for _, column := range []string{"ROLE ID", "NAME", "CHINESE NAME", "ACCESS LEVEL", "MEMBERS"} {
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
	// ID, NAME, CHINESE NAME, ACCESS LEVEL and MEMBERS: four absent fields.
	if got := strings.Count(lines[1], "-"); got != 4 {
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
