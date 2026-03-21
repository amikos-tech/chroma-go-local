---
phase: 04-build-and-test
plan: 02
subsystem: testing
tags: [go, compile-gate, smoke-tests, api-surface, testify]

requires:
  - phase: 03-root-facade
    provides: 9 facade files re-exporting all public API symbols from internal/runtime

provides:
  - Compile-time API surface gate covering all exported symbols (54 types, 37 functions, 14 constants, 5 error vars)
  - 9 behavioral smoke tests validating key entry points through the public import path

affects: [05-docs-cleanup]

tech-stack:
  added: []
  patterns: [external-test-package-api-gate, var-underscore-compile-gate]

key-files:
  created: [compat_test.go]
  modified: []

key-decisions:
  - "All 110 exported symbols referenced via var _ declarations for compile-time regression gate"
  - "9 behavioral smoke tests use top-level TestXxx functions with per-test Init calls"
  - "Option builder tests (backup, rebuild, WAL prune) do not require Init -- pure Go code"

patterns-established:
  - "External test package pattern: package chroma_test with named import chroma for API surface validation"
  - "Compile gate pattern: var _ T (no =) for types, var _ = X (with =) for functions/constants/variables"

requirements-completed: [TEST-02, TEST-03]

duration: 2min
completed: 2026-03-21
---

# Phase 04 Plan 02: Compat Test Summary

**Compile-time API surface gate and behavioral smoke tests for all 110 exported symbols via compat_test.go**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-21T09:51:29Z
- **Completed:** 2026-03-21T09:53:44Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Created compat_test.go at repo root with external test package (package chroma_test) proving public import path works
- Compile-time gate: 54 type declarations + 56 function/constant/error var references = 110 total symbol references
- 9 behavioral smoke tests covering Init, Version, VersionWithError, DefaultServerConfig, NewServer, DefaultEmbeddedConfig, and option builders for backup, rebuild, and WAL prune
- `go build ./...` and `go vet ./...` pass with all symbols resolved

## Task Commits

Each task was committed atomically:

1. **Task 1: Create compat_test.go with compile-time API surface gate** - `4fe8eee` (feat)

## Files Created/Modified
- `compat_test.go` - Compile-time API surface gate (110 var _ declarations) + 9 behavioral smoke tests

## Decisions Made
- Plan listed 87 symbols (40 types + 28 functions + 14 constants + 5 error vars) but actual facade inventory is 110 (54 types + 37 functions + 14 constants + 5 error vars). Followed the plan's exhaustive symbol list rather than the incorrect count.
- Used top-level TestXxx functions (not subtests) for clarity and simplicity
- Option builder tests (TestBackupOptionBuilders, TestRebuildOptionBuilders, TestWALPruneOptionBuilders) do not call Init since they exercise pure Go code with no FFI dependency

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected symbol counts from plan**
- **Found during:** Task 1
- **Issue:** Plan frontmatter and acceptance criteria stated 87 symbols (40 types, 28 functions) but the plan body's exhaustive symbol list contains 110 symbols (54 types, 37 functions, 14 constants, 5 error vars)
- **Fix:** Implemented all symbols from the plan's exhaustive list, which matches the actual facade files. The counts in the acceptance criteria were incorrect.
- **Files modified:** compat_test.go
- **Verification:** `go build ./...` passes -- all symbol references compile successfully
- **Committed in:** 4fe8eee

---

**Total deviations:** 1 auto-fixed (1 bug in plan counts)
**Impact on plan:** Corrected to match actual codebase. All facade symbols are covered. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Compile-time API surface gate is in place -- any symbol removal or rename will cause a build failure
- Behavioral smoke tests are ready to run via `make test` (requires Rust shim build)
- Phase 4 Plan 03 (Makefile/CI/lint fixes) can proceed independently

## Self-Check: PASSED

- [x] compat_test.go exists at repo root
- [x] Commit 4fe8eee exists in git log
- [x] 04-02-SUMMARY.md exists in phase directory

---
*Phase: 04-build-and-test*
*Completed: 2026-03-21*
