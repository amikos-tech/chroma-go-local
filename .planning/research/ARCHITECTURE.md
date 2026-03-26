# Architecture Patterns: Java API Surface Mirroring Go API

**Domain:** Multi-language FFI bindings (Java JNA + Panama wrapping Rust shim) -- expanding scaffold to full API parity with Go
**Researched:** 2026-03-21
**Overall confidence:** HIGH (based on direct codebase analysis of existing Go API, current Java scaffold, and Rust FFI symbols)

---

## Current Architecture (Existing Java Scaffold)

The Java scaffold (v0.3.x) provides a minimal 3-module Gradle project:

```
java/
  build.gradle.kts              # Root Gradle build, version management
  settings.gradle.kts           # includes: core, jna, panama
  core/
    src/main/java/.../core/
      ChromaRuntime.java         # Interface: version(), startEmbedded(yaml), close()
      EmbeddedSession.java       # Handle wrapper: long handle, LongConsumer closeAction
      ChromaException.java       # Unchecked exception type
  jna/
    src/main/java/.../jna/
      JnaChromaRuntime.java      # JNA impl: loads library, binds 5 symbols
    src/test/java/.../jna/
      JnaChromaRuntimeTest.java
  panama/
    src/main/java/.../panama/
      PanamaChromaRuntime.java   # Panama impl: Arena, MethodHandle for 5 symbols
    src/test/java/.../panama/
      PanamaChromaRuntimeTest.java
```

### Current FFI Bindings (5 symbols only)

| FFI Symbol | JNA Binding | Panama Binding |
|------------|-------------|----------------|
| `chroma_version` | `Pointer chroma_version()` | `MethodHandle chromaVersion` |
| `chroma_get_last_error` | `Pointer chroma_get_last_error()` | `MethodHandle chromaGetLastError` |
| `chroma_string_free` | `void chroma_string_free(Pointer)` | `MethodHandle chromaStringFree` |
| `chroma_embedded_start_from_string` | `Pointer chroma_embedded_start_from_string(String)` | `MethodHandle chromaEmbeddedStartFromString` |
| `chroma_embedded_free` | `void chroma_embedded_free(Pointer)` | `MethodHandle chromaEmbeddedFree` |

### Key Observations About Current Design

1. **ChromaRuntime is the entry point** -- it loads the native library and provides factory methods
2. **EmbeddedSession is a handle wrapper** -- stores `long handle` with `AtomicBoolean closed` guard and delegated close action
3. **Close actions are injected** -- `EmbeddedSession` receives a `LongConsumer` that calls the FFI free function, keeping FFI details in the runtime implementation
4. **Thread safety** -- `AtomicBoolean` guards on both ChromaRuntime and EmbeddedSession
5. **No server mode** -- only embedded start/close is implemented
6. **core module has zero FFI dependencies** -- it defines interfaces and handle wrappers only

---

## Go API Surface to Mirror

The Go API provides two runtime modes (Server and Embedded), each with builder configuration, backup, rebuild, compaction, and WAL prune operations. Here is the complete API surface that Java must mirror:

### Server Mode

| Go Type/Function | Purpose | FFI Symbols Used |
|-----------------|---------|------------------|
| `Server` struct | Running server handle with port/address/URL | holds uintptr handle |
| `StartServerConfig` | Raw config (path or YAML string) | -- |
| `ServerConfig` | Typed builder fields (port, address, persist path, etc.) | -- |
| `ServerOption` func type | Functional options for `NewServer` | -- |
| `StartServer(config)` | Start from raw YAML | `chroma_server_start`, `chroma_server_start_from_string`, `chroma_server_port`, `chroma_server_address`, `chroma_server_persist_path` |
| `NewServer(opts...)` | Start from builder options | Same as above (builds YAML internally) |
| `Server.Port()` | Get bound port | cached from startup |
| `Server.Address()` | Get bound address | cached from startup |
| `Server.URL()` | Get full HTTP URL | computed from cached port+address |
| `Server.Stop()` | Graceful stop (no free) | `chroma_server_stop` |
| `Server.Close()` | Stop + free resources | `chroma_server_stop`, `chroma_server_free` |
| `Server.Backup(opts...)` | Stop, snapshot, restart | stop/free + embedded backup + restart |
| `Server.RebuildCollection(name, opts...)` | Stop, rebuild via temp embedded, restart | stop/free + embedded rebuild + restart |
| `Server.CompactCollection(req)` | Stop, compact via temp embedded, restart | stop/free + embedded compact + restart |
| `Server.CompactAll(req)` | Stop, compact all via temp embedded, restart | stop/free + embedded compact_all + restart |
| `Server.PruneCollectionWAL(name, opts...)` | Stop, prune via temp embedded, restart | stop/free + embedded prune + restart |
| `Server.PruneAllWAL(opts...)` | Stop, prune all via temp embedded, restart | stop/free + embedded prune_all + restart |

