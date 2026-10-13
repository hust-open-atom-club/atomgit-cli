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
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func runEditCommand(t *testing.T, f *cmdutil.Factory, args []string, stdin io.Reader) (string, error) {
	t.Helper()
	cmd := NewCmdDiscussion(f)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	if stdin != nil {
		cmd.SetIn(stdin)
	}
	cmd.SetArgs(append([]string{"edit"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func editResponse(t *testing.T, title, body, category string) string {
	t.Helper()
	encoded, err := json.Marshal(map[string]any{
		"id": "d1", "number": 7, "title": title, "md_content": body,
		"category": map[string]string{"id": "c1", "name": category, "icon": "📣"},
		"is_lock":  1, "is_pin": 1, "is_closed": 1, "is_answered": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestDiscussionEditFieldsAndBodySources(t *testing.T) {
	const markdown = "# Heading\n\n\"quotes\" & `code` \\path\nUTF-8 正文\n\n"
	bodyFile := filepath.Join(t.TempDir(), "discussion.md")
	emptyFile := filepath.Join(t.TempDir(), "empty.md")
	if err := os.WriteFile(bodyFile, []byte(markdown), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(emptyFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                  string
		flags                 []string
		stdin                 string
		title, body, category string
		infer                 bool
	}{
		{"title", []string{"--title", "Updated"}, "", "Updated", "# Body\n- item", "公告", false},
		{"body", []string{"--body", markdown}, "", "Roadmap", markdown, "公告", false},
		{"category", []string{"--category", "Ideas"}, "", "Roadmap", "# Body\n- item", "Ideas", false},
		{"combined", []string{"--title", "Updated", "--body", markdown, "--category", "Ideas"}, "", "Updated", markdown, "Ideas", false},
		{"file", []string{"--body-file", bodyFile}, "", "Roadmap", markdown, "公告", false},
		{"stdin and repository inference", []string{"--body-file", "-"}, markdown, "Roadmap", markdown, "公告", true},
		{"empty body", []string{"--body="}, "", "Roadmap", "", "公告", false},
		{"empty file", []string{"--body-file", emptyFile}, "", "Roadmap", "", "公告", false},
		{"empty stdin", []string{"--body-file", "-"}, "", "Roadmap", "", "公告", false},
		{"whitespace body", []string{"--body", " \n\t"}, "", "Roadmap", " \n\t", "公告", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var methods []string
			f := discussionFactory(discussionTestConfig{token: "fake-token"}, func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method)
				if req.URL.Path != "/api/v5/repos/owner/repo/discuss/7" || req.Header.Get("Authorization") != "Bearer fake-token" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
				}
				if req.Method == http.MethodGet {
					return discussionResponse(http.StatusOK, viewDetailFixture), nil
				}
				if req.Method != http.MethodPut || req.Header.Get("Content-Type") != "application/json" {
					t.Fatalf("write = %s, type = %q", req.Method, req.Header.Get("Content-Type"))
				}
				var fields map[string]string
				if err := json.NewDecoder(req.Body).Decode(&fields); err != nil {
					t.Fatal(err)
				}
				want := map[string]string{"title": tc.title, "md_content": tc.body, "category_name": tc.category}
				if !reflect.DeepEqual(fields, want) {
					t.Fatalf("fields = %#v, want %#v", fields, want)
				}
				return discussionResponse(http.StatusOK, editResponse(t, tc.title, tc.body, tc.category)), nil
			})
			resolverCalled := false
			f.RepositoryResolver = func() (cmdutil.Repository, error) {
				resolverCalled = true
				return cmdutil.Repository{Owner: "owner", Name: "repo"}, nil
			}
			args := []string{"owner/repo", "7"}
			if tc.infer {
				args = []string{"7"}
			}
			out, err := runEditCommand(t, f, append(args, tc.flags...), strings.NewReader(tc.stdin))
			if err != nil || out != "Updated discussion #7 for owner/repo\n" {
				t.Fatalf("out = %q, err = %v", out, err)
			}
			if resolverCalled != tc.infer || strings.Join(methods, ",") != "GET,PUT" {
				t.Fatalf("resolver = %v, methods = %v", resolverCalled, methods)
			}
		})
	}
}

type editForbiddenConfig struct {
	discussionTestConfig
	t *testing.T
}

func (c editForbiddenConfig) GetToken() (string, error) {
	c.t.Fatal("invalid input reached authentication")
	return "", nil
}

type editErrorReader struct{}

func (editErrorReader) Read([]byte) (int, error) { return 0, errors.New("stdin unavailable") }

func TestDiscussionEditRejectsInvalidInputBeforeAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"missing number", nil, "accepts between"},
		{"too many arguments", []string{"owner/repo", "7", "extra", "--title", "x"}, "accepts between"},
		{"invalid repository", []string{"owner/repo/extra", "7", "--title", "x"}, "invalid repository"},
		{"invalid number", []string{"owner/repo", "abc", "--title", "x"}, "positive integer"},
		{"zero number", []string{"owner/repo", "0", "--title", "x"}, "positive integer"},
		{"negative number", []string{"owner/repo", "--title=x", "--", "-1"}, "positive integer"},
		{"overflow number", []string{"owner/repo", "999999999999999999999999", "--title", "x"}, "positive integer"},
		{"no edit", []string{"owner/repo", "7"}, "at least one"},
		{"empty title", []string{"owner/repo", "7", "--title="}, "title cannot be empty"},
		{"blank title", []string{"owner/repo", "7", "--title= \t"}, "title cannot be empty"},
		{"empty category", []string{"owner/repo", "7", "--category="}, "category cannot be empty"},
		{"blank category", []string{"owner/repo", "7", "--category= \t"}, "category cannot be empty"},
		{"conflicting bodies", []string{"owner/repo", "7", "--body=", "--body-file=-"}, "none of the others"},
		{"missing file", []string{"owner/repo", "7", "--body-file", filepath.Join(t.TempDir(), "missing")}, "failed to read body file"},
		{"empty file path", []string{"owner/repo", "7", "--body-file="}, "failed to read body file"},
		{"stdin failure", []string{"owner/repo", "7", "--body-file=-"}, "stdin unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &cmdutil.Factory{Config: editForbiddenConfig{t: t}, HttpClient: func() (*http.Client, error) {
				t.Fatal("invalid input created HTTP client")
				return nil, nil
			}}
			out, err := runEditCommand(t, f, tc.args, editErrorReader{})
			if err == nil || !strings.Contains(err.Error(), tc.want) || out != "" {
				t.Fatalf("out = %q, err = %v, want %q", out, err, tc.want)
			}
		})
	}
}

