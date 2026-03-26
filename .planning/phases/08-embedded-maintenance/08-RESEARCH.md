# Phase 8: Embedded Maintenance - Research

**Researched:** 2026-03-26
**Domain:** Java FFI wiring for embedded maintenance operations (rebuild, compaction, WAL prune)
**Confidence:** HIGH

## Summary

Phase 8 wires 5 existing Rust FFI symbols (`chroma_embedded_rebuild_collection`, `chroma_embedded_compact_collection`, `chroma_embedded_compact_all`, `chroma_embedded_prune_wal_collection`, `chroma_embedded_prune_wal_all`) through `EmbeddedSession` in both JNA and Panama backends. No new Rust exports are needed. All 5 symbols share the same C signature: `(handle: *mut c_void, request_json: *const c_char) -> *mut c_char`.

The core module already has all type infrastructure in place from Phase 6: request builders (`RebuildOptions`, `CompactCollectionRequest`, `CompactAllRequest`, `WALPruneOptions`) with `toJson()` methods, result POJOs (`RebuildCollectionResult`, `CompactionResult`, `WALPruneResult`) with Gson deserialization, and validation logic via `Validation` helpers. The Phase 7 `callFfiJson` template method in `AbstractChromaRuntime` provides the exact pattern needed for all 5 calls.

The primary change is expanding `EmbeddedSession` from a minimal `(handle, closeAction)` constructor to include callback slots for each maintenance operation, following the established `ServerSession` pattern. Both JNA and Panama backends then inject lambdas at construction time that call the appropriate FFI symbols through `callFfiJson`.

**Primary recommendation:** Implement in 3 stages: (1) expand `EmbeddedSession` with callback slots and public methods, (2) wire JNA backend symbols and lambdas, (3) wire Panama backend symbols and lambdas. Integration tests go into each backend module.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Callback slots pattern -- add functional interface fields to `EmbeddedSession` constructor, backends inject lambdas at construction time
- **D-02:** Consistent with `ServerSession` pattern -- both session types use callback slots, keeping core module FFI-free
- **D-03:** `EmbeddedSession` constructor signature expands with maintenance callbacks; existing `(long handle, LongConsumer closeAction)` gains additional function parameters
- **D-04:** Eager symbol binding at `init()` -- all 5 `chroma_embedded_*` maintenance symbols bound during backend initialization, same as existing server symbols
- **D-05:** All 5 FFI functions share `(handle, request_json) -> result_json` signature -- maps to `callFfiJson` template method
- **D-06:** Backends create lambdas that call `callFfiJson(() -> nativeCall(handle, json), ResultType.class)` and inject into `EmbeddedSession` at construction
- **D-07:** Two-tier testing: smoke tests on empty/default embedded instances AND data-seeded tests
- **D-08:** Smoke tests: start embedded, call each maintenance op, verify result types are well-formed
- **D-09:** Data-seeded tests: create collection, add records, then verify ops produce measurable results
- **D-10:** Identical test structure in both `:jna:test` and `:panama:test` modules
- **D-11:** Java-side validation for obvious bad inputs (null/empty collection name) -- throws `IllegalArgumentException` before hitting FFI
- **D-12:** Option builder validation at `build()` time (already implemented in Phase 6 core types)
- **D-13:** Runtime errors (nonexistent collection, etc.) delegated to Rust FFI -- returned via `LAST_ERROR` and wrapped as `ChromaException`
- **D-14:** Follows Phase 6 three-tier exception rule: `IllegalArgumentException` for bad inputs, `IllegalStateException` for closed session, `ChromaException` for FFI failures

