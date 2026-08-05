package api

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNumberFormatting(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "string", value: "12", want: "12"},
		{name: "float64", value: float64(12), want: "12"},
		{name: "int", value: 12, want: "12"},
		{name: "int64", value: int64(12), want: "12"},
		{name: "other", value: true, want: "true"},
		{name: "nil", value: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (&PullRequest{Number: tt.value}).GetNumber(); got != tt.want {
				t.Fatalf("PullRequest.GetNumber() = %q, want %q", got, tt.want)
			}
			if got := (&Issue{Number: tt.value}).GetNumber(); got != tt.want {
				t.Fatalf("Issue.GetNumber() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteResponseFixtures(t *testing.T) {
	t.Run("pull request create", func(t *testing.T) {
		data, err := os.ReadFile("testdata/pull_request_create_response.json")
		if err != nil {
			t.Fatal(err)
		}
		var result PullRequestWriteResponse
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		if got := result.GetNumber(); got != "7" {
			t.Fatalf("number = %q, want 7", got)
		}
		if got := result.GetURL(); got != "https://atomgit.com/alice/demo/merge_requests/7" {
			t.Fatalf("URL = %q", got)
		}
	})

	for _, tt := range []struct {
		name string
		file string
		want string
	}{
		{name: "issue comment create", file: "testdata/issue_comment_create_response.json", want: "180041703"},
		{name: "PR comment create", file: "testdata/pull_request_comment_create_response.json", want: "180041704"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatal(err)
			}
			var result CreateCommentResponse
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}
			if got := result.GetID(); got != tt.want {
				t.Fatalf("ID = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPullRequestReviewRequestJSON(t *testing.T) {
	tests := []struct {
		name  string
		force bool
		want  string
	}{
		{name: "normal approval", want: `{"force":false}`},
		{name: "forced approval", force: true, want: `{"force":true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(PullRequestReviewRequest{Force: tt.force})
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("JSON = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestRepositoryContentDecode(t *testing.T) {
	t.Run("file with base64 content", func(t *testing.T) {
		raw := `{"name":"README.md","path":"README.md","sha":"abc123","size":42,"type":"file","encoding":"base64","content":"IyByZWFkbWU=","url":"https://example.test/api/v5/repos/alice/demo/contents/README.md","html_url":"https://example.test/alice/demo/blob/main/README.md","download_url":"https://example.test/alice/demo/raw/main/README.md"}`
		var content RepositoryContent
		if err := json.Unmarshal([]byte(raw), &content); err != nil {
			t.Fatal(err)
		}
		if content.Name != "README.md" || content.Path != "README.md" || content.SHA != "abc123" {
			t.Fatalf("content = %#v", content)
		}
		if content.Size != 42 || content.Type != "file" || content.Encoding != "base64" {
			t.Fatalf("content = %#v", content)
		}
		if content.Content != "IyByZWFkbWU=" || content.HTMLURL == "" || content.DownloadURL == "" {
			t.Fatalf("content = %#v", content)
		}
	})

	t.Run("directory entry omits encoding and content", func(t *testing.T) {
		raw := `{"name":"cmd","path":"cmd","sha":"def456","size":0,"type":"dir"}`
		var content RepositoryContent
		if err := json.Unmarshal([]byte(raw), &content); err != nil {
			t.Fatal(err)
		}
		if content.Type != "dir" || content.Encoding != "" || content.Content != "" || content.ContentPresent {
			t.Fatalf("directory entry should omit encoding/content: %#v", content)
		}
	})

	t.Run("empty file content remains present", func(t *testing.T) {
		raw := `{"name":"empty.txt","path":"empty.txt","sha":"empty","size":0,"type":"file","encoding":"base64","content":""}`
		var content RepositoryContent
		if err := json.Unmarshal([]byte(raw), &content); err != nil {
			t.Fatal(err)
		}
		if !content.ContentPresent || content.Content != "" {
			t.Fatalf("empty content presence was lost: %#v", content)
		}
	})

	t.Run("zero value has empty fields", func(t *testing.T) {
		var content RepositoryContent
		if content.Name != "" || content.Size != 0 || content.Type != "" || content.ContentPresent {
			t.Fatalf("zero value = %#v", content)
		}
	})
}

func TestIssueLinkedPullRequestDecode(t *testing.T) {
	t.Run("full linked PR", func(t *testing.T) {
		raw := `{"id":101,"number":"7","title":"Fix bug","body":"desc","state":"open","html_url":"https://example.test/alice/demo/pulls/7","head":{"ref":"feature"},"base":{"ref":"main"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}`
		var pr IssueLinkedPullRequest
		if err := json.Unmarshal([]byte(raw), &pr); err != nil {
			t.Fatal(err)
		}
		if pr.ID != 101 || pr.GetNumber() != "7" || pr.Title != "Fix bug" {
			t.Fatalf("pr = %#v", pr)
		}
		if pr.State != "open" || pr.HTMLURL == "" {
			t.Fatalf("pr = %#v", pr)
		}
		if pr.Head == nil || pr.Head.Ref != "feature" {
			t.Fatalf("head = %#v", pr.Head)
		}
		if pr.Base == nil || pr.Base.Ref != "main" {
			t.Fatalf("base = %#v", pr.Base)
		}
	})

	t.Run("null head and base", func(t *testing.T) {
		raw := `{"id":102,"number":"8","title":"PR","state":"closed","head":null,"base":null}`
		var pr IssueLinkedPullRequest
		if err := json.Unmarshal([]byte(raw), &pr); err != nil {
			t.Fatal(err)
		}
		if pr.Head != nil || pr.Base != nil {
			t.Fatalf("head/base should be nil: %#v", pr)
		}
	})
}

func TestRelatedBranchesRequestJSON(t *testing.T) {
	req := RelatedBranchesRequest{BranchNames: []string{"main", "feature/x"}}
	got, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"branch_names":["main","feature/x"]}`
	if string(got) != want {
		t.Fatalf("JSON = %s, want %s", got, want)
	}
}

func TestPullRequestCommitDecode(t *testing.T) {
	raw := `{"sha":"abc123","html_url":"https://example.test/alice/demo/pulls/7/commits/abc123","commit":{"message":"Fix bug","author":{"name":"Alice","email":"alice@example.test","date":"2026-01-01T00:00:00Z","login":"alice"}}}`
	var commit PullRequestCommit
	if err := json.Unmarshal([]byte(raw), &commit); err != nil {
		t.Fatal(err)
	}
	if commit.SHA != "abc123" || commit.HTMLURL == "" {
		t.Fatalf("commit = %#v", commit)
	}
	if commit.Commit.Message != "Fix bug" {
		t.Fatalf("message = %q", commit.Commit.Message)
	}
	if commit.Commit.Author.Name != "Alice" || commit.Commit.Author.Email != "alice@example.test" {
		t.Fatalf("author = %#v", commit.Commit.Author)
	}
	if commit.Commit.Author.Date != "2026-01-01T00:00:00Z" || commit.Commit.Author.Login != "alice" {
		t.Fatalf("author = %#v", commit.Commit.Author)
	}
}

func TestPullRequestFileDecode(t *testing.T) {
	t.Run("top-level fields", func(t *testing.T) {
		raw := `{"sha":"abc123","filename":"main.go","status":"modified","additions":10,"deletions":5,"too_large":false,"blob_url":"https://example.test/blob","raw_url":"https://example.test/raw"}`
		var file PullRequestFile
		if err := json.Unmarshal([]byte(raw), &file); err != nil {
			t.Fatal(err)
		}
		if file.Filename != "main.go" || file.Status != "modified" {
			t.Fatalf("file = %#v", file)
		}
		if file.Additions != 10 || file.Deletions != 5 || file.TooLarge {
			t.Fatalf("file = %#v", file)
		}
	})

	t.Run("nested patch fields", func(t *testing.T) {
		raw := `{"sha":"def456","filename":"other.go","status":"added","patch":{"old_path":"","new_path":"other.go","added_lines":20,"removed_lines":0,"too_large":true}}`
		var file PullRequestFile
		if err := json.Unmarshal([]byte(raw), &file); err != nil {
			t.Fatal(err)
		}
		if file.Patch.NewPath != "other.go" || file.Patch.AddedLines != 20 {
			t.Fatalf("patch = %#v", file.Patch)
		}
		if !file.Patch.TooLarge {
			t.Fatal("patch.TooLarge should be true")
		}
	})

	t.Run("zero value defaults", func(t *testing.T) {
		var file PullRequestFile
		if file.Additions != 0 || file.Deletions != 0 || file.TooLarge {
			t.Fatalf("zero value = %#v", file)
		}
	})
}

func TestPullRequestReactionDecode(t *testing.T) {
	raw := `{"id":12345,"user":{"login":"alice","name":"Alice"},"content":"+1","created_at":"2026-01-01T00:00:00Z"}`
	var reaction PullRequestReaction
	if err := json.Unmarshal([]byte(raw), &reaction); err != nil {
		t.Fatal(err)
	}
	if reaction.ID != 12345 || reaction.User.Login != "alice" {
		t.Fatalf("reaction = %#v", reaction)
	}
	if reaction.Content != "+1" || reaction.CreatedAt == "" {
		t.Fatalf("reaction = %#v", reaction)
	}
}

func TestPullRequestCollaborationFields(t *testing.T) {
	t.Run("nullable milestone and empty arrays", func(t *testing.T) {
		raw := `{"id":1,"number":"7","title":"PR","state":"open","assignees":[],"approval_reviewers":[],"testers":[],"milestone":null}`
		var pr PullRequest
		if err := json.Unmarshal([]byte(raw), &pr); err != nil {
			t.Fatal(err)
		}
		if pr.Milestone != nil {
			t.Fatalf("milestone should be nil, got %#v", pr.Milestone)
		}
		if pr.Assignees == nil {
			t.Fatal("assignees should be non-nil empty slice")
		}
		if len(pr.Assignees) != 0 {
			t.Fatalf("assignees = %#v", pr.Assignees)
		}
		if pr.ApprovalReviewers == nil {
			t.Fatal("approval_reviewers should be non-nil empty slice")
		}
		if pr.Testers == nil {
			t.Fatal("testers should be non-nil empty slice")
		}
	})

	t.Run("populated collaboration fields", func(t *testing.T) {
		raw := `{"id":2,"number":"8","title":"PR","state":"open","assignees":[{"login":"alice"}],"approval_reviewers":[{"login":"bob"}],"testers":[{"login":"carol"}],"milestone":{"number":4,"title":"v1.0","state":"open","url":"https://example.test/milestone/4"}}`
		var pr PullRequest
		if err := json.Unmarshal([]byte(raw), &pr); err != nil {
			t.Fatal(err)
		}
		if len(pr.Assignees) != 1 || pr.Assignees[0].Login != "alice" {
			t.Fatalf("assignees = %#v", pr.Assignees)
		}
		if len(pr.ApprovalReviewers) != 1 || pr.ApprovalReviewers[0].Login != "bob" {
			t.Fatalf("approval_reviewers = %#v", pr.ApprovalReviewers)
		}
		if len(pr.Testers) != 1 || pr.Testers[0].Login != "carol" {
			t.Fatalf("testers = %#v", pr.Testers)
		}
		if pr.Milestone == nil || pr.Milestone.Number != 4 || pr.Milestone.Title != "v1.0" {
			t.Fatalf("milestone = %#v", pr.Milestone)
		}
	})
}
