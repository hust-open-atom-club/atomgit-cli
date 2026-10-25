package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type organizationDiscussionWriteCase struct {
	name   string
	method string
	path   string
	fields map[string]string
	call   func(*Client) (any, error)
}

func organizationDiscussionWriteCases() []organizationDiscussionWriteCase {
	const organization = "org/name %?"
	const commentID = "c/1 ?%"
	const body = "# 中文\n"
	const base = "/orgs/org%2Fname%20%25%3F/discuss"
	return []organizationDiscussionWriteCase{
		{
			name: "create", method: http.MethodPost, path: base,
			fields: map[string]string{"title": "Title", "md_content": body, "category_name": "Ideas"},
			call: func(client *Client) (any, error) {
				return CreateOrganizationDiscussion(client, organization, DiscussionCreateRequest{Title: "Title", MDContent: body, CategoryName: "Ideas"})
			},
		},
		{
			name: "edit", method: http.MethodPut, path: base + "/7",
			fields: map[string]string{"title": "Title", "md_content": "", "category_name": "Ideas"},
			call: func(client *Client) (any, error) {
				return UpdateOrganizationDiscussion(client, organization, 7, DiscussionEditRequest{Title: "Title", MDContent: "", CategoryName: "Ideas"})
			},
		},
		{
			name: "delete", method: http.MethodDelete, path: base + "/7",
			call: func(client *Client) (any, error) {
				return nil, DeleteOrganizationDiscussion(client, organization, 7)
			},
		},
		{
			name: "comment-create", method: http.MethodPost, path: base + "/7/comment",
			fields: map[string]string{"md_content": body},
			call: func(client *Client) (any, error) {
				return CreateOrganizationDiscussionComment(client, organization, 7, body)
			},
		},
		{
			name: "comment-reply", method: http.MethodPost, path: base + "/7/comment/c%2F1%20%3F%25/reply",
			fields: map[string]string{"md_content": body},
			call: func(client *Client) (any, error) {
				return ReplyOrganizationDiscussionComment(client, organization, 7, commentID, body)
			},
		},
		{
			name: "comment-edit", method: http.MethodPut, path: base + "/7/comment/c%2F1%20%3F%25",
			fields: map[string]string{"md_content": body},
			call: func(client *Client) (any, error) {
				return UpdateOrganizationDiscussionComment(client, organization, 7, commentID, body)
			},
		},
		{
			name: "comment-delete", method: http.MethodDelete, path: base + "/7/comment/c%2F1%20%3F%25",
			call: func(client *Client) (any, error) {
				return nil, DeleteOrganizationDiscussionComment(client, organization, 7, commentID)
			},
		},
	}
}

func TestOrganizationDiscussionWritesContract(t *testing.T) {
	for _, tt := range organizationDiscussionWriteCases() {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != tt.method || r.URL.EscapedPath() != tt.path {
					t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.EscapedPath(), tt.method, tt.path)
				}
				if r.Header.Get("Accept") != "application/json" {
					t.Errorf("Accept = %q", r.Header.Get("Accept"))
				}
				if tt.fields == nil {
					if r.Body == nil {
						return
					}
					body, err := io.ReadAll(r.Body)
					if err != nil || len(body) != 0 {
						t.Errorf("delete body = %q, error = %v", body, err)
					}
					return
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
				}
				var fields map[string]string
				if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(fields, tt.fields) {
					t.Errorf("fields = %#v, want %#v", fields, tt.fields)
				}
				_, _ = io.WriteString(w, `{"id":"result","number":7,"md_content":"# 中文\n"}`)
			})
			result, err := tt.call(client)
			if err != nil || calls != 1 {
				t.Fatalf("result = %#v, error = %v, calls = %d", result, err, calls)
			}
			switch result := result.(type) {
			case DiscussionDetail:
				if result.Number != 7 || !result.MDContentPresent {
					t.Errorf("detail = %#v", result)
				}
			case DiscussionComment:
				if result.ID != "result" || result.MDContent != "# 中文\n" {
					t.Errorf("comment = %#v", result)
				}
			}
		})
	}
}

func TestOrganizationDiscussionWritesRejectNonOKAndDoNotReplay(t *testing.T) {
	for _, tt := range organizationDiscussionWriteCases() {
		for _, status := range []int{201, 204, 301, 302, 303, 307, 308, 403, 404, 429, 503} {
			t.Run(fmt.Sprintf("%s/%d", tt.name, status), func(t *testing.T) {
				calls, redirects := 0, 0
				hc := &http.Client{
					CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil },
					Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://example.test/other"}}, Body: io.NopCloser(strings.NewReader(`{"message":"denied"}`)), Request: r}, nil
					}),
				}
				_, err := tt.call(NewClientWithBaseURL("fake-token", "https://example.test", hc))
				if !IsHTTPStatus(err, status) || calls != 1 || redirects != 0 {
					t.Fatalf("error = %v, calls = %d, redirects = %d", err, calls, redirects)
				}
				if err := hc.CheckRedirect(nil, nil); err != nil || redirects != 1 {
					t.Fatal("shared HTTP client redirect policy changed")
				}
			})
		}
	}
}

func TestOrganizationDiscussionWritesDoNotRetryNetworkFailures(t *testing.T) {
	for _, tt := range organizationDiscussionWriteCases() {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := NewClientWithBaseURL("fake-token", "https://example.test", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return nil, errors.New("network failure")
			})})
			if _, err := tt.call(client); err == nil || calls != 1 {
				t.Fatalf("error = %v, calls = %d", err, calls)
			}
		})
	}
}

