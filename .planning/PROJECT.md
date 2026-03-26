# chroma-go-local: Local Chroma Runtime Bindings

## What This Is

A multi-language local Chroma runtime package providing Go and Java APIs for running ChromaDB in-process or as a managed server. Go uses purego (no cgo) for FFI, Java offers dual JNA (Java 17+) and Panama (Java 22+) backends. All languages share a common Rust FFI shim.

## Core Value

Java and Go APIs must provide equivalent access to all Chroma runtime capabilities — server lifecycle, embedded mode, backup, rebuild, compaction, and WAL maintenance — through idiomatic language-specific interfaces backed by the same FFI symbols.

## Current Milestone: v0.5.0 Java API Surface

**Goal:** Expand the Java bindings from scaffold (version + embedded start/close) to full API parity with the Go surface — server mode, builder configuration, backup, rebuild, compaction, and WAL prune — in both JNA and Panama backends.

**Target features:**
- Server lifecycle (start, stop, port, address, URL) with Java builder pattern
- Embedded mode extensions (backup, rebuild, compaction, WAL prune on sessions)
- Backup API with option builder
- Collection rebuild API with option builder
- Compaction API (per-collection and all)
- WAL prune API (per-collection and all) with option builder
- Full test coverage for both JNA and Panama backends

## Requirements

### Validated

- ✓ Pure Go FFI via purego (no cgo) — existing
- ✓ Server mode (HTTP) and Embedded mode (in-process) — existing
- ✓ Builder pattern configuration with YAML backing — existing
- ✓ Explicit resource lifecycle with finalizer fallback — existing
- ✓ Java scaffold bindings (JNA + Panama) — existing
- ✓ Rust FFI shim with C-compatible exports — existing
- ✓ Cross-platform support (Linux, macOS, Windows) — existing
- ✓ WAL prune maintenance APIs — v0.4.0 (#26)
- ✓ Collection rebuild maintenance API — v0.4.0 (#25)
- ✓ Go subtree reorganization with facade pattern — v0.4.0

### Active

- [x] Java server lifecycle API (start/stop/port/address/URL) — Validated in Phase 7
- [x] Java builder pattern for server configuration — Validated in Phase 6
- [ ] Java backup API with option builder
- [ ] Java rebuild API with option builder
- [ ] Java compaction API (per-collection and all)
- [ ] Java WAL prune API with option builder
- [ ] JNA and Panama implementations kept in sync
- [ ] Java integration tests for all new APIs

### Out of Scope

- New Go API additions — Go surface is stable
- Rust shim changes — Java reuses existing chroma_* FFI symbols
- Go module path change — must remain `github.com/amikos-tech/chroma-go-local`
- New Chroma features — this is API parity, not new functionality
- Java publishing to Maven Central — separate milestone

## Context

The Java scaffold (v0.3.x) provides basic `ChromaRuntime` interface with `version()` and `startEmbedded()`. The Go side has full server mode, embedded mode, and maintenance APIs (backup, rebuild, compaction, WAL prune). The v0.5.0 milestone bridges this gap by implementing the full Go API surface in Java, reusing the same `chroma_*` FFI symbols that Go calls via purego.

Phase 6 complete — core module now contains all shared types (7 result POJOs, 6 option/request builders, 2 config builders), FFI safety infrastructure (`AbstractChromaRuntime` with global lock), and `ServerSession` with callback slots. Backend modules (JNA, Panama) can now implement against these stable contracts.

Phase 7 complete — both JNA and Panama backends retrofitted to extend `AbstractChromaRuntime`, replacing inline FFI patterns with lock-protected template methods. Server lifecycle (start/stop/close) wired through `ServerSession` with method-reference callbacks. Integration tests verify full error matrix in both backends.

Existing Java architecture: `core` module defines `ChromaRuntime` interface + `EmbeddedSession`, `jna` module implements via JNA, `panama` module implements via Foreign Function & Memory API. Both backends must stay in sync.

GitHub milestone: v0.5.0

## Constraints

- **FFI symbols**: Java must call the same `chroma_*` symbols Go uses — no new Rust shim exports
- **Dual backend**: Every new API must be implemented in both JNA and Panama
- **Java versions**: JNA path requires Java 17+, Panama path requires Java 22+
- **Build system**: `make test-java`, `make test-all` must pass throughout
- **CI**: GitHub Actions must remain green across OS matrix
- **Idiomatic Java**: Builder pattern for configuration, AutoCloseable for lifecycle, checked exceptions where appropriate

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Reuse existing chroma_* FFI symbols | No Rust shim changes needed; Go and Java share same native interface | Validated (Phase 6) |
| Java builder pattern for config | Idiomatic Java; Go uses functional options which don't translate well | Validated (Phase 6) |
| Both JNA and Panama in sync | Maintains Java 17+ support via JNA while offering Panama for Java 22+ | — Pending |
| Full API mirror in one milestone | Shipping partial Java API creates confusing mixed coverage | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-03-26 after Phase 7 completion*
