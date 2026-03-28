---
phase: 10-server-maintenance
plan: 01
subsystem: api
tags: [java, ffi, maintenance, server-lifecycle, builder-pattern]

requires:
  - phase: 08-embedded-maintenance
    provides: EmbeddedSession maintenance methods (rebuild, compact, WAL prune)
  - phase: 09-backup-api
    provides: BackupExecutor pattern, ServerSession backup callback wiring
provides:
  - MaintenanceResult generic result container for server maintenance operations
  - MaintenanceExecutor stop-embed-op-restart orchestration utility
  - ServerSession 5 maintenance callback slots with lock-protected delegation
  - JNA and Panama backend wiring of maintenance callbacks
affects: [10-server-maintenance]

tech-stack:
  added: []
  patterns: [MaintenanceExecutor stop-embed-op-restart pattern, MaintenanceResult partial-failure container]

key-files:
  created:
    - java/core/src/main/java/tech/amikos/chroma/local/core/MaintenanceResult.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/MaintenanceExecutor.java
  modified:
    - java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/ServerSessionTest.java
    - java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java
    - java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java

key-decisions:
  - "MaintenanceResult preserves operation result even on partial failure (restart error) per D-03"
  - "MaintenanceExecutor.execute() mirrors Go rebuild.go error matrix with Java exception semantics"
  - "Server maintenance methods invalidate session after callback (closed.set(true)) matching backup pattern"

patterns-established:
  - "MaintenanceExecutor: stateless utility with stop-embed-op-restart lifecycle for all 5 server maintenance operations"
  - "ServerSession 12-param constructor: 7 original + 5 maintenance callbacks injected by backends"

requirements-completed: [SMNT-01, SMNT-02, SMNT-03]

duration: 6min
completed: 2026-03-28
---

# Phase 10 Plan 01: Server Maintenance Core Types and Backend Wiring Summary

**MaintenanceResult/MaintenanceExecutor core types with stop-embed-op-restart error matrix, ServerSession 5 maintenance callbacks, and JNA/Panama backend wiring**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-28T09:27:06Z
- **Completed:** 2026-03-28T09:32:43Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- MaintenanceResult<R,S> carries operation result, session, and optional restart error for partial failure handling
- MaintenanceExecutor.execute() implements Go's rebuild.go error matrix (stop server, start embedded, run op, close embedded, restart server)
- ServerSession expanded from 7 to 12 constructor params with 5 maintenance callback slots
- Both JNA and Panama backends wire callbacks through MaintenanceExecutor in doStartServer()

## Task Commits

Each task was committed atomically:

1. **Task 1: Create MaintenanceResult and MaintenanceExecutor core types** - `a6badbd` (feat)
2. **Task 2: Expand ServerSession with maintenance callbacks and wire JNA/Panama backends** - `bad15e0` (feat)

## Files Created/Modified
- `java/core/.../MaintenanceResult.java` - Generic result container with result(), session(), restartError()
- `java/core/.../MaintenanceExecutor.java` - Stop-embed-op-restart orchestration matching Go error matrix
- `java/core/.../ServerSession.java` - 12-param constructor, 5 lock-protected maintenance methods returning MaintenanceResult
- `java/core/.../ServerSessionTest.java` - Updated for new constructor signature
- `java/jna/.../JnaChromaRuntime.java` - doStartServer wires 5 MaintenanceExecutor callbacks
- `java/panama/.../PanamaChromaRuntime.java` - doStartServer wires 5 MaintenanceExecutor callbacks

## Decisions Made
- MaintenanceResult preserves operation result even on partial failure (restart error) per D-03 -- result is never lost when operation succeeds
- MaintenanceExecutor mirrors Go rebuild.go:184-233 error matrix with Java exception semantics (suppressed exceptions for secondary failures)
- Server maintenance methods invalidate session after callback (closed.set(true)) matching the backup pattern per D-08

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated ServerSessionTest for 12-param constructor**
- **Found during:** Task 2
- **Issue:** Existing tests used 7-param ServerSession constructor; compilation failed after expansion to 12 params
- **Fix:** Added stub maintenance callbacks to all test constructor calls, updated tests that relied on UnsupportedOperationException
- **Files modified:** java/core/src/test/java/tech/amikos/chroma/local/core/ServerSessionTest.java
- **Verification:** `gradle --no-daemon :core:check` passes
- **Committed in:** bad15e0 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Test update was necessary consequence of constructor change. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 5 server maintenance operations are wired and compilable
- Ready for Plan 02: integration tests for server maintenance operations

---
*Phase: 10-server-maintenance*
*Completed: 2026-03-28*
