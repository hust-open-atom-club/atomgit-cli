package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func TestIsPermissionErr(t *testing.T) {
	if !isPermissionErr(os.ErrPermission) {
		t.Fatal("os.ErrPermission should be recognized")
	}
	if !isPermissionErr(syscall.EACCES) {
		t.Fatal("syscall.EACCES should be recognized")
	}
	if !isPermissionErr(syscall.EPERM) {
		t.Fatal("syscall.EPERM should be recognized")
	}
	if isPermissionErr(errors.New("other")) {
		t.Fatal("unrelated error should not be recognized")
	}
}

func TestPermissionErrors(t *testing.T) {
	path := filepath.Join("tmp", "ag-cli", "token.json")
	pathErr := &os.PathError{Op: "open", Path: path, Err: syscall.EACCES}

	t.Run("secure token file", func(t *testing.T) {
		err := tokenSecurePermissionError(path, pathErr)
		want := fmt.Sprintf(
			"cannot secure token file %s: permission denied\n"+
				"hint: make sure the file is owned by the current user",
			path,
		)
		if err.Error() != want {
			t.Fatalf("error = %q, want %q", err, want)
		}
	})

	t.Run("write token file", func(t *testing.T) {
		err := permissionError(
			"cannot write token file", path, pathErr,
			"check the file and parent directory ownership and owner write permission",
		)
		want := fmt.Sprintf(
			"cannot write token file %s: permission denied\n"+
				"hint: check the file and parent directory ownership and owner write permission",
			path,
		)
		if err.Error() != want {
			t.Fatalf("error = %q, want %q", err, want)
		}
	})

	t.Run("remove token file", func(t *testing.T) {
		err := permissionError(
			"cannot remove token file", path, pathErr,
			"check the file and parent directory ownership and permissions",
		)
		want := fmt.Sprintf(
			"cannot remove token file %s: permission denied\n"+
				"hint: check the file and parent directory ownership and permissions",
			path,
		)
		if err.Error() != want {
			t.Fatalf("error = %q, want %q", err, want)
		}
	})
}

func TestTokenPermissionError(t *testing.T) {
	path := filepath.Join("tmp", "ag-cli", "token.json")
	pathErr := &os.PathError{Op: "open", Path: path, Err: syscall.EACCES}

	t.Run("error message", func(t *testing.T) {
		err := &TokenPermissionError{Path: path, Err: pathErr}
		want := fmt.Sprintf("cannot read token file %s: permission denied\n", path)
		if runtime.GOOS == "windows" {
			want += "hint: check ownership and file read permissions"
		} else {
			want += fmt.Sprintf("hint: check ownership and set permissions with `chmod 600 %q`", path)
		}
		if err.Error() != want {
			t.Fatalf("error = %q, want %q", err, want)
		}
		if strings.Contains(err.Error(), "open "+path) {
			t.Fatalf("error repeats the underlying operation and path: %v", err)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		err := &TokenPermissionError{Path: path, Err: pathErr}
		if !errors.Is(err, syscall.EACCES) {
			t.Fatalf("error does not wrap EACCES: %v", err)
		}
	})

	t.Run("unwrap non-path-error", func(t *testing.T) {
		base := errors.New("some error")
		err := &TokenPermissionError{Path: path, Err: base}
		if !errors.Is(err, base) {
			t.Fatalf("error does not unwrap to base: %v", err)
		}
	})

	t.Run("error message without path error", func(t *testing.T) {
		err := &TokenPermissionError{Path: path, Err: os.ErrPermission}
		want := fmt.Sprintf("cannot read token file %s: permission denied\n", path)
		if runtime.GOOS == "windows" {
			want += "hint: check ownership and file read permissions"
		} else {
			want += fmt.Sprintf("hint: check ownership and set permissions with `chmod 600 %q`", path)
		}
		if err.Error() != want {
			t.Fatalf("error = %q, want %q", err, want)
		}
	})
}
