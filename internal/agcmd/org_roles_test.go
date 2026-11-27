package agcmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestOrgRolesValidationProcess runs the real entry point with a malformed
// organization path so the driver test can observe which failure wins.
func TestOrgRolesValidationProcess(_ *testing.T) {
	if os.Getenv("AG_ORG_ROLES_TEST_PROCESS") != "1" {
		return
	}
	os.Args = []string{"ag-cli", "org", "roles", "a/b"}
	os.Exit(Main())
}

// TestOrgRolesValidatesArgumentsBeforeCredentials mirrors the review repro: a
// corrupt token.json must not hide a malformed organization argument, because
// the entry point loads credential configuration in PersistentPreRunE.
func TestOrgRolesValidatesArgumentsBeforeCredentials(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	dir := filepath.Join(xdg, "ag-cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "token.json"), []byte(`{invalid`), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOrgRolesValidationProcess$")
	cmd.Env = append(os.Environ(),
		"AG_ORG_ROLES_TEST_PROCESS=1",
		"HOME="+home,
		"USERPROFILE="+home,
		"XDG_CONFIG_HOME="+xdg,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected a non-zero exit: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "invalid organization") {
		t.Fatalf("missing argument diagnostics: %q", combined)
	}
	if strings.Contains(strings.ToLower(combined), "credential") {
		t.Fatalf("credential configuration was loaded before argument validation: %q", combined)
	}
}
