// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/kyle/basalt/internal/engine"
	"github.com/kyle/basalt/internal/engines/email"
	"github.com/kyle/basalt/internal/engines/email/modules"
	"github.com/kyle/basalt/internal/engines/username"
	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/output"
	pivotpkg "github.com/kyle/basalt/internal/pivot"
	"github.com/kyle/basalt/internal/resolver"
	"github.com/kyle/basalt/internal/sitedb"
)

var (
	flagSiteDirs      []string
	flagRateLimit     float64
	flagMaxPivotDepth int
	flagNoPivot       bool
	flagProxy         string
)

var scanCmd = &cobra.Command{
	Use:   "scan <seed>",
	Short: "Scan a username, email, or phone number across platforms",
	Long: `Scan takes a seed value (username, email, phone number, or domain) and
checks it across hundreds of platforms to discover associated accounts.

The scan produces a relationship graph in JSON format that can be consumed
by the Basalt dashboard or any compatible visualization tool.

When pivoting is enabled (--max-pivot-depth > 0), discovered emails and
usernames from profile pages are automatically fed back as new seeds.

GDPR NOTICE: Only scan identifiers you own or have explicit consent to search.`,
	Args: cobra.ExactArgs(1),
	RunE: runScan,
}

func init() {
	scanCmd.Flags().StringSliceVar(&flagSiteDirs, "site-dirs", nil, "Additional directories to load site definitions from")
	scanCmd.Flags().Float64Var(&flagRateLimit, "rate-limit", 10, "Global requests per second limit")
	scanCmd.Flags().IntVar(&flagMaxPivotDepth, "max-pivot-depth", 0, "Maximum pivot depth (0 = no pivoting)")
	scanCmd.Flags().BoolVar(&flagNoPivot, "no-pivot", false, "Disable auto-pivoting even if extract rules exist")
	scanCmd.Flags().StringVar(&flagProxy, "proxy", "", "Proxy URL (http://host:port or socks5://host:port) or path to proxy list file")
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	initLogging()
	ctx, cancel := setupShutdown()
	defer cancel()

	seed := resolver.Resolve(args[0])
	slog.Info("Resolved seed", "value", seed.Value, "type", seed.Type)

	client, rateLimiter, err := buildHTTPStack()
	if err != nil {
		return err
	}

	controlCache := engine.NewControlCache(client, rateLimiter, 1*time.Hour)
	registry := buildEngineRegistry(client, rateLimiter, controlCache)

	pivotDepth := flagMaxPivotDepth
	if flagNoPivot {
		pivotDepth = 0
	}

	g := initGraph(seed, pivotDepth)
	pivotCtrl := pivotpkg.NewController(pivotDepth)
	pivotCtrl.MarkSeen(seed)

	runScanLoop(ctx, g, registry, seed, pivotCtrl)

	g.Meta.CompletedAt = time.Now().UTC()
	g.Meta.DurationSecs = g.Meta.CompletedAt.Sub(g.Meta.StartedAt).Seconds()

	printSummary(g, controlCache)
	return writeOutput(g)
}

// initLogging configures structured logging based on verbosity flag.
func initLogging() {
	logLevel := slog.LevelWarn
	if flagVerbose {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
}

// setupShutdown sets up graceful shutdown on SIGINT/SIGTERM.
// First signal cancels context (drains in-flight). Second exits immediately.
func setupShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintf(os.Stderr, "\nInterrupted - finishing in-flight requests (press Ctrl+C again to force quit)...\n")
		cancel()
		<-sigCh
		fmt.Fprintf(os.Stderr, "\nForce quit.\n")
		os.Exit(1)
	}()

	return ctx, cancel
}

// buildHTTPStack creates the HTTP client and rate limiter, including proxy setup.
func buildHTTPStack() (*httpclient.Client, *httpclient.DomainRateLimiter, error) {
	clientOpts := []httpclient.Option{
		httpclient.WithTimeout(time.Duration(flagTimeout) * time.Second),
	}

	if flagProxy != "" {
		transport, proxyCount, err := buildProxyTransport(flagProxy)
		if err != nil {
			return nil, nil, fmt.Errorf("setting up proxy: %w", err)
		}
		clientOpts = append(clientOpts, httpclient.WithTransport(transport))
		fmt.Fprintf(os.Stderr, "Using %d proxy(ies)\n", proxyCount)
	}

	client := httpclient.New(clientOpts...)
	rateLimiter := httpclient.NewDomainRateLimiter(flagRateLimit, int(flagRateLimit)*2)
	return client, rateLimiter, nil
}

