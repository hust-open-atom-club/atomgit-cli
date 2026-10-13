package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPullRequestDetailsCommits(t *testing.T) {
	t.Run("exact HTTP 200 success", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"sha":"abc1234","html_url":"https://atomgit.com/a/b/commit/abc1234","commit":{"message":"fix bug","author":{"name":"Alice","email":"a@b.com","date":"2024-01-01T00:00:00Z"}}}]`)
		})
		commits, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(commits) != 1 || commits[0].SHA != "abc1234" {
			t.Fatalf("commits = %#v", commits)
		}
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"message":"not found"}`)
		})
		_, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "404") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("201 status is rejected", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "201") {
			t.Fatalf("error = %v, want rejection of HTTP 201", err)
		}
	})

	t.Run("path escaped identifiers", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			raw := r.URL.RequestURI()
			if !strings.Contains(raw, "%2F") {
				t.Errorf("owner/repo slashes not escaped in path: %s", raw)
			}
			if !strings.Contains(raw, "%231") {
				t.Errorf("PR number hash not escaped in path: %s", raw)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestCommits(client, "special/owner", "special/repo", "#1", 30)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("pagination parameters in URL", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			page := r.URL.Query().Get("page")
			perPage := r.URL.Query().Get("per_page")
			if page != "1" || perPage != "100" {
				t.Errorf("page=%s per_page=%s, want page=1 per_page=100", page, perPage)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("GET-only method", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("empty array response", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		commits, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(commits) != 0 {
			t.Fatalf("len(commits) = %d, want 0", len(commits))
		}
	})

	t.Run("single commit with all fields", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"sha":"abc1234567890","html_url":"https://atomgit.com/a/b/commit/abc1234","commit":{"message":"fix: resolve issue\n\nbody text","author":{"name":"Alice","email":"alice@example.com","date":"2024-06-15T10:30:00Z","login":"alice"}}}]`)
		})
		commits, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(commits) != 1 {
			t.Fatalf("len(commits) = %d, want 1", len(commits))
		}
		c := commits[0]
		if c.SHA != "abc1234567890" || c.Commit.Message != "fix: resolve issue\n\nbody text" || c.Commit.Author.Login != "alice" {
			t.Fatalf("commit = %#v", c)
		}
	})

	t.Run("multiple commits preserve server order", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"sha":"third","html_url":"","commit":{"message":"c","author":{"name":"","email":"","date":""}}},{"sha":"second","html_url":"","commit":{"message":"b","author":{"name":"","email":"","date":""}}},{"sha":"first","html_url":"","commit":{"message":"a","author":{"name":"","email":"","date":""}}}]`)
		})
		commits, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(commits) != 3 || commits[0].SHA != "third" || commits[2].SHA != "first" {
			t.Fatalf("commits = %#v", commits)
		}
	})

	t.Run("null optional fields handled", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"sha":"abc","html_url":null,"commit":{"message":"msg","author":{"name":null,"email":null,"date":null}}}]`)
		})
		commits, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(commits) != 1 || commits[0].SHA != "abc" {
			t.Fatalf("commits = %#v", commits)
		}
	})

	t.Run("zero limit rejected", func(t *testing.T) {
		_, err := ListPullRequestCommits(nil, "owner", "repo", "42", 0)
		if err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("negative limit rejected", func(t *testing.T) {
		_, err := ListPullRequestCommits(nil, "owner", "repo", "42", -1)
		if err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestPullRequestDetailsFiles(t *testing.T) {
	t.Run("exact HTTP 200 success", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"sha":"abc","filename":"main.go","additions":10,"deletions":2,"too_large":false,"blob_url":"https://...","raw_url":"https://...","patch":{"old_path":"main.go","new_path":"main.go","added_lines":10,"removed_lines":2,"too_large":false,"new_file":false,"renamed_file":false,"deleted_file":false}}]`)
		})
		files, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 1 || files[0].Filename != "main.go" {
			t.Fatalf("files = %#v", files)
		}
		if files[0].GetChangeType() != "modified" {
			t.Fatalf("change type = %q, want modified", files[0].GetChangeType())
		}
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"message":"not found"}`)
		})
		_, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err == nil || !strings.Contains(err.Error(), "404") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("path escaped identifiers", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			raw := r.URL.RequestURI()
			if !strings.Contains(raw, "%2F") {
				t.Errorf("owner slash not escaped in path: %s", raw)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestFiles(client, "special/owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("no speculative pagination parameters", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RawQuery != "" {
				t.Errorf("unexpected query parameters: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("GET-only method", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("empty array response", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		files, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 0 {
			t.Fatalf("len(files) = %d, want 0", len(files))
		}
	})

	t.Run("null response returns empty slice", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `null`)
		})
		files, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
		if files == nil || len(files) != 0 {
			t.Fatalf("files = %#v, want empty slice", files)
		}
	})

	t.Run("patch fallback fields", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"sha":"abc","filename":"old.go","status":"renamed","additions":0,"deletions":0,"too_large":false,"blob_url":"https://b","raw_url":"https://r","patch":{"old_path":"old.go","new_path":"new.go","added_lines":5,"removed_lines":3,"too_large":true}}]`)
		})
		files, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
		f := files[0]
		if f.Patch.OldPath != "old.go" || f.Patch.NewPath != "new.go" || f.Patch.AddedLines != 5 || f.Patch.RemovedLines != 3 || !f.Patch.TooLarge {
			t.Fatalf("patch = %#v", f.Patch)
		}
	})

	t.Run("multiple files preserve server order", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"sha":"a","filename":"second.go","status":"added"},{"sha":"b","filename":"first.go","status":"modified"}]`)
		})
		files, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 2 || files[0].Filename != "second.go" {
			t.Fatalf("files = %#v", files)
		}
	})

	t.Run("204 rejected", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		_, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err == nil {
			t.Fatal("expected error for 204")
		}
	})
}

func TestPullRequestDetailsReactions(t *testing.T) {
	t.Run("exact HTTP 200 success", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"r1","user":{"login":"alice"},"emoji":"+1","emoji_name":"like"}]`)
		})
		reactions, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(reactions) != 1 || reactions[0].ID != "r1" || reactions[0].EmojiName != "like" {
			t.Fatalf("reactions = %#v", reactions)
		}
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"message":"not found"}`)
		})
		_, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "404") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("path escaped identifiers", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			raw := r.URL.RequestURI()
			if !strings.Contains(raw, "%2F") {
				t.Errorf("owner slash not escaped in path: %s", raw)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestReactions(client, "special/owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("paginated query parameters", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("per_page") != "100" {
				t.Errorf("query = %s, want page=1&per_page=100", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("GET-only method", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("empty array response", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		reactions, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(reactions) != 0 {
			t.Fatalf("len(reactions) = %d, want 0", len(reactions))
		}
	})

	t.Run("null response returns empty slice", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `null`)
		})
		reactions, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if reactions == nil || len(reactions) != 0 {
			t.Fatalf("reactions = %#v, want empty slice", reactions)
		}
	})

	t.Run("single reaction with all fields", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"r42","user":{"login":"bob","name":"Bob"},"emoji":"heart","emoji_name":"heart"}]`)
		})
		reactions, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		r := reactions[0]
		if r.ID != "r42" || r.User.Login != "bob" || r.EmojiName != "heart" {
			t.Fatalf("reaction = %#v", r)
		}
	})

	t.Run("multiple reactions preserve server order", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"r1","user":{"login":"a"},"emoji":"+1","emoji_name":"like"},{"id":"r2","user":{"login":"b"},"emoji":"-1","emoji_name":"dislike"}]`)
		})
		reactions, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(reactions) != 2 || reactions[0].ID != "r1" || reactions[1].ID != "r2" {
			t.Fatalf("reactions = %#v", reactions)
		}
	})

	t.Run("204 rejected", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		_, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err == nil {
			t.Fatal("expected error for 204")
		}
	})
}

func TestPullRequestDetailsCommitsInvalidLimitBeforeAuth(t *testing.T) {
	t.Run("zero limit fails without HTTP request", func(t *testing.T) {
		var called bool
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestCommits(client, "owner", "repo", "42", 0)
		if err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Fatalf("error = %v", err)
		}
		if called {
			t.Fatal("HTTP request made for invalid limit")
		}
	})

	t.Run("negative limit fails without HTTP request", func(t *testing.T) {
		var called bool
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestCommits(client, "owner", "repo", "42", -5)
		if err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Fatalf("error = %v", err)
		}
		if called {
			t.Fatal("HTTP request made for invalid limit")
		}
	})
}

func TestPullRequestDetailsEncodedPagination(t *testing.T) {
	t.Run("page and per_page query parameters are positive integers", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			page := r.URL.Query().Get("page")
			perPage := r.URL.Query().Get("per_page")
			if page != "1" {
				t.Errorf("page = %s, want 1", page)
			}
			if perPage != "100" {
				t.Errorf("per_page = %s, want 100", perPage)
			}
			for _, p := range []string{page, perPage} {
				for _, c := range p {
					if c < '0' || c > '9' {
						t.Errorf("non-numeric character in pagination param: %s", p)
						break
					}
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("URL-encoded query parameters", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			rawQuery := r.URL.RawQuery
			if !strings.Contains(rawQuery, "page=1") || !strings.Contains(rawQuery, "per_page=100") {
				t.Errorf("RawQuery = %s", rawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestPullRequestDetailsFilesNoSpeculativeParams(t *testing.T) {
	t.Run("no page parameter", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") != "" {
				t.Error("unexpected page parameter")
			}
			if r.URL.Query().Get("per_page") != "" {
				t.Error("unexpected per_page parameter")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestPullRequestDetailsModifyHistoryNoSpeculativeParams(t *testing.T) {
	t.Run("no page parameter", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") != "" {
				t.Error("unexpected page parameter")
			}
			if r.URL.Query().Get("per_page") != "" {
				t.Error("unexpected per_page parameter")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestPullRequestDetailsOperateLogs(t *testing.T) {
	t.Run("exact HTTP 200 success", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/repos/owner/repo/pulls/42/operate_logs" {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":274531,"action":"add_mr_issue_link","content":"Create mr issue link","merge_request_id":70067,"created_at":"2024-04-23T11:32:08+08:00","updated_at":"2024-04-23T11:32:08+08:00","user":{"login":"alice","name":"Alice"}}]`)
		})
		logs, err := ListPullRequestOperateLogs(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(logs) != 1 || logs[0].ID != 274531 || logs[0].Action != "add_mr_issue_link" || logs[0].User.Login != "alice" {
			t.Fatalf("logs = %#v", logs)
		}
	})

	t.Run("empty array response", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		logs, err := ListPullRequestOperateLogs(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if logs == nil || len(logs) != 0 {
			t.Fatalf("logs = %#v, want empty slice", logs)
		}
	})

	t.Run("pagination stops at limit", func(t *testing.T) {
		requests := 0
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			requests++
			page := r.URL.Query().Get("page")
			perPage := r.URL.Query().Get("per_page")
			if perPage != "100" {
				t.Errorf("per_page = %s, want 100", perPage)
			}
			w.Header().Set("Content-Type", "application/json")
			if page == "1" {
				_, _ = io.WriteString(w, `[{"id":1,"user":{"login":"a"}},{"id":2,"user":{"login":"b"}},{"id":3,"user":{"login":"c"}}]`)
				return
			}
			_, _ = io.WriteString(w, `[]`)
		})
		logs, err := ListPullRequestOperateLogs(client, "owner", "repo", "42", 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(logs) != 2 || logs[0].ID != 1 || logs[1].ID != 2 {
			t.Fatalf("logs = %#v", logs)
		}
		if requests != 1 {
			t.Fatalf("requests = %d, want 1", requests)
		}
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"message":"forbidden"}`)
		})
		_, err := ListPullRequestOperateLogs(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "403") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":`)
		})
		_, err := ListPullRequestOperateLogs(client, "owner", "repo", "42", 30)
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})

	t.Run("path escaped identifiers", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			raw := r.URL.RequestURI()
			if !strings.Contains(raw, "%2F") {
				t.Errorf("owner slash not escaped in path: %s", raw)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestOperateLogs(client, "special/owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestPullRequestDetailsOperateLogsInvalidLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		t.Run(fmt.Sprintf("limit %d fails without HTTP request", limit), func(t *testing.T) {
			var called bool
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `[]`)
			})
			_, err := ListPullRequestOperateLogs(client, "owner", "repo", "42", limit)
			if err == nil || !strings.Contains(err.Error(), "must be positive") {
				t.Fatalf("error = %v", err)
			}
			if called {
				t.Fatal("HTTP request made for invalid limit")
			}
		})
	}
}

