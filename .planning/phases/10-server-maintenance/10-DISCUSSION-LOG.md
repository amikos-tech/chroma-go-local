# Phase 10: Server Maintenance - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-27
**Phase:** 10-server-maintenance
**Areas discussed:** Stop-restart orchestration, Session invalidation, Test strategy, Concurrency/locking, API symmetry, Error model

---

## Stop-restart orchestration

| Option | Description | Selected |
|--------|-------------|----------|
| Core utility class | MaintenanceExecutor in core (like BackupExecutor), backends inject lambdas | ✓ |
| Callback slots on ServerSession | Add 5 maintenance callback slots, each backend handles stop/embed/restart inline | |
| You decide | Claude picks | |

**User's choice:** Core utility class
**Notes:** None

---

| Option | Description | Selected |
|--------|-------------|----------|
| Single generic method | One execute() parameterized by request/result types | ✓ |
| Per-operation methods | Separate executeRebuild(), executeCompaction(), etc. | |
| You decide | Claude picks | |

**User's choice:** Single generic method
**Notes:** None

---

| Option | Description | Selected |
|--------|-------------|----------|
| Return result + throw | Follow Go pattern: return result AND signal restart failure | ✓ |
| Always throw on restart failure | Throw regardless, lose the result | |
| You decide | Claude picks | |

**User's choice:** Return result + throw
**Notes:** Later refined in error model discussion — MaintenanceResult with error field instead of throwing

---

## Session invalidation

| Option | Description | Selected |
|--------|-------------|----------|
| Keep existing session | Handle becomes mutable, session stays valid after restart | |
| Return new session (like backup) | New session, old invalidated, handle stays final | ✓ |

**User's choice:** Initially selected "Keep existing session" but reconsidered after discussing downsides of mutable handles (thread safety risk, callback staleness, invariant weakening). Changed to "New session, like backup."
**Notes:** User raised the concern that mutable handles are more dangerous than immutable sessions with replacement.

---

| Option | Description | Selected |
|--------|-------------|----------|
| New MaintenanceResult<R, S> | Generic over result and session types | ✓ |
| Reuse BackupResult<S> style | Similar pattern with different generics | |
| You decide | Claude picks | |

**User's choice:** New MaintenanceResult<R, S>
**Notes:** None

---

## Test strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Smoke + restart verification | Smoke tier, verify server accessible after restart | |
| Smoke only, no restart check | Just verify op result | |
| Full data-seeded tests | Create collections, add data, run maintenance, verify | ✓ |

**User's choice:** Full data-seeded + smoke tests (custom response combining smoke and data-seeded)
**Notes:** User explicitly wanted full data-seeded tests despite Phase 8 deferring them

---

| Option | Description | Selected |
|--------|-------------|----------|
| HTTP client | java.net.http.HttpClient to call Chroma REST API | ✓ |
| Official Chroma Java client | chromadb-java-client as test dependency | |
| You decide | Claude picks | |

**User's choice:** HTTP client
**Notes:** No external dependencies

---

| Option | Description | Selected |
|--------|-------------|----------|
| Server alive + result fields | Verify HTTP response + result fields + collection exists | ✓ |
| Full data integrity | Verify exact document contents preserved | |
| You decide | Claude picks | |

**User's choice:** Server alive + result fields
**Notes:** None

---

## Concurrency / locking

| Option | Description | Selected |
|--------|-------------|----------|
| Share backupLock | All stop/restart ops serialize through existing lock | ✓ |
| Separate maintenanceLock | Independent lock for maintenance | |
| You decide | Claude picks | |

**User's choice:** Share backupLock
**Notes:** Matches Go's backupMu pattern

---

## API symmetry with EmbeddedSession

| Option | Description | Selected |
|--------|-------------|----------|
| Asymmetry is fine | Embedded returns result, Server returns MaintenanceResult | ✓ |
| Unify both to MaintenanceResult | Both return MaintenanceResult with same/new session | |
| You decide | Claude picks | |

**User's choice:** Asymmetry is fine
**Notes:** Different return types reflect real behavioral differences

---

## MaintenanceExecutor error model

| Option | Description | Selected |
|--------|-------------|----------|
| MaintenanceResult with error field | result(), session(), restartError() — no exception thrown | ✓ |
| Custom exception wrapping result | Throw ServerRestartException carrying the result | |
| Always throw on restart failure | Throw, lose the result | |
| You decide | Claude picks | |

**User's choice:** MaintenanceResult with error field
**Notes:** Refines the earlier "return result + throw" decision — result is never lost, no exception forced on caller

---

## Claude's Discretion

- Exact MaintenanceExecutor.execute() method signature and generic bounds
- HTTP test helper class structure for data seeding
- Chroma REST API endpoint paths for collection/document operations
- Order of implementation

## Deferred Ideas

None — discussion stayed within phase scope
