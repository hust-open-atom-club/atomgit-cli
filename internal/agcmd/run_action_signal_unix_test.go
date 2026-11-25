//go:build !windows

package agcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type runActionPromptCapture struct {
	buffer bytes.Buffer
	ready  chan struct{}
	once   sync.Once
}

func (w *runActionPromptCapture) Write(p []byte) (int, error) {
	n, err := w.buffer.Write(p)
	if strings.Contains(w.buffer.String(), "Apply this run action? [y/N] ") {
		w.once.Do(func() { close(w.ready) })
	}
	return n, err
}

func (w *runActionPromptCapture) String() string { return w.buffer.String() }

func TestRunActionSignalInterruptsConfirmation(t *testing.T) {
	for _, action := range []string{"cancel", "stop", "rerun", "retry"} {
		for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
			t.Run(action+"/"+sig.String(), func(t *testing.T) {
				home, xdg := t.TempDir(), t.TempDir()
				t.Setenv("HOME", home)
				t.Setenv("USERPROFILE", home)
				t.Setenv("XDG_CONFIG_HOME", xdg)
				args, err := json.Marshal([]string{"run", action, "team/demo", "run-1"})
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAgCommandProcess$")
				cmd.Env = append(os.Environ(), "AG_COMMAND_TEST_PROCESS=1", "AG_COMMAND_TEST_ARGS="+string(args))
				// Keep stdin open and send no answer, newline, or EOF. The helper
				// panics on any HTTP request, including retry's job discovery GET.
				stdin, input, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				defer stdin.Close()
				defer input.Close()
				cmd.Stdin = stdin
				var stdout bytes.Buffer
				stderr := &runActionPromptCapture{ready: make(chan struct{})}
				cmd.Stdout, cmd.Stderr = &stdout, stderr
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				done := make(chan error, 1)
				go func() { done <- cmd.Wait() }()
				select {
				case <-stderr.ready:
					if err := cmd.Process.Signal(sig); err != nil {
						t.Fatal(err)
					}
				case err := <-done:
					t.Fatalf("process exited before confirmation: %v", err)
				case <-ctx.Done():
					<-done
					t.Fatal("process did not prompt")
				}
				err = <-done
				exit, ok := errors.AsType[*exec.ExitError](err)
				if ctx.Err() != nil || !ok || exit.ExitCode() != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "context canceled") || strings.Contains(stderr.String(), "panic") {
					t.Fatalf("ctx=%v err=%v stdout=%q stderr=%q", ctx.Err(), err, stdout.String(), stderr.String())
				}
			})
		}
	}
}
