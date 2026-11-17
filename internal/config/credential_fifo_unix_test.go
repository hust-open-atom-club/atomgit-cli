//go:build !windows

package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// TestCredentialLoaderRejectsFIFOWithoutBlocking protects the pre-open regular
// file check: os.Open on a FIFO waits for a writer, so a replaced token.json
// could otherwise hang every command.
func TestCredentialLoaderRejectsFIFOWithoutBlocking(t *testing.T) {
	isolateConfig(t)
	path, err := PrimaryTokenPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(path, 0o600); err != nil {
		t.Skipf("mkfifo is unavailable: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := readCredentialData()
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, ErrTokenFileNotRegular) || !errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrCredentialAccess) {
			t.Fatalf("error = %v, want non-regular file rejection", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("readCredentialData blocked while opening a FIFO credential file")
	}
}
