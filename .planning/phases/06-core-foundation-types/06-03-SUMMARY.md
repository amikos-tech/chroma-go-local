---
phase: 06-core-foundation-types
plan: 03
subsystem: api
tags: [java, ffi-safety, reentrant-lock, template-method, server-session, abstract-class]

requires:
  - phase: 06-core-foundation-types/01
    provides: "Result POJOs (RebuildCollectionResult, CompactionResult, WALPruneResult, BackupManifest) and JsonUtil"
provides:
  - "AbstractChromaRuntime with global static ReentrantLock and FFI template methods"
  - "Three abstract methods (readBorrowedString, readOwnedString, readLastError) for backend contract"
  - "ChromaRuntime interface extended with startServer(String) returning ServerSession"
  - "ServerSession with lifecycle callbacks, accessor callbacks, and maintenance method stubs"
affects: [07-server-lifecycle, 08-embedded-maintenance, 09-backup-api, 10-server-maintenance]

tech-stack:
  added: []
  patterns: [abstract-template-method-with-lock, callback-slot-session, two-step-close-try-finally]

key-files:
  created:
    - java/core/src/main/java/tech/amikos/chroma/local/core/AbstractChromaRuntime.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/AbstractChromaRuntimeTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/ServerSessionTest.java
  modified:
    - java/core/src/main/java/tech/amikos/chroma/local/core/ChromaRuntime.java

key-decisions:
  - "ServerSession created alongside AbstractChromaRuntime in Task 1 commit due to ChromaRuntime interface dependency"
  - "Maintenance methods throw UnsupportedOperationException with phase reference rather than using callback slots -- simpler for Phase 6, Phases 7-10 will replace with actual wiring"
  - "CompactCollectionRequest and CompactAllRequest created as minimal stubs to unblock ServerSession compilation during parallel execution with plan 06-02"

patterns-established:
  - "AbstractChromaRuntime template pattern: static ReentrantLock + callFfiHandle/callFfiJson/callFfiVoid/callFfiBorrowedString methods that acquire lock, invoke FFI, check errors, release lock"
  - "ServerSession callback-slot pattern: final class with functional interface fields (LongConsumer, LongToIntFunction, LongFunction) injected at construction for lifecycle and accessor delegation"
  - "Two-step close pattern: stop in try block, free in finally block, idempotent via AtomicBoolean.compareAndSet"

requirements-completed: [FOUND-01, FOUND-05, FOUND-06]

duration: 5min
completed: 2026-03-22
---

# Phase 06 Plan 03: FFI Safety Infrastructure and ServerSession Summary

**AbstractChromaRuntime with global ReentrantLock and FFI template methods plus ServerSession with callback-slot lifecycle, accessor delegation, and maintenance method stubs for Phase 7-10 wiring**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-22T18:17:44Z
- **Completed:** 2026-03-22T18:22:26Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Created AbstractChromaRuntime with static ReentrantLock providing callFfiHandle, callFfiJson, callFfiVoid, and callFfiBorrowedString template methods mirroring Go's FFI lock pattern
- Extended ChromaRuntime interface with startServer(String configYaml) returning ServerSession
- Created ServerSession with full public API surface: lifecycle (stop+free), accessors (port, address, persistPath, url), and maintenance stubs (rebuild, compact, prune, backup)
- 28 tests across 2 test classes verifying lock serialization, error propagation, constructor validation, close idempotency, and ensureOpen guards

## Task Commits

Each task was committed atomically:

1. **Task 1: AbstractChromaRuntime + ChromaRuntime + ServerSession** - `8d13286` (feat)
2. **Task 2: ServerSession tests** - `67381c7` (test)

## Files Created/Modified
- `java/core/src/main/java/tech/amikos/chroma/local/core/AbstractChromaRuntime.java` - FFI safety base class with global static ReentrantLock and template methods
- `java/core/src/main/java/tech/amikos/chroma/local/core/ChromaRuntime.java` - Added startServer(String configYaml) returning ServerSession
- `java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java` - Server session with callback slots for lifecycle, accessors, and maintenance stubs
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactCollectionRequest.java` - Minimal stub (plan 06-02 will replace)
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactAllRequest.java` - Minimal stub (plan 06-02 will replace)
- `java/core/src/test/java/tech/amikos/chroma/local/core/AbstractChromaRuntimeTest.java` - 9 tests for template methods, error propagation, and lock serialization
- `java/core/src/test/java/tech/amikos/chroma/local/core/ServerSessionTest.java` - 19 tests for constructor validation, accessors, close semantics, and maintenance guards

## Decisions Made
- ServerSession was created as part of Task 1 commit rather than Task 2 because ChromaRuntime interface requires it as a return type, and the test's TestChromaRuntime subclass must implement startServer()
- Maintenance methods use simple UnsupportedOperationException throws instead of null callback slots -- reduces complexity and avoids premature callback interface design that Phases 7-10 may revise
- Created CompactCollectionRequest and CompactAllRequest as minimal stubs during parallel execution since plan 06-02 hadn't created them yet

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created CompactCollectionRequest and CompactAllRequest stubs**
- **Found during:** Task 1 (ServerSession requires these types for maintenance method signatures)
- **Issue:** Plan 06-02 (running in parallel) hadn't created these option types yet, blocking ServerSession compilation
- **Fix:** Created minimal stub classes with private constructor and toJson() method -- plan 06-02 will overwrite with full builder implementations
- **Files modified:** CompactCollectionRequest.java, CompactAllRequest.java
- **Verification:** core:compileJava succeeds
- **Committed in:** 8d13286 (Task 1 commit)

**2. [Rule 3 - Blocking] Combined TDD RED+GREEN phases due to parallel compilation interference**
- **Found during:** Task 1 RED phase
- **Issue:** Plan 06-02's test files (ServerConfigBuilderTest, EmbeddedConfigBuilderTest, WALPruneOptionsTest) reference classes not yet created, causing full test compilation to fail. Cannot isolate just AbstractChromaRuntimeTest compilation in Gradle
- **Fix:** Created implementation alongside tests in same commit since RED phase cannot compile in isolation
- **Verification:** AbstractChromaRuntimeTest and ServerSessionTest both pass when run individually via --tests filter

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both deviations caused by parallel execution contention with plan 06-02. No scope creep -- stubs are minimal and will be replaced.

## Issues Encountered
- Full `core:build` and `core:test` fail due to plan 06-02's test files referencing not-yet-created classes (ServerConfigBuilder, EmbeddedConfigBuilder). This is expected parallel execution contention and will resolve when plan 06-02 completes.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- AbstractChromaRuntime ready for JNA and Panama backends to extend (Phase 7+)
- ServerSession ready for backend construction with callback wiring (Phase 7)
- ChromaRuntime interface complete with both startEmbedded and startServer methods
- All core module main sources compile cleanly (core:compileJava exits 0)
- Maintenance method stubs in ServerSession define the API surface for Phases 8-9

## Self-Check: PASSED

- All 7 created files verified on disk
- Both commits (8d13286, 67381c7) verified in git log
- core:compileJava exits 0 with all main sources compiling
- Zero JNA/Panama imports in core module source

---
*Phase: 06-core-foundation-types*
*Completed: 2026-03-22*