### Claude's Discretion
- Exact functional interface types for maintenance callbacks (BiFunction vs custom)
- Whether to use a single callback type for all JSON-in/JSON-out maintenance calls or typed callbacks per operation
- Test data creation approach (Chroma client API or direct FFI calls)
- Order of implementation (JNA first vs Panama first vs parallel)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EMNT-01 | `EmbeddedSession.rebuildCollection(name, options)` returns RebuildCollectionResult in both backends | Callback slot pattern (D-01/D-06) + `callFfiJson` with `chroma_embedded_rebuild_collection` symbol |
| EMNT-02 | `EmbeddedSession.compactCollection(request)` and `compactAll(request)` return CompactionResult in both backends | Same pattern, 2 separate symbols (`compact_collection`, `compact_all`) |
| EMNT-03 | `EmbeddedSession.pruneCollectionWAL(name, options)` and `pruneAllWAL(options)` return WALPruneResult in both backends | Same pattern, 2 separate symbols (`prune_wal_collection`, `prune_wal_all`) |
| EMNT-04 | Option builders (RebuildOptions, WALPruneOptions) validate inputs at build time | Already implemented in Phase 6; `Validation.validateRequiredName()`, `WALPruneOptions.Builder.build()` with policy checks |
| EMNT-05 | Integration tests verify each embedded maintenance operation in both backends | Two-tier strategy: smoke + data-seeded, both `:jna:test` and `:panama:test` |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **Conventional commits** required for all commits
- **No cgo**: FFI via JNA (Java 17+) and Panama (Java 22+)
- **Rust shim artifact**: `libchroma_shim.dylib` on macOS -- must be built before running Java tests (`make build` or `make build-debug`)
- **Radically simple** architecture and implementation
- **No excessive comments** -- code and function/var names should be self-explanatory
- **Linting**: `gradle --no-daemon :core:check :jna:check :panama:check` for Java
- **Testing**: `make test-java` runs both `:jna:test` and `:panama:test` with `CHROMA_LIB_PATH` auto-set
- **No Rust shim changes** allowed (requirement: "New Rust shim exports" is out of scope)

## Architecture Patterns

### Callback Slot Pattern (EmbeddedSession Expansion)

`ServerSession` already demonstrates the callback slot pattern. The `EmbeddedSession` constructor must expand from 2 parameters to include 5 maintenance callbacks plus the existing close action. Each maintenance operation on `EmbeddedSession` delegates to its callback slot.

**Current EmbeddedSession constructor:**
```java
public EmbeddedSession(long handle, LongConsumer closeAction)
```

**Expanded pattern (recommendation):**
```java
// Use BiFunction<Long, String, T> for the (handle, requestJson) -> resultJson pattern
// A single callback type works because all 5 ops share the same FFI signature
public EmbeddedSession(long handle, LongConsumer closeAction,
        BiFunction<Long, String, RebuildCollectionResult> rebuildAction,
        BiFunction<Long, String, CompactionResult> compactCollectionAction,
        BiFunction<Long, String, CompactionResult> compactAllAction,
        BiFunction<Long, String, WALPruneResult> pruneWalCollectionAction,
        BiFunction<Long, String, WALPruneResult> pruneWalAllAction)
```

### Discretion Decision: Callback Type Strategy

**Recommendation: Use typed `BiFunction<Long, String, ResultType>` per operation.**

Rationale:
- All 5 FFI symbols share identical C signature `(handle, json) -> json`, but each deserializes to a different Java result type
- The backend already handles deserialization via `callFfiJson(LongSupplier, Class<T>)` before creating the lambda
- Typed callbacks mean `EmbeddedSession` methods are simple one-liners: `return rebuildAction.apply(handle, options.toJson())`
- Consistent with `ServerSession` which uses typed function interfaces for each accessor (`LongToIntFunction` for port, `LongFunction<String>` for address)
- Constructor parameter count (7 total) is comparable to `ServerSession` (6 total)

### Discretion Decision: Implementation Order

**Recommendation: JNA first, then Panama.**

Rationale:
- JNA is simpler to debug (no `invokeExact` signature-polymorphic constraints)
- Establishes the pattern for `EmbeddedSession` expansion, which Panama then follows identically
- Both backends share the same test structure, so JNA tests can be copy-adapted for Panama

### Method Implementation Pattern

Each `EmbeddedSession` method follows a consistent pattern:

```java
// In EmbeddedSession
public RebuildCollectionResult rebuildCollection(String name, RebuildOptions options) {
    ensureOpen();
    if (name == null || name.isBlank()) {
        throw new IllegalArgumentException("name is required");
    }
    return rebuildAction.apply(handle, options.toJson());
}

public RebuildCollectionResult rebuildCollection(String name) {
    return rebuildCollection(name, RebuildOptions.defaults(name));
}
```

Note: The `name` validation on `EmbeddedSession` methods (D-11) provides a fast Java-side check before hitting FFI. However, `RebuildOptions.defaults(name)` already validates via `Validation.validateRequiredName()`, so the session-level check is a secondary guard for the case where `options` is pre-built with a valid name but the `name` parameter itself is bad.

### Backend Wiring Pattern (JNA)

