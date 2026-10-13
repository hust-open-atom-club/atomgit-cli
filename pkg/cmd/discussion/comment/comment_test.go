package comment

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type discussionCommentConfig struct {
	token    string
	tokenErr error
}

func (c discussionCommentConfig) GetToken() (string, error) { return c.token, c.tokenErr }
func (discussionCommentConfig) GetUser() (string, error)    { return "tester", nil }
func (discussionCommentConfig) GetHost() string             { return "atomgit.com" }

type discussionCommentTrip func(*http.Request) (*http.Response, error)

func (f discussionCommentTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func discussionCommentResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func discussionCommentFactory(config discussionCommentConfig, transport discussionCommentTrip) *cmdutil.Factory {
	return &cmdutil.Factory{
		Config: config,
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: transport}, nil
		},
	}
}

func executeComment(t *testing.T, cmd *cobra.Command, args []string, stdin string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestDiscussionCommentCommandsRegister(t *testing.T) {
	cmd := NewCmdComment(&cmdutil.Factory{})
	for _, name := range []string{"create", "reply", "edit", "delete"} {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child.Name() != name {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	create, _, _ := cmd.Find([]string{"create"})
	if create.Flags().Lookup("body") == nil || create.Flags().Lookup("body-file") == nil {
		t.Fatal("create must register body flags")
	}
	deleteCmd, _, _ := cmd.Find([]string{"delete"})
	if deleteCmd.Flags().Lookup("yes") == nil {
		t.Fatal("delete must register --yes")
	}
}

func TestDiscussionCommentCreatePostsMarkdown(t *testing.T) {
	var method, path, contentType, body string
	factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		method, path, contentType = req.Method, req.URL.Path, req.Header.Get("Content-Type")
		raw, _ := io.ReadAll(req.Body)
		body = string(raw)
		if req.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("Authorization = %q", req.Header.Get("Authorization"))
		}
		return discussionCommentResponse(http.StatusOK, `{"id":"c-string","md_content":"hello **there**"}`), nil
	})

	stdout, _, err := executeComment(t, newCmdCreate(factory), []string{"alice/demo", "7", "--body", "hello **there**"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/api/v5/repos/alice/demo/discuss/7/comment" || contentType != "application/json" {
		t.Fatalf("request = %s %s %s", method, path, contentType)
	}
	if body != `{"md_content":"hello **there**"}` {
		t.Fatalf("body = %s", body)
	}
	if !strings.Contains(stdout, "Created comment #c-string") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestDiscussionCommentCreateReadsBodyFileAndStdin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("from file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var got string
	factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(req.Body)
		got = string(raw)
		return discussionCommentResponse(http.StatusOK, `{"id":"file-id"}`), nil
	})
	if _, _, err := executeComment(t, newCmdCreate(factory), []string{"alice/demo", "7", "--body-file", path}, ""); err != nil {
		t.Fatal(err)
	}
	if got != `{"md_content":"from file\n"}` {
		t.Fatalf("file body = %s", got)
	}

	got = ""
	if _, _, err := executeComment(t, newCmdCreate(factory), []string{"alice/demo", "7", "--body-file", "-"}, "piped **md**"); err != nil {
		t.Fatal(err)
	}
	if got != `{"md_content":"piped **md**"}` {
		t.Fatalf("stdin body = %s", got)
	}
}

