# Feature Research: Java API Surface Mirroring Go API

**Domain:** Java FFI wrapper for embedded/managed database runtime (ChromaDB)
**Researched:** 2026-03-21
**Confidence:** HIGH (Go API source read directly; Java patterns verified against H2, DuckDB, RocksDB, ArcadeDB APIs)

---

## Context: What Already Exists

The Java scaffold (v0.3.x) provides:

| Existing Java Type | What It Does |
|---|---|
| `ChromaRuntime` interface | `version()`, `startEmbedded(configYaml)`, `close()` -- extends `AutoCloseable` |
| `EmbeddedSession` | Handle wrapper with `AtomicBoolean closed`, `LongConsumer closeAction`, `handle()` accessor |
| `ChromaException` | Unchecked `RuntimeException` with message and cause constructors |
| `JnaChromaRuntime` | JNA backend implementing `ChromaRuntime` -- loads library, binds 5 FFI symbols |
| `PanamaChromaRuntime` | Panama backend implementing `ChromaRuntime` -- loads library via Foreign Function API, binds 5 FFI symbols |

The Go API exposes these categories of functionality not yet in Java:
1. **Server lifecycle** -- `StartServer`, `NewServer`, `Server.Port()`, `Server.Address()`, `Server.URL()`, `Server.Stop()`, `Server.Close()`
2. **Server builder configuration** -- `ServerConfig`, `ServerOption` functional options (`WithPort`, `WithListenAddress`, `WithPersistPath`, etc.)
3. **Embedded builder configuration** -- `EmbeddedConfig`, `EmbeddedOption` functional options (`WithEmbeddedPersistPath`, etc.)
4. **Backup** -- `Server.Backup(...)`, `Embedded.Backup(...)` with `BackupOption` (destination, metadata, leave-stopped/closed), returning `BackupManifest`
5. **Rebuild** -- `Server.RebuildCollection(name, ...)`, `Embedded.RebuildCollection(name, ...)` with `RebuildCollectionOption`, returning `RebuildCollectionResult`
6. **Compaction** -- `CompactCollection`, `CompactAll` on both Server and Embedded, returning `CompactionResult`
7. **WAL Prune** -- `PruneCollectionWAL`, `PruneAllWAL` on both Server and Embedded, with `WALPruneOption`, returning `WALPruneResult`

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features Java users assume exist once a library claims API parity with Go. Missing these means the Java bindings feel like an incomplete toy.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Server lifecycle (start/stop/close)** | H2, DuckDB, and every embedded DB with server mode expose start/stop. Go has it. Without this, Java users cannot run Chroma as an HTTP server from Java. | MEDIUM | Requires binding `chroma_server_start_from_string`, `chroma_server_port`, `chroma_server_address`, `chroma_server_stop`, `chroma_server_free`. New `ServerSession` class mirrors `EmbeddedSession` handle pattern. Both JNA and Panama. |
| **Server port/address/URL accessors** | H2's `Server.getPort()`, `Server.getURL()` set the standard. Users need these to connect HTTP clients to the started server. | LOW | Pure Java computation on top of FFI-returned values. Port is `int`, address is `String`, URL is derived `"http://" + addr + ":" + port`. |
| **Builder-pattern server configuration** | Idiomatic Java demands builders, not raw YAML strings. RocksDB uses `Options().setCreateIfMissing(true)`, H2 uses `Server.createTcpServer("-tcpPort", "9092")`. Go uses functional options which do not translate to Java. | MEDIUM | `ServerConfigBuilder` with fluent methods: `port(int)`, `listenAddress(String)`, `persistPath(String)`, `maxPayloadSize(int)`, `corsAllowOrigins(String...)`, `sqliteFilename(String)`, `allowReset(boolean)`, `openTelemetry(String, String)`, `rawYaml(String)`, `build()` returning a YAML string. Lives in `core` module. |
| **Builder-pattern embedded configuration** | Same rationale as server config. Current `startEmbedded(String configYaml)` forces users to hand-write YAML. Every comparable library (RocksDB `Options`, DuckDB `Properties`) provides programmatic config. | LOW | `EmbeddedConfigBuilder` with: `persistPath(String)`, `sqliteFilename(String)`, `allowReset(boolean)`, `rawYaml(String)`, `build()`. Simpler than server since fewer options. |
| **Backup API on EmbeddedSession** | RocksDB has `BackupEngine.createNewBackup()`. Go's `Embedded.Backup(...)` exists. Data safety is table stakes for any production database wrapper. | HIGH | Binds `chroma_embedded_free` (already bound) + requires stop-snapshot-reopen cycle managed in Java. `BackupOptions` builder: `destination(String)`, `includeMetadata(boolean)`, `leaveClosed(boolean)`. Returns `BackupManifest` POJO. Java manages the file copy + manifest (same logic as Go). |
| **Backup API on ServerSession** | Same as embedded backup but for server mode. Go has `Server.Backup(...)`. | HIGH | Follows same stop-backup-restart pattern as Go. `BackupOptions` adds `leaveStopped(boolean)`. Reuses same `BackupManifest` type. |
| **Compaction API (per-collection + all)** | Go exposes `CompactCollection` and `CompactAll` on both Server and Embedded. Compaction is a core maintenance operation. | MEDIUM | Binds `chroma_embedded_compact_collection`, `chroma_embedded_compact_all`. Request POJOs: `CompactCollectionRequest(name, tenantId, databaseName)`, `CompactAllRequest(tenantId, databaseName)`. Returns `CompactionResult` with per-collection breakdown. Server mode uses stop-embed-compact-restart pattern. |
| **Rebuild API (per-collection)** | Go has `RebuildCollection` on both modes. Index rebuild is essential for data recovery. | MEDIUM | Binds `chroma_embedded_rebuild_collection`. `RebuildOptions` builder: `tenantId(String)`, `databaseName(String)`, `precheck(boolean)`, `keepBackup(boolean)`. Returns `RebuildCollectionResult` POJO. Server mode uses stop-embed-rebuild-restart. |
| **WAL Prune API (per-collection + all)** | Go has `PruneCollectionWAL` and `PruneAllWAL`. WAL pruning controls disk usage growth. | MEDIUM | Binds `chroma_embedded_prune_wal_collection`, `chroma_embedded_prune_wal_all`. `WALPruneOptions` builder: `tenantId`, `databaseName`, `dryRun`, `vacuum`, `maxAge(Duration)`, `maxBytes(long)`, `watermark(long, long)`. Returns `WALPruneResult`. |
| **Structured result types (POJOs)** | Go returns typed structs (`BackupManifest`, `RebuildCollectionResult`, `CompactionResult`, `WALPruneResult`). Java users expect equivalent typed objects, not raw JSON strings. | LOW | Immutable record-style classes with JSON deserialization. These are pure data; no FFI involved. Can use Jackson or manual parsing of the JSON strings returned by FFI calls. |
| **Error codes mirrored** | Go has `Success`, `ErrNullInput`, `ErrConfigParse`, etc. Java's `ChromaException` needs to carry structured error information. | LOW | Enum or constants class `ChromaErrorCode` with values matching Go's `runtime.Success`, `runtime.ErrNullInput`, etc. `ChromaException` gains an optional error code field. |
| **Both JNA and Panama implementations** | Constraint from PROJECT.md. Every new API must work in both backends. | HIGH (multiplier) | Each FFI symbol binding must be implemented twice. Pattern is established: JNA uses `interface JnaBindings extends Library`, Panama uses `MethodHandle` with `FunctionDescriptor`. |

