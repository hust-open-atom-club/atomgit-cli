package repo

import (
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
)

func TestParseRepositoryName(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantOwner string
		wantRepo  string
		wantError bool
		noOwner   bool
	}{
		{name: "current user repository", value: "repo", wantOwner: "current-user", wantRepo: "repo"},
		{name: "explicit user repository", value: "user/repo", wantOwner: "user", wantRepo: "repo"},
		{name: "organization repository", value: "org/project", wantOwner: "org", wantRepo: "project"},
		{name: "trim whitespace", value: " user / repo ", wantOwner: "user", wantRepo: "repo"},
		{name: "missing owner", value: "/repo", wantError: true},
		{name: "missing repository", value: "owner/", wantError: true},
		{name: "empty repository", value: " ", wantError: true},
		{name: "missing default owner", value: "repo", wantError: true, noOwner: true},
		{name: "too many components", value: "owner/group/repo", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultOwner := "current-user"
			if tt.noOwner {
				defaultOwner = ""
			}
			owner, repo, err := parseRepositoryName(tt.value, defaultOwner)
			if tt.wantError {
				if err == nil {
					t.Fatal("parseRepositoryName() expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRepositoryName() unexpected error: %v", err)
			}
			if owner != tt.wantOwner || repo != tt.wantRepo {
				t.Fatalf("parseRepositoryName() = %q/%q, want %q/%q", owner, repo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

func TestCreatedRepositoryURL(t *testing.T) {
	tests := []struct {
		name   string
		result api.Repository
		host   string
		owner  string
		repo   string
		want   string
	}{
		{
			name:   "API response URL",
			result: api.Repository{HTMLURL: "https://atomgit.com/owner/repo"},
			owner:  "owner",
			repo:   "repo",
			want:   "https://atomgit.com/owner/repo",
		},
		{
			name:  "fallback URL",
			host:  "git.example.test",
			owner: "owner",
			repo:  "repo",
			want:  "https://git.example.test/owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createdRepositoryURL(tt.result, tt.host, tt.owner, tt.repo); got != tt.want {
				t.Fatalf("createdRepositoryURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
