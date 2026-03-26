---
phase: 08-embedded-maintenance
plan: 01
subsystem: api
tags: [java, ffi, jna, panama, embedded, maintenance, callback-slots]

requires:
  - phase: 06-core-foundation-types
    provides: "Core types (option builders, result POJOs, AbstractChromaRuntime, EmbeddedSession)"
  - phase: 07-server-lifecycle
    provides: "AbstractChromaRuntime retrofit, callFfiJson pattern, ServerSession callback slots"
provides:
  - "EmbeddedSession with 5 BiFunction callback slots for maintenance operations"
  - "JNA backend with 5 chroma_embedded_* symbol bindings wired to EmbeddedSession"
  - "Panama backend with 5 chroma_embedded_* MethodHandle bindings wired to EmbeddedSession"
  - "7 public maintenance methods on EmbeddedSession (rebuild, compact, prune WAL)"
affects: [08-02, 09-backup, 10-server-maintenance]

tech-stack:
  added: []
  patterns: [BiFunction callback slots for session maintenance, callFfiJson lambda injection at construction]

key-files:
  created: []
  modified:
    - java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedSession.java
    - java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java
    - java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/EmbeddedSessionTest.java

key-decisions:
  - "BiFunction<Long, String, T> callback slots for all 5 maintenance operations"
  - "EmbeddedSession constructor expanded from 2 to 7 parameters"

patterns-established:
  - "BiFunction callback slot injection: backends create callFfiJson lambdas at EmbeddedSession construction time"
  - "ensureOpen guard on all EmbeddedSession public methods"

requirements-completed: [EMNT-01, EMNT-02, EMNT-03, EMNT-04]

duration: 3min
completed: 2026-03-26
---

# Phase 08 Plan 01: Embedded Maintenance Wiring Summary

**EmbeddedSession expanded with 5 BiFunction callback slots for maintenance ops, wired in both JNA and Panama backends via callFfiJson lambdas**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-26T12:53:32Z
- **Completed:** 2026-03-26T12:57:08Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- EmbeddedSession expanded with 7-parameter constructor accepting 5 maintenance callback slots
- 7 public maintenance methods added (rebuildCollection x2, compactCollection, compactAll, pruneCollectionWAL, pruneAllWAL)
- JNA backend binds 5 new chroma_embedded_* symbols and injects callFfiJson lambdas
- Panama backend binds 5 new MethodHandle fields with eager symbol resolution and Arena-scoped lambdas
- All three modules compile and pass lint checks

## Task Commits

Each task was committed atomically:

1. **Task 1: Expand EmbeddedSession with callback slots and maintenance methods** - `09c32c9` (feat)
2. **Task 2: Wire JNA and Panama backends with symbol bindings and lambdas** - `909c187` (feat)

## Files Created/Modified
- `java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedSession.java` - Expanded with 5 BiFunction callback slots, 7 maintenance methods, ensureOpen guard
- `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` - 5 new JnaBindings symbol declarations, callFfiJson lambda injection
- `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` - 5 new Ffi MethodHandle fields, Arena-scoped lambda injection
- `java/core/src/test/java/tech/amikos/chroma/local/core/EmbeddedSessionTest.java` - Updated to use 7-parameter constructor

## Decisions Made
- Used `BiFunction<Long, String, T>` for all 5 callback slots -- typed per result, keeping core module FFI-free
- Expanded constructor from 2 to 7 parameters with null checks for all callbacks
- Added `ensureOpen()` private method matching ServerSession pattern

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated EmbeddedSessionTest to use new constructor signature**
- **Found during:** Task 2 (backend wiring verification)
- **Issue:** Existing EmbeddedSessionTest used old 2-parameter constructor, causing compilation failure in `:core:check`
- **Fix:** Added `create()` helper method with no-op callback stubs, updated all 6 test methods
- **Files modified:** java/core/src/test/java/tech/amikos/chroma/local/core/EmbeddedSessionTest.java
- **Verification:** `:core:check` passes with all 6 tests green
- **Committed in:** 909c187 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Test update was necessary -- constructor signature change is a compile-time break. No scope creep.

## Issues Encountered
- `cargo` not available in sandbox environment so `make test-java` could not run, but direct `gradle :jna:test :panama:test` confirmed all tests pass

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- EmbeddedSession maintenance methods are wired and ready for integration testing (Plan 08-02)
- Both backends compile and pass lint -- ready for smoke and data-seeded tests
- ServerSession maintenance stubs remain as UnsupportedOperationException for Phase 10

## Self-Check: PASSED

All 5 files verified present. Both task commits (09c32c9, 909c187) verified in git log.

---
*Phase: 08-embedded-maintenance*
*Completed: 2026-03-26*