func TestDiscussionEditRequiresAuthentication(t *testing.T) {
	for _, cfg := range []discussionTestConfig{{}, {tokenErr: config.ErrNotAuthenticated}, {tokenErr: errors.New("credential store unavailable")}} {
		f := discussionFactory(cfg, func(*http.Request) (*http.Response, error) {
			t.Fatal("unauthenticated edit sent a request")
			return nil, nil
		})
		out, err := runEditCommand(t, f, []string{"owner/repo", "7", "--title=Updated"}, nil)
		if err == nil || out != "" {
			t.Fatalf("out = %q, err = %v", out, err)
		}
	}
}

func TestDiscussionEditReadFailurePreventsWrite(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"unauthorized", `{}`, http.StatusUnauthorized},
		{"forbidden", `{}`, http.StatusForbidden},
		{"missing", `{}`, http.StatusNotFound},
		{"server failure", `{}`, http.StatusInternalServerError},
		{"invalid JSON", `not JSON`, http.StatusOK},
		{"missing fields", `{}`, http.StatusOK},
		{"missing Markdown", `{"id":"d1","number":7,"title":"Roadmap","category":{"name":"Ideas"}}`, http.StatusOK},
		{"null Markdown", `{"id":"d1","number":7,"title":"Roadmap","md_content":null,"category":{"name":"Ideas"}}`, http.StatusOK},
		{"wrong number", strings.Replace(viewDetailFixture, `"number": 7`, `"number": 8`, 1), http.StatusOK},
		{"missing category", strings.Replace(viewDetailFixture, `"name": "公告"`, `"name": ""`, 1), http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := discussionFactory(discussionTestConfig{token: "fake-token"}, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet {
					t.Fatalf("read failure led to %s", req.Method)
				}
				return discussionResponse(tc.status, tc.body), nil
			})
			out, err := runEditCommand(t, f, []string{"owner/repo", "7", "--title=Updated"}, nil)
			if err == nil || out != "" || calls != 1 || !strings.Contains(err.Error(), "discussion #7 for owner/repo") {
				t.Fatalf("out = %q, err = %v, calls = %d", out, err, calls)
			}
		})
	}
}

