package discussion

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

const viewDetailFixture = `{
	"id": "d1",
	"number": 7,
	"title": "Roadmap",
	"md_content": "# Body\n- item",
	"created_at": "2026-08-09T23:05:18+08:00",
	"updated_at": "2026-08-10T10:00:00+08:00",
	"author": {"id": "u1", "login": "alice", "name": "Alice"},
	"is_lock": 0, "is_pin": 1, "is_category_pin": 0, "is_closed": 0, "is_answered": 1,
	"comment_total": 3,
	"category": {"id": "c1", "name": "公告", "icon": "📣", "description": "news", "type": 3}
}`

const viewCommentsFixture = `[
	{
		"id": "comment-1",
		"author": {"id": "u2", "login": "bob", "name": "Bob"},
		"content": "plain fallback",
		"md_content": "first **comment**",
		"created_at": "2026-08-10T11:00:00+08:00",
		"is_deleted": 0, "is_hide": 0, "like_total": 3, "reply_total": 2
	},
	{
		"id": "comment-2",
		"author": {"id": "u3", "login": "carol", "name": "Carol"},
		"content": "fallback text",
		"md_content": "",
		"created_at": "2026-08-10T12:00:00+08:00",
		"is_deleted": 0, "is_hide": 0, "like_total": 0, "reply_total": 0
	},
	{
		"id": "comment-3",
		"author": {"id": "u5", "login": "erin", "name": "Erin"},
		"content": "",
		"md_content": "",
		"created_at": "2026-08-10T14:00:00+08:00",
		"is_deleted": 1, "is_hide": 0, "like_total": 0, "reply_total": 0
	}
]`

const viewRepliesFixture = `[
	{
		"id": "reply-1",
		"author": {"id": "u4", "login": "dave", "name": "Dave"},
		"content": "reply body",
		"md_content": "nested reply",
		"created_at": "2026-08-10T13:00:00+08:00",
		"is_deleted": 0, "is_hide": 0, "like_total": 0, "reply_total": 0
	},
	{
		"id": "reply-2",
		"author": {"id": "u6", "login": "frank", "name": "Frank"},
		"content": "secret reply",
		"md_content": "secret reply",
		"created_at": "2026-08-10T13:30:00+08:00",
		"is_deleted": 0, "is_hide": 1, "like_total": 0, "reply_total": 0
	}
]`

func runViewCommand(t *testing.T, factory *cmdutil.Factory, args []string, flags map[string]string) (string, error) {
	t.Helper()
	cmd := newCmdDiscussionView(factory)
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.Args(cmd, args); err != nil {
		return output.String(), err
	}
	err := cmd.RunE(cmd, args)
	return output.String(), err
}

func compactJSON(t *testing.T, output string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(output)); err != nil {
		t.Fatalf("invalid JSON %q: %v", output, err)
	}
	return buf.String()
}

func TestNewCmdDiscussionViewRegistersFlagsAndHelp(t *testing.T) {
	cmd := newCmdDiscussionView(&cmdutil.Factory{})
	if cmd.Flags().Lookup("comments") == nil || cmd.Flags().Lookup("json") == nil {
		t.Fatal("discussion view must register --comments and --json")
	}
	if err := cmd.Args(cmd, nil); err == nil {
		t.Fatal("view accepted zero arguments")
	}
	if err := cmd.Args(cmd, []string{"owner/repo", "7", "extra"}); err == nil {
		t.Fatal("view accepted more than two arguments")
	}
	if err := cmd.Args(cmd, []string{"7"}); err != nil {
		t.Fatalf("view rejected number-only inference form: %v", err)
	}
	if err := cmd.Args(cmd, []string{"owner/repo", "7"}); err != nil {
		t.Fatalf("view rejected explicit repository: %v", err)
	}

	var help bytes.Buffer
	cmd.SetOut(&help)
	if err := cmd.Help(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"view [<owner>/<repo>] <number>", "--comments", "--json", "ag discussion view owner/repo 1"} {
		if !strings.Contains(help.String(), want) {
			t.Errorf("help missing %q:\n%s", want, help.String())
		}
	}
}

