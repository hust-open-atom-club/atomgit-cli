package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func isolateConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	return home
}

func writeCredentialsFile(t *testing.T, path string, credentials StoredCredentials) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestGetTokenFilePaths(t *testing.T) {
	home := isolateConfig(t)

	want := []string{
		filepath.Join(home, ".config", appName, tokenFile),
		filepath.Join(home, legacyTokenFile),
	}
	if got := getTokenFilePaths(); !equalStrings(got, want) {
		t.Fatalf("getTokenFilePaths() = %#v, want %#v", got, want)
	}

	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	want = []string{
		filepath.Join(xdg, appName, tokenFile),
		filepath.Join(home, legacyTokenFile),
	}
	if got := getTokenFilePaths(); !equalStrings(got, want) {
		t.Fatalf("getTokenFilePaths() with XDG = %#v, want %#v", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPrimaryTokenPath(t *testing.T) {
	home := isolateConfig(t)
	path, err := PrimaryTokenPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".config", appName, tokenFile); path != want {
		t.Fatalf("PrimaryTokenPath() = %q, want %q", path, want)
	}

	xdg := filepath.Join(t.TempDir(), "custom")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path, err = PrimaryTokenPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(xdg, appName, tokenFile); path != want {
		t.Fatalf("PrimaryTokenPath() = %q, want %q", path, want)
	}
}

func TestLoadStoredCredentials(t *testing.T) {
	t.Run("primary", func(t *testing.T) {
		home := isolateConfig(t)
		path := filepath.Join(home, ".config", appName, tokenFile)
		writeCredentialsFile(t, path, StoredCredentials{AccessToken: "primary", User: "alice"})

		got, err := LoadStoredCredentials()
		if err != nil {
			t.Fatal(err)
		}
		if got.AccessToken != "primary" || got.User != "alice" {
			t.Fatalf("credentials = %#v", got)
		}
	})

	t.Run("legacy fallback", func(t *testing.T) {
		home := isolateConfig(t)
		path := filepath.Join(home, legacyTokenFile)
		writeCredentialsFile(t, path, StoredCredentials{AccessToken: "legacy", User: "bob"})

		got, err := LoadStoredCredentials()
		if err != nil {
			t.Fatal(err)
		}
		if got.AccessToken != "legacy" {
			t.Fatalf("credentials = %#v", got)
		}
	})

	t.Run("missing", func(t *testing.T) {
		home := isolateConfig(t)
		_, err := LoadStoredCredentials()
		if !errors.Is(err, ErrTokenNotFound) {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(err.Error(), filepath.Join(home, legacyTokenFile)) {
			t.Fatalf("error does not list searched paths: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		home := isolateConfig(t)
		path := filepath.Join(home, ".config", appName, tokenFile)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadStoredCredentials(); err == nil || !strings.Contains(err.Error(), "failed to parse") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		home := isolateConfig(t)
		path := filepath.Join(home, ".config", appName, tokenFile)
		writeCredentialsFile(t, path, StoredCredentials{User: "alice"})
		if _, err := LoadStoredCredentials(); err == nil || !strings.Contains(err.Error(), "empty access_token") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("fixes group/other readable permissions (0o644)", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("permission bits are not enforced on Windows")
		}

		home := isolateConfig(t)
		path := filepath.Join(home, ".config", appName, tokenFile)
		creds := StoredCredentials{AccessToken: "leaked", User: "alice"}
		data, err := json.Marshal(creds)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}

		got, err := LoadStoredCredentials()
		if err != nil {
			t.Fatal(err)
		}
		if got.AccessToken != "leaked" || got.User != "alice" {
			t.Fatalf("credentials = %#v", got)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("perm = %#o, want 0600", perm)
		}
	})

	t.Run("preserves stricter owner-only permissions (0o400)", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("permission bits are not enforced on Windows")
		}

		home := isolateConfig(t)
		path := filepath.Join(home, ".config", appName, tokenFile)
		creds := StoredCredentials{AccessToken: "readonly", User: "alice"}
		data, err := json.Marshal(creds)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o400); err != nil {
			t.Fatal(err)
		}

		got, err := LoadStoredCredentials()
		if err != nil {
			t.Fatal(err)
		}
		if got.AccessToken != "readonly" || got.User != "alice" {
			t.Fatalf("credentials = %#v", got)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o400 {
			t.Fatalf("perm = %#o, want 0400", perm)
		}
	})

	t.Run("rejects symlink token file", func(t *testing.T) {
		home := isolateConfig(t)
		target := filepath.Join(home, "real-token.json")
		writeCredentialsFile(t, target, StoredCredentials{AccessToken: "secret", User: "alice"})

		linkPath := filepath.Join(home, ".config", appName, tokenFile)
		if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, linkPath); err != nil {
			t.Fatal(err)
		}

		_, err := LoadStoredCredentials()
		if !errors.Is(err, ErrTokenFileSymlink) {
			t.Fatalf("error = %v, want ErrTokenFileSymlink", err)
		}
	})
}

func TestSaveAndClearCredentials(t *testing.T) {
	isolateConfig(t)
	credentials := &StoredCredentials{
		AccessToken:  "access",
		User:         "alice",
		RefreshToken: "refresh",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}
	if err := SaveCredentials(credentials); err != nil {
		t.Fatal(err)
	}

	path, err := PrimaryTokenPath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}

	loaded, err := LoadStoredCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CreatedAt == 0 || loaded.RefreshToken != "refresh" {
		t.Fatalf("credentials = %#v", loaded)
	}

	removed, err := ClearCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != path {
		t.Fatalf("removed = %#v", removed)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("credential still exists: %v", err)
	}
}

