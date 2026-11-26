package commandmeta

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type failedDiagnosticWriter struct{}

func (failedDiagnosticWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestDeprecationWarningFailurePreservesHooks(t *testing.T) {
	for _, useErrorHook := range []bool{false, true} {
		t.Run(map[bool]string{false: "PreRun", true: "PreRunE"}[useErrorHook], func(t *testing.T) {
			called, ran := false, false
			cmd := &cobra.Command{Use: "old", Short: "Old command", Run: func(*cobra.Command, []string) { ran = true }}
			if useErrorHook {
				cmd.PreRunE = func(*cobra.Command, []string) error { called = true; return nil }
			} else {
				cmd.PreRun = func(*cobra.Command, []string) { called = true }
			}
			Deprecate(cmd, "use new instead")
			cmd.SetErr(failedDiagnosticWriter{})
			cmd.SetOut(io.Discard)
			cmd.SetArgs([]string{})
			if err := cmd.Execute(); err != nil || !called || !ran {
				t.Fatalf("error=%v hook=%t run=%t", err, called, ran)
			}
			if !strings.Contains(cmd.Long, "Deprecated: use new instead") || Deprecation(cmd) != "use new instead" {
				t.Fatalf("metadata/description lost: %q", cmd.Long)
			}
		})
	}
}

func TestDeprecationPreservesHookErrorsAndLongDescription(t *testing.T) {
	cause := errors.New("existing pre-run failure")
	cmd := &cobra.Command{Use: "old", Long: "Existing description", PreRunE: func(*cobra.Command, []string) error { return cause }}
	Deprecate(cmd, "use new instead")
	cmd.SetErr(failedDiagnosticWriter{})
	if err := cmd.PreRunE(cmd, nil); !errors.Is(err, cause) {
		t.Fatalf("error=%v", err)
	}
	if cmd.Long != "Existing description\n\nDeprecated: use new instead" {
		t.Fatalf("Long=%q", cmd.Long)
	}
}
