//go:build windows

package config

import (
	"os"
	"testing"
)

func TestValidateAndFixTokenFilePermWindows(t *testing.T) {
	// On Windows, this should always be a no-op and never return an error.
	if err := validateAndFixTokenFilePerm(nil, `C:\Users\test\token.json`, nil); err != nil {
		t.Fatalf("validateAndFixTokenFilePerm() = %v, want nil", err)
	}
}

func TestTokenPermissionErrorHintWindows(t *testing.T) {
	err := &TokenPermissionError{Path: `C:\Users\test\token.json`, Err: os.ErrPermission}
	want := `cannot read token file C:\Users\test\token.json: permission denied
hint: check ownership and file read permissions`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}
