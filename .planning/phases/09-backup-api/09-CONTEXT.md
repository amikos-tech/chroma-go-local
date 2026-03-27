# Phase 9: Backup API - Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement backup with filesystem copy, manifest generation, and option builder for both embedded and server modes in Java (JNA + Panama). Backup is purely wrapper-side logic — no FFI calls exist in the Rust shim for backup. The Go implementation closes the runtime, copies the persistence directory, writes a JSON manifest, and restarts. Java must replicate this pattern using the existing `BackupOptions` and `BackupManifest` types from Phase 6.

</domain>

<decisions>
## Implementation Decisions

### Backup logic location
- **D-01:** Hybrid approach — core module owns 100% of the filesystem algorithm (directory copy, SHA256 hashing, manifest JSON writing). Backends inject close + restart callbacks via the existing slot pattern. No logic duplication across JNA and Panama.
- **D-02:** A utility class in core (e.g., `BackupExecutor`) implements the close → copy → write-manifest → restart algorithm. Backends construct it with their lifecycle hooks.

### Session restart after backup
- **D-03:** `backup()` returns a `BackupResult<S>` containing both the `BackupManifest` and a new session instance. The old session is invalidated (closed). Caller uses the new session going forward.
- **D-04:** Sessions remain immutable — `final long handle` is preserved. No mutable handle refactoring.

### BackupResult type design
- **D-05:** Generic `BackupResult<S>` — one class with `manifest()` and `session()` getters. Returns `BackupResult<EmbeddedSession>` from embedded backup, `BackupResult<ServerSession>` from server backup.

### Restart callback shape
- **D-06:** Session's `backup()` uses a `Function<BackupOptions, BackupResult<S>>` callback slot injected at construction time. The backend's lambda internally calls core's backup utility with the right restarter (a `Supplier<S>` that creates a fresh session from saved config). The Supplier is internal to the lambda — not exposed in the public API.
- **D-07:** User calls `session.backup(options)` cleanly — no internal plumbing exposed. The callback slot bundles the full close-copy-restart algorithm.

### BackupOptions mode validation
- **D-08:** Validation happens at `backup()` call time inside the core backup algorithm, not at `BackupOptions.build()` time. Embedded rejects `leaveStopped`, server rejects `leaveClosed`. Matches Go's `resolveBackupOptions` pattern.
- **D-09:** `BackupOptions` class stays unchanged from Phase 6 — no new builder subclasses or mode parameters.

### Test strategy
- **D-10:** Filesystem verify + edge cases (B+D tier). Pre-seed a sentinel file in `@TempDir` before starting the session, call `backup()`, verify the sentinel appears in the backup output directory and manifest JSON is parseable with correct `fileCount`.
- **D-11:** Edge case tests: invalid destination (null/empty), non-empty destination directory, cross-mode option rejection (leaveStopped on embedded, leaveClosed on server).
- **D-12:** Identical test structure in both `:jna:test` and `:panama:test` modules — consistent with Phase 7 D-07 and Phase 8 D-10.

### Claude's Discretion
- Exact utility class naming and internal structure for the backup algorithm
- Whether `BackupResult` implements `AutoCloseable` (delegating to the contained session)
- Sentinel file content and naming in tests
- Order of implementation (core utility first vs session wiring first)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Go backup implementation (reference)
- `internal/runtime/backup.go` — Full backup algorithm: `resolveBackupOptions`, `executeBackup`, `copyDirectory`, `copyFile`, `writeManifest`, `ensureEmptyDir`. Mode validation at option resolution. Server/Embedded `Backup()` methods with close-copy-restart pattern.

### Java core types (Phase 6 output — ready to use)
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupOptions.java` — Builder with destination, includeMetadata, leaveStopped, leaveClosed. Unchanged.
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupManifest.java` — Result POJO with all fields matching Go's BackupManifest.
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupFileMetadata.java` — Per-file metadata POJO (path, sizeBytes, mode, sha256, modifiedAt).

### Session types (wiring targets)
- `java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedSession.java` — Must add `backup()` method + callback slot. Currently has 7 constructor params (handle, close, 5 maintenance callbacks).
- `java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java` — Has `backup(BackupOptions)` stub throwing UnsupportedOperationException. Must wire real implementation + add callback slot.

### Backend implementations
- `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` — JNA backend; must construct backup callback lambda and inject into sessions.
- `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` — Panama backend; same changes as JNA.

### FFI reference (no backup symbols — confirms wrapper-side)
- `shim/src/lib.rs` — No `chroma_backup`, `chroma_embedded_backup`, or `chroma_server_backup` symbols. Backup is 100% wrapper-side.

### Requirements
- `.planning/REQUIREMENTS.md` — BKUP-01 through BKUP-04

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `BackupOptions` + `BackupManifest` + `BackupFileMetadata`: All defined in core from Phase 6, ready to use
- `JsonUtil.toJson()` / `JsonUtil.fromJson()`: For manifest serialization/deserialization
- `EmbeddedSession` callback slot pattern: `BiFunction<Long, String, T>` for maintenance ops — backup uses `Function<BackupOptions, BackupResult<S>>` instead (different signature since no FFI)
- `AbstractChromaRuntime.callFfiVoid/callFfiJson`: NOT used for backup (no FFI calls), but backends use these to implement close/restart

### Established Patterns
- Constructor callback injection: all session methods delegate to backend-injected lambdas
- `AtomicBoolean` close guard with `compareAndSet` for idempotent close
- `@TempDir` with `CleanupMode.NEVER` for Windows compatibility in tests
- `EmbeddedConfigBuilder` / `ServerConfigBuilder` for test YAML generation

### Integration Points
- `EmbeddedSession` constructor must expand by 1 parameter (backup callback)
- `ServerSession` constructor must expand by 1 parameter (backup callback)
- Both backend `doStartEmbedded()` and `doStartServer()` methods must construct and inject backup callbacks
- `Makefile` `test-java` target runs both `:jna:test` and `:panama:test`

</code_context>

<specifics>
## Specific Ideas

- Go's backup algorithm copies the persistence directory with `filepath.WalkDir` and computes SHA256 per file — Java equivalent uses `Files.walkFileTree` with `MessageDigest`
- Go validates destination is not inside source persist path (containment check) — replicate in Java
- Go writes manifest as indented JSON via `json.MarshalIndent` — Java uses `JsonUtil` (Gson) with pretty printing
- The sentinel-file test pattern mirrors Go's existing backup tests — pre-write a file into the persist temp dir before starting the session, then verify it appears in the backup snapshot after `backup()` completes
- `BackupResult<S>` should be a simple final class in core with two fields: `BackupManifest manifest` and `S session`

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 09-backup-api*
*Context gathered: 2026-03-27*