func TestOrganizationDiscussionWritesRejectInvalidResponses(t *testing.T) {
	for _, tt := range organizationDiscussionWriteCases() {
		if tt.method == http.MethodDelete {
			continue
		}
		for _, body := range []string{"", "{", "[]", "42"} {
			t.Run(tt.name+"/"+body, func(t *testing.T) {
				client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.WriteString(w, body)
				})
				if _, err := tt.call(client); err == nil {
					t.Fatal("expected invalid response error")
				}
			})
		}
	}
}

func TestFindOrganizationDiscussionCommentPagesAndReplies(t *testing.T) {
	for _, target := range []string{"later", "nested/reply"} {
		t.Run(target, func(t *testing.T) {
			const base = "/orgs/org%2Fname/discuss/7"
			var requests []string
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.RequestURI())
				if r.Method != http.MethodGet {
					t.Errorf("unexpected method %s", r.Method)
				}
				if r.URL.EscapedPath() == base {
					_, _ = io.WriteString(w, `{"number":7,"comment_total":0}`)
					return
				}
				if r.URL.Query().Get("per_page") != "100" {
					t.Errorf("query = %q", r.URL.RawQuery)
				}
				page := r.URL.Query().Get("page")
				switch r.URL.EscapedPath() {
				case base + "/comment":
					if page == "1" {
						_, _ = io.WriteString(w, `[{"id":"parent/1","reply_total":0}]`)
						return
					}
					if page == "2" {
						_, _ = io.WriteString(w, `[{"id":"later"}]`)
						return
					}
				case base + "/comment/parent%2F1/reply":
					if page == "1" {
						_, _ = io.WriteString(w, `[{"id":"first-reply"}]`)
						return
					}
					if page == "2" {
						_, _ = io.WriteString(w, `[{"id":"nested/reply"}]`)
						return
					}
				default:
					t.Errorf("unexpected path %s", r.URL.EscapedPath())
				}
				_, _ = io.WriteString(w, `[]`)
			})
			located, err := FindOrganizationDiscussionComment(client, "org/name", 7, target)
			if err != nil || located.Comment.ID != target || located.TopLevel != (target == "later") {
				t.Fatalf("location = %#v, error = %v, requests = %v", located, err, requests)
			}
			if target == "nested/reply" && located.ParentID != "parent/1" {
				t.Errorf("parent = %q", located.ParentID)
			}
		})
	}
}

func TestFindOrganizationDiscussionCommentRefusesUnverifiedMembership(t *testing.T) {
	tests := []struct {
		name        string
		detail      string
		detailCode  int
		commentCode int
		wantStatus  int
		wantMissing bool
		wantCalls   int
	}{
		{name: "missing-number", detail: `{}`, wantCalls: 1},
		{name: "wrong-number", detail: `{"number":8}`, wantCalls: 1},
		{name: "invalid-detail", detail: `{`, wantCalls: 1},
		{name: "forbidden-detail", detailCode: 403, wantStatus: 403, wantCalls: 1},
		{name: "missing-detail", detailCode: 404, wantStatus: 404, wantCalls: 1},
		{name: "forbidden-comments", detail: `{"number":7}`, commentCode: 403, wantStatus: 403, wantCalls: 2},
		{name: "other-discussion-comment", detail: `{"number":7}`, wantMissing: true, wantCalls: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet {
					t.Errorf("unexpected method %s", r.Method)
				}
				if strings.HasSuffix(r.URL.Path, "/7") {
					if tt.detailCode != 0 {
						w.WriteHeader(tt.detailCode)
					}
					_, _ = io.WriteString(w, tt.detail)
					return
				}
				if tt.commentCode != 0 {
					w.WriteHeader(tt.commentCode)
				}
				_, _ = io.WriteString(w, `[]`)
			})
			_, err := FindOrganizationDiscussionComment(client, "org", 7, "elsewhere")
			if err == nil || calls != tt.wantCalls || errors.Is(err, ErrDiscussionCommentNotFound) != tt.wantMissing {
				t.Fatalf("error = %v, calls = %d", err, calls)
			}
			if tt.wantStatus != 0 && !IsHTTPStatus(err, tt.wantStatus) {
				t.Errorf("error = %v, want HTTP %d", err, tt.wantStatus)
			}
		})
	}
}

func TestRequireTopLevelOrganizationDiscussionComment(t *testing.T) {
	for _, target := range []string{"parent", "child"} {
		t.Run(target, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/orgs/org/discuss/7":
					_, _ = io.WriteString(w, `{"number":7}`)
				case "/orgs/org/discuss/7/comment":
					_, _ = io.WriteString(w, `[{"id":"parent"}]`)
				case "/orgs/org/discuss/7/comment/parent/reply":
					_, _ = io.WriteString(w, `[{"id":"child"}]`)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
				}
			})
			comment, err := RequireTopLevelOrganizationDiscussionComment(client, "org", 7, target)
			if target == "child" {
				if !errors.Is(err, ErrDiscussionReplyParent) {
					t.Errorf("error = %v", err)
				}
			} else if err != nil || comment.ID != target {
				t.Errorf("comment = %#v, error = %v", comment, err)
			}
		})
	}
}
