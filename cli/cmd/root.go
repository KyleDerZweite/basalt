// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagOutput      string
	flagConcurrency int
	flagTimeout     int
	flagVerbose     bool
	flagThreshold   float64
)

var rootCmd = &cobra.Command{
	Use:   "basalt",
	Short: "Basalt - Unified OSINT digital footprint discovery",
	Long: `Basalt is an open-source intelligence tool for discovering your digital footprint.
It checks a username, email, or phone number across thousands of platforms and
builds a relationship graph of all discovered accounts.

IMPORTANT: This tool is designed for authorized security research and self-lookup
only. You must have explicit consent (your own data or the data subject's consent)
before running any scan. Unauthorized use may violate GDPR and local laws.

Licensed under AGPLv3.`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "json", "Output format: json or table")
	rootCmd.PersistentFlags().IntVarP(&flagConcurrency, "concurrency", "c", 20, "Maximum concurrent requests")
	rootCmd.PersistentFlags().IntVarP(&flagTimeout, "timeout", "t", 15, "HTTP request timeout in seconds")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose/debug logging")
	rootCmd.PersistentFlags().Float64Var(&flagThreshold, "threshold", 0.50, "Minimum confidence score to consider a match [0.0-1.0]")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
