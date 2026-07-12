package comment

import (
	"reflect"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
)

func TestConvertHTMLToMarkdown(t *testing.T) {
	plain := "plain text"
	if got := convertHTMLToMarkdown(plain); got != plain {
		t.Fatalf("plain text = %q", got)
	}

	html := `<p>Before</p><table><tr><th>Name</th><th>Link</th></tr><tr><td>Alice</td><td><a href="https://example.com">profile</a></td></tr></table><p>After</p>`
	want := "BeforeName | Link\n--- | ---\nAlice | [profile](https://example.com)\nAfter"
	if got := convertHTMLToMarkdown(html); got != want {
		t.Fatalf("converted table:\n%q\nwant:\n%q", got, want)
	}
}

func TestParseTable(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{name: "empty", html: "<table></table>", want: ""},
		{
			name: "rows",
			html: "<table><tr><th>A</th><th>B</th></tr><tr><td>1</td><td>2</td></tr></table>",
			want: "A | B\n--- | ---\n1 | 2\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTable(tt.html); got != tt.want {
				t.Fatalf("parseTable() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanHTMLTags(t *testing.T) {
	input := `<p>&lt;ok&gt;&nbsp;&amp;&nbsp;&quot;yes&quot; &#9989; &#10060; <a href="https://example.com">link</a></p>`
	want := `<ok> & "yes" ✅ ❌ [link](https://example.com)`
	if got := cleanHTMLTags(input); got != want {
		t.Fatalf("cleanHTMLTags() = %q, want %q", got, want)
	}
}

func TestSortCommentsByTime(t *testing.T) {
	comments := []api.Comment{
		{ID: 2, CreatedAt: "2026-01-02T00:00:00Z"},
		{ID: 1, CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: 3, CreatedAt: "2026-01-03T00:00:00Z"},
	}
	sortCommentsByTime(comments)
	got := []int64{comments[0].ID, comments[1].ID, comments[2].ID}
	if want := []int64{1, 2, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %#v, want %#v", got, want)
	}
}

func TestFormatTime(t *testing.T) {
	if got := formatTime("2026-01-02T03:04:05Z", "2006-01-02"); got != "2026-01-02" {
		t.Fatalf("formatTime() = %q", got)
	}
	if got := formatTime("invalid", "2006"); got != "invalid" {
		t.Fatalf("invalid formatTime() = %q", got)
	}
}

func TestYouMarker(t *testing.T) {
	comment := &api.Comment{User: api.User{Login: "alice"}}
	if got := youMarker(comment, "alice"); got != " (你)" {
		t.Fatalf("marker = %q", got)
	}
	if got := youMarker(comment, "bob"); got != "" {
		t.Fatalf("marker = %q", got)
	}
	if got := youMarker(comment, ""); got != "" {
		t.Fatalf("empty user marker = %q", got)
	}
}

func TestDiffLocation(t *testing.T) {
	tests := []struct {
		name    string
		comment api.Comment
		want    string
	}{
		{name: "empty", want: ""},
		{name: "top-level path", comment: api.Comment{DiffFile: "main.go"}, want: "main.go"},
		{
			name:    "new line range",
			comment: api.Comment{DiffPosition: &api.DiffPosition{NewPath: "new.go", StartNewLine: 2, EndNewLine: 4}},
			want:    "new.go  L2–4",
		},
		{
			name:    "old line",
			comment: api.Comment{DiffPosition: &api.DiffPosition{OldPath: "old.go", StartOldLine: 8, EndOldLine: 8}},
			want:    "old.go  L8",
		},
		{
			name:    "end only",
			comment: api.Comment{Path: "file.go", DiffPosition: &api.DiffPosition{EndNewLine: 9}},
			want:    "file.go  L9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := diffLocation(&tt.comment); got != tt.want {
				t.Fatalf("diffLocation() = %q, want %q", got, tt.want)
			}
		})
	}
}
