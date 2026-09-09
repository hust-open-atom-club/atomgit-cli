//go:build !windows

package config

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// fileLock is an exclusive advisory lock held on an open file descriptor.
type fileLock struct {
	f *os.File
}

// lockFile opens (creating if needed) the file at path and takes an
// exclusive advisory lock on it, blocking until the lock is available. The
// returned lock must be released when the guarded transaction finishes.
func lockFile(path string) (*fileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open alias lock file %s: %w", path, err)
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("lock alias config: %w", err)
	}
	return &fileLock{f: f}, nil
}

// release drops the lock and closes the underlying file.
func (l *fileLock) release() error {
	unlockErr := unix.Flock(int(l.f.Fd()), unix.LOCK_UN)
	closeErr := l.f.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

// syncDir fsyncs a directory so a preceding rename is durable.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
