package app

import (
	"encoding/json"
	"io"
)

// output writes data to the given writer as JSON.
func output(writer io.Writer, data any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(data)
}
