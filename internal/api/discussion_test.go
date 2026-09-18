package api

import (
	"io"
	"net/http"
	"testing"
)

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
