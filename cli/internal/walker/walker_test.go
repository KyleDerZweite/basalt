// SPDX-License-Identifier: AGPL-3.0-or-later

package walker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

// fakeModule implements modules.Module for testing.
type fakeModule struct {
	name      string
	handles   []string
	health    modules.HealthStatus
	healthMsg string
	extractFn func(ctx context.Context, node *graph.Node) ([]*graph.Node, []*graph.Edge, error)
	calls     atomic.Int64
}

func (f *fakeModule) Name() string        { return f.name }
func (f *fakeModule) Description() string { return "fake module for testing" }

func (f *fakeModule) CanHandle(nodeType string) bool {
	for _, h := range f.handles {
		if h == nodeType {
			return true
		}
	}
	return false
}

func (f *fakeModule) Extract(ctx context.Context, node *graph.Node, _ *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	f.calls.Add(1)
	if f.extractFn != nil {
		return f.extractFn(ctx, node)
	}
	return nil, nil, nil
}

func (f *fakeModule) Verify(_ context.Context, _ *httpclient.Client) (modules.HealthStatus, string) {
	return f.health, f.healthMsg
}

// TestWalkerRunsModulesForSeed verifies that a healthy module matching
// the seed type is called exactly once.
func TestWalkerRunsModulesForSeed(t *testing.T) {
	g := graph.New()
	reg := modules.NewRegistry()

	mod := &fakeModule{
		name:    "github",
		handles: []string{"username"},
		health:  modules.Healthy,
	}
	reg.Register(mod)

	w := New(g, reg, WithMaxDepth(1), WithConcurrency(2), WithTimeout(5*time.Second))
	w.Run(context.Background(), []graph.Seed{{Type: "username", Value: "testuser"}})

	if got := mod.calls.Load(); got != 1 {
		t.Fatalf("expected 1 call, got %d", got)
	}
}

// TestWalkerSkipsOfflineModules verifies that offline modules are never called.
func TestWalkerSkipsOfflineModules(t *testing.T) {
	g := graph.New()
	reg := modules.NewRegistry()

	mod := &fakeModule{
		name:    "broken",
		handles: []string{"username"},
		health:  modules.Offline,
	}
	reg.Register(mod)

	w := New(g, reg, WithMaxDepth(1), WithConcurrency(2), WithTimeout(5*time.Second))
	w.Run(context.Background(), []graph.Seed{{Type: "username", Value: "testuser"}})

	if got := mod.calls.Load(); got != 0 {
		t.Fatalf("expected 0 calls for offline module, got %d", got)
	}
}

// TestWalkerPivotsOnDiscoveredNodes verifies that when a module returns a
// pivotable node with a new type, modules handling that type get called.
func TestWalkerPivotsOnDiscoveredNodes(t *testing.T) {
	g := graph.New()
	reg := modules.NewRegistry()

	// github handles "username" and returns an email node with pivot=true.
	githubMod := &fakeModule{
		name:    "github",
		handles: []string{"username"},
		health:  modules.Healthy,
		extractFn: func(_ context.Context, _ *graph.Node) ([]*graph.Node, []*graph.Edge, error) {
			emailNode := graph.NewNode(graph.NodeTypeEmail, "user@example.com", "github")
			emailNode.Pivot = true
			emailNode.Confidence = 0.9
			return []*graph.Node{emailNode}, nil, nil
		},
	}

	// gravatar handles "email" and should be called via pivot.
	gravatarMod := &fakeModule{
		name:    "gravatar",
		handles: []string{"email"},
		health:  modules.Healthy,
	}

	reg.Register(githubMod)
	reg.Register(gravatarMod)

	w := New(g, reg, WithMaxDepth(2), WithConcurrency(5), WithTimeout(5*time.Second))
	w.Run(context.Background(), []graph.Seed{{Type: "username", Value: "testuser"}})

	if got := githubMod.calls.Load(); got != 1 {
		t.Fatalf("expected github called 1 time, got %d", got)
	}
	if got := gravatarMod.calls.Load(); got != 1 {
		t.Fatalf("expected gravatar called 1 time via pivot, got %d", got)
	}
}

// TestWalkerRespectsDepthLimit verifies that pivoting stops at maxDepth.
func TestWalkerRespectsDepthLimit(t *testing.T) {
	g := graph.New()
	reg := modules.NewRegistry()

	// This module always returns a new pivotable node, creating an
	// infinite chain. The walker must stop at maxDepth.
	counter := &atomic.Int64{}
	mod := &fakeModule{
		name:    "chain",
		handles: []string{"username"},
		health:  modules.Healthy,
		extractFn: func(_ context.Context, node *graph.Node) ([]*graph.Node, []*graph.Edge, error) {
			n := counter.Add(1)
			newNode := graph.NewNode(graph.NodeTypeUsername, fmt.Sprintf("user%d", n), "chain")
			newNode.Pivot = true
			newNode.Confidence = 0.8
			return []*graph.Node{newNode}, nil, nil
		},
	}
	reg.Register(mod)

	maxDepth := 2
	w := New(g, reg, WithMaxDepth(maxDepth), WithConcurrency(1), WithTimeout(5*time.Second))
	w.Run(context.Background(), []graph.Seed{{Type: "username", Value: "root"}})

	// Seed is wave 0. Its child is wave 1, grandchild is wave 2.
	// Wave 2 == maxDepth, so wave-2 nodes should still be dispatched
	// (wave <= maxDepth), but their children (wave 3) should not.
	// With maxDepth=2 and concurrency=1 (serial execution):
	//   wave 0 (seed) -> dispatches, produces wave 1 node
	//   wave 1 node  -> dispatches, produces wave 2 node
	//   wave 2 node  -> dispatches, produces wave 3 node (not dispatched)
	// Total calls: 3 (seed + wave1 + wave2).
	got := mod.calls.Load()
	if got != 3 {
		t.Fatalf("expected 3 calls (depth 0,1,2), got %d", got)
	}
}

