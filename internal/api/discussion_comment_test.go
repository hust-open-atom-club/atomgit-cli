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
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
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
			if page == "2" {
				_, _ = io.WriteString(w, `[{"id":"later","reply_total":0}]`)
				return
			}
			_, _ = io.WriteString(w, `[]`)
		case r.URL.EscapedPath() == "/repos/alice/demo/discuss/7/comment/parent%201/reply":
			if r.URL.Query().Get("page") == "1" {
				_, _ = io.WriteString(w, `[{"id":"reply 9","reply_total":0}]`)
				return
			}
			_, _ = io.WriteString(w, `[]`)
		default:
			if strings.HasSuffix(r.URL.EscapedPath(), "/reply") {
				_, _ = io.WriteString(w, `[]`)
				return
			}
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
	if len(paths) != 3 || !strings.Contains(paths[1], "/comment?page=1") || !strings.Contains(paths[2], "parent%201/reply?page=1") || strings.Contains(paths[2], "page=2") {
		t.Fatalf("paths = %#v, want the parent comment page and only its matching reply page", paths)
	}
}

func TestFindDiscussionCommentContinuesAfterShortPage(t *testing.T) {
	var pages []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/repos/alice/demo/discuss/7":
			_, _ = io.WriteString(w, `{"number":7,"comment_total":1}`)
		case "/repos/alice/demo/discuss/7/comment":
			page := r.URL.Query().Get("page")
			pages = append(pages, page)
			switch page {
			case "1":
				_, _ = io.WriteString(w, `[{"id":"first","reply_total":0}]`)
			case "2":
				_, _ = io.WriteString(w, `[{"id":"target","reply_total":0}]`)
			default:
				_, _ = io.WriteString(w, `[]`)
			}
		default:
			if strings.HasSuffix(r.URL.EscapedPath(), "/reply") {
				_, _ = io.WriteString(w, `[]`)
				return
			}
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	})

	located, err := FindDiscussionComment(client, "alice", "demo", 7, "target")
	if err != nil {
		t.Fatal(err)
	}
	if !located.TopLevel || located.Comment.ID != "target" {
		t.Fatalf("location = %#v", located)
	}
	if strings.Join(pages, ",") != "1,2" {
		t.Fatalf("pages = %#v, want the short page and the matching page only", pages)
	}
}

func TestFindDiscussionCommentIgnoresStaleReplyTotal(t *testing.T) {
	var replyPages int
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/repos/alice/demo/discuss/7":
			_, _ = io.WriteString(w, `{"number":7}`)
		case "/repos/alice/demo/discuss/7/comment":
			if r.URL.Query().Get("page") == "1" {
				_, _ = io.WriteString(w, `[{"id":"parent","reply_total":0},{"id":"other","reply_total":0}]`)
				return
			}
			_, _ = io.WriteString(w, `[]`)
		case "/repos/alice/demo/discuss/7/comment/parent/reply":
			replyPages++
			if r.URL.Query().Get("page") == "1" {
				_, _ = io.WriteString(w, `[{"id":"hidden-reply","reply_total":0}]`)
				return
			}
			_, _ = io.WriteString(w, `[]`)
		case "/repos/alice/demo/discuss/7/comment/other/reply":
			_, _ = io.WriteString(w, `[]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	})

	located, err := FindDiscussionComment(client, "alice", "demo", 7, "hidden-reply")
	if err != nil {
		t.Fatal(err)
	}
	if located.TopLevel || located.ParentID != "parent" || located.Comment.ID != "hidden-reply" || replyPages != 1 {
		t.Fatalf("location = %#v, reply pages = %d, want the matching reply page only", located, replyPages)
	}
}

func TestDiscussionCommentWritesDoNotFollowRedirects(t *testing.T) {
	for _, status := range []int{http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls, redirects := 0, 0
			httpClient := &http.Client{
				CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil },
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://api.atomgit.com/api/v5/other"}}, Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}, nil
				}),
			}
			client := NewClientWithHTTPClient("fake-token", httpClient)
			_, err := CreateDiscussionComment(client, "alice", "demo", 7, "body")
			if !IsHTTPStatus(err, status) || calls != 1 || redirects != 0 {
				t.Fatalf("create err = %v, calls = %d, redirects = %d", err, calls, redirects)
			}
			calls = 0
			if err := DeleteDiscussionComment(client, "alice", "demo", 7, "target"); !IsHTTPStatus(err, status) || calls != 1 || redirects != 0 {
				t.Fatalf("delete err = %v, calls = %d, redirects = %d", err, calls, redirects)
			}
			if err := httpClient.CheckRedirect(nil, nil); err != nil || redirects != 1 {
				t.Fatal("shared HTTP client redirect policy changed")
			}
		})
	}
}

func TestRequireTopLevelDiscussionCommentRejectsReply(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/repos/alice/demo/discuss/7":
			_, _ = io.WriteString(w, `{"number":7,"comment_total":1}`)
		case "/repos/alice/demo/discuss/7/comment":
			if r.URL.Query().Get("page") == "1" {
				_, _ = io.WriteString(w, `[{"id":"parent","reply_total":1}]`)
				return
			}
			_, _ = io.WriteString(w, `[]`)
		case "/repos/alice/demo/discuss/7/comment/parent/reply":
			if r.URL.Query().Get("page") == "1" {
				_, _ = io.WriteString(w, `[{"id":"child","reply_total":0}]`)
				return
			}
			_, _ = io.WriteString(w, `[]`)
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
			if r.URL.Query().Get("page") == "1" {
				_, _ = io.WriteString(w, `[{"id":"other","reply_total":0}]`)
				return
			}
			_, _ = io.WriteString(w, `[]`)
		default:
			if strings.HasSuffix(r.URL.EscapedPath(), "/reply") {
				_, _ = io.WriteString(w, `[]`)
				return
			}
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

func TestFindDiscussionCommentStopsWhenTargetIsOnFirstPage(t *testing.T) {
	gets := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gets++
		switch r.URL.EscapedPath() {
		case "/repos/alice/demo/discuss/7":
			_, _ = io.WriteString(w, `{"number":7}`)
		case "/repos/alice/demo/discuss/7/comment":
			if r.URL.Query().Get("page") != "1" {
				t.Fatalf("page = %s, want only the matching first page", r.URL.Query().Get("page"))
			}
			_, _ = io.WriteString(w, `[{"id":"target","reply_total":0},{"id":"later","reply_total":0}]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	})

	located, err := FindDiscussionComment(client, "alice", "demo", 7, "target")
	if err != nil {
		t.Fatal(err)
	}
	if !located.TopLevel || located.Comment.ID != "target" || gets != 2 {
		t.Fatalf("location = %#v, gets = %d, want the discussion and the matching comment page", located, gets)
	}
}

