package agcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestModuleUpdateUncertainDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		cause  error
		detail string
	}{
		{&api.HTTPError{StatusCode: 502, Body: "private-body", Status: "private-status"}, "The server returned HTTP 502."},
		{fmt.Errorf("private-body: %w", api.ErrInvalidResponse), "The server returned an invalid response."},
		{&api.RequestExecutionError{Err: errors.New("private-network")}, "The HTTP request could not be completed."},
		{&api.RequestExecutionError{Err: context.Canceled}, "The HTTP request was canceled."},
	} {
		for _, raw := range []bool{false, true} {
			var out bytes.Buffer
			writer := cmdutil.NewSanitizingWriter(&out)
			if err := writer.SetRaw(raw); err != nil {
				t.Fatal(err)
			}
			renderCommandError(writer, fmt.Errorf("edit failed: %w", &api.RepositoryModuleUpdateUncertainError{Err: tc.cause}))
			if err := cmdutil.FlushWriter(writer); err != nil {
				t.Fatal(err)
			}
			want := "Error: Repository module update could not be confirmed.\n\n" + tc.detail + "\n\nThe module settings may have changed. Check AtomGit before retrying.\n"
			if out.String() != want {
				t.Fatalf("raw=%t got=%q want=%q", raw, out.String(), want)
			}
		}
	}
}

func TestModuleNoChangesFailsBeforeCredentialLoading(t *testing.T) {
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
	args, marshalErr := json.Marshal([]string{"repo", "module", "edit", "team/demo", "--yes"})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAPIErrorProcess$")
	cmd.Env = append(os.Environ(), "AG_API_ERROR_TEST_PROCESS=1", "AG_API_ERROR_TEST_ARGS="+string(args), "AG_API_ERROR_TEST_SCENARIO=execution")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "at least one repository module switch must be provided") || strings.Contains(stderr.String(), "Credential configuration") {
		t.Fatalf("err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
}