func TestPullRequestDetailsModifyHistory(t *testing.T) {
	t.Run("exact HTTP 200 success", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/repos/owner/repo/pulls/42/modify_history" {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"abc","created":true,"created_at":"2024-04-23T11:32:08+08:00","content":"initial","user":{"login":"alice","name":"Alice"},"updated_user":{"login":"alice","name":"Alice"}},{"id":"def","created":false,"deleted":false,"updated_at":"2024-04-24T11:32:08+08:00","content":"edited","user":{"login":"alice"},"updated_user":{"login":"bob"}}]`)
		})
		history, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(history) != 2 || history[0].ID != "abc" || !history[0].Created || history[1].UpdatedUser.Login != "bob" {
			t.Fatalf("history = %#v", history)
		}
	})

	t.Run("empty array response", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		history, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if history == nil || len(history) != 0 {
			t.Fatalf("history = %#v, want empty slice", history)
		}
	})

	t.Run("null response returns empty slice", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `null`)
		})
		history, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if history == nil || len(history) != 0 {
			t.Fatalf("history = %#v, want empty slice", history)
		}
	})

	t.Run("limit truncates locally since endpoint is unpaginated", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"1"},{"id":"2"},{"id":"3"}]`)
		})
		history, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(history) != 2 || history[1].ID != "2" {
			t.Fatalf("history = %#v", history)
		}
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"message":"unauthorized"}`)
		})
		_, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "401") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `not json`)
		})
		_, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 30)
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})
}