func TestFindDiscussionCommentStopsBeforePageCapWhenTargetIsFound(t *testing.T) {
	gets := 0
	commentPages := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gets++
		switch r.URL.EscapedPath() {
		case "/repos/alice/demo/discuss/7":
			_, _ = io.WriteString(w, `{"number":7}`)
		case "/repos/alice/demo/discuss/7/comment":
			commentPages++
			page := r.URL.Query().Get("page")
			if page != "1" {
				t.Fatalf("comment page = %s, want to stop on the full page containing the target", page)
			}
			items := make([]string, 100)
			for i := range items {
				items[i] = fmt.Sprintf(`{"id":"c-%03d","reply_total":0}`, i)
			}
			items[0] = `{"id":"target","reply_total":0}`
			_, _ = io.WriteString(w, "["+strings.Join(items, ",")+"]")
		default:
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	})

	located, err := FindDiscussionComment(client, "alice", "demo", 7, "target")
	if err != nil {
		t.Fatal(err)
	}
	if !located.TopLevel || located.Comment.ID != "target" || gets != 2 || commentPages != 1 {
		t.Fatalf("location = %#v, gets = %d, comment pages = %d", located, gets, commentPages)
	}
}

func TestFindDiscussionReplyStopsBeforePageCapWhenTargetIsFound(t *testing.T) {
	gets := 0
	replyPages := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gets++
		switch r.URL.EscapedPath() {
		case "/repos/alice/demo/discuss/7":
			_, _ = io.WriteString(w, `{"number":7}`)
		case "/repos/alice/demo/discuss/7/comment":
			if r.URL.Query().Get("page") != "1" {
				t.Fatalf("comment page = %s", r.URL.Query().Get("page"))
			}
			_, _ = io.WriteString(w, `[{"id":"parent","reply_total":0}]`)
		case "/repos/alice/demo/discuss/7/comment/parent/reply":
			replyPages++
			page := r.URL.Query().Get("page")
			if page != "1" {
				t.Fatalf("reply page = %s, want to stop on the full page containing the target", page)
			}
			items := make([]string, 100)
			for i := range items {
				items[i] = fmt.Sprintf(`{"id":"r-%03d","reply_total":0}`, i)
			}
			items[0] = `{"id":"nested","reply_total":0}`
			_, _ = io.WriteString(w, "["+strings.Join(items, ",")+"]")
		default:
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	})

	located, err := FindDiscussionComment(client, "alice", "demo", 7, "nested")
	if err != nil {
		t.Fatal(err)
	}
	if located.TopLevel || located.ParentID != "parent" || located.Comment.ID != "nested" || gets != 3 || replyPages != 1 {
		t.Fatalf("location = %#v, gets = %d, reply pages = %d", located, gets, replyPages)
	}
}

