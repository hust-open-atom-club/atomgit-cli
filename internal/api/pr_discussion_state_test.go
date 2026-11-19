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

type discussionStateTransport func(*http.Request) (*http.Response, error)

func (f discussionStateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func discussionStateResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestPRDiscussionIdentifiersRejectBeforeRequest(t *testing.T) {
	for _, id := range []string{"", ".", "..", "../id", "a/b", "a\\b", "id?x=1", "id#frag", "id%2f", " id", "id\n"} {
		t.Run(fmt.Sprintf("%q", id), func(t *testing.T) {
			if err := SetPullRequestDiscussionResolved(nil, "alice", "demo", 8, id, false); err == nil {
				t.Fatal("accepted invalid identifier")
			}
			if _, err := GetPullRequestDiscussionState(nil, "alice", "demo", 8, id); err == nil {
				t.Fatal("accepted invalid identifier")
			}
		})
	}
	if err := SetPullRequestDiscussionResolved(nil, "alice", "demo", 0, "thread", true); err == nil {
		t.Fatal("accepted zero PR number")
	}
}

func TestGetPRDiscussionStateScopesAndPaginates(t *testing.T) {
	var paths []string
	client := NewClientWithHTTPClient("token", &http.Client{Transport: discussionStateTransport(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.URL.RequestURI())
		if req.URL.Path == "/api/v5/repos/alice/demo/pulls/8" {
			return discussionStateResponse(200, `{"number":8}`), nil
		}
		if req.URL.Path != "/api/v5/repos/alice/demo/pulls/8/comments" || req.URL.Query().Get("view") != "all" || req.URL.Query().Get("per_page") != "100" {
			t.Fatalf("unexpected request: %s", req.URL)
		}
		switch req.URL.Query().Get("page") {
		case "1":
			return discussionStateResponse(200, `[{"discussion_id":"other","reply":[{"discussion_id":"opaque-id","comment_type":"diff_comment","resolved":true}]}]`), nil
		case "2":
			return discussionStateResponse(200, `[{"id":42,"discussion_id":"opaque-id","comment_type":"diff_comment","resolved":false}]`), nil
		default:
			t.Fatalf("unexpected page: %s", req.URL)
			return nil, nil
		}
	})})
	state, err := GetPullRequestDiscussionState(client, "alice", "demo", 8, "opaque-id")
	if err != nil || state.Resolved == nil || *state.Resolved || state.DiscussionID != "opaque-id" || len(paths) != 3 {
		t.Fatalf("state=%+v paths=%v err=%v", state, paths, err)
	}
}

func TestGetPRDiscussionStateRejectsWrongOrIncompleteIdentity(t *testing.T) {
	for _, tc := range []struct{ name, pr, comments string }{
		{"wrong PR", `{"number":9}`, `[]`}, {"missing PR", `{}`, `[]`},
		{"note ID", `{"number":8}`, `[{"id":42,"discussion_id":"actual-thread","comment_type":"diff_comment","resolved":true}]`},
		{"plain comment", `{"number":8}`, `[{"discussion_id":"42","comment_type":"pr_comment","resolved":true}]`},
		{"missing resolved", `{"number":8}`, `[{"discussion_id":"42","comment_type":"diff_comment"}]`},
		{"null resolved", `{"number":8}`, `[{"discussion_id":"42","comment_type":"diff_comment","resolved":null}]`},
		{"null page", `{"number":8}`, `null`}, {"malformed page", `{"number":8}`, `{`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClientWithHTTPClient("token", &http.Client{Transport: discussionStateTransport(func(req *http.Request) (*http.Response, error) {
				if !strings.HasSuffix(req.URL.Path, "/comments") {
					return discussionStateResponse(200, tc.pr), nil
				}
				if req.URL.Query().Get("page") != "1" {
					return discussionStateResponse(200, `[]`), nil
				}
				return discussionStateResponse(200, tc.comments), nil
			})})
			if _, err := GetPullRequestDiscussionState(client, "alice", "demo", 8, "42"); err == nil {
				t.Fatal("accepted wrong or incomplete identity")
			}
		})
	}
}

func TestSetPRDiscussionResolvedExplicitBooleanAndEmptyResponse(t *testing.T) {
	for _, resolved := range []bool{true, false} {
		t.Run(fmt.Sprint(resolved), func(t *testing.T) {
			calls := 0
			client := NewClientWithHTTPClient("token", &http.Client{Transport: discussionStateTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPut || req.URL.Path != "/api/v5/repos/alice/demo/pulls/8/comments/opaque-id" || req.Header.Get("Content-Type") != "application/json" || req.Header.Get("Authorization") != "Bearer token" {
					t.Fatalf("request: %s %s", req.Method, req.URL)
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if len(body) != 1 || string(body["resolved"]) != fmt.Sprint(resolved) {
					t.Fatalf("body=%v, wanted explicit %t", body, resolved)
				}
				return discussionStateResponse(200, ""), nil
			})})
			if err := SetPullRequestDiscussionResolved(client, "alice", "demo", 8, "opaque-id", resolved); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("calls=%d", calls)
			}
		})
	}
}

func TestSetPRDiscussionResolvedFailuresNeverRetry(t *testing.T) {
	for _, status := range []int{201, 204, 304, 401, 403, 404, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			client := NewClientWithHTTPClient("token", &http.Client{Transport: discussionStateTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return discussionStateResponse(status, `{}`), nil
			})})
			err := SetPullRequestDiscussionResolved(client, "alice", "demo", 8, "thread", true)
			var httpErr *HTTPError
			if !errors.As(err, &httpErr) || httpErr.StatusCode != status || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
	cause := errors.New("transport failed")
	calls := 0
	client := NewClientWithHTTPClient("token", &http.Client{Transport: discussionStateTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, cause })})
	if err := SetPullRequestDiscussionResolved(client, "alice", "demo", 8, "thread", true); !errors.Is(err, cause) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestSetPRDiscussionResolvedDoesNotFollowRedirect(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls, redirects := 0, 0
			hc := &http.Client{
				CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil },
				Transport: discussionStateTransport(func(req *http.Request) (*http.Response, error) {
					calls++
					if calls > 1 {
						t.Fatal("redirected or replayed write")
					}
					if req.Method != http.MethodPut {
						t.Fatalf("method=%s", req.Method)
					}
					response := discussionStateResponse(status, "")
					response.Header.Set("Location", "https://redirect.test/replayed")
					response.Request = req
					return response, nil
				}),
			}
			client := NewClientWithBaseURL("token", "https://example.test/api/v5", hc)
			err := SetPullRequestDiscussionResolved(client, "alice", "demo", 8, "thread", true)
			var httpErr *HTTPError
			if !errors.As(err, &httpErr) || httpErr.StatusCode != status || calls != 1 || redirects != 0 {
				t.Fatalf("calls=%d redirects=%d err=%v", calls, redirects, err)
			}
			if err := hc.CheckRedirect(nil, nil); err != nil || redirects != 1 {
				t.Fatalf("caller redirect policy was changed: redirects=%d err=%v", redirects, err)
			}
		})
	}
}
