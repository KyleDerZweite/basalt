# Basalt

Basalt is a Go CLI for relational OSINT on identifiers you own or are explicitly authorized to investigate. It runs focused modules against usernames, emails, and domains, then builds a graph of accounts, links, infrastructure, and identity signals.

Basalt also includes:

- a local product API via `basalt serve`
- a local browser workspace via `basalt web`
- persistent targets and aliases for repeated investigations

Unlike spray-and-pray username checkers, Basalt uses per-module extraction logic, health checks, confidence scoring, and graph pivots.

## Install

```bash
git clone https://github.com/KyleDerZweite/basalt.git
cd basalt/cli
go build -o basalt .
```

## Quick Start

```bash
# Scan a username
basalt scan -u kylederzweite

# Scan multiple seeds
basalt scan -u kyle -e kyle@example.com -d kylehub.dev

# Export results
basalt scan -u kyle --export json --export csv

# Run the local API
basalt serve

# Run the local browser workspace
basalt web

# Create and reuse a stored target
basalt target create kyle --name "Kyle"
basalt target alias add kyle username:kylederzweite --primary
basalt scan --target kyle
```

Use `basalt --help` or `basalt <command> --help` for the current CLI surface.

## Core Commands

- `basalt scan` runs an ad hoc scan from username, email, and domain seeds
- `basalt target` manages persistent targets and curated aliases
- `basalt serve` runs the local HTTP API for local clients and integrations
- `basalt web` runs the built-in browser workspace on the same local backend
- `basalt version` prints the current build version

## Docs

- [Local API](docs/local-api.md)
- [Web Workspace](docs/web-ui.md)
- [Future Modules](docs/future-modules.md)

The README intentionally stays high level. API endpoints, workspace behavior, and backlog details live in `docs/`.

## Modules

Basalt currently ships with 38 modules.

| Category | Modules | Seed Types |
|----------|---------|------------|
| Identity | Gravatar, Keybase | email, username |
| Dev/Tech | GitHub, GitLab, Codeberg, Codeforces, StackExchange, Docker Hub, DEV.to, Hacker News | username, email |
| Social | Reddit, YouTube, Twitch, Discord, Instagram, TikTok, Medium, Telegram, Wattpad | username |
| Link-in-Bio | Linktree, Beacons, Carrd, Bento | username |
| Comms | Matrix | username |
| Gaming | Steam, OP.GG, Spotify, Chess.com, Lichess, MyAnimeList, Roblox | username |
| Productivity | Trello | username |
| Domain | WHOIS/RDAP, DNS/CT, Security.txt | domain |
| Infrastructure | Shodan, Wayback Machine, IPinfo | domain |

Modules self-report health before scanning:

- `healthy`: normal operation
- `degraded`: works, but confidence is reduced
- `offline`: skipped

## Configuration

Config lives in `~/.basalt/config` by default and uses `KEY=VALUE` lines:

```bash
GITHUB_TOKEN=ghp_xxxxxxxxxxxx
```

GitHub works without a token but has lower rate limits. Other current modules work without API keys.

Local runtime data is stored in `~/.basalt` by default unless `--data-dir` is overridden.

## Development

```bash
cd web
pnpm install
pnpm build

cd cli
go build ./...
go vet ./...
go test ./...
```

## Dependabot Auto-Merge

This repo includes GitHub Actions that:

- run Go and web CI on every pull request
- enable native GitHub auto-merge for non-draft Dependabot PRs when they are conflict-free

Required GitHub repo settings:

- enable `Allow auto-merge` under `Settings -> General -> Pull Requests`
- add branch protection on your default branch and require the `CLI` and `Web` status checks before merging

With those settings in place, Dependabot PRs will queue for squash-merge automatically and only land after the required checks pass.

## Legal

For self-lookup and authorized research only.

- Only scan identifiers you own or have explicit consent to investigate
- Do not use Basalt to target third parties without authorization
- Public availability of an endpoint does not remove legal or ethical obligations

## License

[AGPLv3](LICENSE)
