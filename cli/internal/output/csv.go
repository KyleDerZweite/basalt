// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/kyle/basalt/internal/graph"
)

// WriteCSV writes a flat CSV of all nodes (one row per node).
func WriteCSV(w io.Writer, g *graph.Graph) error {
	nodes, _ := g.Collect()

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header.
	if err := writer.Write([]string{"id", "type", "label", "source_module", "confidence", "wave"}); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for _, n := range nodes {
		row := []string{
			n.ID,
			n.Type,
			n.Label,
			n.SourceModule,
			fmt.Sprintf("%.2f", n.Confidence),
			fmt.Sprintf("%d", n.Wave),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	return nil
}
