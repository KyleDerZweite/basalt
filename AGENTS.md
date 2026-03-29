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

## Principles

Adhere to these 10 principles. If a situation requires deviating, leave a comment explaining why.

1. **KISS (Keep It Simple, Stupid)**
2. **DRY (Don't Repeat Yourself)**
3. **Open/Closed**
4. **Composition Over Inheritance**
5. **YAGNI (You Aren't Going to Need It)**
6. **Single Responsibility**
7. **Document Your Code**
8. **Separation of Concerns**
9. **Refactor**
10. **Clean Code At All Costs**

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

## Things to Avoid

- Don't duplicate site definitions across YAML files (loader deduplicates by name and URL template, but don't rely on it)
- Don't hardcode URLs in the username engine -- that's what YAML site definitions are for
- Don't add dependencies without justification -- the binary should stay lean
- Don't write docs that restate code -- the codebase is the source of truth
- Don't use em dashes (`—`), double hyphens (`--`), triple dashes (`---`), or generic AI-sounding filler phrasing in docs, commit messages, or agent responses
