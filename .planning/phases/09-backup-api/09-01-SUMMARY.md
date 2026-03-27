---
phase: 09-backup-api
plan: 01
subsystem: api
tags: [backup, filesystem, sha256, java, jna, panama, session-lifecycle]

requires:
  - phase: 06-core-foundation-types
    provides: BackupOptions, BackupManifest, BackupFileMetadata, JsonUtil, AbstractChromaRuntime
  - phase: 07-server-lifecycle
    provides: ServerSession with callback slot pattern
  - phase: 08-embedded-maintenance
    provides: EmbeddedSession with 7-param constructor and callback slot pattern
provides:
  - BackupResult generic type wrapping manifest and new session
  - BackupExecutor core algorithm (close-copy-manifest-restart)
  - Backup callback slots in EmbeddedSession and ServerSession
  - JNA and Panama backup lambda construction
affects: [10-server-maintenance]

tech-stack:
  added: []
  patterns: [backup-executor-utility, session-invalidation-via-new-result]

key-files:
  created:
    - java/core/src/main/java/tech/amikos/chroma/local/core/BackupResult.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/BackupExecutor.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/BackupResultTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/BackupExecutorTest.java
  modified:
    - java/core/src/main/java/tech/amikos/chroma/local/core/BackupManifest.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/BackupFileMetadata.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedSession.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/EmbeddedSessionTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/ServerSessionTest.java
    - java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java
    - java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java

key-decisions:
  - "BackupExecutor made public (not package-private) for cross-module access from JNA/Panama backends"
  - "Persist path extracted from config YAML at embedded session creation time via SnakeYAML parsing"

patterns-established:
  - "BackupExecutor utility: centralized filesystem algorithm with close/restart callback injection"
  - "BackupResult<S>: generic result type for operations that invalidate and recreate sessions"

requirements-completed: [BKUP-01, BKUP-02, BKUP-03]

duration: 8min
completed: 2026-03-27
---

# Phase 9 Plan 1: Backup API Summary

**BackupExecutor with SHA-256 directory copy, JSON manifest, and session callback wiring for both JNA and Panama backends**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-27T05:30:53Z
- **Completed:** 2026-03-27T05:38:53Z
- **Tasks:** 2
- **Files modified:** 13

## Accomplishments
- Core backup algorithm replicating Go's close-copy-manifest-restart pattern in pure Java
- BackupResult generic type enabling session replacement after backup invalidates the old handle
- Both JNA and Panama backends wired with backup lambdas for embedded and server modes
- Mode validation (embedded rejects leaveStopped, server rejects leaveClosed) matching Go behavior

## Task Commits

Each task was committed atomically:

1. **Task 1: Create BackupResult and BackupExecutor core types** - `305153d` (feat)
2. **Task 2: Wire backup callback slots into sessions and backends** - `e47551a` (feat)

## Files Created/Modified
- `java/core/.../BackupResult.java` - Generic result type wrapping BackupManifest + new session
- `java/core/.../BackupExecutor.java` - Core backup algorithm: validate, close, copy with SHA-256, write manifest, restart
- `java/core/.../BackupManifest.java` - Added all-args package-private constructor for BackupExecutor
- `java/core/.../BackupFileMetadata.java` - Added all-args package-private constructor
- `java/core/.../EmbeddedSession.java` - Expanded to 8-param constructor with backup callback slot
- `java/core/.../ServerSession.java` - Expanded to 7-param constructor, backup stub replaced with real implementation
- `java/jna/.../JnaChromaRuntime.java` - Backup lambda construction in doStartEmbedded and doStartServer
- `java/panama/.../PanamaChromaRuntime.java` - Backup lambda construction in doStartEmbedded and doStartServer
- `java/core/.../BackupResultTest.java` - Unit tests for BackupResult type
- `java/core/.../BackupExecutorTest.java` - Unit tests for backup algorithm (copy, manifest, validation)
- `java/core/.../EmbeddedSessionTest.java` - Updated for 8-param constructor, added backup null/close tests
- `java/core/.../ServerSessionTest.java` - Updated for 7-param constructor, added backup null/close tests

## Decisions Made
- BackupExecutor made public instead of package-private -- the plan specified package-private, but JNA and Panama backends in separate packages need to call BackupExecutor.execute(). Made class and key methods public.
- Persist path for embedded sessions extracted from config YAML using SnakeYAML at session creation time, supporting both top-level `persist_path` and nested `chroma.persist_path` keys.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Made BackupExecutor public for cross-module access**
- **Found during:** Task 2 (session wiring)
- **Issue:** Plan specified BackupExecutor as package-private, but JNA/Panama modules cannot access package-private classes from a different package
- **Fix:** Made BackupExecutor class and execute/extractPersistPath methods public
- **Files modified:** java/core/src/main/java/tech/amikos/chroma/local/core/BackupExecutor.java
- **Verification:** :jna:check and :panama:check compile successfully
- **Committed in:** e47551a (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary for cross-module compilation. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Backup API fully wired for both embedded and server modes
- Integration tests (BKUP-04) with real FFI are deferred to a future plan or the verifier
- Phase 10 (Server Maintenance) can proceed -- ServerSession backup is now functional

## Self-Check: PASSED

All created files verified on disk. All commit hashes (305153d, e47551a) found in git log.

---
*Phase: 09-backup-api*
*Completed: 2026-03-27*
