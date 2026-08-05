package api

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetRepositoryContentPathConstruction(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		repo     string
		path     string
		ref      string
		wantPath string
		wantRef  string
	}{
		{
			name:     "simple file path",
			owner:    "alice",
			repo:     "demo",
			path:     "README.md",
			wantPath: "/repos/alice/demo/contents/README.md",
		},
		{
			name:     "nested file path with escaped segments",
			owner:    "alice",
			repo:     "demo",
			path:     "src/main.go",
			wantPath: "/repos/alice/demo/contents/src/main.go",
		},
		{
			name:     "spaces in path segments are escaped",
			owner:    "alice",
			repo:     "demo",
			path:     "my docs/notes.txt",
			wantPath: "/repos/alice/demo/contents/my%20docs/notes.txt",
		},
		{
			name:     "special characters in path",
			owner:    "alice",
			repo:     "demo",
			path:     "file#1.go",
			wantPath: "/repos/alice/demo/contents/file%231.go",
		},
		{
			name:     "owner with special characters",
			owner:    "my-org",
			repo:     "my-repo",
			path:     "file.txt",
			wantPath: "/repos/my-org/my-repo/contents/file.txt",
		},
		{
			name:     "ref query parameter",
			owner:    "alice",
			repo:     "demo",
			path:     "README.md",
			ref:      "main",
			wantPath: "/repos/alice/demo/contents/README.md",
			wantRef:  "main",
		},
		{
			name:     "ref with special characters is encoded",
			owner:    "alice",
			repo:     "demo",
			path:     "file.txt",
			ref:      "feature/branch",
			wantPath: "/repos/alice/demo/contents/file.txt",
			wantRef:  "feature%2Fbranch",
		},
		{
			name:     "empty ref is omitted",
			owner:    "alice",
			repo:     "demo",
			path:     "file.txt",
			ref:      "",
			wantPath: "/repos/alice/demo/contents/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath, capturedQuery string
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.EscapedPath()
				capturedQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"name":"file.txt","path":"file.txt","sha":"abc","size":10,"type":"file","encoding":"base64","content":"SGVsbG8="}`)
			})

			_, err := GetRepositoryContent(client, tt.owner, tt.repo, tt.path, tt.ref)
			if err != nil {
				t.Fatal(err)
			}

			if capturedPath != tt.wantPath {
				t.Errorf("path = %q, want %q", capturedPath, tt.wantPath)
			}
			if tt.wantRef != "" && capturedQuery != "ref="+tt.wantRef {
				t.Errorf("query = %q, want ref=%s", capturedQuery, tt.wantRef)
			}
			if tt.wantRef == "" && capturedQuery != "" {
				t.Errorf("query = %q, want empty", capturedQuery)
			}
		})
	}
}

func TestListRepositoryContentRootDirectory(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/alice/demo/contents" {
			t.Fatalf("path = %q, want /repos/alice/demo/contents", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"name":"README.md","path":"README.md","sha":"abc","size":42,"type":"file"}]`)
	})

	entries, err := ListRepositoryContent(client, "alice", "demo", ".", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "README.md" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestListRepositoryContentSubdirectory(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/alice/demo/contents/src" {
			t.Fatalf("path = %q, want /repos/alice/demo/contents/src", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"name":"main.go","path":"src/main.go","sha":"def","size":100,"type":"file"}]`)
	})

	entries, err := ListRepositoryContent(client, "alice", "demo", "src", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "main.go" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestListRepositoryContentEmptyDirectory(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[]`)
	})

	entries, err := ListRepositoryContent(client, "alice", "demo", "src", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty slice, got %d entries", len(entries))
	}
}

