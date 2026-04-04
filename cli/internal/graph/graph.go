// SPDX-License-Identifier: AGPL-3.0-or-later

package graph

import (
	"encoding/json"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Graph is a thread-safe in-memory directed graph of nodes and edges.
type Graph struct {
	mu    sync.RWMutex
	nodes map[string]*Node
	edges []*Edge

	Meta Meta `json:"meta"`

	edgeCounter atomic.Int64
	modulesRun  atomic.Int64
	nodesFound  atomic.Int64
	errorCount  atomic.Int64
}

// Meta contains scan metadata for the output.
type Meta struct {
	Version      string    `json:"basalt_version"`
	ScanID       string    `json:"scan_id"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	DurationSecs float64   `json:"duration_seconds,omitempty"`
	InitialSeeds []SeedRef `json:"initial_seeds"`
	Config       Config    `json:"config"`
	Stats        Stats     `json:"stats"`
}

// SeedRef is a lightweight reference to a seed for the meta block.
type SeedRef struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

// Config records the scan configuration for reproducibility.
type Config struct {
	MaxPivotDepth int `json:"max_pivot_depth"`
	Concurrency   int `json:"concurrency"`
	TimeoutSecs   int `json:"timeout_seconds"`
}

// Stats summarizes the scan results.
type Stats struct {
	ModulesRun int `json:"modules_run"`
	NodesFound int `json:"nodes_found"`
	Errors     int `json:"errors"`
}

// New creates an empty graph.
func New() *Graph {
	return &Graph{
		nodes: make(map[string]*Node),
	}
}

// AddNode adds a node, deduplicating by ID. Returns true if the node was new.
func (g *Graph) AddNode(n *Node) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.nodes[n.ID]; exists {
		return false
	}
	g.nodes[n.ID] = n
	return true
}

// GetNode returns a node by ID, or nil if not found.
func (g *Graph) GetNode(id string) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodes[id]
}

// AddEdge adds an edge to the graph. Edges are never deduplicated.
func (g *Graph) AddEdge(e *Edge) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.edges = append(g.edges, e)
}

// NextEdgeID returns a monotonically increasing edge ID.
func (g *Graph) NextEdgeID() int {
	return int(g.edgeCounter.Add(1))
}

// IncrModulesRun atomically increments the modules run counter.
func (g *Graph) IncrModulesRun() { g.modulesRun.Add(1) }

// IncrNodesFound atomically increments the nodes found counter.
func (g *Graph) IncrNodesFound() { g.nodesFound.Add(1) }

// IncrErrors atomically increments the error counter.
func (g *Graph) IncrErrors() { g.errorCount.Add(1) }

// Collect returns all nodes (sorted by ID) and all edges.
func (g *Graph) Collect() ([]*Node, []*Edge) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodes := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})

	edges := make([]*Edge, len(g.edges))
	copy(edges, g.edges)

	return nodes, edges
}

// SnapshotStats copies atomic counters into Meta.Stats.
func (g *Graph) SnapshotStats() {
	g.Meta.Stats.ModulesRun = int(g.modulesRun.Load())
	g.Meta.Stats.NodesFound = int(g.nodesFound.Load())
	g.Meta.Stats.Errors = int(g.errorCount.Load())
}

// RestoreStats hydrates atomic counters after loading a persisted graph.
func (g *Graph) RestoreStats(stats Stats, edgeCount int) {
	g.Meta.Stats = stats
	g.modulesRun.Store(int64(stats.ModulesRun))
	g.nodesFound.Store(int64(stats.NodesFound))
	g.errorCount.Store(int64(stats.Errors))
	g.edgeCounter.Store(int64(edgeCount))
}

// graphOutput is the JSON serialization format.
type graphOutput struct {
	Meta  Meta    `json:"meta"`
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

// MarshalJSON produces the final output with nodes as a sorted slice.
func (g *Graph) MarshalJSON() ([]byte, error) {
	g.SnapshotStats()

	nodes, edges := g.Collect()

	out := graphOutput{
		Meta:  g.Meta,
		Nodes: nodes,
		Edges: edges,
	}
	if out.Edges == nil {
		out.Edges = []*Edge{}
	}
	if out.Nodes == nil {
		out.Nodes = []*Node{}
	}

	return json.Marshal(out)
}