### Embedded Mode (Extension of Existing)

| Go Type/Function | Purpose | FFI Symbols Used |
|-----------------|---------|------------------|
| `Embedded.Backup(opts...)` | Close, snapshot, reopen | `chroma_embedded_free` + filesystem ops + restart |
| `Embedded.RebuildCollection(name, opts...)` | Rebuild index for one collection | `chroma_embedded_rebuild_collection` |
| `Embedded.CompactCollection(req)` | Compact one collection | `chroma_embedded_compact_collection` |
| `Embedded.CompactAll(req)` | Compact all collections | `chroma_embedded_compact_all` |
| `Embedded.PruneCollectionWAL(name, opts...)` | Prune WAL for one collection | `chroma_embedded_prune_wal_collection` |
| `Embedded.PruneAllWAL(opts...)` | Prune WAL for all collections | `chroma_embedded_prune_wal_all` |

### Option/Result Types

| Go Type | Fields | Java Equivalent Pattern |
|---------|--------|------------------------|
| `BackupOption` (interface) | destination, includeMetadata, leaveStopped/leaveClosed | Builder class |
| `BackupManifest` | schemaVersion, mode, createdAt, files, totals | Record or immutable POJO |
| `RebuildCollectionOption` (interface) | tenantID, databaseName, precheck, keepBackup | Builder class |
| `RebuildCollectionResult` | collectionID, name, rebuilt, counts, timing | Record or immutable POJO |
| `CompactCollectionRequest` | name, tenantID, databaseName | Builder class |
| `CompactAllRequest` | tenantID, databaseName | Builder class |
| `CompactionResult` | collectionCount, durationMS, totals, per-collection | Record or immutable POJO |
| `WALPruneOption` (interface) | tenantID, databaseName, dryRun, vacuum, maxAge, maxBytes, watermark | Builder class |
| `WALPruneResult` | collectionCount, durationMS, totals, per-collection | Record or immutable POJO |

---

## Recommended Architecture

### Decision: Separate `ChromaServer` Interface, Do NOT Grow `ChromaRuntime`

**Recommendation:** Introduce a new `ChromaServer` interface in core. Do NOT add server methods to `ChromaRuntime`.

**Rationale:**

1. **Distinct lifecycle semantics** -- `ChromaRuntime` is the library loader (one per process); `ChromaServer` is a running server instance (start/stop/query port). These are different concerns. In Go, `Init()` loads the library and `NewServer()` creates the server -- these are separate operations.

2. **Handle separation** -- Server and Embedded each own different native handles (`chroma_server_*` vs `chroma_embedded_*` symbols). Mixing them in one interface creates confusion about which handle is active.

3. **Existing EmbeddedSession already splits from ChromaRuntime** -- The scaffold already established the pattern: `ChromaRuntime.startEmbedded()` returns an `EmbeddedSession` that owns the native handle. Server should follow the same pattern: `ChromaRuntime.startServer()` returns a `ChromaServer`.

4. **AutoCloseable composability** -- Both `EmbeddedSession` and `ChromaServer` implement `AutoCloseable` independently. Users can `try-with-resources` each one without coupling their lifecycles.

### New Interface Hierarchy

```
ChromaRuntime (existing, slightly extended)
  |-- version() : String                    [existing]
  |-- startEmbedded(yaml) : EmbeddedSession [existing]
  |-- startServer(yaml) : ChromaServer      [NEW]
  |-- close()                               [existing]

ChromaServer (NEW interface in core)
  |-- port() : int
  |-- address() : String
  |-- url() : String
  |-- stop()
  |-- backup(BackupOptions) : BackupManifest
  |-- rebuildCollection(name, RebuildOptions) : RebuildResult
  |-- compactCollection(CompactCollectionRequest) : CompactionResult
  |-- compactAll(CompactAllRequest) : CompactionResult
  |-- pruneCollectionWAL(name, WALPruneOptions) : WALPruneResult
  |-- pruneAllWAL(WALPruneOptions) : WALPruneResult
  |-- close()  [AutoCloseable -- calls stop + free]

EmbeddedSession (existing, extended with maintenance ops)
  |-- handle() : long                       [existing]
  |-- backup(BackupOptions) : BackupManifest          [NEW]
  |-- rebuildCollection(name, RebuildOptions) : RebuildResult  [NEW]
  |-- compactCollection(CompactCollectionRequest) : CompactionResult  [NEW]
  |-- compactAll(CompactAllRequest) : CompactionResult  [NEW]
  |-- pruneCollectionWAL(name, WALPruneOptions) : WALPruneResult  [NEW]
  |-- pruneAllWAL(WALPruneOptions) : WALPruneResult  [NEW]
  |-- close()                               [existing]
```

