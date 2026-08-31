//go:build !windows

package agcmd

import (
	"os"
	"reflect"
	"syscall"
	"testing"
)

func TestTerminationSignalsUnix(t *testing.T) {
	want := []os.Signal{os.Interrupt, syscall.SIGTERM}
	if got := terminationSignals(); !reflect.DeepEqual(got, want) {
		t.Fatalf("termination signals = %v, want %v", got, want)
	}
}
