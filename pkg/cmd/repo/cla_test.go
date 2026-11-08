package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func executeCLA(t *testing.T, f *cmdutil.Factory, input string, args ...string) (string, string, error) {
	t.Helper()
	cmd := newCmdRepoCLA(f)
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	return out.String(), stderr.String(), err
}

func TestRepoCLAList(t *testing.T) {
	for _, empty := range []bool{false, true} {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("empty=%t/json=%t", empty, asJSON), func(t *testing.T) {
				f := policyFactory(t, func(r *http.Request) (int, string) {
					if r.Method != "GET" || r.URL.Path != "/api/v5/repos/team/demo/clas" {
						t.Fatalf("request=%s %s", r.Method, r.URL)
					}
					if empty {
						return 200, `{"data":[]}`
					}
					return 200, `{"data":[{"id":"cla-id","name":"Agreement"}]}`
				})
				args := []string{"list", "team/demo"}
				if asJSON {
					args = append(args, "--json")
				}
				out, stderr, err := executeCLA(t, f, "", args...)
				if err != nil || stderr != "" {
					t.Fatalf("error=%v stderr=%s", err, stderr)
				}
				if asJSON {
					var result []map[string]string
					if err := json.Unmarshal([]byte(out), &result); err != nil {
						t.Fatal(err)
					}
					want := 1
					if empty {
						want = 0
					}
					if result == nil || len(result) != want {
						t.Fatalf("output=%s", out)
					}
				} else {
					want := "cla-id\tAgreement\n"
					if empty {
						want = ""
					}
					if out != want {
						t.Fatalf("output=%q", out)
					}
				}
			})
		}
	}
}

func TestRepoCLAListEscapesTSVFields(t *testing.T) {
	id, name := "cla\\id\tpart\nnext\r", "Agreement\tExtra\nFake row\r\\end"
	body, err := json.Marshal(map[string]any{"data": []map[string]string{{"id": id, "name": name}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, asJSON := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%t", asJSON), func(t *testing.T) {
			f := policyFactory(t, func(*http.Request) (int, string) { return 200, string(body) })
			args := []string{"list", "team/demo"}
			if asJSON {
				args = append(args, "--json")
			}
			out, stderr, err := executeCLA(t, f, "", args...)
			if err != nil || stderr != "" {
				t.Fatalf("error=%v stderr=%q", err, stderr)
			}
			if asJSON {
				var got []map[string]string
				if err := json.Unmarshal([]byte(out), &got); err != nil {
					t.Fatal(err)
				}
				if len(got) != 1 || got[0]["id"] != id || got[0]["name"] != name {
					t.Fatalf("JSON changed field values: %q", out)
				}
				return
			}
			want := `cla\\id\tpart\nnext\r` + "\t" + `Agreement\tExtra\nFake row\r\\end` + "\n"
			if out != want {
				t.Fatalf("output=%q want=%q", out, want)
			}
		})
	}
}

