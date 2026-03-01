# Project Status

Last updated: 2026-03-01

## What Works

Basalt is a functional CLI tool that scans usernames and email addresses across thousands of platforms to discover associated accounts and build a relationship graph.

### Username Scanning
- ~1900 site definitions imported from three upstream OSINT databases (Maigret, Sherlock, WhatsMyName), deduplicated and normalized into a single file
- Multi-signal confidence scoring with dual-request verification to eliminate false positives - compares each target response against a control (known-nonexistent user) response using shingled Jaccard similarity
- Five weighted signals: HTTP status match, presence string match, absence string check, content differentiation, and redirect detection
- Soft-404 penalty catches sites that return 200 with generic error pages
- Supports both GET and POST-based site checks, including request body templates and custom content types

### Email Scanning
- 16 Go modules covering major services: Twitter, Spotify, Gravatar, Docker Hub, Duolingo, Pinterest, Imgur, Instagram, GitHub, Discord, Snapchat, Yahoo, Adobe, Office365, Samsung, Amazon
- Techniques include registration endpoint probing, login verification, password recovery flows, and CSRF-protected multi-step forms
- Some modules extract metadata like partially obfuscated recovery emails and phone numbers (Adobe)

### Auto-Pivoting
- Discovered data from one engine feeds into others automatically - a GitHub profile reveals an email, that email triggers email module checks, and vice versa
- Configurable pivot depth with deduplication to prevent loops
- Metadata extraction via CSS selectors and regex patterns defined in site YAML

### Infrastructure
- HTTP client with retry logic (exponential backoff on 429, 5xx, transient network errors)
- Per-domain rate limiting with configurable global and per-site limits
- Proxy rotation supporting HTTP and SOCKS5 proxies (single URL or file of URLs, round-robin)
- Control response caching to avoid redundant requests across site checks
- Graceful shutdown - first Ctrl+C drains in-flight requests and outputs partial results, second force-quits
- Output as JSON graph (nodes + edges) or color-coded terminal table

### Importing
- Import command converts upstream databases (Maigret, Sherlock, WhatsMyName JSON) into Basalt's YAML format
- Re-importing merges new sites with existing ones, deduplicating by URL template

## What Doesn't Work Yet

- **No tests** - no unit tests or integration tests exist. This is the most important next step.
- **No phone engine** - the phone seed type is recognized by the resolver but no engine handles it
- **No domain engine** - same situation as phone
- **No dashboard** - the JSON graph output is designed for a future web dashboard that doesn't exist yet
- **Email module accuracy is unverified** - the 16 modules are implemented based on documented API behavior but haven't been validated against live services at scale. Endpoints may have changed, rate limits may be aggressive, and some modules may produce false results.
- **No CI/CD** - no automated builds, no linting pipeline, no release process
- **No packaging** - no Homebrew formula, no Docker image, no release binaries

## Architecture Overview

The system is built around two abstractions:

**Engine interface** - each engine handles one or more seed types (username, email, phone, domain) and streams results through a channel. The scan command orchestrates engines through a registry, routing each seed to the right engine.

**Username sites as YAML, email sites as Go code** - username checks are declarative (URL template + expected response patterns) so they work well as YAML definitions. Email checks require multi-step HTTP flows with custom logic (CSRF extraction, JSON parsing, session management) that can't be captured declaratively, so each is a Go module implementing a standard interface.

Both engines feed into the same graph structure and pivot system, allowing cross-engine discovery chains.
