# Project Research Summary

**Project:** local-go-chroma Java API Surface (v0.5.0)
**Domain:** Java FFI bindings (JNA + Panama dual-backend) mirroring Go API for ChromaDB local runtime
**Researched:** 2026-03-21
**Confidence:** HIGH

## Executive Summary

This milestone expands the existing Java scaffold (v0.3.x — 5 FFI symbols, embedded-only) to full API parity with the Go wrapper: server lifecycle, builder configuration, backup, rebuild, compaction, and WAL prune. The Go API is the authoritative source of truth and has been read directly from the codebase; the Rust shim exposes all required FFI symbols already. The Java work is purely additive binding — no Rust changes are required. The core design pattern (operations-delegate via injected interface) keeps the `core` module free of FFI dependencies while both JNA and Panama backends implement the same interfaces, eliminating drift risk.

The recommended approach is a strict 5-phase build order driven by dependency: core types first (no FFI, compiles independently), then server lifecycle (7 new symbols per backend), then embedded maintenance ops (5 new symbols per backend, simpler than server-mode), then backup (filesystem I/O + manifest, most complex embedded op), then server-mode maintenance (stop-embedded-restart choreography composing phases 2 and 3). The only new external dependency is Gson 2.13.2 (~300 KB, zero transitive deps) for JSON request/response marshaling on maintenance APIs. Everything else reuses the existing stack unchanged.

The primary risk cluster is correctness of FFI memory management: the Rust shim has three distinct pointer ownership categories (owned/borrowed/static), a global (non-thread-local) error slot that requires a single FFI lock, and a two-step server teardown that must not be simplified to one call. All seven critical pitfalls identified are preventable with established Java patterns and must be addressed in Phase 1 before any API surface expansion begins. The existing scaffold's design choices (AtomicBoolean close guard, confined Arena for call parameters, LongConsumer close injection) are correct and should be extended, not replaced.

## Key Findings

### Recommended Stack

The stack is stable and requires only one addition. JNA 5.14.0, Panama FFM (JDK 22 built-in), JUnit 5.11.4, and Gradle 9+ (Kotlin DSL multi-module) remain unchanged — none of the newer available versions add features needed for this milestone. Gson 2.13.2 is the single new dependency, chosen over Jackson for its zero transitive dependencies, single 300 KB jar, and zero-configuration annotation-free POJO support. The maintenance API payloads are tiny JSON blobs (<1 KB each), making Jackson's streaming parser performance irrelevant. SnakeYAML, Lombok, Immutables, AssertJ, and Mockito were all evaluated and rejected; builder classes and YAML generation are hand-rolled to match the Go pattern exactly, and all tests are integration tests against the real native library.

**Core technologies:**
- JNA 5.14.0: FFI for Java 17+ backend — keep current, sufficient for all new symbols
- Panama FFM (JDK 22): FFI for Java 22+ backend — finalized JEP 454, no external dependency
- Gson 2.13.2: JSON for maintenance API request/response — minimal footprint, zero transitive deps
- Gradle 9+ (Kotlin DSL): existing multi-module build — no changes needed
- JUnit 5.11.4: test framework — existing features sufficient (TempDir, Assumptions, assertThrows)

### Expected Features

The Java API must match the Go API's 7 feature categories. Config builders (server and embedded) are pure Java with no FFI dependency and should be built first. Server lifecycle requires 7 new FFI symbols per backend. Embedded maintenance requires 5 new FFI symbols per backend and is simpler than server-mode because there is no stop-restart cycle. Server-mode maintenance composes server lifecycle and embedded maintenance, making it last. Backup is the highest-complexity single operation due to Java-side filesystem I/O and manifest generation.

**Must have (table stakes):**
- Server lifecycle (start/stop/close) — Go has it; Java cannot claim parity without it
- Server port/address/URL accessors — H2 sets this standard; needed to connect HTTP clients
- Builder-pattern server configuration — idiomatic Java; raw YAML is an anti-pattern for users
- Builder-pattern embedded configuration — existing `startEmbedded(yaml)` forces hand-written YAML
- Backup API (embedded + server) — data safety is table stakes for any production DB wrapper
- Compaction API (per-collection + all, embedded + server) — core maintenance operation
- Rebuild API (per-collection, embedded + server) — essential for index recovery
- WAL Prune API (per-collection + all, embedded + server) — controls disk usage
- Typed result POJOs for all maintenance operations — users expect structured results, not raw JSON
- Both JNA and Panama implementations — project constraint; every API must work in both backends