```java
// In JnaBindings interface -- add 5 new symbols:
Pointer chroma_embedded_rebuild_collection(Pointer handle, String requestJson);
Pointer chroma_embedded_compact_collection(Pointer handle, String requestJson);
Pointer chroma_embedded_compact_all(Pointer handle, String requestJson);
Pointer chroma_embedded_prune_wal_collection(Pointer handle, String requestJson);
Pointer chroma_embedded_prune_wal_all(Pointer handle, String requestJson);

// In doStartEmbedded -- create lambdas using callFfiJson:
return new EmbeddedSession(
    handle,
    this::embeddedFree,
    (h, json) -> callFfiJson(
        () -> Pointer.nativeValue(bindings.chroma_embedded_rebuild_collection(new Pointer(h), json)),
        RebuildCollectionResult.class),
    // ... same pattern for remaining 4 operations
);
```

### Backend Wiring Pattern (Panama)

```java
// In Ffi record -- add 5 new MethodHandle fields:
private record Ffi(
    // ... existing fields ...
    MethodHandle embeddedRebuildCollection,
    MethodHandle embeddedCompactCollection,
    MethodHandle embeddedCompactAll,
    MethodHandle embeddedPruneWalCollection,
    MethodHandle embeddedPruneWalAll) {}

// Symbol binding in init():
linker.downcallHandle(
    requireSymbol(library, "chroma_embedded_rebuild_collection"),
    FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS, ValueLayout.ADDRESS))

// In doStartEmbedded -- create lambdas using callFfiJson:
return new EmbeddedSession(
    handle,
    this::embeddedFree,
    (h, json) -> callFfiJson(() -> {
        try (Arena callArena = Arena.ofConfined()) {
            MemorySegment jsonSeg = callArena.allocateFrom(json);
            MemorySegment result = (MemorySegment) ffi.embeddedRebuildCollection()
                .invokeExact(MemorySegment.ofAddress(h), jsonSeg);
            return result.address();
        } catch (Throwable t) {
            if (t instanceof Error error) throw error;
            throw new ChromaException("failed to call embedded rebuild collection", t);
        }
    }, RebuildCollectionResult.class),
    // ... same pattern for remaining 4 operations
);
```

### Anti-Patterns to Avoid
- **Adding FFI symbols to core module**: Core MUST remain FFI-free. All symbol binding happens in jna/panama backends.
- **Calling FFI without lock**: All calls MUST go through `callFfiJson` which acquires `FFI_LOCK`.
- **Null-checking options in session methods**: The builder pattern already validates at `build()` time. Session methods only need to validate `name` for methods that accept a raw name parameter.
- **Using `callFfiFree` for invokeExact in Panama**: As noted in Phase 7 comments, `invokeExact` cannot be inside lambda bodies passed to `callFfiFree` due to signature-polymorphic bytecode issues.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON serialization | Manual string concatenation | `JsonUtil.toJson()` via Gson with `LOWER_CASE_WITH_UNDERSCORES` | Field naming must match Rust FFI expectations |
| JSON deserialization | Manual parsing | `callFfiJson(supplier, Type.class)` | Handles FFI lock, null pointer check, error slot, string ownership, Gson parsing |
| Input validation | Per-method manual checks | `Validation.validateRequiredName()` et al. | Centralized, consistent, already tested in Phase 6 |
| Request JSON generation | Manual JSON building | `RebuildOptions.toJson()`, `CompactCollectionRequest.toJson()`, etc. | Builder validation + correct field naming already wired |

## Common Pitfalls

### Pitfall 1: Constructor Backward Compatibility
**What goes wrong:** Expanding `EmbeddedSession`'s constructor breaks existing calls in both backends and tests.
**Why it happens:** The constructor currently takes `(long, LongConsumer)` and both `doStartEmbedded()` methods call it.
**How to avoid:** Update both JNA and Panama `doStartEmbedded()` simultaneously. Since `EmbeddedSession` is `public final`, no external subclasses exist. The constructor change is a compile-time break, easily caught.
**Warning signs:** Compilation errors in `JnaChromaRuntime.doStartEmbedded()` and `PanamaChromaRuntime.doStartEmbedded()`.