func TestFindDiscussionCommentKeepsPageCapWhenTargetIsMissing(t *testing.T) {
	commentPages := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/repos/alice/demo/discuss/7":
			_, _ = io.WriteString(w, `{"number":7}`)
		case "/repos/alice/demo/discuss/7/comment":
			commentPages++
			items := make([]string, 100)
			for i := range items {
				items[i] = fmt.Sprintf(`{"id":"c-%d-%03d","reply_total":0}`, commentPages, i)
			}
			_, _ = io.WriteString(w, "["+strings.Join(items, ",")+"]")
		default:
			if strings.HasSuffix(r.URL.EscapedPath(), "/reply") {
				_, _ = io.WriteString(w, `[]`)
				return
			}
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	})

	_, err := FindDiscussionComment(client, "alice", "demo", 7, "missing")
	if err == nil || !strings.Contains(err.Error(), "discussion comment pagination reached the maximum of 100 pages") {
		t.Fatalf("error = %v", err)
	}
	if commentPages != 100 {
		t.Fatalf("comment pages = %d, want the cap before any reply scan", commentPages)
	}
}

func TestFindDiscussionCommentContinuesAfterUnrelatedReplyFailure(t *testing.T) {
	for _, failure := range []string{"page cap", "forbidden"} {
		for _, target := range []string{"top-level", "later reply", "missing"} {
			t.Run(failure+"/"+target, func(t *testing.T) {
				commentPages, replyPages := 0, 0
				client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/repos/alice/demo/discuss/7":
						_, _ = io.WriteString(w, `{"number":7}`)
					case "/repos/alice/demo/discuss/7/comment":
						commentPages++
						switch r.URL.Query().Get("page") {
						case "1":
							items := make([]DiscussionComment, 100)
							for i := range items {
								items[i].ID = fmt.Sprintf("parent-%d", i)
							}
							items[0].ReplyTotal = 10000
							_ = json.NewEncoder(w).Encode(items)
						case "2":
							_, _ = io.WriteString(w, `[{"id":"top-level"}]`)
						default:
							_, _ = io.WriteString(w, `[]`)
						}
					case "/repos/alice/demo/discuss/7/comment/parent-0/reply":
						replyPages++
						if failure == "forbidden" {
							http.Error(w, "forbidden", http.StatusForbidden)
							return
						}
						items := make([]DiscussionComment, 100)
						for i := range items {
							items[i].ID = fmt.Sprintf("unrelated-%d-%d", replyPages, i)
						}
						_ = json.NewEncoder(w).Encode(items)
					case "/repos/alice/demo/discuss/7/comment/top-level/reply":
						if r.URL.Query().Get("page") == "1" {
							_, _ = io.WriteString(w, `[{"id":"later reply"}]`)
						} else {
							_, _ = io.WriteString(w, `[]`)
						}
					default:
						if !strings.HasSuffix(r.URL.Path, "/reply") {
							t.Fatalf("unexpected request: %s", r.URL)
						}
						_, _ = io.WriteString(w, `[]`)
					}
				})
				located, err := FindDiscussionComment(client, "alice", "demo", 7, target)
				if target == "missing" {
					if err == nil || errors.Is(err, ErrDiscussionCommentNotFound) {
						t.Fatalf("error = %v, want incomplete lookup failure", err)
					}
					if failure == "forbidden" && !IsHTTPStatus(err, http.StatusForbidden) {
						t.Fatalf("error = %v, want original 403", err)
					}
					if failure == "page cap" && !strings.Contains(err.Error(), "maximum of 100 pages") {
						t.Fatalf("error = %v, want pagination cap", err)
					}
				} else if err != nil || located.Comment.ID != target || located.TopLevel != (target == "top-level") {
					t.Fatalf("location = %#v, error = %v", located, err)
				}
				wantPages := 2
				if target == "missing" {
					wantPages = 3
				}
				wantReplies := 100
				if failure == "forbidden" {
					wantReplies = 1
				}
				if commentPages != wantPages || replyPages != wantReplies {
					t.Fatalf("comment pages = %d, reply pages = %d", commentPages, replyPages)
				}
			})
		}
	}
}
