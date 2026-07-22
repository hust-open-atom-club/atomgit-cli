package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func newReleaseTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	return newTestClient(t, handler)
}

func singleRelease(tag string) Release {
	return Release{
		TagName:       tag,
		Name:          tag,
		Body:          "",
		ReleaseStatus: "latest",
		Author:        ReleaseAuthor{Login: "mudongliang", Type: "User"},
		Assets:        []ReleaseAsset{},
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}

func TestListReleasesPaging(t *testing.T) {
	requests := 0
	client := newReleaseTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("per_page = %q, want 100", got)
		}
		if got := r.URL.Query().Get("direction"); got != "desc" {
			t.Fatalf("direction = %q, want desc", got)
		}
		if got := r.URL.Query().Get("page"); got != fmt.Sprint(requests) {
			t.Fatalf("page = %q, want %d", got, requests)
		}

		items := make([]Release, 100)
		for i := range items {
			items[i] = singleRelease(fmt.Sprintf("v%d.%d", requests, i))
		}
		writeJSON(t, w, items)
	})

	items, err := ListReleases(client, "owner", "repo", 150)
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
	if len(items) != 150 {
		t.Fatalf("len = %d, want 150", len(items))
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (stop at limit)", requests)
	}
}

func TestListReleasesStopsAtShortPage(t *testing.T) {
	requests := 0
	client := newReleaseTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeJSON(t, w, []Release{singleRelease("v0.1"), singleRelease("v0.2")})
	})

	items, err := ListReleases(client, "owner", "repo", 30)
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
	if len(items) != 2 || requests != 1 {
		t.Fatalf("items = %d, requests = %d", len(items), requests)
	}
}

func TestListReleasesEscapesPath(t *testing.T) {
	client := newReleaseTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.EscapedPath(); !strings.Contains(got, "/repos/owner%20with%20space/repo%2Fslash/releases") {
			t.Fatalf("path not escaped: %s", got)
		}
		writeJSON(t, w, []Release{})
	})

	if _, err := ListReleases(client, "owner with space", "repo/slash", 1); err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
}

func TestListReleasesRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []int{0, -1, -10} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			if _, err := ListReleases(NewClient("t"), "o", "r", limit); err == nil {
				t.Fatal("expected error for invalid limit")
			}
		})
	}
}

func TestGetReleaseByTag(t *testing.T) {
	var gotPath, gotMethod string
	client := newReleaseTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotMethod = r.Method
		writeJSON(t, w, singleRelease("v1.0.0"))
	})

	release, err := GetReleaseByTag(client, "owner", "repo", "v1.0.0")
	if err != nil {
		t.Fatalf("GetReleaseByTag: %v", err)
	}
	if release.TagName != "v1.0.0" {
		t.Fatalf("TagName = %q", release.TagName)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/repos/owner/repo/releases/tags/v1.0.0" {
		t.Fatalf("path = %s", gotPath)
	}
}

func TestGetReleaseByTagEscapesTag(t *testing.T) {
	client := newReleaseTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.EscapedPath(); got != "/repos/o/r/releases/tags/v%2F1" {
			t.Fatalf("tag not escaped: %s", got)
		}
		writeJSON(t, w, singleRelease("v/1"))
	})

	if _, err := GetReleaseByTag(client, "o", "r", "v/1"); err != nil {
		t.Fatalf("GetReleaseByTag: %v", err)
	}
}

func TestCreateRelease(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody CreateReleaseRequest
	client := newReleaseTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(t, w, singleRelease("v2.0.0"))
	})

	release, err := CreateRelease(client, "owner", "repo", CreateReleaseRequest{
		TagName: "v2.0.0", Name: "v2.0.0", Body: "notes",
	})
	if err != nil {
		t.Fatalf("CreateRelease: %v", err)
	}
	if release.TagName != "v2.0.0" {
		t.Fatalf("TagName = %q", release.TagName)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/repos/owner/repo/releases" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotBody.TagName != "v2.0.0" || gotBody.Body != "notes" {
		t.Fatalf("body = %#v", gotBody)
	}
}

func TestCreateReleaseAcceptsBothSuccessCodes(t *testing.T) {
	// Accept both the documented 200 and conventional 201 response.
	tests := []struct {
		name   string
		status int
	}{
		{name: "200 OK", status: http.StatusOK},
		{name: "201 Created", status: http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newReleaseTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				writeJSON(t, w, singleRelease("v2.0.0"))
			})

			if _, err := CreateRelease(client, "o", "r", CreateReleaseRequest{TagName: "v2", Name: "v2", Body: ""}); err != nil {
				t.Fatalf("CreateRelease(%d): %v", tt.status, err)
			}
		})
	}
}

func TestUpdateRelease(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody UpdateReleaseRequest
	client := newReleaseTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		writeJSON(t, w, singleRelease("v1.0.0"))
	})

	release, err := UpdateRelease(client, "owner", "repo", "v1.0.0", UpdateReleaseRequest{
		Name: "v1.0.0", Body: "updated", ReleaseStatus: ReleaseStatusPre,
	})
	if err != nil {
		t.Fatalf("UpdateRelease: %v", err)
	}
	if release.TagName != "v1.0.0" {
		t.Fatalf("TagName = %q", release.TagName)
	}
	if gotMethod != http.MethodPatch {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/repos/owner/repo/releases/v1.0.0" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotBody.Body != "updated" || gotBody.ReleaseStatus != "pre" {
		t.Fatalf("body = %#v", gotBody)
	}
}

func releaseMetadataOps(client *Client) map[string]func() error {
	return map[string]func() error{
		"ListReleases":    func() error { _, err := ListReleases(client, "o", "r", 10); return err },
		"GetReleaseByTag": func() error { _, err := GetReleaseByTag(client, "o", "r", "v1"); return err },
		"CreateRelease": func() error {
			_, err := CreateRelease(client, "o", "r", CreateReleaseRequest{TagName: "v1", Name: "v1", Body: ""})
			return err
		},
		"UpdateRelease": func() error {
			_, err := UpdateRelease(client, "o", "r", "v1", UpdateReleaseRequest{Name: "v1", Body: "x"})
			return err
		},
	}
}

func TestReleaseMetadataErrors(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		errSubstr string
	}{
		{name: "not found", status: http.StatusNotFound, body: `{"error_code":404,"error_message":"not found"}`, errSubstr: "404"},
		{name: "conflict on duplicate tag", status: http.StatusConflict, body: `{"error_code":409,"error_message":"tag exists"}`, errSubstr: "409"},
		{name: "forbidden", status: http.StatusForbidden, body: "denied", errSubstr: "403"},
		{name: "rate limited", status: http.StatusTooManyRequests, body: "slow down", errSubstr: "429"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newReleaseTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			})

			for opName, op := range releaseMetadataOps(client) {
				t.Run(opName, func(t *testing.T) {
					err := op()
					if err == nil {
						t.Fatalf("expected error, got nil")
					}
					if !strings.Contains(err.Error(), tt.errSubstr) {
						t.Fatalf("error %q does not contain %q", err.Error(), tt.errSubstr)
					}
				})
			}
		})
	}
}
