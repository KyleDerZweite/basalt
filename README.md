# Basalt

Relational OSINT tool for discovering your digital footprint. Basalt runs 29 purpose-built modules against usernames, emails, and domains, then builds a relationship graph of everything it finds.

Unlike tools that spray thousands of sites with URL templates, Basalt uses per-module logic with structured API calls, HTML scraping, and module-level health checks. Each module scores its own confidence. No false positives from generic status code matching.

**For self-lookup and authorized research only.** You must have explicit consent before scanning any identifier you don't own.

## Install

```bash
go install github.com/kylederzweite/basalt@latest
```

Or build from source:

```bash
git clone https://github.com/KyleDerZweite/basalt.git
cd basalt && cd cli
go build -o basalt .
```

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
| `--export` | | Export format: `json`, `csv` (repeatable) |
| `-v, --verbose` | `false` | Show module health details |

### Output

Terminal output is a color-coded table sorted by confidence score (green >= 0.80, yellow >= 0.50).

`--export json` writes the full graph (nodes, edges, metadata) to a timestamped JSON file.

`--export csv` writes a flat node list to a timestamped CSV file.

## Modules

29 modules across 8 categories:

| Category | Modules | Seed Types |
|----------|---------|------------|
| Identity | Gravatar, Keybase | email, username |
| Dev/Tech | GitHub, GitLab, StackExchange, Docker Hub, DEV.to, Hacker News | username, email |
| Social | Reddit, YouTube, Twitch, Discord, Instagram, TikTok, Medium, Telegram | username |
| Link-in-Bio | Linktree, Beacons, Carrd, Bento | username |
| Comms | Matrix | username |
| Gaming | Steam, OP.GG, Spotify | username |
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