// buildEngineRegistry creates and populates the engine registry.
func buildEngineRegistry(client *httpclient.Client, rateLimiter *httpclient.DomainRateLimiter, controlCache *engine.ControlCache) *engine.Registry {
	registry := engine.NewRegistry()

	// Load and register username engine.
	sites, err := sitedb.LoadSites(buildSiteDirs()...)
	if err != nil {
		slog.Warn("failed to load site definitions", "error", err)
	}

	if len(sites) > 0 {
		fmt.Fprintf(os.Stderr, "Loaded %d site definitions from YAML\n", len(sites))
	} else {
		fmt.Fprintf(os.Stderr, "No YAML site definitions found, using built-in sites\n")
	}

	usernameOpts := []username.Option{
		username.WithConcurrency(flagConcurrency),
		username.WithRateLimiter(rateLimiter),
		username.WithControlCache(controlCache),
	}
	if len(sites) > 0 {
		usernameOpts = append(usernameOpts, username.WithSites(sites))
	}
	registry.Register(username.New(client, flagThreshold, usernameOpts...))

	// Register email engine.
	registry.Register(email.New(client, flagThreshold,
		email.WithModules(modules.All()),
		email.WithConcurrency(flagConcurrency),
		email.WithRateLimiter(rateLimiter),
	))

	return registry
}

// initGraph creates the output graph with metadata.
func initGraph(seed engine.Seed, pivotDepth int) *graph.Graph {
	g := graph.New()
	g.Meta = graph.Meta{
		Version:   Version,
		ScanID:    uuid.New().String(),
		StartedAt: time.Now().UTC(),
		InitialSeeds: []graph.SeedRef{
			{Value: seed.Value, Type: string(seed.Type)},
		},
		Config: graph.Config{
			ConfidenceThreshold: flagThreshold,
			MaxPivotDepth:       pivotDepth,
			Concurrency:         flagConcurrency,
			TimeoutSeconds:      flagTimeout,
		},
	}
	g.AddNode(graph.NewSeedNode(string(seed.Type), seed.Value, true, "", 0))
	return g
}

// runScanLoop processes seeds across depth levels, running engines and collecting results.
func runScanLoop(ctx context.Context, g *graph.Graph, registry *engine.Registry, seed engine.Seed, pivotCtrl *pivotpkg.Controller) {
	seedQueue := []engine.Seed{seed}

	for depth := 0; len(seedQueue) > 0; depth++ {
		if ctx.Err() != nil {
			break
		}

		if depth > 0 {
			fmt.Fprintf(os.Stderr, "\nPivot depth %d: %d new seeds\n", depth, len(seedQueue))
		} else {
			fmt.Fprintf(os.Stderr, "Scanning %s %q across platforms...\n", seed.Type, seed.Value)
		}

		for _, currentSeed := range seedQueue {
			if ctx.Err() != nil {
				break
			}

			engines := registry.EnginesFor(currentSeed.Type)
			if len(engines) == 0 {
				slog.Debug("no engines for seed type", "type", currentSeed.Type, "value", currentSeed.Value)
				continue
			}

			if depth > 0 {
				fmt.Fprintf(os.Stderr, "  Scanning %s %q...\n", currentSeed.Type, currentSeed.Value)
			}

			runEngines(ctx, g, engines, currentSeed, pivotCtrl)
		}

		if !pivotCtrl.Enabled() {
			break
		}
		seedQueue = pivotCtrl.Drain()
	}
}

// runEngines executes all applicable engines for a seed and collects results into the graph.
func runEngines(ctx context.Context, g *graph.Graph, engines []engine.Engine, seed engine.Seed, pivotCtrl *pivotpkg.Controller) {
	for _, eng := range engines {
		resultsCh := make(chan engine.Result, 200)
		go eng.Check(ctx, seed, resultsCh)

		for result := range resultsCh {
			processResult(g, seed, result, pivotCtrl)
		}
	}
}

