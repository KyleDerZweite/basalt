// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/KyleDerZweite/basalt/internal/app"
	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/output"
	"github.com/spf13/cobra"
)

var (
	flagUsernames   []string
	flagEmails      []string
	flagDomains     []string
	flagDepth       int
	flagConcurrency int
	flagTimeout     int
	flagConfigPath  string
	flagDataDir     string
	flagExport      []string
	flagVerbose     bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for relational OSINT data",
	Long: `Perform relational OSINT scanning starting from seed entities.

Examples:
  basalt scan -u kylederzweite
  basalt scan -e kyle@example.com
  basalt scan -d kylehub.dev
  basalt scan -u kyle -e kyle@example.com`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringSliceVarP(&flagUsernames, "username", "u", nil, "Username seeds")
	scanCmd.Flags().StringSliceVarP(&flagEmails, "email", "e", nil, "Email seeds")
	scanCmd.Flags().StringSliceVarP(&flagDomains, "domain", "d", nil, "Domain seeds")
	scanCmd.Flags().IntVar(&flagDepth, "depth", 2, "Maximum pivot depth")
	scanCmd.Flags().IntVar(&flagConcurrency, "concurrency", 5, "Maximum concurrent requests")
	scanCmd.Flags().IntVar(&flagTimeout, "timeout", 10, "Per-module timeout in seconds")
	scanCmd.Flags().StringVar(&flagConfigPath, "config", "", "Path to config file for API keys")
	scanCmd.Flags().StringVar(&flagDataDir, "data-dir", app.DefaultDataDir(), "Path to local app data directory")
	scanCmd.Flags().StringSliceVar(&flagExport, "export", nil, "Export format: json, csv")
	scanCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Show module health and debug info")
}

func runScan(cmd *cobra.Command, args []string) error {
	request := app.ScanRequest{
		Seeds:          collectSeeds(),
		Depth:          flagDepth,
		Concurrency:    flagConcurrency,
		TimeoutSeconds: flagTimeout,
		ConfigPath:     flagConfigPath,
	}
	if len(request.Seeds) == 0 {
		return fmt.Errorf("at least one seed required (-u, -e, or -d)")
	}

	service, err := app.NewService(Version, flagDataDir)
	if err != nil {
		return err
	}
	defer service.Close()

	health, err := service.ModuleHealth(context.Background(), request)
	if err != nil {
		return fmt.Errorf("verifying modules: %w", err)
	}
	printHealthSummary(health, flagVerbose)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted, finishing in-flight requests...")
		cancel()
	}()

	record, err := service.RunScan(ctx, request)
	if err != nil {
		return err
	}
	if record.Graph == nil {
		return fmt.Errorf("scan completed without a graph")
	}

	if err := output.WriteTable(os.Stdout, record.Graph); err != nil {
		return fmt.Errorf("writing table: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	for _, format := range flagExport {
		path := filepath.Clean(fmt.Sprintf("basalt-scan-%s.%s", timestamp, format))
		file, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("creating %s: %w", path, err)
		}
		if err := service.WriteExport(record.ID, format, file); err != nil {
			file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("closing %s: %w", path, err)
		}
		fmt.Fprintf(os.Stderr, "Exported %s to %s\n", format, path)
	}

	if record.Status == app.ScanStatusPartial || record.Status == app.ScanStatusCanceled {
		fmt.Fprintf(os.Stderr, "Scan finished with status: %s\n", record.Status)
	}
	return nil
}

func collectSeeds() []graph.Seed {
	var seeds []graph.Seed
	for _, username := range flagUsernames {
		seeds = append(seeds, graph.Seed{Type: graph.NodeTypeUsername, Value: username})
	}
	for _, email := range flagEmails {
		seeds = append(seeds, graph.Seed{Type: graph.NodeTypeEmail, Value: email})
	}
	for _, domain := range flagDomains {
		seeds = append(seeds, graph.Seed{Type: graph.NodeTypeDomain, Value: domain})
	}
	return seeds
}

func printHealthSummary(health []app.ModuleStatus, verbose bool) {
	var ready, degraded, offline int
	for _, item := range health {
		switch item.Status {
		case "healthy":
			ready++
		case "degraded":
			degraded++
			if verbose {
				fmt.Fprintf(os.Stderr, "  [degraded] %s: %s\n", item.Name, item.Message)
			}
		case "offline":
			offline++
			if verbose {
				fmt.Fprintf(os.Stderr, "  [offline]  %s: %s\n", item.Name, item.Message)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Modules: %d ready", ready)
	if degraded > 0 {
		fmt.Fprintf(os.Stderr, ", %d degraded", degraded)
	}
	if offline > 0 {
		fmt.Fprintf(os.Stderr, ", %d offline", offline)
	}
	fmt.Fprintln(os.Stderr)
}
