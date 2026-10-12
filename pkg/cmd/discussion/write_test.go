package discussion

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func runWriteCommand(t *testing.T, f *cmdutil.Factory, args []string, in io.Reader) (string, string, error) {
	t.Helper()
	cmd := NewCmdDiscussion(f)
	var out, prompt bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&prompt)
	cmd.SetIn(in)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), prompt.String(), err
}

func TestDiscussionCreateBodySourcesAndRepository(t *testing.T) {
	const body = "# 中文\n\n**Markdown** & `code` \\path\n\n"
	file := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(file, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		flags []string
		infer bool
	}{
		{"flag", []string{"--body", body}, false},
		{"file", []string{"--body-file", file}, false},
		{"stdin", []string{"--body-file", "-"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, resolutions := 0, 0
			f := discussionFactory(discussionTestConfig{token: "fake-token"}, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.Path != "/api/v5/repos/owner/repo/discuss" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "Bearer fake-token" {
					t.Fatalf("request = %s %s", r.Method, r.URL)
				}
				var fields map[string]string
				if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(fields, map[string]string{"title": "计划", "category_name": "开放讨论", "md_content": body}) {
					t.Fatalf("fields = %#v", fields)
				}
				return discussionResponse(200, `{"number":42}`), nil
			})
			f.RepositoryResolver = func() (cmdutil.Repository, error) {
				resolutions++
				return cmdutil.Repository{Owner: "owner", Name: "repo"}, nil
			}
			args := []string{"create"}
			if !tc.infer {
				args = append(args, "owner/repo")
			}
			args = append(args, "--title", "计划", "--category", "开放讨论")
			args = append(args, tc.flags...)
			out, _, err := runWriteCommand(t, f, args, strings.NewReader(body))
			if err != nil || out != "Created discussion #42 for owner/repo\n" || calls != 1 || (resolutions == 1) != tc.infer {
				t.Fatalf("out=%q err=%v calls=%d resolutions=%d", out, err, calls, resolutions)
			}
		})
	}
}

func TestDiscussionWritesValidateBeforeAuth(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(empty, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no title", []string{"create", "o/r", "--category=Ideas", "--body=x"}, "title cannot"},
		{"blank title", []string{"create", "o/r", "--title= \t", "--category=Ideas", "--body=x"}, "title cannot"},
		{"no category", []string{"create", "o/r", "--title=x", "--body=x"}, "category cannot"},
		{"blank category", []string{"create", "o/r", "--title=x", "--category= \n", "--body=x"}, "category cannot"},
		{"no body", []string{"create", "o/r", "--title=x", "--category=Ideas"}, "at least one"},
		{"blank body", []string{"create", "o/r", "--title=x", "--category=Ideas", "--body= \n"}, "body cannot"},
		{"empty file", []string{"create", "o/r", "--title=x", "--category=Ideas", "--body-file", empty}, "body cannot"},
		{"empty stdin", []string{"create", "o/r", "--title=x", "--category=Ideas", "--body-file=-"}, "body cannot"},
		{"conflict", []string{"create", "o/r", "--title=x", "--category=Ideas", "--body=x", "--body-file=-"}, "none of the others"},
		{"missing file", []string{"create", "o/r", "--title=x", "--category=Ideas", "--body-file", filepath.Join(t.TempDir(), "missing")}, "failed to read body file"},
		{"bad repo", []string{"create", "o/r/x", "--title=x", "--category=Ideas", "--body=x"}, "invalid repository"},
		{"extra args", []string{"create", "o/r", "extra", "--title=x", "--category=Ideas", "--body=x"}, "accepts at most"},
		{"delete no number", []string{"delete"}, "accepts between"},
		{"delete zero", []string{"delete", "o/r", "0", "--yes"}, "positive integer"},
		{"delete bad number", []string{"delete", "o/r", "x", "--yes"}, "positive integer"},
		{"delete path", []string{"delete", "o/r", "../1", "--yes"}, "positive integer"},
		{"delete overflow", []string{"delete", "o/r", "999999999999999999999", "--yes"}, "positive integer"},
		{"delete negative", []string{"delete", "o/r", "--yes", "--", "-1"}, "positive integer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &cmdutil.Factory{Config: editForbiddenConfig{t: t}}
			out, _, err := runWriteCommand(t, f, tc.args, strings.NewReader(""))
			if err == nil || !strings.Contains(err.Error(), tc.want) || out != "" {
				t.Fatalf("out=%q err=%v want=%q", out, err, tc.want)
			}
		})
	}
}

func TestDiscussionDeleteConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name, input         string
		yes, infer, deleted bool
	}{
		{"yes", "yes\n", false, false, true}, {"y", "Y\n", false, false, true},
		{"no", "n\n", false, false, false}, {"empty", "\n", false, false, false}, {"EOF", "", false, false, false},
		{"flag", "", true, false, true}, {"inferred normalized number", "", true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := discussionFactory(discussionTestConfig{token: "fake-token"}, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "DELETE" || r.URL.Path != "/api/v5/repos/owner/repo/discuss/7" {
					t.Fatalf("request=%s %s", r.Method, r.URL)
				}
				return discussionResponse(200, ""), nil
			})
			f.RepositoryResolver = func() (cmdutil.Repository, error) {
				if !tc.infer {
					t.Fatal("explicit repo did not take precedence")
				}
				return cmdutil.Repository{Owner: "owner", Name: "repo"}, nil
			}
			args := []string{"delete"}
			if !tc.infer {
				args = append(args, "owner/repo")
			}
			args = append(args, "007")
			if tc.yes {
				args = append(args, "--yes")
			}
			out, prompt, err := runWriteCommand(t, f, args, strings.NewReader(tc.input))
			if err != nil || (calls == 1) != tc.deleted {
				t.Fatalf("out=%q prompt=%q err=%v calls=%d", out, prompt, err, calls)
			}
			if !tc.yes && !strings.Contains(prompt, "Permanently delete discussion #7 from owner/repo?") {
				t.Fatalf("prompt=%q", prompt)
			}
			want := "Deletion cancelled.\n"
			if tc.deleted {
				want = "Deleted discussion #7 for owner/repo\n"
			}
			if out != want || tc.yes && prompt != "" {
				t.Fatalf("out=%q prompt=%q", out, prompt)
			}
		})
	}
}

func TestDiscussionWritesRequireAuth(t *testing.T) {
	for _, args := range [][]string{{"create", "o/r", "--title=x", "--category=Ideas", "--body=x"}, {"delete", "o/r", "7", "--yes"}} {
		f := discussionFactory(discussionTestConfig{}, func(*http.Request) (*http.Response, error) { t.Fatal("unauthenticated request"); return nil, nil })
		out, _, err := runWriteCommand(t, f, args, strings.NewReader(""))
		if err == nil || out != "" {
			t.Fatalf("out=%q err=%v", out, err)
		}
	}
}

func TestDiscussionWritesErrors(t *testing.T) {
	for _, operation := range []string{"create", "delete"} {
		for _, status := range []int{400, 401, 403, 404, 429, 500, 503} {
			t.Run(operation+http.StatusText(status), func(t *testing.T) {
				calls := 0
				f := discussionFactory(discussionTestConfig{token: "fake-token"}, func(*http.Request) (*http.Response, error) {
					calls++
					return discussionResponse(status, `{"message":"Bearer fake-token\u001b[31m","refresh_token":"secret-refresh"}`), nil
				})
				args := []string{"create", "o/r", "--title=x", "--category=Ideas", "--body=x"}
				if operation == "delete" {
					args = []string{"delete", "o/r", "7", "--yes"}
				}
				out, _, err := runWriteCommand(t, f, args, strings.NewReader(""))
				if err == nil || !api.IsHTTPStatus(err, status) || calls != 1 || out != "" || !strings.Contains(err.Error(), "failed to "+operation+" discussion") {
					t.Fatalf("out=%q err=%v calls=%d", out, err, calls)
				}
				for _, unsafe := range []string{"fake-token", "secret-refresh", "\x1b"} {
					if strings.Contains(err.Error(), unsafe) {
						t.Fatalf("unsafe error: %v", err)
					}
				}
			})
		}
	}
}

func TestDiscussionCreateUncertainResponses(t *testing.T) {
	for _, body := range []string{"", `broken`, `null`, `{}`, `{"id":"d1"}`, `{"number":0}`, `{"number":-1}`, `{"number":"7"}`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			f := discussionFactory(discussionTestConfig{token: "fake-token"}, func(*http.Request) (*http.Response, error) { calls++; return discussionResponse(200, body), nil })
			out, _, err := runWriteCommand(t, f, []string{"create", "o/r", "--title=x", "--category=Ideas", "--body=x"}, strings.NewReader(""))
			if err == nil || out != "" || calls != 1 || !strings.Contains(err.Error(), "inspect the discussion list before retrying") {
				t.Fatalf("out=%q err=%v calls=%d", out, err, calls)
			}
		})
	}
}

func TestDiscussionWriteInputFailures(t *testing.T) {
	for _, args := range [][]string{{"create", "o/r", "--title=x", "--category=Ideas", "--body-file=-"}, {"delete", "o/r", "7"}} {
		f := &cmdutil.Factory{Config: editForbiddenConfig{t: t}}
		out, _, err := runWriteCommand(t, f, args, editErrorReader{})
		if err == nil || out != "" || !strings.Contains(err.Error(), "stdin unavailable") {
			t.Fatalf("out=%q err=%v", out, err)
		}
	}
	networkErr := errors.New("connection lost after write")
	for _, args := range [][]string{{"create", "o/r", "--title=x", "--category=Ideas", "--body=x"}, {"delete", "o/r", "7", "--yes"}} {
		calls := 0
		f := discussionFactory(discussionTestConfig{token: "fake-token"}, func(*http.Request) (*http.Response, error) { calls++; return nil, networkErr })
		out, _, err := runWriteCommand(t, f, args, strings.NewReader(""))
		if !errors.Is(err, networkErr) || out != "" || calls != 1 {
			t.Fatalf("out=%q err=%v calls=%d", out, err, calls)
		}
	}
}
