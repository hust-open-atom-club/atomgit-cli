//go:build windows

package config

import (
	"errors"
	"syscall"
)

func isSymlinkPrivilegeNotHeld(err error) bool {
	return errors.Is(err, syscall.Errno(1314))
}