### Pitfall 2: Panama invokeExact Signature Mismatch
**What goes wrong:** `MethodHandle.invokeExact()` fails at runtime with `WrongMethodTypeException` if the return/parameter types don't match exactly.
**Why it happens:** Panama's `invokeExact` is signature-polymorphic; even casting from `Object` to `MemorySegment` at the call site changes behavior.
**How to avoid:** Follow the exact pattern established in Phase 7: `(MemorySegment) ffi.handle().invokeExact(MemorySegment.ofAddress(addr), segment)`. Always cast to `MemorySegment` explicitly.
**Warning signs:** `WrongMethodTypeException` at test time.

### Pitfall 3: Missing Arena for Panama String Arguments
**What goes wrong:** JSON request strings passed to Panama FFI calls leak memory or crash if not allocated in a scoped `Arena`.
**Why it happens:** Panama requires `Arena.ofConfined()` to allocate native memory for the string; the Go/JNA approach of passing `String` directly doesn't work.
**How to avoid:** Always use `try (Arena callArena = Arena.ofConfined()) { callArena.allocateFrom(json); }` inside the lambda, matching the existing `doStartEmbedded` pattern.
**Warning signs:** Native crashes or memory leaks during Panama tests.

### Pitfall 4: WAL Prune Requires a Policy or DryRun
**What goes wrong:** Calling `pruneCollectionWAL` or `pruneAllWAL` with default options fails validation.
**Why it happens:** `WALPruneOptions.Builder.build()` requires either `dryRun(true)` or at least one policy (maxAge, maxBytes, watermark). The `defaults(name)` factory sets `dryRun(true)` to handle this.
**How to avoid:** Smoke tests should use `WALPruneOptions.defaults(name)` (which sets dryRun). Data-seeded tests that want actual pruning must set a policy.
**Warning signs:** `IllegalArgumentException: at least one WAL prune policy is required` in tests.

### Pitfall 5: EmbeddedSession Method Stubs on ServerSession
**What goes wrong:** `ServerSession` already has maintenance method stubs that throw `UnsupportedOperationException`. Phase 8 wires `EmbeddedSession` only -- `ServerSession` remains stubbed (Phase 10 scope).
**Why it happens:** Mistakenly wiring `ServerSession` maintenance methods in Phase 8.
**How to avoid:** Only modify `EmbeddedSession`. Leave `ServerSession` stubs as-is -- they are Phase 10's responsibility.
**Warning signs:** Scope creep into `ServerSession` wiring.

### Pitfall 6: Data-Seeded Tests Need a Real Collection
**What goes wrong:** Rebuild/compact/prune on a nonexistent collection returns an error from Rust FFI.
**Why it happens:** The Rust shim calls `frontend.get_collection(request)` which fails if the collection doesn't exist.
**How to avoid:** Data-seeded tests cannot create collections through the current Java API (no CRUD operations exposed yet -- FUTURE-03). The Go reference tests use direct FFI calls. For Java, there is no `create_collection` symbol exposed. Tests must either: (a) use only operations that work on empty databases (compact_all with 0 collections returns successfully), or (b) investigate whether the Rust shim has a `chroma_embedded_*` create collection symbol available.
**Warning signs:** `ChromaException: get collection failed` in data-seeded tests.

### Pitfall 7: Gson Field Naming Must Match Rust JSON
**What goes wrong:** Deserialized result fields are null because JSON keys don't match.
**Why it happens:** Rust serializes with `snake_case` (e.g., `collection_id`). Java fields use `camelCase` (e.g., `collectionId`). Gson's `LOWER_CASE_WITH_UNDERSCORES` policy handles this mapping.
**How to avoid:** Ensure all result POJOs use `camelCase` field names and rely on `JsonUtil.GSON` for deserialization (which applies the naming policy). Already handled in Phase 6.
**Warning signs:** Null fields in deserialized results.

## Code Examples

### EmbeddedSession Expansion (verified pattern from ServerSession)

```java
// Source: ServerSession.java callback slot pattern
public final class EmbeddedSession implements AutoCloseable {
    private final long handle;
    private final LongConsumer closeAction;
    private final AtomicBoolean closed;
    private final BiFunction<Long, String, RebuildCollectionResult> rebuildAction;
    private final BiFunction<Long, String, CompactionResult> compactCollectionAction;
    private final BiFunction<Long, String, CompactionResult> compactAllAction;
    private final BiFunction<Long, String, WALPruneResult> pruneWalCollectionAction;
    private final BiFunction<Long, String, WALPruneResult> pruneWalAllAction;

    public EmbeddedSession(long handle, LongConsumer closeAction,
            BiFunction<Long, String, RebuildCollectionResult> rebuildAction,
            BiFunction<Long, String, CompactionResult> compactCollectionAction,
            BiFunction<Long, String, CompactionResult> compactAllAction,
            BiFunction<Long, String, WALPruneResult> pruneWalCollectionAction,
            BiFunction<Long, String, WALPruneResult> pruneWalAllAction) {
        // ... null checks for all parameters ...
        this.handle = handle;
        this.closeAction = closeAction;
        this.closed = new AtomicBoolean(false);
        this.rebuildAction = rebuildAction;
        this.compactCollectionAction = compactCollectionAction;
        this.compactAllAction = compactAllAction;
        this.pruneWalCollectionAction = pruneWalCollectionAction;
        this.pruneWalAllAction = pruneWalAllAction;
    }
}
```

