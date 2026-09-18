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

func TestListCommitCommentsPaginatesAndEscapesRef(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if got := r.URL.EscapedPath(); got != "/repos/alice/demo/commits/feature%2Fone/comments" {
			t.Fatalf("path = %q", got)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("per_page = %q", got)
		}
		if got := r.URL.Query().Get("page"); got != fmt.Sprint(requests) {
			t.Fatalf("page = %q, want %d", got, requests)
		}
		if requests == 1 {
			writeJSON(t, w, json.RawMessage(commentsBody(1, 100)))
			return
		}
		writeJSON(t, w, json.RawMessage(commentsBody(101, 2)))
	})

	comments, err := ListCommitComments(client, "alice", "demo", "feature/one", 102)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if len(comments) != 102 {
		t.Fatalf("len(comments) = %d, want 102", len(comments))
	}
	if comments[0].ID != "1" || comments[101].ID != "102" {
		t.Fatalf("unexpected comment IDs: %s..%s", comments[0].ID, comments[101].ID)
	}
}

func TestListCommitCommentsHonorsLimitAndEmptyPage(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeJSON(t, w, json.RawMessage(commentsBody(1, 3)))
	})

	comments, err := ListCommitComments(client, "alice", "demo", "abc1234", 5)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1 (short page stops the walk)", requests)
	}
	if len(comments) != 3 {
		t.Fatalf("len(comments) = %d, want 3", len(comments))
	}
}

func commentsBody(first, count int) string {
	items := make([]string, count)
	for i := range count {
		items[i] = fmt.Sprintf(`{"id":%d,"body":"note %d","user":{"id":1001,"login":"alice"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T10:00:00+08:00"}`, first+i, first+i)
	}
	return "[" + strings.Join(items, ",") + "]"
}

func TestCreateCommitCommentSendsBodyOnly(t *testing.T) {
	var method, path, body string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.WriteHeader(http.StatusCreated)
		writeJSON(t, w, json.RawMessage(`{"id":"9001","body":"多行\n评论","created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T10:00:00+08:00"}`))
	})

	comment, err := CreateCommitComment(client, "alice", "demo", "abc1234", "多行\n评论")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/repos/alice/demo/commits/abc1234/comments" {
		t.Fatalf("request = %s %s", method, path)
	}
	if body != `{"body":"多行\n评论"}` {
		t.Fatalf("body = %q", body)
	}
	if comment.GetID() != "9001" {
		t.Fatalf("id = %q", comment.GetID())
	}
}

func TestGetCommitCommentMapsNoteTypeRejection(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(t, w, json.RawMessage(`{"error_code":400,"error_code_name":"UN_KNOW","error_message":"Note type is not correct.","trace_id":"t"}`))
	})

	_, err := GetCommitComment(client, "alice", "demo", "7")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNotCommitComment) {
		t.Fatalf("err = %v, want ErrNotCommitComment", err)
	}
	if !strings.Contains(err.Error(), "comment #7") {
		t.Fatalf("error missing comment context: %v", err)
	}
}

func TestGetCommitCommentSurfacesNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(t, w, json.RawMessage(`{"error_code":404,"error_code_name":"UN_KNOW","error_message":"note not found by noteId","trace_id":"t"}`))
	})

	_, err := GetCommitComment(client, "alice", "demo", "7")
	if err == nil || !strings.Contains(err.Error(), "get commit comment #7") || !strings.Contains(err.Error(), "404") {
		t.Fatalf("err = %v", err)
	}
	if errors.Is(err, ErrNotCommitComment) {
		t.Fatalf("404 must not map to ErrNotCommitComment: %v", err)
	}
}

func TestGetCommitCommentOtherBadRequestNotMapped(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(t, w, json.RawMessage(`{"error_code":400,"error_message":"Something else"}`))
	})

	_, err := GetCommitComment(client, "alice", "demo", "7")
	if err == nil || errors.Is(err, ErrNotCommitComment) {
		t.Fatalf("err = %v, want a generic 400 error", err)
	}
}

