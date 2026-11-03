package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInspectCredentialsLegacyFallbackAndSymlink(t *testing.T) {
	home := t.TempDir()
	xdg := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	legacy := filepath.Join(home, legacyTokenFile)
	body := []byte(`{"user":"alice","access_token":"synthetic","expires_in":3600}`)
	if err := os.WriteFile(legacy, body, 0o600); err != nil {
		t.Fatal(err)
	}
	c, _, err := InspectCredentials()
	if err != nil || c.User != "alice" || c.CreatedAt != 0 {
		t.Fatalf("%+v %v", c, err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	primary, _ := PrimaryTokenPath()
	if err := os.MkdirAll(filepath.Dir(primary), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(legacy, primary); err != nil {
		t.Fatal(err)
	}
	if _, _, err := InspectCredentials(); err == nil {
		t.Fatal("symlink accepted or fell back to legacy")
	}
}

func TestInspectedFileIdentityPreservesStatFailure(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "credential")
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	_, statErr := f.Stat()
	other, err := os.Stat(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		after   os.FileInfo
		statErr error
		want    error
	}{
		{"stat failed", nil, statErr, ErrCredentialAccess},
		{"changed file", other, nil, ErrTokenFileChanged},
		{"same file", before, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInspectedFileIdentity(f.Name(), before, tc.after, tc.statErr)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if tc.statErr != nil {
				access, ok := errors.AsType[*CredentialAccessError](err)
				if !ok || !errors.Is(err, statErr) || access.Path != f.Name() || errors.Is(err, ErrTokenFileChanged) {
					t.Fatalf("stat failure lost its path/cause or became file change: %v", err)
				}
			}
		})
	}
}

func TestInspectCredentialPermissionFailure(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires Unix permissions and an unprivileged process")
	}
	home := isolateConfig(t)
	path, err := PrimaryTokenPath()
	if err != nil {
		t.Fatal(err)
	}
	writeCredentialsFile(t, path, StoredCredentials{AccessToken: "primary", User: "alice"})
	writeCredentialsFile(t, filepath.Join(home, legacyTokenFile), StoredCredentials{AccessToken: "legacy", User: "bob"})
	dir := filepath.Dir(path)
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Error(err)
		}
	})
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		load func() error
	}{
		{"read", func() error { _, err := readCredentialData(); return err }},
		{"inspection", func() error { _, _, err := InspectCredentials(); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.load()
			cause, ok := errors.AsType[*os.PathError](err)
			if !ok || cause.Path != path || !errors.Is(err, os.ErrPermission) || !errors.Is(err, ErrCredentialAccess) || errors.Is(err, ErrTokenNotFound) {
				t.Fatalf("error = %v, want permission access failure without legacy fallback", err)
			}
		})
	}
}