### Differentiators (Competitive Advantage)

Features that set this library apart from "just another FFI binding." Not required for parity, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **`ServerSession` as AutoCloseable with try-with-resources** | H2's `Server` lacks AutoCloseable. Making `ServerSession` implement `AutoCloseable` with idempotent `close()` is a quality-of-life win Java users notice. RocksDB does this well. | LOW | Already established pattern in `EmbeddedSession`. Same `AtomicBoolean` + `LongConsumer` pattern. |
| **Maintenance operation thread safety** | Go serializes maintenance ops via `backupMu` mutex. Java equivalent: `ReentrantLock` or `synchronized` ensuring backup/rebuild/compact/prune cannot run concurrently. | MEDIUM | RocksDB Java does not expose this concern to users. Doing it internally prevents subtle corruption. Follow Go's lock ordering: maintenance lock -> state lock -> FFI lock. |
| **Dry-run mode for WAL prune** | Go exposes `WithWALPruneDryRun()`. A Java `dryRun(boolean)` on the WAL prune builder lets users preview what would be pruned without mutating data. Not all competing libraries offer this. | LOW | Pass-through to FFI; no extra Java logic. |
| **Precheck mode for rebuild** | Go exposes `WithRebuildPrecheck()`. Java `precheck(boolean)` on rebuild builder lets users validate preconditions without executing rebuild. | LOW | Pass-through to FFI. |
| **Backup manifest as first-class object** | Go returns `BackupManifest` with schema version, timestamps, checksums. Most DB wrappers return void from backup. Returning a structured manifest enables audit trails. | LOW | POJO with `schemaVersion`, `mode`, `createdAt`, `fileCount`, `totalBytes`, `files[]`, etc. |
| **Builder validation at build-time** | Builders should reject invalid configs before FFI call: empty destination path, invalid port ranges, watermark low > high. Fail fast with clear messages. | LOW | RocksDB validates on `open()`, not on builder. Validating at `build()` is a better DX. Already done in Go's `resolveXxxOptions` functions. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems for this specific project.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **JDBC/DataSource interface** | "I want to use Chroma like a SQL database" | ChromaDB is not a relational database. JDBC's `Connection`/`Statement`/`ResultSet` model does not map to vector operations. Implementing JDBC would be a massive mismatch creating a false contract. | Provide clean, purpose-built `ChromaRuntime` / `ServerSession` / `EmbeddedSession` APIs. |
| **Async/CompletableFuture API** | "I want non-blocking operations" | All FFI calls go through a single global lock (`ffiMu` in Go, same constraint in Java). True async would be misleading -- it would just offload to a thread pool that serializes anyway. Adds complexity without real concurrency gain. | Keep synchronous API. Users can wrap in their own executor if needed. Document that FFI calls are serialized. |
| **Connection pooling** | "I want a pool of embedded sessions" | Chroma's embedded mode is a single in-process runtime. Multiple "connections" to the same embedded instance don't make sense. Unlike SQL databases, there's no connection multiplexing benefit. | Single `EmbeddedSession` per runtime. Document this clearly. |
| **Auto-reconnect on server mode** | "If the server crashes, reconnect automatically" | The Java wrapper starts and owns the server process. If the server crashes, the native handle is invalid. "Reconnecting" would mean restarting, which has data integrity implications. | Provide clear error reporting. Let users explicitly restart. |
| **Generic config file loading** | "Load config from application.properties or Spring config" | Adds framework dependencies (Spring, MicroProfile). Chroma config is YAML-based at the FFI level. | Provide `rawYaml(String)` on builders. Users integrate with their frameworks. |
| **Automatic backup scheduling** | "Run backups every N hours" | Scheduling belongs in application code or infrastructure, not in a database wrapper library. Adds thread management complexity and lifecycle concerns. | Expose one-shot `backup()` method. Let users call it from their scheduler. |
| **Checked exceptions on every method** | "Proper Java error handling requires checked exceptions" | ChromaException is already unchecked (RuntimeException). Changing to checked would break existing users and force try/catch boilerplate on every call, including simple ones like `version()`. Modern Java convention (DuckDB, RocksDB) trends toward unchecked for FFI wrappers. | Keep `ChromaException` as unchecked RuntimeException. Add error codes for programmatic handling. |

