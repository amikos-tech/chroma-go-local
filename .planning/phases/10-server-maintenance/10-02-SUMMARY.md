---
phase: 10-server-maintenance
plan: 02
subsystem: api
tags: [java, ffi, maintenance, integration-tests, server-lifecycle]

requires:
  - phase: 10-server-maintenance
    plan: 01
    provides: MaintenanceResult, MaintenanceExecutor, ServerSession maintenance callbacks, JNA/Panama backend wiring
provides:
  - JNA integration tests for all 5 server maintenance operations with data seeding
  - Panama integration tests mirroring JNA for backend parity verification
affects: []

tech-stack:
  added: []
  patterns: [HTTP-based data seeding for server maintenance tests, stop-embed-op-restart verification via heartbeat]

key-files:
  created:
    - java/jna/src/test/java/tech/amikos/chroma/local/jna/JnaServerMaintenanceTest.java
    - java/panama/src/test/java/tech/amikos/chroma/local/panama/PanamaServerMaintenanceTest.java
  modified: []

key-decisions:
  - "Split null-option rejection tests into one server per assertion due to ServerSession closing on IllegalArgumentException in catch block"
  - "HTTP data seeding via Chroma v2 REST API for collection creation and verification in server maintenance tests"

patterns-established:
  - "Server maintenance test pattern: start server, waitForReady, createCollection via HTTP, run maintenance op, verify heartbeat on new session, verify collection survives"

requirements-completed: [SMNT-01, SMNT-02, SMNT-03, SMNT-04]

duration: 6min
completed: 2026-03-28
---

# Phase 10 Plan 02: Server Maintenance Integration Tests Summary

**Data-seeded integration tests for all 5 server maintenance operations in both JNA and Panama backends with HTTP-based collection creation, heartbeat verification, and error path coverage**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-28T09:35:18Z
- **Completed:** 2026-03-28T09:41:20Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- 11 JNA tests covering rebuild, compact, compactAll, pruneWAL, pruneAllWAL with data seeding + 1 throws-after-close + 5 rejects-null-options
- 11 Panama tests mirroring JNA exactly (only class/package/runtime type differ)
- HTTP helpers: waitForReady (heartbeat polling), createCollection (v2 REST API), verifyCollectionExists
- Each data-seeded test creates a collection via HTTP, runs maintenance op (stop-embed-op-restart), verifies new session heartbeat and collection survival
- Full test suite (`make test-java`) passes including all prior phase tests

## Task Commits

Each task was committed atomically:

1. **Task 1: Create JNA server maintenance integration tests with data seeding** - `3b11294` (test)
2. **Task 2: Create Panama server maintenance integration tests (mirror of JNA)** - `6aeb5fd` (test)

## Files Created/Modified
- `java/jna/.../JnaServerMaintenanceTest.java` - 11 tests: 5 data-seeded maintenance ops, 1 throws-after-close, 5 rejects-null per-operation
- `java/panama/.../PanamaServerMaintenanceTest.java` - Exact structural mirror of JNA with PanamaChromaRuntime substitution

## Decisions Made
- Split `serverMaintenanceRejectsNullOptions` into 5 individual tests (one per operation) because ServerSession catch block closes session on IllegalArgumentException, making sequential null checks fail with IllegalStateException
- Used HTTP-based data seeding via Chroma v2 REST API (`/api/v2/tenants/default_tenant/databases/default_database/collections`) for collection creation and verification

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Split null-option rejection tests per operation**
- **Found during:** Task 1
- **Issue:** Plan specified a single `serverMaintenanceRejectsNullOptions` test with 5 assertions. ServerSession.rebuildCollection() catch block runs `closed.set(true)` on IllegalArgumentException (a RuntimeException), so second assertion gets IllegalStateException
- **Fix:** Split into 5 individual tests: serverRebuildRejectsNullOptions, serverCompactCollectionRejectsNullOptions, serverCompactAllRejectsNullOptions, serverPruneWALRejectsNullOptions, serverPruneAllWALRejectsNullOptions
- **Files modified:** JnaServerMaintenanceTest.java, PanamaServerMaintenanceTest.java
- **Verification:** All 11 tests pass in both backends

---

**Total deviations:** 1 auto-fixed (1 bug workaround)
**Impact on plan:** Test count increased from 7 to 11 per backend. Same coverage, better isolation.

## Known Stubs

None - all tests exercise real FFI through stop-embed-op-restart lifecycle.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Self-Check: PASSED

---
*Phase: 10-server-maintenance*
*Completed: 2026-03-28*
