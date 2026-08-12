package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestIssueCollaborationCreate(t *testing.T) {
	tests := []struct {
		name     string
		assignee string
	}{
		{name: "with assignee", assignee: "bob"},
		{name: "without assignee", assignee: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s, want POST", r.Method)
				}
				wantPath := "/repos/alice/demo/issues"
				if r.URL.Path != wantPath {
					t.Fatalf("path = %s, want %s", r.URL.Path, wantPath)
				}
				if got := r.Header.Get("Content-Type"); got != "application/json" {
					t.Fatalf("Content-Type = %q", got)
				}

				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["title"] != "Test issue" {
					t.Fatalf("title = %q", body["title"])
				}
				if body["body"] != "Description" {
					t.Fatalf("body = %q", body["body"])
				}
				if tt.assignee != "" {
					if body["assignee"] != tt.assignee {
						t.Fatalf("assignee = %q, want %q", body["assignee"], tt.assignee)
					}
				} else {
					if _, ok := body["assignee"]; ok {
						t.Fatal("assignee field present when not provided")
					}
				}

				w.WriteHeader(http.StatusCreated)
				_, _ = io.WriteString(w, `{"number":"42","html_url":"https://atomgit.com/alice/demo/issues/42","title":"Test issue","state":"open"}`)
			})

			issue, err := CreateIssueWithAssignee(client, "alice", "demo", "Test issue", "Description", tt.assignee)
			if err != nil {
				t.Fatal(err)
			}
			if issue.GetNumber() != "42" {
				t.Fatalf("issue number = %q", issue.GetNumber())
			}
		})
	}
}

func TestIssueCollaborationCreateRejectsNon201(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
	})

	_, err := CreateIssueWithAssignee(client, "alice", "demo", "Test", "Body", "")
	if err == nil || !strings.Contains(err.Error(), "API error") {
		t.Fatalf("error = %v, want API error", err)
	}
}

