package org

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func executeOrgDiscussionWrite(t *testing.T, f *cmdutil.Factory, args []string, input string) (string, string, error) {
	t.Helper()
	cmd := NewCmdOrg(f)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, prompt bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&prompt)
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(append([]string{"discussion"}, args...))
	err := cmd.Execute()
	return out.String(), prompt.String(), err
}

func TestOrgDiscussionWriteRoutes(t *testing.T) {
	cmd := NewCmdOrg(&cmdutil.Factory{})
	for _, route := range [][]string{{"create"}, {"edit"}, {"delete"}, {"comment", "create"}, {"comment", "reply"}, {"comment", "edit"}, {"comment", "delete"}} {
		child, _, err := cmd.Find(append([]string{"discussion"}, route...))
		if err != nil || child.Name() != route[len(route)-1] {
			t.Fatalf("route %v: %v", route, err)
		}
	}
}

func TestOrgDiscussionCreateBodySources(t *testing.T) {
	const body = "# 中文\n\n**Markdown** & `code` \\path\n\n"
	file := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(file, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		flags []string
	}{
		{"literal", []string{"--body", body}}, {"file", []string{"--body-file", file}}, {"stdin", []string{"--body-file", "-"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.Path != "/api/v5/orgs/my-org/discuss" || r.Header.Get("Authorization") != "Bearer token" || r.Header.Get("Content-Type") != "application/json" {
					t.Fatalf("request = %s %s", r.Method, r.URL)
				}
				var fields map[string]string
				if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(fields, map[string]string{"title": "计划", "category_name": "Ideas", "md_content": body}) {
					t.Fatalf("fields = %#v", fields)
				}
				return orgResponse(200, `{"id":"d7","number":7}`), nil
			})
			f.RepositoryResolver = func() (cmdutil.Repository, error) {
				t.Fatal("organization must be explicit")
				return cmdutil.Repository{}, nil
			}
			args := append([]string{"create", "my-org", "--title", "计划", "--category", "Ideas"}, tc.flags...)
			out, _, err := executeOrgDiscussionWrite(t, f, args, body)
			if err != nil || calls != 1 || out != "Created discussion #7 for organization my-org\n" {
				t.Fatalf("out=%q err=%v calls=%d", out, err, calls)
			}
		})
	}
}

func TestOrgDiscussionEditPreservesUnspecifiedFields(t *testing.T) {
	emptyFile := filepath.Join(t.TempDir(), "empty.md")
	if err := os.WriteFile(emptyFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		flags []string
		body  string
		want  api.DiscussionEditRequest
	}{
		{"title", []string{"--title", "New"}, "", api.DiscussionEditRequest{Title: "New", MDContent: "Old body", CategoryName: "Ideas"}},
		{"category", []string{"--category", "Questions"}, "", api.DiscussionEditRequest{Title: "Old title", MDContent: "Old body", CategoryName: "Questions"}},
		{"body", []string{"--body", " New body\n"}, "", api.DiscussionEditRequest{Title: "Old title", MDContent: " New body\n", CategoryName: "Ideas"}},
		{"empty literal", []string{"--body="}, "", api.DiscussionEditRequest{Title: "Old title", CategoryName: "Ideas"}},
		{"empty stdin", []string{"--body-file=-"}, "", api.DiscussionEditRequest{Title: "Old title", CategoryName: "Ideas"}},
		{"empty file", []string{"--body-file", emptyFile}, "", api.DiscussionEditRequest{Title: "Old title", CategoryName: "Ideas"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/api/v5/orgs/my-org/discuss/7" {
					t.Fatalf("path = %s", r.URL)
				}
				if calls == 1 {
					if r.Method != "GET" {
						t.Fatalf("method = %s", r.Method)
					}
					return orgResponse(200, `{"id":"d7","number":7,"title":"Old title","md_content":"Old body","category":{"name":"Ideas"}}`), nil
				}
				if calls != 2 || r.Method != "PUT" {
					t.Fatalf("unexpected request %d: %s", calls, r.Method)
				}
				var got api.DiscussionEditRequest
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if got != tc.want {
					t.Fatalf("request = %#v, want %#v", got, tc.want)
				}
				raw, _ := json.Marshal(map[string]any{"id": "d7", "number": 7, "title": got.Title, "md_content": got.MDContent, "category": map[string]string{"name": got.CategoryName}})
				return orgResponse(200, string(raw)), nil
			})
			out, _, err := executeOrgDiscussionWrite(t, f, append([]string{"edit", "my-org", "7"}, tc.flags...), tc.body)
			if err != nil || calls != 2 || out != "Updated discussion #7 for organization my-org\n" {
				t.Fatalf("out=%q err=%v calls=%d", out, err, calls)
			}
		})
	}
}