func TestDiscussionViewShowsDetailWithoutComments(t *testing.T) {
	requests := 0
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/owner/repo/discuss/7" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization = %q", got)
		}
		return discussionResponse(http.StatusOK, viewDetailFixture), nil
	})

	output, err := runViewCommand(t, factory, []string{"owner/repo", "7"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1 (comments must not be fetched without --comments)", requests)
	}
	for _, want := range []string{
		"#7", "Roadmap", "alice", "ANSWERED,PINNED", "# Body", "- item",
		"Comments:", "2026-08-10T10:00:00+08:00", "📣 公告",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "first **comment**") {
		t.Errorf("comment body rendered without --comments:\n%s", output)
	}
}

func TestDiscussionViewInfersRepository(t *testing.T) {
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v5/repos/inferred/repo/discuss/7" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		return discussionResponse(http.StatusOK, viewDetailFixture), nil
	})
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "inferred", Name: "repo"}, nil
	}

	if _, err := runViewCommand(t, factory, []string{"7"}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestDiscussionViewCommentsAndNestedReplies(t *testing.T) {
	requests := 0
	var replyPath string
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		requests++
		switch {
		case req.URL.Path == "/api/v5/repos/owner/repo/discuss/7":
			return discussionResponse(http.StatusOK, viewDetailFixture), nil
		case req.URL.Path == "/api/v5/repos/owner/repo/discuss/7/comment":
			return discussionResponse(http.StatusOK, viewCommentsFixture), nil
		default:
			replyPath = req.URL.Path
			return discussionResponse(http.StatusOK, viewRepliesFixture), nil
		}
	})

	output, err := runViewCommand(t, factory, []string{"owner/repo", "7"}, map[string]string{"comments": "true"})
	if err != nil {
		t.Fatal(err)
	}
	// detail + comments + one reply fetch for the single comment with replies
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
	if replyPath != "/api/v5/repos/owner/repo/discuss/7/comment/comment-1/reply" {
		t.Fatalf("reply path = %q", replyPath)
	}
	for _, want := range []string{
		"bob", "first **comment**", // markdown body wins over plain content
		"carol", "fallback text", // plain content used when markdown is empty
		"[deleted]",            // deleted placeholder
		"dave", "nested reply", // nested reply body
		"[hidden]", // hidden reply placeholder
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "plain fallback") {
		t.Errorf("plain content shown although markdown body exists:\n%s", output)
	}
}

