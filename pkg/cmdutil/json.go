package cmdutil

import (
	"encoding/json"
	"io"
)

// WriteJSON writes one indented JSON value followed by a newline.
func WriteJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