func TestRepoCLAWrite(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want map[string]any
	}{
		{"omitted", []string{"set", "team/demo", "--cla-id", "opaque-id"}, map[string]any{"cla_id": "opaque-id"}},
		{"true", []string{"set", "team/demo", "--cla-id", "opaque-id", "--enforce-check"}, map[string]any{"cla_id": "opaque-id", "enforce_check": true}},
		{"false", []string{"set", "team/demo", "--cla-id", "opaque-id", "--enforce-check=false"}, map[string]any{"cla_id": "opaque-id", "enforce_check": false}},
		{"disable", []string{"disable", "team/demo"}, map[string]any{}},
	} {
		for _, confirmation := range []string{"flag", "yes", "no", "eof"} {
			t.Run(tc.name+"/"+confirmation, func(t *testing.T) {
				calls := 0
				f := policyFactory(t, func(r *http.Request) (int, string) {
					calls++
					if r.Method != "PUT" || r.URL.Path != "/api/v5/repos/team/demo/cla" {
						t.Fatalf("request=%s %s", r.Method, r.URL)
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(body, tc.want) {
						t.Fatalf("body=%v want=%v", body, tc.want)
					}
					return 200, ""
				})
				args := append([]string{}, tc.args...)
				input := ""
				if confirmation == "flag" {
					args = append(args, "--yes")
				}
				if confirmation == "yes" {
					input = "yes\n"
				}
				if confirmation == "no" {
					input = "n\n"
				}
				out, stderr, err := executeCLA(t, f, input, args...)
				if err != nil {
					t.Fatal(err)
				}
				if confirmation == "no" || confirmation == "eof" {
					if calls != 0 || out != "" {
						t.Fatalf("cancelled calls=%d out=%s", calls, out)
					}
				} else {
					if calls != 1 || out == "" {
						t.Fatalf("calls=%d out=%s", calls, out)
					}
				}
				if confirmation == "flag" {
					if stderr != "" {
						t.Fatalf("stderr=%s", stderr)
					}
				} else {
					if !strings.Contains(stderr, "Repository: team/demo") || !strings.Contains(stderr, "CLA action: "+tc.args[0]) {
						t.Fatalf("stderr=%s", stderr)
					}
					if tc.name == "false" && !strings.Contains(stderr, "Enforce check: false") {
						t.Fatalf("stderr=%s", stderr)
					}
				}
			})
		}
	}
}

func TestRepoCLAInvalidArgumentsBeforeNetwork(t *testing.T) {
	for _, args := range [][]string{
		{"set", "team/demo", "--yes"}, {"set", "team/demo", "--cla-id="}, {"set", "team/demo", "--cla-id", " "},
		{"disable", "team/demo", "--cla-id", "id"}, {"disable", "team/demo", "--enforce-check=false"},
		{"set", "team/demo", "--cla-id", "id", "--enforce-check=invalid"}, {"list", "a", "b"},
		{"set", "a/b/c", "--cla-id", "id", "--yes"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			f := policyFactory(t, func(*http.Request) (int, string) { t.Fatal("network accessed with invalid args"); return 500, "" })
			f.Config = forbiddenCLAConfig{t}
			if _, _, err := executeCLA(t, f, "", args...); err == nil {
				t.Fatal("accepted invalid arguments")
			}
		})
	}
}

type forbiddenCLAConfig struct{ t *testing.T }

func (f forbiddenCLAConfig) GetToken() (string, error) { f.t.Fatal("read credentials"); return "", nil }
func (f forbiddenCLAConfig) GetUser() (string, error)  { f.t.Fatal("read user"); return "", nil }
func (f forbiddenCLAConfig) GetHost() string           { return "atomgit.com" }

func TestRepoCLARepositoryContextAndErrors(t *testing.T) {
	for _, action := range []string{"list", "set", "disable"} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/explicit=%t", action, explicit), func(t *testing.T) {
				f := policyFactory(t, func(r *http.Request) (int, string) {
					owner := "inferred"
					if explicit {
						owner = "explicit"
					}
					if !strings.HasPrefix(r.URL.Path, "/api/v5/repos/"+owner+"/demo/") {
						t.Fatalf("path=%s", r.URL.Path)
					}
					return 403, `{"message":"forbidden"}`
				})
				f.RepositoryResolver = func() (cmdutil.Repository, error) {
					if explicit {
						t.Fatal("inferred explicit repository")
					}
					return cmdutil.Repository{Owner: "inferred", Name: "demo"}, nil
				}
				args := []string{action}
				if explicit {
					args = append(args, "explicit/demo")
				}
				if action != "list" {
					args = append(args, "--yes")
				}
				if action == "set" {
					args = append(args, "--cla-id", "id")
				}
				out, _, err := executeCLA(t, f, "", args...)
				if err == nil || out != "" || !strings.Contains(err.Error(), "403") {
					t.Fatalf("error=%v out=%s", err, out)
				}
			})
		}
	}
}
