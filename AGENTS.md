# AGENTS.md

Guidelines for AI coding agents working on this codebase.

## Orientation

This is a Go CLI tool. All source is in `cli/`. Read the code, not this file, for implementation details. These docs describe intent and conventions only.

Entry point: `cli/main.go` -> `cli/cmd/root.go` (Cobra CLI).

Module path: `github.com/kyle/basalt`.

## Build & Run

```bash
cd cli
go build -o basalt .    # build
go vet ./...            # lint
go test ./...           # test (when tests exist)
```

## Architecture

Two engine types, one interface (`engine.Engine`):

- **Username engine** (`engines/username/`): YAML-driven site definitions, dual-request confidence scoring
- **Email engine** (`engines/email/`): Go modules per site, each implements `email.Module` interface

Both stream results into the same graph via `chan<- engine.Result`. The scan command in `cmd/scan.go` orchestrates engines through a registry and pivot loop.

## Principles

Adhere to these 10 principles. If a situation requires deviating, leave a comment explaining why.

1. **KISS (Keep It Simple, Stupid)**: solve the problem at hand, nothing more. No speculative abstractions, no premature optimization, no clever one-liners. If a simpler approach works, use it.
2. **DRY (Don't Repeat Yourself)**: extract shared logic rather than copy-pasting. If two modules share a pattern, factor it out. If a doc restates code, delete the doc. Every piece of knowledge should live in exactly one place.
3. **Open/Closed**: code should be open to extension but closed to modification. New engines are added by implementing an interface and registering, not by editing the core engine loop. New email modules are added as new files without touching existing modules. Extend, don't modify.
4. **Composition Over Inheritance**: build complex behavior by combining objects with individual behaviors, not by extending base classes. The `engine.Engine` interface is 3 methods; `email.Module` is 3 methods. An engine *has* a client, a rate limiter, and modules, and it doesn't inherit from them. Keep interfaces narrow and compose them.
5. **YAGNI (You Aren't Going to Need It)**: don't build features, abstractions, or configurability until there's a concrete need. No feature flags for hypothetical futures. Three similar lines are better than a premature helper function.
6. **Single Responsibility**: every package, file, and function does one thing. One engine per package, one module per file, one concern per function. If you can't describe what it does without "and", split it.
7. **Document Your Code**: leave comments to explain *why*, not *what*. Explain non-obvious decisions, edge cases, and intent. Use Go doc comments on exported types and functions. Don't over-document self-evident code.
8. **Separation of Concerns**: each package owns one concern and doesn't reach into others. `httpclient` handles HTTP, `engine` defines contracts, `engines/*` implement them, `graph` structures output, `output` renders it, `cmd` wires everything together. They communicate through interfaces and data types, not internal state.
9. **Refactor**: revisiting and rewriting code is normal. Use growing familiarity with the project to simplify, deduplicate, and clarify. Make it more efficient while keeping results identical. Don't preserve bad patterns out of inertia.
10. **Clean Code At All Costs**: write code for humans, not to impress. No clever tricks, no packing logic into one line, no ego. If your code is easy to read it will be easy to maintain. Clarity always wins over brevity.

## Before You Code

1. Read the file you're modifying. Understand existing patterns before changing anything.
2. Run `go build ./...` and `go vet ./...` after changes.
3. The engine interface in `engine/engine.go` is the central contract, so understand it first.

## Conventions

- **License header**: every Go file starts with `// SPDX-License-Identifier: AGPL-3.0-or-later`
- **Options pattern**: constructors take `New(required, ...Option)` where options are `WithX` functions
- **Error handling**: return errors, don't panic. Log with `slog.Debug` for non-fatal issues.
- **Context**: all long-running operations accept `context.Context` and must check cancellation
- **Streaming**: engines send results on a channel and MUST close it when done
- Site dedup is by both name and URL template (case-insensitive)
- Email modules: one file per service in `engines/email/modules/`, registered in `registry.go`

## Key Patterns

- `engine.Engine` interface: `Name()`, `SeedTypes()`, `Check(ctx, seed, chan<- Result)`; engine MUST close the channel
- HTTP client: use `client.Do()` for GET, `client.DoRequest()` for other methods
- Confidence: username engine uses 5-signal scoring; email engine uses simplified binary (0.95/0.05)
- Context cancellation must be respected everywhere for graceful shutdown

## Adding a New Email Module

1. Create `cli/internal/engines/email/modules/<service>.go`
2. Implement `email.Module` interface: `Name()`, `Category()`, `Check(ctx, email, client)`
3. Return `email.ModuleResult` with `Exists` as `*bool` (nil = inconclusive)
4. Register in `modules/registry.go` `All()` function
5. Handle rate limiting (check for 429, return `RateLimit: true`)

## Adding a New Engine Type

1. Implement `engine.Engine` in a new package under `engines/`
2. Register in `cmd/scan.go` via `registry.Register()`
3. The pivot system handles cross-engine seed propagation automatically

## Things to Avoid

- Don't duplicate site definitions across YAML files (loader deduplicates by name and URL template, but don't rely on it)
- Don't hardcode URLs in the username engine -- that's what YAML site definitions are for
- Don't add dependencies without justification -- the binary should stay lean
- Don't write docs that restate code -- the codebase is the source of truth
- Don't use em dashes (`—`), double hyphens (`--`), triple dashes (`---`), or generic AI-sounding filler phrasing in docs, commit messages, or agent responses
