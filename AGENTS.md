# AGENTS.md

Guidelines for AI coding agents working on this codebase.

## Orientation

This is a Go CLI tool. All source is in `cli/`. Read the code, not this file, for implementation details. These docs describe intent and conventions only.

Entry point: `cli/main.go` -> `cli/cmd/root.go` (Cobra CLI).

Module path: `github.com/KyleDerZweite/basalt`.

## Build & Run

```bash
cd cli
go build -o basalt .    # build
go vet ./...            # lint
go test ./...           # all tests
go test ./internal/modules/github/ -v  # single module
```

## Architecture

```
cli/
  cmd/              Cobra commands (root, scan, version)
  internal/
    config/         KEY=VALUE config file loader
    graph/          Node, Edge, Graph (concurrent-safe, JSON serializable)
    httpclient/     HTTP client with retries, rate limiting, proxy rotation
    modules/        Module interface, Registry, HealthStatus
      gravatar/     Email -> profile, username, avatar, websites
      github/       Username/email -> profile, email, domain, socials
      gitlab/       Username -> profile, name, avatar, website
      ...           (18 modules total, one package each)
    output/         Table (color), JSON graph, CSV flat export
    walker/         Reactive async graph walker with health checks
```

### Key interfaces

**Module** (`internal/modules/module.go`):
```go
type Module interface {
    Name() string
    Description() string
    CanHandle(nodeType string) bool
    Extract(ctx, node, client) (nodes, edges, error)
    Verify(ctx, client) (HealthStatus, string)
}
```

**Walker** (`internal/walker/walker.go`): dispatches modules against graph nodes, manages concurrency, dedup, pivot depth, and health states.

**Graph** (`internal/graph/`): concurrent-safe graph with `AddNode` (dedup by ID), `AddEdge` (no dedup, converging evidence), `Collect()`, `MarshalJSON()`.

### Data flow

Seed -> Walker.dispatch -> Module.Extract -> (new nodes, edges) -> Walker.dispatch (recursive) -> Graph.Collect -> Output

### Node types

`seed`, `account`, `email`, `username`, `domain`, `full_name`, `avatar_url`, `website`

### Node fields

`ID`, `Type`, `Label`, `Properties map`, `Confidence float64`, `Wave int`, `Pivot bool`, `SourceModule string`

Edge IDs are assigned by the walker, not by modules (modules pass `0`).

## Adding a New Module

1. Create `internal/modules/yourmod/yourmod.go`
2. Implement `modules.Module` interface
3. Add `internal/modules/yourmod/yourmod_test.go` with httptest mock server
4. Use `m.baseURL` field (unexported) that defaults to the real URL but is overridable in tests
5. Register in `cmd/scan.go`
6. Run `go test ./internal/modules/yourmod/ -v && go build ./... && go vet ./...`

## Principles

1. **KISS (Keep It Simple, Stupid)**
2. **DRY (Don't Repeat Yourself)**
3. **Open/Closed**
4. **Composition Over Inheritance**
5. **YAGNI (You Aren't Going to Need It)**
6. **Single Responsibility**
7. **Document Your Code**
8. **Separation of Concerns**
9. **Refactor**
10. **Clean Code At All Costs**

## Before You Code

1. Read the file you're modifying. Understand existing patterns before changing anything.
2. Run `go build ./...` and `go vet ./...` after changes.
3. The Module interface in `modules/module.go` and the Walker in `walker/walker.go` are the central contracts.

## Conventions

- **License header**: every Go file starts with `// SPDX-License-Identifier: AGPL-3.0-or-later`
- **Options pattern**: constructors take `New(required, ...Option)` where options are `WithX` functions
- **Error handling**: return errors, don't panic
- **Context**: all long-running operations accept `context.Context` and must check cancellation
- **Testing**: every module has tests using `httptest.NewServer` with a `baseURL` override
- **Confidence**: modules assign their own confidence scores (0.0-1.0). Degraded modules get halved by the walker.

## Things to Avoid

- Don't add dependencies without justification
- Don't write docs that restate code
- Don't use em dashes, double hyphens, triple dashes, or generic AI filler in docs, commits, or responses