func TestDiscussionEditWriteFailureDoesNotReportSuccessOrRetry(t *testing.T) {
	networkErr := errors.New("connection lost")
	for _, tc := range []struct {
		name   string
		status int
		body   string
		err    error
	}{
		{"unauthorized", 401, `{}`, nil},
		{"forbidden", 403, `{"message":"Bearer fake-token\u001b[31m","refresh_token":"hidden-refresh"}`, nil},
		{"missing", 404, `{}`, nil},
		{"rejected empty body", 400, `{"message":"body required"}`, nil},
		{"rate limit", 429, `{}`, nil},
		{"server error", 503, `{}`, nil},
		{"invalid JSON", 200, `not JSON`, nil},
		{"empty response", 200, ``, nil},
		{"network failure", 0, "", networkErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var methods []string
			f := discussionFactory(discussionTestConfig{token: "fake-token"}, func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method)
				if len(methods) == 1 {
					return discussionResponse(http.StatusOK, viewDetailFixture), nil
				}
				if tc.err != nil {
					return nil, tc.err
				}
				return discussionResponse(tc.status, tc.body), nil
			})
			out, err := runEditCommand(t, f, []string{"owner/repo", "7", "--body="}, nil)
			if err == nil || out != "" || strings.Join(methods, ",") != "GET,PUT" {
				t.Fatalf("out = %q, err = %v, methods = %v", out, err, methods)
			}
			if !strings.Contains(err.Error(), "failed to update discussion #7 for owner/repo") || !strings.Contains(err.Error(), "not retried") {
				t.Fatalf("missing error context: %v", err)
			}
			if tc.err != nil && !errors.Is(err, tc.err) || tc.status >= 400 && !api.IsHTTPStatus(err, tc.status) {
				t.Fatalf("underlying error lost: %v", err)
			}
			for _, secret := range []string{"fake-token", "hidden-refresh", "\x1b"} {
				if strings.Contains(out+err.Error(), secret) {
					t.Fatalf("unsafe output contains %q", secret)
				}
			}
		})
	}
}

func TestDiscussionEditVerifiesIncompleteOrMismatchedResponses(t *testing.T) {
	updated := editResponse(t, "Roadmap", "", "公告")
	missingBody := strings.Replace(updated, `"md_content":"",`, "", 1)
	for _, tc := range []struct {
		name, write, read string
		readStatus        int
		wantError         bool
	}{
		{"partial response", `{}`, updated, 200, false},
		{"null response", `null`, updated, 200, false},
		{"stale response", viewDetailFixture, updated, 200, false},
		{"omitted empty body requires readback", missingBody, updated, 200, false},
		{"unchanged after 200", viewDetailFixture, viewDetailFixture, 200, true},
		{"readback failure", `{}`, `{}`, 403, true},
		{"invalid readback", `{}`, `broken JSON`, 200, true},
		{"empty readback", `{}`, `{}`, 200, true},
		{"missing empty body", `{}`, missingBody, 200, true},
		{"wrong ID", `{}`, strings.Replace(updated, `"id":"d1"`, `"id":"d2"`, 1), 200, true},
		{"wrong number", `{}`, strings.Replace(updated, `"number":7`, `"number":8`, 1), 200, true},
		{"changed title", `{}`, strings.Replace(updated, `"Roadmap"`, `"Other"`, 1), 200, true},
		{"changed category", `{}`, strings.Replace(updated, `"公告"`, `"Ideas"`, 1), 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var methods []string
			f := discussionFactory(discussionTestConfig{token: "fake-token"}, func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method)
				switch len(methods) {
				case 1:
					return discussionResponse(http.StatusOK, viewDetailFixture), nil
				case 2:
					return discussionResponse(http.StatusOK, tc.write), nil
				default:
					return discussionResponse(tc.readStatus, tc.read), nil
				}
			})
			out, err := runEditCommand(t, f, []string{"owner/repo", "7", "--body="}, nil)
			if (err != nil) != tc.wantError || strings.Join(methods, ",") != "GET,PUT,GET" {
				t.Fatalf("out = %q, err = %v, methods = %v", out, err, methods)
			}
			if tc.wantError {
				if out != "" || !strings.Contains(err.Error(), "verify") || !strings.Contains(err.Error(), "inspect before retrying") {
					t.Fatalf("out = %q, err = %v", out, err)
				}
			} else if out != "Updated discussion #7 for owner/repo\n" {
				t.Fatalf("out = %q", out)
			}
		})
	}
}
