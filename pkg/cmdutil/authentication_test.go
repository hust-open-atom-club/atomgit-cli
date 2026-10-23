package cmdutil

import (
	"errors"
	"fmt"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
)

func TestAuthenticationError(t *testing.T) {
	if err := AuthenticationError(config.ErrNotAuthenticated); !errors.Is(err, config.ErrNotAuthenticated) || err.Error() != config.ErrNotAuthenticated.Error() {
		t.Fatalf("canonical error = %q", err)
	}

	cause := errors.New("credential store unavailable")
	err := AuthenticationError(cause)
	if !errors.Is(err, cause) || err.Error() != "not authenticated: credential store unavailable" {
		t.Fatalf("wrapped error = %q", err)
	}

	missing := fmt.Errorf("load token: %w; lookup: %w", config.ErrNotAuthenticated, cause)
	//nolint:errorlint // Verify exact passthrough, including wrapper identity and its lookup cause.
	if got := AuthenticationError(missing); got != missing || !errors.Is(got, cause) {
		t.Fatalf("missing credential error = %v, want original error and lookup cause", got)
	}
}
