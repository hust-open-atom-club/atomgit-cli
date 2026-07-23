package cmdutil

import (
	"errors"
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

// AuthenticationError preserves the canonical login guidance without wrapping it twice.
func AuthenticationError(err error) error {
	if errors.Is(err, config.ErrNotAuthenticated) {
		return config.ErrNotAuthenticated
	}
	return fmt.Errorf("not authenticated: %w", err)
}
