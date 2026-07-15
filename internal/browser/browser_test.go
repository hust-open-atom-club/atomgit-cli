package browser

import (
	"errors"
	"fmt"
	"testing"
)

const (
	owner = "hust-open-atom-club"
	repo  = "atomgit-cli"
)

func TestBuildRepoURL(t *testing.T) {
	want := "https://atomgit.com/" + owner + "/" + repo
	if got := BuildRepoURL(owner, repo); got != want {
		t.Errorf("BuildRepoURL() = %q, want %q", got, want)
	}
}

func TestBuildFileURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		branch, path       string
		lineStart, lineEnd int
		want               string
	}{
		{
			name:   "basic",
			branch: "main",
			path:   "README.md",
			want:   fmt.Sprintf("https://atomgit.com/%s/%s/blob/main/README.md", owner, repo),
		},
		{
			name:      "single line",
			branch:    "main",
			path:      "README.md",
			lineStart: 3,
			want:      fmt.Sprintf("https://atomgit.com/%s/%s/blob/main/README.md#L3", owner, repo),
		},
		{
			name:      "line range",
			branch:    "main",
			path:      "README.md",
			lineStart: 5,
			lineEnd:   31,
			want:      fmt.Sprintf("https://atomgit.com/%s/%s/blob/main/README.md#L5-L31", owner, repo),
		},
		{
			name:   "nested path",
			branch: "main",
			path:   "docs/installation.md",
			want:   fmt.Sprintf("https://atomgit.com/%s/%s/blob/main/docs/installation.md", owner, repo),
		},
		{
			name:   "other branch",
			branch: "dev",
			path:   "test.txt",
			want:   fmt.Sprintf("https://atomgit.com/%s/%s/blob/dev/test.txt", owner, repo),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := BuildFileURL(owner, repo, tt.branch, tt.path, tt.lineStart, tt.lineEnd); got != tt.want {
				t.Errorf("BuildFileURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildIssueURL(t *testing.T) {
	want := fmt.Sprintf("https://atomgit.com/%s/%s/issues/29", owner, repo)
	if got := BuildIssueURL(owner, repo, 29); got != want {
		t.Errorf("BuildIssueURL() = %q, want %q", got, want)
	}
}

func TestBuildPRURL(t *testing.T) {
	want := fmt.Sprintf("https://atomgit.com/%s/%s/pull/42", owner, repo)
	if got := BuildPRURL(owner, repo, 42); got != want {
		t.Errorf("BuildPRURL() = %q, want %q", got, want)
	}
}

func TestBuildCommitURL(t *testing.T) {
	want := fmt.Sprintf("https://atomgit.com/%s/%s/commit/46c9d2ed", owner, repo)
	if got := BuildCommitURL(owner, repo, "46c9d2ed"); got != want {
		t.Errorf("BuildCommitURL() = %q, want %q", got, want)
	}
}

func TestBuildReleasesURL(t *testing.T) {
	want := fmt.Sprintf("https://atomgit.com/%s/%s/releases", owner, repo)
	if got := BuildReleasesURL(owner, repo); got != want {
		t.Errorf("BuildReleasesURL() = %q, want %q", got, want)
	}
}

func TestBuildFileURLLineRangeReversed(t *testing.T) {
	want := fmt.Sprintf("https://atomgit.com/%s/%s/blob/main/main.go#L320", owner, repo)
	if got := BuildFileURL(owner, repo, "main", "main.go", 320, 312); got != want {
		t.Errorf("BuildFileURL() = %q, want %q", got, want)
	}
}

func TestParseRemoteURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		raw       string
		wantOwner string
		wantRepo  string
		wantErr   error
	}{
		{
			name:      "https atomgit",
			raw:       fmt.Sprintf("https://atomgit.com/%s/%s.git", owner, repo),
			wantOwner: owner,
			wantRepo:  repo,
		},
		{
			name:      "https gitcode",
			raw:       fmt.Sprintf("https://gitcode.com/%s/%s.git", owner, repo),
			wantOwner: owner,
			wantRepo:  repo,
		},
		{
			name:      "ssh atomgit",
			raw:       fmt.Sprintf("git@atomgit.com/%s/%s.git", owner, repo),
			wantOwner: owner,
			wantRepo:  repo,
		},
		{
			name:      "ssh gitcode",
			raw:       fmt.Sprintf("git@gitcode.com/%s/%s.git", owner, repo),
			wantOwner: owner,
			wantRepo:  repo,
		},
		{
			name:      "ssh protocol",
			raw:       fmt.Sprintf("ssh://git@atomgit.com/%s/%s.git", owner, repo),
			wantOwner: owner,
			wantRepo:  repo,
		},
		{
			name:      "without .git",
			raw:       fmt.Sprintf("https://atomgit.com/%s/%s", owner, repo),
			wantOwner: owner,
			wantRepo:  repo,
		},
		{
			name:    "unknown host",
			raw:     fmt.Sprintf("https://github.com/%s/%s.git", owner, repo),
			wantErr: ErrUnknownHost,
		},
		{
			name:    "bare host",
			raw:     fmt.Sprintf("git@atomgit.com/%s.git", repo),
			wantErr: ErrParseRemoteURL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotOwner, gotRepo, gotErr := ParseRemoteURL(tt.raw)
			if gotErr != nil {
				if !errors.Is(gotErr, tt.wantErr) {
					t.Errorf("ParseRemoteURL() = %v, want %v", gotErr, tt.wantErr)
				}

				return
			}
			if tt.wantErr != nil {
				t.Fatal("ParseRemoteURL() succeeded unexpectedly")
			}
			if gotOwner != tt.wantOwner || gotRepo != tt.wantRepo {
				t.Errorf("ParseRemoteURL() = (%q, %q), want (%q, %q)", gotOwner, gotRepo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}