func TestIssueCollaborationEditAssignee(t *testing.T) {
	tests := []struct {
		name          string
		body          *string
		assignee      string
		clearAssignee bool
		wantAssignee  string
		wantBody      string
	}{
		{name: "set assignee", assignee: "bob", clearAssignee: false, wantAssignee: "bob"},
		{name: "clear assignee", assignee: "", clearAssignee: true, wantAssignee: ""},
		{name: "set assignee with body", body: stringPointer("updated body"), assignee: "bob", wantAssignee: "bob", wantBody: "updated body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Fatalf("method = %s, want PATCH", r.Method)
				}
				wantPath := "/repos/alice/issues/42"
				if r.URL.Path != wantPath {
					t.Fatalf("path = %s, want %s", r.URL.Path, wantPath)
				}
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Fatal(err)
				}
				if got := r.FormValue("repo"); got != "demo" {
					t.Fatalf("repo = %q", got)
				}
				if got := r.FormValue("title"); got != "Issue title" {
					t.Fatalf("title = %q", got)
				}
				if got := r.FormValue("assignee"); got != tt.wantAssignee {
					t.Fatalf("assignee = %q, want %q", got, tt.wantAssignee)
				}
				if got := r.FormValue("body"); got != tt.wantBody {
					t.Fatalf("body = %q, want %q", got, tt.wantBody)
				}
				w.WriteHeader(http.StatusOK)
			})

			err := EditIssueAssignee(client, "alice", "demo", "42", "Issue title", tt.body, tt.assignee, tt.clearAssignee)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestIssueCollaborationEditRejectsNon200(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"bad"}`)
	})

	err := EditIssueAssignee(client, "alice", "demo", "42", "Issue title", nil, "bob", false)
	if err == nil || !strings.Contains(err.Error(), "API error") {
		t.Fatalf("error = %v, want API error", err)
	}
}

func stringPointer(value string) *string {
	return &value
}

func TestIssueCollaborationLinkedPRs(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantLen int
	}{
		{
			name:    "populated array",
			body:    `[{"id":1,"number":7,"title":"Fix bug","body":"details","state":"open","html_url":"https://example.test/alice/demo/pulls/7","url":"https://api.example.test/pulls/7","created_at":"2026-01-01","updated_at":"2026-01-02"}]`,
			wantLen: 1,
		},
		{
			name:    "empty array",
			body:    `[]`,
			wantLen: 0,
		},
		{
			name:    "null response",
			body:    `null`,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				wantPath := "/repos/alice/demo/issues/42/pull_requests"
				if r.URL.Path != wantPath {
					t.Fatalf("path = %s, want %s", r.URL.Path, wantPath)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, tt.body)
			})

			prs, err := ListIssueLinkedPullRequests(client, "alice", "demo", "42")
			if err != nil {
				t.Fatal(err)
			}
			if len(prs) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(prs), tt.wantLen)
			}
			if prs == nil {
				t.Fatal("result is nil, want non-nil slice")
			}
		})
	}
}

func TestIssueCollaborationLinkedPRsRejectsNon200(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := ListIssueLinkedPullRequests(client, "alice", "demo", "42")
	if err == nil || !strings.Contains(err.Error(), "API error") {
		t.Fatalf("error = %v, want API error", err)
	}
}

func TestIssueCollaborationRelatedBranchesGet(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantLen int
	}{
		{
			name:    "populated array",
			body:    `["main","feature/x"]`,
			wantLen: 2,
		},
		{
			name:    "empty array",
			body:    `[]`,
			wantLen: 0,
		},
		{
			name:    "null response",
			body:    `null`,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				wantPath := "/repos/alice/demo/issues/42/related_branches"
				if r.URL.Path != wantPath {
					t.Fatalf("path = %s, want %s", r.URL.Path, wantPath)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, tt.body)
			})

			branches, err := ListIssueRelatedBranches(client, "alice", "demo", "42")
			if err != nil {
				t.Fatal(err)
			}
			if len(branches) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(branches), tt.wantLen)
			}
			if branches == nil {
				t.Fatal("result is nil, want non-nil slice")
			}
		})
	}
}

func TestIssueCollaborationRelatedBranchesPut(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		wantPath := "/repos/alice/demo/issues/42/related_branches"
		if r.URL.Path != wantPath {
			t.Fatalf("path = %s, want %s", r.URL.Path, wantPath)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}

		var body RelatedBranchesRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.BranchNames) != 2 || body.BranchNames[0] != "main" || body.BranchNames[1] != "feature/x" {
			t.Fatalf("branch_names = %#v", body.BranchNames)
		}

		w.WriteHeader(http.StatusOK)
	})

	err := UpdateIssueRelatedBranches(client, "alice", "demo", "42", []string{"main", "feature/x"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIssueCollaborationRelatedBranchesPutEncodesEmptyArray(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := string(body["branch_names"]); got != "[]" {
			t.Fatalf("branch_names = %s, want []", got)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := UpdateIssueRelatedBranches(client, "alice", "demo", "42", nil); err != nil {
		t.Fatal(err)
	}
}

func TestIssueCollaborationRelatedBranchesPutRejectsNon200(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"message":"conflict"}`)
	})

	err := UpdateIssueRelatedBranches(client, "alice", "demo", "42", []string{"main"})
	if err == nil || !strings.Contains(err.Error(), "the remote state may have changed") {
		t.Fatalf("error = %v, want 'the remote state may have changed'", err)
	}
}

func TestIssueCollaborationEscapedIdentifiers(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/repos/my-org/my repo/issues/ABC#123/pull_requests"
		if r.URL.Path != wantPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, wantPath)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `[]`)
	})

	_, err := ListIssueLinkedPullRequests(client, "my-org", "my repo", "ABC#123")
	if err != nil {
		t.Fatal(err)
	}
}

func TestIssueCollaborationUncertainMutationError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})

	err := UpdateIssueRelatedBranches(client, "alice", "demo", "42", []string{"main"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "the remote state may have changed") {
		t.Fatalf("error = %v, want 'the remote state may have changed'", err)
	}
}

func TestIssueCollaborationCreateRejectsWrongStatus(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})

	_, err := CreateIssueWithAssignee(client, "alice", "demo", "test", "body", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create issue with assignee") {
		t.Fatalf("error = %v, want 'create issue with assignee' wrapping", err)
	}
}

func TestIssueCollaborationEditErrorWrapping(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"message":"denied"}`)
	})

	err := EditIssueAssignee(client, "alice", "demo", "42", "title", nil, "bob", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "edit issue assignee") {
		t.Fatalf("error = %v, want 'edit issue assignee' wrapping", err)
	}
}

func TestIssueCollaborationListLinkedPRsErrorWrapping(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := ListIssueLinkedPullRequests(client, "alice", "demo", "42")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "list issue linked pull requests") {
		t.Fatalf("error = %v, want 'list issue linked pull requests' wrapping", err)
	}
}

func TestIssueCollaborationListBranchesErrorWrapping(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := ListIssueRelatedBranches(client, "alice", "demo", "42")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "list issue related branches") {
		t.Fatalf("error = %v, want 'list issue related branches' wrapping", err)
	}
}
