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
}

// Meta contains scan metadata for the output.
type Meta struct {
	Version      string    `json:"basalt_version"`
	ScanID       string    `json:"scan_id"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	DurationSecs float64  `json:"duration_seconds,omitempty"`
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
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	MaxPivotDepth       int     `json:"max_pivot_depth"`
	Concurrency         int     `json:"concurrency"`
	TimeoutSeconds      int     `json:"timeout_seconds"`
}

// Stats summarizes the scan results.
type Stats struct {
	SitesChecked   int `json:"sites_checked"`
	AccountsFound  int `json:"accounts_found"`
	PivotsExecuted int `json:"pivots_executed"`
	Errors         int `json:"errors"`
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

// AddEdge adds an edge to the graph.
func (g *Graph) AddEdge(e *Edge) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.edges = append(g.edges, e)
}

// NextEdgeID returns a monotonically increasing edge ID.
func (g *Graph) NextEdgeID() int {
	return int(g.edgeCounter.Add(1))
}

// IncrSitesChecked atomically increments the sites checked counter.
func (g *Graph) IncrSitesChecked() {
	g.mu.Lock()
	g.Meta.Stats.SitesChecked++
	g.mu.Unlock()
}

// IncrAccountsFound atomically increments the accounts found counter.
func (g *Graph) IncrAccountsFound() {
	g.mu.Lock()
	g.Meta.Stats.AccountsFound++
	g.mu.Unlock()
}

// IncrErrors atomically increments the error counter.
func (g *Graph) IncrErrors() {
	g.mu.Lock()
	g.Meta.Stats.Errors++
	g.mu.Unlock()
}

// IncrPivotsExecuted atomically increments the pivots executed counter.
func (g *Graph) IncrPivotsExecuted() {
	g.mu.Lock()
	g.Meta.Stats.PivotsExecuted++
	g.mu.Unlock()
}

// AccountNodes returns all account nodes and the graph metadata.
func (g *Graph) AccountNodes() ([]*Node, Meta) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []*Node
	for _, n := range g.nodes {
		if n.Type == NodeTypeAccount {
			nodes = append(nodes, n)
		}
	}
	return nodes, g.Meta
}

// graphOutput is the JSON serialization format.
type graphOutput struct {
	Meta  Meta    `json:"meta"`
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

// MarshalJSON produces the final output with nodes as a sorted slice.
func (g *Graph) MarshalJSON() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Convert map to sorted slice for deterministic output.
	nodes := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})

	out := graphOutput{
		Meta:  g.Meta,
		Nodes: nodes,
		Edges: g.edges,
	}
	if out.Edges == nil {
		out.Edges = []*Edge{}
	}

	return json.Marshal(out)
}
