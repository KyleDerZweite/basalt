// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/KyleDerZweite/basalt/internal/app"
	"github.com/KyleDerZweite/basalt/internal/graph"
)

var (
	flagTargetName   string
	flagTargetNotes  string
	flagTargetSlug   string
	flagAliasLabel   string
	flagAliasPrimary bool
)

var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "Manage persistent targets and aliases",
}

var targetCreateCmd = &cobra.Command{
	Use:   "create <slug>",
	Short: "Create a new target",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := app.NewService(Version, flagDataDir)
		if err != nil {
			return err
		}
		defer service.Close()

		target, err := service.CreateTarget(app.Target{
			Slug:        firstNonEmpty(flagTargetSlug, args[0]),
			DisplayName: flagTargetName,
			Notes:       flagTargetNotes,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Created target %s (%s)\n", target.DisplayName, target.Slug)
		return nil
	},
}

var targetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all targets",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := app.NewService(Version, flagDataDir)
		if err != nil {
			return err
		}
		defer service.Close()

		targets, err := service.ListTargets()
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			fmt.Println("No targets found.")
			return nil
		}
		for _, target := range targets {
			fmt.Printf("%s\t%s\t%d aliases\n", target.Slug, target.DisplayName, len(target.Aliases))
		}
		return nil
	},
}

var targetShowCmd = &cobra.Command{
	Use:   "show <slug-or-id>",
	Short: "Show a target and its aliases",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := app.NewService(Version, flagDataDir)
		if err != nil {
			return err
		}
		defer service.Close()

		target, err := service.GetTarget(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("%s (%s)\n", target.DisplayName, target.Slug)
		if target.Notes != "" {
			fmt.Printf("Notes: %s\n", target.Notes)
		}
		if len(target.Aliases) == 0 {
			fmt.Println("Aliases: none")
			return nil
		}
		fmt.Println("Aliases:")
		for _, alias := range target.Aliases {
			primary := ""
			if alias.IsPrimary {
				primary = " [primary]"
			}
			label := alias.SeedValue
			if alias.Label != "" {
				label = alias.Label
			}
			fmt.Printf("  %s:%s\t%s%s\t%s\n", alias.SeedType, alias.SeedValue, alias.ID, primary, label)
		}
		return nil
	},
}

var targetAliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Manage target aliases",
}

var targetAliasAddCmd = &cobra.Command{
	Use:   "add <slug-or-id> <type:value>",
	Short: "Add an alias to a target",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		seed, err := graph.ParseSeed(args[1])
		if err != nil {
			return err
		}
		service, err := app.NewService(Version, flagDataDir)
		if err != nil {
			return err
		}
		defer service.Close()

		alias, err := service.AddTargetAlias(args[0], seed, flagAliasLabel, flagAliasPrimary)
		if err != nil {
			return err
		}
		fmt.Printf("Added alias %s:%s to %s\n", alias.SeedType, alias.SeedValue, args[0])
		return nil
	},
}

var targetAliasRemoveCmd = &cobra.Command{
	Use:   "remove <slug-or-id> <alias-id>",
	Short: "Remove an alias from a target",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := app.NewService(Version, flagDataDir)
		if err != nil {
			return err
		}
		defer service.Close()

		if err := service.RemoveTargetAlias(args[0], strings.TrimSpace(args[1])); err != nil {
			return err
		}
		fmt.Printf("Removed alias %s from %s\n", args[1], args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(targetCmd)

	targetCmd.PersistentFlags().StringVar(&flagDataDir, "data-dir", app.DefaultDataDir(), "Path to local app data directory")
	targetCmd.AddCommand(targetCreateCmd)
	targetCmd.AddCommand(targetListCmd)
	targetCmd.AddCommand(targetShowCmd)
	targetCmd.AddCommand(targetAliasCmd)

	targetCreateCmd.Flags().StringVar(&flagTargetName, "name", "", "Display name for the target")
	targetCreateCmd.Flags().StringVar(&flagTargetNotes, "notes", "", "Notes for the target")
	targetCreateCmd.Flags().StringVar(&flagTargetSlug, "slug", "", "Override the target slug")
	targetCreateCmd.MarkFlagRequired("name")

	targetAliasCmd.AddCommand(targetAliasAddCmd)
	targetAliasCmd.AddCommand(targetAliasRemoveCmd)
	targetAliasAddCmd.Flags().StringVar(&flagAliasLabel, "label", "", "Optional display label for the alias")
	targetAliasAddCmd.Flags().BoolVar(&flagAliasPrimary, "primary", false, "Mark the alias as the primary identifier")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
