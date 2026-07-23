package cmdutil

import (
	"errors"
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
}
