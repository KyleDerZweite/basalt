# Basalt v2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Gut the old site-check architecture and rebuild basalt as a relation-based OSINT tool with 18 purpose-built modules, an async graph walker, and module health system.

**Architecture:** A reactive graph walker dispatches nodes to modules that declare what node types they handle. Modules extract metadata and return new nodes/edges. The walker immediately feeds pivotable nodes back into matching modules. Execution is fully async, bounded by a concurrency semaphore and depth limit.

**Tech Stack:** Go 1.25, Cobra CLI, fatih/color (terminal), goquery (HTML scraping), golang.org/x/time/rate (rate limiting), golang.org/x/net (proxy/SOCKS5)

**Spec:** `docs/superpowers/specs/2026-03-29-basalt-v2-design.md`

**Module path:** `github.com/kyle/basalt` (working directory: `cli/`)

---

## File Map

### Keep unchanged
- `cli/main.go` - entry point
- `cli/internal/httpclient/client.go` - HTTP client with retries
- `cli/internal/httpclient/ratelimit.go` - per-domain rate limiting
- `cli/internal/httpclient/proxy.go` - proxy pool rotation
- `cli/internal/httpclient/dnscache.go` - DNS cache

### Delete
- `cli/internal/engine/` - entire directory (old engine interface)
- `cli/internal/engines/` - entire directory (old username/email checkers)
- `cli/internal/sitedb/` - entire directory (YAML site definitions)
- `cli/internal/pivot/` - entire directory (old pivot controller)
- `cli/internal/resolver/` - entire directory (old resolver)
- `cli/internal/output/table.go` - old table formatter (will rewrite)
- `cli/internal/output/json.go` - old JSON formatter (will rewrite)
- `cli/cmd/scan.go` - old scan command (will rewrite)
- `cli/cmd/importcmd.go` - import command (no longer needed)
- `data/` - YAML site definition files

### Modify
- `cli/internal/graph/node.go` - new node types, promoted fields (Pivot, Wave, Confidence, SourceModule)
- `cli/internal/graph/edge.go` - new edge types and constructors
- `cli/internal/graph/graph.go` - update AccountNodes -> AllNodes, simplify for new model
- `cli/cmd/root.go` - update flags for v2 CLI interface
- `cli/cmd/version.go` - bump version to 2.0.0

### Create
- `cli/internal/modules/module.go` - Module interface, HealthStatus (already exists, will rewrite)
- `cli/internal/modules/registry.go` - module registry
- `cli/internal/walker/walker.go` - async dispatch loop
- `cli/internal/config/config.go` - config file loader
- `cli/internal/output/table.go` - new table formatter
- `cli/internal/output/json.go` - new JSON exporter
- `cli/internal/output/csv.go` - new CSV exporter
- `cli/cmd/scan.go` - new scan command wiring walker + output
- `cli/internal/modules/gravatar/gravatar.go` - Gravatar module
- `cli/internal/modules/linktree/linktree.go` - Linktree module
- `cli/internal/modules/beacons/beacons.go` - Beacons module
- `cli/internal/modules/carrd/carrd.go` - Carrd module
- `cli/internal/modules/bento/bento.go` - Bento module
- `cli/internal/modules/github/github.go` - GitHub module
- `cli/internal/modules/gitlab/gitlab.go` - GitLab module
- `cli/internal/modules/stackexchange/stackexchange.go` - StackExchange module
- `cli/internal/modules/reddit/reddit.go` - Reddit module
- `cli/internal/modules/youtube/youtube.go` - YouTube module
- `cli/internal/modules/twitch/twitch.go` - Twitch module
- `cli/internal/modules/discord/discord.go` - Discord module
- `cli/internal/modules/instagram/instagram.go` - Instagram module
- `cli/internal/modules/tiktok/tiktok.go` - TikTok module
- `cli/internal/modules/matrix/matrix.go` - Matrix module
- `cli/internal/modules/steam/steam.go` - Steam module
- `cli/internal/modules/whois/whois.go` - WHOIS/RDAP module
- `cli/internal/modules/dnsct/dnsct.go` - DNS/CT module

### Test files
- `cli/internal/graph/graph_test.go`
- `cli/internal/modules/registry_test.go`
- `cli/internal/walker/walker_test.go`
- `cli/internal/config/config_test.go`
- `cli/internal/output/table_test.go`
- `cli/internal/output/json_test.go`
- `cli/internal/output/csv_test.go`
- `cli/internal/modules/gravatar/gravatar_test.go`
- `cli/internal/modules/github/github_test.go`
- (one test file per module)

---

## Task Dependency Graph

```
Task 1 (gut old code) ─────────────────────────┐
                                                 v
Task 2 (graph refactor) ──────────> Task 4 (walker)
                                        |
Task 3 (module interface + registry) ───┘
                                        |
Task 5 (config) ────────────────────────┘
                                        |
                                        v
Task 6-12 (modules, parallelizable) ──> Task 13 (output)
                                        |
                                        v
                                   Task 14 (scan command)
                                        |
                                        v
                                   Task 15 (docs)
```

**Parallelizable groups:**
- Tasks 2, 3, 5 can run in parallel (no dependencies on each other after Task 1)
- Tasks 6-12 (all module tasks) can run in parallel after Tasks 2, 3, 5 complete
- Task 13 (output) can start after Task 2 (graph) is done
- Task 14 (scan command) needs Tasks 4, 6-12, 13
- Task 15 (docs) needs Task 14

---

## Task 1: Gut Old Code

**Files:**
- Delete: `cli/internal/engine/` (entire directory)
- Delete: `cli/internal/engines/` (entire directory)
- Delete: `cli/internal/sitedb/` (entire directory)
- Delete: `cli/internal/pivot/` (entire directory)
- Delete: `cli/internal/resolver/` (entire directory)
- Delete: `cli/internal/output/table.go`
- Delete: `cli/internal/output/json.go`
- Delete: `cli/cmd/scan.go`
- Delete: `cli/cmd/importcmd.go`

- [ ] **Step 1: Delete old packages**

```bash
cd cli
rm -rf internal/engine internal/engines internal/sitedb internal/pivot internal/resolver
rm -f internal/output/table.go internal/output/json.go
rm -f cmd/scan.go cmd/importcmd.go
```

- [ ] **Step 2: Remove the output package directory if empty**

```bash
cd cli
rmdir internal/output 2>/dev/null || true
```

- [ ] **Step 3: Clean up go.mod**

Remove `gopkg.in/yaml.v3` from go.mod since YAML site definitions are gone. Keep `goquery`, `color`, `cobra`, `golang.org/x/time`, `golang.org/x/net`.

```bash
cd cli
go mod tidy
```

- [ ] **Step 4: Verify the build compiles**

The build will fail because `cmd/root.go` references flags used by the deleted scan.go, and `main.go` still works. Strip root.go of stale flags.

Edit `cli/cmd/root.go` to remove the old flags (`flagOutput`, `flagConcurrency`, `flagTimeout`, `flagVerbose`, `flagThreshold`) and their `init()` registrations. Keep only the root command skeleton:

```go
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
```

- [ ] **Step 5: Bump version**

Edit `cli/cmd/version.go`: change `Version` from `"0.1.0"` to `"2.0.0-dev"`.

- [ ] **Step 6: Verify clean build**

```bash
cd cli
go build ./...
go vet ./...
```

Expected: builds and vets cleanly. The binary won't do anything useful yet (only `version` subcommand works).

- [ ] **Step 7: Commit**

```bash
cd cli
git add -A
git commit -m "gut old engine, sitedb, pivot, output, and import code

Remove the YAML site-check architecture to make room for v2 module-based
system. Keep graph/, httpclient/, CLI skeleton, and main.go."
```

---

## Task 2: Refactor Graph Package

**Files:**
- Modify: `cli/internal/graph/node.go`
- Modify: `cli/internal/graph/edge.go`
- Modify: `cli/internal/graph/graph.go`
- Create: `cli/internal/graph/graph_test.go`

**Depends on:** Task 1

- [ ] **Step 1: Write graph tests**

