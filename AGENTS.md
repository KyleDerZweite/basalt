# AGENTS.md

Guidelines for AI coding agents working on this codebase.

## Orientation

Two codebases in one repo: a Go CLI (`cli/`) and a React web dashboard (`web/`). The web build output is embedded into the Go binary at `cli/internal/webui/dist/`.

Entry point: `cli/main.go` -> `cli/cmd/root.go` (Cobra CLI).

Module path: `github.com/KyleDerZweite/basalt`.

## Build & Run

```bash
# Go backend
cd cli
go build -o basalt .    # build
go vet ./...            # lint
go test ./...           # all tests
go test ./internal/modules/github/ -v  # single module

# Web frontend
cd web
pnpm install            # install deps
pnpm build              # production build -> cli/internal/webui/dist/
pnpm dev                # dev server with hot reload
pnpm typecheck          # tsc --noEmit
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

## Web Frontend

### Stack

React 19, TypeScript, Vite 8. No CSS framework; all styling is hand-written in `web/src/index.css`. Build output goes to `cli/internal/webui/dist/` and is embedded into the Go binary.

### Key dependencies

- `lucide-react` for icons (tree-shakeable SVGs)
- `cytoscape` for graph visualization (concentric radial layout, no external layout plugins)
- `react-router-dom` for client-side routing
- `@chenglou/pretext` for text layout measurement

### File structure

```
web/src/
  components/       Reusable UI (Sidebar, FindingCard, NodeInspector, ...)
  pages/            Full page views (HomePage, NewScanPage, ScanWorkspacePage, ...)
  hooks/            Custom React hooks (useCytoscapeGraph, useScanEvents, ...)
  lib/              Utilities (api.ts, constants.ts, format.ts, typography.ts)
  types.ts          Shared TypeScript types
  index.css         All styling (design tokens, components, responsive)
  App.tsx            Root component with routing
  main.tsx          React entry point
```

### Design system

All styling lives in `index.css`. No Tailwind, no CSS-in-JS. Key conventions:

- **CSS variables** for all colors, spacing, and radii. Defined in `:root`, overridden in `[data-theme="light"]`.
- **Flat surfaces**: `var(--bg-base)`, `var(--bg-surface)`, `var(--bg-elevated)`. No gradients, no backdrop-filter.
- **Sharp corners**: `--radius-sm: 2px`, `--radius: 4px`, `--radius-lg: 4px`, `--radius-xl: 6px`. Pill shapes (`999px`) only for status pills, chips, dots, and progress bars.
- **Tight shadows**: `--shadow-panel: 0 2px 8px rgba(...)`. Only on overlays (mobile sidebar, panel overlay).
- **Dense padding**: utilitarian, not spacious. Card headers `10px 14px`, card bodies `12px`, page content `20px 24px`.
- **Fonts**: IBM Plex Mono (display/headings), JetBrains Mono (body/code).
- **Accent**: `#d99a71` (warm amber). Status colors: success green, danger red, warning yellow, info blue.

### Icons

All icons use `lucide-react`. Import individual icons:

```tsx
import { Home, ArrowRight, X } from "lucide-react";
<Home size={16} />
```

Standard sizes: `12-14` for inline/button icons, `16` for nav icons, `24` for empty state display icons. Do not use Unicode symbols or emoji as icons.

### Conventions

- No component libraries (shadcn, radix, etc.). Raw HTML elements with CSS classes.
- State management via React hooks only (no Redux, Zustand, etc.).
- API calls go through `lib/api.ts` (thin fetch wrapper hitting the Go backend).
- Theme toggle via `data-theme` attribute on `<html>`, persisted in localStorage.
- Mobile responsiveness via CSS media queries and the `useMediaQuery` hook.

## Things to Avoid

- Don't add dependencies without justification
- Don't write docs that restate code
- Don't use em dashes, double hyphens, triple dashes, or generic AI filler in docs, commits, or responses
