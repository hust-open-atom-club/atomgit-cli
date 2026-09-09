//go:build !windows

package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateTokenFilePerm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		perm    os.FileMode
		want    os.FileMode
		wantErr error
	}{
		// Regression cases from issue requirements and review feedback.
		{
			name: "fixes group/other readable permissions",
			perm: 0o644,
			want: 0o600,
		},
		{
			name: "preserves stricter owner-only permissions",
			perm: 0o400,
			want: 0o400,
		},
		{
			name:    "returns error when owner has no read permission",
			perm:    0o044,
			wantErr: ErrTokenFileUnreadable,
		},
		{
			name:    "returns error when owner has no read permission but has write permission",
			perm:    0o200,
			wantErr: ErrTokenFileUnreadable,
		},
		// Pairwise-generated cases over Unix permission bits:
		// owner/group/other read/write/execute bits.
		{
			name: "rwxrwxrwx",
			perm: 0o777,
			want: 0o700,
		},
		{
			name:    "--------x",
			perm:    0o001,
			wantErr: ErrTokenFileUnreadable,
		},
		{
			name:    "-w-rwx---",
			perm:    0o270,
			wantErr: ErrTokenFileUnreadable,
		},
		{
			name: "r-x------",
			perm: 0o500,
			want: 0o500,
		},
		{
			name: "r--rw----",
			perm: 0o460,
			want: 0o400,
		},
		{
			name:    "-wx--xr--",
			perm:    0o314,
			wantErr: ErrTokenFileUnreadable,
		},
		{
			name:    "-wx------",
			perm:    0o300,
			wantErr: ErrTokenFileUnreadable,
		},
		{
			name: "r--rwx---",
			perm: 0o470,
			want: 0o400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, gotErr := validateTokenFilePerm(tt.perm)
			if gotErr != nil {
				if tt.wantErr == nil {
					t.Errorf("validateTokenFilePerm() failed: %v", gotErr)
				}
				if !errors.Is(gotErr, tt.wantErr) {
					t.Errorf("validateTokenFilePerm() = %v, want %v", gotErr, tt.wantErr)
				}
				return
			}
			if tt.wantErr != nil {
				t.Fatal("validateTokenFilePerm() succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("validateTokenFilePerm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAndFixTokenFilePerm(t *testing.T) {
	t.Run("fixes group/other readable permissions", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "token.json")
		if err := os.WriteFile(path, []byte(`{"access_token":"test"}`), 0o644); err != nil {
			t.Fatal(err)
		}

		// Open read-write so Chmod can be applied on all kernels.
		f, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}

		if err := validateAndFixTokenFilePerm(f, path, info); err != nil {
			t.Fatalf("validateAndFixTokenFilePerm() = %v", err)
		}

		infoAfter, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := infoAfter.Mode().Perm(); perm != 0o600 {
			t.Fatalf("perm = %#o, want 0600", perm)
		}
	})

	t.Run("preserves stricter owner-only permissions", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "token.json")
		if err := os.WriteFile(path, []byte(`{"access_token":"test"}`), 0o400); err != nil {
			t.Fatal(err)
		}

		// 0o400 is read-only for the owner; open read-only — no Chmod needed.
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}

		if err := validateAndFixTokenFilePerm(f, path, info); err != nil {
			t.Fatalf("validateAndFixTokenFilePerm() = %v", err)
		}

		infoAfter, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := infoAfter.Mode().Perm(); perm != 0o400 {
			t.Fatalf("perm = %#o, want 0400", perm)
		}
	})

	t.Run("rejects unreadable file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "token.json")
		if err := os.WriteFile(path, []byte(`{"access_token":"test"}`), 0o200); err != nil {
			t.Fatal(err)
		}

		// 0o200 is write-only; open write-only so we can Stat and inspect.
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}

		err = validateAndFixTokenFilePerm(f, path, info)
		if err == nil {
			t.Fatal("validateAndFixTokenFilePerm() succeeded unexpectedly")
		}
		var tokenErr *TokenPermissionError
		if !errors.As(err, &tokenErr) {
			t.Fatalf("error is %T, want *TokenPermissionError: %v", err, err)
		}
	})
}
