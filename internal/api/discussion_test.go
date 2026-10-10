package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestDiscussionDetailBodyPresence(t *testing.T) {
	var detail DiscussionDetail
	for _, tc := range []struct {
		body    string
		present bool
	}{
		{`{"md_content":""}`, true},
		{`{}`, false},
		{`{"md_content":"body"}`, true},
		{`{"md_content":null}`, false},
	} {
		if err := json.Unmarshal([]byte(tc.body), &detail); err != nil {
			t.Fatal(err)
		}
		if detail.MDContentPresent != tc.present {
			t.Fatalf("%s: body presence = %v", tc.body, detail.MDContentPresent)
		}
		encoded, err := json.Marshal(detail)
		if err != nil || strings.Contains(string(encoded), "Present") {
			t.Fatalf("internal presence field leaked: %s, %v", encoded, err)
		}
	}
}

func TestUpdateDiscussionJSONRequest(t *testing.T) {
	for _, body := range []string{"# Heading\n\"quoted\" & `code` \\path\n", ""} {
		t.Run(body, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.EscapedPath() != "/repos/org%2Fname/re%20po/discuss/7" {
					t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
				}
				if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
					t.Errorf("headers = %v", r.Header)
				}
				var fields map[string]string
				if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
					t.Error(err)
				}
				want := map[string]string{"title": "Updated", "md_content": body, "category_name": "Ideas"}
				if !reflect.DeepEqual(fields, want) {
					t.Errorf("fields = %#v, want %#v", fields, want)
				}
				_, _ = io.WriteString(w, `{"id":"d1","number":7,"title":"Updated","md_content":"","category":{"name":"Ideas"}}`)
			})
			updated, err := UpdateDiscussion(client, "org/name", "re po", 7, DiscussionEditRequest{Title: "Updated", MDContent: body, CategoryName: "Ideas"})
			if err != nil || updated.Number != 7 || !updated.MDContentPresent {
				t.Fatalf("updated = %+v, err = %v", updated, err)
			}
		})
	}
}

func TestUpdateDiscussionDoesNotRetry(t *testing.T) {
	networkErr := errors.New("connection lost after write")
	for _, tc := range []struct {
		name   string
		status int
		err    error
	}{
		{"network", 0, networkErr},
		{"unexpected created", http.StatusCreated, nil},
		{"unexpected no content", http.StatusNoContent, nil},
		{"rate limit", http.StatusTooManyRequests, nil},
		{"server error", http.StatusServiceUnavailable, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := NewClientWithHTTPClient("fake-token", &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodPut {
					t.Fatalf("method = %s", r.Method)
				}
				if tc.err != nil {
					return nil, tc.err
				}
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})})
			_, err := UpdateDiscussion(client, "alice", "demo", 7, DiscussionEditRequest{Title: "Updated", CategoryName: "Ideas"})
			if calls != 1 || err == nil {
				t.Fatalf("calls = %d, err = %v", calls, err)
			}
			if tc.err != nil && !errors.Is(err, tc.err) || tc.status != 0 && !IsHTTPStatus(err, tc.status) {
				t.Fatalf("underlying error lost: %v", err)
			}
		})
	}
}

func TestUpdateDiscussionDoesNotFollowRedirects(t *testing.T) {
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
			_, err := UpdateDiscussion(NewClientWithHTTPClient("fake-token", httpClient), "alice", "demo", 7, DiscussionEditRequest{Title: "Updated", CategoryName: "Ideas"})
			if !IsHTTPStatus(err, status) || calls != 1 || redirects != 0 {
				t.Fatalf("err = %v, calls = %d, redirects = %d", err, calls, redirects)
			}
			if err := httpClient.CheckRedirect(nil, nil); err != nil || redirects != 1 {
				t.Fatal("shared HTTP client redirect policy changed")
			}
		})
	}
}

func TestListDiscussionsEscapesOwnerAndRepo(t *testing.T) {
	var capturedPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		capturedPath = r.URL.EscapedPath()
		if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("per_page") != "100" {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `[{"number":1,"title":"Roadmap"}]`)
	})

	items, err := ListDiscussions(client, "org/name", "re po", 5)
	if err != nil {
		t.Fatal(err)
	}
	if capturedPath != "/repos/org%2Fname/re%20po/discuss" {
		t.Fatalf("path = %q", capturedPath)
	}
	if len(items) != 1 || items[0].Title != "Roadmap" {
		t.Fatalf("items = %#v", items)
	}
}

func TestGetDiscussionUsesExactSuccessStatus(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/repos/alice/demo/discuss/7" {
			t.Fatalf("path = %q", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"number":7,"title":"Roadmap"}`)
	})

	_, err := GetDiscussion(client, "alice", "demo", 7)
	if !IsHTTPStatus(err, http.StatusCreated) {
		t.Fatalf("error = %v, want HTTP 201", err)
	}
}

func TestListDiscussionCommentsAndRepliesEscapeIdentifiers(t *testing.T) {
	var paths []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		_, _ = io.WriteString(w, `[{"id":"c 1","reply_total":1}]`)
	})

	comments, err := ListDiscussionComments(client, "alice", "demo", 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 {
		t.Fatalf("comments = %#v", comments)
	}

	if _, err := ListDiscussionReplies(client, "alice", "demo", 7, "c 1", 1); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0] != "/repos/alice/demo/discuss/7/comment" || paths[1] != "/repos/alice/demo/discuss/7/comment/c%201/reply" {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestListDiscussionCommentsSkipsRequestWhenTotalIsZero(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(http.ResponseWriter, *http.Request) {
		requests++
	})
	comments, err := ListDiscussionComments(client, "alice", "demo", 7, 0)
	if err != nil || len(comments) != 0 || requests != 0 {
		t.Fatalf("comments = %#v, err = %v, requests = %d", comments, err, requests)
	}
}