Create `cli/internal/graph/graph_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package graph

import (
	"encoding/json"
	"testing"
)

func TestNewNode(t *testing.T) {
	n := NewNode("username", "kylederzweite", "github")
	if n.ID != "username:kylederzweite" {
		t.Errorf("got ID %q, want %q", n.ID, "username:kylederzweite")
	}
	if n.Type != NodeTypeUsername {
		t.Errorf("got Type %q, want %q", n.Type, NodeTypeUsername)
	}
	if n.Label != "kylederzweite" {
		t.Errorf("got Label %q, want %q", n.Label, "kylederzweite")
	}
	if n.SourceModule != "github" {
		t.Errorf("got SourceModule %q, want %q", n.SourceModule, "github")
	}
}

func TestNewAccountNode(t *testing.T) {
	n := NewAccountNode("github", "kylederzweite", "https://github.com/kylederzweite", "github")
	if n.ID != "account:github:kylederzweite" {
		t.Errorf("got ID %q, want %q", n.ID, "account:github:kylederzweite")
	}
	if n.Type != NodeTypeAccount {
		t.Errorf("got Type %q, want %q", n.Type, NodeTypeAccount)
	}
	wantLabel := "github - kylederzweite"
	if n.Label != wantLabel {
		t.Errorf("got Label %q, want %q", n.Label, wantLabel)
	}
}

func TestNodePivotAndWave(t *testing.T) {
	n := NewNode("email", "kyle@example.com", "gravatar")
	n.Pivot = true
	n.Wave = 1
	n.Confidence = 0.95

	if !n.Pivot {
		t.Error("expected Pivot=true")
	}
	if n.Wave != 1 {
		t.Errorf("got Wave %d, want 1", n.Wave)
	}
	if n.Confidence != 0.95 {
		t.Errorf("got Confidence %f, want 0.95", n.Confidence)
	}
}

func TestGraphAddNodeDedup(t *testing.T) {
	g := New()
	n1 := NewNode("username", "kyle", "github")
	n2 := NewNode("username", "kyle", "reddit")

	if !g.AddNode(n1) {
		t.Error("first AddNode should return true")
	}
	if g.AddNode(n2) {
		t.Error("second AddNode with same ID should return false")
	}
}

func TestGraphEdgesNotDeduplicated(t *testing.T) {
	g := New()
	e1 := NewEdge(g.NextEdgeID(), "a", "b", EdgeTypeHasAccount, "github")
	e2 := NewEdge(g.NextEdgeID(), "a", "b", EdgeTypeHasAccount, "gravatar")
	g.AddEdge(e1)
	g.AddEdge(e2)

	nodes, edges := g.Collect()
	_ = nodes
	if len(edges) != 2 {
		t.Errorf("got %d edges, want 2 (edges should not be deduplicated)", len(edges))
	}
}

func TestSeedNodeID(t *testing.T) {
	id := SeedNodeID("username", "KyleDerZweite")
	if id != "seed:username:kylederzweite" {
		t.Errorf("got %q, want %q", id, "seed:username:kylederzweite")
	}
}

func TestParseSeed(t *testing.T) {
	s, err := ParseSeed("email:kyle@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if s.Type != "email" || s.Value != "kyle@example.com" {
		t.Errorf("got %+v", s)
	}

	_, err = ParseSeed("invalid")
	if err == nil {
		t.Error("expected error for invalid seed format")
	}
}

func TestGraphMarshalJSON(t *testing.T) {
	g := New()
	n := NewNode("username", "kyle", "test")
	g.AddNode(n)

	data, err := g.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["nodes"]; !ok {
		t.Error("JSON output missing 'nodes' key")
	}
	if _, ok := out["edges"]; !ok {
		t.Error("JSON output missing 'edges' key")
	}
	if _, ok := out["meta"]; !ok {
		t.Error("JSON output missing 'meta' key")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cli
go test ./internal/graph/ -v
```

Expected: compilation errors (NewNode, NewEdge, NewAccountNode signatures don't match, Collect doesn't exist, etc.).

- [ ] **Step 3: Rewrite node.go**

Replace `cli/internal/graph/node.go` with:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package graph

import (
	"fmt"
	"strings"
)

// Node types.
const (
	NodeTypeSeed         = "seed"
	NodeTypeAccount      = "account"
	NodeTypeUsername      = "username"
	NodeTypeEmail        = "email"
	NodeTypeDomain       = "domain"
	NodeTypeIP           = "ip"
	NodeTypeOrganization = "organization"
	NodeTypePhone        = "phone"
	NodeTypeFullName     = "full_name"
	NodeTypeAvatarURL    = "avatar_url"
	NodeTypeWebsite      = "website"
)

// Node represents an entity in the intelligence graph.
type Node struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Label        string                 `json:"label"`
	SourceModule string                 `json:"source_module"`
	Pivot        bool                   `json:"pivot"`
	Wave         int                    `json:"wave"`
	Confidence   float64                `json:"confidence"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// NewNode creates a node with the given type, value, and source module.
// The node ID is "type:lowercased_value".
func NewNode(nodeType, value, sourceModule string) *Node {
	return &Node{
		ID:           fmt.Sprintf("%s:%s", nodeType, strings.ToLower(value)),
		Type:         nodeType,
		Label:        value,
		SourceModule: sourceModule,
		Properties:   make(map[string]interface{}),
	}
}

// NewSeedNode creates a node representing an input seed.
func NewSeedNode(seedType, value string) *Node {
	return &Node{
		ID:           SeedNodeID(seedType, value),
		Type:         NodeTypeSeed,
		Label:        value,
		SourceModule: "seed",
		Pivot:        true,
		Wave:         0,
		Properties: map[string]interface{}{
			"seed_type": seedType,
		},
	}
}

// NewAccountNode creates a node representing a discovered account on a platform.
func NewAccountNode(platform, seedValue, profileURL, sourceModule string) *Node {
	return &Node{
		ID:           AccountNodeID(platform, seedValue),
		Type:         NodeTypeAccount,
		Label:        fmt.Sprintf("%s - %s", platform, seedValue),
		SourceModule: sourceModule,
		Properties: map[string]interface{}{
			"site_name":   platform,
			"profile_url": profileURL,
		},
	}
}

// SeedNodeID generates a deterministic node ID for a seed.
func SeedNodeID(seedType, value string) string {
	return fmt.Sprintf("seed:%s:%s", seedType, strings.ToLower(value))
}

// AccountNodeID generates a deterministic node ID for an account.
func AccountNodeID(platform, seedValue string) string {
	return fmt.Sprintf("account:%s:%s", strings.ToLower(platform), strings.ToLower(seedValue))
}

// Seed represents a seed entity for scanning.
type Seed struct {
	Type  string
	Value string
}

// ParseSeed parses a seed string in format "type:value".
func ParseSeed(s string) (Seed, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Seed{}, fmt.Errorf("invalid seed format %q (expected type:value)", s)
	}
	return Seed{Type: parts[0], Value: parts[1]}, nil
}
```

- [ ] **Step 4: Rewrite edge.go**

Replace `cli/internal/graph/edge.go` with:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package graph

import "fmt"

// Edge types.
const (
	EdgeTypeHasAccount   = "has_account"
	EdgeTypeHasEmail     = "has_email"
	EdgeTypeHasDomain    = "has_domain"
	EdgeTypeHasUsername   = "has_username"
	EdgeTypeRegisteredTo = "registered_to"
	EdgeTypeResolvesTo   = "resolves_to"
	EdgeTypeLinkedTo     = "linked_to"
	EdgeTypeMentions     = "mentions"
)

// Edge represents a directed relationship between two nodes.
type Edge struct {
	ID           string                 `json:"id"`
	Source       string                 `json:"source"`
	Target       string                 `json:"target"`
	Type         string                 `json:"type"`
	SourceModule string                 `json:"source_module"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// NewEdge creates an edge between two nodes.
func NewEdge(id int, source, target, edgeType, sourceModule string) *Edge {
	return &Edge{
		ID:           fmt.Sprintf("e%d", id),
		Source:       source,
		Target:       target,
		Type:         edgeType,
		SourceModule: sourceModule,
	}
}
```

- [ ] **Step 5: Update graph.go**

Replace the `AccountNodes` method with `Collect` which returns all nodes and all edges. Remove the old `graphOutput` struct dependency on the old node types. The rest (New, AddNode, AddEdge, NextEdgeID, atomic counters, MarshalJSON) stays largely the same.

Replace `cli/internal/graph/graph.go` with:

```go
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

	edgeCounter   atomic.Int64
	modulesRun    atomic.Int64
	nodesFound    atomic.Int64
	errorCount    atomic.Int64
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

// Collect returns all nodes (sorted by ID) and all edges, plus metadata.
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
```

- [ ] **Step 6: Run tests**

```bash
cd cli
go test ./internal/graph/ -v
```

Expected: all tests pass.

- [ ] **Step 7: Build and vet**

```bash
cd cli
go build ./...
go vet ./...
```

- [ ] **Step 8: Commit**

```bash
cd cli
git add internal/graph/
git commit -m "refactor graph package for v2 module-based architecture

Promote Pivot, Wave, Confidence, SourceModule to top-level Node fields.
Add new node types (full_name, avatar_url, website) and edge types
(has_account, has_email, has_domain, etc). Simplify NewAccountNode.
Replace AccountNodes with Collect. Add tests."
```

---

## Task 3: Module Interface and Registry

**Files:**
- Rewrite: `cli/internal/modules/module.go`
- Create: `cli/internal/modules/registry.go`
- Create: `cli/internal/modules/registry_test.go`

**Depends on:** Task 1 (old engine deleted), Task 2 (graph types available)

- [ ] **Step 1: Write registry tests**

Create `cli/internal/modules/registry_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"
	"testing"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
)

// stubModule is a minimal Module implementation for testing.
type stubModule struct {
	name       string
	handles    []string
	health     HealthStatus
	healthMsg  string
}

func (s *stubModule) Name() string        { return s.name }
func (s *stubModule) Description() string { return "stub module for testing" }
func (s *stubModule) CanHandle(nodeType string) bool {
	for _, h := range s.handles {
		if h == nodeType {
			return true
		}
	}
	return false
}
func (s *stubModule) Extract(_ context.Context, _ *graph.Node, _ *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	return nil, nil, nil
}
func (s *stubModule) Verify(_ context.Context, _ *httpclient.Client) (HealthStatus, string) {
	return s.health, s.healthMsg
}

func TestRegistryRegisterAndLookup(t *testing.T) {
	reg := NewRegistry()
	m := &stubModule{name: "github", handles: []string{"username", "email"}}
	reg.Register(m)

	got := reg.ModulesFor("username")
	if len(got) != 1 || got[0].Name() != "github" {
		t.Errorf("expected github module for username, got %v", got)
	}

	got = reg.ModulesFor("email")
	if len(got) != 1 || got[0].Name() != "github" {
		t.Errorf("expected github module for email, got %v", got)
	}

	got = reg.ModulesFor("domain")
	if len(got) != 0 {
		t.Errorf("expected no modules for domain, got %v", got)
	}
}

func TestRegistryAll(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubModule{name: "a", handles: []string{"username"}})
	reg.Register(&stubModule{name: "b", handles: []string{"email"}})

	all := reg.All()
	if len(all) != 2 {
		t.Errorf("expected 2 modules, got %d", len(all))
	}
}

func TestRegistryModulesForNodeType(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubModule{name: "gravatar", handles: []string{"email"}})
	reg.Register(&stubModule{name: "github", handles: []string{"username", "email"}})
	reg.Register(&stubModule{name: "whois", handles: []string{"domain"}})

	emailModules := reg.ModulesFor("email")
	if len(emailModules) != 2 {
		t.Errorf("expected 2 email modules, got %d", len(emailModules))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cli
go test ./internal/modules/ -v
```

Expected: compilation errors (HealthStatus, NewRegistry, Register, ModulesFor, All don't exist).

- [ ] **Step 3: Rewrite module.go**

Replace `cli/internal/modules/module.go` with:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
)

// HealthStatus represents the operational state of a module.
type HealthStatus int

const (
	// Healthy means verify passed, results scored normally.
	Healthy HealthStatus = iota
	// Degraded means verify got a response but data was unexpected. Confidence * 0.5.
	Degraded
	// Offline means verify failed entirely. Module skipped.
	Offline
)

// Module is the interface every OSINT extraction module must implement.
type Module interface {
	// Name returns a human-readable identifier (e.g., "github").
	Name() string

	// Description returns what this module does.
	Description() string

	// CanHandle reports whether this module can process the given node type.
	CanHandle(nodeType string) bool

	// Extract processes a node and returns discovered nodes and edges.
	// Must respect context cancellation.
	Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error)

	// Verify runs a lightweight self-test against a known entity.
	// Called once at startup. Returns status and a human-readable message.
	Verify(ctx context.Context, client *httpclient.Client) (HealthStatus, string)
}
```

- [ ] **Step 4: Create registry.go**

Create `cli/internal/modules/registry.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

