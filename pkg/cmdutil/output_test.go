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