func TestPullRequestDetailsModifyHistoryInvalidLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		t.Run(fmt.Sprintf("limit %d fails without HTTP request", limit), func(t *testing.T) {
			var called bool
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `[]`)
			})
			_, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", limit)
			if err == nil || !strings.Contains(err.Error(), "must be positive") {
				t.Fatalf("error = %v", err)
			}
			if called {
				t.Fatal("HTTP request made for invalid limit")
			}
		})
	}
}

func TestPullRequestDetailsMethodEnforcement(t *testing.T) {
	endpoints := []struct {
		name string
		call func(*Client) error
	}{
		{"commits", func(c *Client) error { _, err := ListPullRequestCommits(c, "owner", "repo", "42", 30); return err }},
		{"files", func(c *Client) error { _, err := ListPullRequestFiles(c, "owner", "repo", "42"); return err }},
		{"reactions", func(c *Client) error { _, err := ListPullRequestReactions(c, "owner", "repo", "42", 30); return err }},
	}

	for _, ep := range endpoints {
		t.Run(ep.name+" GET only, no POST/PUT/DELETE", func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("method = %s, want GET", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `[]`)
			})
			if err := ep.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPullRequestDetailsResponseVariants(t *testing.T) {
	t.Run("files empty array", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		files, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
		if files == nil || len(files) != 0 {
			t.Fatalf("files = %#v", files)
		}
	})

	t.Run("reactions single item", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"r1","user":{"login":"a"},"emoji":"+1","emoji_name":"like"}]`)
		})
		reactions, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(reactions) != 1 {
			t.Fatalf("len(reactions) = %d, want 1", len(reactions))
		}
	})

	t.Run("commits missing optional fields", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			raw := `[{"sha":"abc","html_url":"https://example.com","commit":{"message":"msg","author":{"name":"","email":"","date":""}}}]`
			var items []json.RawMessage
			if err := json.Unmarshal([]byte(raw), &items); err != nil {
				t.Fatal(err)
			}
			_, _ = io.WriteString(w, raw)
		})
		commits, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if len(commits) != 1 || commits[0].HTMLURL != "https://example.com" {
			t.Fatalf("commits = %#v", commits)
		}
	})
}

func TestPullRequestDetailsExactRequestCount(t *testing.T) {
	t.Run("commits exact request count", func(t *testing.T) {
		var count int
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			count++
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestCommits(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("requests = %d, want 1", count)
		}
	})

	t.Run("files exact request count", func(t *testing.T) {
		var count int
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			count++
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestFiles(client, "owner", "repo", "42")
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("requests = %d, want 1", count)
		}
	})

	t.Run("reactions exact request count", func(t *testing.T) {
		var count int
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			count++
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		_, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("requests = %d, want 1", count)
		}
	})
}

func TestPullRequestOperateLogsRejectsNullAndInvalidRecords(t *testing.T) {
	t.Run("null record rejected", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[null]`)
		})
		_, err := ListPullRequestOperateLogs(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "id must be positive") {
			t.Fatalf("error = %v, want invalid record error", err)
		}
	})

	t.Run("zero id rejected", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":0,"action":"foo"}]`)
		})
		_, err := ListPullRequestOperateLogs(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "id must be positive") {
			t.Fatalf("error = %v, want invalid record error", err)
		}
	})

	t.Run("bad record beyond limit still errors", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":1,"action":"ok"},null]`)
		})
		_, err := ListPullRequestOperateLogs(client, "owner", "repo", "42", 1)
		if err == nil || !strings.Contains(err.Error(), "id must be positive") {
			t.Fatalf("error = %v, want invalid record error even though bad record is past limit", err)
		}
	})
}

