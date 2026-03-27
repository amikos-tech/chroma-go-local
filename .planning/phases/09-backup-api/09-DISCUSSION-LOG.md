# Phase 9: Backup API - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-27
**Phase:** 09-backup-api
**Areas discussed:** Backup logic location, Session restart after backup, BackupOptions mode validation, Test strategy, BackupResult type design, Restart callback shape, Public backup API

---

## Backup Logic Location

| Option | Description | Selected |
|--------|-------------|----------|
| Core module utility | Core owns full implementation as utility class. Sessions call directly. Core gains filesystem I/O. | |
| Backend callback slots | Same BiFunction pattern as maintenance ops. Both backends implement identical filesystem logic. | |
| Hybrid | Core owns filesystem copy/manifest logic. Backends inject close + restart callbacks. Zero duplication. | ✓ |

**User's choice:** Hybrid
**Notes:** Since backup has zero FFI calls (unlike Phase 8 maintenance ops), putting identical filesystem logic in both backends creates false symmetry. Core owns the algorithm, backends supply lifecycle hooks.

---

## Session Restart After Backup

| Option | Description | Selected |
|--------|-------------|----------|
| Return new session | backup() returns BackupResult(manifest, newSession). Old session invalidated. Sessions stay immutable. | ✓ |
| Callback with resetHandle | Backend restart callback returns new handle. Core calls package-private resetHandle(). | |
| Close-only backup | backup() always leaves session closed. No restart logic. Caller manages restart. | |
| Mutable handle | Change handle from final to AtomicLong. Same object survives like Go. | |

**User's choice:** Return new session
**Notes:** Preserves the immutable session pattern from Phases 6-8. Caller swaps their session reference.

---

## BackupOptions Mode Validation

| Option | Description | Selected |
|--------|-------------|----------|
| At backup() call time | Core backup algorithm validates mode-specific flags. BackupOptions unchanged from Phase 6. | ✓ |
| Two builder subclasses | EmbeddedBackupOptions and ServerBackupOptions. Compile-time safety. | |
| Builder with mode parameter | BackupOptions.Builder(dest, mode) validates at build(). | |

**User's choice:** At backup() call time
**Notes:** Matches Go's resolveBackupOptions pattern. Mode is only known at call time anyway, so this is the earliest possible validation point.

---

## Test Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| A: Smoke only | Start session, backup, assert manifest non-null. | |
| B: Smoke + filesystem verify | Sentinel file in temp dir, verify backup copies it, parse manifest JSON. | |
| B+D: Filesystem + edge cases | Sentinel file verify + manifest parse + error cases (bad dest, wrong mode flags). | ✓ |

**User's choice:** B+D: filesystem + edge cases
**Notes:** Full coverage without needing collection CRUD API. Sentinel file pattern mirrors Go's existing backup tests.

---

## BackupResult Type Design

| Option | Description | Selected |
|--------|-------------|----------|
| Generic BackupResult\<S\> | One class with manifest() + session() getters. Type-safe. | ✓ |
| Two concrete types | EmbeddedBackupResult and ServerBackupResult. No generics, duplicated structure. | |
| Untyped session | BackupResult with Object session(). Caller casts. | |

**User's choice:** Generic BackupResult\<S\>
**Notes:** None

---

## Restart Callback Shape

| Option | Description | Selected |
|--------|-------------|----------|
| Supplier\<S\> at backup call | backup(options, restarter) — backend passes lambda creating fresh session. | |
| Constructor callback slot | Add restart callback to session constructor. Permanent slot. | |

**User's choice:** Initially selected Supplier at call time, then refined to callback slot approach for public API cleanliness.

---

## Public Backup API

| Option | Description | Selected |
|--------|-------------|----------|
| Callback slot | Session gets Function\<BackupOptions, BackupResult\<S\>\> injected at construction. User calls session.backup(options). | ✓ |
| Session calls core directly | Session.backup() calls BackupUtil.execute() with injected restarter. | |

**User's choice:** Callback slot
**Notes:** Backend constructs a lambda that internally uses core's backup utility with the right restarter (a Supplier that creates a fresh session). The Supplier is internal to the lambda, not exposed in the public API.

---

## Claude's Discretion

- Exact utility class naming and internal structure for backup algorithm
- Whether BackupResult implements AutoCloseable
- Sentinel file content and naming in tests
- Order of implementation

## Deferred Ideas

None — discussion stayed within phase scope
