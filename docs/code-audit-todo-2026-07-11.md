# Basalt Audit TODO (2026-07-11)

Prioritized backlog from the architecture/engineering audit. Goal: a correct backend
foundation before frontend refinement. Priorities: P0 blocker, P1 high, P2 medium, P3 low.
Effort: S small, M medium, L large.

## P0 - Blockers (fix first)

- [ ] **Data race: async scan mutates a `ScanRecord` the API serializes.** (S)
  `internal/app/service.go:253` (write) vs `internal/api/server.go:133` (read).
  `StartScan` returns the same `*ScanRecord` the background `executeScan` goroutine keeps
  mutating; `POST /api/scans` JSON-encodes it concurrently. `go test -race ./internal/api/`
  fails deterministically. Torn reads or an encoder panic (see next item) in production.
  Fix: return an immutable snapshot/deep copy from `StartScan`; let the goroutine own the record.

- [ ] **No panic isolation in the walker; one module panic crashes the whole scan/server.** (S)
  `internal/walker/walker.go:200-301`. Dispatch goroutines have no `recover` (none anywhere
  in the tree). Any panic in the 38 modules (or the encoder from the race above) kills the
  long-lived `serve`/`web` process. Violates AGENTS.md invariant.
  Fix: `defer recover()` in the goroutine body -> record error, `IncrErrors`, emit `module_error`.
  Add a walker test with a panicking `fakeModule`.

## P1 - High

- [ ] **Unauthenticated local API + permissive CORS leaks OSINT data to any web origin.** (M)
  `internal/webui/server.go:31`, `internal/api/server.go:478-484`, `cmd/web.go`.
  `web` builds `api.NewServer(service, api.Options{})` with no auth; default CORS reflects
  `Access-Control-Allow-Origin: *` for any Origin. Any site the user visits can read scans
  and `POST /api/scans` against localhost. `serve` shares the default (auth token empty).
  Fix: stop reflecting `*`; default-deny cross-origin, allow same-origin + configured origins;
  consider a generated session token for `web`; check `Origin`/`Sec-Fetch-Site` on writes.

- [ ] **Rate limiting, proxy rotation, and DNS caching are dead code.** (M)
  `internal/httpclient/{ratelimit,proxy,dnscache}.go` have no non-test callers; `DoRequest`
  never throttles. 38 modules at concurrency 5 with pivoting burst third-party services ->
  IP bans. `CHANGELOG.md:87` and `AGENTS.md:32` claim these features exist.
  Fix: wire `DomainRateLimiter.Wait` into `DoRequest`, thread a limiter through `walker.New`,
  expose proxy config via flags/config. If not wiring proxy/DNS now, delete them and fix docs.

## P2 - Medium

- [ ] **`Service.Close()` does not cancel/drain active scans.** (S/M)
  `internal/app/service.go:56-58`; goroutine at `service.go:95-98`. On shutdown, in-flight
  scans write to a closed DB (`sql: database is closed` warnings) and lose events.
  Fix: track a `sync.WaitGroup` + cancel all `s.active`, wait with timeout, then close store.

- [ ] **SQLite: no busy timeout, unbounded write pool.** (S)
  `internal/app/paths.go:32`, `store.go:34`. Concurrent `UpdateScan` vs `AppendEvent` can hit
  `SQLITE_BUSY`. Fix: add `_pragma=busy_timeout(5000)` to DSN and/or `db.SetMaxOpenConns(1)`.

- [ ] **No request body size limits on API decoders.** (S)
  `internal/api/server.go:75,124,150,185,220`. Unbounded `json.NewDecoder(r.Body)` -> memory
  DoS (trivial given no auth). Fix: wrap with `http.MaxBytesReader(w, r.Body, 1<<20)`.

- [ ] **CORS `Allow-Methods` omits `PATCH`/`DELETE` used by target routes.** (S)
  `internal/api/server.go:463` vs handlers at `:183,195,236`. Browser preflight rejects target
  update/delete cross-origin. Fix: add `PATCH, DELETE`.

- [ ] **SSRF: modules fetch seed/discovered domains and follow redirects, no host filtering.** (M)
  `internal/modules/securitytxt/securitytxt.go:113-138` (also whois, wayback, shodan resolve).
  `https://<node.Label>/...` with up to 5 redirects and no private/reserved-IP block; an API
  caller can point a seed at `169.254.169.254`/`127.0.0.1`/internal hosts.
  Fix: SSRF guard in `httpclient` (resolve, reject private/reserved ranges, re-check per redirect).

## P3 - Low

- [ ] **SSE reconnect replays full backlog; small event-loss window.** (S)
  `web/src/hooks/useScanEvents.ts:39`, `internal/api/server.go:379-397`. `EventSource` URL has
  no `after` -> duplicate events on reconnect; backlog fetched before `Subscribe` -> gap.
  Fix: subscribe before reading backlog, dedupe by sequence, persist `after` client-side.

- [ ] **`panic(err)` in `webui.NewServer` violates the no-panic rule.** (S)
  `internal/webui/server.go:28`. Effectively unreachable, but return an error instead.

- [ ] **Docs drift.** (S)
  `AGENTS.md:64` says register modules in `cmd/scan.go`; real location is
  `internal/app/modules.go:50`. `AGENTS.md:32`/`CHANGELOG.md:87` claim live throttling (see P1).

- [ ] **`store.go` (977 LOC) duplication and mixed concerns.** (M, maintainability)
  `scanFromRow`/`scanSummaryFromRow` near-identical (`store.go:723-859`); scans/events/settings/
  targets/health-cache in one file. Factor shared scan decode; split per aggregate. Same lens on
  `service.go` (638) and `workspace.go` (541).

- [ ] **Redundant module-health resolution per CLI scan.** (S)
  `cmd/scan.go:91` then `internal/app/service.go:307` resolve health twice (cache softens it).

- [ ] **Frontend refetch pressure.** (S)
  `web/src/hooks/useScanEvents.ts` triggers a workspace refetch on every SSE event *and* every 3s.
  Debounce/coalesce.

- [ ] **`baseURL` test-override pattern not uniform.** (S)
  `roblox` and `dnsct` diverge from the AGENTS.md convention (multiple endpoints / resolver).

## Done

- [x] **Bump Go toolchain to 1.25.12** (fixes govulncheck GO-2026-5856, crypto/tls).
  `go.mod` now `go 1.25.12` + `toolchain go1.25.12`; `GOTOOLCHAIN=auto`. `govulncheck ./...`
  reports no vulnerabilities; `go build`/`go vet` pass.
</content>
</invoke>
