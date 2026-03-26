# Phase 8: Embedded Maintenance - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-26
**Phase:** 08-embedded-maintenance
**Areas discussed:** EmbeddedSession API shape, FFI call wiring, Integration test scope, Error handling edge cases

---

## EmbeddedSession API shape

| Option | Description | Selected |
|--------|-------------|----------|
| Callback slots | Add callback function fields to EmbeddedSession constructor (matching ServerSession pattern). Backends wire lambdas at construction. Consistent with ServerSession, keeps core module FFI-free. | ✓ |
| Abstract method on runtime | Add abstract methods like doRebuildCollection(handle, json) to AbstractChromaRuntime, EmbeddedSession calls runtime reference. Different pattern from ServerSession's callback slots. | |
| You decide | Claude picks the best approach based on existing patterns. | |

**User's choice:** Callback slots (Recommended)
**Notes:** Keeps consistency with ServerSession pattern established in Phase 6/7.

---

## FFI call wiring

| Option | Description | Selected |
|--------|-------------|----------|
| Eager at init() | Bind all 5 chroma_embedded_* symbols during init(), same as existing server symbols. Fails fast if symbols missing. Consistent with current approach. | ✓ |
| Lazy on first call | Bind each symbol on first maintenance call. Avoids loading symbols that may never be used, but adds complexity and defers errors. | |
| You decide | Claude picks based on existing patterns. | |

**User's choice:** Eager at init() (Recommended)
**Notes:** Consistent with existing init() pattern for server symbols.

---

## Integration test scope

| Option | Description | Selected |
|--------|-------------|----------|
| Smoke tests | Start embedded, call each maintenance op, verify non-null result with expected fields. No pre-populated test data. Mirrors Go test pattern. | |
| Data-seeded tests | Create collections with test data before running maintenance ops. Verifies ops produce meaningful results. More coverage but heavier setup. | |
| You decide | Claude picks the right balance. | |

**User's choice:** Both — smoke tests as baseline, plus data-seeded tests for empirical verification
**Notes:** User feels data-seeded tests are a necessity to ensure not only functional correctness (no throws) but also that maintenance methods actually deliver expected changes by empirically verifying results (e.g., records_scanned > 0).

---

## Error handling edge cases

| Option | Description | Selected |
|--------|-------------|----------|
| Java-side validation | Validate collection name (non-null, non-empty) and option builder constraints in Java before hitting FFI. Matches Phase 6 D-21. Let Rust handle runtime errors. | ✓ |
| Pass-through to Rust | Minimal Java validation, let Rust FFI handle all error cases via LAST_ERROR. Simpler Java code but less descriptive errors. | |
| You decide | Claude picks based on Phase 6 error handling contract. | |

**User's choice:** Java-side validation (Recommended)
**Notes:** Follows Phase 6 three-tier exception rule.

---

## Claude's Discretion

- Exact functional interface types for maintenance callbacks
- Whether to use a single callback type or typed per-operation
- Test data creation approach
- Order of implementation (JNA first vs Panama first)

## Deferred Ideas

None — discussion stayed within phase scope
