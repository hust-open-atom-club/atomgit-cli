//go:build windows

package config

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// fileLock is an exclusive byte-range lock held on an open file handle.
type fileLock struct {
	f  *os.File
	ol windows.Overlapped
}

// lockFile opens (creating if needed) the file at path and takes an
// exclusive byte-range lock on it, blocking until the lock is available. The
// returned lock must be released when the guarded transaction finishes.
func lockFile(path string) (*fileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open alias lock file %s: %w", path, err)
	}
	l := &fileLock{f: f}
	if err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &l.ol); err != nil {
		f.Close()
		return nil, fmt.Errorf("lock alias config: %w", err)
	}
	return l, nil
}

// release drops the lock and closes the underlying file.
func (l *fileLock) release() error {
	unlockErr := windows.UnlockFileEx(windows.Handle(l.f.Fd()), 0, 1, 0, &l.ol)
	closeErr := l.f.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

// syncDir is a no-op on Windows, where directories cannot be fsynced.
func syncDir(dir string) error {
	return nil
}
