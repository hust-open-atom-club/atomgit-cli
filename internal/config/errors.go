package config

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

var (
	// ErrNotAuthenticated is the canonical error returned when a command
	// requires stored credentials but none are available. It is a sentinel so
	// callers can match it with errors.Is, and its message doubles as the
	// actionable "run `ag auth login`" guidance shown to the user.
	ErrNotAuthenticated = errors.New("not authenticated: run `ag auth login`")

	// ErrTokenNotFound is returned when no token file exists in any search path.
	ErrTokenNotFound = errors.New("token file not found")

	// ErrTokenFileSymlink is returned when a token file is a symlink
	ErrTokenFileSymlink = errors.New("token file is a symlink")

	// ErrTokenFileChanged is returned when a token file was changed during open.
	ErrTokenFileChanged = errors.New("token file was changed")

	// ErrTokenFileUnreadable is returned when a token file is not owner-readable.
	ErrTokenFileUnreadable = errors.New("token file is not owner-readable")
)

// TokenPermissionError is returned when a token file cannot be read due to
// file permission issues. The hint varies by platform.
type TokenPermissionError struct {
	Path string
	Err  error
}

func (e *TokenPermissionError) Error() string {
	var pathErr *os.PathError
	err := e.Err
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return fmt.Sprintf("cannot read token file %s: %v\nhint: %s", e.Path, err, e.hint())
}

func (e *TokenPermissionError) Unwrap() error {
	return e.Err
}

// isPermissionErr reports whether err is a permission-related error.
func isPermissionErr(err error) bool {
	return errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)
}

// tokenSecurePermissionError returns an error indicating that the token file
// could not be secured (e.g., Chmod failed).
func tokenSecurePermissionError(path string, err error) error {
	return permissionError(
		"cannot secure token file", path, err,
		"make sure the file is owned by the current user",
	)
}

// permissionError formats a permission-related error with a hint message.
func permissionError(action, path string, err error, hint string) error {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}

	return fmt.Errorf("%s %s: %w\nhint: %s", action, path, err, hint)
}