func TestDiscussionCommentCreateValidatesBeforeRequest(t *testing.T) {
	requests := 0
	factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		requests++
		return discussionCommentResponse(http.StatusOK, `{"id":"x"}`), nil
	})
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"alice/demo", "0", "--body", "a"}, "invalid discussion number"},
		{[]string{"alice/demo", "7"}, "comment body is required"},
		{[]string{"alice/demo", "7", "--body", "a", "--body-file", "note.md"}, "mutually exclusive"},
		{[]string{"alice/demo", "7", "--body", "   "}, "cannot be empty"},
	}
	for _, tc := range cases {
		_, _, err := executeComment(t, newCmdCreate(factory), tc.args, "")
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("args %v error = %v, want %q", tc.args, err, tc.want)
		}
	}
	if requests != 0 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestDiscussionCommentReplyRejectsWrongParent(t *testing.T) {
	writes := 0
	factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			writes++
			t.Fatalf("unexpected write %s %s", req.Method, req.URL.Path)
		}
		switch req.URL.Path {
		case "/api/v5/repos/alice/demo/discuss/7":
			return discussionCommentResponse(http.StatusOK, `{"number":7,"comment_total":1}`), nil
		case "/api/v5/repos/alice/demo/discuss/7/comment":
			if req.URL.Query().Get("page") == "1" {
				return discussionCommentResponse(http.StatusOK, `[{"id":"parent","reply_total":1}]`), nil
			}
			return discussionCommentResponse(http.StatusOK, `[]`), nil
		case "/api/v5/repos/alice/demo/discuss/7/comment/parent/reply":
			if req.URL.Query().Get("page") == "1" {
				return discussionCommentResponse(http.StatusOK, `[{"id":"child","reply_total":0}]`), nil
			}
			return discussionCommentResponse(http.StatusOK, `[]`), nil
		default:
			t.Fatalf("path = %s", req.URL.Path)
			return nil, nil
		}
	})

	_, _, err := executeComment(t, newCmdReply(factory), []string{"alice/demo", "7", "missing", "--body", "no"}, "")
	if err == nil || !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("missing parent error = %v", err)
	}
	_, _, err = executeComment(t, newCmdReply(factory), []string{"alice/demo", "7", "child", "--body", "no"}, "")
	if err == nil || !strings.Contains(err.Error(), "top-level") {
		t.Fatalf("reply parent error = %v", err)
	}
	if writes != 0 {
		t.Fatalf("writes = %d", writes)
	}
}

func TestDiscussionCommentReplyPostsAfterParentCheck(t *testing.T) {
	var methods []string
	factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		methods = append(methods, req.Method+" "+req.URL.Path)
		switch req.URL.Path {
		case "/api/v5/repos/alice/demo/discuss/7":
			return discussionCommentResponse(http.StatusOK, `{"number":7}`), nil
		case "/api/v5/repos/alice/demo/discuss/7/comment":
			if req.Method == http.MethodGet {
				if req.URL.Query().Get("page") == "1" {
					return discussionCommentResponse(http.StatusOK, `[{"id":"parent.id","reply_total":0}]`), nil
				}
				return discussionCommentResponse(http.StatusOK, `[]`), nil
			}
			t.Fatal("create was not expected")
			return nil, nil
		case "/api/v5/repos/alice/demo/discuss/7/comment/parent.id/reply":
			raw, _ := io.ReadAll(req.Body)
			if string(raw) != `{"md_content":"reply"}` || req.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("reply request = %s %s", req.Header.Get("Content-Type"), raw)
			}
			return discussionCommentResponse(http.StatusOK, `{"id":"reply-1"}`), nil
		default:
			t.Fatalf("path = %s", req.URL.EscapedPath())
			return nil, nil
		}
	})

	stdout, _, err := executeComment(t, newCmdReply(factory), []string{"alice/demo", "7", "parent.id", "--body", "reply"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "Created reply #reply-1") {
		t.Fatalf("stdout = %q", stdout)
	}
	if strings.Join(methods, ",") != strings.Join([]string{
		"GET /api/v5/repos/alice/demo/discuss/7",
		"GET /api/v5/repos/alice/demo/discuss/7/comment",
		"GET /api/v5/repos/alice/demo/discuss/7/comment",
		"POST /api/v5/repos/alice/demo/discuss/7/comment/parent.id/reply",
	}, ",") {
		t.Fatalf("methods = %#v", methods)
	}
}

