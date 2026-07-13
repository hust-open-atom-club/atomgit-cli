package cmdutil

import "os"

// IsTerminal returns true if the given file descriptor is attached to a
// terminal (TTY). Used to decide whether to sanitize --json output:
//   - TTY (human reading): sanitize to prevent escape sequence injection
//   - Pipe/redirect (machine reading, e.g. | jq): leave raw for tool compatibility
func IsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
