package cmdutil

import (
	"bytes"
	"testing"
)

func TestPrintResultWithOptionalURL(t *testing.T) {
	tests := []struct {
		name    string
		summary string
		url     string
		want    string
	}{
		{
			name:    "with URL",
			summary: "Updated comment #42",
			url:     "https://atomgit.com/owner/repo/issues/1#comment-42",
			want:    "Updated comment #42: https://atomgit.com/owner/repo/issues/1#comment-42\n",
		},
		{
			name:    "without URL",
			summary: "Updated comment #42",
			want:    "Updated comment #42\n",
		},
		{
			name:    "whitespace-only URL",
			summary: "Created comment #42",
			url:     " \t\n",
			want:    "Created comment #42\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			PrintResultWithOptionalURL(&out, tt.summary, tt.url)
			if got := out.String(); got != tt.want {
				t.Fatalf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOwnerRepoFromWebURL(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		owner string
		repo  string
	}{
		{name: "pull URL", raw: "https://atomgit.com/owner/repo/pull/7", owner: "owner", repo: "repo"},
		{name: "issue URL with fragment", raw: "https://atomgit.com/owner/repo/issues/1#comment-42", owner: "owner", repo: "repo"},
		{name: "whitespace trimmed", raw: "  https://git.example.test/alice/demo  ", owner: "alice", repo: "demo"},
		{name: "escaped segment decodes", raw: "https://atomgit.com/owner%20name/repo/pull/7", owner: "owner name", repo: "repo"},
		{name: "repo root only", raw: "https://atomgit.com/owner/repo", owner: "owner", repo: "repo"},
		{name: "too few segments", raw: "https://atomgit.com/owner", owner: "", repo: ""},
		{name: "empty input", raw: "  ", owner: "", repo: ""},
		{name: "not a URL", raw: "://bad", owner: "", repo: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := OwnerRepoFromWebURL(tt.raw)
			if owner != tt.owner || repo != tt.repo {
				t.Fatalf("OwnerRepoFromWebURL(%q) = (%q, %q), want (%q, %q)", tt.raw, owner, repo, tt.owner, tt.repo)
			}
		})
	}
}

func TestResolveWebURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		host string
		path []string
		want string
	}{
		{name: "API URL", raw: " https://atomgit.com/owner/repo/pull/7 ", want: "https://atomgit.com/owner/repo/pull/7"},
		{name: "default host", path: []string{"owner", "repo", "issues", "7"}, want: "https://atomgit.com/owner/repo/issues/7"},
		{name: "configured host", host: "https://git.example.test/", path: []string{"owner", "repo"}, want: "https://git.example.test/owner/repo"},
		{name: "escape segment", path: []string{"owner name", "repo"}, want: "https://atomgit.com/owner%20name/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveWebURL(tt.raw, tt.host, tt.path...); got != tt.want {
				t.Fatalf("ResolveWebURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