func TestGetRepositoryContentDecodesFile(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"README.md","path":"README.md","sha":"abc123","size":42,"type":"file","encoding":"base64","content":"SGVsbG8gV29ybGQ="}`)
	})

	content, err := GetRepositoryContent(client, "alice", "demo", "README.md", "")
	if err != nil {
		t.Fatal(err)
	}
	if content.Name != "README.md" {
		t.Errorf("Name = %q", content.Name)
	}
	if content.Path != "README.md" {
		t.Errorf("Path = %q", content.Path)
	}
	if content.SHA != "abc123" {
		t.Errorf("SHA = %q", content.SHA)
	}
	if content.Size != 42 {
		t.Errorf("Size = %d", content.Size)
	}
	if content.Type != "file" {
		t.Errorf("Type = %q", content.Type)
	}
	if content.Encoding != "base64" {
		t.Errorf("Encoding = %q", content.Encoding)
	}
	if content.Content != "SGVsbG8gV29ybGQ=" {
		t.Errorf("Content = %q", content.Content)
	}
}

func TestListRepositoryContentDecodesDirectory(t *testing.T) {
	body := `[
		{"name":"cmd","path":"cmd","sha":"aaa","size":0,"type":"dir"},
		{"name":"README.md","path":"README.md","sha":"bbb","size":42,"type":"file"}
	]`
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	})

	entries, err := ListRepositoryContent(client, "alice", "demo", ".", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "cmd" || entries[0].Type != "dir" {
		t.Fatalf("entry[0] = %#v", entries[0])
	}
	if entries[1].Name != "README.md" || entries[1].Type != "file" {
		t.Fatalf("entry[1] = %#v", entries[1])
	}
}

func TestGetRepositoryContentExactStatus200(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantError  bool
	}{
		{name: "200 accepted", statusCode: http.StatusOK, wantError: false},
		{name: "201 rejected", statusCode: http.StatusCreated, wantError: true},
		{name: "204 rejected", statusCode: http.StatusNoContent, wantError: true},
		{name: "404 rejected", statusCode: http.StatusNotFound, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"name":"f.txt","path":"f.txt","sha":"abc","size":1,"type":"file","encoding":"base64","content":"eA=="}`)
				}
			})

			_, err := GetRepositoryContent(client, "alice", "demo", "f.txt", "")
			if tt.wantError && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestListRepositoryContentExactStatus200(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantError  bool
	}{
		{name: "200 accepted", statusCode: http.StatusOK, wantError: false},
		{name: "404 rejected", statusCode: http.StatusNotFound, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `[]`)
				}
			})

			_, err := ListRepositoryContent(client, "alice", "demo", ".", "")
			if tt.wantError && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetRepositoryContentGetOnlyMethod(t *testing.T) {
	var capturedMethod string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"f.txt","path":"f.txt","sha":"abc","size":1,"type":"file","encoding":"base64","content":"eA=="}`)
	})

	_, err := GetRepositoryContent(client, "alice", "demo", "f.txt", "")
	if err != nil {
		t.Fatal(err)
	}
	if capturedMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET", capturedMethod)
	}
}

func TestListRepositoryContentGetOnlyMethod(t *testing.T) {
	var capturedMethod string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[]`)
	})

	_, err := ListRepositoryContent(client, "alice", "demo", ".", "")
	if err != nil {
		t.Fatal(err)
	}
	if capturedMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET", capturedMethod)
	}
}

func TestContentFunctionsNoRequestBody(t *testing.T) {
	t.Run("GetRepositoryContent sends no body", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				if len(body) != 0 {
					t.Fatalf("unexpected request body: %q", body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"name":"f.txt","path":"f.txt","sha":"abc","size":1,"type":"file","encoding":"base64","content":"eA=="}`)
		})

		_, err := GetRepositoryContent(client, "alice", "demo", "f.txt", "")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("ListRepositoryContent sends no body", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				if len(body) != 0 {
					t.Fatalf("unexpected request body: %q", body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})

		_, err := ListRepositoryContent(client, "alice", "demo", ".", "")
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestListRepositoryContentPreservesServerOrder(t *testing.T) {
	body := `[{"name":"z","path":"z","sha":"1","size":0,"type":"file"},{"name":"a","path":"a","sha":"2","size":0,"type":"file"},{"name":"m","path":"m","sha":"3","size":0,"type":"file"}]`
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	})

	entries, err := ListRepositoryContent(client, "alice", "demo", ".", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Name != "z" || entries[1].Name != "a" || entries[2].Name != "m" {
		t.Fatalf("order changed: %v", entries)
	}
}

func TestListRepositoryContentEmptyArray(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[]`)
	})

	entries, err := ListRepositoryContent(client, "alice", "demo", ".", "")
	if err != nil {
		t.Fatal(err)
	}
	if entries == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestContentFunctionsAPIError(t *testing.T) {
	t.Run("GetRepositoryContent reports API error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})

		_, err := GetRepositoryContent(client, "alice", "demo", "missing.txt", "")
		if err == nil || !strings.Contains(err.Error(), "API error") {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(err.Error(), "404") {
			t.Fatalf("error missing status: %v", err)
		}
	})

	t.Run("ListRepositoryContent reports API error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "forbidden", http.StatusForbidden)
		})

		_, err := ListRepositoryContent(client, "alice", "demo", "secret", "")
		if err == nil || !strings.Contains(err.Error(), "API error") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestGetRepositoryContentInvalidJSON(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `not json`)
	})

	_, err := GetRepositoryContent(client, "alice", "demo", "f.txt", "")
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
}

