package repo

import (
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
)

func TestRepositoryVisibility(t *testing.T) {
	tests := []struct {
		name string
		repo api.Repository
		want string
	}{
		{name: "public", want: "public"},
		{name: "internal", repo: api.Repository{Internal: true}, want: "internal"},
		{name: "private", repo: api.Repository{Private: true}, want: "private"},
		{
			name: "private takes precedence over internal",
			repo: api.Repository{Private: true, Internal: true},
			want: "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repositoryVisibility(tt.repo); got != tt.want {
				t.Fatalf("repositoryVisibility() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRepositoryParentName(t *testing.T) {
	tests := []struct {
		name string
		repo api.Repository
		want string
	}{
		{name: "not a fork", repo: api.Repository{ParentFullName: "owner/repo"}, want: ""},
		{name: "fork without parent", repo: api.Repository{Fork: true}, want: ""},
		{
			name: "fork with parent",
			repo: api.Repository{Fork: true, ParentFullName: " owner/repo "},
			want: "owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repositoryParentName(tt.repo); got != tt.want {
				t.Fatalf("repositoryParentName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatRepositoryTime(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "  ", want: ""},
		{
			name:  "timestamp with fractional seconds",
			value: "2026-07-10T18:15:19.088+08:00",
			want:  "2026-07-10 18:15:19 +08:00",
		},
		{
			name:  "UTC timestamp",
			value: "2026-07-10T10:15:19Z",
			want:  "2026-07-10 10:15:19 +00:00",
		},
		{name: "unknown format", value: "yesterday", want: "yesterday"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatRepositoryTime(tt.value); got != tt.want {
				t.Fatalf("formatRepositoryTime() = %q, want %q", got, tt.want)
			}
		})
	}
}
