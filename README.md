# Basalt

Relational OSINT tool for discovering your digital footprint. Basalt runs 37 purpose-built modules against usernames, emails, and domains, then builds a relationship graph of everything it finds.

Basalt also includes a **local product backend**: persisted scan history, local settings, a local HTTP API, and live scan events for local clients built on top of the Go engine.

For interactive local use, Basalt can also serve a browser-based workspace directly from the Go binary with `basalt web`.

Unlike tools that spray thousands of sites with URL templates, Basalt uses per-module logic with structured API calls, HTML scraping, and module-level health checks. Each module scores its own confidence. No false positives from generic status code matching.

**For self-lookup and authorized research only.** You must have explicit consent before scanning any identifier you don't own.

## Install

```bash
git clone https://github.com/KyleDerZweite/basalt.git
cd basalt/cli
go build -o basalt .
```

The canonical repository is `https://github.com/KyleDerZweite/basalt`.

## Usage

```bash
# Scan a username
basalt scan -u kylederzweite

# Scan an email
basalt scan -e kyle@example.com

# Scan a domain
basalt scan -d kylehub.dev

# Multiple seeds at once
basalt scan -u kyle -e kyle@example.com -d kylehub.dev

# Export results
basalt scan -u kyle --export json --export csv

# Verbose mode (show module health details)
basalt scan -u kyle -v

# Run the local product API
basalt serve

# Run the local web workspace and open it in your browser
basalt web

# Run the web workspace without opening a browser
basalt web --no-open

# Run the local API in the background
basalt serve --detach

# Show managed API status
basalt serve --status

# Stop the managed API
basalt serve --stop
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-u, --username` | | Username seed (repeatable) |
| `-e, --email` | | Email seed (repeatable) |
| `-d, --domain` | | Domain seed (repeatable) |
| `--depth` | `2` | Maximum pivot depth |
| `--concurrency` | `5` | Maximum concurrent module requests |
| `--timeout` | `10` | Per-module timeout in seconds |
| `--config` | `~/.basalt/config` | Path to config file for API keys |
| `--data-dir` | `~/.basalt` | Path to local app data, scan history, and SQLite store |
| `--export` | | Export format: `json`, `csv` (repeatable) |
| `-v, --verbose` | `false` | Show module health details |

### Output

Terminal output is a color-coded table sorted by confidence score (green >= 0.80, yellow >= 0.50).

`--export json` writes the full graph (nodes, edges, metadata) to a timestamped JSON file.

`--export csv` writes a flat node list to a timestamped CSV file.

## Local Product API

`basalt serve` starts a local HTTP API on `127.0.0.1:8787` by default. It persists:

- scan history
- scan event streams
- local settings
- exported graph results

Current endpoints:

- `GET /api/scans`
- `POST /api/scans`
- `GET /api/scans/{id}`
- `GET /api/scans/{id}/results`
- `GET /api/scans/{id}/events`
- `GET /api/scans/{id}/export?format=json|csv`
- `POST /api/scans/{id}/cancel`
- `GET /api/modules/health`
- `GET /api/settings`
- `PUT /api/settings`

All scan data is stored locally in `~/.basalt/basalt.db` unless `--data-dir` is overridden.

For local clients and future UI layers, `serve` also supports:

- `--listen 127.0.0.1:0` for an OS-assigned port
- `--auth-token <token>` for bearer auth on every `/api/*` route
- `--allow-origin <origin>` for strict CORS allowlists
- `--detach` to run the API in the background
- `--status` to inspect the managed API process
- `--stop` to gracefully stop the managed API process
- `--force` with `--stop` to hard-stop after timeout
- `--log-file` to override the default log file path
- `--print-listen-json` for machine-readable startup metadata

See `docs/local-api.md` for the current local API contract.

## Local Web Workspace

`basalt web` starts a same-origin browser workspace on `127.0.0.1:8788` by default. It serves:

- the local web UI at `/`
- bootstrap runtime config at `/app/bootstrap`
- the existing local API at `/api/*`

Useful flags:

- `--listen 127.0.0.1:0` for an OS-assigned port
- `--open` to explicitly open the browser after startup
- `--no-open` to keep the server running without launching a browser
- `--data-dir` to point the workspace at a different local SQLite store

The browser workspace is same-origin with the API, so it does not require CORS or bearer-token bootstrapping in normal use.

See `docs/web-ui.md` for the current browser workspace behavior.

## Modules

37 modules across 9 categories:

| Category | Modules | Seed Types |
|----------|---------|------------|
| Identity | Gravatar, Keybase | email, username |
| Dev/Tech | GitHub, GitLab, Codeberg, Codeforces, StackExchange, Docker Hub, DEV.to, Hacker News | username, email |
| Social | Reddit, YouTube, Twitch, Discord, Instagram, TikTok, Medium, Telegram, Wattpad | username |
| Link-in-Bio | Linktree, Beacons, Carrd, Bento | username |
| Comms | Matrix | username |
| Gaming | Steam, OP.GG, Spotify, Chess.com, Lichess, MyAnimeList, Roblox | username |
| Productivity | Trello | username |
| Domain | WHOIS/RDAP, DNS/CT | domain |
| Infrastructure | Shodan, Wayback Machine, IPinfo | domain |

Modules self-report health before scanning:
- **Healthy**: normal operation
- **Degraded**: works but confidence is halved (rate limits, intermittent issues)
- **Offline**: skipped entirely (API down, missing key)

## How It Works

Basalt uses a **reactive graph walker**. There are no tiers or waves. Execution order emerges from data flow:

1. Seed nodes are added to the graph
2. All modules that can handle each seed type are dispatched concurrently
3. When a module returns new nodes (emails, usernames, domains), those are fed back into the walker
4. This continues until pivot depth is reached or no new pivotable nodes are found

Each module independently decides what to extract and what confidence to assign. The walker handles deduplication, concurrency limiting, and depth tracking.

## Configuration

API keys go in `~/.basalt/config` (or pass `--config path`):

```
GITHUB_TOKEN=ghp_xxxxxxxxxxxx
```

GitHub works without a token but has lower rate limits. All other modules work without API keys.

## Legal

This tool queries only publicly accessible endpoints. No authentication bypass, no private data access.

- GDPR: only scan identifiers you own or have explicit consent to search
- Rate limiting is built in and enforced per-domain
- Proxy support is for privacy, not evasion

## License

[AGPLv3](LICENSE): if you use Basalt in a network service, you must share your source code.