---

## Feature Dependencies

```
ChromaRuntime.init() [EXISTING]
    |
    +--- version() [EXISTING]
    |
    +--- startEmbedded(configYaml) [EXISTING]
    |       |
    |       +--- EmbeddedSession.handle() [EXISTING]
    |       |
    |       +--- EmbeddedSession.close() [EXISTING]
    |       |
    |       +--- EmbeddedSession.backup(options)  [NEW - requires backup file I/O]
    |       |       +--- BackupOptions builder [NEW]
    |       |       +--- BackupManifest result [NEW]
    |       |
    |       +--- EmbeddedSession.rebuildCollection(name, options)  [NEW]
    |       |       +--- RebuildOptions builder [NEW]
    |       |       +--- RebuildCollectionResult result [NEW]
    |       |
    |       +--- EmbeddedSession.compactCollection(request)  [NEW]
    |       |       +--- CompactCollectionRequest [NEW]
    |       |       +--- CompactionResult result [NEW]
    |       |
    |       +--- EmbeddedSession.compactAll(request)  [NEW]
    |       |
    |       +--- EmbeddedSession.pruneCollectionWAL(name, options)  [NEW]
    |       |       +--- WALPruneOptions builder [NEW]
    |       |       +--- WALPruneResult result [NEW]
    |       |
    |       +--- EmbeddedSession.pruneAllWAL(options)  [NEW]
    |
    +--- EmbeddedConfigBuilder  [NEW - produces configYaml for startEmbedded]
    |
    +--- startServer(configYaml)  [NEW]
    |       |
    |       +--- ServerSession.port() [NEW]
    |       +--- ServerSession.address() [NEW]
    |       +--- ServerSession.url() [NEW]
    |       +--- ServerSession.stop() [NEW]
    |       +--- ServerSession.close() [NEW - AutoCloseable]
    |       |
    |       +--- ServerSession.backup(options)  [NEW - stop, backup, restart]
    |       +--- ServerSession.rebuildCollection(name, options) [NEW - stop, embed, rebuild, restart]
    |       +--- ServerSession.compactCollection(request) [NEW - stop, embed, compact, restart]
    |       +--- ServerSession.compactAll(request) [NEW - stop, embed, compactAll, restart]
    |       +--- ServerSession.pruneCollectionWAL(name, options) [NEW - stop, embed, prune, restart]
    |       +--- ServerSession.pruneAllWAL(options) [NEW - stop, embed, pruneAll, restart]
    |
    +--- ServerConfigBuilder  [NEW - produces configYaml for startServer]
```

