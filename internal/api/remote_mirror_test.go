package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestListRepositoryPushRemoteMirrorsPaginatesAndSanitizes(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/repos/team/demo/push_remote_mirrors" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != fmt.Sprint(requests) {
			t.Fatalf("page = %q, want %d", got, requests)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("per_page = %q, want 100", got)
		}

		count := 100
		start := 0
		if requests == 2 {
			count = 2
			start = 100
		}
		mirrors := make([]map[string]any, count)
		for index := range mirrors {
			id := start + index + 1
			mirrors[index] = map[string]any{
				"id":            id,
				"update_status": "finished",
				"url":           fmt.Sprintf("https://user:secret-%d@example.com/team/repo.git?access_token=hidden#fragment", id),
			}
		}
		if err := json.NewEncoder(w).Encode(mirrors); err != nil {
			t.Fatal(err)
		}
	})

	mirrors, err := ListRepositoryPushRemoteMirrors(client, "team", "demo", 101)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || len(mirrors) != 101 {
		t.Fatalf("requests = %d, len = %d; want 2, 101", requests, len(mirrors))
	}
	if mirrors[100].ID == nil || *mirrors[100].ID != 101 {
		t.Fatalf("last mirror ID = %v, want 101", mirrors[100].ID)
	}
	if mirrors[0].URL == nil || *mirrors[0].URL != "https://example.com/team/repo.git" {
		t.Fatalf("sanitized URL = %v", mirrors[0].URL)
	}
}

func TestGetRepositoryRemoteMirrorPreservesOptionalValuesAndSanitizes(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/team/demo/repo_remote_mirror" || r.URL.RawQuery != "" {
			t.Fatalf("request URL = %s", r.URL.String())
		}
		fmt.Fprint(w, `{
  "id": 7,
  "project_id": 42,
  "update_status": "failed",
  "url": "ssh://git:private@example.com/team/demo.git?token=hidden#fragment",
  "number_of_failures": 0,
  "mirroring_enabled": false,
  "is_private": true,
  "last_error": "push to https://alice:secret@example.net/team/demo.git?private_token=hidden failed",
  "force": false
}`)
	})

	mirror, err := GetRepositoryRemoteMirror(client, "team", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if mirror.URL == nil || *mirror.URL != "ssh://example.com/team/demo.git" {
		t.Fatalf("URL = %v", mirror.URL)
	}
	if mirror.LastError == nil || *mirror.LastError != "push to https://example.net/team/demo.git failed" {
		t.Fatalf("LastError = %v", mirror.LastError)
	}
	if mirror.MirroringEnabled == nil || *mirror.MirroringEnabled {
		t.Fatalf("MirroringEnabled = %v, want returned false", mirror.MirroringEnabled)
	}
	if mirror.NumberOfFailures == nil || *mirror.NumberOfFailures != 0 {
		t.Fatalf("NumberOfFailures = %v, want returned zero", mirror.NumberOfFailures)
	}
	if mirror.CreatedAt != nil {
		t.Fatalf("CreatedAt = %v, want absent", mirror.CreatedAt)
	}
}

func TestGetRepositoryRemoteMirrorRedactsStandaloneCredentials(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/team/demo/repo_remote_mirror" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{
  "last_error": "authentication failed: password=mirror-password-123; token=mirror-token-456",
  "message": "request rejected for Authorization: Bearer mirror-bearer-789"
}`)
	})

	mirror, err := GetRepositoryRemoteMirror(client, "team", "demo")
	if err != nil {
		t.Fatal(err)
	}
	for field, value := range map[string]*string{
		"last_error": mirror.LastError,
		"message":    mirror.Message,
	} {
		if value == nil {
			t.Fatalf("%s is absent", field)
		}
		for _, secret := range []string{"mirror-password-123", "mirror-token-456", "mirror-bearer-789"} {
			if strings.Contains(*value, secret) {
				t.Fatalf("%s leaked %q: %q", field, secret, *value)
			}
		}
		if !strings.Contains(*value, "<redacted>") {
			t.Fatalf("%s did not mark redacted credentials: %q", field, *value)
		}
	}
}

func TestRemoteMirrorErrorsRedactCredentialBearingURLs(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"message":"failed to reach https://user:top-secret@example.com/repo.git?access_token=hidden#details"}`)
	})

	_, err := GetRepositoryRemoteMirror(client, "team", "demo")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsHTTPStatus(err, http.StatusInternalServerError) {
		t.Fatalf("error no longer unwraps to HTTP status: %v", err)
	}
	for _, secret := range []string{"user", "top-secret", "access_token", "hidden", "details"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked %q: %v", secret, err)
		}
	}
	if !strings.Contains(err.Error(), "https://example.com/repo.git") {
		t.Fatalf("error lost sanitized destination: %v", err)
	}
}

func TestSanitizeRemoteMirrorURLHandlesSCPStyleUserinfo(t *testing.T) {
	if got := sanitizeRemoteMirrorURL("deploy@example.com:team/repo.git"); got != "example.com:team/repo.git" {
		t.Fatalf("sanitizeRemoteMirrorURL() = %q", got)
	}
	if got := sanitizeRemoteMirrorText("failed git@example.com:team/repo.git"); got != "failed example.com:team/repo.git" {
		t.Fatalf("sanitizeRemoteMirrorText() = %q", got)
	}
	if got := sanitizeRemoteMirrorURL("https://user:secret@example.com/bad%zz?token=hidden#fragment"); got != "https://example.com/bad%zz" {
		t.Fatalf("malformed sanitizeRemoteMirrorURL() = %q", got)
	}
	if got := sanitizeRemoteMirrorText(`failed https:\/\/user:secret@example.com/repo.git?token=hidden`); got != "failed https://example.com/repo.git" {
		t.Fatalf("escaped sanitizeRemoteMirrorText() = %q", got)
	}
}
