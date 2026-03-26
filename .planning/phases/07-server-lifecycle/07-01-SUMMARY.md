---
phase: 07-server-lifecycle
plan: 01
subsystem: api
tags: [java, ffi, jna, panama, abstract-class, thread-safety]

requires:
  - phase: 06-core-foundation-types
    provides: AbstractChromaRuntime base class with FFI lock, ServerSession with callback slots, EmbeddedSession
provides:
  - JNA backend extending AbstractChromaRuntime with lock-protected FFI calls
  - Panama backend extending AbstractChromaRuntime with lock-protected FFI calls
  - Thread-safe FFI call serialization in both backends
  - ServerSession wiring with method-reference callbacks in both backends
affects: [08-embedded-maintenance, 09-backup-api, 10-server-maintenance]

tech-stack:
  added: []
  patterns: [template-method for FFI calls, string ownership (borrowed vs owned), lambda-wrapped MethodHandle.invokeExact]

key-files:
  created: []
  modified:
    - java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java
    - java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/AbstractChromaRuntime.java

key-decisions:
  - "serverFree and embeddedFree bypass callFfiVoid -- free operations should not check lastError or acquire FFI lock since they run in finally blocks"
  - "serverPort uses callFfiHandle with negative-to-0L mapping -- port 0 treated as error (valid for started servers)"
  - "Panama MethodHandle.invokeExact wrapped in try-catch inside lambdas -- checked Throwable cannot propagate through LongSupplier/Runnable interfaces"

patterns-established:
  - "Template method pattern: all FFI calls route through callFfiHandle, callFfiBorrowedString, or callFfiVoid for lock-protected execution"
  - "String ownership: readBorrowedString for static/server-owned data (version, address, persistPath), readOwnedString+free for error strings"
  - "Lambda wrapping: Panama MethodHandle.invokeExact calls wrapped with try-catch converting checked Throwable to ChromaException"

requirements-completed: [SRVR-01, SRVR-02, SRVR-03]

duration: 3min
completed: 2026-03-26
---

# Phase 7 Plan 01: Backend AbstractChromaRuntime Retrofit Summary

**Both JNA and Panama backends retrofitted to extend AbstractChromaRuntime with lock-protected FFI template methods and correct string ownership semantics**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-26T08:21:35Z
- **Completed:** 2026-03-26T08:25:19Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- JnaChromaRuntime now extends AbstractChromaRuntime, replacing 179 lines of inline FFI handling with 57 lines using template methods
- PanamaChromaRuntime now extends AbstractChromaRuntime, replacing 147 lines with 113 lines using template methods
- All FFI calls in both backends are now serialized through a shared ReentrantLock via the base class
- ServerSession correctly wired with method-reference callbacks for stop, free, port, address, and persistPath

## Task Commits

Each task was committed atomically:

1. **Task 1: Retrofit JnaChromaRuntime** - `2b7defc` (feat)
2. **Task 2: Retrofit PanamaChromaRuntime** - `15035ec` (feat)

## Files Created/Modified
- `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` - JNA backend now extends AbstractChromaRuntime with 3 abstract method implementations and template-method-routed FFI calls
- `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` - Panama backend now extends AbstractChromaRuntime with 3 abstract method implementations, lambda-wrapped MethodHandle calls, and preserved Windows arena close workaround
- `java/core/src/main/java/tech/amikos/chroma/local/core/AbstractChromaRuntime.java` - Updated comment from "Wired in Phase 8" to reflect current wired state

## Decisions Made
- serverFree and embeddedFree bypass callFfiVoid to avoid acquiring the FFI lock in finally blocks and checking lastError on free operations
- serverPort maps negative return values to 0L, which callFfiHandle treats as error (port 0 is never valid for a started server)
- Panama's MethodHandle.invokeExact checked Throwable is caught inside lambdas and re-thrown as ChromaException to satisfy LongSupplier/Runnable functional interfaces

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Updated AbstractChromaRuntime stale comment**
- **Found during:** Task 2
- **Issue:** AbstractChromaRuntime had comment "Wired in Phase 8" which is now incorrect since both backends are wired
- **Fix:** Updated comment to "Base class for JNA/Panama runtimes -- provides lock-protected FFI call templates"
- **Files modified:** java/core/src/main/java/tech/amikos/chroma/local/core/AbstractChromaRuntime.java
- **Verification:** File compiles cleanly
- **Committed in:** 15035ec (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Minor comment correction for accuracy. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Both backends now extend AbstractChromaRuntime with thread-safe FFI serialization
- ServerSession is fully wired with method-reference callbacks
- Ready for Phase 8 (Embedded Maintenance) to add maintenance operations using callFfiJson and callFfiVoid patterns
- Ready for Phase 9 (Backup API) and Phase 10 (Server Maintenance) which depend on server lifecycle

## Self-Check: PASSED

- All files verified present on disk
- Commit 2b7defc (Task 1) verified in git log
- Commit 15035ec (Task 2) verified in git log

---
*Phase: 07-server-lifecycle*
*Completed: 2026-03-26*
