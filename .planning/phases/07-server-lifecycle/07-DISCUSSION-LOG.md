# Phase 7: Server Lifecycle - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-26
**Phase:** 07-server-lifecycle
**Areas discussed:** AbstractChromaRuntime retrofit, Integration test scope, Error handling edge cases

---

## AbstractChromaRuntime Retrofit

| Option | Description | Selected |
|--------|-------------|----------|
| Full retrofit now | Rewrite both backends to extend AbstractChromaRuntime, use callFfi* for ALL FFI calls | ✓ |
| Server-only retrofit | Only wire server methods through AbstractChromaRuntime, leave version/embedded as-is | |
| Defer entirely | Keep backends as-is, just add integration tests | |

**User's choice:** Full retrofit now (Recommended)
**Notes:** Both backends currently bypass the FFI lock. Full retrofit ensures thread safety for all FFI calls, not just server lifecycle.

---

## Integration Test Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Real server tests | Start actual Chroma server via FFI, verify accessors, use ephemeral ports | ✓ |
| Real + unit hybrid | Real server for happy path, mock callbacks for edge cases | |
| Unit tests only | Mock all FFI callbacks | |

**User's choice:** Real server tests (Recommended)
**Notes:** Matches Go test patterns. Requires Rust shim built.

## Backend Test Parity

| Option | Description | Selected |
|--------|-------------|----------|
| Identical tests | Same test class structure in both :jna and :panama modules | ✓ |
| JNA primary, Panama light | Full coverage in JNA, smoke tests only in Panama | |

**User's choice:** Identical tests (Recommended)

---

## Error Handling Edge Cases

| Option | Description | Selected |
|--------|-------------|----------|
| Happy path + basic errors | Start/accessor/stop/close plus invalid config and double close | |
| Comprehensive error matrix | All of above plus port-already-bound, close-then-access, concurrent starts | ✓ |
| Happy path only | Just verify start/port/address/url/stop/close works | |

**User's choice:** Comprehensive error matrix

---

## Claude's Discretion

- Test class naming and organization
- Shared test base class vs duplicated methods
- Port selection strategy details
- Order of retrofit tasks

## Deferred Ideas

None
