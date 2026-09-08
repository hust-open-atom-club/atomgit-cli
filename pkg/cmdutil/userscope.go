package cmdutil

import (
	"fmt"
	"strings"
)

// UserScopeFlag validates a user-scoped list flag. Only @me is supported: the
// authenticated-user endpoints filter by the token owner, and there is no
// server-side filter for arbitrary usernames.
func UserScopeFlag(flag, value string) (bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return false, nil
	}
	if value != "@me" {
		return false, fmt.Errorf("%s only supports @me, got %q", flag, value)
	}
	return true, nil
}
