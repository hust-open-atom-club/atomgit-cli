package cmdutil

import (
	"errors"
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

// AuthenticationError preserves missing-credential errors and their causes.
// Other errors retain the legacy authentication prefix, without adding it twice.
func AuthenticationError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, config.ErrNotAuthenticated) {
		return err
	}
	if strings.HasPrefix(err.Error(), "not authenticated") {
		return err
	}
	return fmt.Errorf("not authenticated: %w", err)
}
