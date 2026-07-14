package cmdutil

import (
	"bytes"
	"testing"
)

func TestSanitizeTerminal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Safe — must pass through unchanged
		{name: "plain text", input: "Hello, World!", want: "Hello, World!"},
		{name: "newline and tab kept", input: "line1\nline2\tindented", want: "line1\nline2\tindented"},
		{name: "CJK preserved", input: "你好世界", want: "你好世界"},
		{name: "emoji preserved", input: "🎉 test 🎉", want: "🎉 test 🎉"},

		// C0 controls → \xNN
		{name: "ESC BEL (OSC 52 clipboard)", input: "code\x1b]52;c;ZWNobyBQV05FRA==\x07more",
			want: "code\\x1b]52;c;ZWNobyBQV05FRA==\\x07more"},
		{name: "CR NULL BEL combined", input: "a\x0db\x00c\x07d", want: "a\\x0db\\x00c\\x07d"},

		// C1 → \u00NN (valid UTF-8, caught by unicode.IsControl)
		{name: "C1 CSI stripped", input: "text1mstyled", want: "text\\u009b1mstyled"},

		// DEL
		{name: "DEL stripped", input: "del\x7Fchar", want: "del\\x7fchar"},

		// Bidi controls → \uNNNN
		{name: "RLO bidi stripped", input: "text‮over", want: "text\\u202eover"},
		{name: "LRI bidi stripped", input: "text⁦iso", want: "text\\u2066iso"},
		{name: "line separator stripped", input: "line break", want: "line\\u2028break"},
		{name: "ALM stripped", input: "text؜dir", want: "text\\u061cdir"},
		{name: "LRM stripped", input: "text‎dir", want: "text\\u200edir"},
		{name: "RLM stripped", input: "text‏dir", want: "text\\u200fdir"},

		// Realistic attack payloads
		{name: "diff line with OSC 52", input: "  result();  \x1b]52;c;ZWNobyBQV05FRA==\x07",
			want: "  result();  \\x1b]52;c;ZWNobyBQV05FRA==\\x07"},
		{name: "SGR conceal", input: "visible\x1b[8mhidden\x1b[0mvisible", want: "visible\\x1b[8mhidden\\x1b[0mvisible"},

		// Edge
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeTerminal(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeTerminal(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeTerminal_NoPanic(t *testing.T) {
	for _, input := range []string{
		"\x00\x00\x00",
		"\x1b\x1b\x1b",
		"",
		"‮⁦⁩",
	} {
		_ = SanitizeTerminal(input)
	}
}

func TestSanitizingWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewSanitizingWriter(&buf)

	_, err := w.Write([]byte("safe text\n"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write([]byte("has \x1b[8m hidden \x1b[0m end\n"))
	if err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	want := "safe text\nhas \\x1b[8m hidden \\x1b[0m end\n"
	if got != want {
		t.Errorf("SanitizingWriter output = %q, want %q", got, want)
	}
}

func TestSanitizingWriter_UTF8ChunkBoundary(t *testing.T) {
	// "你好" is 6 bytes: \xe4\xbd\xa0 \xe5\xa5\xbd
	// Split after byte 4 so "你" is complete but "好" is split mid-character.
	full := []byte("你好")

	tests := []struct {
		name   string
		chunks [][]byte
		want   string
	}{
		{
			name:   "CJK split after first byte of second char",
			chunks: [][]byte{full[:4], full[4:]},
			want:   "你好",
		},
		{
			name:   "CJK split after second byte of second char",
			chunks: [][]byte{full[:5], full[5:]},
			want:   "你好",
		},
		{
			name:   "emoji split mid-character",
			chunks: [][]byte{[]byte("x\xf0\x9f"), []byte("\x98\x80y")},
			want:   "x\U0001f600y",
		},
		{
			name:   "three-way split of single CJK char",
			chunks: [][]byte{{0xe4}, {0xbd}, {0xa0}},
			want:   "你",
		},
		{
			name:   "no split needed for ASCII",
			chunks: [][]byte{[]byte("abc"), []byte("def")},
			want:   "abcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewSanitizingWriter(&buf)
			for _, chunk := range tt.chunks {
				if _, err := w.Write(chunk); err != nil {
					t.Fatal(err)
				}
			}
			got := buf.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