// Registry maintains all registered modules.
type Registry struct {
	modules []Module
}

// NewRegistry creates an empty module registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a module to the registry.
func (r *Registry) Register(m Module) {
	r.modules = append(r.modules, m)
}

// ModulesFor returns all modules that can handle the given node type.
func (r *Registry) ModulesFor(nodeType string) []Module {
	var result []Module
	for _, m := range r.modules {
		if m.CanHandle(nodeType) {
			result = append(result, m)
		}
	}
	return result
}

// All returns all registered modules.
func (r *Registry) All() []Module {
	return r.modules
}
```

- [ ] **Step 5: Run tests**

```bash
cd cli
go test ./internal/modules/ -v
```

Expected: all tests pass.

- [ ] **Step 6: Build and vet**

```bash
cd cli
go build ./...
go vet ./...
```

- [ ] **Step 7: Commit**

```bash
cd cli
git add internal/modules/
git commit -m "add Module interface with HealthStatus and Registry

Module interface: Name, Description, CanHandle, Extract, Verify.
Three health states: Healthy, Degraded, Offline.
Registry indexes modules and looks up by node type."
```

---

## Task 4: Walker (Async Orchestrator)

**Files:**
- Create: `cli/internal/walker/walker.go`
- Create: `cli/internal/walker/walker_test.go`

**Depends on:** Task 2 (graph), Task 3 (module interface)

- [ ] **Step 1: Write walker tests**

Create `cli/internal/walker/walker_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package walker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

// fakeModule returns predictable results for testing.
type fakeModule struct {
	name      string
	handles   []string
	health    modules.HealthStatus
	healthMsg string
	extractFn func(ctx context.Context, node *graph.Node) ([]*graph.Node, []*graph.Edge, error)
	calls     atomic.Int64
}

func (f *fakeModule) Name() string        { return f.name }
func (f *fakeModule) Description() string { return "fake" }
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

func TestWalkerRunsModulesForSeed(t *testing.T) {
	gh := &fakeModule{
		name:    "github",
		handles: []string{"username"},
		health:  modules.Healthy,
	}

	reg := modules.NewRegistry()
	reg.Register(gh)

	g := graph.New()
	w := New(g, reg, WithMaxDepth(0), WithConcurrency(5), WithTimeout(5*time.Second))

	ctx := context.Background()
	seeds := []graph.Seed{{Type: "username", Value: "testuser"}}
	w.Run(ctx, seeds)

	if gh.calls.Load() != 1 {
		t.Errorf("expected github module called once, got %d", gh.calls.Load())
	}
}

func TestWalkerSkipsOfflineModules(t *testing.T) {
	offline := &fakeModule{
		name:      "steam",
		handles:   []string{"username"},
		health:    modules.Offline,
		healthMsg: "no API key",
	}

	reg := modules.NewRegistry()
	reg.Register(offline)

	g := graph.New()
	w := New(g, reg, WithMaxDepth(0), WithConcurrency(5), WithTimeout(5*time.Second))

	ctx := context.Background()
	w.Run(ctx, []graph.Seed{{Type: "username", Value: "test"}})

	if offline.calls.Load() != 0 {
		t.Errorf("expected offline module not called, got %d calls", offline.calls.Load())
	}
}

func TestWalkerPivotsOnDiscoveredNodes(t *testing.T) {
	emailModule := &fakeModule{
		name:    "gravatar",
		handles: []string{"email"},
		health:  modules.Healthy,
	}

	ghModule := &fakeModule{
		name:    "github",
		handles: []string{"username"},
		health:  modules.Healthy,
		extractFn: func(_ context.Context, _ *graph.Node) ([]*graph.Node, []*graph.Edge, error) {
			email := graph.NewNode("email", "found@example.com", "github")
			email.Pivot = true
			email.Confidence = 0.90
			return []*graph.Node{email}, nil, nil
		},
	}

	reg := modules.NewRegistry()
	reg.Register(ghModule)
	reg.Register(emailModule)

	g := graph.New()
	w := New(g, reg, WithMaxDepth(2), WithConcurrency(5), WithTimeout(5*time.Second))

	ctx := context.Background()
	w.Run(ctx, []graph.Seed{{Type: "username", Value: "test"}})

	// GitHub ran on the username seed, discovered an email.
	// Gravatar should have been dispatched for that email.
	if emailModule.calls.Load() != 1 {
		t.Errorf("expected gravatar called once for pivoted email, got %d", emailModule.calls.Load())
	}
}

func TestWalkerRespectsDepthLimit(t *testing.T) {
	// Module that always discovers a new username, creating infinite pivot chain.
	counter := &atomic.Int64{}
	infinite := &fakeModule{
		name:    "infinite",
		handles: []string{"username"},
		health:  modules.Healthy,
		extractFn: func(_ context.Context, _ *graph.Node) ([]*graph.Node, []*graph.Edge, error) {
			n := counter.Add(1)
			node := graph.NewNode("username", fmt.Sprintf("user%d", n), "infinite")
			node.Pivot = true
			node.Confidence = 0.9
			return []*graph.Node{node}, nil, nil
		},
	}

	reg := modules.NewRegistry()
	reg.Register(infinite)

	g := graph.New()
	w := New(g, reg, WithMaxDepth(2), WithConcurrency(1), WithTimeout(5*time.Second))

	ctx := context.Background()
	w.Run(ctx, []graph.Seed{{Type: "username", Value: "seed"}})

	// Depth 0: seed -> discovers user1 (wave 1)
	// Depth 1: user1 -> discovers user2 (wave 2)
	// Depth 2: user2 -> would be wave 3, exceeds maxDepth=2, stop
	// So module should be called at most 3 times (seed + user1 + user2)
	calls := infinite.calls.Load()
	if calls > 3 {
		t.Errorf("expected at most 3 calls with depth 2, got %d", calls)
	}
}

func TestWalkerDedupsSameModuleNodeCombo(t *testing.T) {
	gh := &fakeModule{
		name:    "github",
		handles: []string{"username"},
		health:  modules.Healthy,
		extractFn: func(_ context.Context, _ *graph.Node) ([]*graph.Node, []*graph.Edge, error) {
			// Returns a username node that is the same as the seed - should not re-trigger.
			node := graph.NewNode("username", "testuser", "github")
			node.Pivot = true
			node.Confidence = 0.9
			return []*graph.Node{node}, nil, nil
		},
	}

	reg := modules.NewRegistry()
	reg.Register(gh)

	g := graph.New()
	w := New(g, reg, WithMaxDepth(2), WithConcurrency(5), WithTimeout(5*time.Second))

	ctx := context.Background()
	w.Run(ctx, []graph.Seed{{Type: "username", Value: "testuser"}})

	// The seed node and the returned node have the same ID.
	// The dispatch dedup should prevent github from running twice.
	if gh.calls.Load() != 1 {
		t.Errorf("expected exactly 1 call (dedup should prevent re-dispatch), got %d", gh.calls.Load())
	}
}

func TestWalkerDegradesPenalizesConfidence(t *testing.T) {
	degraded := &fakeModule{
		name:      "linktree",
		handles:   []string{"username"},
		health:    modules.Degraded,
		healthMsg: "unexpected DOM",
		extractFn: func(_ context.Context, _ *graph.Node) ([]*graph.Node, []*graph.Edge, error) {
			node := graph.NewNode("website", "https://example.com", "linktree")
			node.Confidence = 0.80
			return []*graph.Node{node}, nil, nil
		},
	}

	reg := modules.NewRegistry()
	reg.Register(degraded)

	g := graph.New()
	w := New(g, reg, WithMaxDepth(0), WithConcurrency(5), WithTimeout(5*time.Second))

	ctx := context.Background()
	w.Run(ctx, []graph.Seed{{Type: "username", Value: "test"}})

	node := g.GetNode("website:https://example.com")
	if node == nil {
		t.Fatal("expected website node to exist")
	}
	// 0.80 * 0.5 = 0.40
	if node.Confidence != 0.40 {
		t.Errorf("expected confidence 0.40 (degraded penalty), got %f", node.Confidence)
	}
}

