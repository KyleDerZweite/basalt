# Basalt

> [!NOTE]
> **Status: On Hold / Experimental.** This project is currently paused while I rethink its direction. The Go CLI and local HTTP API work as an early prototype, but active development is on hold and expect breaking changes if it resumes.

Basalt is a Go CLI and local HTTP API for self-lookup and authorized OSINT. It scans usernames, emails, and domains and builds a graph of accounts, links, infrastructure, and identity signals. You can store targets and aliases for repeat scans and export results as JSON or CSV.

## Build

Use Go 1.27.3 or newer.

```bash
git clone https://github.com/KyleDerZweite/basalt.git
cd basalt/cli
go build -o basalt .
```

## Quick start

Run these examples from `basalt/cli` after building.

```bash
# Scan a username
./basalt scan -u kylederzweite

# Scan multiple seeds
./basalt scan -u kyle -e kyle@example.com -d kylehub.dev

# Export results
./basalt scan -u kyle --export json --export csv

# Create and reuse a stored target
./basalt target create kyle --name "Kyle"
./basalt target alias add kyle username:kylederzweite --primary
./basalt scan --target kyle

# Run the local API
./basalt serve
```

Use `./basalt --help` or `./basalt <command> --help` for the current CLI surface.

## Core commands

- `basalt scan` runs an ad hoc scan from username, email, and domain seeds
- `basalt target` manages persistent targets and curated aliases
- `basalt serve` runs the local HTTP API for local clients and integrations
- `basalt version` prints the current build version

## Docs

- [Local API](docs/local-api.md)
- [OSINT toolkit research and open decisions](docs/osint-toolkit-research.md)

## Configuration

Config lives in `~/.basalt/config` by default and uses `KEY=VALUE` lines:

```bash
GITHUB_TOKEN=ghp_xxxxxxxxxxxx
```

GitHub works without a token but has lower rate limits. Other current modules work without API keys.

Local runtime data is stored in `~/.basalt` by default unless `--data-dir` is overridden.

## Development

```bash
cd cli
go build ./...
go vet ./...
go test ./...
```

## Related open-source tools

If you are looking for mature, actively maintained OSINT tools today:

- [Maigret](https://github.com/soxoj/maigret) — Username search across thousands of sites with report generation
- [Sherlock](https://github.com/sherlock-project/sherlock) — Classic username lookup across social networks
- [sn0int](https://github.com/kpcyrd/sn0int) — Semi-automatic OSINT framework and package manager written in Rust
- [SpiderFoot](https://github.com/smicallef/spiderfoot) — Automated OSINT collection and reconnaissance framework

## Legal

For self-lookup and authorized research only.

- Only scan identifiers you own or have explicit consent to investigate
- Do not use Basalt to target third parties without authorization
- Public availability of an endpoint does not remove legal or ethical obligations

## License

[AGPLv3](LICENSE)
