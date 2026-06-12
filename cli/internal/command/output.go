package command

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	outputJSON  = "json"
	outputTable = "table"
)

func writeData(w io.Writer, format string, value any) error {
	switch strings.ToLower(format) {
	case outputJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	case "", outputTable:
		return writeHuman(w, value)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func writeHuman(w io.Writer, value any) error {
	switch typed := value.(type) {
	case string:
		_, err := fmt.Fprintln(w, typed)
		return err
	case map[string]string:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if _, err := fmt.Fprintf(w, "%-24s %s\n", key, typed[key]); err != nil {
				return err
			}
		}
		return nil
	default:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
}
