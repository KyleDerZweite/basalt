# Basalt

Open-source OSINT tool for discovering your digital footprint. Checks a username or email across thousands of platforms and builds a relationship graph of all discovered accounts.

Basalt is inspired by the broader OSINT tool ecosystem covered in our research, streamlined into one focused project.

**Designed for self-lookup and authorized security research only.** You must have explicit consent before scanning any identifier you don't own.

## Install

```bash
go install github.com/kyle/basalt@latest
```

Or build from source:

```bash
cd cli
go build -o basalt .
```

## Usage

Scan a username:

```bash
basalt scan torvalds
basalt scan torvalds --output table
```

Scan an email (runs 16 email verification modules):

```bash
basalt scan user@example.com --output table
```

Auto-pivoting (follows discovered emails/usernames to find more accounts):

```bash
basalt scan torvalds --max-pivot-depth 2
```

Import upstream site definitions:

```bash
basalt import maigret path/to/maigret/resources/data.json
basalt import sherlock path/to/sherlock/sherlock_project/resources/data.json
basalt import wmn path/to/wmn/wmn-data.json
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-o, --output` | `json` | Output format: `json` or `table` |
| `-c, --concurrency` | `20` | Maximum concurrent requests |
| `-t, --timeout` | `15` | HTTP request timeout (seconds) |
| `-v, --verbose` | `false` | Debug logging to stderr |
| `--threshold` | `0.50` | Minimum confidence score to report a match |
| `--max-pivot-depth` | `0` | Auto-pivot depth (0 = disabled) |
| `--no-pivot` | `false` | Disable pivoting entirely |
| `--proxy` | | Proxy URL or path to proxy list file |
| `--site-dirs` | | Additional YAML site definition directories |
| `--rate-limit` | `10` | Global requests per second |

### Output

**JSON** (default) produces a graph with nodes (seeds, accounts) and edges (relationships). Pipe to `jq` or feed into a visualization tool.

**Table** prints a color-coded terminal table sorted by confidence score.

## How It Works

Basalt has two engines:

**Username engine** checks ~1900 sites using YAML-defined rules. Each check makes two HTTP requests (target + control) and scores confidence across 5 signals: HTTP status, presence strings, absence strings, content differentiation (Jaccard similarity), and redirect detection.

**Email engine** runs 16 Go modules that each implement site-specific verification logic (registration probes, login checks, password recovery flows, CSRF-protected forms). These can't be expressed as YAML because they require multi-step flows and custom response parsing.

When pivoting is enabled, discovered emails and usernames from one engine feed into the other automatically.

## Site Definitions

Sites load from these directories (first match wins, duplicates skipped):

1. `data/sites/` relative to the binary
2. `data/sites/` relative to cwd
3. `~/.basalt/sites/`
4. Any `--site-dirs` you pass

Each YAML file contains site definitions with URL templates, expected responses, and optional extraction rules. See `data/sites/example.yaml` for the format.

## Legal

This tool queries only publicly accessible URLs. No authentication bypass, no private data access, no API abuse.

- GDPR: Only scan identifiers you own or have the data subject's explicit consent to search
- Rate limiting is built in and enforced per-domain
- Proxy support is for privacy, not evasion

## License

[AGPLv3](LICENSE): if you use Basalt in a network service, you must share your source code.