### Dependency Notes

- **Config builders depend on nothing** -- they are pure Java string-building utilities producing YAML. Can be implemented first.
- **`startServer()` requires 5 new FFI bindings** -- `chroma_server_start_from_string`, `chroma_server_port`, `chroma_server_address`, `chroma_server_stop`, `chroma_server_free`. Both JNA and Panama must bind these.
- **`ServerSession` mirrors `EmbeddedSession`** pattern -- same `AtomicBoolean` + handle lifecycle. Can reuse the design.
- **Maintenance operations on `EmbeddedSession` require new FFI bindings** -- `chroma_embedded_rebuild_collection`, `chroma_embedded_compact_collection`, `chroma_embedded_compact_all`, `chroma_embedded_prune_wal_collection`, `chroma_embedded_prune_wal_all`. These accept JSON request, return JSON response.
- **Maintenance operations on `ServerSession` require `startEmbedded`** -- Go's pattern is: stop server, open temporary embedded runtime, run operation, close embedded, restart server. Java must implement the same stop-embed-op-restart choreography.
- **Result POJOs depend on nothing** -- pure data classes. Can be implemented independently.
- **Backup on `EmbeddedSession`** has the highest complexity because it involves Java-side file I/O (directory copy, manifest writing, SHA-256 checksums) in addition to FFI lifecycle management.

---

## MVP Definition

### Phase 1: Foundation (launch first)

- [x] `EmbeddedConfigBuilder` -- pure Java, no FFI, enables programmatic config for existing `startEmbedded`
- [x] `ServerConfigBuilder` -- pure Java, no FFI, produces YAML for server startup
- [x] Result POJOs (`BackupManifest`, `RebuildCollectionResult`, `CompactionResult`, `CompactionCollectionResult`, `WALPruneResult`, `WALPruneCollectionResult`) -- pure data, needed by all operations
- [x] Option/request POJOs (`CompactCollectionRequest`, `CompactAllRequest`) -- simple value objects

### Phase 2: Server Lifecycle

