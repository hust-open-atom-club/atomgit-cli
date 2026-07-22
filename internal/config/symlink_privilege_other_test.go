//go:build !windows

package config

func isSymlinkPrivilegeNotHeld(error) bool {
	return false
}
