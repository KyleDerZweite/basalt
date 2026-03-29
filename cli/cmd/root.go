// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "basalt",
	Short: "Basalt - Relation-based OSINT digital footprint discovery",
	Long: `Basalt is an open-source intelligence tool for discovering your digital footprint.
It queries high-value platforms, extracts metadata, and builds a relationship graph
of connected accounts, emails, domains, and identities.

Designed for self-lookup and authorized research only. You must have explicit consent
before running any scan. Unauthorized use may violate GDPR and local laws.

Licensed under AGPLv3.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
