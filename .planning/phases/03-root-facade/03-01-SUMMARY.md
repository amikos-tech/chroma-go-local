---
phase: 03-root-facade
plan: 01
subsystem: api
tags: [go, facade, type-alias, re-export, purego]

requires:
  - phase: 02-file-migration
    provides: "Go implementation in internal/runtime/ and internal/library/"
provides:
  - "Root package chroma with Server, Config, Error re-exports"
  - "Public API surface matching pre-refactor import path"
  - "examples/go/basic/main.go compiles against facade"
affects: [03-root-facade, 04-test-reorg, 05-ci-docs]

tech-stack:
  added: []
  patterns: ["type alias facade (type X = runtime.X)", "thin wrapper functions (func F() { return runtime.F() })"]

key-files:
  created: [doc.go, chroma.go, config.go, errors.go]
  modified: []

key-decisions:
  - "Zero logic in facade files -- all functions are thin wrappers, all types use alias syntax"

patterns-established:
  - "Facade pattern: type alias + thin wrapper for re-exporting internal symbols"
  - "doc.go for package-level documentation only (no imports, no code)"

requirements-completed: [FACADE-01, FACADE-02, FACADE-03, FACADE-04, FACADE-05]

duration: 1min
completed: 2026-03-20
---

# Phase 03 Plan 01: Root Facade Summary

**Thin facade at repo root re-exporting Server, Config, and Error symbols from internal/runtime via type aliases and wrapper functions**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-20T18:58:08Z
- **Completed:** 2026-03-20T18:59:18Z
- **Tasks:** 1
- **Files modified:** 4

## Accomplishments
- Created 4 facade files (doc.go, chroma.go, config.go, errors.go) at repo root
- Restored public API surface: Server, StartServerConfig, ServerConfig, ServerOption types as aliases
- All 11 builder functions (WithPort, WithListenAddress, etc.) forwarded as thin wrappers
- 9 error code constants and 5 sentinel error variables re-exported
- examples/go/basic/main.go compiles successfully against the facade
- `go build ./...` and `go vet ./...` pass cleanly

## Task Commits

Each task was committed atomically:

1. **Task 1: Create doc.go and core facade files** - `d502d6f` (feat)

## Files Created/Modified
- `doc.go` - Package-level documentation for chroma
- `chroma.go` - Server, StartServerConfig types and Init, StartServer, Version, VersionWithError
- `config.go` - ServerConfig, ServerOption types and all With* builder functions plus NewServer, DefaultServerConfig
- `errors.go` - Error code constants and sentinel error variables

## Decisions Made
- Zero logic in facade files: all functions use thin wrapper style (`func F() { return runtime.F() }`), not variable assignment style (`var F = runtime.F`), per plan decision D-04
- Type aliases use `=` syntax to preserve methods and full compatibility

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all facade functions are fully wired to internal/runtime.

## Next Phase Readiness
- Root facade complete for server, config, and errors
- Plan 03-02 (embedded facade) can proceed to add embedded runtime re-exports
- Phase 04 (test reorganization) unblocked for root package test coverage

## Self-Check: PASSED

- All 4 facade files exist at repo root
- Commit d502d6f found in git log
- SUMMARY.md created at expected path

---
*Phase: 03-root-facade*
*Completed: 2026-03-20*