func TestWalkerGracefulShutdown(t *testing.T) {
	slow := &fakeModule{
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

	reg := modules.NewRegistry()
	reg.Register(slow)

	g := graph.New()
	w := New(g, reg, WithMaxDepth(0), WithConcurrency(5), WithTimeout(5*time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	w.Run(ctx, []graph.Seed{{Type: "username", Value: "test"}})
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("expected graceful shutdown within ~100ms, took %s", elapsed)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cli
go test ./internal/walker/ -v
```

Expected: compilation errors (walker package doesn't exist).

- [ ] **Step 3: Implement walker.go**

Create `cli/internal/walker/walker.go`:

```go
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
func WithMaxDepth(d int) Option {
	return func(w *Walker) { w.maxDepth = d }
}

// WithConcurrency sets the maximum concurrent module executions.
func WithConcurrency(n int) Option {
	return func(w *Walker) { w.concurrency = n }
}

// WithTimeout sets the per-module execution timeout.
func WithTimeout(d time.Duration) Option {
	return func(w *Walker) { w.timeout = d }
}

// WithClient sets the HTTP client.
func WithClient(c *httpclient.Client) Option {
	return func(w *Walker) { w.client = c }
}

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
// to get the health summary for display. If not called, Run verifies automatically.
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

	// Phase 3: Dispatch seed nodes.
	for _, node := range seedNodes {
		w.dispatch(ctx, node)
	}

	// Phase 4: Wait for all in-flight work to complete.
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

// dispatch sends a node to all matching healthy modules.
func (w *Walker) dispatch(ctx context.Context, node *graph.Node) {
	for _, mh := range w.healthy {
		if mh.Status == modules.Offline {
			continue
		}
		if !mh.Module.CanHandle(node.Type) {
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
```

- [ ] **Step 4: Add missing import in test file**

The test file uses `fmt.Sprintf` in `TestWalkerRespectsDepthLimit`. Add `"fmt"` to the import block of `walker_test.go`.

- [ ] **Step 5: Run tests**

```bash
cd cli
go test ./internal/walker/ -v -timeout 30s
```

Expected: all tests pass.

- [ ] **Step 6: Build and vet**

```bash
cd cli
go build ./...
go vet ./...
```

- [ ] **Step 7: Commit**

```bash
cd cli
git add internal/walker/
git commit -m "add async Walker orchestrator with health checks and pivoting

Fully async dispatch loop: modules fire immediately as matching nodes
appear. Dedup by module+node combo. Module-level timeout via context.
Degraded modules have confidence penalized by 0.5. Graceful shutdown
via context cancellation."
```

---

## Task 5: Config Package

**Files:**
- Create: `cli/internal/config/config.go`
- Create: `cli/internal/config/config_test.go`

**Depends on:** Task 1

- [ ] **Step 1: Write config tests**

Create `cli/internal/config/config_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := "STEAM_API_KEY=abc123\nGITHUB_TOKEN=ghp_test\n# comment\nEMPTY_VAL=\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if got := cfg.Get("STEAM_API_KEY"); got != "abc123" {
		t.Errorf("STEAM_API_KEY = %q, want %q", got, "abc123")
	}
	if got := cfg.Get("GITHUB_TOKEN"); got != "ghp_test" {
		t.Errorf("GITHUB_TOKEN = %q, want %q", got, "ghp_test")
	}
	if got := cfg.Get("EMPTY_VAL"); got != "" {
		t.Errorf("EMPTY_VAL = %q, want empty", got)
	}
	if got := cfg.Get("MISSING"); got != "" {
		t.Errorf("MISSING = %q, want empty", got)
	}
}

func TestLoadDefaultPath(t *testing.T) {
	// When no file exists, Load should return an empty config, not an error.
	cfg, err := Load("/nonexistent/path/config")
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("ANYTHING"); got != "" {
		t.Errorf("expected empty for missing config, got %q", got)
	}
}

func TestLoadSkipsBlankLinesAndComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := "\n\n# A comment\n\nKEY=value\n\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("KEY"); got != "value" {
		t.Errorf("KEY = %q, want %q", got, "value")
	}
}

func TestLoadQuotedValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := "KEY=\"hello world\"\nKEY2='single quoted'\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("KEY"); got != "hello world" {
		t.Errorf("KEY = %q, want %q", got, "hello world")
	}
	if got := cfg.Get("KEY2"); got != "single quoted" {
		t.Errorf("KEY2 = %q, want %q", got, "single quoted")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cli
go test ./internal/config/ -v
```

Expected: compilation errors (config package doesn't exist).

- [ ] **Step 3: Implement config.go**

Create `cli/internal/config/config.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"bufio"
	"os"
	"strings"
)

// Config holds key-value pairs loaded from a config file.
type Config struct {
	values map[string]string
}

// Load reads a config file in KEY=VALUE format.
// Blank lines and lines starting with # are ignored.
// If the file does not exist, returns an empty Config (not an error).
func Load(path string) (*Config, error) {
	cfg := &Config{values: make(map[string]string)}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Strip surrounding quotes.
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		cfg.values[key] = value
	}

	return cfg, scanner.Err()
}

// Get returns the value for a key, or empty string if not set.
func (c *Config) Get(key string) string {
	return c.values[key]
}
```

- [ ] **Step 4: Run tests**

```bash
cd cli
go test ./internal/config/ -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
cd cli
git add internal/config/
git commit -m "add config package for loading API keys from file

Reads KEY=VALUE format with comment and quote support.
Missing file returns empty config (not error) so modules
can check keys without crashing."
```

---

## Task 6: Gravatar Module

**Files:**
- Create: `cli/internal/modules/gravatar/gravatar.go`
- Create: `cli/internal/modules/gravatar/gravatar_test.go`

**Depends on:** Tasks 2, 3

- [ ] **Step 1: Write tests**

Create `cli/internal/modules/gravatar/gravatar_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package gravatar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

func TestCanHandle(t *testing.T) {
	m := New()
	if !m.CanHandle("email") {
		t.Error("should handle email")
	}
	if m.CanHandle("username") {
		t.Error("should not handle username")
	}
}

func TestExtractFound(t *testing.T) {
	profile := map[string]interface{}{
		"displayName":   "Kyle Test",
		"preferredUsername": "kyletest",
		"thumbnailUrl":  "https://gravatar.com/avatar/abc123",
		"urls": []interface{}{
			map[string]interface{}{"value": "https://kylehub.dev"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL // override for testing

	node := graph.NewNode("email", "kyle@example.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}

	// Check that we got an account node.
	var foundAccount bool
	for _, n := range nodes {
		if n.Type == graph.NodeTypeAccount {
			foundAccount = true
			if n.Confidence < 0.9 {
				t.Errorf("expected high confidence for found profile, got %f", n.Confidence)
			}
		}
	}
	if !foundAccount {
		t.Error("expected an account node")
	}

	if len(edges) == 0 {
		t.Error("expected at least one edge")
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("email", "nobody@example.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for 404, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected no edges for 404, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	profile := map[string]interface{}{
		"displayName": "Test User",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d: %s", status, msg)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cli
go test ./internal/modules/gravatar/ -v
```

Expected: compilation errors.

- [ ] **Step 3: Implement gravatar.go**

Create `cli/internal/modules/gravatar/gravatar.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package gravatar

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const gravatarAPI = "https://en.gravatar.com"

// Module extracts profile data from Gravatar via email MD5 hash.
type Module struct {
	baseURL string
}

// New creates a Gravatar module.
func New() *Module {
	return &Module{baseURL: gravatarAPI}
}

func (m *Module) Name() string        { return "gravatar" }
func (m *Module) Description() string { return "Extract profile data from Gravatar via email hash" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "email" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	email := strings.ToLower(strings.TrimSpace(node.Label))
	hash := fmt.Sprintf("%x", md5.Sum([]byte(email)))
	url := fmt.Sprintf("%s/%s.json", m.baseURL, hash)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("gravatar request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("gravatar returned %d", resp.StatusCode)
	}

	var data struct {
		Entry []struct {
			DisplayName       string `json:"displayName"`
			PreferredUsername  string `json:"preferredUsername"`
			ThumbnailURL      string `json:"thumbnailUrl"`
			URLs              []struct {
				Value string `json:"value"`
			} `json:"urls"`
		} `json:"entry"`
	}

	// The single-profile endpoint returns a flat object, not wrapped in "entry".
	// Try to parse as-is first (for test server), then try the real API format.
	var profile struct {
		DisplayName       string `json:"displayName"`
		PreferredUsername  string `json:"preferredUsername"`
		ThumbnailURL      string `json:"thumbnailUrl"`
		URLs              []struct {
			Value string `json:"value"`
		} `json:"urls"`
	}

	if err := json.Unmarshal([]byte(resp.Body), &data); err == nil && len(data.Entry) > 0 {
		profile = data.Entry[0]
	} else if err := json.Unmarshal([]byte(resp.Body), &profile); err != nil {
		return nil, nil, fmt.Errorf("parsing gravatar response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Account node.
	account := graph.NewAccountNode("gravatar", email, fmt.Sprintf("https://gravatar.com/%s", hash), "gravatar")
	account.Confidence = 0.95
	nodes = append(nodes, account)

	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "gravatar"))

	// Display name.
	if profile.DisplayName != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, profile.DisplayName, "gravatar")
		nameNode.Confidence = 0.85
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "gravatar"))
	}

	// Username.
	if profile.PreferredUsername != "" {
		usernameNode := graph.NewNode(graph.NodeTypeUsername, profile.PreferredUsername, "gravatar")
		usernameNode.Pivot = true
		usernameNode.Confidence = 0.85
		nodes = append(nodes, usernameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, usernameNode.ID, graph.EdgeTypeHasUsername, "gravatar"))
	}

	// Avatar.
	if profile.ThumbnailURL != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, profile.ThumbnailURL, "gravatar")
		avatarNode.Confidence = 0.95
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "gravatar"))
	}

	// Linked URLs (websites/domains).
	for _, u := range profile.URLs {
		if u.Value != "" {
			websiteNode := graph.NewNode(graph.NodeTypeWebsite, u.Value, "gravatar")
			websiteNode.Pivot = true
			websiteNode.Confidence = 0.80
			nodes = append(nodes, websiteNode)
			edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeHasDomain, "gravatar"))
		}
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	// Check a known email hash (WordPress founder's public gravatar).
	hash := fmt.Sprintf("%x", md5.Sum([]byte("test@example.com")))
	url := fmt.Sprintf("%s/%s.json", m.baseURL, hash)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("gravatar: %v", err)
	}
	// 404 is acceptable (means API is up, hash not found).
	if resp.StatusCode == 200 || resp.StatusCode == 404 {
		return modules.Healthy, "gravatar: OK"
	}
	return modules.Degraded, fmt.Sprintf("gravatar: unexpected status %d", resp.StatusCode)
}
```

- [ ] **Step 4: Run tests**

```bash
cd cli
go test ./internal/modules/gravatar/ -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
cd cli
git add internal/modules/gravatar/
git commit -m "add Gravatar module: email -> profile, username, avatar, websites"
```

---

## Task 7: GitHub Module

**Files:**
- Create: `cli/internal/modules/github/github.go`
- Create: `cli/internal/modules/github/github_test.go`

**Depends on:** Tasks 2, 3, 5 (config for optional token)

- [ ] **Step 1: Write tests**

Create `cli/internal/modules/github/github_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

