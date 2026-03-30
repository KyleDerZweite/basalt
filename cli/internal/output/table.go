// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"

	"github.com/kyle/basalt/internal/graph"
)

var (
	green  = color.New(color.FgGreen, color.Bold)
	yellow = color.New(color.FgYellow)
	dim    = color.New(color.Faint)
	bold   = color.New(color.Bold)
)

// WriteTable writes a color-coded table of all non-seed nodes to the writer.
func WriteTable(w io.Writer, g *graph.Graph) error {
	nodes, _ := g.Collect()

	// Filter out seed nodes.
	var display []*graph.Node
	for _, n := range nodes {
		if n.Type != graph.NodeTypeSeed {
			display = append(display, n)
		}
	}

	if len(display) == 0 {
		fmt.Fprintln(w, "No results found.")
		return nil
	}

	// Sort by confidence descending.
	sort.Slice(display, func(i, j int) bool {
		return display[i].Confidence > display[j].Confidence
	})

	// Header.
	fmt.Fprintln(w)
	bold.Fprintf(w, " %-20s  %-12s  %-40s  %-10s  %s\n",
		"PLATFORM", "TYPE", "VALUE", "CONFIDENCE", "SOURCE")
	fmt.Fprintln(w, " "+strings.Repeat("-", 100))

	// Rows.
	for _, n := range display {
		platform := n.SourceModule
		if siteName, ok := n.Properties["site_name"].(string); ok {
			platform = siteName
		}

		value := n.Label
		if profileURL, ok := n.Properties["profile_url"].(string); ok && profileURL != "" {
			value = profileURL
		}

		// Truncate long values.
		if len(value) > 40 {
			value = value[:37] + "..."
		}

		// Color-code confidence.
		confStr := fmt.Sprintf("%.2f", n.Confidence)
		var confColor *color.Color
		switch {
		case n.Confidence >= 0.80:
			confColor = green
		case n.Confidence >= 0.50:
			confColor = yellow
		default:
			confColor = dim
		}

		fmt.Fprintf(w, " %-20s  %-12s  %-40s  ", platform, n.Type, value)
		confColor.Fprintf(w, "%-10s", confStr)
		fmt.Fprintf(w, "  %s\n", n.SourceModule)
	}

	// Summary.
	fmt.Fprintln(w, " "+strings.Repeat("-", 100))
	g.SnapshotStats()
	bold.Fprintf(w, " %d results found", len(display))
	fmt.Fprintf(w, " (%d modules, %.1fs)\n\n", g.Meta.Stats.ModulesRun, g.Meta.DurationSecs)

	return nil
}
