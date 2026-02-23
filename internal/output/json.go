package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// WriteJSON writes all variables as pretty-printed JSON to the writer.
func WriteJSON(w io.Writer, variables map[string]string) error {
	data, err := json.MarshalIndent(variables, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling variables to JSON: %w", err)
	}
	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("writing JSON output: %w", err)
	}
	_, err = w.Write([]byte("\n"))
	return err
}

// WriteVariable writes a single variable value to the writer.
func WriteVariable(w io.Writer, variables map[string]string, name string) error {
	val, ok := variables[name]
	if !ok {
		return fmt.Errorf("unknown variable %q", name)
	}
	_, err := fmt.Fprintln(w, val)
	return err
}

// WriteAll writes all variables as key=value pairs to the writer, sorted by key.
func WriteAll(w io.Writer, variables map[string]string) error {
	keys := make([]string, 0, len(variables))
	for k := range variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if _, err := fmt.Fprintf(w, "%s=%s\n", k, variables[k]); err != nil {
			return err
		}
	}
	return nil
}
