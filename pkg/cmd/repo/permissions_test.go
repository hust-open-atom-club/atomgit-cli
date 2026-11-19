package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

const repoPermissionsFixture = `{"role_info":{"role_uuid":"custom-7","name":"Audit\trole\n\\\u001b[31m","cn_name":"审计员","roles_type":2,"access_level":17,"future_role":9007199254740993123},"resource_trees":[{"resource_id":22,"name":"code","scope":"code","path":"repo/code","is_own":0,"order_num":5,"unknown_resource":{"flag":false},"actions":[{"permission_id":9007199254740993124,"action":"push","name":"push\rnow","selected":false,"unknown_permission":{"flag":false}}]}],"future_matrix":true,"control":"\u007f"}`

func executePermissions(t *testing.T, f *cmdutil.Factory, args ...string) (string, error) {
	t.Helper()
	cmd := newCmdRepoPermissions(f)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(args)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err := cmd.Execute()
	return out.String(), err
}

func TestRepoPermissionsContextAndOutput(t *testing.T) {
	// RawMessage preserves JSON escapes, so send a literal DEL byte to exercise
	// the output escaping instead of relying on an already escaped fixture.
	responseBody := strings.ReplaceAll(repoPermissionsFixture, `\u007f`, "\x7f")
	for _, explicit := range []bool{false, true} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("explicit=%t/json=%t", explicit, asJSON), func(t *testing.T) {
				f := policyFactory(t, func(r *http.Request) (int, string) {
					if r.Method != "GET" || r.URL.Path != "/api/v5/repos/team/demo/collaborators/self-permission" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "Bearer synthetic" {
						t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
					}
					return 200, responseBody
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
				out, err := executePermissions(t, f, args...)
				if err != nil {
					t.Fatal(err)
				}
				if strings.ContainsAny(out, "\t\r\x1b") || strings.Contains(out, "role\n") {
					t.Fatalf("unsafe output=%q", out)
				}
				if asJSON {
					var result permissionsJSON
					if err := json.Unmarshal([]byte(out), &result); err != nil {
						t.Fatal(err)
					}
					if result.Repository != "team/demo" || len(result.Permissions) != 4 {
						t.Fatalf("result=%s", out)
					}
				} else if !strings.HasPrefix(out, "Repository: team/demo\n") {
					t.Fatalf("output=%s", out)
				}
				matrix := out
				if asJSON {
					var envelope struct {
						Permissions json.RawMessage `json:"permissions"`
					}
					if err := json.Unmarshal([]byte(out), &envelope); err != nil {
						t.Fatal(err)
					}
					matrix = string(envelope.Permissions)
				} else {
					parts := strings.SplitN(out, "\n", 3)
					if len(parts) != 3 {
						t.Fatalf("missing matrix: %q", out)
					}
					matrix = parts[2]
				}
				var got, want any
				decode := func(input string, target *any) {
					d := json.NewDecoder(strings.NewReader(input))
					d.UseNumber()
					if err := d.Decode(target); err != nil {
						t.Fatal(err)
					}
				}
				decode(matrix, &got)
				decode(repoPermissionsFixture, &want)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("matrix fields changed: %s", matrix)
				}
				if strings.ContainsRune(out, '\x7f') || !strings.Contains(out, `\u007f`) {
					t.Fatalf("DEL not JSON escaped: %q", out)
				}
			})
		}
	}
}

func TestRepoPermissionsErrorsProduceNoOutput(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{"forbidden", 403, `{"message":"forbidden"}`},
		{"not found", 404, `{"message":"missing"}`},
		{"invalid", 200, `{"role_info":{},"resource_trees":[{"actions":[{}]}]}`},
	} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%t", tt.name, asJSON), func(t *testing.T) {
				f := policyFactory(t, func(*http.Request) (int, string) { return tt.status, tt.body })
				args := []string{"team/demo"}
				if asJSON {
					args = append(args, "--json")
				}
				out, err := executePermissions(t, f, args...)
				if err == nil || out != "" {
					t.Fatalf("output=%q error=%v", out, err)
				}
			})
		}
	}
}

func TestRepoPermissionsEmptyTrees(t *testing.T) {
	f := policyFactory(t, func(*http.Request) (int, string) {
		return 200, `{"role_info":{"name":"Custom","role_uuid":"c"},"resource_trees":[]}`
	})
	out, err := executePermissions(t, f, "team/demo", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"resource_trees": []`) {
		t.Fatalf("output=%s", out)
	}
}

func TestRepoPermissionsValidationAndAuthentication(t *testing.T) {
	cause := errors.New("credential store unavailable")
	for _, args := range [][]string{{"bad"}, {"team/.."}, {"team/a?b"}, {""}, {"team/demo", "extra"}} {
		f := repoFactory(repoCommandConfig{tokenErr: cause}, forkRoundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("invalid args made request"); return nil, nil }))
		out, err := executePermissions(t, f, args...)
		if err == nil || errors.Is(err, cause) || out != "" {
			t.Fatalf("args=%q output=%q error=%v", args, out, err)
		}
	}
	for _, authErr := range []error{config.ErrNotAuthenticated, cause} {
		f := repoFactory(repoCommandConfig{tokenErr: authErr}, forkRoundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("auth failure made request"); return nil, nil }))
		out, err := executePermissions(t, f, "team/demo")
		if !errors.Is(err, authErr) || out != "" {
			t.Fatalf("output=%q error=%v", out, err)
		}
	}
	f := policyFactory(t, func(*http.Request) (int, string) { t.Fatal("resolver failure made request"); return 500, "" })
	f.RepositoryResolver = func() (cmdutil.Repository, error) { return cmdutil.Repository{}, cause }
	if out, err := executePermissions(t, f); !errors.Is(err, cause) || out != "" {
		t.Fatalf("output=%q error=%v", out, err)
	}
}
