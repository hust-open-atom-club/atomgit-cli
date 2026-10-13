package cmdutil

import (
	"bytes"
	"encoding/json"
	"io"
)

// WriteJSON writes one indented JSON value followed by a newline.
func WriteJSON(writer io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	// encoding/json leaves DEL unescaped. Escape it before the terminal writer
	// can turn it into \\x7f, which is not a valid JSON escape sequence.
	data = bytes.ReplaceAll(data, []byte{0x7f}, []byte(`\u007f`))
	_, err = writer.Write(append(data, '\n'))
	return err
}