- [x] FFI bindings for `chroma_server_start_from_string`, `chroma_server_port`, `chroma_server_address`, `chroma_server_stop`, `chroma_server_free` in both JNA and Panama
- [x] `ServerSession` class with `port()`, `address()`, `url()`, `stop()`, `close()` -- AutoCloseable
- [x] `ChromaRuntime.startServer(String configYaml)` returning `ServerSession`
- [x] Integration tests for server start/stop/accessors

### Phase 3: Embedded Maintenance Operations

- [x] FFI bindings for `chroma_embedded_rebuild_collection`, `chroma_embedded_compact_collection`, `chroma_embedded_compact_all`, `chroma_embedded_prune_wal_collection`, `chroma_embedded_prune_wal_all` in both JNA and Panama
- [x] `RebuildOptions` builder with `tenantId`, `databaseName`, `precheck`, `keepBackup`
- [x] `WALPruneOptions` builder with `tenantId`, `databaseName`, `dryRun`, `vacuum`, `maxAge`, `maxBytes`, `watermark`
- [x] Methods on `EmbeddedSession`: `rebuildCollection`, `compactCollection`, `compactAll`, `pruneCollectionWAL`, `pruneAllWAL`
- [x] Integration tests for each embedded maintenance operation

### Phase 4: Server Maintenance + Backup

- [x] Server maintenance operations (rebuild, compact, prune) using stop-embed-op-restart pattern
- [x] Backup implementation for both `EmbeddedSession` and `ServerSession` (file copy, manifest, checksums)
- [x] `BackupOptions` builder
- [x] Integration tests for backup and server maintenance

### Add After Validation (v1.x)

- [ ] `ChromaErrorCode` enum on `ChromaException` -- useful but not blocking parity
- [ ] `BackupEngine`-style backup management (list backups, purge old, restore) -- Go doesn't have this yet either

### Future Consideration (v2+)

- [ ] Embedded data operations (CRUD for collections, documents, queries) -- massive scope, separate milestone
- [ ] Java-native observability (OpenTelemetry integration) -- depends on OTel config working through FFI

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Server lifecycle (start/stop/close) | HIGH | MEDIUM (5 FFI bindings x2 backends) | P1 |
| ServerConfigBuilder | HIGH | LOW (pure Java) | P1 |
| EmbeddedConfigBuilder | HIGH | LOW (pure Java) | P1 |
| Server accessors (port/address/url) | HIGH | LOW (derive from FFI returns) | P1 |
| Compaction API (embedded) | MEDIUM | MEDIUM (2 FFI bindings x2 backends) | P1 |
| Rebuild API (embedded) | MEDIUM | MEDIUM (1 FFI binding x2 backends) | P1 |
| WAL Prune API (embedded) | MEDIUM | MEDIUM (2 FFI bindings x2 backends) | P1 |
| Result POJOs | HIGH | LOW (pure data classes) | P1 |
| Option builders | HIGH | LOW (pure Java) | P1 |
| Server maintenance ops | MEDIUM | HIGH (stop-embed-restart choreography) | P2 |
| Backup (embedded + server) | MEDIUM | HIGH (file I/O + lifecycle + manifest) | P2 |
| Error codes | LOW | LOW (enum + exception field) | P3 |

**Priority key:**
- P1: Must have for API parity claim
- P2: Should have; completes maintenance story
- P3: Nice to have; improves DX

---

## Competitor Feature Analysis

| Feature | H2 Database | DuckDB Java | RocksDB Java | Our Approach |
|---------|-------------|-------------|--------------|--------------|
| Server start/stop | `Server.createTcpServer(...).start()` / `.stop()` -- factory+lifecycle | N/A (JDBC only) | N/A (embedded only) | `runtime.startServer(yaml)` returns `ServerSession` with `AutoCloseable` |
| Configuration | String args array (`"-tcpPort", "9092"`) | `Properties` passed to `DriverManager` | Fluent options: `new Options().setCreateIfMissing(true)` | Fluent builder: `ServerConfigBuilder.create().port(8000).build()` -- produces YAML |
| Port/URL accessors | `getPort()`, `getURL()`, `getStatus()` | N/A | N/A | `port()`, `address()`, `url()` on `ServerSession` |
| Backup | N/A (file-level) | `COPY FROM DATABASE` SQL | `BackupEngine.createNewBackup(db)` with `BackupableDBOptions` | `session.backup(BackupOptions)` returning `BackupManifest` |
| Compaction | Automatic | `CHECKPOINT` SQL | `db.compactRange()` | `session.compactCollection(request)` / `compactAll(request)` returning `CompactionResult` |
| Resource cleanup | Server has no AutoCloseable | `Connection.close()` | `RocksObject` extends `AutoCloseable` with finalizer | Both `ServerSession` and `EmbeddedSession` implement `AutoCloseable` with `AtomicBoolean` guard |
| Exception style | Checked `SQLException` | Checked `SQLException` (JDBC) | Checked `RocksDBException` | Unchecked `ChromaException` (RuntimeException) -- matches existing API |
| Maintenance results | void | void | void | Typed result POJOs with per-collection breakdown, timing, error details |