func TestDiscussionCommentEditFindsTargetOnSecondPage(t *testing.T) {
	var methods []string
	factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		methods = append(methods, req.Method+" "+req.URL.RequestURI())
		switch {
		case req.URL.Path == "/api/v5/repos/alice/demo/discuss/7":
			return discussionCommentResponse(http.StatusOK, `{"number":7,"comment_total":1}`), nil
		case req.URL.Path == "/api/v5/repos/alice/demo/discuss/7/comment" && req.Method == http.MethodGet:
			if req.URL.Query().Get("page") == "1" {
				items := make([]map[string]any, 100)
				for i := range items {
					items[i] = map[string]any{"id": fmt.Sprintf("c-%03d", i), "reply_total": 0}
				}
				encoded, err := json.Marshal(items)
				if err != nil {
					t.Fatal(err)
				}
				return discussionCommentResponse(http.StatusOK, string(encoded)), nil
			}
			if req.URL.Query().Get("page") == "2" {
				return discussionCommentResponse(http.StatusOK, `[{"id":"target","reply_total":0}]`), nil
			}
			return discussionCommentResponse(http.StatusOK, `[]`), nil
		case req.Method == http.MethodPut && req.URL.Path == "/api/v5/repos/alice/demo/discuss/7/comment/target":
			raw, _ := io.ReadAll(req.Body)
			if string(raw) != `{"md_content":"edited"}` {
				t.Fatalf("body = %s", raw)
			}
			return discussionCommentResponse(http.StatusOK, `{"id":"target","md_content":"edited"}`), nil
		default:
			t.Fatalf("request = %s %s", req.Method, req.URL.RequestURI())
			return nil, nil
		}
	})

	stdout, _, err := executeComment(t, newCmdEdit(factory), []string{"alice/demo", "7", "target", "--body", "edited"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "Updated comment #target") {
		t.Fatalf("stdout = %q", stdout)
	}
	joined := strings.Join(methods, ",")
	if !strings.Contains(joined, "page=2") || !strings.HasSuffix(joined, "PUT /api/v5/repos/alice/demo/discuss/7/comment/target") {
		t.Fatalf("methods = %s", joined)
	}
}

func TestDiscussionCommentEditRejectsForeignComment(t *testing.T) {
	writes := 0
	factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			writes++
		}
		switch req.URL.Path {
		case "/api/v5/repos/alice/demo/discuss/7":
			return discussionCommentResponse(http.StatusOK, `{"number":7}`), nil
		case "/api/v5/repos/alice/demo/discuss/7/comment":
			if req.URL.Query().Get("page") == "1" {
				return discussionCommentResponse(http.StatusOK, `[{"id":"other","reply_total":0}]`), nil
			}
			return discussionCommentResponse(http.StatusOK, `[]`), nil
		default:
			t.Fatalf("path = %s", req.URL.Path)
			return nil, nil
		}
	})
	_, _, err := executeComment(t, newCmdEdit(factory), []string{"alice/demo", "7", "missing", "--body", "x"}, "")
	if err == nil || !strings.Contains(err.Error(), "does not belong") || writes != 0 {
		t.Fatalf("error = %v, writes = %d", err, writes)
	}
}

func TestDiscussionCommentDeleteConfirmation(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		yes    bool
		want   int
		output string
	}{
		{name: "cancel", input: "n\n", output: "Discussion comment deletion cancelled\n"},
		{name: "eof", output: "Discussion comment deletion cancelled\n"},
		{name: "confirm", input: "y\n", want: 1, output: "Deleted comment #target\n"},
		{name: "yes", input: "n\n", yes: true, want: 1, output: "Deleted comment #target\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deletes := 0
			factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
				switch {
				case req.URL.Path == "/api/v5/repos/alice/demo/discuss/7":
					return discussionCommentResponse(http.StatusOK, `{"number":7}`), nil
				case req.URL.Path == "/api/v5/repos/alice/demo/discuss/7/comment" && req.Method == http.MethodGet:
					if req.URL.Query().Get("page") == "1" {
						return discussionCommentResponse(http.StatusOK, `[{"id":"target","reply_total":0}]`), nil
					}
					return discussionCommentResponse(http.StatusOK, `[]`), nil
				case req.Method == http.MethodDelete:
					deletes++
					if req.URL.Path != "/api/v5/repos/alice/demo/discuss/7/comment/target" {
						t.Fatalf("delete path = %s", req.URL.Path)
					}
					return discussionCommentResponse(http.StatusOK, ""), nil
				default:
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
					return nil, nil
				}
			})
			cmd := newCmdDelete(factory)
			args := []string{"alice/demo", "7", "target"}
			if tt.yes {
				args = append(args, "--yes")
			}
			stdout, _, err := executeComment(t, cmd, args, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if deletes != tt.want || stdout != tt.output {
				t.Fatalf("deletes = %d stdout = %q", deletes, stdout)
			}
		})
	}
}