func TestListRepositoryContentInvalidJSON(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `not json`)
	})

	_, err := ListRepositoryContent(client, "alice", "demo", ".", "")
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
}

func TestContentJSONFieldMapping(t *testing.T) {
	t.Run("file object", func(t *testing.T) {
		body := `{
			"name":"f.txt",
			"path":"sub/f.txt",
			"sha":"abc123",
			"size":256,
			"type":"file",
			"encoding":"base64",
			"content":"SGVsbG8=",
			"url":"https://api.atomgit.com/api/v5/repos/alice/demo/contents/sub/f.txt",
			"html_url":"https://atomgit.com/alice/demo/blob/main/sub/f.txt",
			"download_url":"https://atomgit.com/alice/demo/raw/main/sub/f.txt"
		}`
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, body)
		})

		c, err := GetRepositoryContent(client, "alice", "demo", "sub/f.txt", "")
		if err != nil {
			t.Fatal(err)
		}
		if c.Name != "f.txt" {
			t.Errorf("Name = %q", c.Name)
		}
		if c.Path != "sub/f.txt" {
			t.Errorf("Path = %q", c.Path)
		}
		if c.SHA != "abc123" {
			t.Errorf("SHA = %q", c.SHA)
		}
		if c.Size != 256 {
			t.Errorf("Size = %d", c.Size)
		}
		if c.Type != "file" {
			t.Errorf("Type = %q", c.Type)
		}
		if c.Encoding != "base64" {
			t.Errorf("Encoding = %q", c.Encoding)
		}
		if c.Content != "SGVsbG8=" {
			t.Errorf("Content = %q", c.Content)
		}
		if c.URL != "https://api.atomgit.com/api/v5/repos/alice/demo/contents/sub/f.txt" {
			t.Errorf("URL = %q", c.URL)
		}
		if c.HTMLURL != "https://atomgit.com/alice/demo/blob/main/sub/f.txt" {
			t.Errorf("HTMLURL = %q", c.HTMLURL)
		}
		if c.DownloadURL != "https://atomgit.com/alice/demo/raw/main/sub/f.txt" {
			t.Errorf("DownloadURL = %q", c.DownloadURL)
		}
	})

	t.Run("directory entry", func(t *testing.T) {
		body := `[{"name":"src","path":"src","sha":"def456","size":0,"type":"dir"}]`
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, body)
		})

		entries, err := ListRepositoryContent(client, "alice", "demo", ".", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Name != "src" {
			t.Errorf("Name = %q", entries[0].Name)
		}
		if entries[0].Path != "src" {
			t.Errorf("Path = %q", entries[0].Path)
		}
		if entries[0].Type != "dir" {
			t.Errorf("Type = %q", entries[0].Type)
		}
		if entries[0].SHA != "def456" {
			t.Errorf("SHA = %q", entries[0].SHA)
		}
		if entries[0].Size != 0 {
			t.Errorf("Size = %d", entries[0].Size)
		}
	})
}

func TestGetRepositoryContentDirJSONDecoding(t *testing.T) {
	body := `[{"name":"dir","path":"dir","sha":"abc","size":0,"type":"dir"}]`
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	})

	_, err := GetRepositoryContent(client, "alice", "demo", "dir", "")
	if err == nil {
		t.Fatal("expected JSON decode error when directory array returned for single-get")
	}
	if !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Fatalf("expected unmarshal error, got: %v", err)
	}
}

func TestContentRequestCount(t *testing.T) {
	t.Run("GetRepositoryContent makes exactly 1 request", func(t *testing.T) {
		var count int
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			count++
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"name":"f.txt","path":"f.txt","sha":"abc","size":1,"type":"file","encoding":"base64","content":"eA=="}`)
		})

		_, err := GetRepositoryContent(client, "alice", "demo", "f.txt", "")
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("request count = %d, want 1", count)
		}
	})

	t.Run("ListRepositoryContent makes exactly 1 request", func(t *testing.T) {
		var count int
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			count++
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})

		_, err := ListRepositoryContent(client, "alice", "demo", ".", "")
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("request count = %d, want 1", count)
		}
	})
}