---

## FFI Symbol Inventory

New FFI symbols required (all exist in the Rust shim; Java just needs to bind them):

| FFI Symbol | Category | JNA Signature | Panama Descriptor |
|---|---|---|---|
| `chroma_server_start_from_string` | Server | `Pointer(String)` | `ADDRESS(ADDRESS)` |
| `chroma_server_port` | Server | `int(Pointer)` | `JAVA_INT(ADDRESS)` |
| `chroma_server_address` | Server | `Pointer(Pointer)` | `ADDRESS(ADDRESS)` |
| `chroma_server_stop` | Server | `int(Pointer)` | `JAVA_INT(ADDRESS)` |
| `chroma_server_free` | Server | `void(Pointer)` | `VOID(ADDRESS)` |
| `chroma_embedded_rebuild_collection` | Maintenance | `Pointer(Pointer, String)` | `ADDRESS(ADDRESS, ADDRESS)` |
| `chroma_embedded_compact_collection` | Maintenance | `Pointer(Pointer, String)` | `ADDRESS(ADDRESS, ADDRESS)` |
| `chroma_embedded_compact_all` | Maintenance | `Pointer(Pointer, String)` | `ADDRESS(ADDRESS, ADDRESS)` |
| `chroma_embedded_prune_wal_collection` | Maintenance | `Pointer(Pointer, String)` | `ADDRESS(ADDRESS, ADDRESS)` |
| `chroma_embedded_prune_wal_all` | Maintenance | `Pointer(Pointer, String)` | `ADDRESS(ADDRESS, ADDRESS)` |

Total: 10 new FFI bindings, each implemented in both JNA and Panama = 20 binding implementations.

Already bound (reusable): `chroma_version`, `chroma_get_last_error`, `chroma_string_free`, `chroma_embedded_start_from_string`, `chroma_embedded_free` = 5 per backend.

---

## Sources

- Go API source code: `chroma.go`, `config.go`, `embedded.go`, `backup.go`, `rebuild.go`, `compaction.go`, `wal_prune.go` -- read directly from codebase (HIGH confidence)
- Java scaffold source: `ChromaRuntime.java`, `EmbeddedSession.java`, `JnaChromaRuntime.java`, `PanamaChromaRuntime.java` -- read directly from codebase (HIGH confidence)
- [H2 Server JavaDoc](https://www.h2database.com/javadoc/org/h2/tools/Server.html) -- HIGH confidence
- [DuckDB Java JDBC Client docs](https://duckdb.org/docs/stable/clients/java) -- HIGH confidence
- [RocksDB Java Basics wiki](https://github.com/facebook/rocksdb/wiki/RocksJava-Basics) -- HIGH confidence
- [RocksDB BackupEngine source](https://github.com/facebook/rocksdb/blob/master/java/src/main/java/org/rocksdb/BackupEngine.java) -- HIGH confidence
- [RocksDB Java FFI blog post](https://rocksdb.org/blog/2024/02/20/foreign-function-interface.html) -- MEDIUM confidence
- [ArcadeDB Embedded Mode](https://arcadedb.com/embedded.html) -- MEDIUM confidence
- [Java AutoCloseable best practices](https://zetcode.com/java/autoclosable/) -- HIGH confidence

---
*Feature research for: Java API surface mirroring Go API (v0.5.0 milestone)*
*Researched: 2026-03-21*
