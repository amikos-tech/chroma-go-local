# Phase 10: Server Maintenance - Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire the 5 server maintenance operations (rebuildCollection, compactCollection, compactAll, pruneCollectionWAL, pruneAllWAL) through `ServerSession` using the stop → start-embedded → run-op → close-embedded → restart-server pattern, in both JNA and Panama backends. Each operation returns a `MaintenanceResult<R, ServerSession>` containing the operation result and a fresh server session. The old session is invalidated after the operation. No new Rust shim exports — reuses existing `chroma_embedded_*` and `chroma_server_*` FFI symbols.

</domain>

<decisions>
## Implementation Decisions

### Stop-restart orchestration
- **D-01:** Core utility class `MaintenanceExecutor` in core module (like `BackupExecutor`) owns 100% of the stop → start-embedded → run-op → close-embedded → restart-server lifecycle. Backends inject lambdas for close, startEmbedded, and startServer. Zero logic duplication across JNA/Panama.
- **D-02:** Single generic `execute()` method parameterized by request/result types — all 5 ops share the same stop-embed-restart skeleton. Mirrors Go's `runRebuild`/`runCompaction`/`runWALPrune` which are nearly identical.
- **D-03:** When the embedded op succeeds but server restart fails: return the result via `MaintenanceResult` with a non-null `restartError()` and null `session()`. The successful op result is never lost.

### Session invalidation
- **D-04:** Server maintenance returns `MaintenanceResult<R, ServerSession>` — a new session wrapping the fresh server handle. The old `ServerSession` is invalidated (closed flag set). `ServerSession.handle` stays `final long` — no mutable handles.
- **D-05:** This follows the same immutable-session pattern as `BackupResult<S>` from Phase 9. The old session's callback lambdas close over the original handle, which would be stale after restart — new session avoids this entirely.

