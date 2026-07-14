package cmdutil

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

func isBidiControl(r rune) bool {
	return (r >= 0x202A && r <= 0x202E) ||
		(r >= 0x2066 && r <= 0x2069) ||
		r == 0x2028 || r == 0x2029 ||
		r == 0x061C ||
		r == 0x200E || r == 0x200F
}

// SanitizeTerminal replaces terminal control sequences and visually deceptive
// characters with visible escape notation, preventing terminal escape sequence
// injection (CWE-150).
//
// Allowed through unchanged: \n (LF), \t (HT), and all normal printable text.
// Replaced:
//
//	C0 controls (0x00–0x1F excl. \n \t)  → \xNN
//	DEL (0x7F)                           → \x7f
//	C1 controls (U+0080–U+009F)         → \u00NN
//	Unicode bidi / line-break controls   → \uNNNN
func SanitizeTerminal(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case unicode.IsControl(r):
			if r < 0x80 {
				fmt.Fprintf(&b, "\\x%02x", r)
			} else {
				fmt.Fprintf(&b, "\\u%04x", r)
			}
		case r == 0x7F:
			b.WriteString("\\x7f")
		case isBidiControl(r):
			fmt.Fprintf(&b, "\\u%04x", r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SanitizingWriter wraps an io.Writer and passes all written bytes through
// SanitizeTerminal before forwarding to the underlying writer.
//
// It buffers up to utf8.UTFMax trailing bytes when a Write chunk ends with
// an incomplete multi-byte UTF-8 sequence, joining them with the next Write
// so that CJK and emoji characters are never corrupted by io.Copy's 32 KB
// chunk boundary.
type SanitizingWriter struct {
	out    io.Writer
	trail  [utf8.UTFMax]byte
	nTrail int
}

func NewSanitizingWriter(out io.Writer) *SanitizingWriter {
	return &SanitizingWriter{out: out}
}

func (w *SanitizingWriter) Write(p []byte) (int, error) {
	n := len(p)
	if n == 0 {
		return 0, nil
	}

	var buf []byte
	if w.nTrail > 0 {
		buf = make([]byte, w.nTrail+n)
		copy(buf, w.trail[:w.nTrail])
		copy(buf[w.nTrail:], p)
		w.nTrail = 0
	} else {
		buf = p
	}

	if tail := incompleteUTF8Tail(buf); tail > 0 {
		w.nTrail = copy(w.trail[:], buf[len(buf)-tail:])
		buf = buf[:len(buf)-tail]
	}

	if len(buf) > 0 {
		_, err := io.WriteString(w.out, SanitizeTerminal(string(buf)))
		if err != nil {
			return 0, err
		}
	}
	return n, nil
}

// incompleteUTF8Tail returns the number of trailing bytes in p that begin
// a multi-byte UTF-8 rune whose continuation bytes have not arrived yet.
// Returns 0 when p ends on a complete rune boundary.
func incompleteUTF8Tail(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	end := len(p)
	start := end - 1
	for start > 0 && start > end-utf8.UTFMax && !utf8.RuneStart(p[start]) {
		start--
	}
	if p[start] < 0x80 {
		return 0
	}
	if utf8.FullRune(p[start:]) {
		return 0
	}
	return end - start
}