### Why EmbeddedSession Grows vs. New Wrapper

The existing `EmbeddedSession` already owns the native embedded handle. The maintenance operations (backup, rebuild, compaction, WAL prune) all operate on that handle via `chroma_embedded_*` FFI symbols. Creating a separate wrapper would require either:
- Exposing the raw handle (breaks encapsulation)
- Creating a wrapper that delegates back to EmbeddedSession for handle access (pointless indirection)

Instead, `EmbeddedSession` should grow to accept additional `LongConsumer`-style action callbacks for each new operation, injected by the runtime implementation. However, this approach quickly becomes unwieldy with 6+ operation callbacks.

**Better approach:** Change `EmbeddedSession` to accept a single operations interface rather than individual callbacks. See the detailed design below.

---

## Detailed Component Design

### Module Placement

| Component | Module | Rationale |
|-----------|--------|-----------|
| `ChromaServer` interface | core | Defines the server contract, no FFI dependency |
| `ChromaServerConfig` builder | core | Pure Java config builder, no FFI dependency |
| `BackupOptions` builder | core | Pure Java, no FFI dependency |
| `RebuildOptions` builder | core | Pure Java, no FFI dependency |
| `CompactCollectionRequest` | core | Pure Java request type |
| `CompactAllRequest` | core | Pure Java request type |
| `WALPruneOptions` builder | core | Pure Java, no FFI dependency |
| Result types (BackupManifest, RebuildResult, CompactionResult, WALPruneResult) | core | Immutable data classes, no FFI dependency |
| `JnaChromaServer` implementation | jna | JNA-specific FFI calls |
| `PanamaChromaServer` implementation | panama | Panama-specific FFI calls |
| Extended EmbeddedSession operations | core (interface) + jna/panama (implementation via callback injection) | Operations interface bridges core and backends |

### EmbeddedSession: Operations Delegate Pattern

The current `EmbeddedSession` receives a `LongConsumer closeAction`. For the extended API with 6+ new operations, inject an operations interface instead of individual lambdas:

```java
// core module
public interface EmbeddedOperations {
    void close(long handle);
    BackupManifest backup(long handle, BackupOptions options);
    RebuildResult rebuildCollection(long handle, String name, RebuildOptions options);
    CompactionResult compactCollection(long handle, CompactCollectionRequest request);
    CompactionResult compactAll(long handle, CompactAllRequest request);
    WALPruneResult pruneCollectionWAL(long handle, String name, WALPruneOptions options);
    WALPruneResult pruneAllWAL(long handle, WALPruneOptions options);
}
```

Each JNA/Panama runtime implements `EmbeddedOperations`, and `EmbeddedSession` delegates to it:

```java
// core module
public final class EmbeddedSession implements AutoCloseable {
    private final long handle;
    private final EmbeddedOperations ops;
    private final AtomicBoolean closed;

    public EmbeddedSession(long handle, EmbeddedOperations ops) { ... }

    public BackupManifest backup(BackupOptions options) {
        ensureOpen();
        return ops.backup(handle, options);
    }
    // ... other operations delegate similarly

    @Override
    public void close() {
        if (closed.compareAndSet(false, true)) {
            ops.close(handle);
        }
    }
}
```

**Migration note:** The existing `EmbeddedSession(long, LongConsumer)` constructor must be deprecated or replaced. Since this is pre-1.0 and the only consumers are JnaChromaRuntime and PanamaChromaRuntime (both in this project), this is a safe breaking change within the scaffold.

### ChromaServer: Parallel Handle Wrapper

`ChromaServer` follows the same pattern as `EmbeddedSession` -- it wraps a native handle and delegates operations to a backend-specific interface:

```java
// core module
public interface ServerOperations {
    int port(long handle);
    String address(long handle);
    void stop(long handle);
    void free(long handle);
    BackupManifest backup(long handle, BackupOptions options);
    RebuildResult rebuildCollection(long handle, String name, RebuildOptions options);
    CompactionResult compactCollection(long handle, CompactCollectionRequest request);
    CompactionResult compactAll(long handle, CompactAllRequest request);
    WALPruneResult pruneCollectionWAL(long handle, String name, WALPruneOptions options);
    WALPruneResult pruneAllWAL(long handle, WALPruneOptions options);
}

// core module
public final class ChromaServer implements AutoCloseable {
    private final long handle;
    private final ServerOperations ops;
    private final int port;
    private final String address;
    private final AtomicBoolean closed;

    // port and address cached at construction (matches Go pattern)
    public ChromaServer(long handle, ServerOperations ops, int port, String address) { ... }

    public int port() { return port; }
    public String address() { return address; }
    public String url() { return "http://" + address + ":" + port; }

    public void stop() { ensureOpen(); ops.stop(handle); }

    @Override
    public void close() {
        if (closed.compareAndSet(false, true)) {
            try { ops.stop(handle); } catch (Exception ignored) {}
            ops.free(handle);
        }
    }
}
```

