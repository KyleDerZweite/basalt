# Basalt Web Workspace

Basalt can serve a local browser workspace directly from the Go binary:

```bash
cd cli
go run . web
```

Default address: `http://127.0.0.1:8788`

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
  - target-centric mindmap graph as the primary view
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
  "version": "2.0.0-dev",
  "data_dir": "/home/kyle/.basalt",
  "default_config_path": "/home/kyle/.basalt/config",
  "api_base_path": "/api",
  "base_url": "http://127.0.0.1:8788"
}
```

## Notes

- The web workspace is a thin local client. Scans, persistence, events, and exports still come from the Go backend.
- Scan history and settings remain in the same SQLite database used by `basalt serve` and `basalt scan`.
- Targets and aliases are persisted locally and can be reused across scans.
- The graph shown in the UI is synthesized from the raw evidence graph so the main view stays readable and target-centric.
- `basalt serve` remains the headless integration path for custom clients.