func TestCanHandle(t *testing.T) {
	m := New("")
	if !m.CanHandle("username") {
		t.Error("should handle username")
	}
	if !m.CanHandle("email") {
		t.Error("should handle email")
	}
	if m.CanHandle("domain") {
		t.Error("should not handle domain")
	}
}

func TestExtractUsername(t *testing.T) {
	user := map[string]interface{}{
		"login":      "kylederzweite",
		"name":       "Kyle",
		"email":      "kyle@kylehub.dev",
		"blog":       "https://kylehub.dev",
		"company":    "ACME",
		"location":   "Germany",
		"bio":        "Developer",
		"html_url":   "https://github.com/kylederzweite",
		"avatar_url": "https://avatars.githubusercontent.com/u/123",
		"twitter_username": "kyletweets",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}))
	defer srv.Close()

	m := New("")
	m.baseURL = srv.URL

	node := graph.NewNode("username", "kylederzweite", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) < 3 {
		t.Fatalf("expected at least 3 nodes (account + email + domain), got %d", len(nodes))
	}
	if len(edges) < 3 {
		t.Fatalf("expected at least 3 edges, got %d", len(edges))
	}

	// Verify account node exists with high confidence.
	var foundAccount bool
	for _, n := range nodes {
		if n.Type == graph.NodeTypeAccount {
			foundAccount = true
			if n.Confidence < 0.9 {
				t.Errorf("expected high confidence, got %f", n.Confidence)
			}
		}
	}
	if !foundAccount {
		t.Error("expected account node")
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := New("")
	m.baseURL = srv.URL

	node := graph.NewNode("username", "nonexistent", "seed")
	client := httpclient.New()

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for 404, got %d", len(nodes))
	}
}

func TestExtractEmail(t *testing.T) {
	// GitHub search-by-email returns an items array.
	result := map[string]interface{}{
		"total_count": 1,
		"items": []interface{}{
			map[string]interface{}{
				"login":    "kylederzweite",
				"html_url": "https://github.com/kylederzweite",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	m := New("")
	m.baseURL = srv.URL

	node := graph.NewNode("email", "kyle@example.com", "seed")
	client := httpclient.New()

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	var foundUsername bool
	for _, n := range nodes {
		if n.Type == graph.NodeTypeUsername && n.Label == "kylederzweite" {
			foundUsername = true
			if !n.Pivot {
				t.Error("discovered username should be pivotable")
			}
		}
	}
	if !foundUsername {
		t.Error("expected discovered username node")
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"login": "octocat",
			"name":  "The Octocat",
		})
	}))
	defer srv.Close()

	m := New("")
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d: %s", status, msg)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cli
go test ./internal/modules/github/ -v
```

Expected: compilation errors.

- [ ] **Step 3: Implement github.go**

Create `cli/internal/modules/github/github.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const githubAPI = "https://api.github.com"

// Module extracts profile data from GitHub's REST API.
type Module struct {
	baseURL string
	token   string
}

// New creates a GitHub module. Token is optional (higher rate limits).
func New(token string) *Module {
	return &Module{baseURL: githubAPI, token: token}
}

func (m *Module) Name() string        { return "github" }
func (m *Module) Description() string { return "Extract profile, email, domain, and social links from GitHub" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "username" || nodeType == "email"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	switch node.Type {
	case "username":
		return m.extractByUsername(ctx, node, client)
	case "email":
		return m.extractByEmail(ctx, node, client)
	default:
		return nil, nil, nil
	}
}

func (m *Module) extractByUsername(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	apiURL := fmt.Sprintf("%s/users/%s", m.baseURL, url.PathEscape(username))

	resp, err := client.Do(ctx, apiURL, m.headers())
	if err != nil {
		return nil, nil, fmt.Errorf("github user request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("github returned %d", resp.StatusCode)
	}

	var user struct {
		Login           string `json:"login"`
		Name            string `json:"name"`
		Email           string `json:"email"`
		Blog            string `json:"blog"`
		Company         string `json:"company"`
		Location        string `json:"location"`
		Bio             string `json:"bio"`
		HTMLURL         string `json:"html_url"`
		AvatarURL       string `json:"avatar_url"`
		TwitterUsername string `json:"twitter_username"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &user); err != nil {
		return nil, nil, fmt.Errorf("parsing github response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Account node.
	account := graph.NewAccountNode("github", user.Login, user.HTMLURL, "github")
	account.Confidence = 0.95
	account.Properties["company"] = user.Company
	account.Properties["location"] = user.Location
	account.Properties["bio"] = user.Bio
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "github"))

	// Full name.
	if user.Name != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, user.Name, "github")
		nameNode.Confidence = 0.90
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "github"))
	}

	// Email (pivotable).
	if user.Email != "" {
		emailNode := graph.NewNode(graph.NodeTypeEmail, user.Email, "github")
		emailNode.Pivot = true
		emailNode.Confidence = 0.90
		nodes = append(nodes, emailNode)
		edges = append(edges, graph.NewEdge(0, account.ID, emailNode.ID, graph.EdgeTypeHasEmail, "github"))
	}

	// Blog/website (pivotable as domain).
	if user.Blog != "" {
		blog := user.Blog
		if !strings.HasPrefix(blog, "http") {
			blog = "https://" + blog
		}
		websiteNode := graph.NewNode(graph.NodeTypeWebsite, blog, "github")
		websiteNode.Pivot = true
		websiteNode.Confidence = 0.85
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeHasDomain, "github"))

		// Also extract the domain for WHOIS/DNS modules.
		if parsed, err := url.Parse(blog); err == nil && parsed.Host != "" {
			domainNode := graph.NewNode(graph.NodeTypeDomain, parsed.Host, "github")
			domainNode.Pivot = true
			domainNode.Confidence = 0.85
			nodes = append(nodes, domainNode)
			edges = append(edges, graph.NewEdge(0, account.ID, domainNode.ID, graph.EdgeTypeHasDomain, "github"))
		}
	}

	// Avatar.
	if user.AvatarURL != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, user.AvatarURL, "github")
		avatarNode.Confidence = 0.95
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "github"))
	}

	// Twitter username (pivotable).
	if user.TwitterUsername != "" {
		twitterNode := graph.NewNode(graph.NodeTypeUsername, user.TwitterUsername, "github")
		twitterNode.Pivot = true
		twitterNode.Confidence = 0.80
		twitterNode.Properties["platform_hint"] = "twitter"
		nodes = append(nodes, twitterNode)
		edges = append(edges, graph.NewEdge(0, account.ID, twitterNode.ID, graph.EdgeTypeHasUsername, "github"))
	}

	return nodes, edges, nil
}

func (m *Module) extractByEmail(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	email := node.Label
	apiURL := fmt.Sprintf("%s/search/users?q=%s+in:email", m.baseURL, url.QueryEscape(email))

	resp, err := client.Do(ctx, apiURL, m.headers())
	if err != nil {
		return nil, nil, fmt.Errorf("github email search: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("github search returned %d", resp.StatusCode)
	}

	var result struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Login   string `json:"login"`
			HTMLURL string `json:"html_url"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		return nil, nil, fmt.Errorf("parsing github search: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	for _, item := range result.Items {
		usernameNode := graph.NewNode(graph.NodeTypeUsername, item.Login, "github")
		usernameNode.Pivot = true
		usernameNode.Confidence = 0.85
		nodes = append(nodes, usernameNode)
		edges = append(edges, graph.NewEdge(0, node.ID, usernameNode.ID, graph.EdgeTypeHasUsername, "github"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/users/octocat", m.baseURL)
	resp, err := client.Do(ctx, apiURL, m.headers())
	if err != nil {
		return modules.Offline, fmt.Sprintf("github: %v", err)
	}
	if resp.StatusCode == 200 {
		var user struct {
			Login string `json:"login"`
		}
		if err := json.Unmarshal([]byte(resp.Body), &user); err == nil && user.Login == "octocat" {
			return modules.Healthy, "github: OK"
		}
		return modules.Degraded, "github: unexpected response format"
	}
	if resp.StatusCode == 403 {
		return modules.Degraded, "github: rate limited (consider setting GITHUB_TOKEN)"
	}
	return modules.Offline, fmt.Sprintf("github: status %d", resp.StatusCode)
}

func (m *Module) headers() map[string]string {
	h := map[string]string{
		"Accept": "application/vnd.github+json",
	}
	if m.token != "" {
		h["Authorization"] = "Bearer " + m.token
	}
	return h
}
```

- [ ] **Step 4: Run tests**

```bash
cd cli
go test ./internal/modules/github/ -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
cd cli
git add internal/modules/github/
git commit -m "add GitHub module: username/email -> profile, email, domain, socials"
```

---

## Task 8: Link-in-Bio Modules (Linktree, Beacons, Carrd, Bento)

**Files:**
- Create: `cli/internal/modules/linktree/linktree.go`
- Create: `cli/internal/modules/linktree/linktree_test.go`
- Create: `cli/internal/modules/beacons/beacons.go`
- Create: `cli/internal/modules/beacons/beacons_test.go`
- Create: `cli/internal/modules/carrd/carrd.go`
- Create: `cli/internal/modules/carrd/carrd_test.go`
- Create: `cli/internal/modules/bento/bento.go`
- Create: `cli/internal/modules/bento/bento_test.go`

**Depends on:** Tasks 2, 3

All four follow the same pattern: scrape a page, parse links from the DOM. Each module is a self-contained file. Each gets its own test with a mock HTTP server serving sample HTML.

Due to plan length constraints, I will show the Linktree module in full. The other three (Beacons, Carrd, Bento) follow the identical pattern with only the URL template and DOM selectors changing. Agents implementing these MUST:

1. Use `httptest.NewServer` in tests serving sample HTML.
2. Override `baseURL` for testing (same pattern as Gravatar/GitHub).
3. Parse links from the response HTML using `goquery` (already a dependency).
4. Return `website` nodes with `pivot: false` (link-in-bio links are mentions, not owned assets).
5. Return an `account` node for the profile itself.

- [ ] **Step 1: Write Linktree test**

Create `cli/internal/modules/linktree/linktree_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package linktree

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const sampleHTML = `<!DOCTYPE html>
<html>
<head><title>@testuser | Linktree</title></head>
<body>
<div id="profile-title">Test User</div>
<a href="https://github.com/testuser" data-testid="LinkButton">GitHub</a>
<a href="https://twitter.com/testuser" data-testid="LinkButton">Twitter</a>
<a href="https://testuser.com" data-testid="LinkButton">Website</a>
</body>
</html>`

func TestCanHandle(t *testing.T) {
	m := New()
	if !m.CanHandle("username") {
		t.Error("should handle username")
	}
	if m.CanHandle("email") {
		t.Error("should not handle email")
	}
}

func TestExtractFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(sampleHTML))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL + "/"

	node := graph.NewNode("username", "testuser", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) < 2 {
		t.Fatalf("expected at least 2 nodes (account + links), got %d", len(nodes))
	}
	if len(edges) < 2 {
		t.Fatalf("expected at least 2 edges, got %d", len(edges))
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL + "/"

	node := graph.NewNode("username", "nonexistent", "seed")
	client := httpclient.New()

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for 404, got %d", len(nodes))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body>OK</body></html>"))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL + "/"

	client := httpclient.New()
	status, _ := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d", status)
	}
}
```

- [ ] **Step 2: Implement linktree.go**

Create `cli/internal/modules/linktree/linktree.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package linktree

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const linktreeURL = "https://linktr.ee/"

// Module scrapes Linktree profile pages for linked accounts.
type Module struct {
	baseURL string
}

func New() *Module { return &Module{baseURL: linktreeURL} }

func (m *Module) Name() string        { return "linktree" }
func (m *Module) Description() string { return "Scrape Linktree profiles for linked social accounts and websites" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := m.baseURL + username

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("linktree request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("linktree returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing linktree HTML: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Account node.
	account := graph.NewAccountNode("linktree", username, url, "linktree")
	account.Confidence = 0.90
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "linktree"))

	// Extract links.
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" || href == "#" {
			return
		}
		if !strings.HasPrefix(href, "http") {
			return
		}
		// Skip linktree's own links.
		if strings.Contains(href, "linktr.ee") {
			return
		}

		websiteNode := graph.NewNode(graph.NodeTypeWebsite, href, "linktree")
		websiteNode.Confidence = 0.75
		websiteNode.Pivot = false // link-in-bio links are mentions, not owned assets
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeMentions, "linktree"))
	})

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	resp, err := client.Do(ctx, m.baseURL+"linktree", nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("linktree: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "linktree: OK"
	}
	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return modules.Degraded, fmt.Sprintf("linktree: %d (may be rate limited)", resp.StatusCode)
	}
	return modules.Offline, fmt.Sprintf("linktree: status %d", resp.StatusCode)
}
```

- [ ] **Step 3: Run Linktree tests**

```bash
cd cli
go test ./internal/modules/linktree/ -v
```

Expected: all tests pass.

- [ ] **Step 4: Implement Beacons, Carrd, Bento modules**

Each follows the same pattern as Linktree. The differences per module:

**Beacons** (`cli/internal/modules/beacons/beacons.go`):
- Base URL: `https://beacons.ai/`
- Verify entity: `beacons.ai/linkinbio`
- Link selector: `a[href]` (filter out beacons.ai internal links)