### Server Maintenance Operations: Server-Mode vs. Embedded-Mode Difference

In Go, server-mode maintenance (backup, rebuild, compact, prune) follows a complex pattern:
1. Stop the server
2. Start a temporary embedded instance with the same config
3. Run the maintenance operation on the embedded instance
4. Close the temporary embedded instance
5. Restart the server

**Java must replicate this pattern.** The `ServerOperations` implementation in JNA/Panama must orchestrate this stop-start-embedded-restart flow internally. The user-facing `ChromaServer.backup()` call should be simple -- the complexity is in the backend implementation.

This is an implementation detail of `JnaServerOperations` and `PanamaServerOperations`, not something exposed in the core interface.

### Builder Pattern for Configuration

Go uses functional options (`ServerOption func(*ServerConfig)`). Java should use classic builder pattern:

```java
// core module
public final class ChromaServerConfig {
    private int port = 8000;
    private String listenAddress = "127.0.0.1";
    private int maxPayloadSizeBytes = 40 * 1024 * 1024;
    private String persistPath = "./chroma";
    private String sqliteFilename = "chroma.sqlite3";
    private boolean allowReset = false;
    // ... other fields

    private ChromaServerConfig() {}

    public static ChromaServerConfig builder() { return new ChromaServerConfig(); }

    public ChromaServerConfig port(int port) { this.port = port; return this; }
    public ChromaServerConfig listenAddress(String addr) { this.listenAddress = addr; return this; }
    // ... other setters

    public String toYaml() { /* builds YAML string from fields */ }
}
```

And the runtime gains an overloaded `startServer`:

```java
// ChromaRuntime interface
ChromaServer startServer(String configYaml);
ChromaServer startServer(ChromaServerConfig config);  // convenience: calls toYaml() internally
```

### Go-to-Java Type Mapping

| Go Type | Java Type | Module | Pattern |
|---------|-----------|--------|---------|
| `Server` struct | `ChromaServer` class | core | Handle wrapper + operations delegate |
| `ServerConfig` struct | `ChromaServerConfig` class | core | Mutable builder with `toYaml()` |
| `ServerOption` func | Not needed | -- | Builder pattern replaces functional options |
| `StartServerConfig` struct | `String configYaml` parameter | -- | Direct parameter (YAML string or config path) |
| `Embedded` struct | `EmbeddedSession` class (existing) | core | Extended with operations delegate |
| `EmbeddedConfig` struct | `EmbeddedConfig` class | core | Mutable builder with `toYaml()` |
| `BackupOption` interface | `BackupOptions` builder class | core | Builder with `build()` returning immutable snapshot |
| `BackupManifest` struct | `BackupManifest` record/class | core | Immutable data |
| `RebuildCollectionOption` interface | `RebuildOptions` builder class | core | Builder |
| `RebuildCollectionResult` struct | `RebuildResult` record/class | core | Immutable data |
| `CompactCollectionRequest` struct | `CompactCollectionRequest` builder/class | core | Builder or direct constructor |
| `CompactAllRequest` struct | `CompactAllRequest` builder/class | core | Builder or direct constructor |
| `CompactionResult` struct | `CompactionResult` record/class | core | Immutable data |
| `CompactionCollectionResult` struct | `CompactionCollectionResult` record/class | core | Immutable data |
| `WALPruneOption` interface | `WALPruneOptions` builder class | core | Builder |
| `WALPruneResult` struct | `WALPruneResult` record/class | core | Immutable data |
| `WALPruneCollectionResult` struct | `WALPruneCollectionResult` record/class | core | Immutable data |

### Note on Records vs Classes for Result Types

Java records (Java 16+) are ideal for immutable result types since core targets Java 17+. Use records for all result/manifest types:

```java
public record BackupManifest(
    String schemaVersion,
    String mode,
    Instant createdAt,
    String wrapperVersion,
    List<String> sourcePaths,
    String destinationPath,
    String snapshotPath,
    String manifestPath,
    boolean includeMetadata,
    int fileCount,
    long totalBytes,
    List<BackupFileMetadata> files
) {}
```

---

## New FFI Symbols Required Per Backend

### Server Lifecycle Symbols (7 new symbols per backend)

