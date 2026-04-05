# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Local product backend with persisted scan history, SQLite storage, local settings, and scan event streams
- `basalt serve` command for the local desktop/web product API
- HTTP API endpoints for scans, results, events, exports, module health, and settings
- Walker progress events for module verification, execution, discoveries, and failures
- Web workspace (`basalt web`) with React 19, Vite 8, TypeScript
- Pages: Home (dashboard), New Scan, Targets, Scan Workspace, Settings
- Graph visualization via cytoscape with dagre layout
- Live scan event timeline via polling
- Target management with persistent aliases
- Dark and light theme support
- `lucide-react` icon library replacing Unicode symbol icons

### Changed
- `scan` now uses the shared application service layer instead of wiring the walker directly in the command
- Scan runs are persisted locally and can be re-exported later from the local API
- Default local app data lives in `~/.basalt`
- Web UI design flattened: removed gradient overlays, backdrop-filter blur, noise textures, and oversized border radii in favor of flat surfaces with 1px borders and tight spacing

## [0.2.0] - 2026-03-31

### Added
- Relational graph architecture: nodes, edges, typed relationships
- Async Walker orchestrator with semaphore-bounded concurrency
- Module health checks (Healthy / Degraded / Offline) with degraded confidence penalty
- Automatic pivoting: discovered emails/usernames trigger further module runs
- 29 purpose-built modules:
  - **Identity**: Gravatar (email), Keybase (username)
  - **Dev/Tech**: GitHub (username/email/domain), GitLab (username), Stack Exchange (username), Docker Hub (username), DEV.to (username), Hacker News (username)
  - **Social**: Reddit, YouTube, Twitch, Discord, Instagram, TikTok, Medium, Telegram
  - **Link-in-bio**: Linktree, Beacons, Carrd, Bento
  - **Communication**: Matrix (username)
  - **Gaming**: Steam (username), OP.GG (username), Spotify (username)
  - **Infrastructure**: WHOIS/RDAP (domain), DNS/CT via crt.sh (domain), Shodan InternetDB (domain), Wayback Machine (domain), IPinfo (domain)
- Per-module confidence scoring
- Three output formats: color-coded table, JSON graph, CSV
- Config file support (`~/.basalt/config`) for API keys
- `--depth`, `--concurrency`, `--timeout` flags
- `--export` flag for JSON/CSV file output
- Seed node type resolution in walker (modules receive resolved types)
- False positive mitigation via OG metadata validation (Twitch, Instagram, TikTok)
- Redirect detection for profile existence (Bento)
- RDAP-compliant User-Agent to avoid rdap.org blocking
- Keybase identity proof extraction with cross-platform pivoting
- DEV.to linked GitHub/Twitter username extraction as pivotable nodes
- Shodan InternetDB integration for port/CVE/CPE discovery (no auth)
- Wayback Machine snapshot availability checking
- IPinfo geolocation and organization lookup for domains
- OP.GG search endpoint with automatic Riot ID tag resolution across 10 regions
- Steam public profile scraping (no API key required)
- Medium, Telegram, Spotify profile detection via OG metadata

### Changed
- Complete rewrite from v0.1 site-template engine to per-module architecture
- HTTP client sends module-specific headers where needed
- Steam module rewritten from API-key-based to public profile scraping

## [0.1.0] - 2025-12-01

### Added
- Initial release with YAML site-template engine
- Username checking across sites with multi-signal confidence scoring
- Email engine with 16 site-specific modules
- Maigret, Sherlock, and WhatsMyName site importers
- Proxy rotation, rate limiting, retry with backoff
- Graceful shutdown with partial result output
- Pivot discovery (email <-> username)