func TestDiscussionViewPaginatesCommentsAndReplies(t *testing.T) {
	detailFixture := strings.Replace(viewDetailFixture, `"comment_total": 3`, `"comment_total": 101`, 1)
	commentPage := func(start, count int, firstReplyTotal int) string {
		items := make([]map[string]any, count)
		for i := range items {
			id := start + i
			items[i] = map[string]any{
				"id":          fmt.Sprintf("comment-%03d", id),
				"author":      map[string]string{"login": fmt.Sprintf("author-%03d", id)},
				"content":     fmt.Sprintf("comment-%03d", id),
				"created_at":  fmt.Sprintf("2026-08-10T%02d:00:00+08:00", (id % 24)),
				"reply_total": 0,
			}
		}
		if firstReplyTotal > 0 && len(items) > 0 {
			items[0]["reply_total"] = firstReplyTotal
		}
		encoded, err := json.Marshal(items)
		if err != nil {
			t.Fatalf("marshal comment page: %v", err)
		}
		return string(encoded)
	}
	replyPage := func(start, count int) string {
		items := make([]map[string]any, count)
		for i := range items {
			id := start + i
			items[i] = map[string]any{
				"id":         fmt.Sprintf("reply-%03d", id),
				"author":     map[string]string{"login": fmt.Sprintf("reply-author-%03d", id)},
				"content":    fmt.Sprintf("reply-%03d", id),
				"created_at": fmt.Sprintf("2026-08-11T%02d:00:00+08:00", (id % 24)),
			}
		}
		encoded, err := json.Marshal(items)
		if err != nil {
			t.Fatalf("marshal reply page: %v", err)
		}
		return string(encoded)
	}

	requests := 0
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		requests++
		page := req.URL.Query().Get("page")
		if got := req.URL.Query().Get("per_page"); req.URL.Path != "/api/v5/repos/owner/repo/discuss/7" && got != "100" {
			t.Fatalf("per_page = %q, want 100 for paginated endpoint", got)
		}
		switch req.URL.Path {
		case "/api/v5/repos/owner/repo/discuss/7":
			return discussionResponse(http.StatusOK, detailFixture), nil
		case "/api/v5/repos/owner/repo/discuss/7/comment":
			switch page {
			case "1":
				return discussionResponse(http.StatusOK, commentPage(1, 100, 101)), nil
			case "2":
				return discussionResponse(http.StatusOK, commentPage(101, 1, 0)), nil
			default:
				t.Fatalf("unexpected comment page %q", page)
				return nil, nil
			}
		case "/api/v5/repos/owner/repo/discuss/7/comment/comment-001/reply":
			switch page {
			case "1":
				return discussionResponse(http.StatusOK, replyPage(1, 100)), nil
			case "2":
				return discussionResponse(http.StatusOK, replyPage(101, 1)), nil
			default:
				t.Fatalf("unexpected reply page %q", page)
				return nil, nil
			}
		default:
			t.Fatalf("unexpected request path %q", req.URL.Path)
			return nil, nil
		}
	})

	output, err := runViewCommand(t, factory, []string{"owner/repo", "7"}, map[string]string{"comments": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 5 {
		t.Fatalf("requests = %d, want detail + 2 comment pages + 2 reply pages", requests)
	}
	for _, want := range []string{"comment-001", "comment-101", "reply-001", "reply-101"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if strings.Index(output, "reply-001") > strings.Index(output, "reply-101") {
		t.Error("replies are out of server order")
	}
	if strings.Index(output, "comment-001") > strings.Index(output, "comment-101") {
		t.Error("comments are out of server order")
	}
}

func TestDiscussionViewJSONIncludesComments(t *testing.T) {
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Path == "/api/v5/repos/owner/repo/discuss/7":
			return discussionResponse(http.StatusOK, viewDetailFixture), nil
		case req.URL.Path == "/api/v5/repos/owner/repo/discuss/7/comment":
			return discussionResponse(http.StatusOK, viewCommentsFixture), nil
		default:
			return discussionResponse(http.StatusOK, viewRepliesFixture), nil
		}
	})

	output, err := runViewCommand(t, factory, []string{"owner/repo", "7"}, map[string]string{"comments": "true", "json": "true"})
	if err != nil {
		t.Fatal(err)
	}
	compact := compactJSON(t, output)
	for _, want := range []string{
		`"number":7`, `"title":"Roadmap"`, `"author":"alice"`, `"status":"ANSWERED,PINNED"`,
		`"category":"📣 公告"`, `"body":"# Body\n- item"`, `"commentTotal":3`,
		`"comments":[{"id":"comment-1"`, `"author":"bob"`, `"body":"first **comment**"`, `"likeTotal":3`,
		`"body":"fallback text"`, `"deleted":true`, `"body":"[deleted]"`,
		`"replies":[{"id":"reply-1"`, `"hidden":true`, `"body":"[hidden]"`,
	} {
		if !strings.Contains(compact, want) {
			t.Errorf("json missing %q:\n%s", want, compact)
		}
	}
}

func TestDiscussionViewJSONWithoutCommentsOmitsKey(t *testing.T) {
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		return discussionResponse(http.StatusOK, viewDetailFixture), nil
	})

	output, err := runViewCommand(t, factory, []string{"owner/repo", "7"}, map[string]string{"json": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if compact := compactJSON(t, output); strings.Contains(compact, "comments") {
		t.Errorf("json contains comments key without --comments:\n%s", compact)
	}
}

func TestDiscussionViewJSONCommentsEmptyArray(t *testing.T) {
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/comment") {
			return discussionResponse(http.StatusOK, `[]`), nil
		}
		return discussionResponse(http.StatusOK, viewDetailFixture), nil
	})

	output, err := runViewCommand(t, factory, []string{"owner/repo", "7"}, map[string]string{"comments": "true", "json": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if compact := compactJSON(t, output); !strings.Contains(compact, `"comments":[]`) {
		t.Errorf("json missing empty comments array:\n%s", compact)
	}
}

func TestDiscussionViewMinimalCommentFields(t *testing.T) {
	factory := discussionFactory(discussionTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(req.URL.Path, "/comment"):
			return discussionResponse(http.StatusOK, `[{"id":"c1"}]`), nil
		default:
			return discussionResponse(http.StatusOK, viewDetailFixture), nil
		}
	})

	output, err := runViewCommand(t, factory, []string{"owner/repo", "7"}, map[string]string{"comments": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "[no content]") {
		t.Errorf("output missing [no content] placeholder:\n%s", output)
	}
	if !strings.Contains(output, "* -") { // missing author renders as "-"
		t.Errorf("output missing missing-author placeholder:\n%s", output)
	}
}

