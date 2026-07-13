package cmdutil

import (
	"fmt"
	"strings"
	"unicode"
)

// isBidiControl returns true for Unicode bidirectional control characters
// and line/paragraph separators that can be used to forge text display.
//
// U+202A LEFT-TO-RIGHT EMBEDDING
// U+202B RIGHT-TO-LEFT EMBEDDING
// U+202C POP DIRECTIONAL FORMATTING
// U+202D LEFT-TO-RIGHT OVERRIDE
// U+202E RIGHT-TO-LEFT OVERRIDE
// U+2066 LEFT-TO-RIGHT ISOLATE
// U+2067 RIGHT-TO-LEFT ISOLATE
// U+2068 FIRST STRONG ISOLATE
// U+2069 POP DIRECTIONAL ISOLATE
// U+2028 LINE SEPARATOR (may inject extra newlines)
// U+2029 PARAGRAPH SEPARATOR (may inject extra newlines)
func isBidiControl(r rune) bool {
	return (r >= 0x202A && r <= 0x202E) ||
		(r >= 0x2066 && r <= 0x2069) ||
		r == 0x2028 || r == 0x2029
}

// SanitizeTerminal strips terminal control sequences and visually deceptive
// characters from a string before it is written to the terminal, preventing
// terminal escape sequence injection (CWE-150).
//
// Allowed: \n (LF), \t (HT)
// Replaced with visible escape notation:
//   - C0 controls (0x00–0x08, 0x0B–0x0C, 0x0E–0x1F)    → \xNN
//   - DEL (0x7F)                                         → \x7f
//   - C1 controls (U+0080–U+009F)                        → \u00NN
//   - Unicode bidi controls (U+202A–U+202E, U+2066–U+2069) → \uNNNN
//   - Line/paragraph separators (U+2028, U+2029)         →   /
//
// The visible escape notation preserves debuggability while being safe for
// terminal output — the backslash is a printable character that does not
// trigger any terminal control function.
//
// Uses unicode.IsControl for C0/C1 detection which correctly handles both
// 8-bit raw bytes and their UTF-8 encoded equivalents.
func SanitizeTerminal(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case unicode.IsControl(r):
			// C0 (0x00–0x1F, except \n \t) and C1 (U+0080–U+009F)
			if r < 0x80 {
				fmt.Fprintf(&b, "\\x%02x", r)
			} else {
				fmt.Fprintf(&b, "\\u%04x", r)
			}
		case r == 0x7F:
			// DEL — not covered by unicode.IsControl
			b.WriteString("\\x7f")
		case isBidiControl(r):
			fmt.Fprintf(&b, "\\u%04x", r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
