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
