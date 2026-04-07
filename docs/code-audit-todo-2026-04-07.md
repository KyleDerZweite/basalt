<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

# Code Audit Remediation TODO

Date: 2026-04-07

## Phase 1: Audit Remediation

- [x] Enable SQLite foreign key enforcement and cover it with tests.
- [x] Stop decoding stored graphs for scan list queries that do not return graph data.
- [x] Fix graph node ingestion so later sightings can merge confidence, properties, and pivotability.
- [x] Harden event persistence so concurrent event writes do not race sequence allocation.
- [x] Fix the brittle embedded web asset test so `go test ./...` passes against current build output.
- [x] Apply the HTTP client connect timeout option to the actual transport.
- [x] Reduce target listing overhead by removing the alias N+1 query pattern.

## Phase 2: Module Health TTL

- [x] Add persistent module health cache storage in SQLite.
- [x] Cache module health by module name, app version, and config hash.
- [x] Use status-aware TTLs: healthy and degraded for 3h, offline for 30m.
- [x] Add CLI flags to refresh health, clear the cache, and override the TTL.
- [x] Reuse cached health in CLI, API, and scan execution to avoid duplicate verification.
- [x] Stop forcing repeated module health verification from the web shell refresh loop.
- [x] Add tests for cache hits, cache expiry, cache invalidation, and override flags.

## Validation

- [x] Run targeted Go tests for changed packages.
- [x] Run `go test ./...`.
- [x] Run `go vet ./...`.
- [x] Run `go build ./...`.
- [x] Run `pnpm typecheck`.
- [x] Commit the completed remediation and TTL changes.
