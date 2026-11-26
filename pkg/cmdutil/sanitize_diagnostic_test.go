package cmdutil

import (
	"bytes"
	"io"
	"testing"
)

type failedOptionalDiagnosticWriter struct{}

func (failedOptionalDiagnosticWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestBestEffortDiagnostic(t *testing.T) {
	for _, raw := range []bool{false, true} {
		var out bytes.Buffer
		w := NewSanitizingWriter(&out)
		if err := w.SetRaw(raw); err != nil {
			t.Fatal(err)
		}
		w.WriteBestEffort([]byte("warning\x1b[31m\n"))
		want := "warning\\x1b[31m\n"
		if raw {
			want = "warning\x1b[31m\n"
		}
		if out.String() != want {
			t.Fatalf("raw=%t output=%q", raw, out.String())
		}
		failed := NewSanitizingWriter(failedOptionalDiagnosticWriter{})
		if err := failed.SetRaw(raw); err != nil {
			t.Fatal(err)
		}
		failed.WriteBestEffort([]byte("warning\n"))
		if err := failed.Flush(); err != nil {
			t.Fatalf("warning left pending failure: %v", err)
		}
	}
}

func TestBestEffortDiagnosticPreservesCallerBufferedBytes(t *testing.T) {
	w := NewSanitizingWriter(failedOptionalDiagnosticWriter{})
	if _, err := w.Write([]byte("required output")); err == nil {
		t.Fatal("required output unexpectedly succeeded")
	}
	w.WriteBestEffort([]byte("optional warning\n"))
	if err := w.Flush(); err == nil {
		t.Fatal("optional warning hid required output failure")
	}
	var out bytes.Buffer
	w.out = &out
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "required output" {
		t.Fatalf("buffered output=%q", out.String())
	}
}
