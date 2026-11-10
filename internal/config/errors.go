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

	// ErrCredentialAccess means a credential source could not be located or read.
	// It does not mean credentials are absent or their contents are invalid.
	ErrCredentialAccess = errors.New("credential access failed")

	// ErrInvalidCredentials means credential data or a file safety check was rejected.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrTokenFileSymlink is returned when a token file is a symlink
	ErrTokenFileSymlink = errors.New("token file is a symlink")

	// ErrTokenFileChanged is returned when a token file was changed during open.
	ErrTokenFileChanged = errors.New("token file was changed")

	// ErrTokenFileUnreadable is returned when a token file is not owner-readable.
	ErrTokenFileUnreadable = errors.New("token file is not owner-readable")

	// ErrTokenFileNotRegular is returned when inspection rejects a non-regular file.
	ErrTokenFileNotRegular = errors.New("credential file must be a regular file")

	// ErrTokenFileTooLarge is returned when inspection rejects an oversized file.
	ErrTokenFileTooLarge = errors.New("credential file exceeds size limit")
)

// FileAccessError records a failure locating or reading the configuration file.
type FileAccessError struct {
	Err error
}

func (e *FileAccessError) Error() string { return e.Err.Error() }
func (e *FileAccessError) Unwrap() error { return e.Err }

// FileDecodeError records a failure decoding configuration file contents.
type FileDecodeError struct {
	Err error
}

func (e *FileDecodeError) Error() string { return e.Err.Error() }
func (e *FileDecodeError) Unwrap() error { return e.Err }

// CredentialAccessError records a failed credential source lookup or read.
// Operation and Path describe the attempted access; Err retains its cause.
type CredentialAccessError struct {
	Operation string
	Path      string
	Err       error
}

func (e *CredentialAccessError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s: %v", e.Operation, e.Err)
	}
	return fmt.Sprintf("%s %s: %v", e.Operation, e.Path, e.Err)
}

func (e *CredentialAccessError) Unwrap() error        { return e.Err }
func (e *CredentialAccessError) Is(target error) bool { return target == ErrCredentialAccess }

// CredentialFileSafetyError records a credential file rejected by a safety check.
// Err retains the specific rejection reason independently of CLI guidance.
type CredentialFileSafetyError struct {
	Path string
	Err  error
}

func (e *CredentialFileSafetyError) Error() string {
	return fmt.Sprintf("unsafe credential file %s: %v", e.Path, e.Err)
}
func (e *CredentialFileSafetyError) Unwrap() error        { return e.Err }
func (e *CredentialFileSafetyError) Is(target error) bool { return target == ErrInvalidCredentials }

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
// file permission issues. It retains the original cause.
type TokenPermissionError struct {
	Path string
	Err  error
}

func (e *TokenPermissionError) Error() string {
	err := e.Err
	if pathErr, ok := errors.AsType[*os.PathError](err); ok {
		err = pathErr.Err
	}
	return fmt.Sprintf("cannot read token file %s: %v", e.Path, err)
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
