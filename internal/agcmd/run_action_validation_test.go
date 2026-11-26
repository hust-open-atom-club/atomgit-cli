package agcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunRetryInvalidJobBeforeCredentialLoading(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "ag-cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "token.json"), []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "build/linux"} {
		args, err := json.Marshal([]string{"run", "retry", "team/demo", "run-1", "--job=" + id, "--yes"})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAPIErrorProcess$")
		cmd.Env = append(os.Environ(), "AG_API_ERROR_TEST_PROCESS=1", "AG_API_ERROR_TEST_ARGS="+string(args), "AG_API_ERROR_TEST_SCENARIO=execution")
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err = cmd.Run()
		cancel()
		exit, ok := errors.AsType[*exec.ExitError](err)
		if !ok || exit.ExitCode() != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "invalid job ID") || strings.Contains(stderr.String(), "Credential configuration") {
			t.Fatalf("id=%q error=%v stdout=%q stderr=%q", id, err, stdout.String(), stderr.String())
		}
	}
}
