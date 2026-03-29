// SPDX-License-Identifier: AGPL-3.0-or-later

package walker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

// ModuleHealth tracks the verified state of a module.
type ModuleHealth struct {
	Module  modules.Module
	Status  modules.HealthStatus
	Message string
}

// Walker is the async orchestrator that dispatches nodes to modules.
type Walker struct {
	graph       *graph.Graph
	registry    *modules.Registry
	client      *httpclient.Client
	maxDepth    int
	concurrency int
	timeout     time.Duration

	healthy   []ModuleHealth
	semaphore chan struct{}
	inflight  sync.WaitGroup
	processed sync.Map // "moduleName:nodeID" -> struct{}
}

// Option configures the Walker.
type Option func(*Walker)

// WithMaxDepth sets the maximum pivot depth.
func WithMaxDepth(d int) Option { return func(w *Walker) { w.maxDepth = d } }

// WithConcurrency sets the maximum number of concurrent module calls.
func WithConcurrency(n int) Option { return func(w *Walker) { w.concurrency = n } }

// WithTimeout sets the per-module timeout.
func WithTimeout(d time.Duration) Option { return func(w *Walker) { w.timeout = d } }

// WithClient sets the HTTP client used by modules.
func WithClient(c *httpclient.Client) Option { return func(w *Walker) { w.client = c } }

// New creates a Walker.
func New(g *graph.Graph, reg *modules.Registry, opts ...Option) *Walker {
	w := &Walker{
		graph:       g,
		registry:    reg,
		maxDepth:    2,
		concurrency: 5,
		timeout:     10 * time.Second,
	}
	for _, opt := range opts {
		opt(w)
	}
	w.semaphore = make(chan struct{}, w.concurrency)
	if w.client == nil {
		w.client = httpclient.New()
	}
	return w
}

// HealthSummary returns the verified health of all modules.
func (w *Walker) HealthSummary() []ModuleHealth {
	return w.healthy
}

// VerifyAll runs Verify on all registered modules. Call before Run
// to get the health summary for display.
func (w *Walker) VerifyAll(ctx context.Context) {
	w.verifyModules(ctx)
}

// Run seeds the graph and dispatches to verified modules.
// If VerifyAll was not called, it verifies modules first.
func (w *Walker) Run(ctx context.Context, seeds []graph.Seed) {
	if len(w.healthy) == 0 {
		w.verifyModules(ctx)
	}

	// Add seed nodes.
	var seedNodes []*graph.Node
	for _, s := range seeds {
		node := graph.NewSeedNode(s.Type, s.Value)
		if w.graph.AddNode(node) {
			seedNodes = append(seedNodes, node)
		}
	}

	// Dispatch seed nodes.
	for _, node := range seedNodes {
		w.dispatch(ctx, node)
	}

	// Wait for all in-flight work to complete.
	w.inflight.Wait()
}

// verifyModules runs Verify on all registered modules concurrently.
func (w *Walker) verifyModules(ctx context.Context) {
	all := w.registry.All()
	results := make([]ModuleHealth, len(all))
	var wg sync.WaitGroup

	for i, m := range all {
		wg.Add(1)
		go func(idx int, mod modules.Module) {
			defer wg.Done()
			status, msg := mod.Verify(ctx, w.client)
			results[idx] = ModuleHealth{Module: mod, Status: status, Message: msg}
		}(i, m)
	}
	wg.Wait()

	w.healthy = results
}

// lookupType returns the type string used for module matching.
// For seed nodes, this is the "seed_type" property; for all others,
// it is node.Type directly.
func lookupType(node *graph.Node) string {
	if node.Type == graph.NodeTypeSeed {
		if st, ok := node.Properties["seed_type"].(string); ok {
			return st
		}
	}
	return node.Type
}

// dispatch sends a node to all matching healthy modules.
func (w *Walker) dispatch(ctx context.Context, node *graph.Node) {
	lt := lookupType(node)

	for _, mh := range w.healthy {
		if mh.Status == modules.Offline {
			continue
		}
		if !mh.Module.CanHandle(lt) {
			continue
		}

		// Dedup: skip if this module+node combo was already dispatched.
		key := mh.Module.Name() + ":" + node.ID
		if _, loaded := w.processed.LoadOrStore(key, struct{}{}); loaded {
			continue
		}

		isDegraded := mh.Status == modules.Degraded
		mod := mh.Module

		w.inflight.Add(1)
		go func() {
			defer w.inflight.Done()

			// Acquire semaphore.
			select {
			case w.semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-w.semaphore }()

			// Module-level timeout.
			modCtx, cancel := context.WithTimeout(ctx, w.timeout)
			defer cancel()

			w.graph.IncrModulesRun()

			nodes, edges, err := mod.Extract(modCtx, node, w.client)
			if err != nil {
				slog.Debug("module error", "module", mod.Name(), "node", node.ID, "err", err)
				w.graph.IncrErrors()
				return
			}

			// Merge results into graph.
			for _, n := range nodes {
				if isDegraded {
					n.Confidence *= 0.5
				}
				n.Wave = node.Wave + 1
				if w.graph.AddNode(n) {
					w.graph.IncrNodesFound()

					// Pivot: dispatch new pivotable nodes within depth limit.
					if n.Pivot && n.Wave <= w.maxDepth {
						w.dispatch(ctx, n)
					}
				}
			}
			for _, e := range edges {
				// Modules pass 0 as edge ID; walker assigns real IDs.
				e.ID = fmt.Sprintf("e%d", w.graph.NextEdgeID())
				w.graph.AddEdge(e)
			}
		}()
	}
}
