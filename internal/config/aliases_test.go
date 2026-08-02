package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// withTempConfig redirects the XDG config directory to a fresh temp dir for
// the duration of a test.
func withTempConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestLoadAliasesMissingFile(t *testing.T) {
	withTempConfig(t)
	aliases, err := LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if len(aliases) != 0 {
		t.Fatalf("LoadAliases() = %v, want empty map", aliases)
	}
}

func TestSaveAndLoadAliases(t *testing.T) {
	withTempConfig(t)
	if err := SaveAlias("pl", "pr list"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}
	if err := SaveAlias("rv", "repo view"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}

	aliases, err := LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if aliases["pl"] != "pr list" {
		t.Errorf("aliases[pl] = %q, want %q", aliases["pl"], "pr list")
	}
	if aliases["rv"] != "repo view" {
		t.Errorf("aliases[rv] = %q, want %q", aliases["rv"], "repo view")
	}
}

func TestSaveAliasOverwrites(t *testing.T) {
	withTempConfig(t)
	if err := SaveAlias("pl", "pr list"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}
	if err := SaveAlias("pl", "pr list --state open"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}

	aliases, err := LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if len(aliases) != 1 {
		t.Fatalf("len(aliases) = %d, want 1", len(aliases))
	}
	if aliases["pl"] != "pr list --state open" {
		t.Errorf("aliases[pl] = %q, want %q", aliases["pl"], "pr list --state open")
	}
}

func TestDeleteAlias(t *testing.T) {
	withTempConfig(t)
	if err := SaveAlias("pl", "pr list"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}

	deleted, err := DeleteAlias("pl")
	if err != nil {
		t.Fatalf("DeleteAlias() error = %v", err)
	}
	if !deleted {
		t.Fatal("DeleteAlias() = false, want true for existing alias")
	}

	aliases, err := LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if len(aliases) != 0 {
		t.Fatalf("len(aliases) = %d, want 0 after delete", len(aliases))
	}
}

func TestDeleteAliasNotFound(t *testing.T) {
	withTempConfig(t)
	deleted, err := DeleteAlias("missing")
	if err != nil {
		t.Fatalf("DeleteAlias() error = %v", err)
	}
	if deleted {
		t.Fatal("DeleteAlias() = true, want false for missing alias")
	}
}

func TestAliasFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not enforced the same way on Windows")
	}
	withTempConfig(t)
	if err := SaveAlias("pl", "pr list"); err != nil {
		t.Fatalf("SaveAlias() error = %v", err)
	}
	path, err := AliasFilePath()
	if err != nil {
		t.Fatalf("AliasFilePath() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("alias file permissions = %o, want 600", got)
	}
	dir := filepath.Dir(path)
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("os.Stat(dir) error = %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("alias config dir permissions = %o, want 700", got)
	}
}
