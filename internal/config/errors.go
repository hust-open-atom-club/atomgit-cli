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
	// callers can match it with errors.Is. Recovery advice belongs to CLI output.
	ErrNotAuthenticated = errors.New("credentials are missing")

	// ErrTokenNotFound is returned when no token file exists in any search path.
	ErrTokenNotFound = errors.New("token file not found")

	// ErrInvalidCredentials means credential data or a file safety check was rejected.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrTokenFileSymlink is returned when a token file is a symlink
	ErrTokenFileSymlink = errors.New("token file is a symlink")

	// ErrTokenFileChanged is returned when a token file was changed during open.
	ErrTokenFileChanged = errors.New("token file was changed")

	// ErrTokenFileUnreadable is returned when a token file is not owner-readable.
	ErrTokenFileUnreadable = errors.New("token file is not owner-readable")
)

// CredentialFormatError describes a credential decoder failure. Offset and
// InvalidFieldType are format facts exposed by config, independently of the
// concrete parser cause retained in Err.
type CredentialFormatError struct {
	Offset           int64
	InvalidFieldType bool
	Err              error
}

func (e *CredentialFormatError) Error() string {
	return fmt.Sprintf("failed to parse token file: %v; cause: %v", ErrInvalidCredentials, e.Err)
}
func (e *CredentialFormatError) Unwrap() error        { return e.Err }
func (e *CredentialFormatError) Is(target error) bool { return target == ErrInvalidCredentials }

// TokenPermissionError is returned when a token file cannot be read due to
// file permission issues. The hint varies by platform.
type TokenPermissionError struct {
	Path string
	Err  error
}

func (e *TokenPermissionError) Error() string {
	err := e.Err
	if pathErr, ok := errors.AsType[*os.PathError](err); ok {
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
	if pathErr, ok := errors.AsType[*os.PathError](err); ok {
		err = pathErr.Err
	}

	return fmt.Errorf("%s %s: %w\nhint: %s", action, path, err, hint)
}
