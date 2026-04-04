# Basalt Local API

Basalt now exposes a local product API through:

```bash
cd cli
go run . serve
```

Default address: `http://127.0.0.1:8787`

If you want the browser-based product surface instead of a headless API, use:

```bash
cd cli
go run . web
```

That mounts the same API under `/api/*` and serves the local UI from the same origin.

Managed serve mode examples:

```bash
cd cli
go run . serve --detach
go run . serve --status
go run . serve --stop
```

For local clients and browser-based frontends, prefer:

```bash
cd cli
go run . serve \
  --listen 127.0.0.1:0 \
  --auth-token local-dev-token \
  --allow-origin http://localhost:5173 \
  --print-listen-json
```

Startup JSON example:

```json
{
  "listen_address": "127.0.0.1:42211",
  "base_url": "http://127.0.0.1:42211",
  "version": "0.1.0",
  "data_dir": "/home/kyle/.basalt"
}
```

All `/api/*` endpoints require `Authorization: Bearer <token>` when `--auth-token` is set.

Detached mode stores runtime metadata in `~/.basalt/run/serve.json` and writes logs to `~/.basalt/logs/serve.log` by default.

## Endpoints

### Root

- `GET /`
  - Returns basic server metadata and data directory details.

### Settings

- `GET /api/settings`
- `PUT /api/settings`

Payload:

```json
{
  "strict_mode": true,
  "disabled_modules": ["github", "reddit"],
  "legal_accepted_at": "2026-04-04T12:00:00Z"
}
```

### Module Health

- `GET /api/modules/health`

Query parameters:

- `depth`
- `concurrency`
- `timeout`
- `strict`

### Scans

- `GET /api/scans`
- `POST /api/scans`
- `GET /api/scans/{id}`
- `GET /api/scans/{id}/results`
- `GET /api/scans/{id}/workspace`
- `GET /api/scans/{id}/events`
- `GET /api/scans/{id}/events?stream=1`
- `GET /api/scans/{id}/export?format=json|csv`
- `POST /api/scans/{id}/cancel`

Create payload:

```json
{
  "seeds": [
    {"type": "username", "value": "kylederzweite"},
    {"type": "domain", "value": "example.com"}
  ],
  "depth": 2,
  "concurrency": 5,
  "timeout_seconds": 10,
  "strict_mode": false,
  "disabled_modules": [],
  "target_ref": "kyle"
}
```

### Targets

- `GET /api/targets`
- `POST /api/targets`
- `GET /api/targets/{id|slug}`
- `PATCH /api/targets/{id|slug}`
- `DELETE /api/targets/{id|slug}`
- `POST /api/targets/{id|slug}/aliases`
- `DELETE /api/targets/{id|slug}/aliases/{aliasId}`
- `GET /api/targets/{id|slug}/scans`

Target payload example:

```json
{
  "display_name": "Kyle",
  "slug": "kyle",
  "notes": "Main OSINT subject"
}
```

Alias payload example:

```json
{
  "seed_type": "username",
  "seed_value": "kylederzweite",
  "label": "main handle",
  "is_primary": true
}
```

## Event Stream

`GET /api/scans/{id}/events` returns JSON events.

`GET /api/scans/{id}/events?stream=1` or `Accept: text/event-stream` returns an SSE stream.

Current event types:

- `scan_queued`
- `scan_status`
- `module_verified`
- `verify_complete`
- `module_started`
- `module_finished`
- `node_discovered`
- `edge_discovered`
- `module_error`
- `scan_finished`
- `scan_failed`

## Storage

All local data is stored in `~/.basalt` by default:

- `~/.basalt/config` for API keys
- `~/.basalt/basalt.db` for scans, settings, events, targets, aliases, and scan insights

## Client Notes

- The Go backend is the source of truth for scans, persistence, and exports.
- Clients should use REST for settings, history, and results, and SSE for scan progress.
- Local clients should use `--listen 127.0.0.1:0` rather than hard-coding a port.
- Managed background mode is per `--data-dir`, so separate data dirs can run separate local APIs.
- `basalt web` is the recommended built-in browser client and uses the same API contract without requiring cross-origin requests.
- `GET /api/scans/{id}/workspace` is the preferred browser endpoint for the investigation graph and top-level scan summary.
