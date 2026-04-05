# Contributing to Basalt

Thanks for your interest in contributing to Basalt! This document covers the process for submitting changes.

## Getting Started

1. Fork the repository
2. Clone your fork and create a branch:
   ```bash
   git clone https://github.com/<your-username>/basalt.git
   cd basalt/cli
   go build ./...
   go test ./...
   ```
3. Make your changes on a feature branch:
   ```bash
   git checkout -b my-feature
   ```

## Development

All Go source lives under `cli/`, all web UI source under `web/`. See [AGENTS.md](AGENTS.md) for architecture, build commands, and conventions.

```bash
# Go backend
cd cli
go build ./...        # build
go test ./...         # test
go vet ./...          # lint

# Web frontend
cd web
pnpm install          # install deps
pnpm dev              # dev server
pnpm build            # production build
pnpm typecheck        # type check
```

## Adding a Module

Each module lives in its own package under `cli/internal/modules/<name>/` and implements the `modules.Module` interface:

- `Name() string` -- short identifier
- `Description() string` -- one-line summary
- `CanHandle(nodeType string) bool` -- which node types this module accepts
- `Extract(ctx, node, client) (nodes, edges, error)` -- the actual work
- `Verify(ctx, client) (HealthStatus, string)` -- health check

Follow the pattern of existing modules (e.g., `twitch`, `github`). Register your module in `cli/cmd/scan.go`.

Every module must include a `*_test.go` file with at least:
- `TestCanHandle` -- verifies accepted node types
- `TestExtractFound` -- happy path with a mock HTTP server
- `TestExtractNotFound` -- 404 handling
- `TestVerify` -- health check

## Working on the Web UI

The frontend is a React 19 + TypeScript SPA in `web/`. Key conventions:

- Icons come from `lucide-react`. Import individual icons, not the barrel export.
- All styling is plain CSS in `web/src/index.css` using CSS custom properties. No Tailwind, no CSS-in-JS.
- Components live in `web/src/components/`, pages in `web/src/pages/`.
- Run `pnpm build` after changes to update the assets embedded in the Go binary.

## Submitting Changes

1. Ensure all tests pass: `cd cli && go test ./...`
2. Keep commits focused -- one logical change per commit
3. Write clear commit messages describing *why*, not just *what*
4. Open a pull request against `main`

## Pull Request Guidelines

- Keep PRs small and focused on a single concern
- Include tests for new functionality
- Update documentation if behavior changes
- Link related issues in the PR description

## Reporting Bugs

Open an issue at [github.com/KyleDerZweite/basalt/issues](https://github.com/KyleDerZweite/basalt/issues) with:
- What you expected to happen
- What actually happened
- Steps to reproduce
- Basalt version (`basalt --version`)

## Code of Conduct

Be respectful and constructive. This project exists to help people understand their own digital footprint. Any use against others without explicit consent is outside the scope of this project and will not be supported.

## License

By contributing, you agree that your contributions will be licensed under the [GNU Affero General Public License v3.0](LICENSE).