**Carrd** (`cli/internal/modules/carrd/carrd.go`):
- Base URL: `https://{username}.carrd.co`
- Verify entity: `carrd.co` (main page, check for 200)
- Link selector: `a[href]` (filter out carrd.co internal links)
- Note: Carrd uses subdomains, so URL construction is `username + ".carrd.co"`

**Bento** (`cli/internal/modules/bento/bento.go`):
- Base URL: `https://bento.me/`
- Verify entity: `bento.me/bento`
- Link selector: `a[href]` (filter out bento.me internal links)

Each module MUST have:
- A test file with `TestCanHandle`, `TestExtractFound`, `TestExtractNotFound`, `TestVerifyHealthy`
- The same `baseURL` override pattern for test HTTP servers
- Account node + website nodes from extracted links

- [ ] **Step 5: Run all link-in-bio module tests**

```bash
cd cli
go test ./internal/modules/linktree/ ./internal/modules/beacons/ ./internal/modules/carrd/ ./internal/modules/bento/ -v
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
cd cli
git add internal/modules/linktree/ internal/modules/beacons/ internal/modules/carrd/ internal/modules/bento/
git commit -m "add link-in-bio modules: Linktree, Beacons, Carrd, Bento

Scrape profile pages for linked accounts and websites.
Each module uses goquery for HTML parsing with httptest-based tests."
```

---

## Task 9: Social Media Modules (Reddit, YouTube, Twitch, Discord, Instagram, TikTok)

**Files:**
- Create: `cli/internal/modules/reddit/reddit.go` + test
- Create: `cli/internal/modules/youtube/youtube.go` + test
- Create: `cli/internal/modules/twitch/twitch.go` + test
- Create: `cli/internal/modules/discord/discord.go` + test
- Create: `cli/internal/modules/instagram/instagram.go` + test
- Create: `cli/internal/modules/tiktok/tiktok.go` + test

**Depends on:** Tasks 2, 3

Each module follows the same structural pattern established in Tasks 6-8. Key implementation details per module:

**Reddit** (`reddit.go`):
- URL: `https://www.reddit.com/user/{username}/about.json`
- Headers: `User-Agent` must be set (Reddit blocks default UA)
- Parse JSON: extract `data.name`, `data.created_utc`, `data.subreddit.public_description`
- Verify entity: `spez`
- Returns: account node + metadata (account age, karma)

**YouTube** (`youtube.go`):
- URL: `https://www.youtube.com/@{username}` (scrape) or channel page
- Parse HTML with goquery: extract channel name, description, links from "About" section
- Verify entity: `@YouTube`
- Returns: account node + linked website nodes

**Twitch** (`twitch.go`):
- URL: `https://www.twitch.tv/{username}`
- Scrape profile page for bio and panel links
- Verify entity: `twitch` (the official Twitch account)
- Returns: account node + linked website nodes from bio/panels

**Discord** (`discord.go`):
- Uses the registration check endpoint to determine if a username is taken
- URL: `https://discord.com/api/v9/unique-username/validate` (POST with `{"username": "value"}`)
- If taken: returns account node with existence confirmation
- If available: returns nothing
- Verify: check that the endpoint responds
- Returns: account node (existence only, no metadata)

**Instagram** (`instagram.go`):
- URL: `https://www.instagram.com/{username}/`
- Parse for `og:title`, `og:description`, `og:image` meta tags
- Verify entity: `instagram`
- Returns: account node + full name + avatar

**TikTok** (`tiktok.go`):
- URL: `https://www.tiktok.com/@{username}`
- Parse for `og:title`, `og:description`, `og:image` meta tags
- Verify entity: `tiktok`
- Returns: account node + full name + avatar

Each module MUST have:
- Test file with mock HTTP server
- `TestCanHandle`, `TestExtractFound`, `TestExtractNotFound`, `TestVerifyHealthy`
- `baseURL` override pattern for testing

- [ ] **Step 1: Implement and test Reddit module**

Follow the pattern: write test first, verify it fails, implement, verify it passes.

- [ ] **Step 2: Implement and test YouTube module**

- [ ] **Step 3: Implement and test Twitch module**

- [ ] **Step 4: Implement and test Discord module**

Note: Discord uses POST for the registration check. Use `client.DoRequest(ctx, "POST", url, body, headers)`.

- [ ] **Step 5: Implement and test Instagram module**

- [ ] **Step 6: Implement and test TikTok module**

- [ ] **Step 7: Run all social media module tests**

```bash
cd cli
go test ./internal/modules/reddit/ ./internal/modules/youtube/ ./internal/modules/twitch/ ./internal/modules/discord/ ./internal/modules/instagram/ ./internal/modules/tiktok/ -v
```

- [ ] **Step 8: Commit**

```bash
cd cli
git add internal/modules/reddit/ internal/modules/youtube/ internal/modules/twitch/ internal/modules/discord/ internal/modules/instagram/ internal/modules/tiktok/
git commit -m "add social media modules: Reddit, YouTube, Twitch, Discord, Instagram, TikTok"
```

---

## Task 10: Developer/Tech Modules (GitLab, StackExchange) + Communication (Matrix)