**Should have (competitive differentiators):**
- AutoCloseable with try-with-resources on ChromaServer — not implementing AutoCloseable is a regression vs. EmbeddedSession
- Maintenance operation thread safety (ReentrantLock serialization) — prevents subtle corruption; Go's backupMu is the reference
- Dry-run mode for WAL prune and precheck mode for rebuild — pass-through to FFI, low cost, high DX value
- Builder validation at build() time — fail fast with clear messages rather than surfacing as FFI errors

**Defer (v2+):**
- ChromaErrorCode enum on ChromaException — useful but not blocking parity
- Embedded data operations (CRUD, queries) — massive scope, separate milestone
- Java-native observability (OpenTelemetry) — depends on OTel config working through FFI
- BackupEngine-style management (list/purge/restore) — Go does not have this yet either

**Anti-features (do not build):**
- JDBC/DataSource interface — ChromaDB is not relational; the contract does not map
- Async/CompletableFuture API — FFI calls serialize through a single lock; async is misleading
- Connection pooling — embedded mode is a single in-process runtime
- Automatic backup scheduling — belongs in application code, not a DB wrapper library

### Architecture Approach

The architecture follows a strict layering principle: `core` module defines all interfaces, handle wrappers, builders, and result types with zero FFI dependencies; `jna` and `panama` modules each implement the backend-specific FFI binding, operations interfaces, and runtime factories. The key pattern is the operations-delegate: `EmbeddedSession` and `ChromaServer` (in `core`) hold a native handle as `long` and delegate all FFI operations to an `EmbeddedOperations` or `ServerOperations` interface, whose implementations live in the backend modules. This ensures `core` compiles cleanly against Java 17 without any JNA or Panama imports. Server-mode maintenance (backup, rebuild, compact, prune) is not a new FFI concern — it is a pure Java orchestration pattern (stop server, open temporary embedded, run operation, close embedded, restart server) implemented entirely within `ServerOperations` impls.

**Major components:**
1. `ChromaRuntime` interface (core) — library loader and factory for `startEmbedded()` / `startServer()`
2. `EmbeddedSession` class (core) — embedded handle wrapper, refactored to delegate to `EmbeddedOperations`
3. `ChromaServer` class (core) — server handle wrapper, delegates to `ServerOperations`; uses AtomicLong for handle mutability across maintenance restarts
4. `EmbeddedOperations` / `ServerOperations` interfaces (core) — FFI operation contracts implemented by each backend
5. `JnaChromaRuntime` / `PanamaChromaRuntime` (jna/panama) — implement ChromaRuntime + operations interfaces
6. Config builders: `ChromaServerConfig`, `EmbeddedConfig` (core) — produce YAML via StringBuilder matching Go's config.go format exactly
7. Option/request types: `BackupOptions`, `RebuildOptions`, `WALPruneOptions`, `CompactCollectionRequest`, `CompactAllRequest` (core) — pure Java, no FFI
8. Result records (core) — immutable, deserialized from FFI JSON via `JsonCodec`; includes `BackupManifest`, `RebuildResult`, `CompactionResult`, `WALPruneResult` + per-collection variants
9. `JsonCodec` utility (core) — shared Gson instance with LOWER_CASE_WITH_UNDERSCORES naming policy
10. Backup utilities (core) — directory copy, manifest writing, SHA-256 checksums; pure Java, backend-independent

### Critical Pitfalls

1. **Global error slot race** — `LAST_ERROR` in the Rust shim is a global mutex, not thread-local. A single `ffiLock` (ReentrantLock or synchronized) must serialize ALL FFI calls in both backends, matching Go's `ffiMu`. Must be retrofitted into existing methods in Phase 1 before any new symbols are added.

2. **Two-step server teardown** — `chroma_server_stop` and `chroma_server_free` are separate operations. Calling free without stop first drops the tokio Runtime while running, potentially crashing the JVM. ChromaServer.close() must call stop in try, free in finally, always.