### JNA Symbol Binding (verified pattern from existing JnaBindings)

```java
// Source: JnaChromaRuntime.java existing pattern
private interface JnaBindings extends Library {
    // ... existing symbols ...
    Pointer chroma_embedded_rebuild_collection(Pointer handle, String requestJson);
    Pointer chroma_embedded_compact_collection(Pointer handle, String requestJson);
    Pointer chroma_embedded_compact_all(Pointer handle, String requestJson);
    Pointer chroma_embedded_prune_wal_collection(Pointer handle, String requestJson);
    Pointer chroma_embedded_prune_wal_all(Pointer handle, String requestJson);
}
```

### Panama Symbol Binding (verified pattern from existing Ffi record)

```java
// Source: PanamaChromaRuntime.java existing pattern
// All 5 share the same FunctionDescriptor: (ADDRESS, ADDRESS) -> ADDRESS
FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS, ValueLayout.ADDRESS)
```

### Integration Test Pattern (smoke, verified from existing tests)

```java
// Source: JnaChromaRuntimeTest.java test structure
@Test
void embeddedRebuildSmoke(@TempDir Path persistDir) {
    String libPath = System.getenv("CHROMA_LIB_PATH");
    Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

    String yaml = new EmbeddedConfigBuilder()
            .persistPath(persistDir.toAbsolutePath().toString())
            .allowReset(true)
            .build();

    try (JnaChromaRuntime runtime = JnaChromaRuntime.init(libPath);
         EmbeddedSession session = runtime.startEmbedded(yaml)) {
        // Rebuild on nonexistent collection should throw ChromaException
        assertThrows(ChromaException.class,
            () -> session.rebuildCollection("nonexistent", RebuildOptions.defaults("nonexistent")));
    }
}

@Test
void embeddedCompactAllSmoke(@TempDir Path persistDir) {
    // ... same setup ...
    try (/* runtime + session */) {
        CompactAllRequest request = new CompactAllRequest.Builder().build();
        CompactionResult result = session.compactAll(request);
        assertNotNull(result);
        assertEquals(0, result.collectionCount());
    }
}

@Test
void embeddedPruneWalDryRunSmoke(@TempDir Path persistDir) {
    // ... same setup ...
    try (/* runtime + session */) {
        WALPruneOptions options = WALPruneOptions.defaults("nonexistent");
        // prune on nonexistent collection should throw ChromaException
        assertThrows(ChromaException.class,
            () -> session.pruneCollectionWAL("nonexistent", options));
    }
}
```

## Open Questions

1. **Data-seeded test feasibility**
   - What we know: The Rust shim exposes `chroma_embedded_rebuild_collection`, `chroma_embedded_compact_collection`, etc. but these require an existing collection. There is no Java-exposed `create_collection` API (that is FUTURE-03).
   - What's unclear: Whether `compact_all` and `prune_wal_all` on an empty database return meaningful results (0 collections processed) or error. The Go tests use the Go API which has collection creation wired through a separate code path.
   - Recommendation: Smoke tests should focus on `compactAll` and `pruneAllWAL` with empty databases (which should succeed with 0 collections). For single-collection ops, testing error handling (nonexistent collection -> ChromaException) is the pragmatic approach. True data-seeded tests can wait for FUTURE-03 or be done via raw FFI calls to `chroma_embedded_*` create collection symbols if they exist.

2. **Collection creation via undocumented FFI symbols**
   - What we know: The Rust shim may expose `chroma_embedded_create_collection` or similar symbols -- the shim file is very large (4800+ lines). If such symbols exist, data-seeded tests could use direct FFI to create a collection, then run maintenance ops.
   - What's unclear: Whether these symbols exist and their exact signatures.
   - Recommendation: Check if `chroma_embedded_create_collection` exists in the shim. If it does, a test-only helper can wrap it for data-seeded scenarios. If not, rely on smoke-level testing for Phase 8 and defer data-seeded tests.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | JUnit Jupiter 5.11.4 |