**Files:**
- Create: `cli/internal/modules/gitlab/gitlab.go` + test
- Create: `cli/internal/modules/stackexchange/stackexchange.go` + test
- Create: `cli/internal/modules/matrix/matrix.go` + test

**Depends on:** Tasks 2, 3

**GitLab** (`gitlab.go`):
- URL: `https://gitlab.com/api/v4/users?username={username}`
- Returns JSON array. If non-empty, user exists.
- Extract: name, email, website, avatar, bio, social links
- Verify entity: `root` (GitLab's default admin account)

**StackExchange** (`stackexchange.go`):
- URL: `https://api.stackexchange.com/2.3/users?inname={username}&site=stackoverflow`
- Returns JSON with `items` array
- Extract: display_name, website_url, location, link (profile URL)
- Verify entity: search for `jonskeet` (famous SO user)

**Matrix** (`matrix.go`):
- Matrix federation API: `https://matrix.org/_matrix/client/v3/profile/@{username}:matrix.org`
- Returns JSON with `displayname` and `avatar_url`
- Verify entity: `@alice:matrix.org` (test account) or check API responds with 404 (API is up)
- Note: Matrix usernames include a homeserver. For v1, default to matrix.org. If the username contains `:`, parse the homeserver from it.

- [ ] **Step 1: Implement and test GitLab module**

- [ ] **Step 2: Implement and test StackExchange module**

- [ ] **Step 3: Implement and test Matrix module**

- [ ] **Step 4: Run tests**

```bash
cd cli
go test ./internal/modules/gitlab/ ./internal/modules/stackexchange/ ./internal/modules/matrix/ -v
```

- [ ] **Step 5: Commit**

```bash
cd cli
git add internal/modules/gitlab/ internal/modules/stackexchange/ internal/modules/matrix/
git commit -m "add GitLab, StackExchange, and Matrix modules"
```

---

## Task 11: Steam Module (API Key Required)

**Files:**
- Create: `cli/internal/modules/steam/steam.go`
- Create: `cli/internal/modules/steam/steam_test.go`

**Depends on:** Tasks 2, 3, 5 (config for API key)

- [ ] **Step 1: Write tests**

Create `cli/internal/modules/steam/steam_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package steam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

func TestCanHandle(t *testing.T) {
	m := New("testkey")
	if !m.CanHandle("username") {
		t.Error("should handle username")
	}
}

func TestVerifyOfflineWithoutKey(t *testing.T) {
	m := New("")
	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Offline {
		t.Errorf("expected Offline without API key, got %d: %s", status, msg)
	}
}

func TestExtractFound(t *testing.T) {
	resolveResp := map[string]interface{}{
		"response": map[string]interface{}{
			"success": 1,
			"steamid": "76561198000000000",
		},
	}
	summaryResp := map[string]interface{}{
		"response": map[string]interface{}{
			"players": []interface{}{
				map[string]interface{}{
					"steamid":         "76561198000000000",
					"personaname":     "TestPlayer",
					"realname":        "Kyle Test",
					"profileurl":      "https://steamcommunity.com/id/testplayer/",
					"avatarfull":      "https://avatars.steamstatic.com/test.jpg",
					"loccountrycode":  "DE",
				},
			},
		},
	}

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if callCount == 0 {
			json.NewEncoder(w).Encode(resolveResp)
		} else {
			json.NewEncoder(w).Encode(summaryResp)
		}
		callCount++
	}))
	defer srv.Close()

	m := New("testkey")
	m.baseURL = srv.URL

	node := graph.NewNode("username", "testplayer", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}
	if len(edges) == 0 {
		t.Fatal("expected at least one edge")
	}
}

func TestExtractNotFound(t *testing.T) {
	resolveResp := map[string]interface{}{
		"response": map[string]interface{}{
			"success": 42,
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resolveResp)
	}))
	defer srv.Close()

	m := New("testkey")
	m.baseURL = srv.URL

	node := graph.NewNode("username", "nonexistent_user_xyz", "seed")
	client := httpclient.New()

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes, got %d", len(nodes))
	}
}
```

- [ ] **Step 2: Implement steam.go**

Create `cli/internal/modules/steam/steam.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const steamAPI = "https://api.steampowered.com"

// Module extracts profile data from Steam via API key.
type Module struct {
	baseURL string
	apiKey  string
}

func New(apiKey string) *Module {
	return &Module{baseURL: steamAPI, apiKey: apiKey}
}

func (m *Module) Name() string        { return "steam" }
func (m *Module) Description() string { return "Extract profile, aliases, and friends from Steam (requires API key)" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	// Step 1: Resolve vanity URL to SteamID.
	resolveURL := fmt.Sprintf("%s/ISteamUser/ResolveVanityURL/v1/?key=%s&vanityurl=%s",
		m.baseURL, url.QueryEscape(m.apiKey), url.QueryEscape(node.Label))

	resp, err := client.Do(ctx, resolveURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("steam resolve: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("steam resolve returned %d", resp.StatusCode)
	}

	var resolve struct {
		Response struct {
			Success int    `json:"success"`
			SteamID string `json:"steamid"`
		} `json:"response"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &resolve); err != nil {
		return nil, nil, fmt.Errorf("parsing steam resolve: %w", err)
	}
	if resolve.Response.Success != 1 {
		return nil, nil, nil // vanity URL not found
	}

	// Step 2: Get player summary.
	summaryURL := fmt.Sprintf("%s/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s",
		m.baseURL, url.QueryEscape(m.apiKey), url.QueryEscape(resolve.Response.SteamID))

	resp, err = client.Do(ctx, summaryURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("steam summary: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("steam summary returned %d", resp.StatusCode)
	}

	var summary struct {
		Response struct {
			Players []struct {
				SteamID        string `json:"steamid"`
				PersonaName    string `json:"personaname"`
				RealName       string `json:"realname"`
				ProfileURL     string `json:"profileurl"`
				AvatarFull     string `json:"avatarfull"`
				LocCountryCode string `json:"loccountrycode"`
			} `json:"players"`
		} `json:"response"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &summary); err != nil {
		return nil, nil, fmt.Errorf("parsing steam summary: %w", err)
	}
	if len(summary.Response.Players) == 0 {
		return nil, nil, nil
	}

	player := summary.Response.Players[0]
	var nodes []*graph.Node
	var edges []*graph.Edge

	// Account node.
	account := graph.NewAccountNode("steam", node.Label, player.ProfileURL, "steam")
	account.Confidence = 0.95
	account.Properties["steamid"] = player.SteamID
	account.Properties["persona_name"] = player.PersonaName
	account.Properties["country"] = player.LocCountryCode
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "steam"))

	// Real name.
	if player.RealName != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, player.RealName, "steam")
		nameNode.Confidence = 0.80
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "steam"))
	}

	// Avatar.
	if player.AvatarFull != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, player.AvatarFull, "steam")
		avatarNode.Confidence = 0.95
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "steam"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	if m.apiKey == "" {
		return modules.Offline, "steam: no API key configured (set STEAM_API_KEY)"
	}

	resolveURL := fmt.Sprintf("%s/ISteamUser/ResolveVanityURL/v1/?key=%s&vanityurl=valve",
		m.baseURL, url.QueryEscape(m.apiKey))

	resp, err := client.Do(ctx, resolveURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("steam: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "steam: OK"
	}
	if resp.StatusCode == 403 {
		return modules.Offline, "steam: invalid API key"
	}
	return modules.Degraded, fmt.Sprintf("steam: unexpected status %d", resp.StatusCode)
}
```

- [ ] **Step 3: Run tests**

```bash
cd cli
go test ./internal/modules/steam/ -v
```

- [ ] **Step 4: Commit**

```bash
cd cli
git add internal/modules/steam/
git commit -m "add Steam module: username -> profile, real name, country (requires API key)"
```

---

## Task 12: Domain Recon Modules (WHOIS/RDAP, DNS/CT)

**Files:**
- Create: `cli/internal/modules/whois/whois.go` + test
- Create: `cli/internal/modules/dnsct/dnsct.go` + test

**Depends on:** Tasks 2, 3

**WHOIS/RDAP** (`whois.go`):
- Use RDAP (REST-based WHOIS replacement): `https://rdap.org/domain/{domain}`
- Parse JSON response for registrant name, email, organization, registration dates
- CanHandle: `"domain"`
- Verify entity: `example.com` (always exists in RDAP)
- Nodes: full_name, email, organization from registrant data
- Note: many domains have redacted WHOIS due to GDPR. Handle gracefully (return account node with whatever data is available).

**DNS/CT** (`dnsct.go`):
- DNS: use Go's `net.LookupHost`, `net.LookupMX`, `net.LookupTXT` for A/MX/TXT records
- Certificate Transparency: `https://crt.sh/?q={domain}&output=json` for subdomains
- CanHandle: `"domain"`
- Verify entity: `example.com` (always resolves)
- Nodes: ip nodes from A records, domain nodes from CT subdomains
- CT subdomains should have `pivot: false` (they're informational, not owned identities)

- [ ] **Step 1: Implement and test WHOIS/RDAP module**

- [ ] **Step 2: Implement and test DNS/CT module**

- [ ] **Step 3: Run tests**

```bash
cd cli
go test ./internal/modules/whois/ ./internal/modules/dnsct/ -v
```

- [ ] **Step 4: Commit**

```bash
cd cli
git add internal/modules/whois/ internal/modules/dnsct/
git commit -m "add domain recon modules: WHOIS/RDAP and DNS/CT

RDAP for registrant data, net.Lookup* for DNS records,
crt.sh for certificate transparency subdomain discovery."
```

---

## Task 13: Output Package (Table, JSON, CSV)

**Files:**
- Create: `cli/internal/output/table.go`
- Create: `cli/internal/output/json.go`
- Create: `cli/internal/output/csv.go`
- Create: `cli/internal/output/table_test.go`
- Create: `cli/internal/output/json_test.go`
- Create: `cli/internal/output/csv_test.go`

**Depends on:** Task 2 (graph model)

- [ ] **Step 1: Write table test**

Create `cli/internal/output/table_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kyle/basalt/internal/graph"
)

func testGraph() *graph.Graph {
	g := graph.New()

	account := graph.NewAccountNode("github", "kyle", "https://github.com/kyle", "github")
	account.Confidence = 0.95
	account.Wave = 1
	g.AddNode(account)

	email := graph.NewNode(graph.NodeTypeEmail, "kyle@example.com", "github")
	email.Confidence = 0.90
	email.Wave = 1
	g.AddNode(email)

	domain := graph.NewNode(graph.NodeTypeDomain, "kylehub.dev", "github")
	domain.Confidence = 0.85
	domain.Wave = 2
	g.AddNode(domain)

	return g
}

func TestWriteTable(t *testing.T) {
	g := testGraph()
	g.Meta.DurationSecs = 1.5

	var buf bytes.Buffer
	if err := WriteTable(&buf, g); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "github") {
		t.Error("table should contain 'github'")
	}
	if !strings.Contains(out, "kyle@example.com") {
		t.Error("table should contain email")
	}
	if !strings.Contains(out, "0.95") {
		t.Error("table should contain confidence score")
	}
}

func TestWriteTableEmpty(t *testing.T) {
	g := graph.New()
	var buf bytes.Buffer
	if err := WriteTable(&buf, g); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No results") {
		t.Error("empty graph should print 'No results'")
	}
}
```

- [ ] **Step 2: Write JSON test**

Create `cli/internal/output/json_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	g := testGraph()
	var buf bytes.Buffer
	if err := WriteJSON(&buf, g); err != nil {
		t.Fatal(err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := out["nodes"]; !ok {
		t.Error("JSON missing 'nodes'")
	}
	if _, ok := out["edges"]; !ok {
		t.Error("JSON missing 'edges'")
	}
}
```

- [ ] **Step 3: Write CSV test**

Create `cli/internal/output/csv_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"bytes"
	"encoding/csv"
	"testing"
)

func TestWriteCSV(t *testing.T) {
	g := testGraph()
	var buf bytes.Buffer
	if err := WriteCSV(&buf, g); err != nil {
		t.Fatal(err)
	}

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	// Header + 3 data rows (account, email, domain).
	if len(records) < 4 {
		t.Errorf("expected at least 4 rows (header + 3 nodes), got %d", len(records))
	}

	// Check header.
	header := records[0]
	expected := []string{"id", "type", "label", "source_module", "confidence", "wave"}
	for i, col := range expected {
		if i >= len(header) || header[i] != col {
			t.Errorf("header[%d] = %q, want %q", i, header[i], col)
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

```bash
cd cli
go test ./internal/output/ -v
```

- [ ] **Step 5: Implement table.go**

Create `cli/internal/output/table.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"

	"github.com/kyle/basalt/internal/graph"
)

var (
	green = color.New(color.FgGreen, color.Bold)
	yellow = color.New(color.FgYellow)
	dim    = color.New(color.Faint)
	bold   = color.New(color.Bold)
)

// WriteTable writes a color-coded table of all non-seed nodes to the writer.
func WriteTable(w io.Writer, g *graph.Graph) error {
	nodes, _ := g.Collect()

	// Filter out seed nodes.
	var display []*graph.Node
	for _, n := range nodes {
		if n.Type != graph.NodeTypeSeed {
			display = append(display, n)
		}
	}

	if len(display) == 0 {
		fmt.Fprintln(w, "No results found.")
		return nil
	}

	// Sort by confidence descending.
	sort.Slice(display, func(i, j int) bool {
		return display[i].Confidence > display[j].Confidence
	})

	// Header.
	fmt.Fprintln(w)
	bold.Fprintf(w, " %-20s  %-12s  %-40s  %-10s  %s\n",
		"PLATFORM", "TYPE", "VALUE", "CONFIDENCE", "SOURCE")
	fmt.Fprintln(w, " "+strings.Repeat("-", 100))

	// Rows.
	for _, n := range display {
		platform := n.SourceModule
		if siteName, ok := n.Properties["site_name"].(string); ok {
			platform = siteName
		}

		value := n.Label
		if profileURL, ok := n.Properties["profile_url"].(string); ok && profileURL != "" {
			value = profileURL
		}

		// Truncate long values.
		if len(value) > 40 {
			value = value[:37] + "..."
		}

		// Color-code confidence.
		confStr := fmt.Sprintf("%.2f", n.Confidence)
		var confColor *color.Color
		switch {
		case n.Confidence >= 0.80:
			confColor = green
		case n.Confidence >= 0.50:
			confColor = yellow
		default:
			confColor = dim
		}

		fmt.Fprintf(w, " %-20s  %-12s  %-40s  ", platform, n.Type, value)
		confColor.Fprintf(w, "%-10s", confStr)
		fmt.Fprintf(w, "  %s\n", n.SourceModule)
	}

	// Summary.
	fmt.Fprintln(w, " "+strings.Repeat("-", 100))
	g.SnapshotStats()
	bold.Fprintf(w, " %d results found", len(display))
	fmt.Fprintf(w, " (%d modules, %.1fs)\n\n", g.Meta.Stats.ModulesRun, g.Meta.DurationSecs)

	return nil
}
```

- [ ] **Step 6: Implement json.go**

Create `cli/internal/output/json.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/kyle/basalt/internal/graph"
)

// WriteJSON writes the full graph as indented JSON.
func WriteJSON(w io.Writer, g *graph.Graph) error {
	data, err := g.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshaling graph: %w", err)
	}

	var raw json.RawMessage = data
	indented, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("indenting JSON: %w", err)
	}

	_, err = w.Write(indented)
	if err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	_, err = w.Write([]byte("\n"))
	return err
}
```

- [ ] **Step 7: Implement csv.go**

Create `cli/internal/output/csv.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/kyle/basalt/internal/graph"
)

