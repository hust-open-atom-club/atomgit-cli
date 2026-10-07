package issue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

var inspectionCases = []struct {
	name, path, body string
	newCmd           func(*cmdutil.Factory) *cobra.Command
	keys             []string
	text             []string
}{
	{"activity", "/api/v5/repos/alice/issues/7/operate_logs", `[{"id":1,"user":{"id":"opaque","login":"alice","name":"Alice"},"content":"changed milestone","created_at":"2026-09-17","update_at":"2026-09-18","action_type":"milestone","issue_id":"42","title":null}]`, newCmdIssueActivity,
		[]string{"id", "author", "content", "createdAt", "updatedAt", "action", "issueId", "title", "body", "head", "base"}, []string{"alice", "milestone", "#7", "2026-09-17", "changed milestone"}},
	{"history", "/api/v5/repos/alice/demo/issues/7/modify_history", `[{"id":"history-a","created_at":"2026-09-17","updated_at":"2026-09-18","created":false,"deleted":false,"content":"new body 中文","user":{"login":"alice"},"updated_user":{"login":"bob"}}]`, newCmdIssueHistory,
		[]string{"id", "createdAt", "updatedAt", "created", "deleted", "content", "author", "updatedBy"}, []string{"bob", "updated", "#7", "2026-09-18", "new body 中文"}},
	{"reactions", "/api/v5/repos/alice/demo/issues/7/user_reactions", `[{"id":"reaction-a","emoji":"👍","emoji_name":"like","user":{"login":"alice","name":"Alice","object_id":"user-a"}}]`, newCmdIssueReactions,
		[]string{"id", "emoji", "emojiName", "author"}, []string{"alice", "like", "👍", "#7"}},
}

func TestIssueInspectionOutput(t *testing.T) {
	for _, tc := range inspectionCases {
		for _, asJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%v", tc.name, asJSON), func(t *testing.T) {
				calls := 0
				f := &cmdutil.Factory{Config: issueTestConfig{}, HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						calls++
						if req.Method != "GET" || req.URL.Path != tc.path {
							t.Fatalf("unexpected request %s %s", req.Method, req.URL)
						}
						q := req.URL.Query()
						if tc.name == "activity" && q.Get("repo") != "demo" {
							t.Fatal("missing repo query")
						}
						if tc.name == "reactions" {
							if q.Get("page") != "1" || q.Get("per_page") != "100" {
								t.Fatalf("query %v", q)
							}
						} else if q.Has("page") || q.Has("per_page") {
							t.Fatalf("invented pagination: %v", q)
						}
						return issueResponse(200, tc.body), nil
					})}, nil
				}}
				cmd := tc.newCmd(f)
				cmd.SetArgs([]string{"alice/demo", "0007"})
				if asJSON {
					if err := cmd.Flags().Set("json", "true"); err != nil {
						t.Fatal(err)
					}
				}
				var out bytes.Buffer
				cmd.SetOut(&out)
				if err := cmd.Execute(); err != nil {
					t.Fatal(err)
				}
				if calls != 1 {
					t.Fatalf("calls=%d", calls)
				}
				if asJSON {
					var rows []map[string]json.RawMessage
					if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
						t.Fatal(err)
					}
					if len(rows) != 1 || len(rows[0]) != len(tc.keys) {
						t.Fatalf("schema %s", out.String())
					}
					for _, key := range tc.keys {
						if _, ok := rows[0][key]; !ok {
							t.Errorf("missing %s", key)
						}
					}
					var actor string
					if err := json.Unmarshal(rows[0]["author"], &actor); err != nil {
						t.Fatal(err)
					}
					if actor != "alice" {
						t.Fatalf("actor %v", actor)
					}
				} else {
					for _, want := range tc.text {
						if !strings.Contains(out.String(), want) {
							t.Errorf("output %q lacks %q", out.String(), want)
						}
					}
				}
			})
		}
	}
}