| FFI Symbol | JNA Signature | Panama Descriptor |
|------------|---------------|-------------------|
| `chroma_server_start` | `Pointer chroma_server_start(String configPath)` | `(ADDRESS) -> ADDRESS` |
| `chroma_server_start_from_string` | `Pointer chroma_server_start_from_string(String configYaml)` | `(ADDRESS) -> ADDRESS` |
| `chroma_server_port` | `int chroma_server_port(Pointer handle)` | `(ADDRESS) -> JAVA_INT` |
| `chroma_server_address` | `Pointer chroma_server_address(Pointer handle)` | `(ADDRESS) -> ADDRESS` |
| `chroma_server_persist_path` | `Pointer chroma_server_persist_path(Pointer handle)` | `(ADDRESS) -> ADDRESS` |
| `chroma_server_stop` | `int chroma_server_stop(Pointer handle)` | `(ADDRESS) -> JAVA_INT` |
| `chroma_server_free` | `void chroma_server_free(Pointer handle)` | `(ADDRESS) -> VOID` |

### Embedded Maintenance Symbols (5 new symbols per backend)

| FFI Symbol | JNA Signature | Panama Descriptor |
|------------|---------------|-------------------|
| `chroma_embedded_rebuild_collection` | `Pointer chroma_embedded_rebuild_collection(Pointer handle, String requestJson)` | `(ADDRESS, ADDRESS) -> ADDRESS` |
| `chroma_embedded_compact_collection` | `Pointer chroma_embedded_compact_collection(Pointer handle, String requestJson)` | `(ADDRESS, ADDRESS) -> ADDRESS` |
| `chroma_embedded_compact_all` | `Pointer chroma_embedded_compact_all(Pointer handle, String requestJson)` | `(ADDRESS, ADDRESS) -> ADDRESS` |
| `chroma_embedded_prune_wal_collection` | `Pointer chroma_embedded_prune_wal_collection(Pointer handle, String requestJson)` | `(ADDRESS, ADDRESS) -> ADDRESS` |
| `chroma_embedded_prune_wal_all` | `Pointer chroma_embedded_prune_wal_all(Pointer handle, String requestJson)` | `(ADDRESS, ADDRESS) -> ADDRESS` |

### Symbols Already Bound (reused, no new work)

| FFI Symbol | Already In |
|------------|-----------|
| `chroma_version` | JNA + Panama |
| `chroma_get_last_error` | JNA + Panama |
| `chroma_string_free` | JNA + Panama |
| `chroma_embedded_start_from_string` | JNA + Panama |
| `chroma_embedded_free` | JNA + Panama |

### Additional Embedded Symbol Needed for Backup

| FFI Symbol | Purpose |
|------------|---------|
| `chroma_embedded_persist_path` | Needed to resolve persist path for backup source directory |

**Total new FFI bindings per backend: 13 symbols (7 server + 5 embedded maintenance + 1 embedded persist path)**

---

## Data Flow Diagrams

### Server Startup Flow

```
User code:
  ChromaServerConfig config = ChromaServerConfig.builder().port(8000).persistPath("./data");
  ChromaServer server = runtime.startServer(config);

ChromaRuntime (JNA/Panama impl):
  1. config.toYaml() -> YAML string
  2. FFI: chroma_server_start_from_string(yaml) -> native handle (Pointer/MemorySegment)
  3. FFI: chroma_server_port(handle) -> int port
  4. FFI: chroma_server_address(handle) -> String address
  5. Create ServerOperations impl (captures FFI bindings)
  6. return new ChromaServer(handle.address(), serverOps, port, address)
```

### Server-Mode Maintenance Flow (e.g., Backup)

```
User code:
  BackupManifest manifest = server.backup(BackupOptions.builder()
      .destination("/tmp/backup")
      .includeMetadata(true)
      .build());

ChromaServer:
  1. ensureOpen()
  2. ops.backup(handle, options)

ServerOperations (JNA/Panama impl):
  1. Acquire mutex (serialize maintenance ops)
  2. Cache server config YAML
  3. FFI: chroma_server_stop(handle) + chroma_server_free(handle)
  4. FFI: chroma_embedded_start_from_string(configYaml) -> embedded handle
  5. FFI: chroma_embedded_persist_path(embeddedHandle) -> persist path
  6. Java filesystem: copy persist dir to destination, write manifest
  7. FFI: chroma_embedded_free(embeddedHandle)
  8. FFI: chroma_server_start_from_string(configYaml) -> new server handle
  9. Update ChromaServer's handle, port, address
  10. Release mutex
  11. Return BackupManifest
```

**Important:** The backup filesystem operations (directory copy, manifest writing, SHA-256 hashing) are pure Java -- they do not go through FFI. This is the same approach Go takes (the `copyDirectory`, `writeManifest` functions are pure Go, not FFI calls). This logic should live in the core module or a shared utility, since it is backend-independent.

### Embedded-Mode Maintenance Flow (e.g., Rebuild)

