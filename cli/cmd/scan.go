// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for relational OSINT data",
	Long: `Perform relational OSINT scanning starting from seed entities.

Examples:
  basalt scan --seed username:john_doe
  basalt scan --seed email:john@example.com
  basalt scan --seed domain:example.com`,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse seeds from flags
		seeds, _ := cmd.Flags().GetStringSlice("seed")

		if len(seeds) == 0 {
			fmt.Println("Error: at least one seed must be provided")
			os.Exit(1)
		}

		// TODO: Implement pivot engine
		fmt.Println("Scanning with seeds:", seeds)

		// For now, just exit
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Flags
	scanCmd.Flags().StringSliceP("seed", "s", []string{}, "Seed entities in format type:value (e.g., username:john, email:john@example.com, domain:example.com)")
	scanCmd.Flags().IntP("max-depth", "d", 2, "Maximum pivot depth")
	scanCmd.Flags().StringP("output", "o", "", "Output file for results (default stdout)")
}
