package cmdutil

import (
	"fmt"
	"strings"
)

// ValidateRef checks that a commit SHA, branch, or tag reference is safe to
// embed in an API path: non-empty, free of control characters and characters
// Git reserves for revision syntax.
func ValidateRef(value, name string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s cannot be empty", name)
	}
	if value == "@" || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "..") || strings.Contains(value, "@{") || strings.Contains(value, "//") {
		return "", fmt.Errorf("invalid %s %q", name, value)
	}
	for _, part := range strings.Split(value, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return "", fmt.Errorf("invalid %s %q", name, value)
		}
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*[\\", r) {
			return "", fmt.Errorf("invalid %s %q", name, value)
		}
	}
	return value, nil
}