func TestDiscussionViewRejectsInvalidNumbersBeforeAuth(t *testing.T) {
	for _, number := range []string{"bad", "3.5", "0", "-3"} {
		t.Run(number, func(t *testing.T) {
			requests := 0
			// A credential error that would surface first if validation ran
			// after GetToken, proving the issue #49 ordering.
			factory := discussionFactory(discussionTestConfig{tokenErr: errors.New("cannot read token file: permission denied")}, func(*http.Request) (*http.Response, error) {
				requests++
				return discussionResponse(http.StatusOK, viewDetailFixture), nil
			})
			_, err := runViewCommand(t, factory, []string{"owner/repo", number}, nil)
			if err == nil || !strings.Contains(err.Error(), "invalid discussion number") {
				t.Fatalf("number %q: error = %v, want invalid discussion number", number, err)
			}
			if requests != 0 {
				t.Fatalf("requests = %d, want 0", requests)
			}
		})
	}
}

func TestDiscussionViewUnauthenticatedFallsBackToPublicAccess(t *testing.T) {
	factory := discussionFactory(discussionTestConfig{tokenErr: fmt.Errorf("wrapped credential error: %w", config.ErrNotAuthenticated)}, func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		return discussionResponse(http.StatusOK, viewDetailFixture), nil
	})

	if _, err := runViewCommand(t, factory, []string{"owner/repo", "7"}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestDiscussionViewPropagatesCredentialErrors(t *testing.T) {
	wantErr := errors.New("cannot read token file: permission denied")
	requests := 0
	factory := discussionFactory(discussionTestConfig{tokenErr: wantErr}, func(*http.Request) (*http.Response, error) {
		requests++
		return discussionResponse(http.StatusOK, viewDetailFixture), nil
	})

	_, err := runViewCommand(t, factory, []string{"owner/repo", "7"}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestDiscussionViewNon2xxErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		flags   map[string]string
		handler func(req *http.Request, calls int) (*http.Response, error)
		wantErr string
	}{
		{
			name:  "detail not found",
			flags: nil,
			handler: func(*http.Request, int) (*http.Response, error) {
				return discussionResponse(http.StatusNotFound, `{"error_message":"request failed"}`), nil
			},
			wantErr: "failed to view discussion #999 for owner/repo",
		},
		{
			name:  "comment failure",
			flags: map[string]string{"comments": "true"},
			handler: func(req *http.Request, calls int) (*http.Response, error) {
				if calls == 1 {
					return discussionResponse(http.StatusOK, viewDetailFixture), nil
				}
				return discussionResponse(http.StatusInternalServerError, `{"error_message":"request failed"}`), nil
			},
			wantErr: "failed to list comments of discussion #999",
		},
		{
			name:  "reply failure",
			flags: map[string]string{"comments": "true"},
			handler: func(req *http.Request, calls int) (*http.Response, error) {
				switch calls {
				case 1:
					return discussionResponse(http.StatusOK, viewDetailFixture), nil
				case 2:
					return discussionResponse(http.StatusOK, viewCommentsFixture), nil
				default:
					return discussionResponse(http.StatusBadGateway, `{"error_message":"request failed"}`), nil
				}
			},
			wantErr: "failed to list replies of comment comment-1 on discussion #999",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			factory := discussionFactory(discussionTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
				calls++
				return tc.handler(req, calls)
			})

			output, err := runViewCommand(t, factory, []string{"owner/repo", "999"}, tc.flags)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want %q", err, tc.wantErr)
			}
			if output != "" {
				t.Fatalf("output = %q, want empty", output)
			}
		})
	}
}
