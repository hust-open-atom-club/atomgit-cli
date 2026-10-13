package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDiscussionCommentWritesUseContract(t *testing.T) {
	const response = `{"id":"c 1","md_content":"hello **there**","author":{"login":"alice"}}`
	tests := []struct {
		name   string
		call   func(*Client) (DiscussionComment, error)
		method string
		path   string
	}{
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/repos/org%2Fname/re%20po/discuss/7/comment",
			call: func(client *Client) (DiscussionComment, error) {
				return CreateDiscussionComment(client, "org/name", "re po", 7, "hello **there**")
			},
		},
		{
			name:   "reply",
			method: http.MethodPost,
			path:   "/repos/alice/demo/discuss/7/comment/c%201/reply",
			call: func(client *Client) (DiscussionComment, error) {
				return ReplyDiscussionComment(client, "alice", "demo", 7, "c 1", "hello **there**")
			},
		},
		{
			name:   "update",
			method: http.MethodPut,
			path:   "/repos/alice/demo/discuss/7/comment/c%201",
			call: func(client *Client) (DiscussionComment, error) {
				return UpdateDiscussionComment(client, "alice", "demo", 7, "c 1", "hello **there**")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != tt.method {
					t.Fatalf("method = %s, want %s", r.Method, tt.method)
				}
				if r.URL.EscapedPath() != tt.path {
					t.Fatalf("path = %q, want %q", r.URL.EscapedPath(), tt.path)
				}
				if got := r.Header.Get("Content-Type"); got != "application/json" {
					t.Fatalf("Content-Type = %q", got)
				}
				var payload struct {
					MDContent string `json:"md_content"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if payload.MDContent != "hello **there**" {
					t.Fatalf("md_content = %q", payload.MDContent)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, response)
			})

			comment, err := tt.call(client)
			if err != nil {
				t.Fatal(err)
			}
			if calls != 1 || comment.ID != "c 1" || comment.MDContent != "hello **there**" {
				t.Fatalf("calls = %d, comment = %#v", calls, comment)
			}
		})
	}
}

func TestDiscussionCommentWritesRejectNonOK(t *testing.T) {
	for _, status := range []int{http.StatusCreated, http.StatusNoContent, http.StatusForbidden, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"message":"denied"}`)
			})
			_, err := CreateDiscussionComment(client, "alice", "demo", 7, "body")
			if !IsHTTPStatus(err, status) {
				t.Fatalf("error = %v, want HTTP %d", err, status)
			}
		})
	}
}

func TestDiscussionCommentWriteDoesNotRetry(t *testing.T) {
	calls := 0
	client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return nil, errors.New("network failure")
		}),
	})

	_, err := CreateDiscussionComment(client, "alice", "demo", 7, "body")
	if err == nil {
		t.Fatal("expected network error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestDeleteDiscussionCommentAcceptsEmptyBody(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodDelete || r.URL.EscapedPath() != "/repos/alice/demo/discuss/7/comment/c%201" {
					t.Fatalf("request = %s %s", r.Method, r.URL.EscapedPath())
				}
				w.WriteHeader(status)
			})
			if err := DeleteDiscussionComment(client, "alice", "demo", 7, "c 1"); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("calls = %d, want 1", calls)
			}
		})
	}
}

func TestFindDiscussionCommentPagesAndNestedReply(t *testing.T) {
	var paths []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		switch {
		case r.URL.EscapedPath() == "/repos/alice/demo/discuss/7":
			_, _ = io.WriteString(w, `{"number":7,"comment_total":0}`)
		case r.URL.EscapedPath() == "/repos/alice/demo/discuss/7/comment":
			page := r.URL.Query().Get("page")
			if page == "1" {
				items := make([]string, 100)
				for i := range items {
					items[i] = fmt.Sprintf(`{"id":"c-%03d","reply_total":0}`, i)
				}
				items[0] = `{"id":"parent 1","reply_total":1}`
				_, _ = io.WriteString(w, "["+strings.Join(items, ",")+"]")
				return
			}
			_, _ = io.WriteString(w, `[{"id":"later","reply_total":0}]`)
		case r.URL.EscapedPath() == "/repos/alice/demo/discuss/7/comment/parent%201/reply":
			_, _ = io.WriteString(w, `[{"id":"reply 9","reply_total":0}]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	})

	located, err := FindDiscussionComment(client, "alice", "demo", 7, "reply 9")
	if err != nil {
		t.Fatal(err)
	}
	if located.TopLevel || located.ParentID != "parent 1" || located.Comment.ID != "reply 9" {
		t.Fatalf("location = %#v", located)
	}
	if len(paths) < 4 || !strings.Contains(paths[2], "page=2") {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestRequireTopLevelDiscussionCommentRejectsReply(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/repos/alice/demo/discuss/7":
			_, _ = io.WriteString(w, `{"number":7,"comment_total":1}`)
		case "/repos/alice/demo/discuss/7/comment":
			_, _ = io.WriteString(w, `[{"id":"parent","reply_total":1}]`)
		case "/repos/alice/demo/discuss/7/comment/parent/reply":
			_, _ = io.WriteString(w, `[{"id":"child","reply_total":0}]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	})

	_, err := RequireTopLevelDiscussionComment(client, "alice", "demo", 7, "child")
	if !errors.Is(err, ErrDiscussionReplyParent) {
		t.Fatalf("error = %v", err)
	}
}

func TestFindDiscussionCommentMissingDoesNotImplySuccess(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/repos/alice/demo/discuss/7":
			_, _ = io.WriteString(w, `{"number":7,"comment_total":1}`)
		case "/repos/alice/demo/discuss/7/comment":
			_, _ = io.WriteString(w, `[{"id":"other","reply_total":0}]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	})

	_, err := FindDiscussionComment(client, "alice", "demo", 7, "missing")
	if !errors.Is(err, ErrDiscussionCommentNotFound) {
		t.Fatalf("error = %v", err)
	}
}

func TestFindDiscussionCommentStopsWhenLookupFails(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() == "/repos/alice/demo/discuss/7" {
			http.NotFound(w, r)
			return
		}
		t.Fatalf("unexpected path %s", r.URL.EscapedPath())
	})
	_, err := FindDiscussionComment(client, "alice", "demo", 7, "missing")
	if !IsHTTPStatus(err, http.StatusNotFound) {
		t.Fatalf("error = %v, want 404", err)
	}
}