func TestPullRequestModifyHistoryRejectsNullAndInvalidRecords(t *testing.T) {
	t.Run("null record rejected", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[null]`)
		})
		_, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "id must be non-empty") {
			t.Fatalf("error = %v, want invalid record error", err)
		}
	})

	t.Run("empty id rejected", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":""}]`)
		})
		_, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "id must be non-empty") {
			t.Fatalf("error = %v, want invalid record error", err)
		}
	})

	t.Run("whitespace id rejected", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"   "}]`)
		})
		_, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "id must be non-empty") {
			t.Fatalf("error = %v, want invalid record error", err)
		}
	})

	t.Run("bad record beyond limit still errors", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"ok"},{"id":""}]`)
		})
		_, err := ListPullRequestModifyHistory(client, "owner", "repo", "42", 1)
		if err == nil || !strings.Contains(err.Error(), "id must be non-empty") {
			t.Fatalf("error = %v, want invalid record error even though bad record is past limit", err)
		}
	})
}

func TestPullRequestReactionsRejectsNullAndInvalidRecords(t *testing.T) {
	t.Run("null record rejected", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[null]`)
		})
		_, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "id must be non-empty") {
			t.Fatalf("error = %v, want invalid record error", err)
		}
	})

	t.Run("empty id rejected", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":""}]`)
		})
		_, err := ListPullRequestReactions(client, "owner", "repo", "42", 30)
		if err == nil || !strings.Contains(err.Error(), "id must be non-empty") {
			t.Fatalf("error = %v, want invalid record error", err)
		}
	})

	t.Run("bad record beyond limit still errors", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"ok"},null]`)
		})
		_, err := ListPullRequestReactions(client, "owner", "repo", "42", 1)
		if err == nil || !strings.Contains(err.Error(), "id must be non-empty") {
			t.Fatalf("error = %v, want invalid record error even though bad record is past limit", err)
		}
	})
}

func TestPullRequestOperateLogsPagination(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "1" {
			rows := make([]string, 100)
			for i := range rows {
				rows[i] = fmt.Sprintf(`{"id":%d}`, i+1)
			}
			_, _ = io.WriteString(w, "["+strings.Join(rows, ",")+"]")
			return
		}
		_, _ = io.WriteString(w, `[{"id":101}]`)
	})
	logs, err := ListPullRequestOperateLogs(client, "owner", "repo", "42", 101)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 101 || requests != 2 {
		t.Fatalf("len=%d requests=%d, want 101 and 2", len(logs), requests)
	}
	if logs[100].ID != 101 {
		t.Fatalf("last log = %#v", logs[100])
	}
}

func TestPullRequestReactionsPagination(t *testing.T) {
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "1" {
			rows := make([]string, 100)
			for i := range rows {
				rows[i] = fmt.Sprintf(`{"id":"r%d"}`, i+1)
			}
			_, _ = io.WriteString(w, "["+strings.Join(rows, ",")+"]")
			return
		}
		_, _ = io.WriteString(w, `[{"id":"r101"}]`)
	})
	reactions, err := ListPullRequestReactions(client, "owner", "repo", "42", 101)
	if err != nil {
		t.Fatal(err)
	}
	if len(reactions) != 101 || requests != 2 {
		t.Fatalf("len=%d requests=%d, want 101 and 2", len(reactions), requests)
	}
	if reactions[100].ID != "r101" {
		t.Fatalf("last reaction = %#v", reactions[100])
	}
}

func TestPullRequestReactionsRejectsInvalidLaterPage(t *testing.T) {
	calls := 0
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 2 {
			_, _ = io.WriteString(w, `[null]`)
			return
		}
		rows := make([]string, 100)
		for i := range rows {
			rows[i] = fmt.Sprintf(`{"id":"r%d"}`, i)
		}
		_, _ = io.WriteString(w, "["+strings.Join(rows, ",")+"]")
	})
	rows, err := ListPullRequestReactions(client, "a", "b", "7", 101)
	if err == nil || len(rows) != 0 || calls != 2 {
		t.Fatalf("rows=%d calls=%d error=%v", len(rows), calls, err)
	}
}