3. **Borrowed vs. owned pointer ownership** — `chroma_server_address`, `chroma_server_persist_path`, and `chroma_embedded_persist_path` return borrowed pointers valid only until the handle is freed — do NOT call `chroma_string_free` on them. Maintenance API responses ARE owned and MUST be freed. Establish `readOwnedString()` / `readBorrowedString()` helpers in Phase 1.

4. **Wrong handle type for server maintenance** — maintenance FFI symbols (`chroma_embedded_rebuild_collection` etc.) operate on embedded handles only. Passing a server handle causes undefined behavior at the Rust level (wrong struct layout). Server maintenance must stop the server, open a temporary embedded session, run the operation on the embedded handle, close embedded, and restart the server.

5. **Server handle mutability after maintenance restart** — unlike EmbeddedSession's `final long handle`, ChromaServer's handle changes on every maintenance operation (each restart produces a new native pointer). Use `AtomicLong` for the handle field and update it atomically after restart.

6. **YAML format must match Go's config.go exactly** — the Rust Figment YAML parser is strict about field types; `port: "8000"` (string) fails where `port: 8000` (integer) succeeds. Mirror Go's `fmt.Fprintf` patterns, never use a YAML library. Write golden YAML tests in Phase 1.

7. **Dual backend drift** — with 18 FFI symbols per backend (up from 5), behavioral divergence between JNA and Panama is high-probability. Extract all shared logic (builders, JSON codec, session wrappers, maintenance orchestration) to `core`. Write a shared contract test suite in `core` that both backend test suites run.

## Implications for Roadmap

Based on combined research, the dependency graph mandates a 5-phase structure. Core types must precede FFI bindings (they define the interfaces the bindings implement). Server lifecycle must precede server-mode maintenance (it provides the stop/restart primitives). Embedded maintenance must precede both backup and server-mode maintenance (backup reuses embedded persist path lookup; server-mode maintenance delegates to embedded operations). Backup is last among single-mode operations because it adds filesystem I/O complexity on top of an already-working embedded lifecycle.

### Phase 1: Core Foundation and FFI Safety Patterns

**Rationale:** All subsequent phases depend on core types and FFI safety patterns. Building types first allows Phases 2-5 to compile against them. Retrofitting the FFI lock and string ownership helpers now prevents regressions when new symbols are added. This phase has no FFI calls and can be fully tested with unit tests (builder YAML output, record construction, golden format tests).

**Delivers:** All new interfaces, builders, result records, and option types in core; EmbeddedSession refactored to EmbeddedOperations delegate pattern; `readOwnedString`/`readBorrowedString` helpers established in both runtime impls; FFI lock (`ffiLock`) retrofitted across all existing call sites in both backends; shared contract test infrastructure.

**Addresses:** EmbeddedConfigBuilder (P1), ServerConfigBuilder (P1), all result POJOs (P1), ChromaServer class, ChromaRuntime.startServer() signature extension.

**Avoids:** Pitfalls 1 (global error slot race), 3 (pointer ownership misclassification), 6 (YAML format divergence), 7 (dual backend drift).

### Phase 2: Server Lifecycle (JNA + Panama)

**Rationale:** Server start/stop/port/address is the foundation for all server-mode work. These are 7 new FFI symbols per backend with no maintenance choreography complexity. Keeping this phase focused on lifecycle only allows independent validation before adding the stop-restart pattern in Phase 5.

**Delivers:** `ChromaRuntime.startServer()` in both JNA and Panama, `ChromaServer` with port/address/url/stop/close, integration tests for server lifecycle in both backends.

**Uses:** 7 server FFI symbols (`chroma_server_start_from_string`, `chroma_server_port`, `chroma_server_address`, `chroma_server_stop`, `chroma_server_free`, plus `chroma_server_start` and `chroma_server_persist_path`).

**Avoids:** Pitfalls 2 (two-step server teardown), 5 (Panama arena lifetime with multiple sessions), 8 (handle mutability — use AtomicLong from the start).

### Phase 3: Embedded Maintenance Operations (JNA + Panama)

