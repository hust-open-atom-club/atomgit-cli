package cmdutil

import (
	"errors"
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

// AuthenticationError preserves the canonical login guidance without wrapping
// it twice.
//
// GetToken (and some command-local helpers) already produce an error whose
// message begins with "not authenticated". Naively wrapping such an error with
// fmt.Errorf("not authenticated: %w", err) yields a doubled
// "not authenticated: not authenticated: …" message. AuthenticationError
// instead returns the error unchanged in that case, so the user sees the
// "run `ag auth login`" hint exactly once.
//
// It recognises two equivalent shapes:
//   - the config.ErrNotAuthenticated sentinel (errors.Is), and
//   - any error whose message already carries the "not authenticated" prefix,
//     which covers callers that build their own equivalent string error.
func AuthenticationError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, config.ErrNotAuthenticated) {
		return config.ErrNotAuthenticated
	}
	if strings.HasPrefix(err.Error(), "not authenticated") {
		return err
	}
	return fmt.Errorf("not authenticated: %w", err)
}