```
User code:
  RebuildResult result = session.rebuildCollection("my_collection",
      RebuildOptions.builder().precheck(true).build());

EmbeddedSession:
  1. ensureOpen()
  2. ops.rebuildCollection(handle, "my_collection", options)

EmbeddedOperations (JNA/Panama impl):
  1. Serialize options to JSON: {"name":"my_collection","precheck":true,"keep_backup":true}
  2. FFI: chroma_embedded_rebuild_collection(handle, requestJson) -> response JSON pointer
  3. Read C string from response pointer
  4. FFI: chroma_string_free(responsePointer)
  5. Deserialize JSON -> RebuildResult
  6. Return RebuildResult
```

---

## Component Boundaries Summary

| Component | Module | Responsibility | Communicates With |
|-----------|--------|---------------|-------------------|
| `ChromaRuntime` interface | core | Library loader + factory for server/embedded sessions | Implemented by jna/panama |
| `ChromaServer` class | core | Server handle wrapper, delegates ops | `ServerOperations` impl in jna/panama |
| `EmbeddedSession` class | core | Embedded handle wrapper, delegates ops | `EmbeddedOperations` impl in jna/panama |
| `ServerOperations` interface | core | Contract for server FFI operations | Implemented by jna/panama |
| `EmbeddedOperations` interface | core | Contract for embedded FFI operations | Implemented by jna/panama |
| Config builders | core | Build YAML from typed fields | Used by runtime impls |
| Result types (records) | core | Immutable data carriers | Returned by operations |
| Option builders | core | Build typed option snapshots | Consumed by operations |
| `JnaChromaRuntime` | jna | JNA FFI binding + JnaServerOperations + JnaEmbeddedOperations | JNA library, core interfaces |
| `PanamaChromaRuntime` | panama | Panama FFI binding + PanamaServerOperations + PanamaEmbeddedOperations | Panama FFM API, core interfaces |
| Backup utilities | core | Filesystem copy, manifest writing, SHA-256 | Called by server/embedded ops impls |

---

## Patterns to Follow

### Pattern 1: Operations Delegate (Recommended for handle wrappers)

**What:** Handle wrapper classes in core delegate FFI operations to a backend-specific interface.

**When:** Any time a core class needs to call FFI symbols that differ between JNA and Panama.

**Why:** Keeps core free of FFI dependencies. The delegate interface is defined in core; implementations live in jna/panama. The handle wrapper (EmbeddedSession, ChromaServer) holds a reference to the delegate and forwards calls.

```java
// core module -- interface
public interface EmbeddedOperations {
    void close(long handle);
    RebuildResult rebuildCollection(long handle, String name, RebuildOptions options);
    // ...
}

// core module -- handle wrapper
public final class EmbeddedSession implements AutoCloseable {
    private final long handle;
    private final EmbeddedOperations ops;

    public RebuildResult rebuildCollection(String name, RebuildOptions options) {
        ensureOpen();
        return ops.rebuildCollection(handle, name, options);
    }
}

// jna module -- implementation
final class JnaEmbeddedOperations implements EmbeddedOperations {
    private final JnaBindings bindings;

    @Override
    public RebuildResult rebuildCollection(long handle, String name, RebuildOptions options) {
        String json = buildRequestJson(name, options);
        Pointer resp = bindings.chroma_embedded_rebuild_collection(new Pointer(handle), json);
        // ... parse response
    }
}
```

### Pattern 2: Builder-to-YAML for Configuration

**What:** Java builder classes produce YAML strings consumed by FFI start functions.

**When:** Server and embedded configuration.

**Why:** The Rust shim accepts YAML configuration. Go builds YAML from its `ServerConfig.toYAML()` and `EmbeddedConfig.toYAML()`. Java mirrors this exactly. Users get type-safe builders; the FFI layer receives the same YAML format.

```java
public final class ChromaServerConfig {
    private int port = 8000;
    // ...

    public ChromaServerConfig port(int port) { this.port = port; return this; }

    public String toYaml() {
        StringBuilder sb = new StringBuilder();
        sb.append("port: ").append(port).append('\n');
        // ... match Go's YAML output format exactly
        return sb.toString();
    }
}
```

### Pattern 3: JSON Request/Response for Maintenance Operations

**What:** Maintenance operations (rebuild, compact, prune) serialize requests as JSON, pass to FFI, and deserialize JSON responses.

**When:** All embedded maintenance operations.

**Why:** The Rust shim expects JSON-encoded request bodies and returns JSON-encoded results. Go does the same (`json.Marshal` for requests, `json.Unmarshal` for responses). Java should use the same wire format.

