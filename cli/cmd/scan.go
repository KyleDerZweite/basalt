// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/kyle/basalt/internal/config"
	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
	"github.com/kyle/basalt/internal/modules/beacons"
	"github.com/kyle/basalt/internal/modules/bento"
	"github.com/kyle/basalt/internal/modules/carrd"
	"github.com/kyle/basalt/internal/modules/discord"
	"github.com/kyle/basalt/internal/modules/dnsct"
	"github.com/kyle/basalt/internal/modules/github"
	"github.com/kyle/basalt/internal/modules/gitlab"
	"github.com/kyle/basalt/internal/modules/gravatar"
	"github.com/kyle/basalt/internal/modules/instagram"
	"github.com/kyle/basalt/internal/modules/linktree"
	"github.com/kyle/basalt/internal/modules/matrix"
	"github.com/kyle/basalt/internal/modules/reddit"
	"github.com/kyle/basalt/internal/modules/stackexchange"
	"github.com/kyle/basalt/internal/modules/steam"
	"github.com/kyle/basalt/internal/modules/tiktok"
	"github.com/kyle/basalt/internal/modules/twitch"
	"github.com/kyle/basalt/internal/modules/whois"
	"github.com/kyle/basalt/internal/modules/youtube"
	"github.com/kyle/basalt/internal/output"
	"github.com/kyle/basalt/internal/walker"
)

var (
	flagUsernames   []string
	flagEmails      []string
	flagDomains     []string
	flagDepth       int
	flagConcurrency int
	flagTimeout     int
	flagConfigPath  string
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
	scanCmd.Flags().StringSliceVar(&flagExport, "export", nil, "Export format: json, csv")
	scanCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Show module health and debug info")
}

func runScan(cmd *cobra.Command, args []string) error {
	// Collect seeds.
	var seeds []graph.Seed
	for _, u := range flagUsernames {
		seeds = append(seeds, graph.Seed{Type: "username", Value: u})
	}
	for _, e := range flagEmails {
		seeds = append(seeds, graph.Seed{Type: "email", Value: e})
	}
	for _, d := range flagDomains {
		seeds = append(seeds, graph.Seed{Type: "domain", Value: d})
	}

	if len(seeds) == 0 {
		return fmt.Errorf("at least one seed required (-u, -e, or -d)")
	}

	// Load config.
	configPath := flagConfigPath
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".basalt", "config")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Build HTTP client.
	client := httpclient.New(
		httpclient.WithTimeout(time.Duration(flagTimeout) * time.Second),
	)

	// Register all modules.
	reg := modules.NewRegistry()
	reg.Register(gravatar.New())
	reg.Register(linktree.New())
	reg.Register(beacons.New())
	reg.Register(carrd.New())
	reg.Register(bento.New())
	reg.Register(github.New(cfg.Get("GITHUB_TOKEN")))
	reg.Register(gitlab.New())
	reg.Register(stackexchange.New())
	reg.Register(reddit.New())
	reg.Register(youtube.New())
	reg.Register(twitch.New())
	reg.Register(discord.New())
	reg.Register(instagram.New())
	reg.Register(tiktok.New())
	reg.Register(matrix.New())
	reg.Register(steam.New(cfg.Get("STEAM_API_KEY")))
	reg.Register(whois.New())
	reg.Register(dnsct.New())

	// Build graph.
	g := graph.New()
	g.Meta.Version = Version
	g.Meta.ScanID = uuid.New().String()
	g.Meta.StartedAt = time.Now()
	g.Meta.Config = graph.Config{
		MaxPivotDepth: flagDepth,
		Concurrency:   flagConcurrency,
		TimeoutSecs:   flagTimeout,
	}
	for _, s := range seeds {
		g.Meta.InitialSeeds = append(g.Meta.InitialSeeds, graph.SeedRef{Value: s.Value, Type: s.Type})
	}

	// Build walker.
	w := walker.New(g, reg,
		walker.WithMaxDepth(flagDepth),
		walker.WithConcurrency(flagConcurrency),
		walker.WithTimeout(time.Duration(flagTimeout)*time.Second),
		walker.WithClient(client),
	)

	// Context with graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted, finishing in-flight requests...")
		cancel()
	}()

	// Verify modules and print health summary before scanning.
	w.VerifyAll(ctx)
	printHealthSummary(w.HealthSummary(), flagVerbose)

	// Run scan.
	w.Run(ctx, seeds)

	// Finalize metadata.
	g.Meta.CompletedAt = time.Now()
	g.Meta.DurationSecs = g.Meta.CompletedAt.Sub(g.Meta.StartedAt).Seconds()

	// Default: table to stdout.
	if err := output.WriteTable(os.Stdout, g); err != nil {
		return fmt.Errorf("writing table: %w", err)
	}

	// Exports.
	timestamp := time.Now().Format("20060102-150405")
	for _, format := range flagExport {
		switch format {
		case "json":
			path := fmt.Sprintf("basalt-scan-%s.json", timestamp)
			f, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("creating %s: %w", path, err)
			}
			if err := output.WriteJSON(f, g); err != nil {
				f.Close()
				return fmt.Errorf("writing JSON: %w", err)
			}
			f.Close()
			fmt.Fprintf(os.Stderr, "Exported JSON to %s\n", path)

		case "csv":
			path := fmt.Sprintf("basalt-scan-%s.csv", timestamp)
			f, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("creating %s: %w", path, err)
			}
			if err := output.WriteCSV(f, g); err != nil {
				f.Close()
				return fmt.Errorf("writing CSV: %w", err)
			}
			f.Close()
			fmt.Fprintf(os.Stderr, "Exported CSV to %s\n", path)

		default:
			fmt.Fprintf(os.Stderr, "Unknown export format: %s (use: json, csv)\n", format)
		}
	}

	return nil
}

func printHealthSummary(health []walker.ModuleHealth, verbose bool) {
	var ready, degraded, offline int
	for _, h := range health {
		switch h.Status {
		case modules.Healthy:
			ready++
		case modules.Degraded:
			degraded++
			if verbose {
				fmt.Fprintf(os.Stderr, "  [degraded] %s\n", h.Message)
			}
		case modules.Offline:
			offline++
			if verbose {
				fmt.Fprintf(os.Stderr, "  [offline]  %s\n", h.Message)
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
