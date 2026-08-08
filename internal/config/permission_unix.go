//go:build !windows

package config

import (
	"errors"
	"fmt"
	"os"
)

// validateTokenFilePerm validates Unix permission bits for a credential file.
//
// It requires the owner read bit to be set and removes group and other
// permission bits.
//
// Returns:
//   - os.FileMode: the sanitized permissions containing only owner bits.
//   - error: ErrTokenFileUnreadable if the file is not readable by the owner.
func validateTokenFilePerm(perm os.FileMode) (os.FileMode, error) {
	if perm&0o400 == 0 {
		return 0, ErrTokenFileUnreadable
	}

	return perm & 0o700, nil
}

// validateAndFixTokenFilePerm validates and fixes Unix permission bits on a
// token file. It ensures the file is owner-readable and strips group/other
// permissions, fixing them via Chmod if needed.
func validateAndFixTokenFilePerm(f *os.File, path string, info os.FileInfo) error {
	perm := info.Mode().Perm()

	fixed, err := validateTokenFilePerm(perm)
	if err != nil {
		if errors.Is(err, ErrTokenFileUnreadable) {
			return &TokenPermissionError{Path: path, Err: err}
		}

		return fmt.Errorf("cannot read %s: %w", path, err)
	}
	if fixed != perm {
		if err := f.Chmod(fixed); err != nil {
			return tokenSecurePermissionError(path, err)
		}
	}

	return nil
}

// hint returns a Unix-specific hint that includes the chmod command to fix
// token file permissions.
func (e *TokenPermissionError) hint() string {
	return fmt.Sprintf("check ownership and set permissions with `chmod 600 %q`", e.Path)
}
