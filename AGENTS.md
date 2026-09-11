# AGENTS.md

Go CLI (`cli/`) + React dashboard (`web/`). The web build output is embedded into the Go binary at `cli/internal/webui/dist/`.

Module path: `github.com/KyleDerZweite/basalt`. Entry point: `cli/main.go` -> `cli/cmd/root.go` (Cobra).

## Build & Test

```bash
# Go (from cli/)
go build -o basalt .
go vet ./...
go test ./...
go test ./internal/modules/github/ -v   # single module

# Web (from web/)
pnpm build       # -> cli/internal/webui/dist/
pnpm dev
pnpm typecheck
```

## Architecture

```
cli/
  cmd/              Cobra commands: scan, target, serve, web, version
  internal/
    api/            HTTP API for the dashboard
    app/            Local product backend (SQLite, scan history, event streams)
    config/         KEY=VALUE config loader
    graph/          Node, Edge, Graph (concurrent-safe, JSON serializable)
    httpclient/     HTTP client with retries, rate limiting, proxy rotation
    modules/        Module interface, Registry, HealthStatus + 38 module packages
    output/         Table, JSON, CSV exporters
    walker/         Async graph walker with health checks
    webui/          Embedded SPA assets + handler
web/src/
  components/ pages/ hooks/ lib/    types.ts    index.css (all styling)
```

Data flow: seed -> `Walker.dispatch` -> `Module.Extract` -> (nodes, edges) -> dispatch recursively -> `Graph.Collect` -> output.

The two central contracts are `Module` (`internal/modules/module.go`) and `Walker` (`internal/walker/walker.go`). Read them before changing anything that touches extraction.

```go
type Module interface {
    Name() string
    Description() string
    CanHandle(nodeType string) bool
    Extract(ctx, node, client) (nodes, edges, error)
    Verify(ctx, client) (HealthStatus, string)
}
```

`Graph` dedups nodes by ID on `AddNode`; `AddEdge` does not dedup (converging evidence is signal). Edge IDs are assigned by the walker, so modules pass `0`.

Node types: `seed`, `account`, `email`, `username`, `domain`, `full_name`, `avatar_url`, `website`, `ip`, `organization`, `phone`.

## Adding a Module

1. Create `internal/modules/yourmod/yourmod.go` implementing `modules.Module`.
2. Give it an unexported `baseURL` field defaulting to the real URL, overridable in tests.
3. Add `yourmod_test.go` using `httptest.NewServer` against that override.
4. Register it in `internal/app/modules.go`.
5. `go test ./internal/modules/yourmod/ -v && go build ./... && go vet ./...`

## Conventions

- Every Go file starts with `// SPDX-License-Identifier: AGPL-3.0-or-later`.
- Constructors use the options pattern: `New(required, ...Option)` with `WithX` functions.
- Return errors, don't panic. Long-running operations take a `context.Context` and check cancellation.
- Modules assign their own confidence scores (0.0-1.0). The walker halves them for degraded modules.
- Web: no CSS framework, no component libraries, no state libraries. Hand-written CSS in `index.css` (CSS variables, flat surfaces, sharp corners, dense padding, accent `#d99a71`). Icons from `lucide-react` only, never emoji. API calls go through `lib/api.ts`.

## Avoid

- Adding dependencies without justification.
- Docs that restate code.
- Em dashes, double hyphens, triple dashes, or generic AI filler in docs, commits, and responses.
