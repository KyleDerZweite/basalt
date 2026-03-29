// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "2.0.0-dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of Basalt",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("basalt v%s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
