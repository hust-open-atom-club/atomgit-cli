package config

import (
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