// WriteCSV writes a flat CSV of all nodes (one row per node).
func WriteCSV(w io.Writer, g *graph.Graph) error {
	nodes, _ := g.Collect()

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header.
	if err := writer.Write([]string{"id", "type", "label", "source_module", "confidence", "wave"}); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for _, n := range nodes {
		row := []string{
			n.ID,
			n.Type,
			n.Label,
			n.SourceModule,
			fmt.Sprintf("%.2f", n.Confidence),
			fmt.Sprintf("%d", n.Wave),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	return nil
}
```

- [ ] **Step 8: Run tests**

```bash
cd cli
go test ./internal/output/ -v
```

- [ ] **Step 9: Commit**

```bash
cd cli
git add internal/output/
git commit -m "add output package: table (color-coded), JSON graph, CSV flat export"
```

---

## Task 14: Scan Command (Wire Everything Together)

**Files:**
- Create: `cli/cmd/scan.go`
- Modify: `cli/cmd/root.go` (no global flags needed, scan has its own)

**Depends on:** Tasks 2-5, 6-12 (all modules), 13 (output)

- [ ] **Step 1: Implement scan.go**

Create `cli/cmd/scan.go`:

```go
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
		MaxPivotDepth:  flagDepth,
		Concurrency:    flagConcurrency,
		TimeoutSecs:    flagTimeout,
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
```

- [ ] **Step 2: Add google/uuid dependency**

```bash
cd cli
go get github.com/google/uuid
```

- [ ] **Step 3: Build and vet**

```bash
cd cli
go build ./...
go vet ./...
```

- [ ] **Step 4: Smoke test**

```bash
cd cli
./basalt version
./basalt scan --help
```

Expected: version prints `basalt v2.0.0-dev`, scan shows all flags.

- [ ] **Step 5: Commit**

```bash
cd cli
git add cmd/scan.go go.mod go.sum
git commit -m "add scan command wiring walker, modules, config, and output

Registers all 18 modules, loads API keys from config, runs the
async walker, outputs table to stdout with optional JSON/CSV export."
```

---

## Task 15: Documentation (README.md, AGENTS.md)

**Files:**
- Rewrite: `README.md`
- Rewrite: `AGENTS.md`

**Depends on:** Task 14 (final CLI interface is locked)

- [ ] **Step 1: Rewrite README.md**

Write a concise, example-driven README covering:
- What basalt is (relation-based OSINT, one paragraph)
- Installation (`go install`)
- Quick start (3 example commands)
- Module list (table: name, seed type, what it extracts)
- Configuration (API keys in `~/.basalt/config`)
- Export formats (`--export json`, `--export csv`)
- How it works (brief: reactive graph walker, modules, pivoting)
- License (AGPLv3)

No walls of text. Each section should be scannable.

- [ ] **Step 2: Rewrite AGENTS.md**

Update for the new architecture:
- Orientation: module-based architecture, walker is the core loop
- Build commands: same (`go build ./...`, `go vet ./...`, `go test ./...`)
- Central contract: `modules.Module` interface (not engine)
- One package per module convention
- How to add a module: implement interface, register in `cmd/scan.go`, add verify entity
- Conventions: keep existing style rules, add module-specific ones
- Things to avoid: no YAML site definitions, no global confidence formula, no headless browsers

- [ ] **Step 3: Build, vet, and run full test suite**

```bash
cd cli
go build ./...
go vet ./...
go test ./... -v
```

- [ ] **Step 4: Commit**

```bash
git add README.md AGENTS.md
git commit -m "rewrite README and AGENTS for v2 module-based architecture"
```

---

## Final Verification

After all tasks are complete:

```bash
cd cli
go build -o basalt .
go vet ./...
go test ./... -v
./basalt version       # should print "basalt v2.0.0-dev"
./basalt scan --help   # should show all flags
```

The binary should build, all tests should pass, and the scan command should be fully wired.
