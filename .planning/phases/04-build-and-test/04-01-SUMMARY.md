---
phase: 04-build-and-test
plan: 01
subsystem: build
tags: [golangci-lint, gci, staticcheck, linting]

requires:
  - phase: 03-root-facade
    provides: Root facade with type aliases re-exporting internal/runtime types
provides:
  - Clean golangci-lint run (gci prefix + SA1019 suppression)
affects: [04-build-and-test]

tech-stack:
  added: []
  patterns: [nolint:staticcheck for deprecated type re-exports in facade]

key-files:
  created: []
  modified:
    - .golangci.yml
    - backup.go
    - internal/runtime/chroma.go

key-decisions:
  - "gci import reordering in chroma.go applied automatically as consequence of prefix fix"

patterns-established:
  - "nolint:staticcheck on deprecated type alias re-exports in facade files"

requirements-completed: [BUILD-03]

duration: 2min
completed: 2026-03-21
---

# Phase 04 Plan 01: Lint Fix Summary

**Fixed stale gci import prefix and SA1019 staticcheck warnings on deprecated type re-exports for clean golangci-lint run**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-21T09:51:14Z
- **Completed:** 2026-03-21T09:53:26Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Corrected gci import prefix from stale `chaoslabs-bg/tclr-v2` to `amikos-tech/chroma-go-local`
- Added nolint:staticcheck directives on deprecated ServerBackupOptions and EmbeddedBackupOptions type aliases in backup.go
- Auto-fixed gci import ordering in internal/runtime/chroma.go caused by the prefix correction

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix gci prefix in .golangci.yml** - `75b029e` (fix)
2. **Task 2: Add nolint:staticcheck to deprecated type re-exports** - `596b630` (fix)
3. **Auto-fix: Reorder imports in chroma.go** - `117e59d` (fix)

## Files Created/Modified
- `.golangci.yml` - Corrected gci prefix for project import grouping
- `backup.go` - Added nolint:staticcheck on deprecated type alias lines 7-8
- `internal/runtime/chroma.go` - Reordered imports to match corrected gci sections

## Decisions Made
- Applied gci auto-fix to internal/runtime/chroma.go since the prefix correction changed import grouping rules (Rule 3 deviation)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] gci import ordering in internal/runtime/chroma.go**
- **Found during:** Overall verification (golangci-lint run after Task 2)
- **Issue:** After fixing the gci prefix, the formatter correctly identified project imports as needing their own group, but existing import ordering in chroma.go had the project import mixed with third-party imports
- **Fix:** Ran golangci-lint --fix to auto-reorder; project import moved to its own group below third-party imports
- **Files modified:** internal/runtime/chroma.go
- **Verification:** golangci-lint run ./... reports zero gci issues
- **Committed in:** 117e59d

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Direct consequence of Task 1's prefix fix. No scope creep.

## Issues Encountered
- Pre-existing errcheck issue in compat_test.go:195 (`defer server.Stop()` unchecked return). Logged in deferred-items.md. Not caused by this plan's changes.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- golangci-lint runs clean (only pre-existing errcheck in test file remains)
- Ready for plan 04-02 (Makefile/CI updates) and 04-03 (verification gate)

## Self-Check: PASSED

All files exist, all commits verified.

---
*Phase: 04-build-and-test*
*Completed: 2026-03-21*