**Rationale:** The 5 embedded maintenance symbols follow an identical JSON request/response pattern and can be built as a batch. Simpler than server-mode (no stop-restart), this phase establishes JsonCodec, request/response serialization, and EmbeddedOperations implementations that Phase 4 (backup) and Phase 5 (server maintenance) both depend on.

**Delivers:** rebuildCollection, compactCollection, compactAll, pruneCollectionWAL, pruneAllWAL on EmbeddedSession in both backends; JsonCodec with Gson; RebuildOptions/WALPruneOptions builders exercised end-to-end; integration tests for each operation; `chroma_embedded_persist_path` binding (needed by Phase 4).

**Uses:** 5 embedded maintenance FFI symbols + `chroma_embedded_persist_path`; Gson 2.13.2 added to core's build.gradle.kts as an `api` dependency.

**Avoids:** Pitfalls 10 (JSON field name mismatch — snake_case via FieldNamingPolicy), 11 (chroma_string_free on wrong pointer type).

### Phase 4: Backup API (Embedded + Server)

**Rationale:** Backup is the most complex single-mode operation: Java filesystem I/O, SHA-256 checksums, manifest generation, and embedded lifecycle management. Separating it lets Phase 3 (embedded maintenance) be validated before layering on file I/O complexity. Backup utility code in `core` is backend-independent and is built once.

**Delivers:** EmbeddedSession.backup() and ChromaServer.backup() in both backends; backup utilities in core (directory copy, manifest writing, SHA-256); BackupManifest deserialization with golden manifest tests comparing output format against Go's json.Marshal; BackupOptions builder.

**Avoids:** Anti-Pattern 5 (duplicated backup logic in JNA and Panama — utilities live in core).

### Phase 5: Server-Mode Maintenance Operations (JNA + Panama)

**Rationale:** Server-mode maintenance composes Phases 2 and 3: stop server, run embedded maintenance, restart server. Building this last means both foundation layers are validated and stable. The stop-embedded-restart template method can be factored once in ServerOperations and reused for all four maintenance types (rebuild, compact, WAL prune).

**Delivers:** ChromaServer.rebuildCollection(), compactCollection(), compactAll(), pruneCollectionWAL(), pruneAllWAL() in both backends; ReentrantLock maintenance serialization (`maintenanceLock`); AtomicLong handle swap on restart; integration tests for each server-mode operation.

**Avoids:** Pitfall 4 (wrong handle type — never pass server handle to embedded FFI), Pitfall 9 (missing maintenance serialization — one lock guards all maintenance methods).

### Phase Ordering Rationale

- Core types must precede FFI bindings because operations interfaces defined in core are implemented in jna/panama; backends cannot compile without core.
- FFI lock and ownership helpers (Phase 1) must precede any new FFI calls (Phases 2-5); retrofitting them later creates an audit burden and risks missing call sites.
- Server lifecycle (Phase 2) must precede server-mode maintenance (Phase 5) because maintenance requires stop/restart primitives.
- Embedded maintenance (Phase 3) must precede server-mode maintenance (Phase 5) because server maintenance delegates to embedded operations, and must precede backup (Phase 4) because backup needs `chroma_embedded_persist_path`.
- Backup (Phase 4) before server maintenance (Phase 5) is a soft ordering — they are independent — but keeping backup in Phase 4 leaves Phase 5 focused purely on the stop-restart orchestration pattern.

### Research Flags

Phases with well-documented patterns (research-phase not needed):
- **Phase 1:** Pure Java type design — standard patterns, direct codebase examination complete, HIGH confidence.
- **Phase 2:** 7 known FFI symbol signatures verified from Rust shim source — no ambiguity.
- **Phase 3:** 5 known FFI symbol signatures, JSON wire format confirmed from Go source — no ambiguity.