func TestUpdateCommitCommentPatchesBodyOnly(t *testing.T) {
	var method, body string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		if got := r.URL.Path; got != "/repos/alice/demo/comments/7" {
			t.Fatalf("path = %q", got)
		}
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		writeJSON(t, w, json.RawMessage(`{"id":7,"body":"new","user":{"id":1001,"login":"alice"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T11:00:00+08:00"}`))
	})

	comment, err := UpdateCommitComment(client, "alice", "demo", "7", "new")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPatch || body != `{"body":"new"}` {
		t.Fatalf("request = %s %q", method, body)
	}
	if comment.ID != "7" || comment.UpdatedAt != "2026-09-15T11:00:00+08:00" {
		t.Fatalf("comment = %+v", comment)
	}
}

func TestDeleteCommitCommentSucceedsOnNoContent(t *testing.T) {
	called := false
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete || r.URL.Path != "/repos/alice/demo/comments/7" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeleteCommitComment(client, "alice", "demo", "7"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("delete was not sent")
	}
}

func TestCommitCommentDecodesDocumentedUserShapes(t *testing.T) {
	numeric := decodeCommitComment(t, `{"id":7,"user":{"id":1001,"login":"alice","name":"Alice"}}`)
	if numeric.User.ID != "1001" || numeric.User.Login != "alice" || numeric.User.Name != "Alice" {
		t.Fatalf("numeric user = %+v", numeric.User)
	}
	textual := decodeCommitComment(t, `{"id":8,"user":{"id":"u1001","login":"bob","name":"Bob"}}`)
	if textual.User.ID != "u1001" || textual.User.Login != "bob" || textual.User.Name != "Bob" {
		t.Fatalf("string user = %+v", textual.User)
	}
	missing := decodeCommitComment(t, `{"id":9}`)
	if missing.User.ID != "" || missing.User.Login != "" {
		t.Fatalf("missing user = %+v", missing.User)
	}
	nullID := decodeCommitComment(t, `{"id":10,"user":{"id":null,"login":"carol"}}`)
	if nullID.User.ID != "" || nullID.User.Login != "carol" {
		t.Fatalf("null-id user = %+v", nullID.User)
	}
}

func decodeCommitComment(t *testing.T, response string) CommitComment {
	t.Helper()
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, json.RawMessage(response))
	})
	comment, err := GetCommitComment(client, "alice", "demo", "7")
	if err != nil {
		t.Fatal(err)
	}
	return comment
}

func TestCommitCommentEndpointsAcceptOpaqueStringIDs(t *testing.T) {
	// The create endpoint's documented success example returns the id as the
	// string "12312sadsa"; that exact identifier must flow into the
	// get/update/delete paths unchanged.
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/repos/alice/demo/comments/12312sadsa" {
			t.Fatalf("%s path = %q", r.Method, got)
		}
		switch r.Method {
		case http.MethodGet, http.MethodPatch:
			writeJSON(t, w, json.RawMessage(`{"id":"12312sadsa","body":"LGTM","user":{"id":1001,"login":"alice"}}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})

	if _, err := GetCommitComment(client, "alice", "demo", "12312sadsa"); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateCommitComment(client, "alice", "demo", "12312sadsa", "LGTM"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteCommitComment(client, "alice", "demo", "12312sadsa"); err != nil {
		t.Fatal(err)
	}
}

func TestCommitCommentIDEscapedInPath(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.EscapedPath(); got != "/repos/alice/demo/comments/a%20b%2Fc" {
			t.Fatalf("escaped path = %q", got)
		}
		writeJSON(t, w, json.RawMessage(`{"id":"a b/c","body":"x"}`))
	})

	comment, err := GetCommitComment(client, "alice", "demo", "a b/c")
	if err != nil {
		t.Fatal(err)
	}
	if comment.ID != "a b/c" {
		t.Fatalf("id = %q", comment.ID)
	}
}

func TestCommitCommentDecodesDocumentedIDShapes(t *testing.T) {
	numeric := decodeCommitComment(t, `{"id":7,"body":"x"}`)
	if numeric.ID != "7" {
		t.Fatalf("numeric id = %q", numeric.ID)
	}
	textual := decodeCommitComment(t, `{"id":"12312sadsa","body":"x"}`)
	if textual.ID != "12312sadsa" {
		t.Fatalf("string id = %q", textual.ID)
	}
	missing := decodeCommitComment(t, `{"body":"x"}`)
	if missing.ID != "" {
		t.Fatalf("missing id = %q", missing.ID)
	}
	null := decodeCommitComment(t, `{"id":null,"body":"x"}`)
	if null.ID != "" {
		t.Fatalf("null id = %q", null.ID)
	}
}
