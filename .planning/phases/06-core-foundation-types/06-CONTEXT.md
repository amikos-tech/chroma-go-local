# Phase 6: Core Foundation Types - Context

**Gathered:** 2026-03-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Define all shared interfaces, builders, result POJOs, and FFI safety patterns in the `core` module with no FFI dependencies. This is the contract layer — backend modules (JNA, Panama) implement against these stable contracts. No new Rust shim exports.

</domain>

<decisions>
## Implementation Decisions

### Result type design
- **D-01:** Optional numeric fields (Go's `*uint64`) use boxed `Long` in Java — `null` means absent
- **D-02:** JSON deserialization via Gson dependency in core module (`com.google.code.gson:gson`)
- **D-03:** Result POJOs are final-field classes (not records) with package-private constructors for Gson
- **D-04:** Getter style is accessor-based (`collectionId()`, `recordsScanned()`) — not JavaBean `getX()`

### Config builder shape
- **D-05:** YAML output via SnakeYAML dependency (`org.yaml.snakeyaml:snakeyaml`) — proper YAML serialization
- **D-06:** Strict validation at `build()` time — port range, non-null paths, address format, mutually exclusive options
- **D-07:** `rawYaml(String)` escape hatch on both builders — overrides all other fields when set (mirrors Go's `WithRawYAML`)
- **D-08:** `ServerConfigBuilder` and `EmbeddedConfigBuilder` are fully independent — no shared base class, duplicated fields are minimal (persistPath, sqliteFilename, allowReset)

### FFI safety infrastructure
- **D-09:** `AbstractChromaRuntime` abstract class in core — holds global static `ReentrantLock` mirroring Go's package-level `ffiMu sync.Mutex`
- **D-10:** Lock is global (static) — Rust `LAST_ERROR` is a `static Mutex<Option<String>>` (per-process, not thread-local), so all FFI calls must serialize
- **D-11:** String ownership via abstract methods on `AbstractChromaRuntime`: `readBorrowedString(long)` (don't free) and `readOwnedString(long)` (free after read) — backends implement with JNA Pointer / Panama MemorySegment
- **D-12:** Integrated `callFfi()` template method: acquires lock → calls FFI → checks null return → reads `lastError()` → releases lock. Backends supply FFI call as lambda

### Session type hierarchy
- **D-13:** `EmbeddedSession` and `ServerSession` are independent types — no shared interface for maintenance operations. Matches Go where `Embedded` and `Server` are separate structs
- **D-14:** `ChromaRuntime` interface adds `startServer(String configYaml)` returning `ServerSession`, symmetric with existing `startEmbedded(String configYaml)`
- **D-15:** `ServerSession` is a concrete final class in core (same pattern as `EmbeddedSession`) — wraps `long` handle with callback slots injected by backends
- **D-16:** `ServerSession` fully defined in Phase 6 with all callback slots (lifecycle, accessors, maintenance) — Phases 7-10 just wire up backends

### Option builder pattern
- **D-17:** Every option type gets a nested `Builder` with fluent API and strict validation at `build()` — consistent with config builders (RebuildOptions, WALPruneOptions, BackupOptions, CompactCollectionRequest)
- **D-18:** Option types produce JSON via `toJson()` using Gson — backends pass JSON string directly to FFI calls
- **D-19:** No-options overloads on session methods use internal defaults (e.g., `session.rebuildCollection("coll")` uses `RebuildOptions.defaults()`)

### Error handling contract
- **D-20:** `readLastError()` is an abstract method on `AbstractChromaRuntime`, integrated into `callFfi()` template — backends implement the FFI pointer read
- **D-21:** Three-tier exception rule: `IllegalArgumentException` for bad inputs, `IllegalStateException` for lifecycle violations (closed session/runtime), `ChromaException` for all FFI/native failures

### Test strategy
- **D-22:** Golden YAML tests use inline expected strings in test methods — no external fixture files
- **D-23:** Result POJO tests use hand-crafted JSON strings covering required, optional/null, list, and nested fields

### Claude's Discretion
- Exact Gson configuration (custom TypeAdapter vs annotation-based)
- Internal structure of `callFfi()` overloads (void returns, string returns, JSON returns)
- Test class organization within core module
- SnakeYAML Dumper options for consistent output formatting

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Go API surface (reference implementation)
- `internal/runtime/config.go` — ServerConfig and EmbeddedConfig with YAML generation; field names and defaults must match
- `internal/runtime/chroma.go` — FFI function signatures, global `ffiMu` lock pattern, library loading via `sync.Once`
- `internal/runtime/embedded.go` — EmbeddedConfig, StartEmbeddedConfig, Embedded struct pattern
- `internal/runtime/backup.go` — BackupManifest, BackupFileMetadata, BackupOptions, BackupMode types
- `internal/runtime/rebuild.go` — RebuildCollectionResult, rebuildCollectionRequest, option types
- `internal/runtime/compaction.go` — CompactionResult, CompactionCollectionResult, CompactCollectionRequest, CompactAllRequest
- `internal/runtime/wal_prune.go` — WALPruneResult, WALPruneCollectionResult, option types and policies

### Rust FFI shim (symbol reference)
- `shim/src/lib.rs` — All `chroma_*` FFI symbols, `LAST_ERROR` static mutex (line 79), string ownership conventions

### Existing Java scaffold
- `java/core/src/main/java/tech/amikos/chroma/local/core/ChromaRuntime.java` — Current interface to extend
- `java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedSession.java` — Concrete session pattern to follow
- `java/core/src/main/java/tech/amikos/chroma/local/core/ChromaException.java` — Existing exception type
- `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` — JNA backend (must retrofit to extend AbstractChromaRuntime)
- `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` — Panama backend (must retrofit to extend AbstractChromaRuntime)

### Build configuration
- `java/core/build.gradle.kts` — Core module build; must add Gson + SnakeYAML dependencies
- `.planning/REQUIREMENTS.md` — FOUND-01 through FOUND-06 requirements for this phase

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `EmbeddedSession`: Proven pattern for handle-wrapping with `AtomicBoolean` close guard and `LongConsumer` callback — reuse for `ServerSession`
- `ChromaException`: Already established unchecked exception — extend usage, no changes needed
- `ChromaRuntime` interface: Extend with `startServer()` method

### Established Patterns
- Handle-based lifecycle: opaque `long` handle + close action callback injected by backend
- `AtomicBoolean` for idempotent close with `compareAndSet`
- `ensureOpen()` guard method throwing `IllegalStateException`
- Backend init via static `init(String libraryPath)` factory

### Integration Points
- `AbstractChromaRuntime` will sit between `ChromaRuntime` interface and existing JNA/Panama implementations — requires retrofitting both backends to extend it
- Config builders produce YAML consumed by existing `startEmbedded(String configYaml)` and new `startServer(String configYaml)`
- Result POJOs will be returned by maintenance methods added to sessions in Phases 7-10

</code_context>

<specifics>
## Specific Ideas

- Config builder defaults must match Go's `DefaultServerConfig()` and `DefaultEmbeddedConfig()` field-for-field
- `ServerSession` callback slots should be designed so backends wire them at construction time (similar to `EmbeddedSession`'s `LongConsumer closeAction`)
- The `callFfi()` template should handle the common pattern visible in both backends: call → check null → read error → throw or return

</specifics>

<deferred>
## Deferred Ideas

- Per-handle error isolation for concurrent multi-instance support — future phase (user wants to move away from global LAST_ERROR toward handle-scoped errors, enabling true concurrent instances on different directories)
- `ChromaErrorCode` enum on `ChromaException` for programmatic error handling — tracked as FUTURE-01 in REQUIREMENTS.md

</deferred>

---

*Phase: 06-core-foundation-types*
*Context gathered: 2026-03-22*
