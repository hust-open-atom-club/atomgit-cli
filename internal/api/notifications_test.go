package api

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

func notificationPageJSON(total int, list string) string {
	return `{"total":` + strconv.Itoa(total) + `,"list":[` + list + `]}`
}

const sampleNotification = `{
	"id": "292ecbec857e4f27b426d66f2157938c",
	"content": "fix: normalize unauthenticated error handling",
	"type": "merge_requests_open",
	"unread": true,
	"update_at": "2026-08-14T23:30:20+08:00",
	"html_url": "https://gitcode.com/hust-open-atom-club/atomgit-cli/merge_requests/78",
	"actor": {"id": 6403465, "login": "mudongliang", "name": "mudongliang"},
	"repository": {"id": 10342125, "full_name": "hust-open-atom-club/atomgit-cli", "human_name": "华中科技大学开放原子开源俱乐部 / atomgit-cli", "url": "https://gitcode.com/hust-open-atom-club/atomgit-cli"}
}`

func TestListNotificationsBuildsQueryAndDecodesEnvelope(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(notificationPageJSON(1, sampleNotification)))
	})

	got, err := ListNotifications(client, "atom club", "atom/git-cli", NotificationListOptions{
		Limit:      30,
		UnreadOnly: true,
		Since:      "2026-08-14T00:00:00+08:00",
		Before:     "2026-08-15T00:00:00+08:00",
	})
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}

	const wantPath = "/repos/atom%20club/atom%2Fgit-cli/notifications"
	if gotPath != wantPath {
		t.Fatalf("path = %q, want %q", gotPath, wantPath)
	}
	if gotQuery.Get("unread") != "true" {
		t.Fatalf("unread = %q, want true", gotQuery.Get("unread"))
	}
	// RFC 3339 timestamps carry "+" which must stay percent-encoded.
	if gotQuery.Get("since") != "2026-08-14T00:00:00+08:00" {
		t.Fatalf("since = %q", gotQuery.Get("since"))
	}
	if gotQuery.Get("before") != "2026-08-15T00:00:00+08:00" {
		t.Fatalf("before = %q", gotQuery.Get("before"))
	}
	if gotQuery.Get("page") != "1" || gotQuery.Get("per_page") != "100" {
		t.Fatalf("page/per_page = %q/%q, want 1/100", gotQuery.Get("page"), gotQuery.Get("per_page"))
	}

	if len(got) != 1 {
		t.Fatalf("len(notifications) = %d, want 1", len(got))
	}
	n := got[0]
	if n.ID != "292ecbec857e4f27b426d66f2157938c" || n.Type != "merge_requests_open" || !n.Unread {
		t.Fatalf("decoded notification = %+v", n)
	}
	if n.Content != "fix: normalize unauthenticated error handling" {
		t.Fatalf("content = %q", n.Content)
	}
	if n.Actor == nil || n.Actor.Login != "mudongliang" {
		t.Fatalf("actor = %+v", n.Actor)
	}
	if n.Repository == nil || n.Repository.FullName != "hust-open-atom-club/atomgit-cli" {
		t.Fatalf("repository = %+v", n.Repository)
	}
}

func TestListNotificationsDefaultsToListEverything(t *testing.T) {
	var gotQuery url.Values
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":0,"list":[]}`))
	})

	if _, err := ListNotifications(client, "owner", "repo", NotificationListOptions{Limit: 30}); err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	// The server defaults to unread-only, so listing everything must send
	// unread=false explicitly.
	if gotQuery.Get("unread") != "false" {
		t.Fatalf("unread = %q, want false", gotQuery.Get("unread"))
	}
}

func TestListNotificationsTypeFilterAndPaging(t *testing.T) {
	// Page 1 must be full (notificationsPerPage items) so the walk continues;
	// it carries one type match plus filler. Page 2 is short and carries the
	// second match, which ends the walk.
	filler := ""
	for i := 0; i < notificationsPerPage-1; i++ {
		if i > 0 {
			filler += ","
		}
		filler += `{"id":"filler` + strconv.Itoa(i) + `","type":"issue_open"}`
	}
	pages := map[string]string{
		"1": notificationPageJSON(notificationsPerPage, `{"id":"a","type":"merge_requests_open"}`+","+filler),
		"2": notificationPageJSON(notificationsPerPage+2, `{"id":"b","type":"issue_open"},{"id":"c","type":"merge_requests_open"}`),
	}
	var requestedPages []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		requestedPages = append(requestedPages, page)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(pages[page]))
	})

	got, err := ListNotifications(client, "owner", "repo", NotificationListOptions{
		Limit: 2,
		Type:  "merge_requests_open",
	})
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}

	if len(requestedPages) != 2 || requestedPages[0] != "1" || requestedPages[1] != "2" {
		t.Fatalf("requested pages = %v, want [1 2]", requestedPages)
	}
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "c" {
		t.Fatalf("filtered notifications = %+v, want ids [a c]", got)
	}
}

func TestListNotificationsStopsOnEmptyPageAndTruncatesToLimit(t *testing.T) {
	fullPage := ""
	for i := 0; i < notificationsPerPage; i++ {
		if i > 0 {
			fullPage += ","
		}
		fullPage += `{"id":"x"}`
	}
	var requests int
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "1" {
			_, _ = w.Write([]byte(notificationPageJSON(notificationsPerPage, fullPage)))
			return
		}
		_, _ = w.Write([]byte(`{"total":0,"list":[]}`))
	})

	got, err := ListNotifications(client, "owner", "repo", NotificationListOptions{Limit: notificationsPerPage + 5})
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if len(got) != notificationsPerPage {
		t.Fatalf("len(notifications) = %d, want %d", len(got), notificationsPerPage)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (full page then empty page)", requests)
	}
}

func TestListNotificationsRejectsNonPositiveLimit(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request expected for an invalid limit")
	})
	if _, err := ListNotifications(client, "owner", "repo", NotificationListOptions{Limit: 0}); err == nil ||
		err.Error() != "invalid limit: 0 (must be positive)" {
		t.Fatalf("err = %v, want invalid limit error", err)
	}
}

func TestMarkNotificationsReadSendsExactFormIDs(t *testing.T) {
	var gotMethod, gotPath, gotContentType string
	var gotBody string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		// The endpoint answers 200 with an empty body.
		w.WriteHeader(http.StatusOK)
	})

	if err := MarkNotificationsRead(client, "owner", "repo", []string{"id-one", "id-two"}); err != nil {
		t.Fatalf("MarkNotificationsRead: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/repos/owner/repo/notifications" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("content type = %q", gotContentType)
	}
	parsed, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatalf("parse form body %q: %v", gotBody, err)
	}
	if got := parsed["ids"]; len(got) != 2 || got[0] != "id-one" || got[1] != "id-two" {
		t.Fatalf("ids = %v, want [id-one id-two]", got)
	}
}

func TestMarkNotificationsReadRejectsEmptyIDs(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request expected without IDs")
	})
	if err := MarkNotificationsRead(client, "owner", "repo", nil); err == nil ||
		err.Error() != "at least one notification ID is required" {
		t.Fatalf("err = %v, want empty-ID error", err)
	}
}

func TestMarkNotificationsReadSurfacesAPIError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	})
	err := MarkNotificationsRead(client, "owner", "repo", []string{"id-one"})
	if err == nil || !IsHTTPStatus(err, http.StatusForbidden) {
		t.Fatalf("err = %v, want 403 API error", err)
	}
}
