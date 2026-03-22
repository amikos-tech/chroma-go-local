---
phase: 03-root-facade
plan: 02
subsystem: api
tags: [go, facade, type-alias, re-export, embedded, backup, rebuild, compaction, wal-prune]

requires:
  - phase: 03-root-facade
    plan: 01
    provides: "Root facade with Server, Config, Error re-exports"
  - phase: 02-file-migration
    provides: "Go implementation in internal/runtime/ and internal/library/"
provides:
  - "Embedded mode types, config, and builder functions at root import path"
  - "Backup types, mode constants, and option functions at root import path"
  - "Rebuild, compaction, and WAL prune types and option functions at root import path"
  - "Complete public API surface (~110 explicit re-exports) from root package"
affects: [04-test-reorg, 05-ci-docs]

tech-stack:
  added: []
  patterns: ["type alias facade (type X = runtime.X)", "thin wrapper functions for re-exporting internal options"]

key-files:
  created: [embedded.go, backup.go, rebuild.go, compaction.go, wal_prune.go]
  modified: []

key-decisions:
  - "Followed D-04 wrapper style consistently across all 5 files"
  - "compaction.go has zero functions -- methods auto-forward via type alias"
  - "wal_prune.go imports both time and internal/runtime for WithWALPruneMaxAge(time.Duration)"

patterns-established:
  - "Facade files for interface-based options (BackupOption, WALPruneOption, RebuildCollectionOption) use thin wrapper functions"
  - "Type-only facades (compaction.go) omit function wrappers since methods auto-forward via alias"

requirements-completed: [FACADE-01, FACADE-02, FACADE-03, FACADE-04]

duration: 3min
completed: 2026-03-20
---

# Phase 03 Plan 02: Embedded and Maintenance Facade Summary

**Facade completing all ~75 remaining symbol re-exports: embedded mode types, backup, rebuild, compaction, and WAL prune APIs accessible from root import path**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-20T19:01:29Z
- **Completed:** 2026-03-20T19:04:30Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Created embedded.go with 32 type aliases, 3 constants, and 7 wrapper functions
- Created backup.go with 7 type aliases, 2 mode constants, and 4 option wrapper functions
- Created rebuild.go with 2 type aliases and 4 option wrapper functions
- Created compaction.go with 4 type aliases (zero functions, methods auto-forward)
- Created wal_prune.go with 3 type aliases and 7 option wrapper functions (dual import: time + runtime)
- Total across all 9 facade files: 54 type aliases, verified by grep count
- `go build ./...`, `go vet ./...`, and `go build ./examples/go/basic/` all pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Create embedded.go facade** - `15413fc` (feat)
2. **Task 2: Create backup, rebuild, compaction, wal_prune facades** - `0468775` (feat)

## Files Created/Modified
- `embedded.go` - Embedded, EmbeddedConfig, ~30 request/response types, NewEmbedded, StartEmbedded, builder functions
- `backup.go` - BackupMode, BackupOption, BackupManifest, mode constants, With* option functions
- `rebuild.go` - RebuildCollectionResult, RebuildCollectionOption, WithRebuild* option functions
- `compaction.go` - CompactCollectionRequest, CompactAllRequest, CompactionCollectionResult, CompactionResult
- `wal_prune.go` - WALPruneCollectionResult, WALPruneResult, WALPruneOption, WithWALPrune* option functions

## Decisions Made
- Followed D-04 wrapper style from Plan 01 consistently across all 5 files
- compaction.go contains zero function declarations because compaction has no package-level functions (all are methods on Server/Embedded that auto-forward via type alias)
- wal_prune.go uses dual import block (`time` + `internal/runtime`) for WithWALPruneMaxAge's time.Duration parameter

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all facade files are fully wired to internal/runtime.

## Next Phase Readiness
- Full root facade is complete (Plans 01 + 02): 9 files, 54 type aliases, all public symbols re-exported
- Phase 04 (test reorganization) is unblocked for root package test coverage
- Phase 05 (CI/docs) is unblocked for documentation updates

## Self-Check: PASSED

- All 5 facade files exist at repo root
- Commit 15413fc found in git log
- Commit 0468775 found in git log
- SUMMARY.md created at expected path

---
*Phase: 03-root-facade*
*Completed: 2026-03-20*