func TestIssueInspectionEmptyAndErrors(t *testing.T) {
	for _, tc := range inspectionCases {
		for _, test := range []struct {
			name, body string
			status     int
			wantErr    bool
		}{
			{"empty", "[]", 200, false}, {"null", "null", 200, false},
			{"null element", "[null]", 200, true},
			{"empty element", "[{}]", 200, true},
			{"missing ID", `[{"user":{"login":"alice"},"content":"unexpected"}]`, 200, true},
			{"empty body", "", 200, true}, {"object", "{}", 200, true}, {"malformed", "[", 200, true}, {"wrong user type", `[{"user":true}]`, 200, true},
			{"permission", `{"message":"denied"}`, 403, true}, {"not found", `{}`, 404, true},
		} {
			for _, asJSON := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/%v", tc.name, test.name, asJSON), func(t *testing.T) {
					f := &cmdutil.Factory{Config: issueTestConfig{}, HttpClient: func() (*http.Client, error) {
						return &http.Client{Transport: issueRoundTripFunc(func(*http.Request) (*http.Response, error) { return issueResponse(test.status, test.body), nil })}, nil
					}}
					cmd := tc.newCmd(f)
					cmd.SetArgs([]string{"alice/demo", "7"})
					if asJSON {
						if err := cmd.Flags().Set("json", "true"); err != nil {
							t.Fatal(err)
						}
					}
					var out bytes.Buffer
					cmd.SetOut(&out)
					cmd.SetErr(&bytes.Buffer{})
					cmd.SilenceUsage = true
					cmd.SilenceErrors = true
					err := cmd.Execute()
					if test.wantErr {
						if err == nil || !strings.Contains(err.Error(), "failed to list issue #7 "+tc.name) {
							t.Fatalf("error %v", err)
						}
						if out.Len() != 0 {
							t.Fatal("partial output")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if asJSON {
						if strings.TrimSpace(out.String()) != "[]" {
							t.Fatalf("output %q", out.String())
						}
					} else if !strings.Contains(out.String(), "No "+tc.name+" found") {
						t.Fatalf("output %q", out.String())
					}
				})
			}
		}
	}
}

func TestIssueInspectionValidationBeforeAuthentication(t *testing.T) {
	for _, tc := range inspectionCases {
		for _, args := range [][]string{{"alice/demo", "0"}, {"alice/demo", "-1"}, {"alice/demo", "abc"}, {"alice/demo", "7", "--limit", "0"}, {"alice/demo", "7", "--limit", "-1"}, {"alice/demo", "7", "extra"}, {}} {
			t.Run(tc.name+strings.Join(args, "/"), func(t *testing.T) {
				cmd := tc.newCmd(&cmdutil.Factory{}) // A credential access would panic.
				cmd.SetArgs(args)
				cmd.SetOut(&bytes.Buffer{})
				cmd.SetErr(&bytes.Buffer{})
				if err := cmd.Execute(); err == nil {
					t.Fatal("invalid input accepted")
				}
			})
		}
	}
}

func TestIssueInspectionLimits(t *testing.T) {
	for _, tc := range inspectionCases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := &cmdutil.Factory{Config: issueTestConfig{}, HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if tc.name == "reactions" && req.URL.Query().Get("page") != fmt.Sprint(calls) {
						t.Fatalf("page %s", req.URL.RawQuery)
					}
					var original []map[string]any
					if err := json.Unmarshal([]byte(tc.body), &original); err != nil {
						t.Fatalf("decode inspection fixture: %v", err)
					}
					var rows []map[string]any
					count := 103
					if tc.name == "reactions" {
						count = 100
						if calls == 2 {
							count = 3
						}
					}
					for i := 0; i < count; i++ {
						item := map[string]any{}
						maps.Copy(item, original[0])
						id := (calls-1)*100 + i + 1
						if tc.name == "activity" {
							item["id"] = id
						} else {
							item["id"] = fmt.Sprint(id)
						}
						rows = append(rows, item)
					}
					data, _ := json.Marshal(rows)
					return issueResponse(200, string(data)), nil
				})}, nil
			}}
			cmd := tc.newCmd(f)
			cmd.SetArgs([]string{"alice/demo", "7", "--limit", "101", "--json"})
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var rows []map[string]any
			if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
				t.Fatal(err)
			}
			if len(rows) != 101 {
				t.Fatalf("rows %d", len(rows))
			}
			want := 1
			if tc.name == "reactions" {
				want = 2
			}
			if calls != want {
				t.Fatalf("calls %d", calls)
			}
		})
	}
}

func TestIssueInspectionSanitizesText(t *testing.T) {
	for _, tc := range inspectionCases {
		t.Run(tc.name, func(t *testing.T) {
			actor := "alice"
			if tc.name == "history" {
				actor = "bob"
			}
			body := strings.ReplaceAll(tc.body, actor, actor+`\u001b[31m`)
			f := &cmdutil.Factory{Config: issueTestConfig{}, HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: issueRoundTripFunc(func(*http.Request) (*http.Response, error) { return issueResponse(200, body), nil })}, nil
			}}
			cmd := tc.newCmd(f)
			cmd.SetArgs([]string{"alice/demo", "7"})
			var out bytes.Buffer
			cmd.SetOut(cmdutil.NewSanitizingWriter(&out))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "\x1b") || !strings.Contains(out.String(), actor) {
				t.Fatalf("output %q", out.String())
			}
		})
	}
}

