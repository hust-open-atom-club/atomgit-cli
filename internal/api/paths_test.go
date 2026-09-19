package api

import "testing"

func TestRepositoryPathEscapesEachSegmentOnce(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		repo     string
		segments []string
		want     string
	}{
		{
			name:  "repository only",
			owner: "alice",
			repo:  "demo",
			want:  "/repos/alice/demo",
		},
		{
			name:     "label with space",
			owner:    "alice",
			repo:     "demo",
			segments: []string{"labels", "help wanted"},
			want:     "/repos/alice/demo/labels/help%20wanted",
		},
		{
			name:     "slash in owner repo and ref",
			owner:    "org/name",
			repo:     "re po",
			segments: []string{"branches", "feature/a"},
			want:     "/repos/org%2Fname/re%20po/branches/feature%2Fa",
		},
		{
			name:     "already percent-encoded raw value is encoded again",
			owner:    "alice",
			repo:     "demo",
			segments: []string{"branches", "feature%2Fa"},
			want:     "/repos/alice/demo/branches/feature%252Fa",
		},
		{
			name:     "query reserved characters stay in the path segment",
			owner:    "alice",
			repo:     "demo",
			segments: []string{"labels", "a?#b"},
			want:     "/repos/alice/demo/labels/a%3F%23b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RepositoryPath(tt.owner, tt.repo, tt.segments...)
			if got != tt.want {
				t.Fatalf("RepositoryPath() = %q, want %q", got, tt.want)
			}
		})
	}
}
