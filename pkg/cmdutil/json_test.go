package cmdutil

import (
	"bytes"
	"encoding/json"
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

func TestWriteJSONThroughSanitizingWriter(t *testing.T) {
	want := map[string]string{"name\x7f": "line\n\t\"\\\x00\x1b\x7f\u0085\u202e"}
	var output bytes.Buffer
	if err := WriteJSON(NewSanitizingWriter(&output), want); err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v; %q", err, output.String())
	}
	if len(got) != len(want) || got["name\x7f"] != want["name\x7f"] {
		t.Fatalf("decoded JSON = %#v, want %#v", got, want)
	}
}

func TestWriteJSONEncodingCompatibility(t *testing.T) {
	cases := []struct {
		name  string
		value any
	}{
		{name: "null", value: nil},
		{name: "empty array", value: []string{}},
		{name: "nested fields", value: map[string]any{"author": "中文", "assets": []any{map[string]any{"id": int64(9223372036854775807), "draft": false}}}},
		{name: "escaped strings", value: "<>&\n\r\t\\\"\u2028\u2029"},
		{name: "raw JSON", value: json.RawMessage(`{"id":9223372036854775807,"body":"<notes>\n\"quoted\""}`)},
		{name: "unencodable value", value: make(chan int)},
		{name: "invalid raw JSON", value: json.RawMessage(`{`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got, want bytes.Buffer
			encoder := json.NewEncoder(&want)
			encoder.SetIndent("", "  ")
			wantErr := encoder.Encode(tc.value)
			gotErr := WriteJSON(&got, tc.value)
			if (gotErr == nil) != (wantErr == nil) || got.String() != want.String() {
				t.Fatalf("output = %q, error = %v; want %q, error = %v", got.String(), gotErr, want.String(), wantErr)
			}
		})
	}
}