// processResult handles a single engine result: updates the graph, logs progress, enqueues pivots.
func processResult(g *graph.Graph, seed engine.Seed, result engine.Result, pivotCtrl *pivotpkg.Controller) {
	g.IncrSitesChecked()

	if result.Err != nil {
		g.IncrErrors()
		slog.Debug("check error", "site", result.SiteName, "error", result.Err)
		return
	}

	seedNodeID := graph.SeedNodeID(string(seed.Type), seed.Value)
	accountNode := graph.NewAccountNode(
		result.SiteName,
		result.ProfileURL,
		result.Category,
		result.Confidence,
		result.Exists,
		result.Signals,
		result.Metadata,
		result.HTTPStatus,
		result.ResponseTime.Milliseconds(),
		seed.Value,
	)

	g.AddNode(accountNode)

	if result.Exists {
		g.IncrAccountsFound()
	}

	g.AddEdge(graph.NewDiscoveredEdge(
		g.NextEdgeID(),
		seedNodeID,
		accountNode.ID,
		result.EngineName,
		result.Confidence,
	))

	// Handle discovered seeds from extraction rules.
	if result.Exists && len(result.DiscoveredSeeds) > 0 {
		for _, ds := range result.DiscoveredSeeds {
			newSeedNode := graph.NewSeedNode(string(ds.Type), ds.Value, false, accountNode.ID, ds.Depth)
			if g.AddNode(newSeedNode) {
				g.AddEdge(graph.NewExtractedSeedEdge(
					g.NextEdgeID(),
					accountNode.ID,
					newSeedNode.ID,
					"extract",
					"regex/selector",
				))
				g.IncrPivotsExecuted()
				fmt.Fprintf(os.Stderr, "  [*] Discovered %s: %s (from %s)\n",
					ds.Type, ds.Value, result.SiteName)
			}
		}
		pivotCtrl.Enqueue(result.DiscoveredSeeds)
	}

	// Log progress.
	if result.Exists {
		meta := formatMetadata(result.Metadata)
		fmt.Fprintf(os.Stderr, "  [+] %-25s  confidence=%.2f  %s%s\n",
			result.SiteName, result.Confidence, result.ProfileURL, meta)
	} else if flagVerbose {
		fmt.Fprintf(os.Stderr, "  [-] %-25s  confidence=%.2f\n",
			result.SiteName, result.Confidence)
	}
}

func formatMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	s := ""
	for k, v := range metadata {
		s += fmt.Sprintf(" %s=%q", k, v)
	}
	return s
}

func printSummary(g *graph.Graph, controlCache *engine.ControlCache) {
	fmt.Fprintf(os.Stderr, "\nScan complete: %d sites checked, %d accounts found, %d pivots, %d errors (%.1fs)\n",
		g.Meta.Stats.SitesChecked,
		g.Meta.Stats.AccountsFound,
		g.Meta.Stats.PivotsExecuted,
		g.Meta.Stats.Errors,
		g.Meta.DurationSecs,
	)
	fmt.Fprintf(os.Stderr, "Control cache: %d entries cached (saved %d duplicate HTTP requests)\n",
		controlCache.Len(), max(0, int(g.Meta.Stats.SitesChecked)-controlCache.Len()),
	)
}

func writeOutput(g *graph.Graph) error {
	switch flagOutput {
	case "table":
		return output.WriteTable(os.Stdout, g)
	default:
		return output.WriteJSON(os.Stdout, g)
	}
}

// buildProxyTransport creates an HTTP transport from a proxy URL or proxy list file.
func buildProxyTransport(proxyArg string) (transport *http.Transport, count int, err error) {
	var proxyURLs []string

	if _, statErr := os.Stat(proxyArg); statErr == nil {
		proxyURLs, err = httpclient.LoadProxyFile(proxyArg)
		if err != nil {
			return nil, 0, fmt.Errorf("loading proxy file: %w", err)
		}
	} else {
		proxyURLs = []string{proxyArg}
	}

	pool, err := httpclient.NewProxyPool(proxyURLs)
	if err != nil {
		return nil, 0, err
	}

	t, ok := pool.Transport().(*http.Transport)
	if !ok {
		return nil, 0, fmt.Errorf("unexpected transport type")
	}
	return t, pool.Len(), nil
}

// buildSiteDirs returns the list of directories to search for YAML site definitions.
func buildSiteDirs() []string {
	var dirs []string

	exe, err := os.Executable()
	if err == nil {
		dirs = append(dirs, filepath.Join(filepath.Dir(exe), "data", "sites"))
	}

	dirs = append(dirs, "data/sites")

	home, err := os.UserHomeDir()
	if err == nil {
		dirs = append(dirs, filepath.Join(home, ".basalt", "sites"))
	}

	dirs = append(dirs, flagSiteDirs...)
	return dirs
}