Phases that may benefit from targeted investigation during planning:
- **Phase 4 (Backup):** The Java backup manifest must match Go's backup.go JSON field format exactly. Verify SHA-256 digest encoding (hex vs. base64) and manifest JSON field ordering against Go's `json.Marshal` before signing off. A golden test is mandatory before shipping.
- **Phase 5 (Server-mode maintenance):** The stop-embedded-restart pattern involves handle mutation (AtomicLong swap) under a maintenance lock. Edge cases around concurrent access (a caller holding a reference to the ChromaServer while maintenance runs, and what methods remain valid) should be analyzed during detailed design.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All technologies directly verified from codebase; only Gson version sourced from Maven Central (WebSearch) |
| Features | HIGH | Go API source read directly; all FFI symbols confirmed in shim/src/lib.rs; Java patterns compared against H2/DuckDB/RocksDB |
| Architecture | HIGH | Based on direct source inspection of all Go, Java, and Rust files; patterns are extensions of existing scaffold design |
| Pitfalls | HIGH | All critical pitfalls sourced from direct codebase analysis (LAST_ERROR mutex, ServerHandle struct layout, borrowed pointer docs); cross-referenced with JNA/Panama documentation |

**Overall confidence:** HIGH

### Gaps to Address

- **Gson version currency:** Gson 2.13.2 was verified via WebSearch (Maven Central). Confirm this version aligns with any project dependency management policy before adding. Non-blocking.
- **Backup manifest field format parity:** The exact JSON field names and types in BackupManifest need a golden test comparing Java deserialization output against Go's `json.Marshal` output for the same backup. Must be done in Phase 4 before shipping.
- **Panama parameter arena for new server symbols:** The existing `startEmbedded` uses `Arena.ofConfined()` in try-with-resources for parameter strings. Verify this pattern is correctly applied to all 7 new server symbols — particularly `chroma_server_start_from_string` which takes a potentially longer YAML string. Validate UTF-8/null-termination correctness with an integration test.
- **Concurrent session + runtime close ordering (Panama):** The existing Panama runtime does not track live sessions. With server sessions added, the risk of calling FFI through an invalidated arena increases. Option A (document ordering, no enforcement) is acceptable for v0.5.0; Option B (reference counting) can be deferred to post-v0.5.0 hardening.

## Sources

### Primary (HIGH confidence)
- `shim/src/lib.rs` — all FFI symbol signatures, pointer ownership docs (borrowed vs. owned), LAST_ERROR global mutex pattern
- `internal/runtime/chroma.go` — ffiMu global lock pattern, Server.Stop/Close two-step teardown
- `internal/runtime/backup.go`, `rebuild.go`, `compaction.go`, `wal_prune.go` — maintenance stop-embedded-restart pattern
- `java/core/`, `java/jna/`, `java/panama/` — existing scaffold design, EmbeddedSession pattern, Panama Arena usage
- `chroma.go`, `config.go`, `embedded.go` (root) — Go API surface, YAML generation format
- [JEP 454: Foreign Function & Memory API](https://openjdk.org/jeps/454) — Panama FFM finalized in JDK 22
- [JNA GitHub releases](https://github.com/java-native-access/jna/releases) — version confirmation
- [JUnit 5 Release Notes](https://junit.org/junit5/docs/current/release-notes/index.html) — feature confirmation
- [H2 Server JavaDoc](https://www.h2database.com/javadoc/org/h2/tools/Server.html) — server API comparison
- [RocksDB Java Basics](https://github.com/facebook/rocksdb/wiki/RocksJava-Basics) — embedded DB Java patterns
- [Java Builder Pattern (Baeldung)](https://www.baeldung.com/java-builder-pattern) — standard pattern reference

### Secondary (MEDIUM confidence)
- [Gson Maven Central](https://mvnrepository.com/artifact/com.google.code.gson/gson) — version 2.13.2 current
- [Jackson vs Gson comparison (Baeldung)](https://www.baeldung.com/jackson-vs-gson) — size/complexity tradeoff
- [Common JNA Pitfalls](https://javanexus.com/blog/common-jna-pitfalls-avoid) — thread safety patterns
- [Project Panama FFM in Production](https://www.javacodegeeks.com/2026/03/project-panamas-ffm-api-in-production-replacing-jni-without-writing-c-wrappers.html) — arena lifetime semantics
- [DuckDB Java JDBC Client docs](https://duckdb.org/docs/stable/clients/java) — embedded DB Java API comparison

---
*Research completed: 2026-03-21*
*Ready for roadmap: yes*
