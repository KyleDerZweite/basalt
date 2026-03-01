// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kyle/basalt/internal/sitedb"
)

var flagImportOutput string

var importCmd = &cobra.Command{
	Use:   "import <format> <path>",
	Short: "Import upstream site databases (maigret, sherlock, wmn)",
	Long: `Import converts upstream OSINT tool databases into Basalt's YAML format.

Supported formats:
  maigret   - Maigret's data.json (3000+ sites)
  sherlock  - Sherlock's data.json (400+ sites)
  wmn       - WhatsMyName's wmn-data.json

Examples:
  basalt import maigret ~/maigret/resources/data.json
  basalt import sherlock ~/sherlock/sherlock_project/resources/data.json
  basalt import wmn ~/WhatsMyName/wmn-data.json`,
	Args: cobra.ExactArgs(2),
	RunE: runImport,
}

func init() {
	importCmd.Flags().StringVarP(&flagImportOutput, "out", "O", "", "Output directory (default: ./data/sites/)")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	formatStr := args[0]
	inputPath := args[1]

	var format sitedb.UpstreamFormat
	switch formatStr {
	case "maigret":
		format = sitedb.FormatMaigret
	case "sherlock":
		format = sitedb.FormatSherlock
	case "wmn":
		format = sitedb.FormatWMN
	default:
		return fmt.Errorf("unknown format %q (use: maigret, sherlock, wmn)", formatStr)
	}

	// Determine output directory.
	outDir := flagImportOutput
	if outDir == "" {
		// Find the data/sites directory relative to the binary or cwd.
		outDir = "data/sites"
	}

	fmt.Fprintf(os.Stderr, "Importing %s database from %s...\n", formatStr, inputPath)

	// Import and convert.
	sites, err := sitedb.ImportUpstream(inputPath, format)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Converted %d site definitions\n", len(sites))

	// Validate.
	validationErrors := sitedb.ValidateAll(sites)
	if len(validationErrors) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: %d sites have validation issues (skipped):\n", len(validationErrors))
		i := 0
		for name, errs := range validationErrors {
			if i >= 5 {
				fmt.Fprintf(os.Stderr, "  ... and %d more\n", len(validationErrors)-5)
				break
			}
			fmt.Fprintf(os.Stderr, "  %s: %v\n", name, errs)
			i++
		}
	}

	// Filter out invalid sites.
	var valid []sitedb.SiteDefinition
	for _, s := range sites {
		errs := sitedb.Validate(&s)
		if len(errs) == 0 {
			valid = append(valid, s)
		}
	}

	// Write all valid sites to a single file.
	outPath := filepath.Join(outDir, "sites.yaml")

	// If the file already exists, load existing sites and merge (dedup by URL template).
	existing, _ := sitedb.LoadSites(outDir)
	if len(existing) > 0 {
		seen := make(map[string]struct{}, len(existing))
		for _, s := range existing {
			seen[strings.ToLower(s.URLTemplate)] = struct{}{}
		}
		for _, s := range valid {
			if _, dup := seen[strings.ToLower(s.URLTemplate)]; !dup {
				existing = append(existing, s)
				seen[strings.ToLower(s.URLTemplate)] = struct{}{}
			}
		}
		valid = existing
	}

	fmt.Fprintf(os.Stderr, "Writing %d sites to %s\n", len(valid), outPath)

	if err := sitedb.WriteSites(outPath, valid); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "Import complete.\n")
	return nil
}
