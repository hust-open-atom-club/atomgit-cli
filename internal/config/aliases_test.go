package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
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

// TestConcurrentAliasUpdatesHelper is the subprocess entry point for
// TestConcurrentAliasUpdates. It saves a single alias in an isolated
// environment and reports success or failure to the parent.
func TestConcurrentAliasUpdatesHelper(t *testing.T) {
	name := os.Getenv("ALIAS_HELPER_NAME")
	if name == "" {
		t.Skip("helper subprocess only")
	}
	if err := SaveAlias(name, "pr list"); err != nil {
		t.Fatalf("SaveAlias(%q) error = %v", name, err)
	}
}

// TestConcurrentAliasUpdates spawns many real processes that save aliases at
// the same time and verifies none of the writes is lost. This is a
// regression test for concurrent read-modify-write updates of the alias
// config that previously silently overwrote each other.
func TestConcurrentAliasUpdates(t *testing.T) {
	if os.Getenv("ALIAS_HELPER_NAME") != "" {
		t.Skip("helper subprocess only")
	}
	withTempConfig(t)

	const n = 24
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("alias-%03d", i)
			cmd := exec.Command(os.Args[0], "-test.run=^TestConcurrentAliasUpdatesHelper$")
			cmd.Env = append(os.Environ(),
				"XDG_CONFIG_HOME="+os.Getenv("XDG_CONFIG_HOME"),
				"ALIAS_HELPER_NAME="+name,
			)
			if out, err := cmd.CombinedOutput(); err != nil {
				errCh <- fmt.Errorf("helper for %s failed: %v: %s", name, err, out)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}

	aliases, err := LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if len(aliases) != n {
		t.Errorf("len(aliases) = %d, want %d (concurrent updates lost)", len(aliases), n)
	}
	for i := 0; i < n; i++ {
		if _, ok := aliases[fmt.Sprintf("alias-%03d", i)]; !ok {
			t.Errorf("alias-%03d missing after concurrent saves", i)
		}
	}
}
