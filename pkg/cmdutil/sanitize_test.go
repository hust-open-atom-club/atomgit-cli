package cmdutil

import "testing"

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

		// Realistic attack payloads — must NOT produce raw escape bytes
		{name: "diff line with OSC 52", input: "  result();  \x1b]52;c;ZWNobyBQV05FRA==\x07",
			want: "  result();  \\x1b]52;c;ZWNobyBQV05FRA==\\x07"},
		{name: "PR title with OSC 0", input: "Add\x1b]0;FAKE\x07 feature", want: "Add\\x1b]0;FAKE\\x07 feature"},
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
		"",  // C1
		"‮⁦⁩",  // bidi
	} {
		_ = SanitizeTerminal(input)
	}
}
