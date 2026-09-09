//go:build windows

package config

import "os"

// validateAndFixTokenFilePerm is a no-op on Windows, where Unix permission
// bits are not enforced.
func validateAndFixTokenFilePerm(*os.File, string, os.FileInfo) error {
	return nil
}

// hint returns a Windows-friendly hint that does not reference Unix-specific
// chmod commands.
func (e *TokenPermissionError) hint() string {
	return "check ownership and file read permissions"
}
