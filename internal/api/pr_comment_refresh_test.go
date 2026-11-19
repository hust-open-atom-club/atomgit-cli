package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRefreshPRCommentsScopes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request PullRequestCommentRefreshRequest
		want    map[string]any
	}{
		{"paths", PullRequestCommentRefreshRequest{Paths: []string{"src/a,b.go", "docs/a b.md", " src/spaced.go "}}, map[string]any{"refresh_by_paths": []any{"src/a,b.go", "docs/a b.md", " src/spaced.go "}}},
		{"comment", PullRequestCommentRefreshRequest{NoteID: "hex-comment-id"}, map[string]any{"refresh_by_note_id": "hex-comment-id"}},
		{"all", PullRequestCommentRefreshRequest{All: true}, map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type key struct{}
			ctx := context.WithValue(t.Context(), key{}, true)
			calls := 0
			client := NewClientWithBaseURL("fixture", "https://example.test/api/v5", &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.EscapedPath() != "/api/v5/repos/team%2Fspace/repo%20name/pulls/12/refresh_position" || r.URL.RawQuery != "" {
					t.Fatalf("request=%s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer fixture" || r.Header.Get("Content-Type") != "application/json" || r.Context().Value(key{}) != true {
					t.Fatal("lost headers or context")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(body, tc.want) {
					t.Fatalf("body=%v want=%v", body, tc.want)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"note_ids":[0,9223372036854775807]}`)), Request: r}, nil
			})})
			result, err := RefreshPullRequestCommentPositions(client.WithContext(ctx), "team/space", "repo name", 12, tc.request)
			if err != nil || calls != 1 || !reflect.DeepEqual(result.NoteIDs, []int64{0, 9223372036854775807}) {
				t.Fatalf("result=%v calls=%d err=%v", result, calls, err)
			}
		})
	}
}

func TestRefreshPRCommentsNoReplay(t *testing.T) {
	for _, timeout := range []time.Duration{0, 30 * time.Second} {
		for _, status := range []int{201, 204, 301, 302, 303, 307, 308, 400, 401, 403, 404, 429, 500, 503} {
			t.Run(fmt.Sprintf("timeout=%s/%s", timeout, http.StatusText(status)), func(t *testing.T) {
				calls, redirects := 0, 0
				hc := &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil }, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if calls > 1 {
						t.Fatal("write replayed")
					}
					return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://other.test/"}, "Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(`{"note_ids":[]}`)), Request: r}, nil
				})}
				_, err := RefreshPullRequestCommentPositions(NewClientWithHTTPClient("fixture", hc), "team", "repo", 1, PullRequestCommentRefreshRequest{All: true})
				if !IsHTTPStatus(err, status) || calls != 1 || redirects != 0 {
					t.Fatalf("err=%v calls=%d redirects=%d", err, calls, redirects)
				}
				_ = hc.CheckRedirect(httptest.NewRequest(http.MethodGet, "https://other.test/", nil), nil)
				if redirects != 1 {
					t.Fatal("modified caller redirect policy")
				}
			})
		}
	}
	cause := errors.New("network failed")
	calls := 0
	client := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, cause })})
	_, err := RefreshPullRequestCommentPositions(client, "team", "repo", 1, PullRequestCommentRefreshRequest{All: true})
	if !errors.Is(err, cause) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestRefreshPRCommentsResponse(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"note_ids":null}`, `{"note_ids":[null]}`, `{"note_ids":["1"]}`, `{"note_ids":[1.5]}`, `{"note_ids":[9223372036854775808]}`, `{"note_ids":{}}`, `[]`, `{`, ``} {
		t.Run(body, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, body) })
			result, err := RefreshPullRequestCommentPositions(c, "team", "repo", 1, PullRequestCommentRefreshRequest{All: true})
			if err == nil || result.NoteIDs != nil {
				t.Fatalf("result=%v err=%v", result, err)
			}
		})
	}
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"note_ids":[]}`) })
	result, err := RefreshPullRequestCommentPositions(c, "team", "repo", 1, PullRequestCommentRefreshRequest{All: true})
	if err != nil || result.NoteIDs == nil || len(result.NoteIDs) != 0 {
		t.Fatalf("result=%v err=%v", result, err)
	}
}

func TestRefreshPRCommentsValidation(t *testing.T) {
	c := NewClientWithHTTPClient("fixture", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("invalid input sent request"); return nil, nil })})
	for _, r := range []PullRequestCommentRefreshRequest{{}, {Paths: []string{""}}, {Paths: []string{"a", " \t"}}, {NoteID: " \t"}, {All: true, NoteID: "1"}, {All: true, Paths: []string{"a"}}, {Paths: []string{"a"}, NoteID: "1"}} {
		if _, err := RefreshPullRequestCommentPositions(c, "team", "repo", 1, r); err == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
	for _, tc := range []struct {
		owner, repo string
		n           int
	}{{"", "repo", 1}, {"team", "", 1}, {"team", "repo", 0}} {
		if _, err := RefreshPullRequestCommentPositions(c, tc.owner, tc.repo, tc.n, PullRequestCommentRefreshRequest{All: true}); err == nil {
			t.Fatal("accepted invalid target")
		}
	}
}
