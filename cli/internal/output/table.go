// SPDX-License-Identifier: AGPL-3.0-or-later
//
// Color-coded terminal table output for scan results.

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
	red    = color.New(color.FgRed)
	yellow = color.New(color.FgYellow)
	cyan   = color.New(color.FgCyan)
	bold   = color.New(color.Bold)
	dim    = color.New(color.Faint)
)

// WriteTable writes a color-coded table to the given writer.
func WriteTable(w io.Writer, g *graph.Graph) error {
	nodes, meta := g.AccountNodes()

	if len(nodes) == 0 {
		fmt.Fprintln(w, "No accounts found.")
		return nil
	}

	// Sort: found accounts first (by confidence desc), then not found.
	sort.Slice(nodes, func(i, j int) bool {
		ei := nodes[i].Properties["exists"]
		ej := nodes[j].Properties["exists"]
		existsI, _ := ei.(bool)
		existsJ, _ := ej.(bool)
		if existsI != existsJ {
			return existsI
		}
		ci, _ := nodes[i].Properties["confidence"].(float64)
		cj, _ := nodes[j].Properties["confidence"].(float64)
		return ci > cj
	})

	// Print header.
	bold.Fprintf(w, "\n%-4s  %-25s  %-10s  %-12s  %s\n",
		"", "SITE", "CONFIDENCE", "CATEGORY", "URL")
	fmt.Fprintln(w, strings.Repeat("─", 100))

	// Print rows.
	found := 0
	for _, node := range nodes {
		exists, _ := node.Properties["exists"].(bool)
		conf, _ := node.Properties["confidence"].(float64)
		siteName, _ := node.Properties["site_name"].(string)
		category, _ := node.Properties["category"].(string)
		profileURL, _ := node.Properties["profile_url"].(string)

		if !exists {
			continue
		}
		found++

		// Color-code confidence.
		var confColor *color.Color
		switch {
		case conf >= 0.80:
			confColor = green
		case conf >= 0.60:
			confColor = yellow
		default:
			confColor = red
		}

		indicator := green.Sprint(" [+]")
		fmt.Fprintf(w, "%s  %-25s  ", indicator, siteName)
		confColor.Fprintf(w, "%-10.2f", conf)
		fmt.Fprintf(w, "  ")
		cyan.Fprintf(w, "%-12s", category)
		fmt.Fprintf(w, "  %s", profileURL)

		// Print extracted metadata inline.
		switch md := node.Properties["metadata"].(type) {
		case map[string]string:
			for k, v := range md {
				dim.Fprintf(w, "  %s=%s", k, v)
			}
		case map[string]interface{}:
			for k, v := range md {
				dim.Fprintf(w, "  %s=%v", k, v)
			}
		}
		fmt.Fprintln(w)
	}

	// Summary line.
	fmt.Fprintln(w, strings.Repeat("─", 100))
	bold.Fprintf(w, "Found %d accounts", found)
	fmt.Fprintf(w, " across %d sites checked", meta.Stats.SitesChecked)
	if meta.Stats.PivotsExecuted > 0 {
		fmt.Fprintf(w, ", %d pivots", meta.Stats.PivotsExecuted)
	}
	if meta.Stats.Errors > 0 {
		yellow.Fprintf(w, ", %d errors", meta.Stats.Errors)
	}
	fmt.Fprintf(w, " (%.1fs)\n\n", meta.DurationSecs)

	return nil
}