func TestIssueInspectionRepositoryContext(t *testing.T) {
	for _, tc := range inspectionCases {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/explicit=%v", tc.name, explicit), func(t *testing.T) {
				resolved := 0
				f := &cmdutil.Factory{Config: issueTestConfig{}, RepositoryResolver: func() (cmdutil.Repository, error) {
					resolved++
					return cmdutil.Repository{Owner: "alice", Name: "demo"}, nil
				}, HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						if req.URL.Path != tc.path {
							t.Fatalf("path %s", req.URL.Path)
						}
						return issueResponse(200, "[]"), nil
					})}, nil
				}}
				cmd := tc.newCmd(f)
				args := []string{"7"}
				if explicit {
					args = append([]string{"alice/demo"}, args...)
				}
				cmd.SetArgs(args)
				cmd.SetOut(&bytes.Buffer{})
				if err := cmd.Execute(); err != nil {
					t.Fatal(err)
				}
				want := 1
				if explicit {
					want = 0
				}
				if resolved != want {
					t.Fatalf("resolver calls %d", resolved)
				}
			})
		}
	}
}

func TestIssueHistoryActions(t *testing.T) {
	for _, tc := range []struct{ flags, action string }{{`"created":true`, "created"}, {`"created":false`, "updated"}, {`"created":true,"deleted":true`, "deleted"}} {
		t.Run(tc.action, func(t *testing.T) {
			f := &cmdutil.Factory{Config: issueTestConfig{}, HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: issueRoundTripFunc(func(*http.Request) (*http.Response, error) {
					return issueResponse(200, `[{"id":"h","user":{"name":"Display Name"},"created_at":"2026-09-17",`+tc.flags+`}]`), nil
				})}, nil
			}}
			cmd := newCmdIssueHistory(f)
			cmd.SetArgs([]string{"alice/demo", "7"})
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{tc.action, "Display Name", "2026-09-17"} {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("output %q lacks %q", out.String(), want)
				}
			}
		})
	}
}

func TestIssueActivityJSONPreservesAuditContext(t *testing.T) {
	for _, tc := range []struct{ name, fields, expected string }{
		{"linked PR", `"body":"PR body\n中文","head":{"ref":"feature","sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","repo":{"path":"demo-fork","name":"Fork"},"assigner":{"login":"alice","name":"Alice"}},"base":{"ref":"main","sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","repo":{"path":"demo","name":"Demo"},"assigner":null}`,
			`{"body":"PR body\n中文","head":{"ref":"feature","sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","repo":{"path":"demo-fork","name":"Fork"},"assigner":{"login":"alice","name":"Alice"}},"base":{"ref":"main","sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","repo":{"path":"demo","name":"Demo"},"assigner":null}}`},
		{"null context", `"body":null,"head":null,"base":null`, `{"body":"","head":null,"base":null}`},
		{"missing context", `"content":"milestone changed"`, `{"body":"","head":null,"base":null}`},
		{"partial branch", `"head":{"ref":"feature"}`, `{"body":"","head":{"ref":"feature","sha":"","repo":null,"assigner":null},"base":null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &cmdutil.Factory{Config: issueTestConfig{}, HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.Method != "GET" || req.URL.Path != "/api/v5/repos/alice/issues/7/operate_logs" || req.URL.Query().Get("repo") != "demo" {
						t.Fatalf("unexpected request %s %s", req.Method, req.URL)
					}
					return issueResponse(200, `[{"id":1,"action_type":"add_issue_mr_link",`+tc.fields+`}]`), nil
				})}, nil
			}}
			cmd := newCmdIssueActivity(f)
			cmd.SetArgs([]string{"alice/demo", "7", "--json"})
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var rows []map[string]any
			if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 {
				t.Fatalf("rows %d", len(rows))
			}
			var want map[string]any
			if err := json.Unmarshal([]byte(tc.expected), &want); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"body", "head", "base"} {
				value, present := rows[0][key]
				if !present || !reflect.DeepEqual(value, want[key]) {
					t.Errorf("%s = %#v, want %#v", key, value, want[key])
				}
			}
		})
	}
}