// TestWalkerDedupsSameModuleNodeCombo verifies that a module is not called
// twice for the same node ID.
func TestWalkerDedupsSameModuleNodeCombo(t *testing.T) {
	g := graph.New()
	reg := modules.NewRegistry()

	// This module returns a node with the same ID as the seed node.
	mod := &fakeModule{
		name:    "echo",
		handles: []string{"username"},
		health:  modules.Healthy,
		extractFn: func(_ context.Context, _ *graph.Node) ([]*graph.Node, []*graph.Edge, error) {
			// Return a username node that would generate the same ID as the seed.
			// Since the seed ID is "seed:username:testuser", returning a different
			// type node won't collide. Instead, return the same typed node to
			// test graph-level dedup (AddNode returns false for duplicates).
			dup := graph.NewNode(graph.NodeTypeUsername, "discovered", "echo")
			dup.Pivot = true
			dup.Confidence = 0.8
			return []*graph.Node{dup}, nil, nil
		},
	}
	reg.Register(mod)

	w := New(g, reg, WithMaxDepth(3), WithConcurrency(1), WithTimeout(5*time.Second))
	w.Run(context.Background(), []graph.Seed{{Type: "username", Value: "testuser"}})

	// The module is called for the seed (wave 0). It returns "discovered"
	// which is new, so it gets dispatched (wave 1). That call also returns
	// "discovered" but AddNode returns false (duplicate), so no further dispatch.
	// Total: 2 calls.
	got := mod.calls.Load()
	if got != 2 {
		t.Fatalf("expected 2 calls (seed + discovered once), got %d", got)
	}
}

// TestWalkerDegradesPenalizesConfidence verifies that a degraded module's
// returned confidence is halved.
func TestWalkerDegradesPenalizesConfidence(t *testing.T) {
	g := graph.New()
	reg := modules.NewRegistry()

	mod := &fakeModule{
		name:    "degraded-mod",
		handles: []string{"username"},
		health:  modules.Degraded,
		extractFn: func(_ context.Context, _ *graph.Node) ([]*graph.Node, []*graph.Edge, error) {
			n := graph.NewNode(graph.NodeTypeAccount, "somesite-testuser", "degraded-mod")
			n.Confidence = 0.80
			return []*graph.Node{n}, nil, nil
		},
	}
	reg.Register(mod)

	w := New(g, reg, WithMaxDepth(1), WithConcurrency(2), WithTimeout(5*time.Second))
	w.Run(context.Background(), []graph.Seed{{Type: "username", Value: "testuser"}})

	node := g.GetNode("account:somesite-testuser")
	if node == nil {
		t.Fatal("expected account node to be in graph")
	}
	if node.Confidence != 0.40 {
		t.Fatalf("expected confidence 0.40, got %.2f", node.Confidence)
	}
}

// TestWalkerGracefulShutdown verifies that cancelling the context causes
// Run to return promptly even if a module is slow.
func TestWalkerGracefulShutdown(t *testing.T) {
	g := graph.New()
	reg := modules.NewRegistry()

	mod := &fakeModule{
		name:    "slow",
		handles: []string{"username"},
		health:  modules.Healthy,
		extractFn: func(ctx context.Context, _ *graph.Node) ([]*graph.Node, []*graph.Edge, error) {
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(10 * time.Second):
				return nil, nil, nil
			}
		},
	}
	reg.Register(mod)

	ctx, cancel := context.WithCancel(context.Background())
	w := New(g, reg, WithMaxDepth(1), WithConcurrency(2), WithTimeout(30*time.Second))

	done := make(chan struct{})
	go func() {
		w.Run(ctx, []graph.Seed{{Type: "username", Value: "testuser"}})
		close(done)
	}()

	// Cancel after 100ms.
	time.AfterFunc(100*time.Millisecond, cancel)

	select {
	case <-done:
		// Run returned promptly after cancellation.
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return within 2s after context cancellation")
	}
}

func TestWalkerEmitsEvents(t *testing.T) {
	g := graph.New()
	reg := modules.NewRegistry()

	mod := &fakeModule{
		name:    "github",
		handles: []string{"username"},
		health:  modules.Healthy,
		extractFn: func(_ context.Context, _ *graph.Node) ([]*graph.Node, []*graph.Edge, error) {
			node := graph.NewNode(graph.NodeTypeEmail, "user@example.com", "github")
			node.Confidence = 0.9
			edge := graph.NewEdge(0, "seed:username:testuser", node.ID, graph.EdgeTypeHasEmail, "github")
			return []*graph.Node{node}, []*graph.Edge{edge}, nil
		},
	}
	reg.Register(mod)

	var (
		mu     sync.Mutex
		events []Event
	)
	w := New(
		g,
		reg,
		WithEventHandler(func(event Event) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, event)
		}),
	)
	w.Run(context.Background(), []graph.Seed{{Type: "username", Value: "testuser"}})

	mu.Lock()
	defer mu.Unlock()

	if len(events) == 0 {
		t.Fatal("expected walker events")
	}

	var foundVerify, foundStarted, foundNode, foundEdge, foundFinished bool
	for _, event := range events {
		switch event.Type {
		case "module_verified":
			foundVerify = true
		case "module_started":
			foundStarted = true
		case "node_discovered":
			foundNode = true
		case "edge_discovered":
			foundEdge = true
		case "module_finished":
			foundFinished = true
		}
	}

	if !foundVerify || !foundStarted || !foundNode || !foundEdge || !foundFinished {
		t.Fatalf("missing expected events: %+v", events)
	}
}
