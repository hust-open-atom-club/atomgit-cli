package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratorWithIsolatedBrokenConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "ag-cli"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "ag-cli", "token.json"), []byte("broken config"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "manuals")
	args := []string{"--output", output, "--version", "v1.2.3", "--date", "2026-10-07"}
	if err := run(args, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := run(append(args, "--check"), io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ag-cli.1", "ag-cli-auth-login.1", "ag-cli-pr-create.1"} {
		if _, err := os.Stat(filepath.Join(output, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := run([]string{"--output", t.TempDir(), "--check"}, io.Discard); err == nil {
		t.Fatal("accepted missing output")
	}
	for _, invalid := range [][]string{{"positional"}, {"--unknown"}, {"--output", output, "--date", "invalid"}} {
		if err := run(invalid, io.Discard); err == nil {
			t.Fatal("accepted invalid generator options")
		}
	}
}
