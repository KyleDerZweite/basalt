# Basalt v2 Design Spec

Relation-based OSINT tool that builds intelligence graphs from high-value, purpose-built modules.

## Problem

Existing open-source OSINT tools (Sherlock, Maigret, WhatsMyName) spray 1400+ sites with URL template checks. This produces high false-positive rates, low-confidence results, and no relationship data. They answer "does this username exist on site X?" but not "how are this person's accounts connected?"

Basalt v2 replaces that model entirely. Instead of blind enumeration, it queries a curated set of 18 platforms via their APIs and known endpoints, extracts structured metadata, and follows discovered relationships to build a connected picture.

## Goals

- Zero false positives over broad coverage
- Relation-based output (graph of connected entities, not a flat list)
- High-value extraction (metadata, linked accounts, domains) over existence checking
- EU-focused platform selection (no .ru domains, no low-value sites)
- Self-healing module system that degrades gracefully when endpoints change
- Fast, concurrent execution with no artificial bottlenecks

## Non-Goals

- Replacing Sherlock/Maigret for broad site coverage (that's their job, not ours)
- Investigation/surveillance tooling (this is for researching yourself and friends)
- Headless browser automation (HTTP requests only)
- Real-time monitoring or continuous scanning

## Architecture Overview

```
CLI (Cobra)
  |
  v
Walker (async dispatch loop)
  |
  +-- Module Registry (18 modules)
  |     +-- each module: CanHandle() + Extract() + Verify()
  |
  +-- Graph (thread-safe node/edge store)
  |
  +-- HTTP Client (retries, per-domain rate limiting, proxy, DNS cache)
  |
  v
Output (table to terminal, JSON/CSV to file)
```

The walker is the core orchestrator. It feeds graph nodes to modules that can handle them, merges results back into the graph, and immediately dispatches new pivotable nodes to matching modules. Execution is fully async and event-driven.

## Module Interface

```go
type HealthStatus int

const (
    Healthy  HealthStatus = iota  // results scored normally
    Degraded                       // confidence multiplied by 0.5
    Offline                        // module skipped entirely
)

type Module interface {
    // Name returns a human-readable identifier (e.g., "github").
    Name() string

    // Description returns what this module does (e.g., "Extract profile data from GitHub").
    Description() string

    // CanHandle reports whether this module can process the given node type.
    CanHandle(nodeType string) bool

    // Extract processes a node and returns discovered nodes and edges.
    // Must respect context cancellation.
    Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error)

    // Verify runs a lightweight self-test against a known entity.
    // Called once at startup. Returns status and a human-readable message
    // (e.g., "Steam: no API key configured", "Linktree: unexpected DOM structure").
    Verify(ctx context.Context, client *httpclient.Client) (HealthStatus, string)
}
```

### Key behaviors

- `CanHandle` determines which modules fire for a given node. A module can handle multiple types (e.g., GitHub handles both "username" and "email").
- `Extract` returns nodes tagged with `pivot: true` or `pivot: false`. Only pivotable nodes trigger further module execution.
- `Verify` checks a known-existing entity (e.g., GitHub checks "octocat"). Returns Healthy, Degraded, or Offline.
- Modules requiring API keys receive them via config at construction. Missing key = Verify returns Offline with a descriptive message. The module is skipped silently.

## Graph Model

### Node types

```
username, email, domain, phone, full_name,
account, organization, ip, avatar_url, website
```

The `account` type is platform-agnostic. The platform is encoded in the node ID (`account:github:kyle`) and in the `site_name` property, not in the type. This keeps the type system small and avoids proliferating types like `github_account`, `reddit_account`, etc.

### Node properties

- `source_module` - which module created this node
- `pivot` (bool) - whether the walker should dispatch this node to further modules
- `wave` (int) - discovery depth (0 = initial seed, 1 = discovered from seed, etc.)
- `confidence` (float64) - module's confidence in this discovery

### Edge types

| Type | Meaning | Example |
|------|---------|---------|
| `has_account` | entity owns an account | username -> GitHub account |
| `has_email` | entity has this email | account -> email |
| `has_domain` | entity owns this domain | account -> domain |
| `has_username` | account links to another username | GitHub account -> Reddit username |
| `registered_to` | domain registered to entity | domain -> full_name |
| `resolves_to` | domain resolves to IP | domain -> ip |
| `linked_to` | weak/generic association | any -> any |
| `mentions` | entity references another | contact page -> email |

### Node merging

When two modules discover the same entity (e.g., both Gravatar and GitHub find kyle@example.com), the graph deduplicates by node ID. The second discovery adds a new edge (different source_module) to the existing node. An entity confirmed by multiple modules naturally has more edges, which signals higher confidence.

### Edge policy

Edges are NOT deduplicated. If two modules both confirm the same relationship (e.g., both GitHub and Keybase confirm username -> email), both edges are kept with their respective `source_module` tags. Multiple edges between the same nodes represent converging evidence and are valuable for confidence assessment.

### Concurrency and locking

The graph uses a single `sync.RWMutex` for all mutations. At the target concurrency of 5, lock hold time for `AddNode` (map insert) and `AddEdge` (slice append) is sub-microsecond, making contention negligible. Node-level locking would add complexity without measurable benefit at this scale. If concurrency requirements grow significantly (50+), this can be revisited.

### Changes from current graph package

- Node gains top-level struct fields: `Pivot bool`, `Wave int`, `Confidence float64`, `SourceModule string`. These are promoted out of the Properties map for type safety and direct access by the walker. The Properties map remains for module-specific metadata.
- New node types and edge types added
- NewAccountNode simplified (old one had too many params tied to site-check model)
- Atomic counters and Meta/Stats stay as-is

## Walker (Async Orchestrator)

The walker replaces the old scan command logic, engine registry, and pivot controller.

### State

```go
type Walker struct {
    graph       *graph.Graph
    modules     []Module          // all registered modules
    healthy     []Module          // post-Verify filtering
    maxDepth    int               // default 2
    semaphore   chan struct{}      // concurrency limit, default 5
    inflight    sync.WaitGroup
    processed   sync.Map          // "moduleID:nodeID" dedup
}
```

### Startup sequence

1. Parse seeds from CLI flags, add as nodes at wave 0 with pivot = true
2. Run Verify() on all modules concurrently, classify as healthy/degraded/offline
3. Print health summary to stderr
4. Begin async dispatch

### Dispatch logic

```
dispatch(node):
    for each healthy module where CanHandle(node.Type):
        skip if module+node combo already in processed map
        acquire semaphore slot
        launch goroutine:
            nodes, edges, err = module.Extract(ctx, node, client)
            if err: log, increment error count, continue
            if module is degraded: multiply confidence on returned nodes by 0.5 (mutate node.Confidence in-place before merging into graph)
            tag returned nodes with wave = node.wave + 1
            merge nodes and edges into graph
            for each returned node where pivot == true AND wave < maxDepth:
                dispatch(node)  // immediately trigger matching modules
            release semaphore slot
            inflight.Done()
```

### Key properties

- **Fully async**: no wave barriers. GitHub returns fast? Its discoveries trigger WHOIS immediately while Gravatar is still responding.
- **Module-level timeout**: the walker wraps each `Extract()` call with `context.WithTimeout(ctx, timeout)` where timeout comes from the `--timeout` flag (default 10s). This bounds the entire module execution (HTTP requests + parsing), not just individual network calls. The HTTP client's own timeout serves as an inner bound on individual requests.
- **Dedup at dispatch level**: processed map tracks "module:node" combos to prevent duplicate work. Graph-level dedup handles duplicate data.
- **Termination**: inflight WaitGroup hits zero when all dispatched goroutines complete and no new pivotable nodes were produced.
- **Graceful shutdown**: context cancellation on Ctrl+C. In-flight goroutines finish current request, walker stops dispatching, outputs partial graph.

## Confidence Model

No global confidence formula. Each module assigns its own confidence based on what it found.

Examples:
- GitHub API returns exact username match with full profile -> 0.95
- GitHub API returns username but profile is empty -> 0.70
- Reddit JSON endpoint returns valid profile -> 0.90
- Linktree scraper finds page but only partial data extracted -> 0.50

The only global modifier is the degraded multiplier (0.5) applied to all results from modules whose Verify() returned Degraded.

This replaces the old 5-signal weighted system (HTTP status + presence + absence + content diff + redirect). That system was designed for blind URL checking. The new modules know exactly what they're querying.

## Module Health System

### Startup verification

Each module's Verify() tests a known entity with a single HTTP request:
- GitHub: checks that "octocat" resolves
- Gravatar: checks hash of a known email
- Reddit: checks that "spez" returns valid JSON

### Three states

| State | Condition | Effect |
|-------|-----------|--------|
| Healthy | Verify returned expected data | Results scored normally |
| Degraded | Got response but data was unexpected (missing fields, changed schema) | Confidence * 0.5 |
| Offline | Failed entirely (network error, 403, missing API key) | Module skipped, listed in startup summary |

## CLI Interface

### Commands

```
basalt scan -u <username>           # seed with username
basalt scan -e <email>              # seed with email
basalt scan -d <domain>             # seed with domain
basalt scan -u kyle -e kyle@x.com   # multiple seeds

Flags:
  --depth N          max pivot depth (default 2)
  --concurrency N    max concurrent requests (default 5)
  --config PATH      path to config file for API keys
  --export json      write graph JSON to file
  --export csv       write flat CSV to file
  --timeout N        per-request timeout in seconds (default 10)
  --verbose          show module health, progress, debug info
```

### Default output (terminal table)

```
$ basalt scan -u kylederzweite

 Modules: 16 ready, 1 degraded (Linktree), 1 skipped (Steam: no API key)

 PLATFORM     TYPE        VALUE                    CONFIDENCE  SOURCE
 GitHub       account     github.com/kylederzweite 0.95        username
 GitHub       email       kyle@kylehub.dev         0.90        extracted
 GitHub       domain      kylehub.dev              0.85        extracted
 Gravatar     avatar      gravatar.com/abc123      0.95        email
 kylehub.dev  registrant  Kyle [Redacted]          0.80        whois
 kylehub.dev  dns         104.21.x.x               1.00        a-record
 Reddit       account     reddit.com/u/kylederzw.. 0.90        username
 ...

 Scan complete: 8 accounts, 3 emails, 2 domains found (18 modules, 2.3s)
```

Table sorted by confidence descending. Color-coded: green (>0.8), yellow (0.5-0.8), dim (<0.5). Errors and skipped modules shown in --verbose only.

### Export formats

- `--export json` writes full graph (nodes + edges + meta) to basalt-scan-TIMESTAMP.json
- `--export csv` writes flat table (one row per entity) to basalt-scan-TIMESTAMP.csv
- Both can be combined

### Config file

Located at `~/.basalt/config` or specified via `--config`:

```env
STEAM_API_KEY=abc123
GITHUB_TOKEN=ghp_xxx
```

Modules read their key at construction. Missing key = Verify returns Offline. A future `basalt setup` command will guide users through key provisioning (where to create keys, what permissions to grant). This is a separate feature, not part of v1 core.

## V1 Module List (18 modules)

### Identity Aggregators (5)

| Module | Seed | Auth | Method | Extracts |
|--------|------|------|--------|----------|
| Gravatar | email | none | API (MD5 hash) | username, full name, avatar, linked websites |
| Linktree | username | none | scrape | linked social accounts, websites, contact info |
| Beacons | username | none | scrape | linked social accounts, websites |
| Carrd | username | none | scrape (subdomain) | linked content, contact info |
| Bento | username | none | scrape | linked social accounts, websites |

### Developer/Tech (3)

| Module | Seed | Auth | Method | Extracts |
|--------|------|------|--------|----------|
| GitHub | username, email | optional token | REST API | full name, email, company, location, website, social links, commit emails |
| GitLab | username | none | REST API | full name, email, website, social links |
| StackExchange | username | none | API | websites, location, professional interests |

### Social Media (6)

| Module | Seed | Auth | Method | Extracts |
|--------|------|------|--------|----------|
| Reddit | username | none | JSON endpoint | account age, connected accounts, active subreddits |
| YouTube | username | none | scrape/API | channel info, about page, linked socials |
| Twitch | username | none | API | profile, about panel links |
| Discord | username | none | registration endpoint | existence check (taken/available) |
| Instagram | username | none | scrape | public profile data |
| TikTok | username | none | scrape | public profile data |

### Communication (1)

| Module | Seed | Auth | Method | Extracts |
|--------|------|------|--------|----------|
| Matrix | username | none | federation API | homeserver profile, avatar, display name |

### Gaming (1)

| Module | Seed | Auth | Method | Extracts |
|--------|------|------|--------|----------|
| Steam | username | API key | API | real name, country, aliases, friends list |

### Domain Recon (2)

| Module | Seed | Auth | Method | Extracts |
|--------|------|------|--------|----------|
| WHOIS/RDAP | domain | none | RDAP/WHOIS | registrant name, email, organization, dates |
| DNS/CT | domain | none | DNS + crt.sh API | A/AAAA/MX/TXT records, subdomains |

### Future modules (not v1)

- `discord-authenticated` - user token for friends list, linked accounts
- `basalt setup` - interactive API key provisioning guide

## Codebase Changes

### Keep (refactor as needed)

- `graph/` - expand node types, add Pivot/Wave fields, new edge constructors
- `httpclient/` - client, retries, rate limiting, proxy rotation, DNS cache (unchanged)
- `cmd/root.go` - Cobra CLI skeleton
- `main.go` - entry point

### Delete entirely

- `engine/` - old Seed/Result/Check/SignalScores/confidence system
- `sitedb/` - YAML site definitions, Sherlock/Maigret/WMN importers, loader, validator
- `engines/username/` - old username checker
- `engines/email/` - old email engine and 16 email modules
- `pivot/` - old pivot controller tied to engine.Seed
- `output/` - old table/json formatters tied to old Result type
- `data/sites/` - YAML site definition files
- `resolver/` - old resolver if only used by deleted code

### New packages

- `modules/` - Module interface, HealthStatus, registry
- `modules/github/` - GitHub module implementation
- `modules/gravatar/` - Gravatar module implementation
- (one sub-package per module, 18 total)
- `walker/` - async dispatch loop, health checking, concurrency control
- `output/` - new table formatter and JSON/CSV export writers
- `config/` - config file loading, API key distribution to modules

## Documentation

### README.md (rewrite)

- What basalt is (relation-based OSINT, not a site checker)
- Installation
- Quick start with examples
- Module list with extraction capabilities
- Configuration (API keys, config file)
- Export formats
- Brief "how it works" section
- License (AGPLv3)

Concise, example-driven, scannable. No walls of text.

### AGENTS.md (rewrite)

- Orientation updated for module-based architecture
- Walker is the central orchestrator (not engine)
- Module interface is the central contract
- One package per module convention
- "How to add a module" guide: implement interface, register, add verify entity
- Things to avoid updated: no YAML sites, no global confidence formula, no headless browsers

### Module documentation

No separate doc files per module. Each module's Name() and Description() methods serve as inline docs. Runtime discovery via `--verbose` flag and a future `basalt modules` list command.
