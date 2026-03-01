// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/kyle/basalt/internal/graph"
)

// WriteJSON writes the graph as indented JSON to the given writer.
func WriteJSON(w io.Writer, g *graph.Graph) error {
	data, err := g.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshaling graph: %w", err)
	}

	// Re-marshal with indentation for readable output.
	var raw json.RawMessage = data
	indented, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("indenting JSON: %w", err)
	}

	_, err = w.Write(indented)
	if err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	_, err = w.Write([]byte("\n"))
	return err
}