func TestOrgDiscussionEditPrereadFailurePreventsWrite(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"server failure", `{"message":"denied"}`, 403},
		{"missing id", `{"number":7,"title":"Old","md_content":"body","category":{"name":"Ideas"}}`, 200},
		{"blank id", `{"id":"  ","number":7,"title":"Old","md_content":"body","category":{"name":"Ideas"}}`, 200},
		{"wrong number", `{"id":"d8","number":8,"title":"Old","md_content":"body","category":{"name":"Ideas"}}`, 200},
		{"missing body", `{"id":"d7","number":7,"title":"Old","category":{"name":"Ideas"}}`, 200},
		{"missing title", `{"id":"d7","number":7,"md_content":"body","category":{"name":"Ideas"}}`, 200},
		{"missing category", `{"id":"d7","number":7,"title":"Old","md_content":"body"}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" {
					t.Fatal("sent write after failed preread")
				}
				return orgResponse(tc.status, tc.body), nil
			})
			out, _, err := executeOrgDiscussionWrite(t, f, []string{"edit", "my-org", "7", "--title", "New"}, "")
			if err == nil || calls != 1 || out != "" {
				t.Fatalf("out=%q err=%v calls=%d", out, err, calls)
			}
		})
	}
}

func TestOrgDiscussionEditVerifiesReadback(t *testing.T) {
	const initial = `{"id":"d7","number":7,"title":"Old","md_content":"body","category":{"name":"Ideas"}}`
	for _, tc := range []struct {
		name, readback string
		status         int
		wantErr        bool
	}{
		{"verified", `{"id":"d7","number":7,"title":"New","md_content":"body","category":{"name":"Ideas"}}`, 200, false},
		{"mismatch", initial, 200, true},
		{"read failed", `{"message":"denied"}`, 403, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
				calls++
				switch calls {
				case 1:
					return orgResponse(200, initial), nil
				case 2:
					if r.Method != "PUT" {
						t.Fatalf("method = %s", r.Method)
					}
					return orgResponse(200, `{}`), nil
				case 3:
					if r.Method != "GET" {
						t.Fatalf("method = %s", r.Method)
					}
					return orgResponse(tc.status, tc.readback), nil
				default:
					t.Fatal("unexpected retry")
					return nil, nil
				}
			})
			out, _, err := executeOrgDiscussionWrite(t, f, []string{"edit", "my-org", "7", "--title", "New"}, "")
			if (err != nil) != tc.wantErr || calls != 3 || (tc.wantErr && out != "") {
				t.Fatalf("out=%q err=%v calls=%d", out, err, calls)
			}
		})
	}
}

func TestOrgDiscussionWritesValidateBeforeAuthentication(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(empty, nil, 0600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		args []string
	}{
		{"missing organization", []string{"create", "--title=x", "--category=Ideas", "--body=x"}},
		{"extran organization", []string{"create", "my-org", "extra", "--title=x", "--category=Ideas", "--body=x"}},
		{"no title", []string{"create", "my-org", "--category=Ideas", "--body=x"}},
		{"blank title", []string{"create", "my-org", "--title= \t", "--category=Ideas", "--body=x"}},
		{"blank category", []string{"create", "my-org", "--title=x", "--category= \n", "--body=x"}},
		{"no body", []string{"create", "my-org", "--title=x", "--category=Ideas"}},
		{"blank body", []string{"create", "my-org", "--title=x", "--category=Ideas", "--body= \n"}},
		{"empty file", []string{"create", "my-org", "--title=x", "--category=Ideas", "--body-file", empty}},
		{"empty stdin", []string{"create", "my-org", "--title=x", "--category=Ideas", "--body-file=-"}},
		{"body conflict", []string{"create", "my-org", "--title=x", "--category=Ideas", "--body=x", "--body-file=-"}},
		{"missing file", []string{"create", "my-org", "--title=x", "--category=Ideas", "--body-file", empty + "-missing"}},
		{"edit no fields", []string{"edit", "my-org", "7"}},
		{"edit blank title", []string{"edit", "my-org", "7", "--title= \t"}},
		{"edit blank category", []string{"edit", "my-org", "7", "--category= \n"}},
		{"edit body conflict", []string{"edit", "my-org", "7", "--body=x", "--body-file=-"}},
	}
	for _, org := range []string{"", ".", "..", "a/b", "a?b", "a#b", "a%b", "a\\b", " a", "a\n"} {
		cases = append(cases, struct {
			name string
			args []string
		}{"invalid org " + fmt.Sprintf("%q", org), []string{"delete", org, "7", "--yes"}})
	}
	for _, num := range []string{"0", "x", "../1", "999999999999999999999"} {
		cases = append(cases, struct {
			name string
			args []string
		}{"invalid number " + num, []string{"delete", "my-org", num, "--yes"}})
	}
	for _, op := range []string{"create", "reply", "edit"} {
		base := []string{"comment", op, "my-org", "7"}
		if op != "create" {
			base = append(base, "c1")
		}
		for _, flags := range [][]string{nil, {"--body="}, {"--body= \n"}, {"--body-file", empty}, {"--body-file=-"}, {"--body=x", "--body-file=-"}} {
			cases = append(cases, struct {
				name string
				args []string
			}{"comment " + op + " " + strings.Join(flags, " "), append(append([]string{}, base...), flags...)})
		}
	}
	for _, id := range []string{"", ".", "..", "a/b", "a?b", "a#b", "a%b", " a", "a\n"} {
		cases = append(cases, struct {
			name string
			args []string
		}{"invalid comment " + fmt.Sprintf("%q", id), []string{"comment", "delete", "my-org", "7", id, "--yes"}})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authCalls := 0
			f := orgFactory(orgTestConfig{tokenCalls: &authCalls}, func(*http.Request) (*http.Response, error) {
				t.Fatal("sent request with invalid arguments")
				return nil, nil
			})
			out, _, err := executeOrgDiscussionWrite(t, f, tc.args, "")
			if err == nil || authCalls != 0 || out != "" {
				t.Fatalf("out=%q err=%v authentication calls=%d", out, err, authCalls)
			}
		})
	}
}

func TestOrgDiscussionDeleteConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name, input  string
		yes, deleted bool
	}{{"yes", "yes\n", false, true}, {"y", "Y\n", false, true}, {"no", "n\n", false, false}, {"empty", "\n", false, false}, {"EOF", "", false, false}, {"flag", "", true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "DELETE" || r.URL.Path != "/api/v5/orgs/my-org/discuss/7" {
					t.Fatalf("request = %s %s", r.Method, r.URL)
				}
				return orgResponse(200, ""), nil
			})
			args := []string{"delete", "my-org", "7"}
			if tc.yes {
				args = append(args, "--yes")
			}
			out, prompt, err := executeOrgDiscussionWrite(t, f, args, tc.input)
			if err != nil || (calls == 1) != tc.deleted || (tc.deleted && !strings.Contains(out, "Deleted discussion #7")) || (!tc.deleted && !strings.Contains(out, "cancelled")) || (tc.yes && prompt != "") {
				t.Fatalf("out=%q prompt=%q err=%v calls=%d", out, prompt, err, calls)
			}
		})
	}
}

func orgDiscussionCommentTransport(t *testing.T, write func(*http.Request) (*http.Response, error)) orgRoundTripFunc {
	t.Helper()
	return func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" {
			return write(r)
		}
		switch r.URL.Path {
		case "/api/v5/orgs/my-org/discuss/7":
			return orgResponse(200, `{"id":"d7","number":7}`), nil
		case "/api/v5/orgs/my-org/discuss/7/comment":
			return orgResponse(200, `[{"id":"c1"}]`), nil
		default:
			t.Fatalf("unexpected GET %s", r.URL)
			return nil, nil
		}
	}
}

func TestOrgDiscussionCommentBodySources(t *testing.T) {
	const body = " **中文**\n\n"
	file := filepath.Join(t.TempDir(), "comment.md")
	if err := os.WriteFile(file, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"create", "reply", "edit"} {
		for _, source := range []struct {
			name  string
			flags []string
		}{{"literal", []string{"--body", body}}, {"file", []string{"--body-file", file}}, {"stdin", []string{"--body-file=-"}}} {
			t.Run(op+"/"+source.name, func(t *testing.T) {
				writes := 0
				f := orgFactory(orgTestConfig{}, orgDiscussionCommentTransport(t, func(r *http.Request) (*http.Response, error) {
					writes++
					path, method := "/api/v5/orgs/my-org/discuss/7/comment", "POST"
					if op == "reply" {
						path += "/c1/reply"
					}
					if op == "edit" {
						path += "/c1"
						method = "PUT"
					}
					if r.Method != method || r.URL.Path != path {
						t.Fatalf("request = %s %s", r.Method, r.URL)
					}
					var fields map[string]string
					if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(fields, map[string]string{"md_content": body}) {
						t.Fatalf("fields = %#v", fields)
					}
					return orgResponse(200, `{"id":"c1"}`), nil
				}))
				args := []string{"comment", op, "my-org", "7"}
				if op != "create" {
					args = append(args, "c1")
				}
				args = append(args, source.flags...)
				out, _, err := executeOrgDiscussionWrite(t, f, args, body)
				if err != nil || writes != 1 || !strings.Contains(out, "#c1") {
					t.Fatalf("out=%q err=%v writes=%d", out, err, writes)
				}
			})
		}
	}
}

func TestOrgDiscussionCommentDeleteConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name, input  string
		yes, deleted bool
	}{{"confirmed", "yes\n", false, true}, {"cancelled", "n\n", false, false}, {"EOF", "", false, false}, {"yes flag", "", true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			f := orgFactory(orgTestConfig{}, orgDiscussionCommentTransport(t, func(r *http.Request) (*http.Response, error) {
				writes++
				if r.Method != "DELETE" || r.URL.Path != "/api/v5/orgs/my-org/discuss/7/comment/c1" {
					t.Fatalf("request = %s %s", r.Method, r.URL)
				}
				return orgResponse(200, ""), nil
			}))
			args := []string{"comment", "delete", "my-org", "7", "c1"}
			if tc.yes {
				args = append(args, "--yes")
			}
			out, prompt, err := executeOrgDiscussionWrite(t, f, args, tc.input)
			if err != nil || (writes == 1) != tc.deleted || (tc.deleted && !strings.Contains(out, "Deleted comment #c1")) || (!tc.deleted && !strings.Contains(out, "cancelled")) || (tc.yes && prompt != "") {
				t.Fatalf("out=%q prompt=%q err=%v writes=%d", out, prompt, err, writes)
			}
			if !tc.yes && !strings.Contains(prompt, "Delete comment c1 on discussion #7 for organization my-org?") {
				t.Fatalf("confirmation omits target context: %q", prompt)
			}
		})
	}
}

func TestOrgDiscussionCommentEditAndDeleteReplies(t *testing.T) {
	for _, op := range []string{"edit", "delete"} {
		t.Run(op, func(t *testing.T) {
			writes := 0
			f := orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					writes++
					method := "PUT"
					if op == "delete" {
						method = "DELETE"
					}
					if r.Method != method || r.URL.Path != "/api/v5/orgs/my-org/discuss/7/comment/reply.id" {
						t.Fatalf("request = %s %s", r.Method, r.URL)
					}
					if op == "delete" {
						return orgResponse(200, ""), nil
					}
					return orgResponse(200, `{"id":"reply.id"}`), nil
				}
				switch r.URL.Path {
				case "/api/v5/orgs/my-org/discuss/7":
					return orgResponse(200, `{"id":"d7","number":7}`), nil
				case "/api/v5/orgs/my-org/discuss/7/comment":
					return orgResponse(200, `[{"id":"top"}]`), nil
				case "/api/v5/orgs/my-org/discuss/7/comment/top/reply":
					return orgResponse(200, `[{"id":"reply.id"}]`), nil
				default:
					t.Fatalf("unexpected request %s", r.URL)
					return nil, nil
				}
			})
			args := []string{"comment", op, "my-org", "7", "reply.id"}
			if op == "delete" {
				args = append(args, "--yes")
			} else {
				args = append(args, "--body=Updated")
			}
			out, _, err := executeOrgDiscussionWrite(t, f, args, "")
			if err != nil || writes != 1 || !strings.Contains(out, "#reply.id") {
				t.Fatalf("out=%q err=%v writes=%d", out, err, writes)
			}
		})
	}
}

func TestOrgDiscussionCommentMembershipPreventsWrites(t *testing.T) {
	for _, op := range []string{"reply", "edit", "delete"} {
		for _, failure := range []string{"unknown", "wrong discussion", "read failure"} {
			t.Run(op+"/"+failure, func(t *testing.T) {
				f := orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
					if r.Method != "GET" {
						t.Fatalf("unconfirmed comment write: %s %s", r.Method, r.URL)
					}
					if failure == "read failure" {
						return orgResponse(403, `{"message":"denied"}`), nil
					}
					if r.URL.Path == "/api/v5/orgs/my-org/discuss/7" {
						if failure == "wrong discussion" {
							return orgResponse(200, `{"id":"d8","number":8}`), nil
						}
						return orgResponse(200, `{"id":"d7","number":7}`), nil
					}
					return orgResponse(200, `[]`), nil
				})
				args := []string{"comment", op, "my-org", "7", "c1"}
				if op == "delete" {
					args = append(args, "--yes")
				} else {
					args = append(args, "--body=x")
				}
				out, _, err := executeOrgDiscussionWrite(t, f, args, "")
				if err == nil || out != "" {
					t.Fatalf("out=%q err=%v", out, err)
				}
			})
		}
	}
}

func TestOrgDiscussionCommentReplyRejectsReplyParent(t *testing.T) {
	f := orgFactory(orgTestConfig{}, func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" {
			t.Fatal("posted nested reply")
		}
		switch r.URL.Path {
		case "/api/v5/orgs/my-org/discuss/7":
			return orgResponse(200, `{"id":"d7","number":7}`), nil
		case "/api/v5/orgs/my-org/discuss/7/comment":
			return orgResponse(200, `[{"id":"top"}]`), nil
		case "/api/v5/orgs/my-org/discuss/7/comment/top/reply":
			return orgResponse(200, `[{"id":"reply"}]`), nil
		default:
			t.Fatalf("unexpected request %s", r.URL)
			return nil, nil
		}
	})
	out, _, err := executeOrgDiscussionWrite(t, f, []string{"comment", "reply", "my-org", "7", "reply", "--body=x"}, "")
	if err == nil || !strings.Contains(err.Error(), "top-level") || out != "" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestOrgDiscussionWritesRejectMalformedSuccess(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		response string
	}{
		{"create missing number", []string{"create", "my-org", "--title=x", "--category=Ideas", "--body=x"}, `{}`},
		{"create missing id", []string{"create", "my-org", "--title=x", "--category=Ideas", "--body=x"}, `{"number":7}`},
		{"create blank id", []string{"create", "my-org", "--title=x", "--category=Ideas", "--body=x"}, `{"id":"  ","number":7}`},
		{"create zero number", []string{"create", "my-org", "--title=x", "--category=Ideas", "--body=x"}, `{"number":0}`},
		{"comment create missing id", []string{"comment", "create", "my-org", "7", "--body=x"}, `{}`},
		{"comment create invalid id", []string{"comment", "create", "my-org", "7", "--body=x"}, `{"id":"../bad"}`},
		{"comment reply missing id", []string{"comment", "reply", "my-org", "7", "c1", "--body=x"}, `{}`},
		{"comment edit wrong id", []string{"comment", "edit", "my-org", "7", "c1", "--body=x"}, `{"id":"other"}`},
		{"comment edit missing id", []string{"comment", "edit", "my-org", "7", "c1", "--body=x"}, `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			f := orgFactory(orgTestConfig{}, orgDiscussionCommentTransport(t, func(*http.Request) (*http.Response, error) { writes++; return orgResponse(200, tc.response), nil }))
			out, _, err := executeOrgDiscussionWrite(t, f, tc.args, "")
			if err == nil || out != "" || writes != 1 {
				t.Fatalf("out=%q err=%v writes=%d", out, err, writes)
			}
		})
	}
}

func TestOrgDiscussionWritesRequireAuthentication(t *testing.T) {
	for _, args := range [][]string{
		{"create", "my-org", "--title=x", "--category=Ideas", "--body=x"}, {"edit", "my-org", "7", "--title=x"}, {"delete", "my-org", "7", "--yes"},
		{"comment", "create", "my-org", "7", "--body=x"}, {"comment", "reply", "my-org", "7", "c1", "--body=x"}, {"comment", "edit", "my-org", "7", "c1", "--body=x"}, {"comment", "delete", "my-org", "7", "c1", "--yes"},
	} {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			f := orgFactory(orgTestConfig{tokenErr: errors.New("missing test credentials")}, func(*http.Request) (*http.Response, error) {
				t.Fatal("request sent without authentication")
				return nil, nil
			})
			out, _, err := executeOrgDiscussionWrite(t, f, args, "")
			if err == nil || out != "" {
				t.Fatalf("out=%q err=%v", out, err)
			}
		})
	}
}