func TestDiscussionCommentDeleteNestedReplyAndEmptyBody(t *testing.T) {
	var sawDelete bool
	factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		switch req.URL.EscapedPath() {
		case "/api/v5/repos/alice/demo/discuss/7":
			return discussionCommentResponse(http.StatusOK, `{"number":7,"comment_total":1}`), nil
		case "/api/v5/repos/alice/demo/discuss/7/comment":
			if req.URL.Query().Get("page") == "1" {
				return discussionCommentResponse(http.StatusOK, `[{"id":"parent","reply_total":1}]`), nil
			}
			return discussionCommentResponse(http.StatusOK, `[]`), nil
		case "/api/v5/repos/alice/demo/discuss/7/comment/parent/reply":
			if req.URL.Query().Get("page") == "1" {
				return discussionCommentResponse(http.StatusOK, `[{"id":"child.id","reply_total":0}]`), nil
			}
			return discussionCommentResponse(http.StatusOK, `[]`), nil
		case "/api/v5/repos/alice/demo/discuss/7/comment/child.id":
			if req.Method != http.MethodDelete {
				t.Fatalf("method = %s", req.Method)
			}
			sawDelete = true
			return discussionCommentResponse(http.StatusOK, ""), nil
		default:
			t.Fatalf("path = %s", req.URL.EscapedPath())
			return nil, nil
		}
	})
	stdout, _, err := executeComment(t, newCmdDelete(factory), []string{"alice/demo", "7", "child.id", "--yes"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !sawDelete || stdout != "Deleted comment #child.id\n" {
		t.Fatalf("sawDelete = %v stdout = %q", sawDelete, stdout)
	}
}

func TestDiscussionCommentDeleteDoesNotReportFailedWriteAsSuccess(t *testing.T) {
	factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/api/v5/repos/alice/demo/discuss/7":
			return discussionCommentResponse(http.StatusOK, `{"number":7}`), nil
		case "/api/v5/repos/alice/demo/discuss/7/comment":
			if req.URL.Query().Get("page") == "1" {
				return discussionCommentResponse(http.StatusOK, `[{"id":"target","reply_total":0}]`), nil
			}
			return discussionCommentResponse(http.StatusOK, `[]`), nil
		case "/api/v5/repos/alice/demo/discuss/7/comment/target":
			return discussionCommentResponse(http.StatusForbidden, `{"message":"missing permission"}`), nil
		default:
			t.Fatalf("path = %s", req.URL.Path)
			return nil, nil
		}
	})
	stdout, _, err := executeComment(t, newCmdDelete(factory), []string{"alice/demo", "7", "target", "--yes"}, "")
	if err == nil || !strings.Contains(err.Error(), "failed to delete") || strings.Contains(stdout, "Deleted") {
		t.Fatalf("stdout = %q err = %v", stdout, err)
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("error = %v", err)
	}
}

func TestDiscussionCommentRejectsBadIDsBeforeAuth(t *testing.T) {
	cfg := discussionCommentConfig{tokenErr: errors.New("token failed")}
	factory := &cmdutil.Factory{Config: cfg}
	_, _, err := executeComment(t, newCmdDelete(factory), []string{"alice/demo", "7", "../x", "--yes"}, "")
	if err == nil || !strings.Contains(err.Error(), "invalid comment ID") || strings.Contains(err.Error(), "token failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestDiscussionCommentCreateReportsAPIFailure(t *testing.T) {
	factory := discussionCommentFactory(discussionCommentConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		return discussionCommentResponse(http.StatusNotFound, `{"message":"missing discussion"}`), nil
	})
	stdout, _, err := executeComment(t, newCmdCreate(factory), []string{"alice/demo", "7", "--body", "x"}, "")
	if err == nil || !strings.Contains(err.Error(), "failed to create") || strings.Contains(stdout, "Created") {
		t.Fatalf("stdout = %q err = %v", stdout, err)
	}
}