### MaintenanceResult type
- **D-06:** New `MaintenanceResult<R, S>` generic type in core module. Has `result()` (the op result, always non-null on success), `session()` (new session, null if restart failed), and `restartError()` (exception if restart failed, null otherwise).
- **D-07:** API asymmetry with `EmbeddedSession` is intentional — embedded maintenance returns just the result (no session swap needed since embedded doesn't restart). The different return types reflect real behavioral differences.

### Concurrency / locking
- **D-08:** All server operations that involve stop/restart (backup + 5 maintenance ops) serialize through the existing `backupLock` on `ServerSession`. Matches Go's `backupMu` pattern. Prevents concurrent backup + maintenance which would both try to stop the server.

### Test strategy
- **D-09:** Full data-seeded tests + smoke tests. Use `java.net.http.HttpClient` to call the Chroma REST API (create collections, add documents) before running maintenance ops. No external dependencies.
- **D-10:** After each maintenance op: verify server responds to HTTP requests (heartbeat or collection list), verify result fields are populated (e.g., records scanned, WAL prune counts), verify the collection still exists via HTTP.
- **D-11:** Identical test structure in both `:jna:test` and `:panama:test` modules — consistent with Phase 7 D-07 and Phase 8 D-10.

### Claude's Discretion
- Exact `MaintenanceExecutor.execute()` method signature and generic bounds
- HTTP test helper class structure for data seeding
- Chroma REST API endpoint paths for collection/document operations
- Order of implementation (core types first vs wiring first vs tests first)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Go server maintenance implementation (reference pattern)
- `internal/runtime/rebuild.go` — `Server.RebuildCollection()` and `runRebuild()`: stop → embed → op → close-embed → restart pattern, error handling with result preservation
- `internal/runtime/compaction.go` — `Server.CompactCollection()`, `CompactAll()`, and `runCompaction()`: same pattern as rebuild
- `internal/runtime/wal_prune.go` — `Server.PruneCollectionWAL()`, `PruneAllWAL()`, and `runWALPrune()`: same pattern
- `internal/runtime/backup.go` — `Server.snapshotBackupInputs()` and `Server.restartFromConfig()`: config capture and restart helpers

### Java core types (Phase 6 + Phase 9 output)
- `java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java` — Current session with 5 maintenance stubs throwing UnsupportedOperationException; has backupLock, final handle, and backup callback slot
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupExecutor.java` — Reference for close → op → restart pattern in Java; MaintenanceExecutor follows same architecture
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupResult.java` — Reference for result + new session pattern
- `java/core/src/main/java/tech/amikos/chroma/local/core/AbstractChromaRuntime.java` — `callFfiJson` template method, `doStartServer`, `doStartEmbedded`
- `java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedSession.java` — Callback slot pattern with 5 maintenance BiFunction callbacks

### Maintenance request/result types (Phase 6 output)
- `java/core/src/main/java/tech/amikos/chroma/local/core/RebuildOptions.java` — Option builder for rebuild
- `java/core/src/main/java/tech/amikos/chroma/local/core/RebuildCollectionResult.java` — Result POJO
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactCollectionRequest.java` — Request for compact single
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactAllRequest.java` — Request for compact all
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactionResult.java` — Result POJO
- `java/core/src/main/java/tech/amikos/chroma/local/core/WALPruneOptions.java` — Option builder for WAL prune
- `java/core/src/main/java/tech/amikos/chroma/local/core/WALPruneResult.java` — Result POJO

### Backend implementations (wiring targets)
- `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` — JNA backend; `doStartServer()` creates ServerSession with callback lambdas
- `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` — Panama backend; same pattern as JNA

### Rust FFI shim (symbol reference — no new symbols needed)
- `shim/src/lib.rs` — `chroma_embedded_rebuild_collection`, `chroma_embedded_compact_collection`, `chroma_embedded_compact_all`, `chroma_embedded_prune_wal_collection`, `chroma_embedded_prune_wal_all` (used via temporary embedded session); `chroma_server_start_from_string`, `chroma_server_stop`, `chroma_server_free` (used for stop/restart)

### Requirements
- `.planning/REQUIREMENTS.md` — SMNT-01 through SMNT-04

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `BackupExecutor`: Close → op → restart pattern already implemented in Java. `MaintenanceExecutor` follows the same architecture but substitutes filesystem backup for embedded maintenance FFI calls.
- `BackupResult<S>`: Pattern for result + new session. `MaintenanceResult<R, S>` extends this with an additional `restartError()` field.
- `AbstractChromaRuntime.callFfiJson()`: Template method for JSON-in/JSON-out FFI calls — used by temporary embedded session during maintenance ops.
- `EmbeddedSession` callback slots: 5 maintenance `BiFunction<Long, String, T>` callbacks already wired in Phase 8 — used by the temporary embedded session to run the actual maintenance op.
- `ServerConfigBuilder` / `EmbeddedConfigBuilder`: Config builders for test YAML generation.
- `java.net.http.HttpClient`: JDK built-in HTTP client for data-seeded test setup (collection/document creation via Chroma REST API).

### Established Patterns
- Constructor callback injection: all session methods delegate to backend-injected lambdas
- `AtomicBoolean` close guard with `compareAndSet` for idempotent close
- `backupLock` on `ServerSession` for serializing stop/restart operations
- `@TempDir` with `CleanupMode.NEVER` for Windows compatibility in tests
- Identical test structure across JNA and Panama modules

### Integration Points
- `ServerSession` constructor must expand with 5 maintenance callback slots (one per operation)
- Both backend `doStartServer()` methods must construct and inject maintenance callbacks that call `MaintenanceExecutor.execute()`
- `MaintenanceExecutor.execute()` internally calls `doStartEmbedded()` then `doStartServer()` via injected suppliers
- Existing server backup tests coexist alongside new maintenance tests

</code_context>

<specifics>
## Specific Ideas

- Go's `snapshotBackupInputs()` captures the server config before stopping — Java's `MaintenanceExecutor` should capture the configYaml from the `doStartServer` call for restart
- The temporary embedded session created during maintenance uses the same persist path as the server — config conversion from server YAML to embedded YAML may be needed (Go does this explicitly with separate config types)
- HTTP test helper should handle waiting for server startup (poll heartbeat endpoint) since server start is async
- `MaintenanceResult.restartError()` allows callers to log and handle restart failures without losing the successful op result — important for observability

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 10-server-maintenance*
*Context gathered: 2026-03-27*