func TestSaveCredentialsValidation(t *testing.T) {
	isolateConfig(t)
	tests := []struct {
		name        string
		credentials *StoredCredentials
		want        string
	}{
		{name: "nil", want: "credentials are nil"},
		{name: "empty token", credentials: &StoredCredentials{User: "alice"}, want: "access_token is empty"},
		{name: "empty user", credentials: &StoredCredentials{AccessToken: "token"}, want: "user is empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SaveCredentials(tt.credentials)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestSaveTokenPreservesOAuthFields(t *testing.T) {
	isolateConfig(t)
	if err := SaveCredentials(&StoredCredentials{
		AccessToken:  "old",
		User:         "old-user",
		RefreshToken: "refresh",
		ExpiresIn:    7200,
		TokenType:    "Bearer",
	}); err != nil {
		t.Fatal(err)
	}

	if err := SaveToken("new", "new-user"); err != nil {
		t.Fatal(err)
	}
	got, err := LoadStoredCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "new" || got.User != "new-user" || got.RefreshToken != "refresh" || got.ExpiresIn != 7200 {
		t.Fatalf("credentials = %#v", got)
	}
}

func TestNewConfigWithoutCredentials(t *testing.T) {
	isolateConfig(t)
	cfg, err := NewConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GetHost() != defaultHost {
		t.Fatalf("host = %q", cfg.GetHost())
	}
	if _, err := cfg.GetToken(); err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("GetToken error = %v", err)
	}
	if _, err := cfg.GetUser(); err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("GetUser error = %v", err)
	}
}

func TestIsPermissionErr(t *testing.T) {
	if !isPermissionErr(os.ErrPermission) {
		t.Fatal("os.ErrPermission should be recognized")
	}
	if isPermissionErr(errors.New("other")) {
		t.Fatal("unrelated error should not be recognized")
	}
}

func TestValidateTokenFilePerm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		perm    os.FileMode
		want    os.FileMode
		wantErr error
	}{
		{
			name: "allows safe 0o600 permissions",
			perm: 0o600,
			want: 0o600,
		},
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
			name:    "returns error when the file is not owner readable",
			perm:    0o044,
			wantErr: ErrTokenFileUnreadable,
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