**JSON library consideration:** Use minimal JSON handling. Since the project avoids external dependencies in core where possible, options are:
- **org.json** (small, no extra deps) -- but adds a dependency
- **Manual StringBuilder** for simple request JSON (matching Go's `marshalRequestJSON`)
- **Jackson** (heavyweight) -- avoid

Recommendation: Use a lightweight JSON library or manual serialization for request building, and manual parsing for responses. The request structures are simple (5-10 fields), and response parsing can use basic string operations or a minimal parser. However, for robustness and maintainability, adding a small JSON dependency (e.g., `com.google.code.gson:gson` or keeping the manual approach) is acceptable.

### Pattern 4: Mutex Serialization for Server Maintenance

**What:** Server-mode maintenance operations (backup, rebuild, compact, prune) are mutually exclusive and require stopping the server.

**When:** Any maintenance operation on `ChromaServer`.

**Why:** Go uses `sync.Mutex` (backupMu) to serialize these operations. Java must use `ReentrantLock` or `synchronized` to achieve the same. The lock must be held across the entire stop-operate-restart cycle.

```java
public final class ChromaServer implements AutoCloseable {
    private final ReentrantLock maintenanceLock = new ReentrantLock();

    public BackupManifest backup(BackupOptions options) {
        maintenanceLock.lock();
        try {
            return ops.backup(handle, options);
        } finally {
            maintenanceLock.unlock();
        }
    }
}
```

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Growing ChromaRuntime Into a God Interface

**What:** Adding `startServer()`, `backup()`, `rebuildCollection()`, `compactCollection()`, etc. all to `ChromaRuntime`.

**Why bad:** `ChromaRuntime` represents the loaded native library, not a running server or embedded session. Mixing factory methods with operational methods violates SRP and makes the interface unwieldy. Users would need to track which methods require a server handle vs. an embedded handle.

**Instead:** `ChromaRuntime` stays lean: `version()`, `startEmbedded()`, `startServer()`, `close()`. All operational methods live on the returned handle wrappers (`ChromaServer`, `EmbeddedSession`).

### Anti-Pattern 2: Exposing Raw Native Handles in Core API

**What:** Making `ChromaServer.handle()` public or using `long handle` parameters in user-facing methods.

**Why bad:** Leaks native implementation details. Users might cache handles, pass them to wrong functions, or use them after close.

**Instead:** `ChromaServer` and `EmbeddedSession` encapsulate their handles. Only the `Operations` interfaces (which are package-private or internal) work with raw handles.

Note: `EmbeddedSession.handle()` is currently public. For backward compatibility during this milestone, it can remain public but should be documented as advanced/internal use. Long-term, it should be package-private or removed.

### Anti-Pattern 3: Backend-Specific Types in Core Module

**What:** Importing `com.sun.jna.Pointer` or `java.lang.foreign.MemorySegment` in core classes.

**Why bad:** Core module must compile with Java 17 and have zero FFI dependencies. JNA types in core break the module boundary.

**Instead:** All FFI types stay in jna/panama modules. Core uses `long` for handle values and interfaces for operations.

### Anti-Pattern 4: Separate JSON Libraries Per Module

**What:** JNA module uses Gson, Panama module uses Jackson, core uses org.json.

**Why bad:** Three different JSON approaches for the same wire format. Inconsistent serialization, triple the dependency surface.

**Instead:** Choose one approach and use it consistently. Core defines the serialization contract (e.g., `toJson()` on request builders, `fromJson()` static factory on result types). Both backends use core's serialization.

### Anti-Pattern 5: Backup Logic Duplicated in JNA and Panama

**What:** Implementing filesystem copy, manifest writing, and SHA-256 hashing separately in JnaServerOperations and PanamaServerOperations.

**Why bad:** Backup's filesystem operations are pure Java -- identical across backends. Duplicating them doubles maintenance burden and risks divergence.

**Instead:** Extract backup utilities into core (or a shared utility class). Backend operations call into core utilities for filesystem work, only using FFI for handle stop/start/persist-path operations.

---

## Build Order (Suggested Phase Structure)

The dependency chain determines build order. Each step must produce a compiling, testable increment.

### Phase 1: Core Foundation Types

**Build:** All new interfaces, builders, and result types in core module.

- `ChromaServer` interface (or class)
- `ServerOperations` interface
- `EmbeddedOperations` interface
- `ChromaServerConfig` builder
- `EmbeddedConfig` builder (Java equivalent)
- `BackupOptions` builder
- `BackupManifest` record
- `BackupFileMetadata` record
- `RebuildOptions` builder
- `RebuildResult` record
- `CompactCollectionRequest` class
- `CompactAllRequest` class
- `CompactionResult` record
- `CompactionCollectionResult` record
- `WALPruneOptions` builder
- `WALPruneResult` record
- `WALPruneCollectionResult` record
- Extend `ChromaRuntime` interface with `startServer(String configYaml)`
- Refactor `EmbeddedSession` to use `EmbeddedOperations` instead of `LongConsumer`

**Tests:** Unit tests for all builders (toYaml output matches Go format), all records (construction, field access).

**Why first:** Everything else depends on core types. No FFI needed. Compiles and tests independently.

### Phase 2: Server Lifecycle (JNA + Panama)

**Build:** Server start/stop/port/address/URL in both backends.

- Add 7 new server FFI symbol bindings to JNA `JnaBindings` interface
- Add 7 new server FFI symbol MethodHandles to Panama
- Implement `ChromaRuntime.startServer()` in both backends
- Implement basic `ServerOperations` (port, address, stop, free -- no maintenance yet)
- Wire up `ChromaServer` construction with cached port/address

**Tests:** Smoke tests for server start/stop/port/address/URL in both backends (mirrors existing embedded smoke tests).

**Why second:** Server lifecycle is the foundation for server-mode maintenance. No maintenance ops yet.

### Phase 3: Embedded Maintenance Operations

**Build:** Rebuild, compact, WAL prune on EmbeddedSession in both backends.

- Add 5 new embedded maintenance FFI symbols to both backends
- Add `chroma_embedded_persist_path` binding (needed for backup)
- Implement `EmbeddedOperations` methods for rebuild, compact (collection + all), prune (collection + all)
- Wire JSON request serialization and response deserialization

**Tests:** Integration tests for each maintenance operation on EmbeddedSession in both backends.

**Why third:** Embedded maintenance ops are simpler than server-mode (no stop-restart dance). Build and validate these first.

### Phase 4: Backup API (Embedded + Server)

**Build:** Backup for both embedded and server modes.

- Implement backup utilities in core (directory copy, manifest writing, SHA-256)
- Implement `EmbeddedOperations.backup()` -- close, copy, reopen
- Implement `ServerOperations.backup()` -- stop, copy via temp embedded, restart

**Tests:** Integration tests for backup in both modes, both backends. Verify manifest structure matches Go output format.

**Why fourth:** Backup is the most complex operation (filesystem I/O, manifest generation). Depends on both server lifecycle (Phase 2) and embedded operations (Phase 3).

### Phase 5: Server-Mode Maintenance Operations

**Build:** Rebuild, compact, WAL prune on ChromaServer in both backends.

- Implement `ServerOperations.rebuildCollection()` -- stop, temp embedded, rebuild, restart
- Implement `ServerOperations.compactCollection()` / `compactAll()` -- stop, temp embedded, compact, restart
- Implement `ServerOperations.pruneCollectionWAL()` / `pruneAllWAL()` -- stop, temp embedded, prune, restart

**Tests:** Integration tests for each server-mode maintenance operation in both backends.

**Why fifth:** Depends on Phase 2 (server lifecycle) and Phase 3 (embedded maintenance) being solid. The server-mode pattern (stop -> embedded -> restart) composes both.

---

## Scalability Considerations

| Concern | Current (scaffold) | After v0.5.0 | Future |
|---------|-------------------|-------------|--------|
| FFI symbol count | 5 per backend | 18 per backend | Grows with Rust shim features |
| Core interfaces | 1 (ChromaRuntime) | 4 (ChromaRuntime + ChromaServer + EmbeddedOperations + ServerOperations) | Stable; new features add methods |
| Core types | 3 classes | ~20 classes/records | Grows with feature surface |
| Test matrix | 2 backends x 1 smoke test | 2 backends x ~15 integration tests | Proportional to API surface |
| Build time | <5s | <15s (more compilation units) | Gradle incremental mitigates |

---

## Sources

- Go codebase analysis: `internal/runtime/chroma.go`, `config.go`, `embedded.go`, `backup.go`, `rebuild.go`, `compaction.go`, `wal_prune.go` -- HIGH confidence (direct source inspection)
- Java scaffold analysis: `java/core/`, `java/jna/`, `java/panama/` -- HIGH confidence (direct source inspection)
- Rust FFI surface: `shim/src/lib.rs` (50+ `extern "C"` functions) -- HIGH confidence (direct source inspection)
- Go facade files: `chroma.go`, `config.go`, `embedded.go`, `backup.go`, `rebuild.go`, `compaction.go`, `wal_prune.go` at repo root -- HIGH confidence
- [Project Panama FFM API in Production](https://www.javacodegeeks.com/2026/03/project-panamas-ffm-api-in-production-replacing-jni-without-writing-c-wrappers.html) -- MEDIUM confidence
- [Guide to Java Project Panama | Baeldung](https://www.baeldung.com/java-project-panama) -- MEDIUM confidence
- [Java Builder Pattern | Baeldung](https://www.baeldung.com/java-builder-pattern) -- HIGH confidence (standard pattern)