| Config file | `java/build.gradle.kts` (root), `java/jna/build.gradle.kts`, `java/panama/build.gradle.kts` |
| Quick run command | `make test-java` (builds shim + runs `:jna:test` and `:panama:test`) |
| Full suite command | `make test-all` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EMNT-01 | rebuildCollection returns RebuildCollectionResult | integration | `make test-java` (covers both backends) | No -- Wave 0 |
| EMNT-02 | compactCollection/compactAll return CompactionResult | integration | `make test-java` | No -- Wave 0 |
| EMNT-03 | pruneCollectionWAL/pruneAllWAL return WALPruneResult | integration | `make test-java` | No -- Wave 0 |
| EMNT-04 | Option builders validate at build time | unit (core) | Already tested in Phase 6 core tests | Yes (Phase 6) |
| EMNT-05 | Integration tests in both backends | integration | `make test-java` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `make build-java` (compile check)
- **Per wave merge:** `make test-java` (full integration)
- **Phase gate:** `make test-all` green before verify

### Wave 0 Gaps
- [ ] `java/jna/src/test/.../JnaEmbeddedMaintenanceTest.java` -- covers EMNT-01, EMNT-02, EMNT-03, EMNT-05
- [ ] `java/panama/src/test/.../PanamaEmbeddedMaintenanceTest.java` -- covers EMNT-01, EMNT-02, EMNT-03, EMNT-05
- EMNT-04 is already covered by Phase 6 core module tests

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Java | All Java compilation/tests | Yes | 26 | -- |
| Gradle | Build and test | No (no system gradle, no gradlew) | -- | Install via `brew install gradle` or add Gradle wrapper |
| Rust/Cargo | Build libchroma_shim | No | -- | `make build` requires Rust toolchain |
| libchroma_shim.dylib | Integration tests | No (not built) | -- | `make build` to create |

**Missing dependencies with no fallback:**
- Gradle must be available to compile and test Java code. The Makefile uses `$(JAVA_GRADLE)` which resolves to system `gradle`. Install Gradle 9+ or add a Gradle wrapper to `java/`.
- Rust toolchain must be available to build the native shim via `make build`.

**Missing dependencies with fallback:**
- None -- all dependencies are required.

## Sources

### Primary (HIGH confidence)
- `EmbeddedSession.java` -- current minimal session implementation (read directly)
- `ServerSession.java` -- callback slot pattern reference (read directly)
- `AbstractChromaRuntime.java` -- `callFfiJson`, `callFfiHandle`, FFI_LOCK pattern (read directly)
- `JnaChromaRuntime.java` -- JNA symbol binding, `doStartEmbedded`, string handling (read directly)
- `PanamaChromaRuntime.java` -- Panama symbol binding, Arena usage, `invokeExact` patterns (read directly)
- `shim/src/lib.rs` (lines 4443-4810) -- all 5 FFI symbol signatures verified (read directly)
- `RebuildOptions.java`, `CompactCollectionRequest.java`, `CompactAllRequest.java`, `WALPruneOptions.java` -- builder and toJson patterns (read directly)
- `RebuildCollectionResult.java`, `CompactionResult.java`, `WALPruneResult.java` -- result POJO structures (read directly)
- Go reference: `internal/runtime/rebuild.go`, `compaction.go`, `wal_prune.go`, `embedded.go` -- request JSON formats and API shapes (read directly)

### Secondary (MEDIUM confidence)
- Phase 7 decisions in STATE.md regarding `serverFree`/`embeddedFree` bypassing `callFfiVoid`

### Tertiary (LOW confidence)
- Data-seeded test feasibility -- unclear whether empty-database `compact_all`/`prune_wal_all` returns success or error without running the actual FFI calls

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries and patterns already established in Phases 6-7
- Architecture: HIGH -- callback slot pattern proven in ServerSession, callFfiJson proven in Phase 7
- Pitfalls: HIGH -- derived from direct code reading and Phase 7 precedent
- Testing: MEDIUM -- data-seeded test strategy depends on collection creation availability

**Research date:** 2026-03-26
**Valid until:** 2026-04-26 (stable domain, no external dependency changes expected)
