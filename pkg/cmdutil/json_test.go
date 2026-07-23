package cmdutil

import (
	"bytes"
	"errors"
	"testing"
)

type failingJSONWriter struct{ err error }

func (writer failingJSONWriter) Write([]byte) (int, error) { return 0, writer.err }

func TestWriteJSON(t *testing.T) {
	t.Run("encodes indented JSON", func(t *testing.T) {
		var output bytes.Buffer
		if err := WriteJSON(&output, map[string]string{"name": "demo"}); err != nil {
			t.Fatal(err)
		}
		if got, want := output.String(), "{\n  \"name\": \"demo\"\n}\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})

	t.Run("returns writer error", func(t *testing.T) {
		wantErr := errors.New("write failed")
		if err := WriteJSON(failingJSONWriter{err: wantErr}, struct{}{}); !errors.Is(err, wantErr) {
			t.Fatalf("error = %v", err)
		}
	})
}
