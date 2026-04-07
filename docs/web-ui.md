# Basalt Web Workspace

Basalt can serve a local browser workspace directly from the Go binary:

```bash
cd cli
go run . web
```

Default address: `http://127.0.0.1:8788`

To rebuild the embedded frontend assets during development:

```bash
cd web
pnpm install
pnpm build
```

To keep the server running without opening a browser:

```bash
cd cli
go run . web --no-open
```

To let the OS pick a free port:

```bash
cd cli
go run . web --listen 127.0.0.1:0
```

## Runtime

`basalt web` serves:

- `/` for the browser workspace shell
- `/assets/*` for embedded static assets
- `/app/bootstrap` for runtime metadata
- `/api/*` for the existing local API

The web workspace and API run on the same localhost origin, so normal product use does not require CORS or bearer auth bootstrapping.

## Functional Surface

Current browser functionality focuses on operations rather than styling:

- Home:
  - recent scans
  - target list
  - module health summary
  - runtime summary
  - settings summary
- Targets:
  - create persistent targets
  - add and remove aliases
  - review curated identifiers before scanning
- New Scan:
  - choose a target or run ad hoc
  - add/remove username, email, and domain seeds
  - set depth, concurrency, timeout
  - toggle strict mode
  - disable modules by name
- Scan Workspace:
  - top summary cards and findings
  - concentric radial mindmap graph with target/seed at center as the primary view
  - persisted node and edge tables as secondary views
  - live event timeline over SSE
  - evidence/details panel for selected synthesized graph nodes
  - cancel running scan
  - JSON and CSV export actions
- Settings:
  - strict mode default
  - disabled modules
  - legal acceptance timestamp
  - runtime information

## Bootstrap Payload

`GET /app/bootstrap` returns runtime metadata used by the browser client:

```json
{
  "name": "basalt",
  "product": "web",
  "version": "0.3.0",
  "data_dir": "/home/kyle/.basalt",
  "default_config_path": "/home/kyle/.basalt/config",
  "api_base_path": "/api",
  "base_url": "http://127.0.0.1:8788"
}
```

## Technology Stack

| Layer | Choice |
|---|---|
| Framework | React 19, TypeScript 5.9 |
| Build | Vite 8, pnpm |
| Routing | react-router-dom 7 |
| Icons | lucide-react (tree-shakeable SVGs) |
| Graph | cytoscape (built-in concentric layout) |
| Text layout | @chenglou/pretext |
| Styling | Hand-written CSS (no Tailwind, no component library) |

The build outputs to `cli/internal/webui/dist/` and is embedded into the Go binary. The frontend has no server of its own; `basalt web` serves both the SPA and the API on the same localhost origin.

## Design System

All styling is in `web/src/index.css` using CSS custom properties. Key tokens:

- **Colors**: accent `#d99a71`, backgrounds `--bg-base` / `--bg-surface` / `--bg-elevated`, borders `--border-dim` / `--border` / `--border-strong`, text `--text-primary` / `--text-secondary` / `--text-muted`
- **Radii**: 2-6px for structural elements, `999px` for pills/chips/dots only
- **Shadows**: `0 2px 8px` on overlays only (mobile sidebar, panel overlay)
- **Fonts**: IBM Plex Mono (headings), JetBrains Mono (body)
- **Theming**: dark (default) and light modes via `[data-theme="light"]` on `<html>`

The design is intentionally flat and dense for a data-heavy OSINT tool. No gradients, no backdrop-filter, no noise textures.

## Icons

All icons are from `lucide-react`. Imported individually per component:

```tsx
import { Home, ArrowRight, X } from "lucide-react";
```

Standard sizing: `12-14px` for inline buttons, `16px` for navigation, `24px` for empty states.

## Notes

- The web workspace is a thin local client. Scans, persistence, events, and exports still come from the Go backend.
- Scan history and settings remain in the same SQLite database used by `basalt serve` and `basalt scan`.
- Targets and aliases are persisted locally and can be reused across scans.
- The graph uses a concentric radial layout: root at center (depth 0), seeds and category branches in the first ring (depth 1), leaf discoveries in the outer ring (depth 2). Single-seed scans promote the seed to the center root.
- The graph shown in the UI is synthesized from the raw evidence graph so the main view stays readable and target-centric.
- Selecting a node highlights it and its connected edges.
- `basalt serve` remains the headless integration path for custom clients.
