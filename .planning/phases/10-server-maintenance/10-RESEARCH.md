# Phase 10: Server Maintenance - Research

**Researched:** 2026-03-27
**Domain:** Java FFI session lifecycle, stop-restart orchestration, HTTP-based integration testing
**Confidence:** HIGH

## Summary

Phase 10 wires five server maintenance operations (rebuildCollection, compactCollection, compactAll, pruneCollectionWAL, pruneAllWAL) through `ServerSession` using the stop-embed-op-restart pattern. This is a pure Java implementation phase -- no Rust shim changes, no new FFI symbols. The Go reference implementation (`runRebuild`, `runCompaction`, `runWALPrune`) demonstrates the exact lifecycle: lock -> snapshot config -> stop server -> start embedded -> run op -> close embedded -> restart server. All five Go functions are structurally identical, differing only in the operation lambda passed to the embedded session.

The Java implementation follows the same consolidation: a single `MaintenanceExecutor.execute()` method parameterized by operation type, with backends injecting lambdas for close, startEmbedded, and startServer. The `BackupExecutor` from Phase 9 provides the closest architectural reference. The key difference: backup performs filesystem I/O between stop and restart, while maintenance starts a temporary embedded session and runs FFI calls.

**Primary recommendation:** Implement `MaintenanceExecutor` in core module with a generic `execute()` method, expand `ServerSession` constructor with 5 maintenance callback slots, wire callbacks in both JNA and Panama `doStartServer()` methods, then write data-seeded integration tests using `java.net.http.HttpClient` against the Chroma v2 REST API.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Core utility class `MaintenanceExecutor` in core module (like `BackupExecutor`) owns 100% of the stop -> start-embedded -> run-op -> close-embedded -> restart-server lifecycle. Backends inject lambdas for close, startEmbedded, and startServer. Zero logic duplication across JNA/Panama.
- **D-02:** Single generic `execute()` method parameterized by request/result types -- all 5 ops share the same stop-embed-restart skeleton. Mirrors Go's `runRebuild`/`runCompaction`/`runWALPrune` which are nearly identical.
- **D-03:** When the embedded op succeeds but server restart fails: return the result via `MaintenanceResult` with a non-null `restartError()` and null `session()`. The successful op result is never lost.
- **D-04:** Server maintenance returns `MaintenanceResult<R, ServerSession>` -- a new session wrapping the fresh server handle. The old `ServerSession` is invalidated (closed flag set). `ServerSession.handle` stays `final long` -- no mutable handles.
- **D-05:** This follows the same immutable-session pattern as `BackupResult<S>` from Phase 9. The old session's callback lambdas close over the original handle, which would be stale after restart -- new session avoids this entirely.
- **D-06:** New `MaintenanceResult<R, S>` generic type in core module. Has `result()` (the op result, always non-null on success), `session()` (new session, null if restart failed), and `restartError()` (exception if restart failed, null otherwise).
- **D-07:** API asymmetry with `EmbeddedSession` is intentional -- embedded maintenance returns just the result (no session swap needed since embedded doesn't restart). The different return types reflect real behavioral differences.
- **D-08:** All server operations that involve stop/restart (backup + 5 maintenance ops) serialize through the existing `backupLock` on `ServerSession`. Matches Go's `backupMu` pattern. Prevents concurrent backup + maintenance which would both try to stop the server.
- **D-09:** Full data-seeded tests + smoke tests. Use `java.net.http.HttpClient` to call the Chroma REST API (create collections, add documents) before running maintenance ops. No external dependencies.
- **D-10:** After each maintenance op: verify server responds to HTTP requests (heartbeat or collection list), verify result fields are populated (e.g., records scanned, WAL prune counts), verify the collection still exists via HTTP.
- **D-11:** Identical test structure in both `:jna:test` and `:panama:test` modules -- consistent with Phase 7 D-07 and Phase 8 D-10.

### Claude's Discretion
- Exact `MaintenanceExecutor.execute()` method signature and generic bounds
- HTTP test helper class structure for data seeding
- Chroma REST API endpoint paths for collection/document operations
- Order of implementation (core types first vs wiring first vs tests first)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SMNT-01 | `ServerSession.rebuildCollection(name, options)` uses stop-embed-op-restart pattern | MaintenanceExecutor.execute() with rebuild callback; Go reference: `Server.runRebuild()` |
| SMNT-02 | `ServerSession.compactCollection(request)` and `compactAll(request)` use stop-embed-op-restart pattern | Same MaintenanceExecutor.execute() with compact callbacks; Go reference: `Server.runCompaction()` |
| SMNT-03 | `ServerSession.pruneCollectionWAL(name, options)` and `pruneAllWAL(options)` use stop-embed-op-restart pattern | Same MaintenanceExecutor.execute() with prune callbacks; Go reference: `Server.runWALPrune()` |
| SMNT-04 | Integration tests verify server maintenance operations in both backends | Data-seeded tests via Chroma v2 REST API + heartbeat verification post-restart |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **No cgo**: purego-based FFI (not applicable to Java phase, but context)
- **Facade pattern**: Root Go package re-exports; Java has different architecture but same layering principle (core module has zero FFI dependency)
- **Testing**: `make test-java` runs `:jna:test` and `:panama:test`; `make test-all` for full suite
- **Linting**: Java: `gradle --no-daemon :core:check :jna:check :panama:check`
- **Conventional commits**: Required for all commits
- **Keep things radically simple**: No over-engineering
- **No scope creep**: Do exactly what is asked

## Architecture Patterns

### Go Reference: Stop-Embed-Op-Restart Skeleton

All three Go functions (`runRebuild`, `runCompaction`, `runWALPrune`) follow this identical structure:

```
1. Lock backupMu
2. Snapshot server config (snapshotBackupInputs)
3. Close server (s.Close())
4. Start temporary embedded session (StartEmbedded with same config)
   - If embedded start fails: try restart server, return error
5. Run operation via embedded session
6. Close embedded session
7. Restart server (restartFromConfig)
8. Return result + error based on which steps failed
```

The error handling matrix is identical across all three:
- **Op failed + restart ok**: return nil result, op error
- **Op failed + restart failed**: return nil result, combined error with "server remains stopped"
- **Op ok + close-embedded failed + restart ok**: return result, close error
- **Op ok + close-embedded failed + restart failed**: return result, combined error
- **Op ok + restart failed**: return result, restart error with "server remains stopped"
- **Op ok + restart ok**: return result, nil error

### Java Target: MaintenanceExecutor

```
java/core/src/main/java/tech/amikos/chroma/local/core/
  MaintenanceExecutor.java    # NEW - stop-embed-op-restart orchestration
  MaintenanceResult.java      # NEW - result + new session + optional restart error
  ServerSession.java          # MODIFIED - add 5 maintenance callback slots + methods
```

### MaintenanceExecutor Design

```java
public final class MaintenanceExecutor {
    private MaintenanceExecutor() {}

    @FunctionalInterface
    public interface EmbeddedOperation<R> {
        R execute(long embeddedHandle) throws Exception;
    }

    public static <R> MaintenanceResult<R, ServerSession> execute(
            String configYaml,
            Runnable closeServerAction,
            Function<String, EmbeddedSession> startEmbeddedAction,
            Function<String, ServerSession> startServerAction,
            Function<EmbeddedSession, R> operation) {
        // 1. Close server
        // 2. Start temporary embedded session
        // 3. Run operation
        // 4. Close embedded session
        // 5. Restart server
        // 6. Return MaintenanceResult with result, new session, and optional restart error
    }
}
```

### MaintenanceResult Design

```java
public final class MaintenanceResult<R, S> {
    private final R result;
    private final S session;
    private final Exception restartError;

    // result() - always non-null on success
    // session() - null if restart failed
    // restartError() - null if restart succeeded
}
```

### ServerSession Constructor Expansion

Current constructor has 7 parameters. Must expand to 12 (add 5 maintenance callbacks). Each callback is a `Function<RequestType, MaintenanceResult<ResultType, ServerSession>>`:

```java
// 5 new callback slots:
Function<RebuildOptions, MaintenanceResult<RebuildCollectionResult, ServerSession>> rebuildAction
Function<CompactCollectionRequest, MaintenanceResult<CompactionResult, ServerSession>> compactCollectionAction
Function<CompactAllRequest, MaintenanceResult<CompactionResult, ServerSession>> compactAllAction
Function<WALPruneOptions, MaintenanceResult<WALPruneResult, ServerSession>> pruneWalCollectionAction
Function<WALPruneOptions, MaintenanceResult<WALPruneResult, ServerSession>> pruneWalAllAction
```

### Backend Wiring Pattern

In `doStartServer()` for both JNA and Panama, the maintenance callbacks call `MaintenanceExecutor.execute()` with:
- `configYaml` captured from the `doStartServer` parameter
- `closeServerAction`: stop + free the current handle
- `startEmbeddedAction`: call `doStartEmbedded(configYaml)`
- `startServerAction`: call `doStartServer(configYaml)`
- `operation`: delegate to the appropriate embedded session method

### Recommended Implementation Order

```
Step 1: Core types (MaintenanceResult, MaintenanceExecutor)
Step 2: ServerSession constructor expansion + method rewiring
Step 3: Backend wiring (JNA + Panama doStartServer)
Step 4: Integration tests (both modules)
```

### Anti-Patterns to Avoid
- **Mutable session handles**: Never replace `ServerSession.handle` after construction. The Go implementation can mutate `Server.handle` because Go has different ownership semantics. Java uses immutable sessions with session swap on `MaintenanceResult`.
- **Duplicating lifecycle logic in backends**: All stop-embed-restart logic MUST live in `MaintenanceExecutor`. Backends only inject lambdas.
- **Ignoring close-embedded errors**: The Go reference carefully propagates embedded-close errors. Java must do the same -- suppress them if the op also failed, but report them otherwise.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Stop-restart orchestration | Per-method lifecycle code in each backend | `MaintenanceExecutor.execute()` | 5 ops x 2 backends = 10 places to get error handling wrong |
| HTTP testing | Custom HTTP framework | `java.net.http.HttpClient` (JDK built-in) | Zero dependencies, available since Java 11 |
| JSON in tests | Manual string construction | Gson or String.format for test JSON | Less error-prone |
| Server readiness polling | Single GET with timeout | Retry loop on heartbeat endpoint | Server start is async, needs polling |

## Common Pitfalls

### Pitfall 1: Config YAML Reuse Between Server and Embedded
**What goes wrong:** The server config YAML includes server-specific fields (port, listen_address) that the embedded session ignores or rejects.
**Why it happens:** Go's `runRebuild` explicitly passes the same ConfigPath/ConfigString to both `StartServer` and `StartEmbedded`. The Rust shim's embedded start ignores server-only fields.
**How to avoid:** Pass the original `configYaml` string directly to `doStartEmbedded()` -- the Chroma runtime internally ignores server-specific fields in embedded mode. Verified by Go reference implementation which does exactly this.
**Warning signs:** Embedded start failure after server stop.

### Pitfall 2: Session Invalidation Timing
**What goes wrong:** Old session is used after maintenance op returns new session.
**Why it happens:** Caller holds reference to old `ServerSession` and calls methods on it after receiving `MaintenanceResult`.
**How to avoid:** Set `closed` flag on old session before returning `MaintenanceResult`. This is what `BackupResult` pattern does -- `ServerSession.backup()` sets `closed.set(true)` before returning.
**Warning signs:** `IllegalStateException: session is closed` at unexpected points.

### Pitfall 3: Lock Ordering with backupLock
**What goes wrong:** Deadlock between concurrent backup and maintenance operations.
**Why it happens:** Both backup and maintenance need to stop/restart the server. If they don't use the same lock, they can interleave.
**How to avoid:** All 6 operations (backup + 5 maintenance) must acquire `backupLock` before any server mutation. D-08 mandates this.
**Warning signs:** Test hangs when running backup and maintenance concurrently.

### Pitfall 4: Server Restart on Different Port
**What goes wrong:** After restart, the server listens on a different port than expected because the port was dynamically allocated.
**Why it happens:** If the original port was 0 (auto-assign), the restart might get a different port.
**How to avoid:** Tests use explicit port assignment via `findFreePort()` + `ServerConfigBuilder.port()`. The `MaintenanceResult.session()` contains the new session with correct port from the restarted server.
**Warning signs:** HTTP requests to old port fail after maintenance op.

### Pitfall 5: Embedded Session Start Failure After Server Stop
**What goes wrong:** Server is stopped but embedded session fails to start. Server cannot restart because the port is still marked as recently used.
**Why it happens:** OS port recycling delays, or embedded start fails due to locked database files.
**How to avoid:** `MaintenanceExecutor` must attempt server restart even if embedded start fails (matching Go pattern). Use retry-with-delay for heartbeat verification in tests.
**Warning signs:** "server remains stopped" errors in test logs.

### Pitfall 6: HTTP Data Seeding Race Condition
**What goes wrong:** Test creates collection/adds documents before server is fully ready.
**Why it happens:** Server start returns before the HTTP listener is accepting connections.
**How to avoid:** Poll heartbeat endpoint (`GET /api/v2/heartbeat`) before any data operations. Go tests do this via `requireServerHeartbeat()` which polls with 100ms intervals for up to 10 seconds.
**Warning signs:** Connection refused errors in test data seeding.

## Code Examples

### MaintenanceExecutor.execute() Structure

Based on Go's `runRebuild`/`runCompaction`/`runWALPrune` error-handling matrix:

```java
// Source: Go reference internal/runtime/rebuild.go lines 184-233
public static <R> MaintenanceResult<R, ServerSession> execute(
        String configYaml,
        Runnable closeServerAction,
        Function<String, EmbeddedSession> startEmbeddedAction,
        Function<String, ServerSession> startServerAction,
        Function<EmbeddedSession, R> operation) {

    // Step 1: Close server
    closeServerAction.run();

    // Step 2: Start temporary embedded session
    EmbeddedSession embedded;
    try {
        embedded = startEmbeddedAction.apply(configYaml);
    } catch (RuntimeException startErr) {
        // Try restart server even though embedded failed
        try {
            ServerSession restarted = startServerAction.apply(configYaml);
            // Return error with restarted session
        } catch (RuntimeException restartErr) {
            // Both failed -- throw combined error
        }
        throw startErr;
    }

    // Step 3: Run operation
    R result = null;
    RuntimeException opError = null;
    try {
        result = operation.apply(embedded);
    } catch (RuntimeException e) {
        opError = e;
    }

    // Step 4: Close embedded
    RuntimeException closeError = null;
    try {
        embedded.close();
    } catch (RuntimeException e) {
        closeError = e;
    }

    // Step 5: Restart server
    ServerSession newSession = null;
    Exception restartError = null;
    try {
        newSession = startServerAction.apply(configYaml);
    } catch (RuntimeException e) {
        restartError = e;
    }

    // Step 6: Error matrix (matches Go switch statement)
    // ... propagate errors following Go pattern
}
```

### ServerSession Maintenance Method Pattern

Based on existing `backup()` method at line 131-149:

```java
// Source: existing ServerSession.java backup() pattern
public MaintenanceResult<RebuildCollectionResult, ServerSession> rebuildCollection(RebuildOptions options) {
    backupLock.lock();
    try {
        ensureOpen();
        if (options == null) throw new IllegalArgumentException("options is required");
        MaintenanceResult<RebuildCollectionResult, ServerSession> result = rebuildAction.apply(options);
        closed.set(true);  // Invalidate old session
        return result;
    } catch (RuntimeException e) {
        closed.set(true);  // Invalidate on error too
        throw e;
    } finally {
        backupLock.unlock();
    }
}
```

### Backend doStartServer Wiring

Based on existing JNA `doStartServer()` at line 132-147:

```java
// Source: existing JnaChromaRuntime.java doStartServer() pattern
@Override
protected ServerSession doStartServer(String configYaml) {
    long handle = callFfiHandle(...);
    String persistPath = serverPersistPath(handle);
    String version = doVersion();
    return new ServerSession(
            handle,
            this::serverStop,
            this::serverFree,
            this::serverPort,
            this::serverAddress,
            this::serverPersistPath,
            // Existing backup callback
            opts -> BackupExecutor.execute(...),
            // NEW: 5 maintenance callbacks
            opts -> MaintenanceExecutor.execute(configYaml,
                    () -> { serverStop(handle); serverFree(handle); },
                    this::doStartEmbedded,
                    this::doStartServer,
                    emb -> emb.rebuildCollection(opts)),
            req -> MaintenanceExecutor.execute(configYaml, ...),  // compactCollection
            req -> MaintenanceExecutor.execute(configYaml, ...),  // compactAll
            opts -> MaintenanceExecutor.execute(configYaml, ...),  // pruneWalCollection
            opts -> MaintenanceExecutor.execute(configYaml, ...)); // pruneWalAll
}
```

### HTTP Test Helper for Data Seeding

Based on D-09 and Chroma v2 REST API:

```java
// Source: Chroma v2 REST API (confirmed by Go test heartbeat paths: /api/v2/heartbeat)
class ChromaHttpHelper {
    private final HttpClient client = HttpClient.newHttpClient();
    private final String baseUrl;

    void waitForReady(Duration timeout) {
        // Poll GET /api/v2/heartbeat until 200 OK
    }

    String createCollection(String name) {
        // POST /api/v2/tenants/default_tenant/databases/default_database/collections
        // Body: {"name": "...", "get_or_create": true}
        // Returns collection ID from response
    }

    void addDocuments(String collectionId, List<String> ids,
                      List<List<Float>> embeddings, List<String> documents) {
        // POST /api/v2/tenants/default_tenant/databases/default_database/collections/{id}/add
        // Body: {"ids": [...], "embeddings": [...], "documents": [...]}
    }

    List<String> listCollectionNames() {
        // GET /api/v2/tenants/default_tenant/databases/default_database/collections
    }
}
```

### Chroma v2 REST API Endpoints (for tests)

| Operation | Method | Path | Notes |
|-----------|--------|------|-------|
| Heartbeat | GET | `/api/v2/heartbeat` | Returns `{"nanosecond_heartbeat": N}` |
| Create collection | POST | `/api/v2/tenants/default_tenant/databases/default_database/collections` | Body: `{"name":"...", "get_or_create": true}` |
| Add documents | POST | `/api/v2/tenants/default_tenant/databases/default_database/collections/{id}/add` | Body: `{"ids":[...], "embeddings":[...], "documents":[...]}` |
| List collections | GET | `/api/v2/tenants/default_tenant/databases/default_database/collections` | Returns JSON array |
| Count records | GET | `/api/v2/tenants/default_tenant/databases/default_database/collections/{id}/count` | Returns count |

**Confidence: MEDIUM** -- These paths are confirmed as v2 by Go test code (`/api/v2/heartbeat`). Collection CRUD paths follow the standard Chroma v2 pattern but have not been independently verified against the actual OpenAPI spec in this research. The implementer should verify the exact paths by hitting `http://localhost:{port}/docs` during a test run if any path fails.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Chroma v1 API paths (`/api/v1/...`) | Chroma v2 API paths (`/api/v2/...`) | 2024 | Tests must use v2 paths |
| Mutable session handles | Immutable session + session swap | Phase 9 (backup) | Maintenance follows same pattern |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | JUnit 5 (Jupiter) via Gradle |
| Config file | `java/jna/build.gradle.kts` and `java/panama/build.gradle.kts` |
| Quick run command | `gradle --no-daemon :jna:test --tests '*ServerMaintenanceTest*'` |
| Full suite command | `make test-java` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SMNT-01 | rebuildCollection stop-embed-restart | integration | `gradle --no-daemon :jna:test --tests '*ServerMaintenanceTest.serverRebuild*'` | Wave 0 |
| SMNT-02 | compactCollection/compactAll stop-embed-restart | integration | `gradle --no-daemon :jna:test --tests '*ServerMaintenanceTest.serverCompact*'` | Wave 0 |
| SMNT-03 | pruneCollectionWAL/pruneAllWAL stop-embed-restart | integration | `gradle --no-daemon :jna:test --tests '*ServerMaintenanceTest.serverPrune*'` | Wave 0 |
| SMNT-04 | Both backends pass identical tests | integration | `make test-java` | Wave 0 |

### Sampling Rate
- **Per task commit:** `gradle --no-daemon :jna:test :panama:test`
- **Per wave merge:** `make test-java`
- **Phase gate:** `make test-all` green before verification

### Wave 0 Gaps
- [ ] `java/jna/src/test/java/.../JnaServerMaintenanceTest.java` -- covers SMNT-01, SMNT-02, SMNT-03 (JNA side of SMNT-04)
- [ ] `java/panama/src/test/java/.../PanamaServerMaintenanceTest.java` -- covers SMNT-01, SMNT-02, SMNT-03 (Panama side of SMNT-04)
- [ ] No new framework install needed -- JUnit 5 already configured

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Java 17+ | JNA module | Yes | Java 26 (build 26+35-2893) | -- |
| Java 22+ | Panama module | Yes | Java 26 | -- |
| Gradle 9+ | Build system | Yes | 9.4.1 | -- |
| Rust 1.70+ | Shim build | Yes | 1.89.0 | -- |
| Cargo | Shim build | Yes | 1.89.0 | -- |
| java.net.http.HttpClient | Test data seeding | Yes (JDK built-in since Java 11) | -- | -- |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None.

## Open Questions

1. **Exact Chroma v2 collection CRUD paths**
   - What we know: Heartbeat is confirmed at `/api/v2/heartbeat` by Go tests. Collection paths follow v2 pattern.
   - What's unclear: Exact path structure for `create_collection` and `add` in v2 API (v1 was `/api/v1/collections`; v2 adds tenant/database to path).
   - Recommendation: The implementer should verify by starting a server in a test and hitting `/docs` endpoint, or use trial-and-error with the known v2 base path. The Go compaction_test.go seeds data via the embedded Go API (not HTTP), so no HTTP seeding reference exists in the Go codebase. If the v2 paths are wrong, fall back to creating collections via a temporary embedded session before starting the server (same approach as Go's `startTestServerWithCollection`).

2. **Convenience overloads on ServerSession maintenance methods**
   - What we know: `ServerSession.rebuildCollection(String name)` and `pruneCollectionWAL(String name)` convenience overloads exist (delegate to the options-based version). `compactCollection(String name)` also exists.
   - What's unclear: Whether these convenience overloads should also return `MaintenanceResult` or just the raw result type.
   - Recommendation: They MUST return `MaintenanceResult` -- the session swap is the fundamental behavioral change. Callers need the new session regardless of which overload they use.

## Sources

### Primary (HIGH confidence)
- Go reference: `internal/runtime/rebuild.go` -- `Server.runRebuild()` lines 184-233
- Go reference: `internal/runtime/compaction.go` -- `Server.runCompaction()` lines 174-223
- Go reference: `internal/runtime/wal_prune.go` -- `Server.runWALPrune()` lines 379-428
- Go reference: `internal/runtime/backup.go` -- `Server.restartFromConfig()`, `snapshotBackupInputs()`
- Java existing: `ServerSession.java` -- current constructor, backup method, backupLock pattern
- Java existing: `BackupExecutor.java` -- close-op-restart pattern reference
- Java existing: `EmbeddedSession.java` -- 5 maintenance BiFunction callback slots
- Java existing: `JnaChromaRuntime.java` -- `doStartServer()` and `doStartEmbedded()` wiring
- Java existing: `PanamaChromaRuntime.java` -- same pattern as JNA

### Secondary (MEDIUM confidence)
- Chroma v2 API paths: Confirmed via Go test code using `/api/v2/heartbeat` (backup_test.go, chroma_test.go)
- Collection CRUD paths: Based on standard Chroma v2 pattern with tenant/database scoping

### Tertiary (LOW confidence)
- Exact POST body format for collection creation and document addition via REST API -- needs runtime verification

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all types and patterns exist in codebase, extending established patterns
- Architecture: HIGH -- direct translation from Go reference with clear 1:1 mapping
- Pitfalls: HIGH -- derived from actual Go error-handling code and existing Java session patterns
- Test strategy: MEDIUM -- HTTP data seeding via REST API needs path verification at runtime

**Research date:** 2026-03-27
**Valid until:** 2026-04-27 (stable -- extending existing patterns with no external dependency changes)
