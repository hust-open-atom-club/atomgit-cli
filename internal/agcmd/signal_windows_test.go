//go:build windows

package agcmd

import (
	"os"
	"reflect"
	"testing"
)

func TestTerminationSignalsWindows(t *testing.T) {
	want := []os.Signal{os.Interrupt}
	if got := terminationSignals(); !reflect.DeepEqual(got, want) {
		t.Fatalf("termination signals = %v, want %v", got, want)
	}
}
