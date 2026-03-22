# Phase 6: Core Foundation Types - Research

**Researched:** 2026-03-22
**Domain:** Java library design -- shared types, builders, FFI safety abstractions (no FFI dependencies in core)
**Confidence:** HIGH

## Summary

Phase 6 establishes the contract layer in the `java/core` module: config builders, result POJOs, FFI safety infrastructure, and session types. All downstream phases (7-10) implement against these stable contracts. The core module must have zero JNA or Panama imports.

The Go reference implementation provides an exact specification for every type. Config builders produce YAML matching Go's `toYAML()` output. Result POJOs mirror Go's JSON-deserialized structs (which in turn match the Rust shim's `Serialize` structs). The FFI lock pattern mirrors Go's `ffiMu sync.Mutex` with `callFFIHandle`/`callFFIPointer` as the template. String ownership (borrowed vs owned) is a binary property per FFI symbol -- the Rust shim documents which pointers must be freed.

Two new dependencies are needed in core: Gson (JSON deserialization of FFI responses) and SnakeYAML (YAML generation for config builders). Both are stable, widely used, and have no transitive FFI dependencies.

**Primary recommendation:** Implement types bottom-up: result POJOs first (pure data, testable immediately), then config builders (YAML output, golden-testable), then FFI safety abstractions (abstract class with template method), then session types (ServerSession following EmbeddedSession's proven pattern).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Optional numeric fields (Go's `*uint64`) use boxed `Long` in Java -- `null` means absent
- **D-02:** JSON deserialization via Gson dependency in core module (`com.google.code.gson:gson`)
- **D-03:** Result POJOs are final-field classes (not records) with package-private constructors for Gson
- **D-04:** Getter style is accessor-based (`collectionId()`, `recordsScanned()`) -- not JavaBean `getX()`
- **D-05:** YAML output via SnakeYAML dependency (`org.yaml.snakeyaml:snakeyaml`) -- proper YAML serialization
- **D-06:** Strict validation at `build()` time -- port range, non-null paths, address format, mutually exclusive options
- **D-07:** `rawYaml(String)` escape hatch on both builders -- overrides all other fields when set (mirrors Go's `WithRawYAML`)
- **D-08:** `ServerConfigBuilder` and `EmbeddedConfigBuilder` are fully independent -- no shared base class, duplicated fields are minimal (persistPath, sqliteFilename, allowReset)
- **D-09:** `AbstractChromaRuntime` abstract class in core -- holds global static `ReentrantLock` mirroring Go's package-level `ffiMu sync.Mutex`
- **D-10:** Lock is global (static) -- Rust `LAST_ERROR` is a `static Mutex<Option<String>>` (per-process, not thread-local), so all FFI calls must serialize
- **D-11:** String ownership via abstract methods on `AbstractChromaRuntime`: `readBorrowedString(long)` (don't free) and `readOwnedString(long)` (free after read) -- backends implement with JNA Pointer / Panama MemorySegment
- **D-12:** Integrated `callFfi()` template method: acquires lock -> calls FFI -> checks null return -> reads `lastError()` -> releases lock. Backends supply FFI call as lambda
- **D-13:** `EmbeddedSession` and `ServerSession` are independent types -- no shared interface for maintenance operations. Matches Go where `Embedded` and `Server` are separate structs
- **D-14:** `ChromaRuntime` interface adds `startServer(String configYaml)` returning `ServerSession`, symmetric with existing `startEmbedded(String configYaml)`
- **D-15:** `ServerSession` is a concrete final class in core (same pattern as `EmbeddedSession`) -- wraps `long` handle with callback slots injected by backends
- **D-16:** `ServerSession` fully defined in Phase 6 with all callback slots (lifecycle, accessors, maintenance) -- Phases 7-10 just wire up backends
- **D-17:** Every option type gets a nested `Builder` with fluent API and strict validation at `build()` -- consistent with config builders (RebuildOptions, WALPruneOptions, BackupOptions, CompactCollectionRequest)
- **D-18:** Option types produce JSON via `toJson()` using Gson -- backends pass JSON string directly to FFI calls
- **D-19:** No-options overloads on session methods use internal defaults (e.g., `session.rebuildCollection("coll")` uses `RebuildOptions.defaults()`)
- **D-20:** `readLastError()` is an abstract method on `AbstractChromaRuntime`, integrated into `callFfi()` template -- backends implement the FFI pointer read
- **D-21:** Three-tier exception rule: `IllegalArgumentException` for bad inputs, `IllegalStateException` for lifecycle violations (closed session/runtime), `ChromaException` for all FFI/native failures
- **D-22:** Golden YAML tests use inline expected strings in test methods -- no external fixture files
- **D-23:** Result POJO tests use hand-crafted JSON strings covering required, optional/null, list, and nested fields

### Claude's Discretion
- Exact Gson configuration (custom TypeAdapter vs annotation-based)
- Internal structure of `callFfi()` overloads (void returns, string returns, JSON returns)
- Test class organization within core module
- SnakeYAML Dumper options for consistent output formatting

### Deferred Ideas (OUT OF SCOPE)
- Per-handle error isolation for concurrent multi-instance support -- future phase
- `ChromaErrorCode` enum on `ChromaException` for programmatic error handling -- tracked as FUTURE-01
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FOUND-01 | Core module contains all shared interfaces, builders, and result types with no FFI dependencies | Gson + SnakeYAML are pure Java; `AbstractChromaRuntime` uses abstract methods, not FFI imports; `gradle :core:build` must pass with zero JNA/Panama imports |
| FOUND-02 | `ServerConfigBuilder` produces valid YAML for server startup with fluent API | Go's `ServerConfig.toYAML()` provides exact field names, defaults, and output format; SnakeYAML `DumperOptions` with BLOCK flow style produces matching output |
| FOUND-03 | `EmbeddedConfigBuilder` produces valid YAML for embedded startup with fluent API | Go's `EmbeddedConfig.toYAML()` provides exact field names and defaults (3 fields: persist_path, sqlite_filename, allow_reset) |
| FOUND-04 | Result POJOs defined for all maintenance operations | Go structs + Rust Serialize structs provide authoritative field lists; Gson with `LOWER_CASE_WITH_UNDERSCORES` policy handles snake_case mapping |
| FOUND-05 | FFI serialization lock pattern established | Go's `ffiMu sync.Mutex` + `callFFIHandle`/`callFFIPointer` provide the template; Java `ReentrantLock` is the direct equivalent |
| FOUND-06 | String ownership helpers distinguish owned vs borrowed native pointers | Rust shim documents ownership per symbol; `chroma_version` and handle accessors are borrowed; `chroma_get_last_error` and response strings are owned |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| com.google.code.gson:gson | 2.13.2 | JSON deserialization of FFI response strings | Most widely used Java JSON library; annotation-based snake_case mapping via `FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES`; handles final fields via UnsafeAllocator |
| org.yaml:snakeyaml | 2.6 | YAML generation for config builders | De facto standard Java YAML library; `DumperOptions` controls output formatting; no transitive dependencies |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| org.junit.jupiter:junit-jupiter | 5.11.4 | Unit testing | Already in use; golden YAML tests and POJO tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Gson | Jackson | Jackson is heavier (3+ JARs), more features not needed here; Gson is single JAR |
| SnakeYAML | Manual string builder | Go uses manual `fmt.Fprintf` but Java builder has strict validation + proper YAML escaping needs |

**Installation (in java/core/build.gradle.kts):**
```kotlin
dependencies {
    implementation("com.google.code.gson:gson:2.13.2")
    implementation("org.yaml:snakeyaml:2.6")
}
```

**Version verification:** Gson 2.13.2 (Sep 2025 release). SnakeYAML 2.6 (Feb 2026 release). Both verified via Maven Central search results.

## Architecture Patterns

### Recommended Project Structure
```
java/core/src/main/java/tech/amikos/chroma/local/core/
├── ChromaRuntime.java          # interface (extend with startServer)
├── AbstractChromaRuntime.java  # NEW: abstract class with FFI lock + template
├── ChromaException.java        # existing exception type
├── EmbeddedSession.java        # existing session (add maintenance method signatures)
├── ServerSession.java          # NEW: server session with callback slots
├── ServerConfigBuilder.java    # NEW: fluent builder -> YAML
├── EmbeddedConfigBuilder.java  # NEW: fluent builder -> YAML
├── BackupManifest.java         # NEW: result POJO
├── BackupFileMetadata.java     # NEW: nested result POJO
├── BackupOptions.java          # NEW: option type with nested Builder
├── RebuildCollectionResult.java    # NEW: result POJO
├── RebuildOptions.java             # NEW: option type with nested Builder
├── CompactionResult.java           # NEW: result POJO
├── CompactionCollectionResult.java # NEW: nested result POJO
├── CompactCollectionRequest.java   # NEW: request with nested Builder
├── CompactAllRequest.java          # NEW: request with nested Builder
├── WALPruneResult.java             # NEW: result POJO
├── WALPruneCollectionResult.java   # NEW: nested result POJO
└── WALPruneOptions.java            # NEW: option type with nested Builder
```

### Pattern 1: Result POJO with Gson Deserialization
**What:** Final-field classes with package-private constructors, deserialized via Gson's UnsafeAllocator
**When to use:** All FFI response types
**Example:**
```java
// Gson with LOWER_CASE_WITH_UNDERSCORES handles snake_case automatically
public final class RebuildCollectionResult {
    private final String collectionId;
    private final String name;
    private final String tenantId;
    private final String databaseName;
    private final boolean precheck;
    private final boolean wouldRebuild;
    private final boolean rebuilt;
    private final long recordsScanned;
    private final long vectorsReindexed;
    private final long durationMs;
    private final String backupPath;
    private final List<String> warnings;

    // Package-private constructor for Gson
    RebuildCollectionResult() {
        this.collectionId = null;
        this.name = null;
        this.tenantId = null;
        this.databaseName = null;
        this.precheck = false;
        this.wouldRebuild = false;
        this.rebuilt = false;
        this.recordsScanned = 0;
        this.vectorsReindexed = 0;
        this.durationMs = 0;
        this.backupPath = null;
        this.warnings = List.of();
    }

    public String collectionId() { return collectionId; }
    public String name() { return name; }
    // ... accessor methods following D-04
}
```

**Key Gson configuration:**
```java
private static final Gson GSON = new GsonBuilder()
    .setFieldNamingPolicy(FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES)
    .create();
```
This automatically maps `collectionId` <-> `collection_id`, `recordsScanned` <-> `records_scanned`, etc. No `@SerializedName` annotations needed on individual fields.

### Pattern 2: Config Builder with YAML Output
**What:** Fluent builder that validates at `build()` and produces YAML via SnakeYAML
**When to use:** `ServerConfigBuilder`, `EmbeddedConfigBuilder`
**Example:**
```java
public final class ServerConfigBuilder {
    private int port = 8000;
    private String listenAddress = "127.0.0.1";
    private int maxPayloadSizeBytes = 40 * 1024 * 1024;
    private String persistPath = "./chroma";
    private String sqliteFilename = "chroma.sqlite3";
    private boolean allowReset = false;
    private List<String> corsAllowOrigins;
    private String otelEndpoint;
    private String otelServiceName;
    private String rawYaml;

    public ServerConfigBuilder port(int port) { this.port = port; return this; }
    // ... other setters

    public String build() {
        if (rawYaml != null) return rawYaml;
        validate();
        return toYaml();
    }

    private void validate() {
        if (port < 1 || port > 65535)
            throw new IllegalArgumentException("port must be 1-65535");
        if (persistPath == null || persistPath.isBlank())
            throw new IllegalArgumentException("persistPath must be set");
        // ... other validations per D-06
    }

    private String toYaml() {
        // Use SnakeYAML with DumperOptions for consistent formatting
        DumperOptions options = new DumperOptions();
        options.setDefaultFlowStyle(DumperOptions.FlowStyle.BLOCK);
        // Build a LinkedHashMap to control field order matching Go output
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("port", port);
        map.put("listen_address", listenAddress);
        map.put("max_payload_size_bytes", maxPayloadSizeBytes);
        map.put("persist_path", persistPath);
        map.put("sqlite_filename", sqliteFilename);
        map.put("allow_reset", allowReset);
        // conditional fields...
        Yaml yaml = new Yaml(options);
        return yaml.dump(map);
    }
}
```

### Pattern 3: AbstractChromaRuntime with Template Method
**What:** Abstract class holding static `ReentrantLock`, providing `callFfi()` template that acquires lock, calls FFI lambda, checks null, reads error, releases lock
**When to use:** Base class for JNA and Panama runtime implementations
**Example:**
```java
public abstract class AbstractChromaRuntime implements ChromaRuntime {
    private static final ReentrantLock FFI_LOCK = new ReentrantLock();

    // Backends implement these
    protected abstract String readBorrowedString(long address);
    protected abstract String readOwnedString(long address);
    protected abstract String readLastError();

    // Template method for FFI calls returning a handle (long)
    protected long callFfiHandle(LongSupplier ffiCall) {
        FFI_LOCK.lock();
        try {
            long result = ffiCall.getAsLong();
            if (result == 0L) {
                throw new ChromaException(readLastError());
            }
            return result;
        } finally {
            FFI_LOCK.unlock();
        }
    }

    // Template method for FFI calls returning owned JSON string
    protected <T> T callFfiJson(LongSupplier ffiCall, Class<T> type) {
        FFI_LOCK.lock();
        try {
            long ptr = ffiCall.getAsLong();
            if (ptr == 0L) {
                throw new ChromaException(readLastError());
            }
            String json = readOwnedString(ptr);
            return GSON.fromJson(json, type);
        } finally {
            FFI_LOCK.unlock();
        }
    }
}
```

### Pattern 4: ServerSession with Callback Slots
**What:** Concrete final class mirroring `EmbeddedSession` -- wraps `long` handle, callbacks injected by backends at construction
**When to use:** Server lifecycle management
**Example:**
```java
public final class ServerSession implements AutoCloseable {
    private final long handle;
    private final AtomicBoolean closed;

    // Lifecycle callbacks
    private final LongConsumer stopAction;
    private final LongConsumer freeAction;

    // Accessor callbacks
    private final LongToIntFunction portAccessor;
    private final LongFunction<String> addressAccessor;
    private final LongFunction<String> persistPathAccessor;

    // Maintenance callbacks (wired in later phases)
    // ...

    public int port() { ensureOpen(); return portAccessor.apply(handle); }
    public String address() { ensureOpen(); return addressAccessor.apply(handle); }
    public String url() { return "http://" + address() + ":" + port(); }

    @Override
    public void close() {
        if (closed.compareAndSet(false, true)) {
            try {
                stopAction.accept(handle);
            } finally {
                freeAction.accept(handle);
            }
        }
    }
}
```

### Anti-Patterns to Avoid
- **Shared base class for config builders:** D-08 explicitly forbids this. Duplicating 3 fields (persistPath, sqliteFilename, allowReset) is acceptable and simpler than inheritance.
- **Records for result POJOs:** Java records require all-args constructors. Gson's UnsafeAllocator works better with classes that have a no-arg (or package-private) constructor. D-03 locks this decision.
- **Per-field @SerializedName annotations:** Use `FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES` globally instead. Only annotate fields where the naming convention fails to produce the correct key.
- **Synchronized methods instead of ReentrantLock:** D-09/D-10 specify `ReentrantLock`. It provides explicit lock/unlock semantics matching Go's `ffiMu.Lock()`/`defer ffiMu.Unlock()` pattern and enables future tryLock/timeout extensions.
- **FFI imports in core:** The entire point of this phase is that core has zero JNA/Panama dependencies. Abstract methods and functional interfaces are the contracts.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON deserialization | Custom parser for FFI response strings | Gson with `LOWER_CASE_WITH_UNDERSCORES` | Handles nulls, optional fields, nested objects, lists automatically |
| YAML generation | `String.format()` / `StringBuilder` | SnakeYAML with `DumperOptions` | Handles quoting, escaping, list formatting, nested maps correctly; Go uses manual formatting but Java has more edge cases with string escaping |
| Thread-safe FFI serialization | `synchronized` blocks | `ReentrantLock` (static, in abstract base) | Matches Go's explicit Mutex pattern; future-proof for tryLock/timeout |
| String ownership tracking | Per-call free/don't-free logic | `readBorrowedString()` / `readOwnedString()` abstract methods | Encapsulates JNA `Pointer.getString()` vs Panama `MemorySegment.getString()` + free logic |

**Key insight:** The core module is entirely about defining contracts. Every piece of actual FFI interaction is pushed to abstract methods that backends implement. This ensures core compiles and tests without any native library present.

## Common Pitfalls

### Pitfall 1: YAML Output Mismatch with Go
**What goes wrong:** Java's SnakeYAML produces YAML that is semantically equivalent but textually different from Go's `fmt.Fprintf` output. Golden tests fail.
**Why it happens:** Go uses `%q` (Go-style double-quoted strings). SnakeYAML defaults to unquoted or single-quoted scalars.
**How to avoid:** Use `DumperOptions` to control quoting. For golden tests, compare YAML semantics (parse both sides and compare maps) rather than raw string equality. Alternatively, configure SnakeYAML's `ScalarStyle` or use a `Representer` that quotes string values. The golden tests should verify that the YAML produced by the Java builder, when parsed back, yields the same key-value pairs as Go's output.
**Warning signs:** Tests pass locally but break when SnakeYAML version changes default quoting behavior.

### Pitfall 2: Gson Final Field Initialization
**What goes wrong:** Gson uses UnsafeAllocator to bypass constructors when deserializing into final-field classes. If `disableJdkUnsafe()` is called, deserialization fails.
**Why it happens:** Final fields require special reflection or Unsafe to set after construction.
**How to avoid:** Do NOT call `GsonBuilder.disableJdkUnsafe()`. The package-private no-arg constructor (D-03) provides a fallback, but Gson's default behavior (using Unsafe) is what makes final fields work. Test deserialization early and verify all fields populate correctly.
**Warning signs:** Fields are null or zero after deserialization even though JSON contains values.

### Pitfall 3: Optional Long Fields (null vs 0)
**What goes wrong:** Go's `*uint64` is null when absent. Java's `long` is 0 when absent. Boxed `Long` is needed (D-01), but Gson maps `0` and absent differently.
**Why it happens:** Rust shim uses `#[serde(skip_serializing_if = "Option::is_none")]` -- absent fields simply don't appear in JSON. Gson deserializes missing keys as `null` for boxed `Long` (correct) but as `0` for primitive `long` (wrong for absent semantics).
**How to avoid:** Use boxed `Long` for all optional numeric fields. Verify with JSON that omits the field entirely, not just sets it to `null`.
**Warning signs:** Code treats `0L` as "zero" instead of "absent" in WALPruneCollectionResult's sequence fields.

### Pitfall 4: Lock Scope in callFfi Template
**What goes wrong:** Lock held too long (blocks all FFI calls) or not long enough (error slot read races with another thread's FFI call).
**Why it happens:** Go's pattern is: lock -> call FFI -> if null, read error -> unlock. The error read MUST happen inside the lock because `LAST_ERROR` is global.
**How to avoid:** The `callFfi()` template method must read the error string AND free the owned pointer all within the lock scope. Return the Java String (which is a copy), then release the lock.
**Warning signs:** Intermittent wrong error messages under concurrent access.

### Pitfall 5: ServerSession Close Ordering
**What goes wrong:** Server session calls `stop` then `free`, but if `stop` fails, `free` is still needed to avoid handle leaks.
**Why it happens:** Go's `Server.Close()` calls `chromaServerStop` then `chromaServerFree` unconditionally. If stop returns "already stopped", the handle still needs freeing.
**How to avoid:** Use try-finally in `close()`: stop in try, free in finally. Mirror Go's pattern where `ErrServerAlreadyStop` is swallowed.
**Warning signs:** Native memory leaks when server stop fails.

## Code Examples

### Go ServerConfig Default YAML Output (Golden Reference)
```
port: 8000
listen_address: "127.0.0.1"
max_payload_size_bytes: 41943040
persist_path: "./chroma"
sqlite_filename: "chroma.sqlite3"
allow_reset: false
```
Source: `internal/runtime/config.go` lines 33-41, 109-139

### Go EmbeddedConfig Default YAML Output (Golden Reference)
```
persist_path: "./chroma"
sqlite_filename: "chroma.sqlite3"
allow_reset: false
```
Source: `internal/runtime/embedded.go` lines 53-59, 89-99

### Go ServerConfig YAML with Optional Fields
```
port: 9090
listen_address: "0.0.0.0"
max_payload_size_bytes: 41943040
persist_path: "/data/chroma"
sqlite_filename: "chroma.sqlite3"
allow_reset: true
cors_allow_origins:
  - "http://localhost:3000"
  - "https://example.com"
open_telemetry:
  endpoint: "http://otel:4317"
  service_name: "chroma-dev"
```
Source: `internal/runtime/config.go` lines 109-139

### Rust Shim String Ownership Reference
```
BORROWED (don't free):
  chroma_version()             -> static string, process lifetime
  chroma_server_address()      -> handle-owned CString, handle lifetime
  chroma_server_persist_path() -> handle-owned CString, handle lifetime
  chroma_embedded_persist_path() -> handle-owned CString, handle lifetime

OWNED (must free with chroma_string_free):
  chroma_get_last_error()                -> CString::into_raw()
  chroma_embedded_rebuild_collection()   -> CString::into_raw()
  chroma_embedded_compact_collection()   -> CString::into_raw()
  chroma_embedded_compact_all()          -> CString::into_raw()
  chroma_embedded_prune_wal_collection() -> CString::into_raw()
  chroma_embedded_prune_wal_all()        -> CString::into_raw()
  (all chroma_embedded_* returning *mut c_char) -> CString::into_raw()
```
Source: `shim/src/lib.rs` lines 4897-4935 and function signatures throughout

### Go FFI Lock Pattern (Template Reference)
```go
func callFFIHandle(call func() uintptr) (uintptr, error) {
    ffiMu.Lock()
    defer ffiMu.Unlock()
    handle := call()
    if handle == 0 {
        return 0, nullPointerError(getLastErrorUnlocked())
    }
    return handle, nil
}
```
Source: `internal/runtime/chroma.go` lines 166-175

### Result POJO Field Mapping (Go -> Java)

**RebuildCollectionResult:**
| Go Field | Go Type | JSON Key | Java Field | Java Type |
|----------|---------|----------|------------|-----------|
| CollectionID | string | collection_id | collectionId | String |
| Name | string | name | name | String |
| TenantID | string | tenant_id | tenantId | String |
| DatabaseName | string | database_name | databaseName | String |
| Precheck | bool | precheck | precheck | boolean |
| WouldRebuild | bool | would_rebuild | wouldRebuild | boolean |
| Rebuilt | bool | rebuilt | rebuilt | boolean |
| RecordsScanned | uint64 | records_scanned | recordsScanned | long |
| VectorsReindexed | uint64 | vectors_reindexed | vectorsReindexed | long |
| DurationMS | uint64 | duration_ms | durationMs | long |
| BackupPath | string | backup_path | backupPath | String |
| Warnings | []string | warnings | warnings | List\<String\> |

**CompactionCollectionResult:**
| Go Field | Go Type | JSON Key | Java Field | Java Type |
|----------|---------|----------|------------|-----------|
| CollectionID | string | collection_id | collectionId | String |
| Name | string | name | name | String |
| TenantID | string | tenant_id | tenantId | String |
| DatabaseName | string | database_name | databaseName | String |
| PendingOpsBefore | *uint64 | pending_ops_before | pendingOpsBefore | Long |
| PendingOpsAfter | *uint64 | pending_ops_after | pendingOpsAfter | Long |
| PendingOpsBeforeError | string | pending_ops_before_error | pendingOpsBeforeError | String |
| PendingOpsAfterError | string | pending_ops_after_error | pendingOpsAfterError | String |
| Error | string | error | error | String |

**CompactionResult:**
| Go Field | Go Type | JSON Key | Java Field | Java Type |
|----------|---------|----------|------------|-----------|
| CollectionCount | uint32 | collection_count | collectionCount | int |
| DurationMS | uint64 | duration_ms | durationMs | long |
| PendingOpsBeforeTotal | uint64 | pending_ops_before_total | pendingOpsBeforeTotal | long |
| PendingOpsAfterTotal | uint64 | pending_ops_after_total | pendingOpsAfterTotal | long |
| Collections | []CompactionCollectionResult | collections | collections | List\<CompactionCollectionResult\> |

**WALPruneCollectionResult:**
| Go Field | Go Type | JSON Key | Java Field | Java Type |
|----------|---------|----------|------------|-----------|
| CollectionID | string | collection_id | collectionId | String |
| Name | string | name | name | String |
| TenantID | string | tenant_id | tenantId | String |
| DatabaseName | string | database_name | databaseName | String |
| SafeSeqCutoff | *uint64 | safe_seq_cutoff | safeSeqCutoff | Long |
| CandidateSeqMin | *uint64 | candidate_seq_min | candidateSeqMin | Long |
| CandidateSeqMax | *uint64 | candidate_seq_max | candidateSeqMax | Long |
| PrunedSeqMin | *uint64 | pruned_seq_min | prunedSeqMin | Long |
| PrunedSeqMax | *uint64 | pruned_seq_max | prunedSeqMax | Long |
| CandidateCount | uint64 | candidate_count | candidateCount | long |
| CandidateBytes | uint64 | candidate_bytes | candidateBytes | long |
| PrunedCount | uint64 | pruned_count | prunedCount | long |
| PrunedBytes | uint64 | pruned_bytes | prunedBytes | long |
| Error | string | error | error | String |

**WALPruneResult:**
| Go Field | Go Type | JSON Key | Java Field | Java Type |
|----------|---------|----------|------------|-----------|
| CollectionCount | uint32 | collection_count | collectionCount | int |
| DurationMS | uint64 | duration_ms | durationMs | long |
| DryRun | bool | dry_run | dryRun | boolean |
| VacuumRequested | bool | vacuum_requested | vacuumRequested | boolean |
| VacuumExecuted | bool | vacuum_executed | vacuumExecuted | boolean |
| Warning | string | warning | warning | String |
| CandidateCountTotal | uint64 | candidate_count_total | candidateCountTotal | long |
| CandidateBytesTotal | uint64 | candidate_bytes_total | candidateBytesTotal | long |
| PrunedCountTotal | uint64 | pruned_count_total | prunedCountTotal | long |
| PrunedBytesTotal | uint64 | pruned_bytes_total | prunedBytesTotal | long |
| Collections | []WALPruneCollectionResult | collections | collections | List\<WALPruneCollectionResult\> |

**BackupManifest** (note: this is generated by Go/Rust, not deserialized from FFI in this phase, but defined for future use):
| Go Field | Go Type | JSON Key | Java Field | Java Type |
|----------|---------|----------|------------|-----------|
| SchemaVersion | string | schema_version | schemaVersion | String |
| Mode | BackupMode(string) | mode | mode | String |
| CreatedAt | time.Time | created_at | createdAt | String |
| WrapperVersion | string | wrapper_version | wrapperVersion | String |
| SourcePaths | []string | source_paths | sourcePaths | List\<String\> |
| DestinationPath | string | destination_path | destinationPath | String |
| SnapshotPath | string | snapshot_path | snapshotPath | String |
| ManifestPath | string | manifest_path | manifestPath | String |
| IncludeMetadata | bool | include_metadata | includeMetadata | boolean |
| FileCount | int | file_count | fileCount | int |
| TotalBytes | int64 | total_bytes | totalBytes | long |
| Files | []BackupFileMetadata | files | files | List\<BackupFileMetadata\> |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| SnakeYAML 1.x (unsafe by default) | SnakeYAML 2.x (SafeConstructor default) | Feb 2023 | No security concern for our use case (we only dump, never load untrusted YAML) |
| Gson manual @SerializedName per field | Gson FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES | Available since Gson 2.x | Eliminates per-field annotation boilerplate |
| Java records for DTOs | Final-field classes with package-private constructors | Project decision D-03 | Records require all-args constructors incompatible with Gson's deserialization model |

## Open Questions

1. **YAML quoting style for golden tests**
   - What we know: Go uses `%q` which produces Go-style double-quoted strings (e.g., `"127.0.0.1"`). SnakeYAML by default may not quote simple strings.
   - What's unclear: Whether golden tests should compare exact string output or parse-and-compare semantics.
   - Recommendation: Configure SnakeYAML `Representer` to use `ScalarStyle.DOUBLE_QUOTED` for string values to match Go output. If that proves brittle, fall back to parse-and-compare. The YAML must be valid and semantically correct for the Rust shim to parse it -- exact string format is secondary to correctness.

2. **Gson shared instance location**
   - What we know: A single `Gson` instance with `LOWER_CASE_WITH_UNDERSCORES` policy is needed for all result POJO deserialization and option type `toJson()` serialization.
   - What's unclear: Whether it lives as a package-private constant in a utility class or as a static field on `AbstractChromaRuntime`.
   - Recommendation: Package-private utility class (e.g., `JsonUtil`) in core that exposes the shared `Gson` instance. Keeps it accessible to both result POJOs and option builders without polluting the public API.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | JUnit Jupiter 5.11.4 |
| Config file | `java/build.gradle.kts` (JUnit platform configured in `subprojects` block) |
| Quick run command | `cd java && gradle --no-daemon :core:test` |
| Full suite command | `cd java && gradle --no-daemon :core:test :jna:test :panama:test` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FOUND-01 | Core module compiles with no FFI deps | build | `cd java && gradle --no-daemon :core:build` | N/A (build check) |
| FOUND-02 | ServerConfigBuilder default YAML matches Go | unit | `cd java && gradle --no-daemon :core:test --tests '*ServerConfigBuilderTest*'` | Wave 0 |
| FOUND-02 | ServerConfigBuilder with all options set | unit | `cd java && gradle --no-daemon :core:test --tests '*ServerConfigBuilderTest*'` | Wave 0 |
| FOUND-02 | ServerConfigBuilder validation rejects invalid input | unit | `cd java && gradle --no-daemon :core:test --tests '*ServerConfigBuilderTest*'` | Wave 0 |
| FOUND-03 | EmbeddedConfigBuilder default YAML matches Go | unit | `cd java && gradle --no-daemon :core:test --tests '*EmbeddedConfigBuilderTest*'` | Wave 0 |
| FOUND-04 | RebuildCollectionResult deserializes from JSON | unit | `cd java && gradle --no-daemon :core:test --tests '*RebuildCollectionResultTest*'` | Wave 0 |
| FOUND-04 | CompactionResult deserializes from JSON | unit | `cd java && gradle --no-daemon :core:test --tests '*CompactionResultTest*'` | Wave 0 |
| FOUND-04 | WALPruneResult deserializes from JSON | unit | `cd java && gradle --no-daemon :core:test --tests '*WALPruneResultTest*'` | Wave 0 |
| FOUND-04 | BackupManifest deserializes from JSON | unit | `cd java && gradle --no-daemon :core:test --tests '*BackupManifestTest*'` | Wave 0 |
| FOUND-05 | AbstractChromaRuntime lock pattern (mock test) | unit | `cd java && gradle --no-daemon :core:test --tests '*AbstractChromaRuntimeTest*'` | Wave 0 |
| FOUND-06 | String ownership abstract methods defined | unit | `cd java && gradle --no-daemon :core:test --tests '*AbstractChromaRuntimeTest*'` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd java && gradle --no-daemon :core:test`
- **Per wave merge:** `cd java && gradle --no-daemon :core:test :jna:test :panama:test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `java/core/src/test/java/tech/amikos/chroma/local/core/ServerConfigBuilderTest.java` -- covers FOUND-02
- [ ] `java/core/src/test/java/tech/amikos/chroma/local/core/EmbeddedConfigBuilderTest.java` -- covers FOUND-03
- [ ] `java/core/src/test/java/tech/amikos/chroma/local/core/RebuildCollectionResultTest.java` -- covers FOUND-04
- [ ] `java/core/src/test/java/tech/amikos/chroma/local/core/CompactionResultTest.java` -- covers FOUND-04
- [ ] `java/core/src/test/java/tech/amikos/chroma/local/core/WALPruneResultTest.java` -- covers FOUND-04
- [ ] `java/core/src/test/java/tech/amikos/chroma/local/core/BackupManifestTest.java` -- covers FOUND-04
- [ ] `java/core/src/test/java/tech/amikos/chroma/local/core/AbstractChromaRuntimeTest.java` -- covers FOUND-05, FOUND-06
- [ ] Gson + SnakeYAML dependencies: add to `java/core/build.gradle.kts`

## Sources

### Primary (HIGH confidence)
- `internal/runtime/config.go` -- Go ServerConfig/EmbeddedConfig with exact field names, defaults, YAML format
- `internal/runtime/embedded.go` -- Go EmbeddedConfig, StartEmbeddedConfig
- `internal/runtime/chroma.go` -- Go FFI lock pattern (`ffiMu`, `callFFIHandle`, `callFFIPointer`), string reading
- `internal/runtime/rebuild.go` -- RebuildCollectionResult fields, option types
- `internal/runtime/compaction.go` -- CompactionResult, CompactionCollectionResult fields
- `internal/runtime/wal_prune.go` -- WALPruneResult, WALPruneCollectionResult fields
- `internal/runtime/backup.go` -- BackupManifest, BackupFileMetadata fields, BackupOptions
- `shim/src/lib.rs` -- Rust `LAST_ERROR` static mutex (line 79), string ownership per symbol, response struct definitions
- `java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedSession.java` -- Existing session pattern
- `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` -- Current JNA backend
- `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` -- Current Panama backend

### Secondary (MEDIUM confidence)
- [Maven Central Gson](https://central.sonatype.com/artifact/com.google.code.gson/gson) -- Gson 2.13.2 version verified
- [Maven Central SnakeYAML](https://central.sonatype.com/artifact/org.yaml/snakeyaml) -- SnakeYAML 2.6 version verified
- [Gson User Guide](http://google.github.io/gson/UserGuide.html) -- FieldNamingPolicy, UnsafeAllocator behavior
- [SnakeYAML DumperOptions](https://javadoc.io/static/org.yaml/snakeyaml/2.1/org/yaml/snakeyaml/DumperOptions.html) -- YAML formatting controls

### Tertiary (LOW confidence)
- None. All findings verified against primary or secondary sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- Gson and SnakeYAML are locked decisions from CONTEXT.md, versions verified on Maven Central
- Architecture: HIGH -- All patterns derived from existing Go implementation and existing Java scaffold code
- Pitfalls: HIGH -- Based on direct code analysis of Go FFI patterns, Rust shim ownership rules, and Gson behavior with final fields

**Research date:** 2026-03-22
**Valid until:** 2026-04-22 (30 days -- stable domain, locked decisions)
